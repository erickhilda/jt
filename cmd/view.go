package cmd

import (
	"fmt"
	"strings"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/store"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view <TICKET-KEY>",
	Short: "Print a local ticket to stdout",
	Long:  "Prints the locally saved markdown file to stdout for piping to other tools.",
	Args:  cobra.ExactArgs(1),
	RunE:  runView,
}

func init() {
	rootCmd.AddCommand(viewCmd)
}

func runView(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	content, err := store.Load(cfg.TicketsDir, key)
	if err != nil {
		return fmt.Errorf("ticket %s not found locally; run 'jt pull %s' first", key, key)
	}

	fmt.Print(content)
	return nil
}
