package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/jira"
	"github.com/erickhilda/jt/internal/renderer"
	"github.com/erickhilda/jt/internal/store"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push <TICKET-KEY>",
	Short: "Push locally-edited sections back to Jira",
	Long: `Reads the local ticket file and pushes changed sections back to the Jira description.
Only sections that differ from the remote content are updated.
Warns and exits if Jira was updated after your last pull.`,
	Args: cobra.ExactArgs(1),
	RunE: runPush,
}

func init() {
	pushCmd.Flags().Bool("dry-run", false, "Print what would be sent without updating Jira")
	pushCmd.Flags().String("sections", "Technical Requirements,Release Notes",
		"Comma-separated section headings to push (without ## prefix)")
	rootCmd.AddCommand(pushCmd)
}

func runPush(cmd *cobra.Command, args []string) error {
	ticketKey := strings.ToUpper(strings.TrimSpace(args[0]))
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	sectionsFlag, _ := cmd.Flags().GetString("sections")

	targetSections := parseSectionNames(sectionsFlag)
	if len(targetSections) == 0 {
		return fmt.Errorf("--sections must list at least one section name")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	localContent, err := store.Load(cfg.TicketsDir, ticketKey)
	if err != nil {
		return fmt.Errorf("no local file for %s; run 'jt pull %s' first", ticketKey, ticketKey)
	}

	meta := store.ParseMeta(localContent)
	if meta == nil {
		return fmt.Errorf("local file for %s has no jt:meta header; try 'jt pull %s' to refresh it", ticketKey, ticketKey)
	}

	token, err := config.GetToken(cfg)
	if err != nil {
		return fmt.Errorf("retrieving token: %w", err)
	}

	client := jira.NewClient(cfg.Instance, cfg.Email, token)

	issue, err := client.GetIssue(ticketKey)
	if err != nil {
		if errors.Is(err, jira.ErrNotFound) {
			return fmt.Errorf("ticket %s not found", ticketKey)
		}
		if errors.Is(err, jira.ErrUnauthorized) {
			return fmt.Errorf("authentication failed: check 'jt auth test'")
		}
		return fmt.Errorf("fetching remote ticket: %w", err)
	}

	// Conflict check: warn if Jira was updated after last pull.
	if issue.Fields.Updated != "" {
		updated, parseErr := time.Parse(time.RFC3339, issue.Fields.Updated)
		if parseErr == nil && updated.After(meta.Fetched) {
			return fmt.Errorf(
				"ticket %s was updated on Jira after your last pull\n"+
					"  remote updated: %s\n"+
					"  local fetched:  %s\n"+
					"Run 'jt pull %s' or 'jt sync' first, then re-apply your edits",
				ticketKey,
				updated.Format(time.RFC3339),
				meta.Fetched.Format(time.RFC3339),
				ticketKey,
			)
		}
	}

	// Convert remote description ADF → Markdown for section comparison.
	remoteMarkdown := renderer.RenderIssue(issue)

	// Determine which sections have local changes.
	type sectionUpdate struct {
		heading  string
		newNodes []jira.ADFNode
	}
	var updates []sectionUpdate

	for _, name := range targetSections {
		heading := "## " + name
		localSection := store.ExtractSection(localContent, heading)
		remoteSection := store.ExtractSection(remoteMarkdown, heading)

		if localSection == "" {
			continue // section absent locally — skip
		}
		if localSection == remoteSection {
			continue // no change — skip
		}

		// Convert only the section body (strip the heading line) to ADF nodes.
		bodyMD := sectionBody(localSection, heading)
		newNodes := jira.MarkdownToADF(bodyMD).Content

		updates = append(updates, sectionUpdate{heading: name, newNodes: newNodes})
	}

	if len(updates) == 0 {
		fmt.Println("Nothing to push: no local changes detected in target sections.")
		return nil
	}

	// Build the updated ADF description by splicing each changed section.
	updatedDoc := issue.Fields.Description
	if updatedDoc == nil {
		updatedDoc = &jira.ADFDoc{Type: "doc", Version: 1}
	}
	for _, u := range updates {
		updatedDoc = jira.SpliceSection(updatedDoc, u.heading, u.newNodes)
	}

	if dryRun {
		sectionNames := make([]string, len(updates))
		for i, u := range updates {
			sectionNames[i] = u.heading
		}
		fmt.Printf("Would push sections: %s\n\n", strings.Join(sectionNames, ", "))
		out, _ := json.MarshalIndent(updatedDoc, "", "  ")
		fmt.Println(string(out))
		return nil
	}

	if err := client.UpdateDescription(ticketKey, updatedDoc); err != nil {
		if errors.Is(err, jira.ErrUnauthorized) {
			return fmt.Errorf("authentication failed: check 'jt auth test'")
		}
		return fmt.Errorf("updating ticket: %w", err)
	}

	for _, u := range updates {
		fmt.Printf("Updated '%s' in %s\n", u.heading, ticketKey)
	}
	return nil
}

// parseSectionNames splits the --sections flag value into trimmed names.
func parseSectionNames(flag string) []string {
	var names []string
	for _, part := range strings.Split(flag, ",") {
		name := strings.TrimSpace(part)
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// sectionBody returns the markdown content of a section without the heading line.
func sectionBody(section, heading string) string {
	after := strings.TrimPrefix(section, heading)
	return strings.TrimLeft(after, "\n")
}
