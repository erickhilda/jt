package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/jira"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up jt configuration",
	Long:  "Interactive wizard to configure Jira Cloud connection.",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	exists, err := config.Exists()
	if err != nil {
		return err
	}
	if exists {
		ok, err := promptConfirm("Configuration already exists. Overwrite?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Println("Aborted.")
			return nil
		}
	}

	reader := bufio.NewReader(os.Stdin)

	instance, err := promptString(reader, "Jira instance URL (e.g. https://myorg.atlassian.net): ")
	if err != nil {
		return err
	}
	instance = strings.TrimRight(instance, "/")
	if !strings.HasPrefix(instance, "https://") {
		return fmt.Errorf("instance URL must start with https://")
	}

	email, err := promptString(reader, "Email: ")
	if err != nil {
		return err
	}
	if email == "" {
		return fmt.Errorf("email is required")
	}

	fmt.Print("API token: ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return fmt.Errorf("API token is required")
	}

	defaultProject, err := promptString(reader, "Default project key (optional, press Enter to skip): ")
	if err != nil {
		return err
	}

	storage, err := config.SetToken(email, token)
	if err != nil {
		return fmt.Errorf("storing token: %w", err)
	}

	cfg := &config.Config{
		Instance:       instance,
		Email:          email,
		DefaultProject: strings.ToUpper(strings.TrimSpace(defaultProject)),
		TicketsDir:     "~/.jt/tickets",
		TokenStorage:   storage,
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Config saved (token stored via %s).\n", storage)

	// Verify credentials.
	client := jira.NewClient(instance, email, token)
	user, err := client.Myself()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: credential verification failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "Config was saved. You can fix credentials with 'jt config set' and retry with 'jt auth test'.")
		return nil
	}

	fmt.Printf("Authenticated as %s (%s)\n", user.DisplayName, user.Email)
	return nil
}

func promptString(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func promptConfirm(prompt string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes", nil
}
