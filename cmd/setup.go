package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	var (
		setupName   string
		setupHook   string
		setupType   string
		setupNoTest bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure HookTap with a guided first-run wizard",
		Example: `  hooktap setup
  hooktap setup --name ci --hook https://hooks.hooktap.me/webhook/YOUR_ID --type push`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			reader := bufio.NewReader(cmd.InOrStdin())
			name := strings.TrimSpace(firstNonEmpty(setupName, flagProfile))
			if name == "" {
				name = prompt(reader, cmd.OutOrStdout(), "Profile name", config.DefaultProfileName)
			}
			if name == "" {
				name = config.DefaultProfileName
			}

			hook := strings.TrimSpace(firstNonEmpty(setupHook, flagHook))
			if hook == "" {
				hook = prompt(reader, cmd.OutOrStdout(), "Webhook id or full URL", "")
			}
			if hook == "" {
				return fmt.Errorf("%w: webhook id or URL is required", errUsage)
			}

			eventType := strings.TrimSpace(setupType)
			if eventType == "" {
				eventType = prompt(reader, cmd.OutOrStdout(), "Default event type", client.DefaultType)
			}
			if eventType == "" {
				eventType = client.DefaultType
			}
			if !client.ValidType(eventType) {
				return fmt.Errorf("%w: invalid type %q, must be one of push, feed, widget", errUsage, eventType)
			}

			base, id := splitWebhook(hook)
			if id == "" {
				return fmt.Errorf("%w: could not determine webhook id from %q", errUsage, hook)
			}
			if flagURL != "" {
				base = flagURL
			}

			prof := config.Profile{HookID: id, Type: eventType}
			if base != "" && base != client.DefaultBaseURL {
				prof.URL = strings.TrimRight(base, "/") + "/webhook/" + id
				prof.HookID = ""
			}
			cfg.Profiles[name] = prof
			cfg.Default = name

			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved profile %q\n", name)

			if setupNoTest {
				return nil
			}

			c := client.New(base)
			resp, err := c.Send(context.Background(), id, client.Payload{
				Type:  eventType,
				Title: "HookTap CLI setup complete",
				Body:  "Your terminal is connected.",
			})
			if err != nil {
				return fmt.Errorf("saved profile, but test send failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "test sent (%s) event %s\n", resp.Type, resp.EventID)
			return nil
		},
	}

	cmd.Flags().StringVar(&setupName, "name", "", "profile name to create or update")
	cmd.Flags().StringVar(&setupHook, "hook", "", "webhook id or full URL")
	cmd.Flags().StringVarP(&setupType, "type", "t", client.DefaultType, "default event type: push|feed|widget")
	cmd.Flags().BoolVar(&setupNoTest, "no-test", false, "save configuration without sending a test notification")
	return cmd
}

func prompt(reader *bufio.Reader, out io.Writer, label, fallback string) string {
	if fallback != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, fallback)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return text
}
