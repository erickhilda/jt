package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/jira"
	"github.com/erickhilda/jt/internal/renderer"
	"github.com/erickhilda/jt/internal/store"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var diffCmd = &cobra.Command{
	Use:   "diff <TICKET-KEY>",
	Short: "Show diff between local and remote ticket",
	Long:  "Fetches the latest version from Jira and shows a unified diff against the local file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDiff,
}

func init() {
	diffCmd.Flags().String("color", "auto", "Color output: auto, always, never")
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, args []string) error {
	ticketKey := strings.ToUpper(strings.TrimSpace(args[0]))
	colorFlag, _ := cmd.Flags().GetString("color")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Load local file.
	localContent, err := store.Load(cfg.TicketsDir, ticketKey)
	if err != nil {
		return fmt.Errorf("ticket %s not found locally; run 'jt pull %s' first", ticketKey, ticketKey)
	}

	// Fetch fresh from Jira.
	token, err := config.GetToken(cfg)
	if err != nil {
		return fmt.Errorf("retrieving token: %w", err)
	}

	client := jira.NewClient(cfg.Instance, cfg.Email, token)
	fetchComments := cfg.ShouldFetchComments()
	issue, err := client.GetIssueWithFields(ticketKey, issueFieldsFor(fetchComments))
	if err != nil {
		if errors.Is(err, jira.ErrNotFound) {
			return fmt.Errorf("ticket %s not found on Jira", ticketKey)
		}
		if errors.Is(err, jira.ErrUnauthorized) {
			return fmt.Errorf("authentication failed: %w", err)
		}
		return fmt.Errorf("fetching ticket: %w", err)
	}

	// When comments aren't fetched, strip any stale local "## Comments" block
	// so the two sides are symmetric and don't produce phantom diffs.
	if !fetchComments {
		localContent = store.RemoveSection(localContent, "## Comments")
	}

	// Render fresh content, preserving local notes.
	remoteContent := renderer.RenderIssue(issue)
	remoteContent = preserveNotes(localContent, remoteContent)

	if localContent == remoteContent {
		fmt.Printf("No changes for %s\n", ticketKey)
		return nil
	}

	// Generate unified diff.
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(localContent),
		B:        difflib.SplitLines(remoteContent),
		FromFile: ticketKey + " (local)",
		ToFile:   ticketKey + " (remote)",
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return fmt.Errorf("generating diff: %w", err)
	}

	useColor := shouldColor(colorFlag)
	if useColor {
		text = colorizeDiff(text)
	}

	fmt.Print(text)
	return nil
}

func shouldColor(flag string) bool {
	switch flag {
	case "always":
		return true
	case "never":
		return false
	default:
		return term.IsTerminal(int(os.Stdout.Fd()))
	}
}

const (
	ansiRed   = "\033[31m"
	ansiGreen = "\033[32m"
	ansiCyan  = "\033[36m"
	ansiReset = "\033[0m"
)

func colorizeDiff(text string) string {
	lines := strings.Split(text, "\n")
	var b strings.Builder
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			// File header lines - keep as-is.
			b.WriteString(line)
		case strings.HasPrefix(line, "+"):
			b.WriteString(ansiGreen + line + ansiReset)
		case strings.HasPrefix(line, "-"):
			b.WriteString(ansiRed + line + ansiReset)
		case strings.HasPrefix(line, "@@"):
			b.WriteString(ansiCyan + line + ansiReset)
		default:
			b.WriteString(line)
		}
		if i < len(lines)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
