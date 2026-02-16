package cmd

import (
	"fmt"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/jira"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
}

var authTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Verify Jira credentials",
	Long:  "Calls the Jira /rest/api/3/myself endpoint to verify your credentials are valid.",
	RunE:  runAuthTest,
}

func init() {
	authCmd.AddCommand(authTestCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuthTest(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	token, err := config.GetToken(cfg)
	if err != nil {
		return fmt.Errorf("retrieving token: %w", err)
	}

	client := jira.NewClient(cfg.Instance, cfg.Email, token)
	user, err := client.Myself()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Authenticated as %s (%s)\n", user.DisplayName, user.Email)
	fmt.Printf("Account ID: %s\n", user.AccountID)
	fmt.Printf("Time zone:  %s\n", user.TimeZone)
	fmt.Printf("Active:     %v\n", user.Active)
	return nil
}
