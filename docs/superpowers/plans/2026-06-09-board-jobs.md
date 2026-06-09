# Board Jobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add board-level job scraping (Remotive, Remote OK, We Work Remotely, Jobicy) that fetches jobs across all companies on each board, stores results in a separate `board_jobs` table, and shows them in a new "Board Jobs" dashboard page.

**Architecture:** Four new board adapters implement a new `BoardAdapter` interface (no token — public APIs). A `scrape_boards` config table controls which boards are enabled with toggle UI in Settings. New MCP tools `fetch_board_jobs`/`save_board_jobs` replace `fetch_ats_jobs`/`save_scraped_jobs` as the default skill flow. The existing company-scraping path is untouched.

**Tech Stack:** Go (adapters, MCP tools, HTTP handlers), SQLite (golang-migrate), Vue 3 + Tailwind CSS.

---

## File Map

**Create:**
- `internal/db/migrations/000009_add_board_jobs.up.sql`
- `internal/db/migrations/000009_add_board_jobs.down.sql`
- `internal/scrape/remotive.go` + `remotive_test.go`
- `internal/scrape/remoteok.go` + `remoteok_test.go`
- `internal/scrape/weworkremotely.go` + `weworkremotely_test.go`
- `internal/scrape/jobicy.go` + `jobicy_test.go`
- `internal/scrape/testdata/remotive.json`
- `internal/scrape/testdata/remoteok.json`
- `internal/scrape/testdata/weworkremotely.xml`
- `internal/scrape/testdata/jobicy.json`
- `frontend/src/views/BoardJobsView.vue`

**Modify:**
- `internal/scrape/scrape.go` — add `BoardJob` struct + `BoardAdapter` interface + `BoardRegistry()`
- `internal/db/scrape.go` — add `ScrapeBoard` + `BoardJob` structs + 6 new methods
- `internal/db/scrape_test.go` — add board + board_jobs tests
- `internal/mcp/scrape_tools.go` — add `registerFetchBoardJobs`, `registerSaveBoardJobs`; update `get_scrape_preferences`; call new registrations from `registerScrapeTools`
- `internal/mcp/scrape_tools_test.go` — add board MCP tool tests
- `internal/dashboard/handlers_scrape.go` — add 4 new handlers
- `internal/dashboard/handlers_scrape_test.go` — add board handler tests
- `internal/dashboard/server.go` — add 4 new routes
- `frontend/src/api.js` — add `boards` + `boardJobs` namespaces
- `frontend/src/views/SettingsView.vue` — add Job Boards toggle section
- `frontend/src/router/index.js` — add `/board-jobs` route
- `frontend/src/App.vue` — add Board Jobs nav link
- `plugins/skills/scrape-jobs.md` — update to use board tools

---

## Task 1: Migration 009 — scrape_boards + board_jobs tables

**Files:**
- Create: `internal/db/migrations/000009_add_board_jobs.up.sql`
- Create: `internal/db/migrations/000009_add_board_jobs.down.sql`

- [ ] **Step 1: Write the up migration**

```sql
-- internal/db/migrations/000009_add_board_jobs.up.sql
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

- [ ] **Step 2: Write the down migration**

```sql
-- internal/db/migrations/000009_add_board_jobs.down.sql
DROP TABLE board_jobs;
DROP TABLE scrape_boards;
```

- [ ] **Step 3: Verify migration runs**

Run: `just test`
Expected: PASS (migrations auto-run on `db.Open` in tests)

- [ ] **Step 4: Commit**

```bash
git add internal/db/migrations/000009_add_board_jobs.up.sql \
        internal/db/migrations/000009_add_board_jobs.down.sql
git commit -m "feat: migration 009 — scrape_boards + board_jobs tables"
```

---

## Task 2: DB layer — ScrapeBoard + BoardJob CRUD

**Files:**
- Modify: `internal/db/scrape.go`
- Modify: `internal/db/scrape_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/db/scrape_test.go`:

```go
func TestScrapeBoardsSeededAndToggle(t *testing.T) {
	store := openTestStore(t)

	boards, err := store.ListScrapeBoards()
	require.NoError(t, err)
	require.Len(t, boards, 4) // seeded by migration

	var remotiveID int64
	for _, b := range boards {
		if b.Name == "remotive" {
			remotiveID = b.ID
			assert.True(t, b.Enabled)
			assert.Equal(t, "Remotive", b.Label)
			assert.Equal(t, "never", b.LastScrapeStatus)
		}
	}
	require.NotZero(t, remotiveID)

	require.NoError(t, store.ToggleScrapeBoard(remotiveID, false))
	boards, _ = store.ListScrapeBoards()
	for _, b := range boards {
		if b.ID == remotiveID {
			assert.False(t, b.Enabled)
		}
	}

	require.NoError(t, store.UpdateScrapeBoardStatus(remotiveID, "2026-06-09 10:00:00", "ok", "", 42))
	boards, _ = store.ListScrapeBoards()
	for _, b := range boards {
		if b.ID == remotiveID {
			assert.Equal(t, "ok", b.LastScrapeStatus)
			assert.Equal(t, 42, b.LastJobCount)
		}
	}
}

func TestBoardJobsSaveAndList(t *testing.T) {
	store := openTestStore(t)

	jobs := []db.BoardJob{
		{SourceBoard: "remotive", CompanyName: "Acme", Title: "SRE", Location: "Worldwide", URL: "https://remotive.com/1", MatchReason: "worldwide remote", MatchedSkills: "Go", SkillScore: 1},
		{SourceBoard: "remoteok", CompanyName: "Beta", Title: "DevOps Engineer", Location: "Remote", URL: "https://remoteok.com/2", MatchReason: "worldwide remote"},
	}

	added, err := store.SaveBoardJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	// duplicate URL is ignored
	added2, err := store.SaveBoardJobs(jobs[:1])
	require.NoError(t, err)
	assert.Equal(t, 0, added2)

	list, err := store.ListBoardJobs("new")
	require.NoError(t, err)
	require.Len(t, list, 2)
	// ordered by skill_score DESC
	assert.Equal(t, "SRE", list[0].Title)
	assert.Equal(t, "Go", list[0].MatchedSkills)

	require.NoError(t, store.SetBoardJobStatus(list[1].ID, "dismissed"))
	newList, _ := store.ListBoardJobs("new")
	assert.Len(t, newList, 1)
	dismissed, _ := store.ListBoardJobs("dismissed")
	assert.Len(t, dismissed, 1)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/... -v -run TestScrapeBoards`
Expected: FAIL — `store.ListScrapeBoards` undefined

- [ ] **Step 3: Add ScrapeBoard + BoardJob structs and methods to internal/db/scrape.go**

Append after `SetScrapedJobStatus`:

```go
type ScrapeBoard struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	Label            string `json:"label"`
	Enabled          bool   `json:"enabled"`
	LastScrapedAt    string `json:"last_scraped_at"`
	LastScrapeStatus string `json:"last_scrape_status"`
	LastScrapeError  string `json:"last_scrape_error"`
	LastJobCount     int    `json:"last_job_count"`
}

func (s *Store) ListScrapeBoards() ([]ScrapeBoard, error) {
	rows, err := s.db.Query(
		`SELECT id, name, label, enabled, last_scraped_at, last_scrape_status, last_scrape_error, last_job_count
		 FROM scrape_boards ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScrapeBoard
	for rows.Next() {
		var b ScrapeBoard
		var enabled int
		if err := rows.Scan(&b.ID, &b.Name, &b.Label, &enabled,
			&b.LastScrapedAt, &b.LastScrapeStatus, &b.LastScrapeError, &b.LastJobCount); err != nil {
			return nil, err
		}
		b.Enabled = enabled == 1
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) UpdateScrapeBoardStatus(id int64, scrapedAt, status, errMsg string, jobCount int) error {
	_, err := s.db.Exec(
		`UPDATE scrape_boards SET last_scraped_at=?, last_scrape_status=?, last_scrape_error=?, last_job_count=? WHERE id=?`,
		scrapedAt, status, errMsg, jobCount, id,
	)
	return err
}

func (s *Store) ToggleScrapeBoard(id int64, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE scrape_boards SET enabled=? WHERE id=?`, v, id)
	return err
}

type BoardJob struct {
	ID                   int64         `json:"id"`
	SourceBoard          string        `json:"source_board"`
	CompanyName          string        `json:"company_name"`
	Title                string        `json:"title"`
	Location             string        `json:"location"`
	URL                  string        `json:"url"`
	MatchReason          string        `json:"match_reason"`
	MatchedSkills        string        `json:"matched_skills"`
	SkillScore           int           `json:"skill_score"`
	Status               string        `json:"status"`
	ScrapedAt            string        `json:"scraped_at"`
	AppliedApplicationID sql.NullInt64 `json:"applied_application_id"`
}

func (s *Store) SaveBoardJobs(jobs []BoardJob) (int, error) {
	added := 0
	for _, j := range jobs {
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO board_jobs
			   (source_board, company_name, title, location, url, match_reason, matched_skills, skill_score)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			j.SourceBoard, j.CompanyName, j.Title, j.Location, j.URL, j.MatchReason, j.MatchedSkills, j.SkillScore,
		)
		if err != nil {
			return added, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			added++
		}
	}
	return added, nil
}

func (s *Store) ListBoardJobs(status string) ([]BoardJob, error) {
	rows, err := s.db.Query(
		`SELECT id, source_board, company_name, title, location, url, match_reason,
		        matched_skills, skill_score, status, scraped_at, applied_application_id
		 FROM board_jobs WHERE status=?
		 ORDER BY skill_score DESC, scraped_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BoardJob
	for rows.Next() {
		var j BoardJob
		if err := rows.Scan(&j.ID, &j.SourceBoard, &j.CompanyName, &j.Title, &j.Location,
			&j.URL, &j.MatchReason, &j.MatchedSkills, &j.SkillScore,
			&j.Status, &j.ScrapedAt, &j.AppliedApplicationID); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) SetBoardJobStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE board_jobs SET status=? WHERE id=?`, status, id)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/db/... -v -run "TestScrapeBoards|TestBoardJobs"`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/db/scrape.go internal/db/scrape_test.go
git commit -m "feat: DB layer — ScrapeBoard + BoardJob CRUD"
```

---

## Task 3: scrape.go — BoardJob struct + BoardAdapter interface + BoardRegistry

**Files:**
- Modify: `internal/scrape/scrape.go`

- [ ] **Step 1: Add BoardJob, BoardAdapter, and BoardRegistry to internal/scrape/scrape.go**

Add after the existing `Registry()` function:

```go
// BoardJob is a single vacancy as returned by a board adapter.
type BoardJob struct {
	CompanyName string
	Title       string
	Location    string
	URL         string
}

// BoardAdapter fetches jobs from a public job board with no per-company token.
type BoardAdapter interface {
	Name() string
	FetchJobs(ctx context.Context) ([]BoardJob, error)
}

// BoardRegistry returns all supported board adapters keyed by their name.
func BoardRegistry() map[string]BoardAdapter {
	return map[string]BoardAdapter{
		"remotive":       Remotive{},
		"remoteok":       RemoteOK{},
		"weworkremotely": WeWorkRemotely{},
		"jobicy":         Jobicy{},
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/scrape/...`
Expected: FAIL — Remotive, RemoteOK, WeWorkRemotely, Jobicy undefined (correct; adapters don't exist yet)

- [ ] **Step 3: Stub the four adapters so it compiles**

Create `internal/scrape/remotive.go`:
```go
package scrape

import "context"

type Remotive struct{ BaseURL string }

func (r Remotive) Name() string                              { return "remotive" }
func (r Remotive) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
```

Create `internal/scrape/remoteok.go`:
```go
package scrape

import "context"

type RemoteOK struct{ BaseURL string }

func (r RemoteOK) Name() string                              { return "remoteok" }
func (r RemoteOK) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
```

Create `internal/scrape/weworkremotely.go`:
```go
package scrape

import "context"

type WeWorkRemotely struct{ BaseURL string }

func (w WeWorkRemotely) Name() string                              { return "weworkremotely" }
func (w WeWorkRemotely) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
```

Create `internal/scrape/jobicy.go`:
```go
package scrape

import "context"

type Jobicy struct{ BaseURL string }

func (j Jobicy) Name() string                              { return "jobicy" }
func (j Jobicy) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
```

- [ ] **Step 4: Verify it compiles**

Run: `go build ./internal/scrape/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scrape/scrape.go internal/scrape/remotive.go \
        internal/scrape/remoteok.go internal/scrape/weworkremotely.go \
        internal/scrape/jobicy.go
git commit -m "feat: BoardAdapter interface + stub adapters"
```

---

## Task 4: Remotive adapter

**Files:**
- Modify: `internal/scrape/remotive.go`
- Create: `internal/scrape/remotive_test.go`
- Create: `internal/scrape/testdata/remotive.json`

- [ ] **Step 1: Create testdata**

`internal/scrape/testdata/remotive.json`:
```json
{
  "job-count": 2,
  "jobs": [
    {
      "id": 1001,
      "url": "https://remotive.com/remote-jobs/devops/senior-sre-1001",
      "title": "Senior SRE",
      "company_name": "Acme Corp",
      "candidate_required_location": "Worldwide"
    },
    {
      "id": 1002,
      "url": "https://remotive.com/remote-jobs/marketing/office-manager-1002",
      "title": "Office Manager",
      "company_name": "Beta Corp",
      "candidate_required_location": "New York, NY"
    }
  ]
}
```

- [ ] **Step 2: Write the failing test**

`internal/scrape/remotive_test.go`:
```go
package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemotiveParse(t *testing.T) {
	data, err := os.ReadFile("testdata/remotive.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Remotive{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Senior SRE", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://remotive.com/remote-jobs/devops/senior-sre-1001", jobs[0].URL)
	assert.Equal(t, "Office Manager", jobs[1].Title)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/scrape/... -v -run TestRemotiveParse`
Expected: FAIL — returns nil jobs (stub)

- [ ] **Step 4: Implement the Remotive adapter**

Replace `internal/scrape/remotive.go`:
```go
package scrape

import "context"

// Remotive reads the public remote jobs API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Remotive struct{ BaseURL string }

func (r Remotive) Name() string { return "remotive" }

func (r Remotive) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := r.BaseURL
	if base == "" {
		base = "https://remotive.com/api/remote-jobs"
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			URL      string `json:"url"`
			Company  string `json:"company_name"`
			Location string `json:"candidate_required_location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	out := make([]BoardJob, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		out = append(out, BoardJob{CompanyName: j.Company, Title: j.Title, Location: j.Location, URL: j.URL})
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/scrape/... -v -run TestRemotiveParse`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/remotive.go internal/scrape/remotive_test.go \
        internal/scrape/testdata/remotive.json
git commit -m "feat: Remotive board adapter"
```

---

## Task 5: Remote OK adapter

**Files:**
- Modify: `internal/scrape/remoteok.go`
- Create: `internal/scrape/remoteok_test.go`
- Create: `internal/scrape/testdata/remoteok.json`

- [ ] **Step 1: Create testdata**

`internal/scrape/testdata/remoteok.json`:
```json
[
  {"legal": "Remote OK is a remote job board. By accessing this API you agree to our terms of service."},
  {
    "id": "2001",
    "url": "https://remoteok.com/remote-jobs/2001",
    "position": "DevOps Engineer",
    "company": "Acme Corp",
    "location": "Worldwide"
  },
  {
    "id": "2002",
    "url": "https://remoteok.com/remote-jobs/2002",
    "position": "Backend Engineer",
    "company": "Beta Corp",
    "location": "Remote"
  }
]
```

- [ ] **Step 2: Write the failing test**

`internal/scrape/remoteok_test.go`:
```go
package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteOKParse(t *testing.T) {
	data, err := os.ReadFile("testdata/remoteok.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := RemoteOK{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].Company)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://remoteok.com/remote-jobs/2001", jobs[0].URL)
}
```

Wait — `RemoteOK` returns `BoardJob` which has `CompanyName`, not `Company`. Fix the test:

`internal/scrape/remoteok_test.go`:
```go
package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoteOKParse(t *testing.T) {
	data, err := os.ReadFile("testdata/remoteok.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := RemoteOK{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://remoteok.com/remote-jobs/2001", jobs[0].URL)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/scrape/... -v -run TestRemoteOKParse`
Expected: FAIL — returns nil jobs (stub)

- [ ] **Step 4: Implement the Remote OK adapter**

Replace `internal/scrape/remoteok.go`:
```go
package scrape

import (
	"context"
	"encoding/json"
	"fmt"
)

// RemoteOK reads the public API. The first array element is a legal notice and is skipped.
// BaseURL is overridable in tests; empty means the real endpoint.
type RemoteOK struct{ BaseURL string }

func (r RemoteOK) Name() string { return "remoteok" }

func (r RemoteOK) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := r.BaseURL
	if base == "" {
		base = "https://remoteok.com/api"
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) < 2 {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	var out []BoardJob
	for _, item := range raw[1:] { // first element is a legal notice object
		var j struct {
			Position string `json:"position"`
			Company  string `json:"company"`
			Location string `json:"location"`
			URL      string `json:"url"`
		}
		if err := json.Unmarshal(item, &j); err != nil || j.URL == "" {
			continue
		}
		out = append(out, BoardJob{CompanyName: j.Company, Title: j.Position, Location: j.Location, URL: j.URL})
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/scrape/... -v -run TestRemoteOKParse`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/remoteok.go internal/scrape/remoteok_test.go \
        internal/scrape/testdata/remoteok.json
git commit -m "feat: Remote OK board adapter"
```

---

## Task 6: We Work Remotely adapter

**Files:**
- Modify: `internal/scrape/weworkremotely.go`
- Create: `internal/scrape/weworkremotely_test.go`
- Create: `internal/scrape/testdata/weworkremotely.xml`

- [ ] **Step 1: Create testdata**

`internal/scrape/testdata/weworkremotely.xml`:
```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>We Work Remotely: Remote Jobs</title>
    <item>
      <title><![CDATA[Acme Corp: Senior DevOps Engineer]]></title>
      <link>https://weworkremotely.com/remote-jobs/view/3001</link>
      <region><![CDATA[Worldwide]]></region>
    </item>
    <item>
      <title><![CDATA[Beta Corp: Marketing Manager]]></title>
      <link>https://weworkremotely.com/remote-jobs/view/3002</link>
      <region><![CDATA[USA Only]]></region>
    </item>
  </channel>
</rss>
```

- [ ] **Step 2: Write the failing test**

`internal/scrape/weworkremotely_test.go`:
```go
package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWeWorkRemotelyParse(t *testing.T) {
	data, err := os.ReadFile("testdata/weworkremotely.xml")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := WeWorkRemotely{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Senior DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://weworkremotely.com/remote-jobs/view/3001", jobs[0].URL)
	assert.Equal(t, "Beta Corp", jobs[1].CompanyName)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/scrape/... -v -run TestWeWorkRemotelyParse`
Expected: FAIL — returns nil jobs (stub)

- [ ] **Step 4: Implement the We Work Remotely adapter**

Replace `internal/scrape/weworkremotely.go`:
```go
package scrape

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// WeWorkRemotely reads the public RSS feed.
// BaseURL is overridable in tests; empty means the real endpoint.
// Item titles have the format "Company Name: Job Title".
type WeWorkRemotely struct{ BaseURL string }

func (w WeWorkRemotely) Name() string { return "weworkremotely" }

func (w WeWorkRemotely) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := w.BaseURL
	if base == "" {
		base = "https://weworkremotely.com/remote-jobs.rss"
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Items []struct {
			Title  string `xml:"title"`
			Link   string `xml:"link"`
			Region string `xml:"region"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	var out []BoardJob
	for _, item := range doc.Items {
		company, title := splitWWRTitle(item.Title)
		if title == "" {
			continue
		}
		out = append(out, BoardJob{CompanyName: company, Title: title, Location: item.Region, URL: item.Link})
	}
	return out, nil
}

// splitWWRTitle splits "Company Name: Job Title" into its two parts.
func splitWWRTitle(raw string) (company, title string) {
	idx := strings.Index(raw, ": ")
	if idx < 0 {
		return "", strings.TrimSpace(raw)
	}
	return strings.TrimSpace(raw[:idx]), strings.TrimSpace(raw[idx+2:])
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/scrape/... -v -run TestWeWorkRemotelyParse`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/weworkremotely.go internal/scrape/weworkremotely_test.go \
        internal/scrape/testdata/weworkremotely.xml
git commit -m "feat: We Work Remotely board adapter"
```

---

## Task 7: Jobicy adapter

**Files:**
- Modify: `internal/scrape/jobicy.go`
- Create: `internal/scrape/jobicy_test.go`
- Create: `internal/scrape/testdata/jobicy.json`

- [ ] **Step 1: Create testdata**

`internal/scrape/testdata/jobicy.json`:
```json
{
  "status": "success",
  "jobs": [
    {
      "id": 4001,
      "url": "https://jobicy.com/jobs/4001-platform-engineer",
      "jobTitle": "Platform Engineer",
      "companyName": "Acme Corp",
      "jobGeo": "Worldwide"
    },
    {
      "id": 4002,
      "url": "https://jobicy.com/jobs/4002-sales-manager",
      "jobTitle": "Sales Manager",
      "companyName": "Beta Corp",
      "jobGeo": "Remote"
    }
  ]
}
```

- [ ] **Step 2: Write the failing test**

`internal/scrape/jobicy_test.go`:
```go
package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobicyParse(t *testing.T) {
	data, err := os.ReadFile("testdata/jobicy.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Jobicy{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://jobicy.com/jobs/4001-platform-engineer", jobs[0].URL)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/scrape/... -v -run TestJobicyParse`
Expected: FAIL — returns nil jobs (stub)

- [ ] **Step 4: Implement the Jobicy adapter**

Replace `internal/scrape/jobicy.go`:
```go
package scrape

import "context"

// Jobicy reads the public remote jobs API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Jobicy struct{ BaseURL string }

func (j Jobicy) Name() string { return "jobicy" }

func (j Jobicy) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := j.BaseURL
	if base == "" {
		base = "https://jobicy.com/api/v2/remote-jobs"
	}
	var resp struct {
		Jobs []struct {
			Title   string `json:"jobTitle"`
			Company string `json:"companyName"`
			Geo     string `json:"jobGeo"`
			URL     string `json:"url"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	out := make([]BoardJob, 0, len(resp.Jobs))
	for _, job := range resp.Jobs {
		out = append(out, BoardJob{CompanyName: job.Company, Title: job.Title, Location: job.Geo, URL: job.URL})
	}
	return out, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/scrape/... -v -run TestJobicyParse`
Expected: PASS

- [ ] **Step 6: Run all scrape tests**

Run: `go test ./internal/scrape/... -v`
Expected: PASS (all adapter tests pass)

- [ ] **Step 7: Commit**

```bash
git add internal/scrape/jobicy.go internal/scrape/jobicy_test.go \
        internal/scrape/testdata/jobicy.json
git commit -m "feat: Jobicy board adapter"
```

---

## Task 8: MCP tools — fetch_board_jobs + save_board_jobs + update get_scrape_preferences

**Files:**
- Modify: `internal/mcp/scrape_tools.go`
- Modify: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/mcp/scrape_tools_test.go`:

```go
func TestGetScrapePreferencesIncludesBoards(t *testing.T) {
	_, srv := setup(t)

	out, err := srv.CallTool(context.Background(), "get_scrape_preferences", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Boards []string `json:"boards"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	// migration seeds 4 boards all enabled
	assert.Len(t, got.Boards, 4)
	assert.Contains(t, got.Boards, "Remotive")
}

func TestSaveBoardJobs(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()

	jobsJSON := `[
		{"source_board":"remotive","company_name":"Acme","title":"SRE","location":"Worldwide","url":"https://remotive.com/1","match_reason":"worldwide remote","matched_skills":"Go","skill_score":1},
		{"source_board":"remoteok","company_name":"Beta","title":"DevOps","location":"Remote","url":"https://remoteok.com/2","match_reason":"worldwide remote"}
	]`

	out, err := srv.CallTool(context.Background(), "save_board_jobs", map[string]any{"jobs": jobsJSON})
	require.NoError(t, err)
	assert.Contains(t, out, `"new":2`)

	jobs, _ := store.ListBoardJobs("new")
	assert.Len(t, jobs, 2)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcp/... -v -run "TestGetScrapePreferencesIncludesBoards|TestSaveBoardJobs"`
Expected: FAIL — tools not yet registered

- [ ] **Step 3: Update get_scrape_preferences to include boards**

In `internal/mcp/scrape_tools.go`, find the `get_scrape_preferences` handler. After fetching skills and before marshalling the payload, add:

```go
boards, err := s.store.ListScrapeBoards()
if err != nil {
    return mcplib.NewToolResultError(err.Error()), nil
}
var enabledBoards []string
for _, b := range boards {
    if b.Enabled {
        enabledBoards = append(enabledBoards, b.Label)
    }
}
if enabledBoards == nil {
    enabledBoards = []string{}
}
```

Then add `"boards": enabledBoards` to the `payload` map:

```go
payload := map[string]any{
    "companies":      companies,
    "roles":          roles,
    "skills":         skills,
    "boards":         enabledBoards,
    "home_timezone":  prefs.HomeTimezone,
    "location_notes": prefs.LocationNotes,
}
```

Also update the `get_scrape_preferences` description to mention boards:

```go
mcplib.WithDescription("Return the user's saved job-scraping preferences: companies, target roles, target skills, enabled job boards, and location preferences (home timezone + notes) used to judge job fit."),
```

- [ ] **Step 4: Add registerFetchBoardJobs to internal/mcp/scrape_tools.go**

Add a new method after `registerFetchATSJobs`:

```go
func (s *Server) registerFetchBoardJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"fetch_board_jobs",
			mcplib.WithDescription("Fetch current job listings from all enabled job boards (Remotive, Remote OK, We Work Remotely, Jobicy). Returns a compact list of {board, company_name, title, location, url} plus per-board errors. Records each board's scrape status. Apply timezone/region/role/skill judgment to the returned list, then call save_board_jobs with the matches."),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			boards, err := s.store.ListScrapeBoards()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			reg := scrape.BoardRegistry()

			type outJob struct {
				Board       string `json:"board"`
				CompanyName string `json:"company_name"`
				Title       string `json:"title"`
				Location    string `json:"location"`
				URL         string `json:"url"`
			}
			type outErr struct {
				Board string `json:"board"`
				Error string `json:"error"`
			}
			var jobs []outJob
			var errs []outErr

			first := true
			for _, b := range boards {
				if !b.Enabled {
					continue
				}
				if !first {
					time.Sleep(jitter())
				}
				first = false
				now := time.Now().Format("2006-01-02 15:04:05")
				adapter, ok := reg[b.Name]
				if !ok {
					msg := "Unsupported board: " + b.Name
					s.store.UpdateScrapeBoardStatus(b.ID, now, "error", msg, 0) //nolint:errcheck
					errs = append(errs, outErr{Board: b.Label, Error: msg})
					continue
				}
				fetched, ferr := adapter.FetchJobs(ctx)
				if ferr != nil {
					msg := scrape.ClassifyError(b.Name, ferr)
					s.store.UpdateScrapeBoardStatus(b.ID, now, "error", msg, 0) //nolint:errcheck
					errs = append(errs, outErr{Board: b.Label, Error: msg})
					continue
				}
				kept := 0
				for _, j := range fetched {
					if !scrape.HasRemoteSignal(j.Location) {
						continue
					}
					jobs = append(jobs, outJob{
						Board: b.Name, CompanyName: j.CompanyName,
						Title: j.Title, Location: j.Location, URL: j.URL,
					})
					kept++
				}
				s.store.UpdateScrapeBoardStatus(b.ID, now, "ok", "", kept) //nolint:errcheck
			}

			if jobs == nil {
				jobs = []outJob{}
			}
			payload := map[string]any{"jobs": jobs, "errors": errs}
			b, _ := json.Marshal(payload)
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}
```

- [ ] **Step 5: Add registerSaveBoardJobs to internal/mcp/scrape_tools.go**

Add after `registerFetchBoardJobs`:

```go
func (s *Server) registerSaveBoardJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"save_board_jobs",
			mcplib.WithDescription("Save matched board jobs (those that pass the user's timezone/region/role/skill rules). Jobs whose URL already exists are ignored. Returns counts of new vs already-seen."),
			mcplib.WithString("jobs", mcplib.Required(),
				mcplib.Description(`JSON array of {"source_board": "remotive", "company_name": "...", "title": "...", "location": "...", "url": "...", "match_reason": "...", "matched_skills": "...", "skill_score": <int>}`)),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			jobsJSON := req.GetString("jobs", "")
			var entries []struct {
				SourceBoard   string `json:"source_board"`
				CompanyName   string `json:"company_name"`
				Title         string `json:"title"`
				Location      string `json:"location"`
				URL           string `json:"url"`
				MatchReason   string `json:"match_reason"`
				MatchedSkills string `json:"matched_skills"`
				SkillScore    int    `json:"skill_score"`
			}
			if err := json.Unmarshal([]byte(jobsJSON), &entries); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("invalid jobs JSON: %v", err)), nil
			}
			var jobs []db.BoardJob
			for _, e := range entries {
				jobs = append(jobs, db.BoardJob{
					SourceBoard:   e.SourceBoard,
					CompanyName:   e.CompanyName,
					Title:         e.Title,
					Location:      e.Location,
					URL:           e.URL,
					MatchReason:   e.MatchReason,
					MatchedSkills: e.MatchedSkills,
					SkillScore:    e.SkillScore,
				})
			}
			added, err := s.store.SaveBoardJobs(jobs)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			b, _ := json.Marshal(map[string]any{"new": added, "already_seen": len(jobs) - added})
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}
```

- [ ] **Step 6: Register both new tools in registerScrapeTools**

In `registerScrapeTools`, add the two calls:

```go
func (s *Server) registerScrapeTools() {
    // ... existing get_scrape_preferences tool registration ...
    s.registerFetchATSJobs()
    s.registerSaveScrapedJobs()
    s.registerFetchBoardJobs()
    s.registerSaveBoardJobs()
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `go test ./internal/mcp/... -v -run "TestGetScrapePreferencesIncludesBoards|TestSaveBoardJobs"`
Expected: PASS

- [ ] **Step 8: Run full test suite**

Run: `just test`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/mcp/scrape_tools.go internal/mcp/scrape_tools_test.go
git commit -m "feat: fetch_board_jobs + save_board_jobs MCP tools; add boards to get_scrape_preferences"
```

---

## Task 9: Dashboard handlers + routes

**Files:**
- Modify: `internal/dashboard/handlers_scrape.go`
- Modify: `internal/dashboard/handlers_scrape_test.go`
- Modify: `internal/dashboard/server.go`

- [ ] **Step 1: Write the failing tests**

Add to `internal/dashboard/handlers_scrape_test.go`:

```go
func TestBoardToggleAndList(t *testing.T) {
	srv := newTestServer(t)

	// list returns all 4 seeded boards
	req := httptest.NewRequest(http.MethodGet, "/api/scrape/boards", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var boards []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &boards))
	require.Len(t, boards, 4)

	// find remotive id
	var remotiveID float64
	for _, b := range boards {
		if b["name"] == "remotive" {
			remotiveID = b["id"].(float64)
			assert.Equal(t, true, b["enabled"])
		}
	}
	require.NotZero(t, remotiveID)

	// toggle off
	form := strings.NewReader("enabled=0")
	toggleReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/scrape/boards/%d/toggle", int(remotiveID)), form)
	toggleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tw := httptest.NewRecorder()
	srv.ServeHTTP(tw, toggleReq)
	assert.Equal(t, http.StatusNoContent, tw.Code)

	// verify disabled
	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/boards", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	var boards2 []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &boards2)
	for _, b := range boards2 {
		if b["name"] == "remotive" {
			assert.Equal(t, false, b["enabled"])
		}
	}
}

func TestBoardJobsListAndStatus(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	store.SaveBoardJobs([]db.BoardJob{
		{SourceBoard: "remotive", CompanyName: "Acme", Title: "SRE", Location: "Worldwide", URL: "https://remotive.com/1", MatchReason: "worldwide"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/board-jobs?status=new", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SRE")

	var jobs []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobs))
	id := int(jobs[0]["id"].(float64))

	form := strings.NewReader("status=dismissed")
	statusReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/board-jobs/%d/status", id), form)
	statusReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sw := httptest.NewRecorder()
	srv.ServeHTTP(sw, statusReq)
	assert.Equal(t, http.StatusNoContent, sw.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/board-jobs?status=new", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, "[]", strings.TrimSpace(w2.Body.String()))
}
```

These tests use `db.BoardJob` — add `"github.com/oxGrad/spicebag/internal/db"` to the import in `handlers_scrape_test.go` if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/dashboard/... -v -run "TestBoardToggle|TestBoardJobs"`
Expected: FAIL — routes not registered

- [ ] **Step 3: Add four handlers to internal/dashboard/handlers_scrape.go**

Append to `internal/dashboard/handlers_scrape.go`:

```go
func (s *Server) handleAPIBoardsList(w http.ResponseWriter, r *http.Request) {
	bs, err := s.store.ListScrapeBoards()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bs == nil {
		bs = []db.ScrapeBoard{}
	}
	writeJSON(w, bs)
}

func (s *Server) handleAPIBoardToggle(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	enabled := r.FormValue("enabled") == "1"
	if err := s.store.ToggleScrapeBoard(id, enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIBoardJobsList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "new"
	}
	jobs, err := s.store.ListBoardJobs(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []db.BoardJob{}
	}
	writeJSON(w, jobs)
}

func (s *Server) handleAPIBoardJobStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")
	switch status {
	case "new", "dismissed":
	default:
		http.Error(w, "status must be new or dismissed", http.StatusBadRequest)
		return
	}
	if err := s.store.SetBoardJobStatus(id, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Register the routes in internal/dashboard/server.go**

Add these four lines after the existing scrape routes (after `s.mux.HandleFunc("POST /api/scrape/jobs/{id}/status", ...)`):

```go
s.mux.HandleFunc("GET /api/scrape/boards", s.handleAPIBoardsList)
s.mux.HandleFunc("POST /api/scrape/boards/{id}/toggle", s.handleAPIBoardToggle)
s.mux.HandleFunc("GET /api/board-jobs", s.handleAPIBoardJobsList)
s.mux.HandleFunc("POST /api/board-jobs/{id}/status", s.handleAPIBoardJobStatus)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/dashboard/... -v -run "TestBoardToggle|TestBoardJobs"`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `just test`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/handlers_scrape.go \
        internal/dashboard/handlers_scrape_test.go \
        internal/dashboard/server.go
git commit -m "feat: board jobs dashboard handlers + routes"
```

---

## Task 10: Frontend — api.js + BoardJobsView + Settings boards + router + nav

**Files:**
- Modify: `frontend/src/api.js`
- Create: `frontend/src/views/BoardJobsView.vue`
- Modify: `frontend/src/views/SettingsView.vue`
- Modify: `frontend/src/router/index.js`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add boards + boardJobs to frontend/src/api.js**

In `api.js`, add two new namespaces inside the `export const api = { ... }` object, after the `scrape` block:

```js
boards: {
  list: () => get("/scrape/boards"),
  toggle: (id, enabled) => post(`/scrape/boards/${id}/toggle`, { enabled: enabled ? "1" : "0" }),
},
boardJobs: {
  list: (status = "new") => get(`/board-jobs?status=${status}`),
  setStatus: (id, status) => post(`/board-jobs/${id}/status`, { status }),
},
```

- [ ] **Step 2: Add Board Jobs nav link to frontend/src/App.vue**

In `App.vue`, add after the existing `<RouterLink to="/jobs">` block:

```html
<RouterLink to="/board-jobs"
  active-class="bg-white/10 text-white font-medium"
  class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
>Board Jobs</RouterLink>
```

- [ ] **Step 3: Add /board-jobs route to frontend/src/router/index.js**

Add after the `/jobs` route:

```js
{ path: "/board-jobs", component: () => import("../views/BoardJobsView.vue") },
```

- [ ] **Step 4: Create frontend/src/views/BoardJobsView.vue**

```vue
<template>
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold">Board Jobs</h1>
    <label class="flex items-center gap-2 text-sm text-gray-500">
      <input type="checkbox" v-model="showDismissed" @change="load" />
      Show dismissed
    </label>
  </div>

  <div v-if="failedBoards.length" class="mb-4 rounded-md bg-amber-50 border border-amber-200 px-4 py-2 text-sm text-amber-800">
    ⚠ {{ failedBoards.length }} of {{ boards.length }} boards failed to scrape —
    <RouterLink to="/settings?tab=scraping" class="underline">see Settings</RouterLink>.
  </div>

  <p class="text-xs text-gray-400 mb-4">
    Run <code>/scrape-jobs</code> in Claude Code to refresh this list.
  </p>

  <div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 text-xs">
        <tr>
          <th class="text-left px-4 py-3">Source</th>
          <th class="text-left px-4 py-3">Company</th>
          <th class="text-left px-4 py-3">Role</th>
          <th class="text-left px-4 py-3">Location</th>
          <th class="text-left px-4 py-3">Why matched</th>
          <th class="text-left px-4 py-3">Found</th>
          <th class="text-left px-4 py-3">Action</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="jobs.length === 0">
          <td colspan="7" class="px-4 py-8 text-center text-gray-400">
            No board jobs. Run <code>/scrape-jobs</code> in Claude Code.
          </td>
        </tr>
        <tr v-for="job in jobs" :key="job.id" class="hover:bg-gray-50 align-top">
          <td class="px-4 py-3">
            <span class="inline-block bg-blue-100 text-blue-700 text-xs font-medium px-2 py-0.5 rounded-full">
              {{ boardLabel(job.source_board) }}
            </span>
          </td>
          <td class="px-4 py-3 font-medium">{{ job.company_name }}</td>
          <td class="px-4 py-3">
            <a :href="job.url" target="_blank" rel="noopener" class="text-blue-600 hover:underline">{{ job.title }}</a>
            <div v-if="job.matched_skills" class="flex flex-wrap gap-1 mt-1">
              <span
                v-for="sk in job.matched_skills.split(',').map(s => s.trim())"
                :key="sk"
                class="inline-block bg-indigo-100 text-indigo-700 text-xs font-medium px-2 py-0.5 rounded-full"
              >{{ sk }}</span>
            </div>
          </td>
          <td class="px-4 py-3 text-gray-500">{{ job.location || '—' }}</td>
          <td class="px-4 py-3 text-gray-500 text-xs">{{ job.match_reason || '—' }}</td>
          <td class="px-4 py-3 text-gray-400 text-xs">{{ (job.scraped_at || '').slice(0, 10) }}</td>
          <td class="px-4 py-3">
            <div class="flex items-center gap-2">
              <button @click="toggleApply(job)" class="text-xs border rounded px-2 py-1 text-gray-600 hover:bg-gray-50">
                {{ openApplyId === job.id ? 'Hide' : 'Apply' }}
              </button>
              <button v-if="job.status !== 'dismissed'" @click="setStatus(job, 'dismissed')"
                class="text-xs text-gray-400 hover:text-red-600">Dismiss</button>
              <button v-else @click="setStatus(job, 'new')"
                class="text-xs text-gray-400 hover:text-blue-600">Restore</button>
            </div>
            <div v-if="openApplyId === job.id" class="mt-2 bg-gray-900 text-gray-100 rounded px-3 py-2 text-xs font-mono flex items-center justify-between gap-3">
              <span class="truncate">/apply {{ job.url }}</span>
              <button @click="copyCmd(job)" class="shrink-0 text-gray-300 hover:text-white">
                {{ copiedId === job.id ? '✓ Copied' : 'Copy' }}
              </button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import { api } from '../api.js'

const jobs        = ref([])
const boards      = ref([])
const showDismissed = ref(false)
const openApplyId = ref(null)
const copiedId    = ref(null)
const failedBoards = ref([])

const boardLabels = {
  remotive:       'Remotive',
  remoteok:       'Remote OK',
  weworkremotely: 'We Work Remotely',
  jobicy:         'Jobicy',
}
function boardLabel(name) { return boardLabels[name] || name }

async function load() {
  jobs.value   = await api.boardJobs.list(showDismissed.value ? 'dismissed' : 'new')
  boards.value = await api.boards.list()
  failedBoards.value = boards.value.filter(b => b.last_scrape_status === 'error')
}

function toggleApply(job) {
  openApplyId.value = openApplyId.value === job.id ? null : job.id
}

async function copyCmd(job) {
  await navigator.clipboard.writeText(`/apply ${job.url}`)
  copiedId.value = job.id
  setTimeout(() => { copiedId.value = null }, 1500)
}

async function setStatus(job, status) {
  await api.boardJobs.setStatus(job.id, status)
  await load()
}

onMounted(load)
</script>
```

- [ ] **Step 5: Add Job Boards section to SettingsView.vue**

In `SettingsView.vue`, add the Job Boards section inside the `<!-- Job Scraping tab -->` div, just before the existing `<section>` for "Scrape Companies":

```html
<section class="bg-white rounded-lg shadow p-5 mb-6">
  <h2 class="font-semibold mb-3">Job Boards</h2>
  <p class="text-xs text-gray-500 mb-3">Enable boards to fetch jobs from. Run <code>/scrape-jobs</code> in Claude Code to pull listings.</p>
  <ul class="divide-y divide-gray-100">
    <li v-for="b in boards" :key="b.id" class="flex items-center justify-between py-3 text-sm">
      <div>
        <span class="font-medium">{{ b.label }}</span>
        <div class="text-xs mt-0.5" :class="b.last_scrape_status === 'error' ? 'text-red-600' : 'text-gray-400'">
          <template v-if="b.last_scrape_status === 'ok'">✅ {{ b.last_job_count }} jobs · {{ (b.last_scraped_at||'').slice(0,16) }}</template>
          <template v-else-if="b.last_scrape_status === 'error'">🔴 {{ b.last_scrape_error }}</template>
          <template v-else>Not scraped yet</template>
        </div>
      </div>
      <label class="relative inline-flex items-center cursor-pointer">
        <input type="checkbox" :checked="b.enabled" @change="toggleBoard(b)" class="sr-only peer">
        <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:bg-blue-600 after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full"></div>
      </label>
    </li>
  </ul>
</section>
```

- [ ] **Step 6: Add boards state + loadScrape update to SettingsView.vue script**

In the `<script setup>` block of `SettingsView.vue`, add `boards` ref (alongside the existing scraping refs like `companies`, `roles`, etc.):

Find where `const companies = ref([])` is declared and add:
```js
const boards = ref([])
```

In `loadScrape()` (or equivalent function that loads the scraping tab data), add:
```js
boards.value = await api.boards.list()
```

Add the `toggleBoard` function alongside the existing `deleteCompany`, `addRole`, etc.:
```js
async function toggleBoard(b) {
  await api.boards.toggle(b.id, !b.enabled)
  boards.value = await api.boards.list()
}
```

- [ ] **Step 7: Build the frontend**

Run: `just build-frontend-dev`
Expected: Build succeeds with no errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/api.js \
        frontend/src/views/BoardJobsView.vue \
        frontend/src/views/SettingsView.vue \
        frontend/src/router/index.js \
        frontend/src/App.vue
git commit -m "feat: Board Jobs page + Settings Job Boards toggles"
```

---

## Task 11: Skill update

**Files:**
- Modify: `plugins/skills/scrape-jobs.md`

- [ ] **Step 1: Rewrite plugins/skills/scrape-jobs.md**

Replace the entire file content:

```markdown
---
name: scrape-jobs
description: Scrape job vacancies from public job boards, filter them by your timezone/region/role/skill preferences, and save the matches for review in the dashboard.
---

Scrape and filter job vacancies from public job boards.

## Process

### 1. Load preferences

Call `get_scrape_preferences`. It returns `roles`, `skills`, `boards`, `home_timezone`,
and `location_notes`.

Stop early and tell the user to configure the dashboard **Settings → Job
Scraping** sections if any of these are empty or missing:
- both `roles` and `skills` are empty — "Add at least one target role or skill in Settings."
- empty `home_timezone` or `location_notes` — "Set your Location Preferences in Settings."

If `boards` is empty (all boards disabled), warn: "All job boards are disabled — enable at least
one in Settings → Job Scraping → Job Boards." Do **not** stop; continue with zero results.

### 2. Fetch listings

Call `fetch_board_jobs` (no arguments). It returns:
- `jobs`: `[{board, company_name, title, location, url}]` — already coarse-filtered for a remote signal
- `errors`: `[{board, error}]` — boards that failed this run (already recorded for the dashboard)

Do not abort if some boards errored; continue with the jobs you did get.

### 3. Judge each job

**Location is the gate — check this first.** A job that fails location is dropped regardless of role or skill.

**Location fit** — apply `home_timezone` + `location_notes`. Accept worldwide/anywhere; accept any
stated timezone window that includes the home timezone; accept the regions the notes list as
acceptable. Reject roles locked to regions that exclude the home timezone (e.g. "US only",
"EMEA only").

Keep a job only if location passes AND either:

- **Role fit** — the title is a semantic match for one of the user's target roles. Match meaning,
  not substrings: "Site Reliability Engineer" matches an "SRE" target; "Developer Relations"
  does NOT match "DevOps". "CI Engineer" DOES match "DevOps Engineer" because it's clearly
  an infra/pipeline role.

- **Skill fit** — the title signals one or more target skills. Match semantically: "Golang"
  matches "Go"; "k8s" matches "Kubernetes"; "Rust developer" matches "Rust". A job can match
  on skill alone — the title does not need to match any target role.

For each kept job, record:
- `match_reason` — one short line, e.g. `"worldwide remote"`, `"APAC includes UTC+7 · Go,Kubernetes"`
- `matched_skills` — comma-separated skills found (e.g. `"Go,Kubernetes"`); empty string if role-only match
- `skill_score` — count of matched skills (0 for role-only matches)

### 4. Save matches

Call `save_board_jobs` with the kept jobs as a JSON array of
`{source_board, company_name, title, location, url, match_reason, matched_skills, skill_score}`.
It ignores URLs already saved and returns `{new, already_seen}`.

### 5. Report

Summarize:
- "X new matches saved, Y already seen."
- Break down by match type: "N role-only, M skill-only, K role+skill."
- If any boards errored: "Z boards failed: Remotive (network error), …"
- "Open the dashboard **Board Jobs** page to review and apply."

## Rules

- Never invent jobs — only judge and save what `fetch_board_jobs` returned.
- Location fit is required for all jobs — skills do not override location.
- Keep `match_reason` to one short line; it shows in the dashboard.
- Do not fetch full job descriptions here — that happens later via `/apply`.
```

- [ ] **Step 2: Run full test suite to confirm nothing broke**

Run: `just test`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add plugins/skills/scrape-jobs.md
git commit -m "feat: update scrape-jobs skill to use board tools"
```

---

## Done

All 11 tasks complete. The board jobs feature is fully implemented:
- 4 board adapters (Remotive, Remote OK, We Work Remotely, Jobicy)
- `scrape_boards` config table with toggles in Settings
- `board_jobs` table with separate dashboard page
- `fetch_board_jobs` + `save_board_jobs` MCP tools
- `/scrape-jobs` skill updated to use board tools
