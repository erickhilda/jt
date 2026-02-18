package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/store"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of local tickets",
	Long:  "Lists all locally saved tickets with freshness info. No API calls are made.",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	tickets, err := store.ListTickets(cfg.TicketsDir)
	if err != nil {
		return fmt.Errorf("listing tickets: %w", err)
	}

	if len(tickets) == 0 {
		fmt.Println("No local tickets found.")
		return nil
	}

	now := time.Now()
	var stale, veryStale int
	for _, t := range tickets {
		age := now.Sub(t.Fetched)
		if age > 7*24*time.Hour {
			veryStale++
		} else if age > 24*time.Hour {
			stale++
		}
	}

	fmt.Printf("Local tickets: %d total", len(tickets))
	if stale > 0 || veryStale > 0 {
		parts := []string{}
		if stale > 0 {
			parts = append(parts, fmt.Sprintf("%d stale (>24h)", stale))
		}
		if veryStale > 0 {
			parts = append(parts, fmt.Sprintf("%d very stale (>7d)", veryStale))
		}
		fmt.Printf(", %s", strings.Join(parts, ", "))
	}
	fmt.Println()

	// Group by project prefix.
	groups := map[string][]store.TicketInfo{}
	for _, t := range tickets {
		proj := t.Key
		if idx := strings.Index(t.Key, "-"); idx >= 0 {
			proj = t.Key[:idx]
		}
		groups[proj] = append(groups[proj], t)
	}

	// Sort project names.
	projects := make([]string, 0, len(groups))
	for p := range groups {
		projects = append(projects, p)
	}
	sort.Strings(projects)

	multiProject := len(projects) > 1

	fmt.Println()
	for _, proj := range projects {
		tix := groups[proj]
		if multiProject {
			fmt.Printf("## %s (%d)\n", proj, len(tix))
		}
		fmt.Printf("%-14s %-20s %s\n", "Key", "Fetched", "Age")
		for _, t := range tix {
			fetchedStr := t.Fetched.Local().Format("2006-01-02 15:04")
			ageStr := humanAge(now, t.Fetched)
			fmt.Printf("%-14s %-20s %s\n", t.Key, fetchedStr, ageStr)
		}
		if multiProject {
			fmt.Println()
		}
	}

	return nil
}

func humanAge(now time.Time, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		return fmt.Sprintf("%dh ago", h)
	default:
		days := int(d.Hours() / 24)
		suffix := ""
		if days >= 7 {
			suffix = " (stale)"
		}
		return fmt.Sprintf("%dd ago%s", days, suffix)
	}
}
