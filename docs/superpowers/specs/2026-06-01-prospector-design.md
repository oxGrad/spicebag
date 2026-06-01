# Prospector — Design Spec
_Date: 2026-06-01_

## Overview

Prospector is a personal CV and cover letter management tool that integrates with Claude Code. It consists of a Go binary (MCP server + web dashboard), a Gotenberg Docker container for PDF export, and a Claude Code plugin published to the marketplace. Markdown files are the source of truth for CV and cover letter content. SQLite stores computed experience data and application tracking.

---

## 1. Architecture

### Components

| Component | Description |
|---|---|
| `prospector` binary | Go binary with two modes: `prospector mcp` (MCP server over stdio) and `prospector serve` (web dashboard HTTP server) |
| Gotenberg | Docker container for HTML+CSS → PDF rendering via REST API |
| Claude Code plugin | Marketplace-installable package containing the binary, slash command skills, default themes, and install script |

### Entrypoints

- `prospector mcp` — started by Claude Code automatically via MCP config; exposes tools over stdio
- `prospector serve` — started manually by user; serves web dashboard (default port in `config.toml`)
- `prospector up` — starts Gotenberg via docker-compose
- `prospector sync` — re-parses all base CVs and refreshes SQLite (also available as dashboard button)

---

## 2. Config & File Layout

All user data lives under `~/.config/prospector/`:

```
~/.config/prospector/
  config.toml                     ← dashboard port, gotenberg URL, etc.
  prospector.db                   ← SQLite database
  cv/                             ← base/template CV markdown files
    cv-backend-2025-01-15.md
    cv-devops-2025-01-15.md
  cover-letters/                  ← standalone cover letter templates
    cover-letter-general-2025-01-15.md
  themes/                         ← CSS theme files
    minimal.css
    modern.css
  applications/                   ← one folder per application
    stripe/
      backend-engineer/
        2025-05-20/
          cv.md
          cover-letter.md
          job-post.md             ← saved copy of the job post
          metadata.toml           ← company, role, notes
```

### metadata.toml fields
```toml
company = "Stripe"
role = "Backend Engineer"
applied_date = "2025-05-20"
base_cv_used = "cv-backend-2025-01-15.md"
notes = ""
```

### config.toml fields
```toml
dashboard_port = 8080
gotenberg_url = "http://localhost:3000"
```

---

## 3. Data Model (SQLite)

### `experience` table
Extracted from base CV markdown files on sync. Never manually edited.

```sql
CREATE TABLE experience (
  id          INTEGER PRIMARY KEY,
  role_type   TEXT NOT NULL,      -- "backend", "devops", etc.
  company     TEXT NOT NULL,
  start_date  DATE NOT NULL,
  end_date    DATE,               -- NULL means ongoing
  synced_from TEXT NOT NULL       -- source CV filename
);
```

### `applications` table
Seeded from `metadata.toml` when a new application folder is detected.

```sql
CREATE TABLE applications (
  id           INTEGER PRIMARY KEY,
  company      TEXT NOT NULL,
  role         TEXT NOT NULL,
  applied_date DATE NOT NULL,
  base_cv_used TEXT,
  notes        TEXT,
  folder_path  TEXT NOT NULL      -- relative path under applications/
);
```

### `application_status_history` table
Append-only. Status changes are made via the dashboard UI only (not files), so timestamps are reliable.

```sql
CREATE TABLE application_status_history (
  id             INTEGER PRIMARY KEY,
  application_id INTEGER NOT NULL REFERENCES applications(id),
  status         TEXT NOT NULL,   -- applied|assessment|interview|offer|rejected|withdrawn|ghosted
  changed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes          TEXT
);
```

---

## 4. MCP Tools

Exposed by `prospector mcp` to Claude Code:

| Tool | Parameters | Description |
|---|---|---|
| `list_cvs` | — | List base CV files with name and modified date |
| `read_cv` | `filename` | Read a base CV's markdown content |
| `write_cv` | `filename`, `content` | Save a new base CV file |
| `list_cover_letters` | — | List standalone cover letter files |
| `read_cover_letter` | `filename` | Read a cover letter's markdown content |
| `write_cover_letter` | `filename`, `content` | Save a new standalone cover letter file |
| `list_themes` | — | List available CSS theme names |
| `export_pdf` | `file_path`, `theme` | Render a markdown file to PDF via Gotenberg; saves to a temp file and returns the output path |
| `get_experience_stats` | — | Return years of experience totals and breakdown by role type |
| `create_application` | `company`, `role`, `date`, `cv_content`, `cover_letter_content`, `job_post_content` | Create full application folder structure with all files and metadata.toml |

---

## 5. Slash Commands (Skills)

Installed to `.claude/commands/` via the plugin install script.

### `/customize-cv`
Takes a job post (`@file` reference or URL) and optionally a target role type. Creates a new **base CV variant** (not job-specific — use `/apply` for that). Claude:
1. Calls `list_cvs` and selects the most relevant base CV for the role
2. Reads the job post content
3. Reads the base CV via `read_cv`
4. Tailors the CV content for that role type
5. Saves the result via `write_cv` as a new date-stamped base CV in `cv/`

### `/write-cover-letter`
Takes a job post and optionally a specific CV filename. Claude:
1. Reads the job post
2. Reads the specified CV (or selects the most relevant base CV)
3. Calls `get_experience_stats` for accurate years of experience
4. Generates a cover letter
5. Saves via `write_cover_letter` with a date-stamped filename

### `/apply`
The combined command. Takes a job post (`@file` or URL), company name, and role. Claude:
1. Reads the job post
2. Selects the most relevant base CV and reads it
3. Calls `get_experience_stats`
4. Tailors the CV and generates a cover letter in one pass
5. Calls `create_application` to save both files, the job post, and metadata to `applications/company/role/date/`

---

## 6. Web Dashboard

Served by `prospector serve`. Go renders HTML with `html/template`. HTMX handles dynamic interactions (no page reloads for status updates, theme switching, sync). Tailwind CSS via CDN for styling. No build step — all assets embedded in the binary.

### Pages

**Applications (home)**
- Table: company, role, date applied, current status
- Click row: application detail view with status history timeline
- Status update: dropdown + confirm → appends to `application_status_history` with current timestamp
- Links to view/export CV and cover letter for that application

**CV Library**
- List of base CVs in `cv/`
- View: rendered markdown with live theme picker (HTMX swaps preview on theme change)
- Export to PDF: pick theme → calls export endpoint → PDF download
- Create new: editor with empty template or clone of existing CV

**Cover Letter Library**
- Same as CV Library but for `cover-letters/`

**Experience Stats**
- Calculated from SQLite: total years, years per role type, work history timeline
- Sync button: triggers `prospector sync`, refreshes stats via HTMX

**Themes**
- List CSS files in `themes/`
- Preview a theme applied to a selected CV
- Upload new theme (CSS file upload)

### PDF Export Flow
1. User picks a file + theme in dashboard
2. POST to `/export` endpoint
3. Go binary: renders markdown → HTML, injects theme CSS, posts to Gotenberg REST API
4. Gotenberg returns PDF bytes
5. Browser receives PDF as download

---

## 7. Plugin Packaging

### Repository structure
```
prospector/
  cmd/prospector/         ← binary entrypoint
  internal/
    mcp/                  ← MCP server and tool handlers
    dashboard/            ← HTTP server, templates, handlers
    db/                   ← SQLite schema, queries
    pdf/                  ← Gotenberg client
    parser/               ← markdown parser, experience extractor
    config/               ← config.toml loader
  skills/
    customize-cv.md       ← slash command templates
    write-cover-letter.md
    apply.md
  themes/
    minimal.css
    modern.css
  docker-compose.yml      ← Gotenberg service definition
  plugin.json             ← Claude marketplace manifest
  install.sh              ← setup script
  README.md
```

### `plugin.json` (Claude marketplace manifest)
The exact manifest schema is defined by the Claude plugin marketplace. The fields below are illustrative — confirm against the official plugin publishing docs when packaging.

```json
{
  "name": "prospector",
  "version": "1.0.0",
  "description": "CV, cover letter, and job application manager for Claude Code",
  "author": "dev@graditya.com",
  "mcp": {
    "command": "prospector",
    "args": ["mcp"]
  },
  "skills": ["skills/customize-cv.md", "skills/write-cover-letter.md", "skills/apply.md"],
  "install": "install.sh"
}
```

### `install.sh` responsibilities
1. Create `~/.config/prospector/` directory structure
2. Copy default themes to `~/.config/prospector/themes/`
3. Create default `config.toml` if not present
4. Register MCP server in Claude Code's MCP config (`~/.claude/mcp.json` or project-level)
5. Print next steps: run `prospector up`, run `prospector serve`, add first CV

---

## 8. Analytics (from status history)

Queries the dashboard can run against `application_status_history`:
- Time from `applied` to first `interview` (average, per company/role type)
- Offer rate: applications reaching `offer` / total applications
- Ghost rate: applications stuck in `applied` with no movement after N days
- Funnel: count per status stage across all applications
- Applications per month over time
