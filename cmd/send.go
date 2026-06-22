package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	var (
		flagType  string
		flagBody  string
		flagTitle string
		flagRaw   bool
	)

	cmd := &cobra.Command{
		Use:   "send [title]",
		Short: "Send an event to a HookTap webhook",
		Example: `  hooktap send "Build finished"
  echo "Staging is live" | hooktap send --title "Deploy"
  hooktap send "CI failed" --body "main branch" --type push
  cat report.txt | hooktap send "Nightly"
  generate-payload.sh | hooktap send --raw`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			titleArg := ""
			if len(args) == 1 {
				titleArg = strings.TrimSpace(args[0])
			}

			stdin, hasStdin, err := readStdin(cmd.InOrStdin())
			if err != nil {
				return fmt.Errorf("read stdin: %w", err)
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			prof := cfg.Profile(cfg.ResolveName(flagProfile))

			s, err := resolveSettings(flagURL, flagHook, flagType, prof)
			if err != nil {
				return err
			}
			c := client.New(s.baseURL)

			// ── Raw mode: pipe a complete JSON body straight through ──────────
			if flagRaw {
				if !hasStdin {
					return fmt.Errorf("%w: --raw expects a JSON body on stdin", errUsage)
				}
				if !json.Valid(stdin) {
					return fmt.Errorf("%w: --raw stdin is not valid JSON", errUsage)
				}
				resp, err := c.SendRaw(context.Background(), s.hookID, stdin)
				if err != nil {
					return err
				}
				printSent(cmd.ErrOrStderr(), resp)
				return nil
			}

			// ── Structured mode ───────────────────────────────────────────────
			eventType := s.defaultType
			if !client.ValidType(eventType) {
				return fmt.Errorf("%w: invalid --type %q, must be one of push, feed, widget", errUsage, eventType)
			}

			title, body, err := resolveContent(titleArg, flagTitle, flagBody, defaultTitle(), string(stdin), hasStdin)
			if err != nil {
				return err
			}

			resp, err := c.Send(context.Background(), s.hookID, client.Payload{
				Type:  eventType,
				Title: title,
				Body:  body,
			})
			if err != nil {
				return err
			}
			printSent(cmd.ErrOrStderr(), resp)
			return nil
		},
	}

	cmd.Flags().StringVarP(&flagType, "type", "t", "", "event type: push|feed|widget (default push)")
	cmd.Flags().StringVarP(&flagBody, "body", "b", "", "event body text (overrides stdin)")
	cmd.Flags().StringVar(&flagTitle, "title", "", "event title (alternative to the positional argument)")
	cmd.Flags().BoolVar(&flagRaw, "raw", false, "send a complete JSON body read from stdin verbatim")
	return cmd
}

// readStdin returns piped/redirected input. When stdin is an interactive
// terminal it returns hasData=false without blocking on a read.
func readStdin(in io.Reader) (data []byte, hasData bool, err error) {
	// Only os.Stdin carries TTY information; an injected reader (tests) is
	// always treated as piped input.
	if f, ok := in.(*os.File); ok {
		stat, statErr := f.Stat()
		if statErr == nil && stat.Mode()&os.ModeCharDevice != 0 {
			return nil, false, nil // interactive terminal — nothing piped
		}
	}
	data, err = io.ReadAll(in)
	if err != nil {
		return nil, false, err
	}
	return data, len(strings.TrimSpace(string(data))) > 0, nil
}

// resolveContent applies the title/body precedence rules:
//   - body:  --body flag, else trimmed stdin, else empty
//   - title: positional arg, else --title flag, else (if a body exists)
//     defaultTitle, else a usage error (the server requires a non-empty title)
func resolveContent(titleArg, titleFlag, bodyFlag, defaultTitle, stdin string, hasStdin bool) (title, body string, err error) {
	switch {
	case bodyFlag != "":
		body = strings.TrimSpace(bodyFlag)
	case hasStdin:
		body = strings.TrimSpace(stdin)
	}

	switch {
	case titleArg != "":
		title = titleArg
	case strings.TrimSpace(titleFlag) != "":
		title = strings.TrimSpace(titleFlag)
	case body != "":
		title = defaultTitle
	default:
		return "", "", fmt.Errorf("%w: a title is required — pass it as an argument, --title, or pipe text to stdin", errUsage)
	}
	return title, body, nil
}

// defaultTitle is used when text is piped without an explicit title.
func defaultTitle() string {
	if h, err := os.Hostname(); err == nil && h != "" {
		return "Notification from " + h
	}
	return "Webhook Event"
}

func printSent(w io.Writer, resp *client.Response) {
	fmt.Fprintf(w, "✓ sent (%s) event %s\n", resp.Type, resp.EventID)
}
