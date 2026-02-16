package cmd

import (
	"fmt"
	"strings"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/store"
	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path <TICKET-KEY>",
	Short: "Print the local file path for a ticket",
	Long:  "Prints the full filesystem path to the ticket's markdown file.\nUseful for scripting: claude < $(jt path PROJ-123)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPath,
}

func init() {
	rootCmd.AddCommand(pathCmd)
}

func runPath(cmd *cobra.Command, args []string) error {
	key := strings.ToUpper(strings.TrimSpace(args[0]))

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	path, err := store.TicketPath(cfg.TicketsDir, key)
	if err != nil {
		return err
	}

	fmt.Println(path)
	return nil
}
