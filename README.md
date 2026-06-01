# 🌶️ Spice Bag

**Season Every Application Perfectly**

Your CV is like a great recipe. You have all the right ingredients — experience, skills, achievements — but without the right seasoning, it might not stand out.

Spice Bag is an AI-powered tool that helps you tailor your CV and cover letter for every job application. Like a chef selecting the perfect spices for a dish, it analyzes each job posting and helps you emphasize the right experiences, skills, and achievements to make your application stand out.

No more generic CVs. Just perfectly seasoned applications for every role you apply to.

## What it does

- **Your Spice Bag** — stores base CV variants and cover letters as Markdown in `~/.config/spicebag/`
- **Flavor extraction** — pulls experience stats (years per role type) from CV frontmatter into SQLite
- **Application tracking** — tracks every job application with full status history
- **PDF export** — exports to print-quality PDF via Gotenberg (no browser print dialogs)
- **Three slash commands** — `/customize-cv`, `/write-cover-letter`, `/apply` directly in Claude Code
- **Dashboard** — browse and manage everything at `http://localhost:8080`

## Prerequisites

- Go 1.22+
- Docker (for Gotenberg PDF export)
- Claude Code

## Installation

### Option A — Homebrew

```bash
brew install oxGrad/tap/spicebag
```

### Option B — go install

```bash
go install github.com/oxGrad/spicebag/cmd/spicebag@latest
```

### Setup (Options A and B)

```bash
spicebag init
```

This creates `~/.config/spicebag/`, registers the MCP server with Claude Code, installs default themes to `~/.config/spicebag/themes/`, and installs slash commands to `~/.claude/commands/spicebag/`.

### Open the dashboard

```bash
spicebag start        # foreground
spicebag start -d     # background (logs to ~/.config/spicebag/spicebag.log)
spicebag stop         # stop background server
```

Open `http://localhost:8080` in your browser. The dashboard includes a Gotenberg status widget — use it to start the PDF export service when needed.

## Claude Plugin Marketplace

Install directly from the marketplace — no `spicebag init` needed.

**1. Add the marketplace** (first time only):

```bash
claude plugin marketplace add oxGrad/spicebag
```

**2. Install the plugin:**

```bash
claude plugin install spicebag@spicebag
```

This registers the MCP server, installs default themes, and adds slash commands automatically.

## Slash commands

Use these inside Claude Code. Arguments can be a file reference (`@job-post.md`), a URL, or pasted text.

| Command | What it does |
|---|---|
| `/customize-cv <job post or role>` | Select the recipe — tailor your most relevant base CV for the role and save a new version |
| `/write-cover-letter <job post>` | A fresh cover letter that explains why you're the perfect spice blend for their team |
| `/apply <job post>` | Full application in one pass: tailored CV + cover letter + saved job post |

## The seasoning process

**Step 1 — Your Spice Bag:** Upload your CV — your full ingredient list.

**Step 2 — Select the Recipe:** Paste the job description. Spice Bag understands what flavor profile they're looking for.

**Step 3 — Let's Season:** Claude analyzes the recipe and selects the perfect spice blend from your experience for this specific role.

**Step 4 — Taste Test:** Review your tailored CV and cover letter in the dashboard.

**Step 5 — Serve it Up:** Export to PDF and apply — perfectly seasoned.

## CLI reference

| Command | Description |
|---|---|
| `spicebag init` | First-time setup: config dir, MCP registration, themes, slash commands |
| `spicebag start` | Start the dashboard (foreground) |
| `spicebag start -d` | Start the dashboard (background) |
| `spicebag stop` | Stop the background dashboard server |
| `spicebag sync` | Sync experience stats from CV frontmatter to SQLite |
| `spicebag mcp` | Run the MCP server (invoked automatically by Claude Code) |

## MCP tools

These tools are available to Claude Code via the MCP server:

`list_cvs`, `read_cv`, `write_cv`, `list_cover_letters`, `read_cover_letter`, `write_cover_letter`, `get_experience_stats`, `create_application`, `list_applications`, `export_pdf`

## CV frontmatter (for experience stats)

Add an `experience` block to your CV Markdown files — this is your spice collection. Run `spicebag sync` after editing to update the database.

```yaml
---
experience:
  - role_type: backend
    company: Acme Corp
    start: "2020-01-01"
    end: "2023-06-01"
  - role_type: frontend
    company: Beta Inc
    start: "2023-07-01"
    end: ""
---
# Your CV content here
```

`role_type` is a free-form string (e.g. `backend`, `frontend`, `devops`, `management`). Leave `end` empty for current roles.

## Dashboard

Open `http://localhost:8080` to:

- Browse and export CVs and cover letters with CSS themes
- Track application status (applied → interview → offer / rejected)
- View experience stats
- Upload custom CSS themes

## Themes

CSS files in `~/.config/spicebag/themes/` control how CVs and cover letters render in the dashboard and PDF export. Think of them as your plating style.

Two themes are installed by default:

- `minimal` — serif font, clean layout
- `modern` — sans-serif, blue accents

To add more: upload a `.css` file via the dashboard, or drop it directly into `~/.config/spicebag/themes/`.

## Config directory layout

```
~/.config/spicebag/
  config.toml       dashboard port, Gotenberg URL
  spicebag.db       SQLite: experience stats + application tracking
  cv/               base CV markdown files (your spice bag)
  cover-letters/    cover letter templates
  themes/           CSS themes
  applications/     one folder per job application
  spicebag.log      background dashboard log (when using start -d)
```

## Uninstall

```bash
rm $(which spicebag)
rm -rf ~/.config/spicebag
rm -rf ~/.claude/commands/spicebag
# Remove the "spicebag" entry from ~/.claude/mcp.json under mcpServers
```
