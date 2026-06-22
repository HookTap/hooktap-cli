// Package cmd wires the cobra command tree on top of internal/client.
package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/hooktap/hooktap-cli/internal/client"
	"github.com/spf13/cobra"
)

// Exit codes are part of the CLI contract so scripts can branch on them.
const (
	exitOK         = 0
	exitError      = 1 // HTTP/network/server error
	exitUsage      = 2 // bad flags or input
	exitRateLimit  = 4 // HTTP 429 — retryable
)

// Persistent flags shared by all subcommands.
var (
	flagURL     string // override base URL (staging/self-host)
	flagHook    string // webhook id (or full url)
	flagProfile string // config profile to use
)

// BuildInfo carries version metadata from main (set by GoReleaser).
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func newRootCmd(b BuildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "hooktap",
		Short:         "Send webhook events to the HookTap app from your terminal",
		Version:       fmt.Sprintf("%s (commit %s, built %s)", b.Version, b.Commit, b.Date),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("hooktap {{.Version}}\n")
	root.PersistentFlags().StringVar(&flagURL, "url", "", "base URL override (default https://hooks.hooktap.me)")
	root.PersistentFlags().StringVar(&flagHook, "hook", "", "webhook id (or HOOKTAP_HOOK_ID / HOOKTAP_WEBHOOK_URL)")
	root.PersistentFlags().StringVarP(&flagProfile, "profile", "p", "", "config profile to use (default: the file's default profile)")

	root.AddCommand(newSendCmd())
	root.AddCommand(newConfigCmd())
	return root
}

// Execute runs the CLI and returns the process exit code.
func Execute(b BuildInfo) int {
	if err := newRootCmd(b).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "hooktap:", err)
		return exitCodeFor(err)
	}
	return exitOK
}

// exitCodeFor maps an error to its CLI exit code.
func exitCodeFor(err error) int {
	switch {
	case err == nil:
		return exitOK
	case errors.Is(err, errUsage):
		return exitUsage
	case errors.Is(err, client.ErrRateLimited):
		return exitRateLimit
	default:
		return exitError
	}
}

// errUsage marks errors caused by bad user input (→ exitUsage).
var errUsage = errors.New("usage error")
