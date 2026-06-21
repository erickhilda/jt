package cmd

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/erickhilda/jt/internal/config"
	"github.com/erickhilda/jt/internal/confluence"
	"github.com/erickhilda/jt/internal/renderer"
	"github.com/erickhilda/jt/internal/store"
	"github.com/spf13/cobra"
)

// pageURLRe extracts the numeric page id from a Confluence page URL, handling
// both view (/pages/<id>/<slug>) and edit (/pages/edit-v2/<id>) forms.
var pageURLRe = regexp.MustCompile(`pages/(?:edit-v2/)?(\d+)`)

// spaceKeyRe extracts the space key from a webui link (/spaces/<KEY>/...).
var spaceKeyRe = regexp.MustCompile(`/spaces/([^/]+)`)

var pageCmd = &cobra.Command{
	Use:   "page <PAGE-ID | URL>",
	Short: "Fetch a Confluence page and save as markdown",
	Long: `Fetches a Confluence Cloud page (title, metadata, body) and saves it as local
markdown for offline reading and LLM context, mirroring 'jt pull' for tickets.

Reference forms:
  jt page 12345                                                     numeric page ID
  jt page https://acme.atlassian.net/wiki/spaces/ENG/pages/12345/Title   full page URL

Uses your existing Jira API token (same Atlassian account) -- no separate auth is
needed as long as the token has Confluence access (unscoped API tokens do).`,
	Args: cobra.ExactArgs(1),
	RunE: runPage,
}

func init() {
	pageCmd.Flags().Bool("dry-run", false, "Show what would change without saving")
	rootCmd.AddCommand(pageCmd)
}

func runPage(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	id, err := resolvePageRef(args[0])
	if err != nil {
		return err
	}

	token, err := config.GetToken(cfg)
	if err != nil {
		return fmt.Errorf("retrieving token: %w", err)
	}

	client := confluence.NewClient(cfg.Instance, cfg.Email, token)
	page, err := client.GetPage(id)
	if err != nil {
		return wrapConfluenceError(err, id)
	}

	spaceKey := spaceKeyFromWebUI(page.Links.WebUI)
	if spaceKey == "" {
		spaceKey = page.SpaceID
	}
	webURL := pageWebURL(cfg.Instance, page)

	// Attachments are secondary: a failure here should not block saving the page.
	attachments, aerr := client.GetPageAttachments(id)
	if aerr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch attachments for page %s: %v\n", id, aerr)
	}

	content := renderer.RenderPage(page, spaceKey, webURL, attachments)

	pagesDir := cfg.PagesDirOrDefault()
	key := pageFileKey(spaceKey, page.ID, page.Title)

	// Preserve a hand-added "## My Notes" section across re-pulls.
	if existing, lerr := store.Load(pagesDir, key); lerr == nil {
		content = preserveNotes(existing, content)
	}

	if dryRun {
		return showDryRunDir(pagesDir, key, content)
	}

	if err := store.Save(pagesDir, key, content); err != nil {
		return fmt.Errorf("saving page: %w", err)
	}

	path, _ := store.TicketPath(pagesDir, key)
	fmt.Printf("Saved Confluence page %s to %s\n", page.ID, path)
	return nil
}

// resolvePageRef parses a page reference (numeric id or page URL) into a page id.
func resolvePageRef(arg string) (string, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", fmt.Errorf("page reference is required")
	}
	if isAllDigits(arg) {
		return arg, nil
	}
	if m := pageURLRe.FindStringSubmatch(arg); m != nil {
		return m[1], nil
	}
	return "", fmt.Errorf("invalid page reference %q: expected a numeric page ID or a Confluence page URL (.../pages/<id>/...)", arg)
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// spaceKeyFromWebUI extracts the human space key (e.g. "ENG") from a relative
// webui link; returns "" when the link doesn't match.
func spaceKeyFromWebUI(webui string) string {
	if m := spaceKeyRe.FindStringSubmatch(webui); m != nil {
		return m[1]
	}
	return ""
}

// pageWebURL builds the absolute page URL from the v2 links, preferring the
// API-provided base and falling back to <instance>/wiki + the relative webui.
func pageWebURL(instance string, p *confluence.Page) string {
	if p.Links.WebUI == "" {
		return ""
	}
	if p.Links.Base != "" {
		return strings.TrimRight(p.Links.Base, "/") + p.Links.WebUI
	}
	return strings.TrimRight(instance, "/") + "/wiki" + p.Links.WebUI
}

// pageFileKey builds the flat, space-prefixed file key for a saved page.
func pageFileKey(spaceKey, id, title string) string {
	sp := spaceKey
	if sp == "" {
		sp = "page"
	}
	slug := slugify(title)
	if slug == "" {
		return fmt.Sprintf("%s__%s", sp, id)
	}
	return fmt.Sprintf("%s__%s__%s", sp, id, slug)
}

// slugify lowercases a title and collapses non-alphanumeric runs to single
// dashes, capping the result so filenames stay reasonable.
func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	dash := false
	for _, r := range s {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		default:
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	const maxLen = 60
	if len(slug) > maxLen {
		slug = strings.Trim(slug[:maxLen], "-")
	}
	return slug
}

func wrapConfluenceError(err error, id string) error {
	switch {
	case errors.Is(err, confluence.ErrUnauthorized):
		return fmt.Errorf("authentication failed: %w (your Jira token may be invalid; re-run 'jt init')", err)
	case errors.Is(err, confluence.ErrForbidden):
		return err
	case errors.Is(err, confluence.ErrNotFound):
		return fmt.Errorf("page %s not found or no access", id)
	default:
		return err
	}
}
