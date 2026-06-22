package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/spf13/cobra"
)

func newPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Check that the HookTap service is reachable (GET /health)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// ping needs no webhook — only the base URL (flag/env, else default).
			baseURL := firstNonEmpty(flagURL, os.Getenv("HOOKTAP_BASE_URL"))
			c := client.New(baseURL)

			h, err := c.Health(context.Background())
			if err != nil {
				return err
			}

			if flagJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				return enc.Encode(h)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ok — %s (%s)\n", h.Service, h.Timestamp)
			return nil
		},
	}
}
