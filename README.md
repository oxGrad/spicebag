# Prospector

CV, cover letter, and job application manager for Claude Code.

Manage versioned CVs and cover letters in Markdown, export to PDF, track job applications, and let Claude Code customize your CV and write cover letters — all from your terminal.

## What it does

- Stores base CV variants and cover letters as Markdown in `~/.config/prospector/`
- Extracts experience stats (years per role type) from CV frontmatter into SQLite
- Tracks job applications with status history
- Exports PDFs via Gotenberg (Docker container)
- Adds three slash commands to Claude Code: `/customize-cv`, `/write-cover-letter`, `/apply`
- Web dashboard at `http://localhost:8080`

## Prerequisites

- Go 1.22+
- Docker
- Claude Code

## Installation

### 1. Install the binary

```bash
go install github.com/graditya/prospector/cmd/prospector@latest
```

### 2. Run first-time setup

```bash
prospector init
```

This creates `~/.config/prospector/`, registers the MCP server with Claude Code, installs default themes to `~/.config/prospector/themes/`, and installs slash commands to `~/.claude/commands/prospector/`.

### 3. Start Gotenberg (PDF export)

```bash
prospector up
```

### 4. Start the dashboard

```bash
prospector serve        # foreground
prospector serve -d     # background (logs to ~/.config/prospector/prospector.log)
prospector stop         # stop background server
```

Open `http://localhost:8080` in your browser.

## Slash commands

Use these inside Claude Code. Arguments can be a file reference (`@job-post.md`), a URL, or pasted text.

| Command | What it does |
|---|---|
| `/customize-cv <job post or role>` | Tailors the most relevant base CV to the role and saves a new versioned file |
| `/write-cover-letter <job post>` | Writes a cover letter and saves it to the library |
| `/apply <job post>` | Full application: tailored CV + cover letter + saved job post |

## CLI reference

| Command | Description |
|---|---|
| `prospector init` | First-time setup: config dir, MCP registration, themes, slash commands |
| `prospector up` | Start Gotenberg via docker compose |
| `prospector serve` | Start the dashboard (foreground) |
| `prospector serve -d` | Start the dashboard (background) |
| `prospector stop` | Stop the background dashboard server |
| `prospector sync` | Sync experience stats from CV frontmatter to SQLite |
| `prospector mcp` | Run the MCP server (invoked automatically by Claude Code) |

## MCP tools

These tools are available to Claude Code via the MCP server:

`list_cvs`, `read_cv`, `write_cv`, `list_cover_letters`, `read_cover_letter`, `write_cover_letter`, `get_experience_stats`, `create_application`, `list_applications`, `export_pdf`

## CV frontmatter

Add an `experience` block to your CV Markdown files. Run `prospector sync` after editing to update the SQLite database.

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
- View experience stats and application analytics
- Upload custom CSS themes

## Themes

CSS files in `~/.config/prospector/themes/` control how CVs and cover letters render in the dashboard and PDF export.

Two themes are installed by default:
- `minimal` — serif font, clean layout
- `modern` — sans-serif, blue accents

To add more: upload a `.css` file via the dashboard, or drop it directly into `~/.config/prospector/themes/`.

## Config directory layout

```
~/.config/prospector/
  config.toml          dashboard port, Gotenberg URL
  prospector.db        SQLite: experience stats + application tracking
  cv/                  base CV markdown files
  cover-letters/       cover letter templates
  themes/              CSS themes
  applications/        one folder per job application
  prospector.log       background dashboard log (when using serve -d)
```

## Uninstall

```bash
rm $(which prospector)
rm -rf ~/.config/prospector
rm -rf ~/.claude/commands/prospector
# Remove the "prospector" entry from ~/.claude/mcp.json under mcpServers
```
