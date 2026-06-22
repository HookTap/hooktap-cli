package main

import (
	"os"

	"github.com/hooktap/hooktap-cli/cmd"
)

// Build metadata, injected by GoReleaser via -ldflags (-X main.version=…).
// Defaults make `go run .` and source builds report a sensible value.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cmd.Execute(cmd.BuildInfo{Version: version, Commit: commit, Date: date}))
}
