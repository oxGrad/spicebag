# Board Jobs Implementation Design

## Goal

Replace the per-company ATS scraping flow with board-level scraping: fetch jobs from public job board APIs (Remotive, Remote OK, We Work Remotely, Jobicy), filter by role/skill/location preferences, and store results in a separate `board_jobs` table with its own dashboard page.

## Architecture

A new `BoardAdapter` interface sits alongside the existing `Adapter` interface. Board adapters take no token — boards are public. Results land in a new `board_jobs` table (separate from `scraped_jobs`). A `scrape_boards` config table controls which boards are enabled. Two new MCP tools (`fetch_board_jobs`, `save_board_jobs`) handle the board scraping flow. The existing company-scraping path (`scrape_companies`, `scraped_jobs`, `fetch_ats_jobs`) is untouched.

## Tech Stack

Go (board adapters, MCP tools, HTTP handlers), SQLite (two new tables), Vue 3 + Tailwind (new BoardJobsView page, Settings toggles).

---

## Data Model

### Migration 009

**`scrape_boards`** — one row per supported board:

```sql
CREATE TABLE scrape_boards (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  name               TEXT NOT NULL UNIQUE,
  label              TEXT NOT NULL,
  enabled            INTEGER NOT NULL DEFAULT 1,
  last_scraped_at    TEXT NOT NULL DEFAULT '',
  last_scrape_status TEXT NOT NULL DEFAULT 'never',
  last_scrape_error  TEXT NOT NULL DEFAULT '',
  last_job_count     INTEGER NOT NULL DEFAULT 0
);

INSERT INTO scrape_boards (name, label, enabled) VALUES
  ('remotive',       'Remotive',          1),
  ('remoteok',       'Remote OK',         1),
  ('weworkremotely', 'We Work Remotely',  1),
  ('jobicy',         'Jobicy',            1);
```

**`board_jobs`** — board-sourced job matches, separate from `scraped_jobs`:

```sql
CREATE TABLE board_jobs (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  source_board           TEXT NOT NULL,
  company_name           TEXT NOT NULL,
  title                  TEXT NOT NULL,
  location               TEXT NOT NULL DEFAULT '',
  url                    TEXT NOT NULL UNIQUE,
  match_reason           TEXT NOT NULL DEFAULT '',
  matched_skills         TEXT NOT NULL DEFAULT '',
  skill_score            INTEGER NOT NULL DEFAULT 0,
  status                 TEXT NOT NULL DEFAULT 'new',
  scraped_at             TEXT NOT NULL DEFAULT (datetime('now')),
  applied_application_id INTEGER REFERENCES applications(id)
);
```

---

## Go Backend

### New interface + struct (`internal/scrape/scrape.go`)

```go
type BoardJob struct {
    CompanyName string
    Job                  // embeds Title, Location, URL
}

type BoardAdapter interface {
    Name() string
    FetchJobs(ctx context.Context) ([]BoardJob, error)
}

func BoardRegistry() map[string]BoardAdapter { ... }
```

### New board adapter files

- `internal/scrape/remotive.go` — `GET https://remotive.com/api/remote-jobs` (JSON)
- `internal/scrape/remoteok.go` — `GET https://remoteok.com/api` (JSON)
- `internal/scrape/weworkremotely.go` — RSS feed (XML)
- `internal/scrape/jobicy.go` — `GET https://jobicy.com/api/v2/remote-jobs` (JSON)

Each implements `BoardAdapter`. Each has a corresponding `_test.go` using a local `httptest.Server` with fixture data in `testdata/`.

### DB methods (`internal/db/scrape.go`)

New structs and methods:

```go
type ScrapeBoard struct {
    ID               int64
    Name             string
    Label            string
    Enabled          bool
    LastScrapedAt    string
    LastScrapeStatus string
    LastScrapeError  string
    LastJobCount     int
}

type BoardJob struct {
    ID                  int64
    SourceBoard         string
    CompanyName         string
    Title               string
    Location            string
    URL                 string
    MatchReason         string
    MatchedSkills       string
    SkillScore          int
    Status              string
    ScrapedAt           string
    AppliedApplicationID *int64
}

func (s *Store) ListScrapeBoards() ([]ScrapeBoard, error)
func (s *Store) UpdateScrapeBoardStatus(id int64, at, status, errMsg string, count int) error
func (s *Store) ToggleScrapeBoard(id int64, enabled bool) error
func (s *Store) SaveBoardJobs(jobs []BoardJob) (int, error)   // upsert by URL, returns new count
func (s *Store) ListBoardJobs(status string) ([]BoardJob, error) // ORDER BY skill_score DESC, scraped_at DESC
func (s *Store) SetBoardJobStatus(id int64, status string) error
```

### MCP tools (`internal/mcp/scrape_tools.go`)

**`fetch_board_jobs`** — reads enabled boards from DB, calls each `BoardAdapter`, applies `HasRemoteSignal` pre-filter, records per-board scrape status, returns:
```json
{"jobs": [{"board": "remotive", "company_name": "...", "title": "...", "location": "...", "url": "..."}], "errors": [...]}
```

**`save_board_jobs`** — accepts JSON array of `{source_board, company_name, title, location, url, match_reason, matched_skills, skill_score}`, writes to `board_jobs`, returns `{new, already_seen}`.

**`get_scrape_preferences`** — adds `boards` field to response: list of enabled board names. Claude uses this to report which boards were active and to warn if all are disabled.

---

## Dashboard

### New API endpoints (`internal/dashboard/handlers_scrape.go`)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/board-jobs` | List board jobs (`?status=new\|dismissed`) |
| `PATCH` | `/api/board-jobs/:id/status` | Set job status |
| `GET` | `/api/scrape/boards` | List boards with scrape status |
| `PATCH` | `/api/scrape/boards/:id/toggle` | Enable/disable a board |

### New Vue page (`frontend/src/views/BoardJobsView.vue`)

Table columns: **Source** (board pill badge), **Company**, **Role** (with skill badges), **Location**, **Why matched**, **Found**, **Action**.

Same dismiss/restore and Apply (`/apply <url>`) flow as `JobsView.vue`. Warning banner if any boards failed last scrape. Hint text: `Run /scrape-jobs in Claude Code to refresh this list.`

Route: `/board-jobs`. Nav link: **"Board Jobs"** added to sidebar.

### Settings page (`frontend/src/views/SettingsView.vue`)

New **"Job Boards"** section (between company and location sections): toggle switch per board, showing label, last scrape status, and last job count. Uses `GET /api/scrape/boards` and `PATCH /api/scrape/boards/:id/toggle`.

---

## Skill (`plugins/skills/scrape-jobs.md`)

Updated flow:

1. `get_scrape_preferences` — stop early if `boards` list is empty (all disabled): "Enable at least one job board in Settings → Job Boards."
2. `fetch_board_jobs` — replaces `fetch_ats_jobs` as the primary fetch call.
3. Judge role/skill/location fit — unchanged.
4. `save_board_jobs` — replaces `save_scraped_jobs`.
5. Report — directs user to **Board Jobs** page.

The existing company-scraping tools (`fetch_ats_jobs`, `save_scraped_jobs`) remain available but are no longer part of the default skill flow.
