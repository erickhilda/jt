# jt — Jira Ticket CLI

> A lightweight CLI tool to pull Jira Cloud tickets into local markdown files for offline access and LLM context feeding.

---

## Vision

A simple, fast, Go-based CLI (inspired by `gh`, `gcloud`, `bird`) that bridges Jira Cloud and your local filesystem. Pull tickets once, work offline, refresh when needed. No browser, no copy-paste.

**Core workflow:**

```
jt pull PROJ-123        →  ~/.jt/tickets/PROJ-123.md
jt pull PROJ-123        →  updates the same file with latest content
cat ~/.jt/tickets/PROJ-123.md | claude   →  instant context
```

---

## Local File Format

Each ticket becomes a self-contained markdown file:

```markdown
<!-- jt:meta ticket=PROJ-123 fetched=2026-02-14T10:30:00Z -->
# PROJ-123: Implement OAuth2 flow

| Field       | Value                          |
|-------------|--------------------------------|
| Status      | In Progress                    |
| Type        | Story                          |
| Priority    | High                           |
| Assignee    | you@company.com                |
| Reporter    | pm@company.com                 |
| Sprint      | Sprint 14                      |
| Epic        | PROJ-80: Authentication        |
| Labels      | backend, security              |
| Created     | 2026-02-01                     |
| Updated     | 2026-02-13                     |

## Description

As a user, I want to authenticate via OAuth2 so that...

### Acceptance Criteria

- [ ] Support Google and GitHub providers
- [ ] Token refresh works silently
- [ ] Logout clears all stored tokens

## Subtasks

- [x] PROJ-124: Research OAuth2 libraries (Done)
- [ ] PROJ-125: Implement token storage (In Progress)
- [ ] PROJ-126: Add logout endpoint (To Do)

## Linked Issues

- blocks PROJ-130: Protected API endpoints
- is blocked by PROJ-110: User model migration

## Pull Requests (1)

- [MERGED] [PROJ-123: implement OAuth2 flow](https://bitbucket.org/acme/repo/pull-requests/42) (#42)
  - Branch: feature/PROJ-123 -> develop
  - Author: Alice
  - Approved by: Bob

## Comments (5)

### Alice — 2026-02-10 09:15
We should use PKCE for the mobile app flow.

### Bob — 2026-02-12 14:22
+1 on PKCE. I've updated the design doc in Confluence.

### You — 2026-02-13 11:00
Started implementation. Will push a draft PR today.
```

---

## Development Roadmap

### Phase 0 — Project Setup (Day 1) [DONE]

- [x] Initialize Go module (`github.com/erickhilda/jt`)
- [x] Set up project structure (see Architecture below)
- [x] Choose CLI framework: **cobra** (industry standard, used by `kubectl`, `gh`, `hugo`)
- [x] Set up CI with goreleaser for cross-platform binaries
- [ ] Write README with installation instructions

### Phase 1 — Auth & Config (Days 2–3) [DONE]

**Goal:** Connect to Jira Cloud securely.

- [x] `jt init` — Interactive setup wizard
  - Prompt for Jira instance URL (`https://yourcompany.atlassian.net`)
  - Prompt for email + API token (masked input via `x/term`)
  - Prompt for default project key (optional)
  - Save config to `~/.jt/config.yaml`
  - Verify credentials via `/rest/api/3/myself`
- [x] `jt config set <key> <value>` — Update individual settings
- [x] `jt config show` — Display current config (mask token)
- [x] Store API token securely (system keyring via `go-keyring`, fallback to `~/.jt/credentials` with 0600 perms)
- [x] `jt auth test` — Verify credentials work

**Config file (`~/.jt/config.yaml`):**

```yaml
instance: https://yourcompany.atlassian.net
email: you@company.com
default_project: PROJ
tickets_dir: ~/.jt/tickets    # configurable
token_storage: keyring         # or "file" if keyring unavailable
```

### Phase 2 — Pull & View (Days 4–7) [DONE]

**Goal:** Fetch tickets and save as local markdown.

- [x] `jt pull <TICKET-KEY>` — Fetch ticket from Jira REST API v3, render to markdown, save to `tickets_dir`
  - Fetches: summary, description, status, assignee, reporter, priority, type, labels, sprint, epic, comments, subtasks, linked issues
  - Converts Jira's ADF (Atlassian Document Format) to markdown
  - Saves as `<TICKET-KEY>.md`
  - If file exists, overwrites with fresh content (preserves any local `## Notes` section — see below)
- [x] `jt pull <TICKET-KEY> --comments-only` — Only update the comments section
- [x] `jt pull <TICKET-KEY> --dry-run` — Show diff of what would change
- [x] `jt view <TICKET-KEY>` — Print local markdown to stdout (for piping)
- [x] `jt open <TICKET-KEY>` — Open ticket in default browser
- [x] `jt path <TICKET-KEY>` — Print the file path (useful for scripts: `claude < $(jt path PROJ-123)`)
- [x] Handle ADF → Markdown conversion:
  - Headings, paragraphs, lists (ordered/unordered)
  - Code blocks (with language)
  - Tables
  - Mentions (@user)
  - Links, images (media nodes -> markdown image refs + an `## Attachments` section, Phase 9) — **Tier 1 done; inline local download (Tier 2) deferred**
  - Panels (info/warning/error → blockquotes with prefix)

**Local notes preservation:**
If the user adds a `## My Notes` section at the bottom of the file, `jt pull` should preserve it across updates. This lets you annotate tickets locally.

### Phase 3 — Sync & Diff (Days 8–10)

**Goal:** Keep local files fresh with minimal effort.

- [ ] `jt sync` — Re-pull all locally saved tickets that have been updated on Jira since last fetch
  - Uses `updated` field from Jira REST API
  - Only fetches tickets where remote `updated > local fetched` timestamp
- [ ] `jt sync --project PROJ` — Sync only tickets from a specific project
- [ ] `jt diff <TICKET-KEY>` — Show what changed since last pull (like `git diff`)
  - Color-coded: new comments in green, status changes highlighted
- [ ] `jt status` — Overview of all local tickets: how many are stale, recently updated, etc.

### Phase 4 — Search & List (Days 11–13)

**Goal:** Browse and search tickets without leaving the terminal.

- [ ] `jt list` — List locally saved tickets (from filesystem)
  - Show: key, title, status, last fetched
  - Flags: `--sort`, `--filter-status`
- [ ] `jt search <JQL>` — Run a JQL query against Jira, display results
  - e.g., `jt search "assignee = currentUser() AND status = 'In Progress'"`
- [ ] `jt mine` — Shortcut for tickets assigned to you (current sprint)
- [ ] `jt sprint` — Show current sprint board for default project
- [ ] `jt pull --jql <JQL>` — Bulk pull all tickets matching a query
  - e.g., `jt pull --jql "sprint = currentSprint() AND assignee = currentUser()"`
  - Great for pulling your entire sprint at once

### Phase 5 — Quality of Life (Days 14–16)

**Goal:** Polish the experience.

- [ ] `jt alias` — Create short aliases for common JQL queries
  - `jt alias add wip "assignee = currentUser() AND status = 'In Progress'"`
  - `jt wip` → runs the saved query
- [ ] Shell completions (bash, zsh, fish) — auto-complete ticket keys from local files
- [ ] `jt export <TICKET-KEY> --format json` — Export as JSON (for programmatic use)
- [ ] `jt clean` — Remove local files for tickets that are Done/Closed
- [ ] `jt log <TICKET-KEY>` — Show pull history (when was this ticket last fetched?)
- [ ] Rich terminal output with color (but plain text when piped — detect TTY)
- [ ] `--output` flag on all commands: `table`, `json`, `markdown`, `plain`
- [ ] Man pages / `jt help <command>` with examples

### Phase 6 — Stretch Goals (Future)

- [ ] `jt watch <TICKET-KEY>` — Poll for changes and notify (desktop notification)
- [ ] `jt comment <TICKET-KEY> "message"` — Post a comment from CLI
- [ ] `jt transition <TICKET-KEY> "In Review"` — Change ticket status
- [ ] Confluence integration: `jt pull --include-confluence` fetches linked Confluence pages
- [ ] Git integration: `jt pull --from-branch` infers ticket key from current branch name (e.g., `feature/PROJ-123-oauth`)
- [ ] MCP server mode: expose as a tool for Claude Desktop / Claude Code
- [ ] Offline full-text search across all local tickets (using bleve or similar)

### Phase 7 — Bitbucket PR support (`jt pr`) [DONE]

**Goal:** Pull a Bitbucket Cloud PR (diff + comments + metadata) into a local
markdown file for code-review context, mirroring `jt pull` for tickets.

Read-only, self-serve via a scoped Bitbucket API token — useful when the official
Atlassian MCP Bitbucket integration isn't available.

- [x] Milestone 0 — auth spike: validated `email:token` + read scopes against `api.bitbucket.org`
- [x] Milestone 1 — `internal/bitbucket` client + `jt pr <id>` (git-remote inference), `--no-diff`, My Notes preservation, `~/.jt/prs/<workspace>__<repo>__<id>.md`, Jira-key linking
- [x] `jt pr list [repo]` — repo-scoped PR table on stdout (`--state` open|merged|declined|all, `--limit`), newest-updated first, Jira-key column; no files written
- [ ] Deferred (v2): write-back (approve/comment/merge), `jt pr view/open/path`, workspace-wide `jt pr list --workspace` + `--mine`, `--json`, diff path-filtering, Bitbucket Server/DC

### Phase 8 — Confluence page support (`jt page`) [DONE]

**Goal:** Pull a Confluence Cloud page (title + metadata + body) into a local markdown
file for offline reading and LLM context, mirroring `jt pull` for tickets.

Same Atlassian host and Basic auth as Jira, so it reuses the existing Jira token and
the ADF-to-markdown converter (`jira.RenderADF`).

- [x] `internal/confluence` client — `GetPage(id)` against `/wiki/api/v2/pages/{id}?body-format=atlas_doc_format`
- [x] `renderer.RenderPage` — metadata table + `## Content` (ADF body reused via `jira.RenderADF`)
- [x] `jt page <id | url>` — numeric ID or page URL, reuses the Jira token, `--dry-run`, My Notes preservation, `~/.jt/pages/<space>__<id>__<slug>.md` (`pages_dir`)
- [ ] Deferred (v2): child-page expansion, page comments, attachments/labels, `jt page view/open/path/list`, CQL search, sync/diff for pages, scoped-token `jt auth confluence`

### Phase 9 — Image / attachment handling (Tier 1) [DONE]

**Goal:** stop silently dropping embedded images. Render media nodes as markdown
image references and list every attachment with its download URL, for **Jira tickets
and Confluence pages**. Pure markdown — no binaries written (Tier 1).

Root cause was a missing `media` case in the shared ADF converter: images fell
through to `default` and emitted nothing.

- [x] `internal/jira/adf.go` — `media` / `mediaInline` rendering via `mediaMarkdown`
  (external -> `![alt](url)`; file -> `![alt](filename)`, `![image](<id>)` fallback)
- [x] Jira `## Attachments` — `Attachment` field (free in the existing fetch) +
  `RenderIssue` section
- [x] Confluence `## Attachments` — `GetPageAttachments` (paginated) + `RenderPage`
  section, relative `downloadLink` resolved to absolute via `absURL`
- [ ] Deferred (Tier 2): opt-in `--assets` to download images into `<key>_assets/`
  and rewrite refs to relative paths (self-contained, offline, multimodal); Bitbucket
  PR image download

---

## Architecture

```
jt/
├── cmd/                    # CLI commands (cobra)
│   ├── root.go
│   ├── init.go             # Interactive setup wizard
│   ├── auth.go             # jt auth test
│   ├── config.go           # jt config show/set
│   ├── pull.go             # (Phase 2)
│   ├── view.go             # (Phase 2)
│   ├── list.go             # (Phase 3)
│   ├── search.go           # (Phase 3)
│   ├── sync.go             # (Phase 4)
│   └── mine.go             # (Phase 3)
├── internal/
│   ├── config/             # Config loading/saving
│   │   ├── config.go       # Config struct, Load/Save/Validate
│   │   └── credentials.go  # Token storage (keyring + file fallback)
│   ├── jira/               # Jira API client
│   │   ├── client.go       # HTTP client, Basic auth
│   │   ├── types.go        # API response types
│   │   ├── errors.go       # APIError, ErrUnauthorized
│   │   └── adf.go          # ADF → Markdown converter (Phase 2)
│   ├── renderer/           # Ticket → Markdown renderer
│   │   └── markdown.go     # (Phase 2)
│   ├── store/              # Local file management
│   │   └── store.go        # Read/write/list local tickets (Phase 2)
│   └── tui/                # Terminal UI helpers
│       └── output.go       # Colors, tables, TTY detection (Phase 5)
├── go.mod
├── go.sum
├── main.go
├── Makefile
└── README.md
```

---

## Key Dependencies

| Package | Purpose | Status |
|---------|---------|--------|
| `github.com/spf13/cobra` | CLI framework | In use |
| `gopkg.in/yaml.v3` | Config marshal/unmarshal | In use |
| `github.com/zalando/go-keyring` | Secure token storage | In use |
| `golang.org/x/term` | Password masking | In use |
| `github.com/charmbracelet/lipgloss` | Terminal styling | Phase 5 |
| `github.com/charmbracelet/glamour` | Markdown rendering in terminal | Phase 5 |

---

## API Endpoints Used

All via Jira Cloud REST API v3 (`/rest/api/3/`):

| Endpoint | Used By |
|----------|---------|
| `GET /rest/api/3/issue/{key}` | `jt pull` — full ticket with comments |
| `GET /rest/api/3/issue/{key}?expand=renderedFields,names,changelog` | Extended pull |
| `GET /rest/api/3/issue/{key}/comment` | Comments (paginated) |
| `GET /rest/api/3/search?jql=...` | `jt search`, `jt mine`, `jt sprint` |
| `GET /rest/api/3/myself` | `jt auth test` |
| `GET /rest/api/3/project/{key}` | Project info |
| `POST /rest/api/3/issue/{key}/comment` | `jt comment` (Phase 6) |
| `POST /rest/api/3/issue/{key}/transitions` | `jt transition` (Phase 6) |

**Auth:** Basic auth with email + API token (Base64 encoded in `Authorization` header).

---

## Installation Plan

```bash
# Homebrew (macOS/Linux)
brew install <you>/tap/jt

# Go install
go install github.com/<you>/jt@latest

# Binary download (goreleaser)
curl -sSL https://github.com/<you>/jt/releases/latest/download/jt_$(uname -s)_$(uname -m).tar.gz | tar xz
```

---

## Estimated Timeline

| Phase | Scope | Time |
|-------|-------|------|
| Phase 0 | Project setup | 1 day |
| Phase 1 | Auth & config | 2 days |
| Phase 2 | Pull & view (core) | 4 days |
| Phase 3 | Search & list | 3 days |
| Phase 4 | Sync & diff | 3 days |
| Phase 5 | Quality of life | 3 days |
| **MVP (Phases 0–2)** | **Usable product** | **~1 week** |
| **Full v1.0 (Phases 0–5)** | **Complete CLI** | **~2.5 weeks** |

---

## Success Criteria

- **MVP:** `jt init` + `jt pull PROJ-123` + `jt view PROJ-123` works end-to-end
- **v1.0:** Can replace the browser-based Jira workflow for daily ticket reading
- **Stretch:** Claude can access ticket context without any manual copy-paste
