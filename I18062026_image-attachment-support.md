# Plan: Image / attachment handling for tickets and pages (Tier 1)

## Context

Goal: stop silently dropping images. When a Jira ticket or Confluence page embeds
an image, the materialized markdown previously contained no trace of it — the image
vanished. This makes the local `.md` a lossy copy and hurts it as LLM context.

Root cause: both tickets and pages render their body through the shared ADF
converter (`internal/jira/adf.go`). Images in ADF are `media` nodes wrapped in
`mediaSingle` / `mediaGroup`. The converter handled the wrappers (recursed into
children) but had **no `media` case**, so the image node fell through to `default`,
which has no text/children to emit — the image was dropped with no placeholder. On
top of that the attachment metadata wasn't even fetched (Jira `IssueFields` had no
`attachment` field; the Confluence client only called `GetPage`).

Bitbucket PRs are unaffected: their description/comments are already markdown, so an
embedded `![](url)` survives as text. They were intentionally left out of this change.

## Scope

**Tier 1 — never lose images, no binary downloads.** Render media nodes as markdown
image references inline (preserving position + alt text) and add an `## Attachments`
section that lists every attachment with its download URL. Pure markdown; no files
written beside the `.md`. Sources: **Jira tickets + Confluence pages**.

Tier 2 (download images into a sibling `<key>_assets/` dir and rewrite refs to
relative paths) is deferred — see Follow-ups.

## Resolved decisions

| # | Decision | Choice | Rationale |
|---|----------|--------|-----------|
| Q1 | How to render an embedded image | **Markdown image ref** `![alt](src)` inline + an `## Attachments` list | Preserves position and alt text; the Attachments section is the authoritative download list readers correlate by filename. |
| Q2 | `src` for **external** media | **The direct `attrs.url`** | Fully resolvable; renders anywhere. |
| Q3 | `src` for **file** media | **`attrs.alt` (usually the filename)**, falling back to `![image](<media-id>)` | The ADF media node's `attrs.id` is a media-services UUID, not the attachment id, so there is no reliable node-level download URL. Filename is the practical bridge to the Attachments section; media-id keeps it traceable when alt is absent. |
| Q4 | Where attachment metadata comes from (Jira) | **The issue's `attachment` field** — already returned | Both the default field set and the no-comments path (`*all,-comment`) include `attachment`. No extra request, no field-query change. |
| Q5 | Where attachment metadata comes from (Confluence) | **`GET /wiki/api/v2/pages/{id}/attachments`** (paginated) | v2 has no body-embedded attachment list; a dedicated call is required. |
| Q6 | Attachment-fetch failure (Confluence) | **Non-fatal** — warn to stderr, still save the page | Attachments are secondary; a 403/blip shouldn't block materializing the page body. |
| Q7 | Confluence `downloadLink` is relative | **Resolve to absolute against `page._links.base`** via `absURL` | v2 returns `/download/attachments/...`; `absURL` prefixes the base and de-duplicates a leading `/wiki`. |

## Local file format (additions)

Inline, where an image appeared in the body:

```markdown
![diagram.png](diagram.png)
```

New section (tickets after Description; pages after Content):

```markdown
## Attachments

- arch.png (image/png) - https://acme.atlassian.net/.../attachment/content/42
- notes.txt (text/plain) - https://acme.atlassian.net/.../attachment/content/43
```

## Changes

`internal/jira/adf.go`:
- Add `media` / `mediaInline` cases (block + inline) and a `mediaMarkdown(node)`
  helper: external -> `![alt](url)`; file -> `![alt](filename)` with an
  `![image](<id>)` fallback. Block media emits its own trailing blank line; the
  `mediaSingle` / `mediaGroup` wrappers just recurse.

`internal/jira/types.go`:
- Add `Attachment` type and `Attachment []Attachment` to `IssueFields` (`attachment`).

`internal/renderer/markdown.go`:
- `RenderIssue` emits `## Attachments` after Description when present.
- New shared `writeAttachment(b, name, mime, url)` helper (`- name (mime) - url`,
  segments omitted when empty).

`internal/confluence/types.go`:
- Add `Attachment` (`id`, `title`, `mediaType`, `fileSize`, `downloadLink`) and the
  internal `attachmentList` page wrapper.

`internal/confluence/client.go`:
- `GetPageAttachments(id)` — `/wiki/api/v2/pages/{id}/attachments?limit=250`, follows
  `_links.next` pagination, returns an empty slice (not an error) when there are none.

`internal/renderer/page.go`:
- `RenderPage` gains an `attachments []confluence.Attachment` param and emits
  `## Attachments` after Content; new `absURL(base, link)` resolves relative
  download links against `page._links.base`.

`cmd/page.go`:
- Fetch attachments after the page (non-fatal warn on error) and pass them to
  `RenderPage`.

**Reused unchanged:** the ADF converter pipeline for both sources, `writeRow` /
`formatDate`, the confluence `do`/`getJSON` path-or-URL idiom (handles relative
`next`), `store.*`, notes preservation.

## Verification

- ADF unit tests: file media with alt, file media without alt (id fallback), external
  media (direct url), inline media inside a paragraph. All assert the exact markdown.
- Renderer unit tests: Jira `## Attachments` rendered with mime + url; no section when
  there are no attachments; Confluence `## Attachments` with relative -> absolute URL
  resolution; `absURL` table (relative, `/wiki`-dedup, absolute pass-through, empty
  base, empty link).
- Confluence client tests: `GetPageAttachments` follows pagination (two requests) and
  the empty-list case. httptest, mirroring `TestGetPage`.
- `go build ./...`, `go test ./...`, `go vet ./...` all clean.

## Risks / Follow-ups

- **File-media inline ref is best-effort:** because the node carries a media-services
  UUID rather than the attachment id, `![alt](filename)` relies on `alt` holding the
  filename. The `## Attachments` section is the authoritative download list. Tier 2
  removes this caveat.
- **Tier 2 (deferred):** opt-in `--assets` flag to download images into
  `<key>_assets/` and rewrite refs to relative paths — self-contained markdown that
  renders offline and feeds multimodal models.
- **Bitbucket PRs (deferred):** image URLs already survive as markdown text but are
  auth-gated; downloading them is a Tier-2 concern.
- **Auth-gated URLs:** the Attachments download URLs require the same Atlassian
  credentials; they are pointers, not embedded content, in Tier 1.
