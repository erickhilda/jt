package cmd

import (
	"fmt"
	"os"

	"github.com/erickhilda/atlit/internal/config"
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "atlit",
	Short:   "Atlassian context CLI",
	Long:    "A lightweight CLI that pulls Atlassian content -- Jira tickets, Bitbucket PRs, and Confluence pages -- into local markdown files.",
	Version: version,
}

func Execute() {
	maybeLegacyStateHint()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// maybeLegacyStateHint prints a one-line nudge to stderr when pre-rename ~/.jt
// state is detected, pointing the user at `atlit migrate`. Skipped for the
// migrate command itself.
func maybeLegacyStateHint() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		return
	}
	if config.HasLegacyState() {
		fmt.Fprintln(os.Stderr, "note: legacy ~/.jt state detected -- run 'atlit migrate' to upgrade")
	}
}
