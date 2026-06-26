package cmd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
	"github.com/spf13/cobra"
)

type commandExitError struct {
	code int
	err  error
}

func (e commandExitError) Error() string {
	if e.err != nil {
		return e.err.Error()
	}
	return fmt.Sprintf("command exited with code %d", e.code)
}

func newWatchCmd() *cobra.Command {
	var (
		watchTitle  string
		watchType   string
		watchNoFail bool
	)

	cmd := &cobra.Command{
		Use:   "watch [flags] -- <command>",
		Short: "Run a command and notify your iPhone when it finishes",
		Example: `  hooktap watch -- npm run build
  hooktap watch --title "Deploy" -- make deploy`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && args[0] == "--" {
				args = args[1:]
			}
			if len(args) == 0 {
				return fmt.Errorf("%w: command is required", errUsage)
			}
			if watchType == "" {
				watchType = client.DefaultType
			}
			if !client.ValidType(watchType) {
				return fmt.Errorf("%w: invalid type %q, must be one of push, feed, widget", errUsage, watchType)
			}

			start := time.Now()
			child := exec.CommandContext(context.Background(), args[0], args[1:]...)
			child.Stdin = cmd.InOrStdin()
			child.Stdout = cmd.OutOrStdout()
			child.Stderr = cmd.ErrOrStderr()

			runErr := child.Run()
			duration := time.Since(start).Round(time.Millisecond)
			exitCode := commandExitCode(runErr)

			title := strings.TrimSpace(watchTitle)
			if title == "" {
				title = "Command finished"
			}
			status := "succeeded"
			if exitCode != 0 {
				status = fmt.Sprintf("failed with exit code %d", exitCode)
			}
			body := fmt.Sprintf("%s\n%s in %s", strings.Join(args, " "), status, duration)

			if notifyErr := sendWatchNotification(watchType, title, body); notifyErr != nil {
				if runErr != nil && watchNoFail {
					return commandExitError{code: exitCode, err: runErr}
				}
				return notifyErr
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "hooktap: notified (%s)\n", status)

			if runErr != nil {
				return commandExitError{code: exitCode, err: runErr}
			}
			return nil
		},
	}
	cmd.Flags().SetInterspersed(false)
	cmd.Flags().StringVar(&watchTitle, "title", "", "notification title")
	cmd.Flags().StringVarP(&watchType, "type", "t", client.DefaultType, "event type: push|feed|widget")
	cmd.Flags().BoolVar(&watchNoFail, "no-fail-on-notify", false, "preserve the command exit code if notification delivery fails")
	return cmd
}

func sendWatchNotification(eventType, title, body string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	prof := cfg.Profile(cfg.ResolveName(flagProfile))
	s, err := resolveSettings(flagURL, flagHook, eventType, prof)
	if err != nil {
		return err
	}
	_, err = client.New(s.baseURL).Send(context.Background(), s.hookID, client.Payload{
		Type:  eventType,
		Title: title,
		Body:  body,
	})
	return err
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return 1
}
