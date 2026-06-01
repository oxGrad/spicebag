# Prospector

CV, cover letter, and job application manager for Claude Code.

Manage versioned CVs and cover letters in Markdown, export to PDF, track job applications, and let Claude Code customize your CV and write cover letters — all from your terminal.

## Installation

### 1. Install the binary

```bash
go install github.com/graditya/prospector/cmd/prospector@latest
prospector init
```

> Binary releases via Homebrew tap are coming once published through GoReleaser.

### 2. Install the Claude Code plugin

Install the **Prospector** plugin from the Claude Code marketplace. This adds the `/customize-cv`, `/write-cover-letter`, and `/apply` slash commands to Claude Code.

### 3. Start Gotenberg (PDF export)

Requires Docker.

```bash
prospector up
```

### 4. Start the dashboard

```bash
prospector serve
```

Then open `http://localhost:8080` in your browser.

## Usage

### Slash commands (in Claude Code)

| Command | What it does |
|---|---|
| `/apply` | Customize your CV + write a cover letter for a job post, saved under `applications/company/role/date/` |
| `/customize-cv` | Create a new base CV variant tailored to a role type |
| `/write-cover-letter` | Write a cover letter for a job post against an existing CV |

Pass a job post as a file reference (`@job-post.md`) or a URL.

### Dashboard

Open `http://localhost:8080` to:
- Browse and export CVs and cover letters with CSS themes
- Track application status (applied → interview → offer/rejected)
- View experience stats and application analytics
- Sync experience data from your CV files


## Config

All data lives in `~/.config/prospector/`:

```
~/.config/prospector/
  config.toml        ← dashboard port, Gotenberg URL
  prospector.db      ← SQLite (experience data + application tracking)
  cv/                ← base CV markdown files
  cover-letters/     ← standalone cover letter templates
  themes/            ← CSS themes (applies in dashboard + PDF export)
  applications/      ← one folder per job application
```
