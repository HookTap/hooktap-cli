package cmd

import (
	"fmt"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/hooktap/hooktap-cli/internal/config"
	"github.com/spf13/cobra"
)

// configurable profile keys, exposed by `config set`/`config get`.
const (
	keyHookID = "hook_id"
	keyURL    = "url"
	keyType   = "type"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage saved webhook profiles",
	}
	cmd.AddCommand(newConfigSetCmd(), newConfigGetCmd(), newConfigListCmd(), newConfigUseCmd(), newConfigPathCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a profile value (keys: hook_id, url, type)",
		Example: `  hooktap config set hook_id abc12345
  hooktap config set type push --profile ci`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := cfg.ResolveName(flagProfile)
			prof := cfg.Profile(name)

			switch key {
			case keyHookID:
				prof.HookID = value
			case keyURL:
				prof.URL = value
			case keyType:
				if !client.ValidType(value) {
					return fmt.Errorf("%w: invalid type %q, must be one of push, feed, widget", errUsage, value)
				}
				prof.Type = value
			default:
				return fmt.Errorf("%w: unknown key %q (valid: hook_id, url, type)", errUsage, key)
			}

			cfg.Profiles[name] = prof
			// First profile created becomes the default so `send` works at once.
			if cfg.Default == "" {
				cfg.Default = name
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s = %s (profile %q)\n", key, value, name)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a profile value (keys: hook_id, url, type)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			prof := cfg.Profile(cfg.ResolveName(flagProfile))

			var val string
			switch args[0] {
			case keyHookID:
				val = prof.HookID
			case keyURL:
				val = prof.URL
			case keyType:
				val = prof.Type
			default:
				return fmt.Errorf("%w: unknown key %q (valid: hook_id, url, type)", errUsage, args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			names := cfg.ProfileNames()
			if len(names) == 0 {
				fmt.Fprintln(out, "no profiles configured — run 'hooktap config set hook_id <id>'")
				return nil
			}
			defaultName := cfg.ResolveName("")
			for _, name := range names {
				marker := "  "
				if name == defaultName {
					marker = "* "
				}
				p := cfg.Profile(name)
				target := p.HookID
				if target == "" {
					target = p.URL
				}
				fmt.Fprintf(out, "%s%s\t%s\t%s\n", marker, name, target, p.Type)
			}
			return nil
		},
	}
}

func newConfigUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <profile>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Profiles[name]; !ok {
				return fmt.Errorf("%w: no profile named %q", errUsage, name)
			}
			cfg.Default = name
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "default profile is now %q\n", name)
			return nil
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}
