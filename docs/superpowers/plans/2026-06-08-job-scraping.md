# Job Scraping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Scrape job vacancies from ATS-backed company career pages, filter them against the user's remote/timezone/region/role preferences via Claude, and surface matches in the dashboard with a one-click `/apply` handoff.

**Architecture:** Go fetches structured job data from ATS platforms (public JSON APIs; rod browser for Workday) and stores preferences + results in SQLite. Claude (via the `/scrape-jobs` skill + 3 MCP tools) reads preferences, fetches a compact job list, judges timezone/region/role fit, and saves matches. The Vue dashboard displays matches, hands off a copyable `/apply <url>` command, and shows per-company scrape status. Applications created from a scraped job auto-link back by normalized URL and are tagged "Scraped".

**Tech Stack:** Go 1.26, `modernc.org/sqlite`, `golang-migrate`, `go-rod/rod` (already vendored), `mark3labs/mcp-go`, Vue 3 + Vue Router + Tailwind, Vite.

**Source spec:** `docs/superpowers/specs/2026-06-08-job-scraping-design.md`

**Phasing:** Phases 1–5 deliver a working feature (Greenhouse + Lever + Ashby). Phase 6 adds 7 more adapters. Phase 7 wires the skill + docs.

---

## File Structure

**New files:**
- `internal/db/migrations/000007_add_job_scraping.up.sql` — 4 new tables + default prefs row
- `internal/db/migrations/000007_add_job_scraping.down.sql` — drop the 4 tables
- `internal/db/scrape.go` — store methods for companies, roles, prefs, scraped_jobs, linkage
- `internal/db/scrape_test.go` — db-layer tests
- `internal/scrape/scrape.go` — `Adapter` interface, `Job` type, shared HTTP helper, `Registry`
- `internal/scrape/detect.go` — `Detect(url)` + `NormalizeURL(url)`
- `internal/scrape/detect_test.go`
- `internal/scrape/greenhouse.go`, `lever.go`, `ashby.go`, `smartrecruiters.go`, `workable.go`, `recruitee.go`, `breezy.go`, `bamboohr.go`, `personio.go`, `workday.go` — one adapter each
- `internal/scrape/*_test.go` — one test per adapter, using fixtures
- `internal/scrape/testdata/*.json|*.xml` — recorded ATS responses
- `internal/mcp/scrape_tools.go` — `get_scrape_preferences`, `fetch_ats_jobs`, `save_scraped_jobs`
- `internal/mcp/scrape_tools_test.go`
- `internal/dashboard/handlers_scrape.go` — REST handlers for companies/roles/prefs/jobs
- `internal/dashboard/handlers_scrape_test.go`
- `frontend/src/views/JobsView.vue` — scraped-jobs page
- `plugins/skills/scrape-jobs.md` — the skill

**Modified files:**
- `internal/db/applications.go` — add `FromScrape` field + `EXISTS` subquery; add linkage call site note
- `internal/mcp/server.go:43-50` — register scrape tools
- `internal/mcp/application_tools.go` — call linkage after `UpsertApplication`
- `internal/dashboard/server.go:52-96` — register scrape routes + `/jobs` is handled by SPA fallback
- `frontend/src/router/index.js` — add `/jobs` route
- `frontend/src/App.vue` — add "Jobs" nav link
- `frontend/src/api.js` — add `scrape` API section
- `frontend/src/views/SettingsView.vue` — add Companies / Roles / Location Preferences sections
- `frontend/src/views/AppsView.vue` — add "Scraped" badge

---

# Phase 1 — Data layer

## Task 1: Migration 007 — job scraping tables

**Files:**
- Create: `internal/db/migrations/000007_add_job_scraping.up.sql`
- Create: `internal/db/migrations/000007_add_job_scraping.down.sql`
- Test: `internal/db/db_test.go` (add one test)

- [ ] **Step 1: Write the up migration**

Create `internal/db/migrations/000007_add_job_scraping.up.sql`:

```sql
CREATE TABLE scrape_companies (
  id                 INTEGER PRIMARY KEY AUTOINCREMENT,
  name               TEXT NOT NULL,
  ats_platform       TEXT NOT NULL,
  ats_token          TEXT NOT NULL,
  careers_url        TEXT NOT NULL,
  last_scraped_at    TEXT NOT NULL DEFAULT '',
  last_scrape_status TEXT NOT NULL DEFAULT 'never',
  last_scrape_error  TEXT NOT NULL DEFAULT '',
  last_job_count     INTEGER NOT NULL DEFAULT 0,
  UNIQUE(ats_platform, ats_token)
);

CREATE TABLE scrape_roles (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  keyword TEXT NOT NULL UNIQUE
);

CREATE TABLE scrape_prefs (
  id             INTEGER PRIMARY KEY CHECK (id = 1),
  home_timezone  TEXT NOT NULL DEFAULT '',
  location_notes TEXT NOT NULL DEFAULT ''
);

INSERT INTO scrape_prefs (id, home_timezone, location_notes) VALUES (1, '', '');

CREATE TABLE scraped_jobs (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  company_id             INTEGER NOT NULL REFERENCES scrape_companies(id) ON DELETE CASCADE,
  company_name           TEXT NOT NULL,
  title                  TEXT NOT NULL,
  location               TEXT NOT NULL DEFAULT '',
  url                    TEXT NOT NULL UNIQUE,
  match_reason           TEXT NOT NULL DEFAULT '',
  status                 TEXT NOT NULL DEFAULT 'new',
  scraped_at             TEXT NOT NULL DEFAULT (datetime('now')),
  applied_application_id INTEGER REFERENCES applications(id)
);
```

- [ ] **Step 2: Write the down migration**

Create `internal/db/migrations/000007_add_job_scraping.down.sql`:

```sql
DROP TABLE scraped_jobs;
DROP TABLE scrape_prefs;
DROP TABLE scrape_roles;
DROP TABLE scrape_companies;
```

- [ ] **Step 3: Write a test that the new tables exist**

In `internal/db/db_test.go`, add:

```go
func TestMigration007Tables(t *testing.T) {
	store := openTestStore(t)
	tables := []string{"scrape_companies", "scrape_roles", "scrape_prefs", "scraped_jobs"}
	for _, table := range tables {
		var name string
		err := store.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %s missing", table)
		assert.Equal(t, table, name)
	}

	// Default prefs row must exist.
	var n int
	require.NoError(t, store.DB().QueryRow(`SELECT COUNT(*) FROM scrape_prefs WHERE id=1`).Scan(&n))
	assert.Equal(t, 1, n)
}
```

- [ ] **Step 4: Run the test**

Run: `go test ./internal/db/ -run TestMigration007Tables -v`
Expected: PASS (migrations are embedded and run on `db.Open`).

- [ ] **Step 5: Commit**

```bash
git add internal/db/migrations/000007_add_job_scraping.up.sql internal/db/migrations/000007_add_job_scraping.down.sql internal/db/db_test.go
git commit -m "feat: migration 007 — job scraping tables"
```

---

## Task 2: DB — companies, roles, preferences

**Files:**
- Create: `internal/db/scrape.go`
- Test: `internal/db/scrape_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/db/scrape_test.go`:

```go
package db_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScrapeCompaniesCRUD(t *testing.T) {
	store := openTestStore(t)

	c, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme",
		CareersURL: "https://boards.greenhouse.io/acme",
	})
	require.NoError(t, err)
	assert.Greater(t, c.ID, int64(0))

	list, err := store.ListScrapeCompanies()
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "Acme", list[0].Name)
	assert.Equal(t, "never", list[0].LastScrapeStatus)

	require.NoError(t, store.UpdateScrapeCompanyStatus(c.ID, "2026-06-08 10:00:00", "ok", "", 12))
	list, _ = store.ListScrapeCompanies()
	assert.Equal(t, "ok", list[0].LastScrapeStatus)
	assert.Equal(t, 12, list[0].LastJobCount)

	require.NoError(t, store.DeleteScrapeCompany(c.ID))
	list, _ = store.ListScrapeCompanies()
	assert.Len(t, list, 0)
}

func TestScrapeRolesCRUD(t *testing.T) {
	store := openTestStore(t)

	r, err := store.AddScrapeRole("SRE")
	require.NoError(t, err)
	assert.Greater(t, r.ID, int64(0))

	roles, err := store.ListScrapeRoles()
	require.NoError(t, err)
	require.Len(t, roles, 1)
	assert.Equal(t, "SRE", roles[0].Keyword)

	require.NoError(t, store.DeleteScrapeRole(r.ID))
	roles, _ = store.ListScrapeRoles()
	assert.Len(t, roles, 0)
}

func TestScrapePrefs(t *testing.T) {
	store := openTestStore(t)

	prefs, err := store.GetScrapePrefs()
	require.NoError(t, err)
	assert.Equal(t, "", prefs.HomeTimezone)

	require.NoError(t, store.UpdateScrapePrefs(db.ScrapePrefs{
		HomeTimezone: "UTC+7", LocationNotes: "Accept anywhere; APAC; Indonesia",
	}))
	prefs, err = store.GetScrapePrefs()
	require.NoError(t, err)
	assert.Equal(t, "UTC+7", prefs.HomeTimezone)
	assert.Contains(t, prefs.LocationNotes, "APAC")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/ -run 'TestScrape' -v`
Expected: FAIL — `undefined: store.AddScrapeCompany` etc.

- [ ] **Step 3: Implement the store methods**

Create `internal/db/scrape.go`:

```go
package db

type ScrapeCompany struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	ATSPlatform      string `json:"ats_platform"`
	ATSToken         string `json:"ats_token"`
	CareersURL       string `json:"careers_url"`
	LastScrapedAt    string `json:"last_scraped_at"`
	LastScrapeStatus string `json:"last_scrape_status"`
	LastScrapeError  string `json:"last_scrape_error"`
	LastJobCount     int    `json:"last_job_count"`
}

type ScrapeRole struct {
	ID      int64  `json:"id"`
	Keyword string `json:"keyword"`
}

type ScrapePrefs struct {
	HomeTimezone  string `json:"home_timezone"`
	LocationNotes string `json:"location_notes"`
}

func (s *Store) AddScrapeCompany(c ScrapeCompany) (ScrapeCompany, error) {
	res, err := s.db.Exec(
		`INSERT INTO scrape_companies (name, ats_platform, ats_token, careers_url)
		 VALUES (?, ?, ?, ?)`,
		c.Name, c.ATSPlatform, c.ATSToken, c.CareersURL,
	)
	if err != nil {
		return ScrapeCompany{}, err
	}
	id, _ := res.LastInsertId()
	c.ID = id
	c.LastScrapeStatus = "never"
	return c, nil
}

func (s *Store) ListScrapeCompanies() ([]ScrapeCompany, error) {
	rows, err := s.db.Query(
		`SELECT id, name, ats_platform, ats_token, careers_url,
		        last_scraped_at, last_scrape_status, last_scrape_error, last_job_count
		 FROM scrape_companies ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapeCompany
	for rows.Next() {
		var c ScrapeCompany
		if err := rows.Scan(&c.ID, &c.Name, &c.ATSPlatform, &c.ATSToken, &c.CareersURL,
			&c.LastScrapedAt, &c.LastScrapeStatus, &c.LastScrapeError, &c.LastJobCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) UpdateScrapeCompanyStatus(id int64, scrapedAt, status, errMsg string, jobCount int) error {
	_, err := s.db.Exec(
		`UPDATE scrape_companies
		 SET last_scraped_at=?, last_scrape_status=?, last_scrape_error=?, last_job_count=?
		 WHERE id=?`,
		scrapedAt, status, errMsg, jobCount, id,
	)
	return err
}

func (s *Store) DeleteScrapeCompany(id int64) error {
	_, err := s.db.Exec(`DELETE FROM scrape_companies WHERE id=?`, id)
	return err
}

func (s *Store) AddScrapeRole(keyword string) (ScrapeRole, error) {
	res, err := s.db.Exec(`INSERT INTO scrape_roles (keyword) VALUES (?)`, keyword)
	if err != nil {
		return ScrapeRole{}, err
	}
	id, _ := res.LastInsertId()
	return ScrapeRole{ID: id, Keyword: keyword}, nil
}

func (s *Store) ListScrapeRoles() ([]ScrapeRole, error) {
	rows, err := s.db.Query(`SELECT id, keyword FROM scrape_roles ORDER BY keyword`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapeRole
	for rows.Next() {
		var r ScrapeRole
		if err := rows.Scan(&r.ID, &r.Keyword); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) DeleteScrapeRole(id int64) error {
	_, err := s.db.Exec(`DELETE FROM scrape_roles WHERE id=?`, id)
	return err
}

func (s *Store) GetScrapePrefs() (ScrapePrefs, error) {
	var p ScrapePrefs
	err := s.db.QueryRow(
		`SELECT home_timezone, location_notes FROM scrape_prefs WHERE id=1`,
	).Scan(&p.HomeTimezone, &p.LocationNotes)
	return p, err
}

func (s *Store) UpdateScrapePrefs(p ScrapePrefs) error {
	_, err := s.db.Exec(
		`UPDATE scrape_prefs SET home_timezone=?, location_notes=? WHERE id=1`,
		p.HomeTimezone, p.LocationNotes,
	)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/db/ -run 'TestScrape' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/db/scrape.go internal/db/scrape_test.go
git commit -m "feat: db layer for scrape companies, roles, preferences"
```

---

## Task 3: DB — scraped jobs + apply linkage

**Files:**
- Modify: `internal/db/scrape.go`
- Modify: `internal/db/applications.go` (add `FromScrape` field + EXISTS subquery)
- Test: `internal/db/scrape_test.go`

- [ ] **Step 1: Write failing tests**

Add to `internal/db/scrape_test.go`:

```go
func TestScrapedJobsSaveAndList(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme",
		CareersURL: "https://boards.greenhouse.io/acme",
	})

	jobs := []db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", Location: "Remote", URL: "https://j/1", MatchReason: "worldwide"},
		{CompanyID: c.ID, CompanyName: "Acme", Title: "DevOps", Location: "Remote APAC", URL: "https://j/2", MatchReason: "APAC"},
	}
	added, err := store.SaveScrapedJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 2, added)

	// Re-saving the same URLs adds nothing (INSERT OR IGNORE).
	added, err = store.SaveScrapedJobs(jobs)
	require.NoError(t, err)
	assert.Equal(t, 0, added)

	list, err := store.ListScrapedJobs("new")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestScrapedJobStatusUpdate(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://j/1"}})

	list, _ := store.ListScrapedJobs("new")
	require.Len(t, list, 1)
	require.NoError(t, store.SetScrapedJobStatus(list[0].ID, "dismissed"))

	assert.Len(t, mustList(t, store, "new"), 0)
	assert.Len(t, mustList(t, store, "dismissed"), 1)
}

func TestApplyLinkageByURL(t *testing.T) {
	store := openTestStore(t)
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://boards.greenhouse.io/acme/jobs/42"},
	})

	appID, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "SRE", FolderPath: "/tmp/acme",
		JobURL: "https://boards.greenhouse.io/acme/jobs/42?utm_source=x",
	})
	require.NoError(t, err)

	linked, err := store.LinkApplicationToScrapedJob(appID, "https://boards.greenhouse.io/acme/jobs/42?utm_source=x")
	require.NoError(t, err)
	assert.True(t, linked)

	// The scraped job is now 'applied' and out of the 'new' list.
	assert.Len(t, mustList(t, store, "new"), 0)
	applied := mustList(t, store, "applied")
	require.Len(t, applied, 1)
	assert.Equal(t, appID, applied[0].AppliedApplicationID.Int64)

	// The application reports FromScrape.
	apps, err := store.ListApplicationsWithStatus()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.True(t, apps[0].FromScrape)
}

func mustList(t *testing.T, store *db.Store, status string) []db.ScrapedJob {
	t.Helper()
	l, err := store.ListScrapedJobs(status)
	require.NoError(t, err)
	return l
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/db/ -run 'TestScrapedJobs|TestScrapedJobStatus|TestApplyLinkage' -v`
Expected: FAIL — undefined methods + `FromScrape`.

- [ ] **Step 3: Implement scraped-jobs methods + linkage**

Add to `internal/db/scrape.go` (add `"database/sql"` to imports if not present):

```go
import "database/sql"

type ScrapedJob struct {
	ID                   int64         `json:"id"`
	CompanyID            int64         `json:"company_id"`
	CompanyName          string        `json:"company_name"`
	Title                string        `json:"title"`
	Location             string        `json:"location"`
	URL                  string        `json:"url"`
	MatchReason          string        `json:"match_reason"`
	Status               string        `json:"status"`
	ScrapedAt            string        `json:"scraped_at"`
	AppliedApplicationID sql.NullInt64 `json:"applied_application_id"`
}

// SaveScrapedJobs inserts jobs, ignoring any whose URL already exists.
// Returns the count of newly inserted rows.
func (s *Store) SaveScrapedJobs(jobs []ScrapedJob) (int, error) {
	added := 0
	for _, j := range jobs {
		res, err := s.db.Exec(
			`INSERT OR IGNORE INTO scraped_jobs
			   (company_id, company_name, title, location, url, match_reason)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			j.CompanyID, j.CompanyName, j.Title, j.Location, j.URL, j.MatchReason,
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

func (s *Store) ListScrapedJobs(status string) ([]ScrapedJob, error) {
	rows, err := s.db.Query(
		`SELECT id, company_id, company_name, title, location, url, match_reason,
		        status, scraped_at, applied_application_id
		 FROM scraped_jobs WHERE status=? ORDER BY id DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ScrapedJob
	for rows.Next() {
		var j ScrapedJob
		if err := rows.Scan(&j.ID, &j.CompanyID, &j.CompanyName, &j.Title, &j.Location,
			&j.URL, &j.MatchReason, &j.Status, &j.ScrapedAt, &j.AppliedApplicationID); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) SetScrapedJobStatus(id int64, status string) error {
	_, err := s.db.Exec(`UPDATE scraped_jobs SET status=? WHERE id=?`, status, id)
	return err
}

// LinkApplicationToScrapedJob finds a scraped job whose normalized URL matches
// the application's normalized job URL, links them, and marks it applied.
// Returns true if a match was found and linked.
func (s *Store) LinkApplicationToScrapedJob(appID int64, jobURL string) (bool, error) {
	norm := NormalizeJobURL(jobURL)
	if norm == "" {
		return false, nil
	}
	res, err := s.db.Exec(
		`UPDATE scraped_jobs
		 SET applied_application_id=?, status='applied'
		 WHERE status != 'applied'
		   AND lower(rtrim(substr(url, 1, CASE WHEN instr(url,'?')>0 THEN instr(url,'?')-1 ELSE length(url) END), '/')) = ?`,
		appID, norm,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
```

NOTE: The SQL-side normalization above must match `NormalizeJobURL`. To avoid SQL/Go drift, implement the match in Go instead — replace the body of `LinkApplicationToScrapedJob` with:

```go
func (s *Store) LinkApplicationToScrapedJob(appID int64, jobURL string) (bool, error) {
	norm := NormalizeJobURL(jobURL)
	if norm == "" {
		return false, nil
	}
	rows, err := s.db.Query(`SELECT id, url FROM scraped_jobs WHERE status != 'applied'`)
	if err != nil {
		return false, err
	}
	var matchID int64
	for rows.Next() {
		var id int64
		var u string
		if err := rows.Scan(&id, &u); err != nil {
			rows.Close()
			return false, err
		}
		if NormalizeJobURL(u) == norm {
			matchID = id
			break
		}
	}
	rows.Close()
	if matchID == 0 {
		return false, nil
	}
	_, err = s.db.Exec(
		`UPDATE scraped_jobs SET applied_application_id=?, status='applied' WHERE id=?`,
		appID, matchID,
	)
	return err == nil, err
}

// NormalizeJobURL strips query/fragment, trailing slash, and lowercases the host.
func NormalizeJobURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	p := strings.TrimRight(u.Path, "/")
	return u.Scheme + "://" + u.Host + p
}
```

Add `"net/url"` and `"strings"` to the imports of `internal/db/scrape.go`. Delete the first (SQL-based) version of `LinkApplicationToScrapedJob` — keep only the Go-based one.

- [ ] **Step 4: Add `FromScrape` to applications**

In `internal/db/applications.go`, change the `ApplicationWithStatus` struct:

```go
type ApplicationWithStatus struct {
	Application
	CurrentStatus string
	FromScrape    bool
}
```

In `ListApplicationsWithStatus`, change the SELECT to add the EXISTS column (place it after `current_status`):

```go
	rows, err := s.db.Query(`
		SELECT a.id, a.company, a.role, a.applied_date, a.base_cv_used, a.notes, a.folder_path, a.source, a.job_url, a.job_summary, a.match_score, a.tailoring_notes,
		       COALESCE(
		         (SELECT status FROM application_status_history
		          WHERE application_id = a.id
		          ORDER BY changed_at DESC, id DESC LIMIT 1),
		         'unknown'
		       ) AS current_status,
		       EXISTS(SELECT 1 FROM scraped_jobs sj WHERE sj.applied_application_id = a.id) AS from_scrape
		FROM applications a
		ORDER BY a.id DESC
	`)
```

And update the `Scan` in that function to append `&a.FromScrape`:

```go
		if err := rows.Scan(
			&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath, &a.Source, &a.JobURL, &a.JobSummary, &a.MatchScore, &a.TailoringNotes,
			&a.CurrentStatus, &a.FromScrape,
		); err != nil {
			return nil, err
		}
```

Leave `ListApplicationsByBaseCV` unchanged (it does not need `FromScrape`).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/db/ -run 'TestScrapedJobs|TestScrapedJobStatus|TestApplyLinkage' -v`
Expected: PASS

- [ ] **Step 6: Run the whole db package to catch regressions**

Run: `go test ./internal/db/...`
Expected: PASS (ensures the modified `ListApplicationsWithStatus` Scan still matches existing callers).

- [ ] **Step 7: Commit**

```bash
git add internal/db/scrape.go internal/db/scrape_test.go internal/db/applications.go
git commit -m "feat: scraped jobs store + apply linkage by normalized URL"
```

---

# Phase 2 — Scraper core + URL detection

## Task 4: URL detection & normalization

**Files:**
- Create: `internal/scrape/detect.go`
- Test: `internal/scrape/detect_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/scrape/detect_test.go`:

```go
package scrape_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/scrape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		url, platform, token string
	}{
		{"https://boards.greenhouse.io/acme", "greenhouse", "acme"},
		{"https://boards.greenhouse.io/acme/jobs/123", "greenhouse", "acme"},
		{"https://jobs.lever.co/acme", "lever", "acme"},
		{"https://jobs.ashbyhq.com/acme", "ashby", "acme"},
		{"https://careers.smartrecruiters.com/Acme", "smartrecruiters", "Acme"},
		{"https://apply.workable.com/acme/", "workable", "acme"},
		{"https://acme.recruitee.com", "recruitee", "acme"},
		{"https://acme.breezy.hr", "breezy", "acme"},
		{"https://acme.bamboohr.com/careers", "bamboohr", "acme"},
		{"https://acme.jobs.personio.de", "personio", "acme"},
		{"https://acme.wd1.myworkdayjobs.com/External", "workday", "acme"},
	}
	for _, c := range cases {
		platform, token, err := scrape.Detect(c.url)
		require.NoError(t, err, c.url)
		assert.Equal(t, c.platform, platform, c.url)
		assert.Equal(t, c.token, token, c.url)
	}
}

func TestDetectUnsupported(t *testing.T) {
	_, _, err := scrape.Detect("https://acme.com/careers")
	assert.Error(t, err)
}

func TestNormalizeURL(t *testing.T) {
	assert.Equal(t,
		"https://boards.greenhouse.io/acme/jobs/42",
		scrape.NormalizeURL("https://Boards.Greenhouse.io/acme/jobs/42/?utm=x#frag"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/scrape/ -run 'TestDetect|TestNormalize' -v`
Expected: FAIL — package/functions do not exist.

- [ ] **Step 3: Implement detect + normalize**

Create `internal/scrape/detect.go`:

```go
package scrape

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL strips query/fragment, trailing slash, lowercases host.
// Kept identical in spirit to db.NormalizeJobURL so linkage agrees.
func NormalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.Scheme + "://" + u.Host + u.Path
}

// Detect identifies the ATS platform and extracts the company token from a
// careers URL. Returns an error for unsupported hosts.
func Detect(raw string) (platform, token string, err error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", fmt.Errorf("invalid URL: %q", raw)
	}
	host := strings.ToLower(u.Host)
	segs := pathSegments(u.Path)
	first := ""
	if len(segs) > 0 {
		first = segs[0]
	}
	sub := subdomain(host)

	switch {
	case host == "boards.greenhouse.io" && first != "":
		return "greenhouse", first, nil
	case host == "jobs.lever.co" && first != "":
		return "lever", first, nil
	case host == "jobs.ashbyhq.com" && first != "":
		return "ashby", first, nil
	case host == "careers.smartrecruiters.com" && first != "":
		return "smartrecruiters", first, nil
	case host == "apply.workable.com" && first != "":
		return "workable", first, nil
	case strings.HasSuffix(host, ".recruitee.com") && sub != "":
		return "recruitee", sub, nil
	case strings.HasSuffix(host, ".breezy.hr") && sub != "":
		return "breezy", sub, nil
	case strings.HasSuffix(host, ".bamboohr.com") && sub != "":
		return "bamboohr", sub, nil
	case strings.HasSuffix(host, ".jobs.personio.de") && sub != "":
		return "personio", sub, nil
	case strings.HasSuffix(host, ".myworkdayjobs.com") && sub != "":
		return "workday", sub, nil
	}
	return "", "", fmt.Errorf("unsupported ATS host: %q", host)
}

func pathSegments(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// subdomain returns the left-most label (e.g. "acme" from "acme.recruitee.com",
// "acme" from "acme.wd1.myworkdayjobs.com", "acme" from "acme.jobs.personio.de").
func subdomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/scrape/ -run 'TestDetect|TestNormalize' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scrape/detect.go internal/scrape/detect_test.go
git commit -m "feat: ATS URL detection and normalization"
```

---

## Task 5: Adapter interface, Job type, HTTP helper, registry

**Files:**
- Create: `internal/scrape/scrape.go`
- Test: covered by adapter tests in later tasks (no standalone test here)

- [ ] **Step 1: Implement the core types and helpers**

Create `internal/scrape/scrape.go`:

```go
package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Job is a single vacancy as returned by an adapter.
type Job struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	URL      string `json:"url"`
}

// Adapter fetches jobs for one ATS platform given a company token.
type Adapter interface {
	FetchJobs(ctx context.Context, token string) ([]Job, error)
}

// userAgent is a realistic desktop Chrome UA so public endpoints don't reject us.
const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// httpGetJSON performs a GET and decodes the JSON body into out.
// A non-2xx status returns a *HTTPError so callers can classify it.
func httpGetJSON(ctx context.Context, url string, out any) error {
	body, err := httpGet(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	return nil
}

// httpGet performs a GET and returns the raw body.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/xml, */*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrBlocked
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrUnexpectedFormat, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Sentinel errors for classification into user-facing messages.
var (
	ErrNotFound         = fmt.Errorf("not found")
	ErrUnexpectedFormat = fmt.Errorf("unexpected format")
	ErrNetwork          = fmt.Errorf("network error")
	ErrBlocked          = fmt.Errorf("blocked")
)

// Registry maps platform names to adapters. Each adapter task below adds its
// own entry, so the build stays green at every step. Workday is intentionally
// absent — it is special-cased in fetch_ats_jobs because it needs CareersURL.
func Registry() map[string]Adapter {
	return map[string]Adapter{
		// adapters are registered by Tasks 6-8 and 18-23
	}
}

// ClassifyError maps a fetch error to a human-readable, platform-aware message.
func ClassifyError(platform string, err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotFound):
		return fmt.Sprintf("Company not found on %s — the token may have changed", platform)
	case errors.Is(err, ErrBlocked):
		return "Request was blocked — try again later"
	case errors.Is(err, ErrNetwork):
		return fmt.Sprintf("Couldn't reach %s (network/timeout)", platform)
	case errors.Is(err, ErrUnexpectedFormat):
		return fmt.Sprintf("Couldn't parse %s response — format may have changed", platform)
	default:
		return err.Error()
	}
}
```

The full import block for `internal/scrape/scrape.go` is:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/scrape/`
Expected: SUCCESS — the registry is empty, so `scrape.go` compiles standalone alongside `detect.go`.

- [ ] **Step 3: Commit**

```bash
git add internal/scrape/scrape.go
git commit -m "feat: scrape adapter interface, HTTP helper, registry, error classification"
```

---

## Task 6: Greenhouse adapter

**Files:**
- Create: `internal/scrape/greenhouse.go`
- Create: `internal/scrape/testdata/greenhouse.json`
- Test: `internal/scrape/greenhouse_test.go`
- Modify: `internal/scrape/scrape.go` (add the `greenhouse` entry to `Registry()`)

- [ ] **Step 1: Add the fixture**

Create `internal/scrape/testdata/greenhouse.json`:

```json
{
  "jobs": [
    {
      "id": 42,
      "title": "Senior SRE",
      "absolute_url": "https://boards.greenhouse.io/acme/jobs/42",
      "location": { "name": "Remote - Anywhere" }
    },
    {
      "id": 43,
      "title": "Office Manager",
      "absolute_url": "https://boards.greenhouse.io/acme/jobs/43",
      "location": { "name": "New York, NY" }
    }
  ]
}
```

- [ ] **Step 2: Write the failing test**

Create `internal/scrape/greenhouse_test.go`:

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

func TestGreenhouseParse(t *testing.T) {
	data, err := os.ReadFile("testdata/greenhouse.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	gh := Greenhouse{BaseURL: srv.URL}
	jobs, err := gh.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Senior SRE", jobs[0].Title)
	assert.Equal(t, "Remote - Anywhere", jobs[0].Location)
	assert.Equal(t, "https://boards.greenhouse.io/acme/jobs/42", jobs[0].URL)
}
```

- [ ] **Step 3: Implement the adapter**

Create `internal/scrape/greenhouse.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

// Greenhouse reads the public boards API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Greenhouse struct{ BaseURL string }

func (g Greenhouse) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := g.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", token)
	}
	var resp struct {
		Jobs []struct {
			Title       string `json:"title"`
			AbsoluteURL string `json:"absolute_url"`
			Location    struct {
				Name string `json:"name"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location.Name, URL: j.AbsoluteURL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Register the adapter**

In `internal/scrape/scrape.go` `Registry()`, add the entry:

```go
		"greenhouse": Greenhouse{},
```

- [ ] **Step 5: Run the test**

Run: `go test ./internal/scrape/ -run TestGreenhouseParse -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/scrape.go internal/scrape/greenhouse.go internal/scrape/greenhouse_test.go internal/scrape/testdata/greenhouse.json
git commit -m "feat: Greenhouse adapter"
```

---

## Task 7: Lever adapter

**Files:**
- Create: `internal/scrape/lever.go`
- Create: `internal/scrape/testdata/lever.json`
- Test: `internal/scrape/lever_test.go`
- Modify: `internal/scrape/scrape.go` (add `"lever"` to registry)

- [ ] **Step 1: Add the fixture**

Create `internal/scrape/testdata/lever.json`:

```json
[
  {
    "text": "Platform Engineer",
    "hostedUrl": "https://jobs.lever.co/acme/abc-123",
    "categories": { "location": "Remote (APAC)" }
  },
  {
    "text": "Receptionist",
    "hostedUrl": "https://jobs.lever.co/acme/def-456",
    "categories": { "location": "London" }
  }
]
```

- [ ] **Step 2: Write the failing test**

Create `internal/scrape/lever_test.go`:

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

func TestLeverParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/lever.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Lever{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Remote (APAC)", jobs[0].Location)
	assert.Equal(t, "https://jobs.lever.co/acme/abc-123", jobs[0].URL)
}
```

- [ ] **Step 3: Implement the adapter**

Create `internal/scrape/lever.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

type Lever struct{ BaseURL string }

func (l Lever) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := l.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", token)
	}
	var resp []struct {
		Text       string `json:"text"`
		HostedURL  string `json:"hostedUrl"`
		Categories struct {
			Location string `json:"location"`
		} `json:"categories"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp))
	for _, j := range resp {
		jobs = append(jobs, Job{Title: j.Text, Location: j.Categories.Location, URL: j.HostedURL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Add to registry**

In `internal/scrape/scrape.go` `Registry()`, add: `"lever": Lever{},`

- [ ] **Step 5: Run the test**

Run: `go test ./internal/scrape/ -run TestLeverParse -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/lever.go internal/scrape/lever_test.go internal/scrape/testdata/lever.json internal/scrape/scrape.go
git commit -m "feat: Lever adapter"
```

---

## Task 8: Ashby adapter

**Files:**
- Create: `internal/scrape/ashby.go`
- Create: `internal/scrape/testdata/ashby.json`
- Test: `internal/scrape/ashby_test.go`
- Modify: `internal/scrape/scrape.go` (add `"ashby"`)

- [ ] **Step 1: Add the fixture**

Create `internal/scrape/testdata/ashby.json`:

```json
{
  "jobs": [
    {
      "title": "Site Reliability Engineer",
      "location": "Remote - Worldwide",
      "jobUrl": "https://jobs.ashbyhq.com/acme/111"
    },
    {
      "title": "Sales Lead",
      "location": "San Francisco",
      "jobUrl": "https://jobs.ashbyhq.com/acme/222"
    }
  ]
}
```

- [ ] **Step 2: Write the failing test**

Create `internal/scrape/ashby_test.go`:

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

func TestAshbyParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/ashby.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Ashby{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Site Reliability Engineer", jobs[0].Title)
	assert.Equal(t, "Remote - Worldwide", jobs[0].Location)
	assert.Equal(t, "https://jobs.ashbyhq.com/acme/111", jobs[0].URL)
}
```

- [ ] **Step 3: Implement the adapter**

Create `internal/scrape/ashby.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

type Ashby struct{ BaseURL string }

func (a Ashby) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := a.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token)
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			Location string `json:"location"`
			JobURL   string `json:"jobUrl"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location, URL: j.JobURL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Add to registry**

In `internal/scrape/scrape.go` `Registry()`, add: `"ashby": Ashby{},`

- [ ] **Step 5: Run the test**

Run: `go test ./internal/scrape/ -run TestAshbyParse -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/ashby.go internal/scrape/ashby_test.go internal/scrape/testdata/ashby.json internal/scrape/scrape.go
git commit -m "feat: Ashby adapter"
```

---

# Phase 3 — MCP tools

## Task 9: `get_scrape_preferences` tool

**Files:**
- Create: `internal/mcp/scrape_tools.go`
- Modify: `internal/mcp/server.go:43-50` (register)
- Test: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Register the tool group**

In `internal/mcp/server.go`, add to the registration block (after `s.registerMemoryTools()`):

```go
	s.registerScrapeTools()
```

- [ ] **Step 2: Write the failing test**

Create `internal/mcp/scrape_tools_test.go`:

```go
package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetScrapePreferences(t *testing.T) {
	_, srv := setup(t) // helper in internal/mcp/mcp_test.go: returns (root, *Server)
	store := srv.Store()

	_, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u",
	})
	require.NoError(t, err)
	_, err = store.AddScrapeRole("SRE")
	require.NoError(t, err)
	require.NoError(t, store.UpdateScrapePrefs(db.ScrapePrefs{HomeTimezone: "UTC+7", LocationNotes: "APAC"}))

	out, err := srv.CallTool(context.Background(), "get_scrape_preferences", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Companies []db.ScrapeCompany `json:"companies"`
		Roles     []db.ScrapeRole    `json:"roles"`
		HomeTZ    string             `json:"home_timezone"`
		Notes     string             `json:"location_notes"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Len(t, got.Companies, 1)
	assert.Len(t, got.Roles, 1)
	assert.Equal(t, "UTC+7", got.HomeTZ)
	assert.Equal(t, "APAC", got.Notes)
}
```

The `setup(t)` helper already exists in `internal/mcp/mcp_test.go` and returns `(root string, srv *Server)`; all scrape MCP tests reuse it.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/mcp/ -run TestGetScrapePreferences -v`
Expected: FAIL — `registerScrapeTools` undefined / tool not found.

- [ ] **Step 4: Implement the tool**

Create `internal/mcp/scrape_tools.go`:

```go
// internal/mcp/scrape_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/oxGrad/spicebag/internal/db"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerScrapeTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"get_scrape_preferences",
			mcplib.WithDescription("Return the user's saved job-scraping companies, target roles, and location preferences (home timezone + notes) used to judge job fit."),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			companies, err := s.store.ListScrapeCompanies()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			roles, err := s.store.ListScrapeRoles()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			prefs, err := s.store.GetScrapePrefs()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			if companies == nil {
				companies = []db.ScrapeCompany{}
			}
			if roles == nil {
				roles = []db.ScrapeRole{}
			}
			payload := map[string]any{
				"companies":      companies,
				"roles":          roles,
				"home_timezone":  prefs.HomeTimezone,
				"location_notes": prefs.LocationNotes,
			}
			b, _ := json.Marshal(payload)
			return mcplib.NewToolResultText(string(b)), nil
		},
	)

	s.registerFetchATSJobs()
	s.registerSaveScrapedJobs()
}

// Placeholder declarations so the file compiles before Tasks 10–11 add bodies.
// Remove these two stubs when implementing those tasks.
func (s *Server) registerFetchATSJobs()   {}
func (s *Server) registerSaveScrapedJobs() {}
```

The Task 9 imports are exactly `context`, `encoding/json`, `github.com/oxGrad/spicebag/internal/db`, and `mcplib "github.com/mark3labs/mcp-go/mcp"` — no `fmt` yet (Tasks 10–11 add `time`, `math/rand`, and the scrape import).

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/mcp/ -run TestGetScrapePreferences -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/scrape_tools.go internal/mcp/scrape_tools_test.go internal/mcp/server.go
git commit -m "feat: get_scrape_preferences MCP tool"
```

---

## Task 10: `fetch_ats_jobs` tool

**Files:**
- Modify: `internal/mcp/scrape_tools.go` (replace the `registerFetchATSJobs` stub)
- Test: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/mcp/scrape_tools_test.go`:

```go
func TestFetchATSJobsRecordsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	_, srv := setup(t)
	store := srv.Store()

	// A company on an unsupported/unreachable token: the run must not abort,
	// and the company's status must become 'error'.
	c, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "does-not-exist-xyz", CareersURL: "u",
	})
	require.NoError(t, err)

	out, err := srv.CallTool(context.Background(), "fetch_ats_jobs", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Jobs   []map[string]any `json:"jobs"`
		Errors []map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	// Live network may fail or return 404; either way the company is recorded.
	list, _ := store.ListScrapeCompanies()
	require.Len(t, list, 1)
	assert.NotEqual(t, "never", list[0].LastScrapeStatus)
	_ = c
}
```

NOTE: this test hits the live Greenhouse endpoint with a bogus token, which returns 404 → `error` status. It is gated behind `testing.Short()` (already in the snippet) so offline runs skip it. Run network tests without `-short`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mcp/ -run TestFetchATSJobsRecordsStatus -v`
Expected: FAIL — tool `fetch_ats_jobs` not found (stub registers nothing).

- [ ] **Step 3: Implement the tool**

In `internal/mcp/scrape_tools.go`, remove the `registerFetchATSJobs` stub, extend the imports, and implement:

```go
import (
	"context"
	"encoding/json"
	"time"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/scrape"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerFetchATSJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"fetch_ats_jobs",
			mcplib.WithDescription("Fetch current job listings from all registered ATS companies. Returns a compact list of {company_id, company, title, location, url} plus per-company errors. Records each company's scrape status. Apply timezone/region/role judgment to the returned list, then call save_scraped_jobs with the matches."),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			companies, err := s.store.ListScrapeCompanies()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			reg := scrape.Registry()

			type outJob struct {
				CompanyID int64  `json:"company_id"`
				Company   string `json:"company"`
				Title     string `json:"title"`
				Location  string `json:"location"`
				URL       string `json:"url"`
			}
			type outErr struct {
				Company string `json:"company"`
				Error   string `json:"error"`
			}
			var jobs []outJob
			var errs []outErr

			for i, c := range companies {
				if i > 0 {
					time.Sleep(jitter())
				}
				now := time.Now().Format("2006-01-02 15:04:05")
				adapter, ok := reg[c.ATSPlatform]
				if !ok {
					msg := "Unsupported platform: " + c.ATSPlatform
					s.store.UpdateScrapeCompanyStatus(c.ID, now, "error", msg, 0) //nolint:errcheck
					errs = append(errs, outErr{Company: c.Name, Error: msg})
					continue
				}
				fetched, ferr := adapter.FetchJobs(ctx, c.ATSToken)
				if ferr != nil {
					msg := scrape.ClassifyError(c.ATSPlatform, ferr)
					s.store.UpdateScrapeCompanyStatus(c.ID, now, "error", msg, 0) //nolint:errcheck
					errs = append(errs, outErr{Company: c.Name, Error: msg})
					continue
				}
				kept := 0
				for _, j := range fetched {
					if !scrape.HasRemoteSignal(j.Location) {
						continue // coarse pre-filter; Claude does the real judgment
					}
					jobs = append(jobs, outJob{
						CompanyID: c.ID, Company: c.Name,
						Title: j.Title, Location: j.Location, URL: j.URL,
					})
					kept++
				}
				s.store.UpdateScrapeCompanyStatus(c.ID, now, "ok", "", kept) //nolint:errcheck
			}

			payload := map[string]any{"jobs": jobs, "errors": errs}
			b, _ := json.Marshal(payload)
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}

// jitter returns a small randomized delay between company fetches.
func jitter() time.Duration {
	return time.Duration(300+randInt(700)) * time.Millisecond
}
```

Add a `randInt` helper and `HasRemoteSignal` (in the scrape package) — see Step 4.

- [ ] **Step 4: Add `randInt` and `HasRemoteSignal`**

Add to `internal/mcp/scrape_tools.go`:

```go
import "math/rand"

func randInt(n int) int { return rand.Intn(n) }
```

Create `internal/scrape/filter.go`:

```go
package scrape

import "strings"

// HasRemoteSignal is a loose coarse filter: it keeps anything that might be
// remote so Claude can make the real call. It only drops listings that clearly
// state an on-site location with no remote hint.
func HasRemoteSignal(location string) bool {
	if strings.TrimSpace(location) == "" {
		return true // unknown — let Claude decide
	}
	l := strings.ToLower(location)
	return strings.Contains(l, "remote") ||
		strings.Contains(l, "anywhere") ||
		strings.Contains(l, "worldwide") ||
		strings.Contains(l, "distributed") ||
		strings.Contains(l, "global")
}
```

Add a test `internal/scrape/filter_test.go`:

```go
package scrape

import "testing"

func TestHasRemoteSignal(t *testing.T) {
	keep := []string{"", "Remote", "Remote - APAC", "Anywhere", "Worldwide", "Global"}
	drop := []string{"New York, NY", "London", "San Francisco"}
	for _, s := range keep {
		if !HasRemoteSignal(s) {
			t.Errorf("expected keep: %q", s)
		}
	}
	for _, s := range drop {
		if HasRemoteSignal(s) {
			t.Errorf("expected drop: %q", s)
		}
	}
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/scrape/ -run TestHasRemoteSignal -v`
Expected: PASS

Run: `go test ./internal/mcp/ -run TestFetchATSJobsRecordsStatus -v`
Expected: PASS (with network) — company status becomes `error` for the bogus token.

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/scrape_tools.go internal/mcp/scrape_tools_test.go internal/scrape/filter.go internal/scrape/filter_test.go
git commit -m "feat: fetch_ats_jobs MCP tool + coarse remote pre-filter"
```

---

## Task 11: `save_scraped_jobs` tool

**Files:**
- Modify: `internal/mcp/scrape_tools.go` (replace the `registerSaveScrapedJobs` stub)
- Test: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/mcp/scrape_tools_test.go`:

```go
func TestSaveScrapedJobs(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})

	jobsJSON := fmt.Sprintf(`[
		{"company_id":%d,"title":"SRE","location":"Remote","url":"https://j/1","match_reason":"worldwide"},
		{"company_id":%d,"title":"DevOps","location":"Remote APAC","url":"https://j/2","match_reason":"APAC includes UTC+7"}
	]`, c.ID, c.ID)

	out, err := srv.CallTool(context.Background(), "save_scraped_jobs", map[string]any{"jobs": jobsJSON})
	require.NoError(t, err)
	assert.Contains(t, out, "2")

	jobs, _ := store.ListScrapedJobs("new")
	assert.Len(t, jobs, 2)
}
```

Add `"fmt"` to the imports of `internal/mcp/scrape_tools_test.go` for `fmt.Sprintf`.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mcp/ -run TestSaveScrapedJobs -v`
Expected: FAIL — tool not found.

- [ ] **Step 3: Implement the tool**

In `internal/mcp/scrape_tools.go`, remove the `registerSaveScrapedJobs` stub and implement:

This follows the repo's established array pattern (`write_application_answers` in `question_tools.go`): the argument is a JSON-string the handler unmarshals.

```go
func (s *Server) registerSaveScrapedJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"save_scraped_jobs",
			mcplib.WithDescription("Save matched jobs (those that pass the user's timezone/region/role rule). Jobs whose URL already exists are ignored. Returns counts of new vs already-seen."),
			mcplib.WithString("jobs", mcplib.Required(),
				mcplib.Description(`JSON array of {"company_id": <id>, "title": "...", "location": "...", "url": "...", "match_reason": "..."}`)),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			jobsJSON := req.GetString("jobs", "")
			var entries []struct {
				CompanyID   int64  `json:"company_id"`
				Title       string `json:"title"`
				Location    string `json:"location"`
				URL         string `json:"url"`
				MatchReason string `json:"match_reason"`
			}
			if err := json.Unmarshal([]byte(jobsJSON), &entries); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("invalid jobs JSON: %v", err)), nil
			}
			var jobs []db.ScrapedJob
			for _, e := range entries {
				jobs = append(jobs, db.ScrapedJob{
					CompanyID:   e.CompanyID,
					CompanyName: s.companyNameByID(e.CompanyID),
					Title:       e.Title,
					Location:    e.Location,
					URL:         e.URL,
					MatchReason: e.MatchReason,
				})
			}
			added, err := s.store.SaveScrapedJobs(jobs)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			b, _ := json.Marshal(map[string]any{"new": added, "already_seen": len(jobs) - added})
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}

func (s *Server) companyNameByID(id int64) string {
	companies, err := s.store.ListScrapeCompanies()
	if err != nil {
		return ""
	}
	for _, c := range companies {
		if c.ID == id {
			return c.Name
		}
	}
	return ""
}
```

This adds `"fmt"` to the file's imports (for the error message).

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/mcp/ -run TestSaveScrapedJobs -v`
Expected: PASS

- [ ] **Step 5: Run the whole mcp package**

Run: `go test ./internal/mcp/...`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/mcp/scrape_tools.go internal/mcp/scrape_tools_test.go
git commit -m "feat: save_scraped_jobs MCP tool"
```

---

## Task 12: Wire apply linkage into create_application

**Files:**
- Modify: `internal/mcp/application_tools.go`
- Test: `internal/mcp/scrape_tools_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/mcp/scrape_tools_test.go`:

```go
func TestCreateApplicationLinksScrapedJob(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://boards.greenhouse.io/acme/jobs/42"},
	})

	_, err := srv.CallTool(context.Background(), "create_application", map[string]any{
		"company":              "Acme",
		"role":                 "SRE",
		"date":                 "2026-06-08",
		"cv_content":           "<h1>CV</h1>",
		"cover_letter_content": "<p>CL</p>",
		"job_post_content":     "JD",
		"job_url":              "https://boards.greenhouse.io/acme/jobs/42?utm_source=x",
	})
	require.NoError(t, err)

	// The scraped job should now be 'applied' and out of 'new'.
	newJobs, _ := store.ListScrapedJobs("new")
	assert.Len(t, newJobs, 0)
	applied, _ := store.ListScrapedJobs("applied")
	require.Len(t, applied, 1)
	assert.True(t, applied[0].AppliedApplicationID.Valid)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/mcp/ -run TestCreateApplicationLinksScrapedJob -v`
Expected: FAIL — scraped job still in `new` (no linkage wired yet).

- [ ] **Step 3: Wire the linkage**

In `internal/mcp/application_tools.go`, find where `UpsertApplication` returns `id` (around line 57-69) and add the linkage call immediately after the existing `AddStatusHistory` call:

```go
			if err := s.store.AddStatusHistory(id, "pending", ""); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("setting initial status: %v", err)), nil
			}

			if jobURL != "" {
				s.store.LinkApplicationToScrapedJob(id, jobURL) //nolint:errcheck
			}
```

(`jobURL` is already a local variable in this handler.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/mcp/ -run TestCreateApplicationLinksScrapedJob -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mcp/application_tools.go internal/mcp/scrape_tools_test.go
git commit -m "feat: link scraped job to application on create_application"
```

---

# Phase 4 — Dashboard backend

## Task 13: REST handlers for companies, roles, prefs

**Files:**
- Create: `internal/dashboard/handlers_scrape.go`
- Modify: `internal/dashboard/server.go:52-96` (routes)
- Test: `internal/dashboard/handlers_scrape_test.go`

- [ ] **Step 1: Add routes**

In `internal/dashboard/server.go` `routes()`, add after the sources routes:

```go
	s.mux.HandleFunc("GET /api/scrape/companies", s.handleAPIScrapeCompaniesList)
	s.mux.HandleFunc("POST /api/scrape/companies", s.handleAPIScrapeCompanyCreate)
	s.mux.HandleFunc("DELETE /api/scrape/companies/{id}", s.handleAPIScrapeCompanyDelete)
	s.mux.HandleFunc("GET /api/scrape/roles", s.handleAPIScrapeRolesList)
	s.mux.HandleFunc("POST /api/scrape/roles", s.handleAPIScrapeRoleCreate)
	s.mux.HandleFunc("DELETE /api/scrape/roles/{id}", s.handleAPIScrapeRoleDelete)
	s.mux.HandleFunc("GET /api/scrape/prefs", s.handleAPIScrapePrefsGet)
	s.mux.HandleFunc("POST /api/scrape/prefs", s.handleAPIScrapePrefsUpdate)
	s.mux.HandleFunc("GET /api/scrape/jobs", s.handleAPIScrapeJobsList)
	s.mux.HandleFunc("POST /api/scrape/jobs/{id}/status", s.handleAPIScrapeJobStatus)
```

- [ ] **Step 2: Write the failing test**

Create `internal/dashboard/handlers_scrape_test.go`. The existing `newTestServer(t)` helper (in `internal/dashboard/handlers_test.go`) returns a `*dashboard.Server` driven via `httptest.NewRequest` + `srv.ServeHTTP` — match that pattern:

```go
package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrapeCompanyAddDetectsPlatform(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("careers_url=https://boards.greenhouse.io/acme")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "greenhouse")

	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/companies", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "acme")
}

func TestScrapeCompanyAddRejectsUnsupported(t *testing.T) {
	srv := newTestServer(t)
	form := strings.NewReader("careers_url=https://acme.com/careers")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScrapeRoleAddAndPrefs(t *testing.T) {
	srv := newTestServer(t)

	roleReq := httptest.NewRequest(http.MethodPost, "/api/scrape/roles", strings.NewReader("keyword=SRE"))
	roleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()
	srv.ServeHTTP(rw, roleReq)
	assert.Equal(t, http.StatusCreated, rw.Code)

	prefReq := httptest.NewRequest(http.MethodPost, "/api/scrape/prefs",
		strings.NewReader("home_timezone=UTC%2B7&location_notes=APAC"))
	prefReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pw := httptest.NewRecorder()
	srv.ServeHTTP(pw, prefReq)
	assert.Equal(t, http.StatusNoContent, pw.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/scrape/prefs", nil)
	gw := httptest.NewRecorder()
	srv.ServeHTTP(gw, getReq)
	assert.Contains(t, gw.Body.String(), "UTC+7")
}
```

(`newTestServer` already exists in `handlers_test.go`; do not redefine it.)

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/dashboard/ -run 'TestScrape' -v`
Expected: FAIL — handlers undefined.

- [ ] **Step 4: Implement the handlers**

Create `internal/dashboard/handlers_scrape.go`:

```go
package dashboard

import (
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/scrape"
)

func (s *Server) handleAPIScrapeCompaniesList(w http.ResponseWriter, r *http.Request) {
	cs, err := s.store.ListScrapeCompanies()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if cs == nil {
		cs = []db.ScrapeCompany{}
	}
	writeJSON(w, cs)
}

func (s *Server) handleAPIScrapeCompanyCreate(w http.ResponseWriter, r *http.Request) {
	careersURL := r.FormValue("careers_url")
	name := r.FormValue("name") // optional override
	if careersURL == "" {
		http.Error(w, "careers_url is required", http.StatusBadRequest)
		return
	}
	platform, token, err := scrape.Detect(careersURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if name == "" {
		name = token
	}
	c, err := s.store.AddScrapeCompany(db.ScrapeCompany{
		Name: name, ATSPlatform: platform, ATSToken: token, CareersURL: careersURL,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, c)
}

func (s *Server) handleAPIScrapeCompanyDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteScrapeCompany(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapeRolesList(w http.ResponseWriter, r *http.Request) {
	rs, err := s.store.ListScrapeRoles()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rs == nil {
		rs = []db.ScrapeRole{}
	}
	writeJSON(w, rs)
}

func (s *Server) handleAPIScrapeRoleCreate(w http.ResponseWriter, r *http.Request) {
	keyword := r.FormValue("keyword")
	if keyword == "" {
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return
	}
	role, err := s.store.AddScrapeRole(keyword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	writeJSON(w, role)
}

func (s *Server) handleAPIScrapeRoleDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteScrapeRole(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapePrefsGet(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetScrapePrefs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, p)
}

func (s *Server) handleAPIScrapePrefsUpdate(w http.ResponseWriter, r *http.Request) {
	err := s.store.UpdateScrapePrefs(db.ScrapePrefs{
		HomeTimezone:  r.FormValue("home_timezone"),
		LocationNotes: r.FormValue("location_notes"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIScrapeJobsList(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "new"
	}
	jobs, err := s.store.ListScrapedJobs(status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if jobs == nil {
		jobs = []db.ScrapedJob{}
	}
	writeJSON(w, jobs)
}

func (s *Server) handleAPIScrapeJobStatus(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.SetScrapedJobStatus(id, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/dashboard/ -run 'TestScrape' -v`
Expected: PASS

- [ ] **Step 6: Run the whole dashboard package**

Run: `go test ./internal/dashboard/...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/handlers_scrape.go internal/dashboard/handlers_scrape_test.go internal/dashboard/server.go
git commit -m "feat: dashboard REST handlers for scrape companies/roles/prefs/jobs"
```

---

# Phase 5 — Frontend

## Task 14: API client + router + nav

**Files:**
- Modify: `frontend/src/api.js`
- Modify: `frontend/src/router/index.js`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add the scrape API section**

In `frontend/src/api.js`, add inside the `api` object (after the `sources` block):

```js
  scrape: {
    companies: () => get("/scrape/companies"),
    addCompany: (careersUrl, name) => post("/scrape/companies", { careers_url: careersUrl, name: name ?? "" }),
    deleteCompany: (id) => del(`/scrape/companies/${id}`),
    roles: () => get("/scrape/roles"),
    addRole: (keyword) => post("/scrape/roles", { keyword }),
    deleteRole: (id) => del(`/scrape/roles/${id}`),
    prefs: () => get("/scrape/prefs"),
    updatePrefs: (homeTimezone, locationNotes) =>
      post("/scrape/prefs", { home_timezone: homeTimezone, location_notes: locationNotes }),
    jobs: (status = "new") => get(`/scrape/jobs?status=${status}`),
    setJobStatus: (id, status) => post(`/scrape/jobs/${id}/status`, { status }),
  },
```

- [ ] **Step 2: Add the route**

In `frontend/src/router/index.js`, add to `routes`:

```js
  { path: "/jobs", component: () => import("../views/JobsView.vue") },
```

- [ ] **Step 3: Add the nav link**

In `frontend/src/App.vue`, add after the Applications `RouterLink`:

```html
      <RouterLink to="/jobs"
        active-class="bg-white/10 text-white font-medium"
        class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
      >Jobs</RouterLink>
```

- [ ] **Step 4: Verify the frontend builds**

Run: `just build-frontend-dev`
Expected: Build succeeds (JobsView.vue is created in Task 15; if the build fails on the missing import, proceed to Task 15 then re-run). To avoid a broken intermediate build, do Step 1-3 and Task 15 Step 1 (create the file) before building.

- [ ] **Step 5: Commit (together with Task 15)**

Defer commit to the end of Task 15 so the build is green.

---

## Task 15: JobsView page

**Files:**
- Create: `frontend/src/views/JobsView.vue`

- [ ] **Step 1: Create the view**

Create `frontend/src/views/JobsView.vue`:

```vue
<template>
  <div class="flex items-center justify-between mb-6">
    <h1 class="text-2xl font-bold">Scraped Jobs</h1>
    <label class="flex items-center gap-2 text-sm text-gray-500">
      <input type="checkbox" v-model="showDismissed" @change="load" />
      Show dismissed
    </label>
  </div>

  <div v-if="failedCompanies.length" class="mb-4 rounded-md bg-amber-50 border border-amber-200 px-4 py-2 text-sm text-amber-800">
    ⚠ {{ failedCompanies.length }} of {{ companies.length }} companies failed to scrape —
    <RouterLink to="/settings" class="underline">see Settings</RouterLink>.
  </div>

  <p class="text-xs text-gray-400 mb-4">
    Run <code>/scrape-jobs</code> in Claude Code to refresh this list.
  </p>

  <div class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 text-xs">
        <tr>
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
          <td colspan="6" class="px-4 py-8 text-center text-gray-400">
            No scraped jobs. Run <code>/scrape-jobs</code> in Claude Code.
          </td>
        </tr>
        <tr v-for="job in jobs" :key="job.id" class="hover:bg-gray-50 align-top">
          <td class="px-4 py-3 font-medium">{{ job.company_name }}</td>
          <td class="px-4 py-3">
            <a :href="job.url" target="_blank" rel="noopener" class="text-blue-600 hover:underline">{{ job.title }}</a>
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
import { api } from '../api.js'

const jobs = ref([])
const companies = ref([])
const showDismissed = ref(false)
const openApplyId = ref(null)
const copiedId = ref(null)

const failedCompanies = ref([])

async function load() {
  jobs.value = await api.scrape.jobs(showDismissed.value ? 'dismissed' : 'new')
  companies.value = await api.scrape.companies()
  failedCompanies.value = companies.value.filter(c => c.last_scrape_status === 'error')
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
  await api.scrape.setJobStatus(job.id, status)
  await load()
}

onMounted(load)
</script>
```

- [ ] **Step 2: Build the frontend**

Run: `just build-frontend-dev`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/api.js frontend/src/router/index.js frontend/src/App.vue frontend/src/views/JobsView.vue
git commit -m "feat: Jobs dashboard page + nav + API client"
```

---

## Task 16: Settings sections + Scraped badge

**Files:**
- Modify: `frontend/src/views/SettingsView.vue`
- Modify: `frontend/src/views/AppsView.vue`

- [ ] **Step 1: Read the existing SettingsView structure**

Open `frontend/src/views/SettingsView.vue` and identify how the Sources section is rendered (it lists + adds + deletes via `api.sources`). Mirror that exact structure for the three new sections.

- [ ] **Step 2: Add the three sections to SettingsView**

Add a "Job Scraping" area with three blocks. Insert this template fragment alongside the existing Sources section (adapt class names to match the file's conventions):

```html
<section class="bg-white rounded-lg shadow p-5 mb-6">
  <h2 class="font-semibold mb-3">Scrape Companies</h2>
  <form @submit.prevent="addCompany" class="flex gap-2 mb-3">
    <input v-model="newCompanyURL" type="text" placeholder="Paste a careers URL (Greenhouse, Lever, Ashby, …)"
      class="flex-1 border rounded px-2 py-1.5 text-sm" />
    <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
  </form>
  <p v-if="companyError" class="text-xs text-red-600 mb-2">{{ companyError }}</p>
  <ul class="divide-y divide-gray-100">
    <li v-for="c in companies" :key="c.id" class="flex items-center justify-between py-2 text-sm">
      <div>
        <span class="font-medium">{{ c.name }}</span>
        <span class="ml-2 text-xs bg-gray-100 text-gray-600 rounded px-1.5 py-0.5">{{ c.ats_platform }}</span>
        <div class="text-xs mt-0.5"
          :class="c.last_scrape_status === 'error' ? 'text-red-600' : 'text-gray-400'">
          <template v-if="c.last_scrape_status === 'ok'">✅ {{ c.last_job_count }} jobs · {{ (c.last_scraped_at||'').slice(0,16) }}</template>
          <template v-else-if="c.last_scrape_status === 'error'">🔴 {{ c.last_scrape_error }} · {{ (c.last_scraped_at||'').slice(0,16) }}</template>
          <template v-else>Not scraped yet</template>
        </div>
      </div>
      <button @click="deleteCompany(c.id)" class="text-xs text-gray-400 hover:text-red-600">Delete</button>
    </li>
  </ul>
</section>

<section class="bg-white rounded-lg shadow p-5 mb-6">
  <h2 class="font-semibold mb-3">Target Roles</h2>
  <form @submit.prevent="addRole" class="flex gap-2 mb-3">
    <input v-model="newRole" type="text" placeholder="e.g. SRE, DevOps, Platform Engineer"
      class="flex-1 border rounded px-2 py-1.5 text-sm" />
    <button class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Add</button>
  </form>
  <div class="flex flex-wrap gap-2">
    <span v-for="r in roles" :key="r.id" class="flex items-center gap-1 bg-gray-100 rounded-full px-2.5 py-1 text-xs">
      {{ r.keyword }}
      <button @click="deleteRole(r.id)" class="text-gray-400 hover:text-red-600">×</button>
    </span>
  </div>
</section>

<section class="bg-white rounded-lg shadow p-5 mb-6">
  <h2 class="font-semibold mb-3">Location Preferences</h2>
  <label class="block text-xs text-gray-500 mb-1">Home timezone</label>
  <input v-model="homeTimezone" type="text" placeholder="UTC+7"
    class="border rounded px-2 py-1.5 text-sm w-32 mb-3" />
  <label class="block text-xs text-gray-500 mb-1">Acceptance notes (Claude reads this)</label>
  <textarea v-model="locationNotes" rows="4" class="w-full border rounded px-2 py-1.5 text-sm"
    placeholder="Accept: anywhere/worldwide; any role whose required timezone window includes UTC+7; APAC, Asia, Southeast Asia, Indonesia. Reject: US-only, EMEA-only, Americas-only."></textarea>
  <button @click="savePrefs" class="mt-2 bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Save preferences</button>
  <span v-if="prefsSaved" class="ml-2 text-xs text-green-600">Saved</span>
</section>
```

- [ ] **Step 3: Add the script logic to SettingsView**

In the `<script setup>` of `SettingsView.vue`, add (merge with existing imports/refs):

```js
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const companies = ref([])
const roles = ref([])
const newCompanyURL = ref('')
const newRole = ref('')
const companyError = ref('')
const homeTimezone = ref('')
const locationNotes = ref('')
const prefsSaved = ref(false)

async function loadScrape() {
  companies.value = await api.scrape.companies()
  roles.value = await api.scrape.roles()
  const p = await api.scrape.prefs()
  homeTimezone.value = p.home_timezone
  locationNotes.value = p.location_notes
}

async function addCompany() {
  companyError.value = ''
  try {
    await api.scrape.addCompany(newCompanyURL.value)
    newCompanyURL.value = ''
    await loadScrape()
  } catch (e) {
    companyError.value = 'Could not add — unsupported or invalid careers URL.'
  }
}
async function deleteCompany(id) { await api.scrape.deleteCompany(id); await loadScrape() }
async function addRole() {
  if (!newRole.value.trim()) return
  await api.scrape.addRole(newRole.value.trim()); newRole.value = ''; await loadScrape()
}
async function deleteRole(id) { await api.scrape.deleteRole(id); await loadScrape() }
async function savePrefs() {
  await api.scrape.updatePrefs(homeTimezone.value, locationNotes.value)
  prefsSaved.value = true
  setTimeout(() => { prefsSaved.value = false }, 1500)
}

onMounted(loadScrape)
```

If `SettingsView.vue` already has an `onMounted`, call `loadScrape()` inside the existing one instead of adding a second `onMounted`.

- [ ] **Step 4: Add the Scraped badge to AppsView**

In `frontend/src/views/AppsView.vue`, in the Company cell, add a badge when `app.FromScrape` is true:

```html
          <td class="px-4 py-3 font-medium">
            {{ app.Company }}
            <span v-if="app.FromScrape" class="ml-1.5 text-xs bg-purple-100 text-purple-700 rounded px-1.5 py-0.5 align-middle">Scraped</span>
          </td>
```

- [ ] **Step 5: Build the frontend**

Run: `just build-frontend-dev`
Expected: Build succeeds.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/SettingsView.vue frontend/src/views/AppsView.vue
git commit -m "feat: Settings sections for scraping + Scraped badge on applications"
```

---

## Task 17: The `/scrape-jobs` skill

**Files:**
- Create: `plugins/skills/scrape-jobs.md`

- [ ] **Step 1: Write the skill**

Create `plugins/skills/scrape-jobs.md`:

```markdown
---
name: scrape-jobs
description: Scrape job vacancies from your registered ATS companies, filter them by your timezone/region/role preferences, and save the matches for review in the dashboard.
---

Scrape and filter job vacancies from the user's registered ATS company career pages.

## Process

### 1. Load preferences

Call `get_scrape_preferences`. It returns `companies`, `roles`, `home_timezone`,
and `location_notes`.

Stop early and tell the user to configure the dashboard **Settings → Job
Scraping** sections if any of these are empty:
- no `companies` — "Add at least one company (paste a careers URL) in Settings."
- no `roles` — "Add at least one target role in Settings."
- empty `home_timezone` or `location_notes` — "Set your Location Preferences in Settings."

### 2. Fetch listings

Call `fetch_ats_jobs` (no arguments). It returns:
- `jobs`: `[{company_id, company, title, location, url}]` — already coarse-filtered for a remote signal
- `errors`: `[{company, error}]` — companies that failed this run (already recorded for the dashboard)

Do not abort if some companies errored; continue with the jobs you did get.

### 3. Judge each job

Keep a job only if BOTH hold:

- **Role fit** — the title is a semantic match for one of the user's target
  roles. Match meaning, not substrings: "Site Reliability Engineer" matches an
  "SRE" target; "Developer Relations" does NOT match "DevOps".
- **Location fit** — apply `home_timezone` + `location_notes`. Accept
  worldwide/anywhere; accept any stated timezone window that includes the home
  timezone; accept the regions the notes list as acceptable. Reject roles locked
  to regions that exclude the home timezone (e.g. "US only", "EMEA only").

For each kept job, write a short one-line `match_reason` (e.g. "APAC includes
UTC+7", "worldwide remote", "explicitly Indonesia-friendly").

### 4. Save matches

Call `save_scraped_jobs` with the kept jobs:
`[{company_id, title, location, url, match_reason}]`. It ignores URLs already
saved and returns `{new, already_seen}`.

### 5. Report

Summarize:
- "X new matches saved, Y already seen."
- If any companies errored: "Z companies failed: Acme (token not found), …"
- "Open the dashboard **Jobs** page to review and apply."

## Rules

- Never invent jobs — only judge and save what `fetch_ats_jobs` returned.
- Only save jobs that pass BOTH role and location fit.
- Keep `match_reason` to one short line; it shows in the dashboard.
- Do not fetch full job descriptions here — that happens later via `/apply`.
```

- [ ] **Step 2: Verify the skill is discoverable**

The plugin already registers `./plugins/skills/` in `.claude-plugin/plugin.json`, so no manifest change is needed. Confirm the file parses (valid frontmatter with `name` and `description`).

- [ ] **Step 3: Commit**

```bash
git add plugins/skills/scrape-jobs.md
git commit -m "feat: /scrape-jobs skill"
```

---

# Phase 6 — Additional ATS adapters

Each task here follows the exact shape of Tasks 6-8: add a fixture, write a
parse test, implement the adapter, add it to `Registry()`, run, commit. Full
code is given for each.

## Task 18: SmartRecruiters adapter

**Files:** `internal/scrape/smartrecruiters.go`, `testdata/smartrecruiters.json`, `smartrecruiters_test.go`, modify `scrape.go` registry.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/smartrecruiters.json`:

```json
{
  "content": [
    {
      "name": "Infrastructure Engineer",
      "location": { "city": "Remote", "country": "Worldwide" },
      "ref": "https://jobs.smartrecruiters.com/acme/infra-1"
    }
  ]
}
```

- [ ] **Step 2: Test** — `internal/scrape/smartrecruiters_test.go`:

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

func TestSmartRecruitersParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/smartrecruiters.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := SmartRecruiters{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Infrastructure Engineer", jobs[0].Title)
	assert.Equal(t, "Remote, Worldwide", jobs[0].Location)
	assert.Equal(t, "https://jobs.smartrecruiters.com/acme/infra-1", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/smartrecruiters.go`:

```go
package scrape

import (
	"context"
	"fmt"
	"strings"
)

type SmartRecruiters struct{ BaseURL string }

func (s SmartRecruiters) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := s.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.smartrecruiters.com/v1/companies/%s/postings", token)
	}
	var resp struct {
		Content []struct {
			Name     string `json:"name"`
			Ref      string `json:"ref"`
			Location struct {
				City    string `json:"city"`
				Country string `json:"country"`
			} `json:"location"`
		} `json:"content"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Content))
	for _, j := range resp.Content {
		loc := strings.TrimPrefix(strings.Join([]string{j.Location.City, j.Location.Country}, ", "), ", ")
		jobs = append(jobs, Job{Title: j.Name, Location: loc, URL: j.Ref})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"smartrecruiters": SmartRecruiters{},` to `Registry()`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestSmartRecruitersParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/smartrecruiters.go internal/scrape/smartrecruiters_test.go internal/scrape/testdata/smartrecruiters.json internal/scrape/scrape.go && git commit -m "feat: SmartRecruiters adapter"`

---

## Task 19: Workable adapter

**Files:** `internal/scrape/workable.go`, `testdata/workable.json`, `workable_test.go`, modify registry.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/workable.json`:

```json
{
  "jobs": [
    { "title": "DevOps Engineer", "location": { "location_str": "Remote, Indonesia" }, "url": "https://apply.workable.com/acme/j/ABC123" }
  ]
}
```

- [ ] **Step 2: Test** — `internal/scrape/workable_test.go`:

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

func TestWorkableParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/workable.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := Workable{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Remote, Indonesia", jobs[0].Location)
	assert.Equal(t, "https://apply.workable.com/acme/j/ABC123", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/workable.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

type Workable struct{ BaseURL string }

func (wk Workable) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := wk.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", token)
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			URL      string `json:"url"`
			Location struct {
				LocationStr string `json:"location_str"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location.LocationStr, URL: j.URL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"workable": Workable{},`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestWorkableParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/workable.go internal/scrape/workable_test.go internal/scrape/testdata/workable.json internal/scrape/scrape.go && git commit -m "feat: Workable adapter"`

---

## Task 20: Recruitee adapter

**Files:** `internal/scrape/recruitee.go`, `testdata/recruitee.json`, `recruitee_test.go`, modify registry.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/recruitee.json`:

```json
{
  "offers": [
    { "title": "Cloud Engineer", "location": "Remote - Asia", "careers_url": "https://acme.recruitee.com/o/cloud-engineer" }
  ]
}
```

- [ ] **Step 2: Test** — `internal/scrape/recruitee_test.go`:

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

func TestRecruiteeParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/recruitee.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := Recruitee{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Cloud Engineer", jobs[0].Title)
	assert.Equal(t, "Remote - Asia", jobs[0].Location)
	assert.Equal(t, "https://acme.recruitee.com/o/cloud-engineer", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/recruitee.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

type Recruitee struct{ BaseURL string }

func (rc Recruitee) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := rc.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.recruitee.com/api/offers/", token)
	}
	var resp struct {
		Offers []struct {
			Title      string `json:"title"`
			Location   string `json:"location"`
			CareersURL string `json:"careers_url"`
		} `json:"offers"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Offers))
	for _, j := range resp.Offers {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location, URL: j.CareersURL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"recruitee": Recruitee{},`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestRecruiteeParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/recruitee.go internal/scrape/recruitee_test.go internal/scrape/testdata/recruitee.json internal/scrape/scrape.go && git commit -m "feat: Recruitee adapter"`

---

## Task 21: Breezy HR adapter

**Files:** `internal/scrape/breezy.go`, `testdata/breezy.json`, `breezy_test.go`, modify registry.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/breezy.json`:

```json
[
  { "name": "Backend Engineer", "url": "https://acme.breezy.hr/p/abc", "location": { "name": "Remote" } }
]
```

- [ ] **Step 2: Test** — `internal/scrape/breezy_test.go`:

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

func TestBreezyParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/breezy.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := Breezy{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Backend Engineer", jobs[0].Title)
	assert.Equal(t, "Remote", jobs[0].Location)
	assert.Equal(t, "https://acme.breezy.hr/p/abc", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/breezy.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

type Breezy struct{ BaseURL string }

func (b Breezy) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := b.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.breezy.hr/json", token)
	}
	var resp []struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp))
	for _, j := range resp {
		jobs = append(jobs, Job{Title: j.Name, Location: j.Location.Name, URL: j.URL})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"breezy": Breezy{},`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestBreezyParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/breezy.go internal/scrape/breezy_test.go internal/scrape/testdata/breezy.json internal/scrape/scrape.go && git commit -m "feat: Breezy HR adapter"`

---

## Task 22: BambooHR adapter

**Files:** `internal/scrape/bamboohr.go`, `testdata/bamboohr.json`, `bamboohr_test.go`, modify registry.

NOTE: BambooHR's list endpoint returns job ids + titles + a location object but
not absolute URLs; construct the URL as `https://{token}.bamboohr.com/careers/{id}`.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/bamboohr.json`:

```json
{
  "result": [
    { "id": "55", "jobOpeningName": "DevOps Engineer", "location": { "city": "Remote" } }
  ]
}
```

- [ ] **Step 2: Test** — `internal/scrape/bamboohr_test.go`:

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

func TestBambooHRParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/bamboohr.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := BambooHR{BaseURL: srv.URL, Token: "acme"}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Remote", jobs[0].Location)
	assert.Equal(t, "https://acme.bamboohr.com/careers/55", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/bamboohr.go`:

```go
package scrape

import (
	"context"
	"fmt"
)

// BambooHR's list endpoint omits absolute URLs, so we build them from the token.
// Token is set by the test; in production FetchJobs receives it as the argument.
type BambooHR struct {
	BaseURL string
	Token   string // test override for URL construction
}

func (b BambooHR) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	if b.Token != "" {
		token = b.Token
	}
	base := b.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.bamboohr.com/careers/list", token)
	}
	var resp struct {
		Result []struct {
			ID             string `json:"id"`
			JobOpeningName string `json:"jobOpeningName"`
			Location       struct {
				City string `json:"city"`
			} `json:"location"`
		} `json:"result"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Result))
	for _, j := range resp.Result {
		url := fmt.Sprintf("https://%s.bamboohr.com/careers/%s", token, j.ID)
		jobs = append(jobs, Job{Title: j.JobOpeningName, Location: j.Location.City, URL: url})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"bamboohr": BambooHR{},`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestBambooHRParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/bamboohr.go internal/scrape/bamboohr_test.go internal/scrape/testdata/bamboohr.json internal/scrape/scrape.go && git commit -m "feat: BambooHR adapter"`

---

## Task 23: Personio adapter (XML)

**Files:** `internal/scrape/personio.go`, `testdata/personio.xml`, `personio_test.go`, modify registry.

- [ ] **Step 1: Fixture** — `internal/scrape/testdata/personio.xml`:

```xml
<?xml version="1.0" encoding="utf-8"?>
<workzag-jobs>
  <position>
    <id>9001</id>
    <name>Platform Engineer</name>
    <office>Remote</office>
  </position>
</workzag-jobs>
```

- [ ] **Step 2: Test** — `internal/scrape/personio_test.go`:

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

func TestPersonioParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/personio.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := Personio{BaseURL: srv.URL, Token: "acme"}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Remote", jobs[0].Location)
	assert.Equal(t, "https://acme.jobs.personio.de/job/9001", jobs[0].URL)
}
```

- [ ] **Step 3: Implement** — `internal/scrape/personio.go`:

```go
package scrape

import (
	"context"
	"encoding/xml"
	"fmt"
)

type Personio struct {
	BaseURL string
	Token   string // test override for URL construction
}

func (p Personio) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	if p.Token != "" {
		token = p.Token
	}
	base := p.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.jobs.personio.de/xml", token)
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Positions []struct {
			ID     string `xml:"id"`
			Name   string `xml:"name"`
			Office string `xml:"office"`
		} `xml:"position"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	jobs := make([]Job, 0, len(doc.Positions))
	for _, j := range doc.Positions {
		url := fmt.Sprintf("https://%s.jobs.personio.de/job/%s", token, j.ID)
		jobs = append(jobs, Job{Title: j.Name, Location: j.Office, URL: url})
	}
	return jobs, nil
}
```

- [ ] **Step 4: Registry** — add `"personio": Personio{},`.
- [ ] **Step 5: Run** — `go test ./internal/scrape/ -run TestPersonioParse -v` → PASS
- [ ] **Step 6: Commit** — `git add internal/scrape/personio.go internal/scrape/personio_test.go internal/scrape/testdata/personio.xml internal/scrape/scrape.go && git commit -m "feat: Personio adapter (XML)"`

---

## Task 24: Workday adapter (rod + stealth)

**Files:** `internal/scrape/workday.go`, `workday_test.go`, modify registry.

NOTE: Workday is tenant-specific and bot-defended. The token stored is the
tenant subdomain (e.g. `acme` from `acme.wd1.myworkdayjobs.com`). The careers
URL contains both the data-center segment (`wd1`) and the site name (e.g.
`External`). Store the full careers URL and derive the CXS endpoint from it.
Because Workday requires the rendered site / a POST with a real browser, this
adapter uses rod (mirroring `internal/pdf/chrome.go`). The unit test only
covers URL/endpoint derivation, not a live fetch (no network/browser in tests).

- [ ] **Step 1: Write the derivation test** — `internal/scrape/workday_test.go`:

```go
package scrape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkdayEndpointFromCareersURL(t *testing.T) {
	ep, err := workdayEndpoint("https://acme.wd1.myworkdayjobs.com/External")
	require.NoError(t, err)
	assert.Equal(t, "https://acme.wd1.myworkdayjobs.com/wday/cxs/acme/External/jobs", ep)
}

func TestWorkdayEndpointInvalid(t *testing.T) {
	_, err := workdayEndpoint("https://acme.com/careers")
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/scrape/ -run TestWorkdayEndpoint -v`
Expected: FAIL — `workdayEndpoint` undefined.

- [ ] **Step 3: Implement** — `internal/scrape/workday.go`:

```go
package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Workday struct {
	CareersURL string // full URL needed to derive the CXS endpoint
}

// workdayEndpoint derives the CXS jobs API endpoint from a Workday careers URL.
// https://acme.wd1.myworkdayjobs.com/External
//   -> https://acme.wd1.myworkdayjobs.com/wday/cxs/acme/External/jobs
func workdayEndpoint(careersURL string) (string, error) {
	u, err := url.Parse(careersURL)
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Host), ".myworkdayjobs.com") {
		return "", fmt.Errorf("%w: not a Workday URL", ErrUnexpectedFormat)
	}
	tenant := strings.Split(u.Host, ".")[0]
	segs := pathSegments(u.Path)
	if tenant == "" || len(segs) == 0 {
		return "", fmt.Errorf("%w: cannot derive Workday site", ErrUnexpectedFormat)
	}
	site := segs[len(segs)-1]
	return fmt.Sprintf("https://%s/wday/cxs/%s/%s/jobs", u.Host, tenant, site), nil
}

func (wd Workday) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	endpoint, err := workdayEndpoint(wd.CareersURL)
	if err != nil {
		return nil, err
	}

	path, found := launcher.LookPath()
	if !found {
		return nil, fmt.Errorf("%w: chrome not found for Workday", ErrNetwork)
	}
	u, err := launcher.New().Bin(path).Headless(true).NoSandbox(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("%w: launch chrome: %v", ErrNetwork, err)
	}
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("%w: connect chrome: %v", ErrNetwork, err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("%w: open page: %v", ErrNetwork, err)
	}
	defer page.Close()

	// POST the CXS endpoint from within the browser context so Workday sees a
	// real Chrome session. The endpoint returns JSON of job postings.
	js := fmt.Sprintf(`async () => {
		const res = await fetch(%q, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({appliedFacets:{}, limit:100, offset:0, searchText:''})
		});
		return await res.text();
	}`, endpoint)

	time.Sleep(jitterScrape())
	res, err := page.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("%w: workday fetch: %v", ErrBlocked, err)
	}

	var payload struct {
		JobPostings []struct {
			Title         string `json:"title"`
			LocationsText string `json:"locationsText"`
			ExternalPath  string `json:"externalPath"`
		} `json:"jobPostings"`
	}
	if err := json.Unmarshal([]byte(res.Value.Str()), &payload); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}

	host := ""
	if pu, perr := url.Parse(wd.CareersURL); perr == nil {
		host = pu.Scheme + "://" + pu.Host
	}
	jobs := make([]Job, 0, len(payload.JobPostings))
	for _, j := range payload.JobPostings {
		jobs = append(jobs, Job{
			Title:    j.Title,
			Location: j.LocationsText,
			URL:      host + j.ExternalPath,
		})
	}
	return jobs, nil
}

// jitterScrape is a local delay before the Workday call (kept separate from the
// mcp-layer jitter to avoid a cross-package dependency).
func jitterScrape() time.Duration { return 800 * time.Millisecond }
```

NOTE: The `Workday` adapter needs the company's `CareersURL`, which the
zero-value `Workday{}` in `Registry()` does not carry. Handle this in
`fetch_ats_jobs`: when `c.ATSPlatform == "workday"`, construct
`scrape.Workday{CareersURL: c.CareersURL}` explicitly instead of using the
registry entry. Update Task 10's loop accordingly (see Step 4 below).

- [ ] **Step 4: Special-case Workday in fetch_ats_jobs**

In `internal/mcp/scrape_tools.go`, inside the company loop, replace the adapter
lookup with:

```go
				var adapter scrape.Adapter
				if c.ATSPlatform == "workday" {
					adapter = scrape.Workday{CareersURL: c.CareersURL}
				} else {
					a, ok := reg[c.ATSPlatform]
					if !ok {
						msg := "Unsupported platform: " + c.ATSPlatform
						s.store.UpdateScrapeCompanyStatus(c.ID, now, "error", msg, 0) //nolint:errcheck
						errs = append(errs, outErr{Company: c.Name, Error: msg})
						continue
					}
					adapter = a
				}
```

Keep `"workday": Workday{}` out of `Registry()` (or leave it — the special-case
above takes precedence). For clarity, do NOT add Workday to `Registry()`.

- [ ] **Step 5: Run the derivation test**

Run: `go test ./internal/scrape/ -run TestWorkdayEndpoint -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/scrape/workday.go internal/scrape/workday_test.go internal/mcp/scrape_tools.go
git commit -m "feat: Workday adapter via rod + browser-context CXS fetch"
```

---

# Phase 7 — Integration & docs

## Task 25: Full build, test sweep, and docs

**Files:**
- Modify: `README.md` (document the new tools + skill)
- Modify: `CLAUDE.md` (note the new `internal/scrape` package)

- [ ] **Step 1: Run the full Go test suite**

Run: `go test ./...`
Expected: PASS (network-dependent tests skip with `-short`; run `go test -short ./...` in offline environments).

- [ ] **Step 2: Run the full build**

Run: `just build`
Expected: Frontend builds, Go binary builds, no errors.

- [ ] **Step 3: Document in README**

In `README.md`, add the three new MCP tools to the tools list (`get_scrape_preferences`, `fetch_ats_jobs`, `save_scraped_jobs`) and a short "Job Scraping" usage section: configure companies/roles/location in Settings, run `/scrape-jobs`, review on the Jobs page, click Apply.

- [ ] **Step 4: Document in CLAUDE.md**

In `CLAUDE.md` under "Internal package layout", add:

```
- `internal/scrape` — ATS adapters (Greenhouse, Lever, Ashby, SmartRecruiters, Workable, Recruitee, Breezy, BambooHR, Personio, Workday), URL detection/normalization, and the coarse remote pre-filter
```

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md
git commit -m "docs: document job scraping tools, skill, and scrape package"
```

- [ ] **Step 6: Manual smoke test (optional, requires real tokens)**

1. `just dev` to build + start the dashboard.
2. In Settings, add a real Greenhouse company (e.g. paste a known `boards.greenhouse.io/<token>` URL), add a role ("DevOps"), set home timezone "UTC+7" + notes.
3. Run `/scrape-jobs` in Claude Code.
4. Confirm matches appear on the Jobs page with reasons; confirm a failed/bogus company shows red in Settings.
5. Click Apply, copy `/apply <url>`, run it, and confirm the new application shows the "Scraped" badge and the job leaves the Jobs list.

---

## Self-Review Notes (for the implementer)

- **Type consistency:** `db.ScrapedJob.AppliedApplicationID` is `sql.NullInt64`; the frontend reads `applied_application_id` only indirectly (never rendered directly). `FromScrape` is the only application-facing flag.
- **URL normalization** is defined twice intentionally (`db.NormalizeJobURL` and `scrape.NormalizeURL`) to avoid a db→scrape import cycle; keep their logic identical. If you change one, change both, and they share the test expectation in `TestNormalizeURL` / `TestApplyLinkageByURL`.
- **Registry vs Workday:** Workday is special-cased in `fetch_ats_jobs` because it needs `CareersURL`; do not rely on the registry zero-value for it.
- **Network tests:** `TestFetchATSJobsRecordsStatus` hits live Greenhouse; gate it behind `if testing.Short() { t.Skip() }`.
