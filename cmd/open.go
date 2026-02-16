package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/erickhilda/jt/internal/config"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open <TICKET-KEY>",
	Short: "Open a ticket in the default browser",
	Long:  "Opens the Jira issue page in your default web browser.",
	Args:  cobra.ExactArgs(1),
	RunE:  runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	url := strings.TrimRight(cfg.Instance, "/") + "/browse/" + key
	fmt.Printf("Opening %s\n", url)

	return openBrowser(url)
}

func openBrowser(url string) error {
	var cmd string
	var cmdArgs []string

	switch runtime.GOOS {
	case "linux":
		cmd = "xdg-open"
		cmdArgs = []string{url}
	case "darwin":
		cmd = "open"
		cmdArgs = []string{url}
	case "windows":
		cmd = "rundll32"
		cmdArgs = []string{"url.dll,FileProtocolHandler", url}
	default:
		return fmt.Errorf("unsupported platform %s; open manually: %s", runtime.GOOS, url)
	}

	return exec.Command(cmd, cmdArgs...).Start()
}
