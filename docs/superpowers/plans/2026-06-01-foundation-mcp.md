# Prospector — Foundation & MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the core Prospector Go binary with config, SQLite database, file system layer, markdown parser, Gotenberg PDF client, and full MCP server — giving Claude Code all 10 tools needed to manage CVs, cover letters, and applications.

**Architecture:** Single Go binary (`prospector`) built with Cobra subcommands. Internal packages are strictly layered: `config` and `fs` have no dependencies on other internal packages; `db` and `parser` depend only on `config`; `pdf` is standalone; `mcp` depends on all of them. Tests use a temp directory to avoid touching `~/.config/prospector`.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra` (CLI), `modernc.org/sqlite` (pure-Go SQLite, no CGO), `github.com/BurntSushi/toml` (config), `github.com/mark3labs/mcp-go` (MCP protocol), `gopkg.in/yaml.v3` (CV frontmatter), `github.com/stretchr/testify` (assertions)

---

## File Map

```
prospector/
  go.mod
  go.sum
  cmd/prospector/
    main.go              ← cobra root + registers all subcommands
    cmd_init.go          ← `prospector init`
    cmd_mcp.go           ← `prospector mcp`
    cmd_up.go            ← `prospector up`
    cmd_sync.go          ← `prospector sync`
  internal/
    config/
      config.go          ← Config struct, Load(), Save(), defaults
      config_test.go
    db/
      db.go              ← Open(), migrations, schema constants
      experience.go      ← UpsertExperience(), ListExperience(), DeleteBySyncedFrom()
      applications.go    ← UpsertApplication(), AddStatusHistory(), GetStats()
      db_test.go
    fs/
      cv.go              ← ListCVs(), ReadCV(), WriteCV()
      coverletter.go     ← ListCoverLetters(), ReadCoverLetter(), WriteCoverLetter()
      application.go     ← CreateApplication(), ListApplications(), ReadMetadata()
      themes.go          ← ListThemes(), ReadTheme()
      fs_test.go
    parser/
      parser.go          ← ParseExperience(markdown) → []ExperienceEntry
      parser_test.go
    pdf/
      client.go          ← Client struct, RenderPDF(html, css) → []byte
      client_test.go
    mcp/
      server.go          ← NewServer(), registers all tools, ServeStdio()
      cv_tools.go        ← list_cvs, read_cv, write_cv handlers
      coverletter_tools.go ← list_cover_letters, read_cover_letter, write_cover_letter handlers
      theme_tools.go     ← list_themes handler
      pdf_tools.go       ← export_pdf handler
      experience_tools.go ← get_experience_stats handler
      application_tools.go ← create_application handler
      mcp_test.go
  docker-compose.yml
  plugin.json
```

---

## Deviations from spec

- **metadata.yaml not metadata.toml**: The spec says `metadata.toml` but the plan uses `metadata.yaml` — we already import `gopkg.in/yaml.v3` for CV frontmatter, so using YAML avoids adding a second config-parsing dependency. Functionally identical.

---

## CV Frontmatter Format

CV markdown files **must** include a YAML frontmatter block so experience data can be extracted reliably. Claude Code maintains this block when customizing CVs.

```markdown
---
experience:
  - role_type: backend
    company: "Acme Corp"
    start: "2018-01-01"
    end: "2020-06-01"
  - role_type: devops
    company: "FooCo"
    start: "2022-03-01"
    end: ""        # empty string = ongoing
---

# John Doe
...rest of CV narrative content...
```

---

## Task 1: Go module scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/prospector/main.go`

- [ ] **Step 1: Initialise the Go module**

```bash
cd /home/graditya/projects/prospector
go mod init github.com/oxGrad/spicebag
```

Expected: `go.mod` created with `module github.com/oxGrad/spicebag` and `go 1.22`.

- [ ] **Step 2: Install dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/BurntSushi/toml@latest
go get modernc.org/sqlite@latest
go get github.com/mark3labs/mcp-go@latest
go get gopkg.in/yaml.v3@latest
go get github.com/stretchr/testify@latest
go mod tidy
```

- [ ] **Step 3: Create main.go**

```go
// cmd/prospector/main.go
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "prospector",
		Short: "CV, cover letter, and job application manager for Claude Code",
	}

	root.AddCommand(
		newInitCmd(),
		newMCPCmd(),
		newUpCmd(),
		newSyncCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./cmd/prospector/
```

Expected: binary `prospector` created in current directory with no errors.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/
git commit -m "feat: scaffold Go module and cobra CLI"
```

---

## Task 2: Config package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.DashboardPort)
	assert.Equal(t, "http://localhost:3000", cfg.GotenbergURL)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := os.WriteFile(path, []byte(`dashboard_port = 9090\ngotenberg_url = "http://gotenberg:3000"\n`), 0644)
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.DashboardPort)
	assert.Equal(t, "http://gotenberg:3000", cfg.GotenbergURL)
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Config{DashboardPort: 7070, GotenbergURL: "http://x:3000"}

	err := config.Save(path, cfg)
	require.NoError(t, err)

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/config/...
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Implement config.go**

```go
// internal/config/config.go
package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DashboardPort int    `toml:"dashboard_port"`
	GotenbergURL  string `toml:"gotenberg_url"`
}

func defaults() Config {
	return Config{
		DashboardPort: 8080,
		GotenbergURL:  "http://localhost:3000",
	}
}

func Load(path string) (Config, error) {
	cfg := defaults()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	_, err = toml.Decode(string(data), &cfg)
	return cfg, err
}

func Save(path string, cfg Config) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(cfg)
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/config/... -v
```

Expected: all 3 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: config package with TOML load/save and defaults"
```

---

## Task 3: SQLite schema + connection

**Files:**
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/db/db_test.go
package db_test

import (
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := db.Open(path)
	require.NoError(t, err)
	defer store.Close()
	assert.NotNil(t, store)
}

func TestOpenRunsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := db.Open(path)
	require.NoError(t, err)
	defer store.Close()

	// tables must exist
	tables := []string{"experience", "applications", "application_status_history"}
	for _, table := range tables {
		var name string
		err := store.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %s missing", table)
		assert.Equal(t, table, name)
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/db/... 2>&1 | head -5
```

Expected: compile error.

- [ ] **Step 3: Implement db.go**

```go
// internal/db/db.go
package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS experience (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  role_type   TEXT NOT NULL,
  company     TEXT NOT NULL,
  start_date  TEXT NOT NULL,
  end_date    TEXT,
  synced_from TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS applications (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  company      TEXT NOT NULL,
  role         TEXT NOT NULL,
  applied_date TEXT NOT NULL,
  base_cv_used TEXT,
  notes        TEXT,
  folder_path  TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS application_status_history (
  id             INTEGER PRIMARY KEY AUTOINCREMENT,
  application_id INTEGER NOT NULL REFERENCES applications(id),
  status         TEXT NOT NULL CHECK(status IN ('applied','assessment','interview','offer','rejected','withdrawn','ghosted')),
  changed_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  notes          TEXT
);
`

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) DB() *sql.DB { return s.db }

func (s *Store) Close() error { return s.db.Close() }
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/db/... -v -run TestOpen
```

Expected: both tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/db/db.go internal/db/db_test.go
git commit -m "feat: SQLite store with schema migrations"
```

---

## Task 4: Experience DB queries

**Files:**
- Create: `internal/db/experience.go`

- [ ] **Step 1: Write failing tests** (add to `internal/db/db_test.go`)

```go
func TestExperience(t *testing.T) {
	store := openTestStore(t)

	entries := []db.ExperienceEntry{
		{RoleType: "backend", Company: "Acme", StartDate: "2018-01-01", EndDate: "2020-06-01", SyncedFrom: "cv-backend.md"},
		{RoleType: "devops", Company: "FooCo", StartDate: "2022-03-01", EndDate: "", SyncedFrom: "cv-backend.md"},
	}

	err := store.UpsertExperience(entries)
	require.NoError(t, err)

	loaded, err := store.ListExperience()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "backend", loaded[0].RoleType)
	assert.Equal(t, "Acme", loaded[0].Company)

	err = store.DeleteExperienceBySyncedFrom("cv-backend.md")
	require.NoError(t, err)

	loaded, err = store.ListExperience()
	require.NoError(t, err)
	assert.Len(t, loaded, 0)
}

// helper used by all db tests
func openTestStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
go test ./internal/db/... -run TestExperience
```

Expected: compile error — `ExperienceEntry`, `UpsertExperience`, `ListExperience`, `DeleteExperienceBySyncedFrom` undefined.

- [ ] **Step 3: Implement experience.go**

```go
// internal/db/experience.go
package db

type ExperienceEntry struct {
	ID         int
	RoleType   string
	Company    string
	StartDate  string
	EndDate    string // empty = ongoing
	SyncedFrom string
}

func (s *Store) UpsertExperience(entries []ExperienceEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, e := range entries {
		_, err = tx.Exec(`
			INSERT INTO experience (role_type, company, start_date, end_date, synced_from)
			VALUES (?, ?, ?, ?, ?)`,
			e.RoleType, e.Company, e.StartDate, e.EndDate, e.SyncedFrom,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) ListExperience() ([]ExperienceEntry, error) {
	rows, err := s.db.Query(`SELECT id, role_type, company, start_date, end_date, synced_from FROM experience ORDER BY start_date`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []ExperienceEntry
	for rows.Next() {
		var e ExperienceEntry
		if err := rows.Scan(&e.ID, &e.RoleType, &e.Company, &e.StartDate, &e.EndDate, &e.SyncedFrom); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (s *Store) DeleteExperienceBySyncedFrom(filename string) error {
	_, err := s.db.Exec(`DELETE FROM experience WHERE synced_from = ?`, filename)
	return err
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/db/... -v -run TestExperience
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/db/experience.go internal/db/db_test.go
git commit -m "feat: experience DB queries (upsert, list, delete by source)"
```

---

## Task 5: Applications + status history DB queries

**Files:**
- Create: `internal/db/applications.go`

- [ ] **Step 1: Write failing tests** (add to `internal/db/db_test.go`)

```go
func TestApplications(t *testing.T) {
	store := openTestStore(t)

	app := db.Application{
		Company:     "Stripe",
		Role:        "Backend Engineer",
		AppliedDate: "2025-05-20",
		BaseCVUsed:  "cv-backend-2025-01.md",
		FolderPath:  "stripe/backend-engineer/2025-05-20",
	}

	id, err := store.UpsertApplication(app)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	apps, err := store.ListApplications()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, "Stripe", apps[0].Company)

	err = store.AddStatusHistory(id, "applied", "")
	require.NoError(t, err)
	err = store.AddStatusHistory(id, "interview", "phone screen")
	require.NoError(t, err)

	history, err := store.GetStatusHistory(id)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, "applied", history[0].Status)
	assert.Equal(t, "interview", history[1].Status)
}

func TestExperienceStats(t *testing.T) {
	store := openTestStore(t)
	entries := []db.ExperienceEntry{
		{RoleType: "backend", Company: "A", StartDate: "2018-01-01", EndDate: "2020-01-01", SyncedFrom: "cv.md"},
		{RoleType: "backend", Company: "B", StartDate: "2022-01-01", EndDate: "2023-01-01", SyncedFrom: "cv.md"},
		{RoleType: "devops", Company: "C", StartDate: "2020-01-01", EndDate: "2022-01-01", SyncedFrom: "cv.md"},
	}
	require.NoError(t, store.UpsertExperience(entries))

	stats, err := store.GetExperienceStats()
	require.NoError(t, err)
	assert.InDelta(t, 3.0, stats.ByRole["backend"], 0.1)
	assert.InDelta(t, 2.0, stats.ByRole["devops"], 0.1)
	assert.InDelta(t, 5.0, stats.Total, 0.1)
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/db/... -run "TestApplications|TestExperienceStats"
```

Expected: compile error.

- [ ] **Step 3: Implement applications.go**

```go
// internal/db/applications.go
package db

import "time"

type Application struct {
	ID          int64
	Company     string
	Role        string
	AppliedDate string
	BaseCVUsed  string
	Notes       string
	FolderPath  string
}

type StatusHistoryEntry struct {
	ID            int64
	ApplicationID int64
	Status        string
	ChangedAt     time.Time
	Notes         string
}

type ExperienceStats struct {
	Total  float64
	ByRole map[string]float64
}

func (s *Store) UpsertApplication(app Application) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO applications (company, role, applied_date, base_cv_used, notes, folder_path)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(folder_path) DO UPDATE SET
			company=excluded.company, role=excluded.role,
			applied_date=excluded.applied_date, base_cv_used=excluded.base_cv_used,
			notes=excluded.notes`,
		app.Company, app.Role, app.AppliedDate, app.BaseCVUsed, app.Notes, app.FolderPath,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) ListApplications() ([]Application, error) {
	rows, err := s.db.Query(`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path FROM applications ORDER BY applied_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

func (s *Store) AddStatusHistory(applicationID int64, status, notes string) error {
	_, err := s.db.Exec(`INSERT INTO application_status_history (application_id, status, notes) VALUES (?, ?, ?)`,
		applicationID, status, notes)
	return err
}

func (s *Store) GetStatusHistory(applicationID int64) ([]StatusHistoryEntry, error) {
	rows, err := s.db.Query(`SELECT id, application_id, status, changed_at, notes FROM application_status_history WHERE application_id = ? ORDER BY changed_at`,
		applicationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []StatusHistoryEntry
	for rows.Next() {
		var h StatusHistoryEntry
		if err := rows.Scan(&h.ID, &h.ApplicationID, &h.Status, &h.ChangedAt, &h.Notes); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}

func (s *Store) GetExperienceStats() (ExperienceStats, error) {
	entries, err := s.ListExperience()
	if err != nil {
		return ExperienceStats{}, err
	}

	now := time.Now()
	stats := ExperienceStats{ByRole: make(map[string]float64)}

	for _, e := range entries {
		start, err := time.Parse("2006-01-02", e.StartDate)
		if err != nil {
			continue
		}
		end := now
		if e.EndDate != "" {
			end, err = time.Parse("2006-01-02", e.EndDate)
			if err != nil {
				continue
			}
		}
		years := end.Sub(start).Hours() / (24 * 365.25)
		stats.ByRole[e.RoleType] += years
		stats.Total += years
	}
	return stats, nil
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
go test ./internal/db/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/db/applications.go internal/db/db_test.go
git commit -m "feat: applications, status history, and experience stats DB queries"
```

---

## Task 6: File system layer — CVs and cover letters

**Files:**
- Create: `internal/fs/cv.go`
- Create: `internal/fs/coverletter.go`
- Create: `internal/fs/fs_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/fs/fs_test.go
package fs_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCVs(t *testing.T) {
	root := t.TempDir()
	cvDir := filepath.Join(root, "cv")

	// write two CV files
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV"))
	require.NoError(t, fs.WriteCV(root, "cv-devops-2025-01-01.md", "# DevOps CV"))

	files, err := fs.ListCVs(root)
	require.NoError(t, err)
	require.Len(t, files, 2)
	_ = cvDir
}

func TestReadCV(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV\nContent here"))

	content, err := fs.ReadCV(root, "cv-backend-2025-01-01.md")
	require.NoError(t, err)
	assert.Equal(t, "# Backend CV\nContent here", content)
}

func TestWriteCVCreatesDir(t *testing.T) {
	root := t.TempDir()
	err := fs.WriteCV(root, "cv-new-2025-01-01.md", "content")
	require.NoError(t, err)

	content, err := fs.ReadCV(root, "cv-new-2025-01-01.md")
	require.NoError(t, err)
	assert.Equal(t, "content", content)
}

func TestListCoverLetters(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.md", "Dear Hiring Manager"))

	files, err := fs.ListCoverLetters(root)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "cl-general-2025-01-01.md", files[0].Name)
}

// FileInfo is used in test assertions below
var _ = time.Now
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/fs/... 2>&1 | head -5
```

Expected: compile error.

- [ ] **Step 3: Implement cv.go**

```go
// internal/fs/cv.go
package fs

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

type FileInfo struct {
	Name       string
	ModifiedAt time.Time
	Size       int64
}

func cvDir(root string) string { return filepath.Join(root, "cv") }

func ListCVs(root string) ([]FileInfo, error) {
	return listMarkdownFiles(cvDir(root))
}

func ReadCV(root, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(cvDir(root), filename))
	return string(data), err
}

func WriteCV(root, filename, content string) error {
	dir := cvDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func listMarkdownFiles(dir string) ([]FileInfo, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{Name: e.Name(), ModifiedAt: info.ModTime(), Size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModifiedAt.After(files[j].ModifiedAt) })
	return files, nil
}
```

- [ ] **Step 4: Implement coverletter.go**

```go
// internal/fs/coverletter.go
package fs

import (
	"os"
	"path/filepath"
)

func coverLetterDir(root string) string { return filepath.Join(root, "cover-letters") }

func ListCoverLetters(root string) ([]FileInfo, error) {
	return listMarkdownFiles(coverLetterDir(root))
}

func ReadCoverLetter(root, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(coverLetterDir(root), filename))
	return string(data), err
}

func WriteCoverLetter(root, filename, content string) error {
	dir := coverLetterDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/fs/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/fs/
git commit -m "feat: file system layer for CVs and cover letters"
```

---

## Task 7: File system layer — applications and themes

**Files:**
- Create: `internal/fs/application.go`
- Create: `internal/fs/themes.go`

- [ ] **Step 1: Write failing tests** (add to `internal/fs/fs_test.go`)

```go
func TestCreateApplication(t *testing.T) {
	root := t.TempDir()
	req := fs.ApplicationRequest{
		Company:            "Stripe",
		Role:               "Backend Engineer",
		Date:               "2025-05-20",
		CVContent:          "# CV",
		CoverLetterContent: "Dear Stripe",
		JobPostContent:     "We are hiring...",
		BaseCVUsed:         "cv-backend-2025-01.md",
	}

	folderPath, err := fs.CreateApplication(root, req)
	require.NoError(t, err)
	assert.Equal(t, "stripe/backend-engineer/2025-05-20", folderPath)

	cv, err := os.ReadFile(filepath.Join(root, "applications", folderPath, "cv.md"))
	require.NoError(t, err)
	assert.Equal(t, "# CV", string(cv))

	meta, err := fs.ReadApplicationMetadata(root, folderPath)
	require.NoError(t, err)
	assert.Equal(t, "Stripe", meta.Company)
}

func TestListThemes(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "themes")
	require.NoError(t, os.MkdirAll(themeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "minimal.css"), []byte("body{}"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "modern.css"), []byte("body{}"), 0644))

	themes, err := fs.ListThemes(root)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"minimal", "modern"}, themes)
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/fs/... -run "TestCreateApplication|TestListThemes"
```

Expected: compile error.

- [ ] **Step 3: Implement application.go**

```go
// internal/fs/application.go
package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ApplicationRequest struct {
	Company            string
	Role               string
	Date               string // YYYY-MM-DD
	CVContent          string
	CoverLetterContent string
	JobPostContent     string
	BaseCVUsed         string
	Notes              string
}

type ApplicationMetadata struct {
	Company     string `yaml:"company"`
	Role        string `yaml:"role"`
	AppliedDate string `yaml:"applied_date"`
	BaseCVUsed  string `yaml:"base_cv_used"`
	Notes       string `yaml:"notes"`
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

func CreateApplication(root string, req ApplicationRequest) (string, error) {
	folderPath := fmt.Sprintf("%s/%s/%s", slugify(req.Company), slugify(req.Role), req.Date)
	dir := filepath.Join(root, "applications", folderPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	files := map[string]string{
		"cv.md":           req.CVContent,
		"cover-letter.md": req.CoverLetterContent,
		"job-post.md":     req.JobPostContent,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return "", err
		}
	}

	meta := ApplicationMetadata{
		Company:     req.Company,
		Role:        req.Role,
		AppliedDate: req.Date,
		BaseCVUsed:  req.BaseCVUsed,
		Notes:       req.Notes,
	}
	metaBytes, err := yaml.Marshal(meta)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.yaml"), metaBytes, 0644); err != nil {
		return "", err
	}
	return folderPath, nil
}

func ReadApplicationMetadata(root, folderPath string) (ApplicationMetadata, error) {
	data, err := os.ReadFile(filepath.Join(root, "applications", folderPath, "metadata.yaml"))
	if err != nil {
		return ApplicationMetadata{}, err
	}
	var meta ApplicationMetadata
	return meta, yaml.Unmarshal(data, &meta)
}
```

- [ ] **Step 4: Implement themes.go**

```go
// internal/fs/themes.go
package fs

import (
	"os"
	"path/filepath"
	"strings"
)

func themeDir(root string) string { return filepath.Join(root, "themes") }

func ListThemes(root string) ([]string, error) {
	entries, err := os.ReadDir(themeDir(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var themes []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".css" {
			themes = append(themes, strings.TrimSuffix(e.Name(), ".css"))
		}
	}
	return themes, nil
}

func ReadTheme(root, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(themeDir(root), name+".css"))
	return string(data), err
}
```

- [ ] **Step 5: Run all fs tests**

```bash
go test ./internal/fs/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/fs/application.go internal/fs/themes.go internal/fs/fs_test.go
git commit -m "feat: application folder creation and theme listing"
```

---

## Task 8: CV frontmatter parser

**Files:**
- Create: `internal/parser/parser.go`
- Create: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/parser/parser_test.go
package parser_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cvWithFrontmatter = `---
experience:
  - role_type: backend
    company: "Acme Corp"
    start: "2018-01-01"
    end: "2020-06-01"
  - role_type: devops
    company: "FooCo"
    start: "2022-03-01"
    end: ""
---

# John Doe
Senior Engineer
`

func TestParseExperience(t *testing.T) {
	entries, err := parser.ParseExperience(cvWithFrontmatter)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "backend", entries[0].RoleType)
	assert.Equal(t, "Acme Corp", entries[0].Company)
	assert.Equal(t, "2018-01-01", entries[0].StartDate)
	assert.Equal(t, "2020-06-01", entries[0].EndDate)

	assert.Equal(t, "devops", entries[1].RoleType)
	assert.Equal(t, "", entries[1].EndDate) // ongoing
}

func TestParseExperienceNoFrontmatter(t *testing.T) {
	entries, err := parser.ParseExperience("# Just a CV\nNo frontmatter here")
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/parser/... 2>&1 | head -5
```

Expected: compile error.

- [ ] **Step 3: Implement parser.go**

```go
// internal/parser/parser.go
package parser

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type ExperienceEntry struct {
	RoleType  string `yaml:"role_type"`
	Company   string `yaml:"company"`
	StartDate string `yaml:"start"`
	EndDate   string `yaml:"end"`
}

type frontmatter struct {
	Experience []ExperienceEntry `yaml:"experience"`
}

func ParseExperience(markdown string) ([]ExperienceEntry, error) {
	if !strings.HasPrefix(markdown, "---") {
		return nil, nil
	}
	// extract content between first --- and second ---
	rest := strings.TrimPrefix(markdown, "---")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return nil, nil
	}
	yamlBlock := strings.TrimSpace(rest[:end])

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, err
	}
	return fm.Experience, nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/parser/... -v
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/parser/
git commit -m "feat: CV frontmatter parser extracts experience entries"
```

---

## Task 9: Gotenberg PDF client

**Files:**
- Create: `internal/pdf/client.go`
- Create: `internal/pdf/client_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/pdf/client_test.go
package pdf_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oxGrad/spicebag/internal/pdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderPDF(t *testing.T) {
	// mock Gotenberg server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/forms/chromium/convert/html", r.URL.Path)
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-fake"))
	}))
	defer srv.Close()

	client := pdf.NewClient(srv.URL)
	result, err := client.RenderPDF("<html><body>Hello</body></html>", "body { color: red; }")
	require.NoError(t, err)
	assert.Equal(t, []byte("%PDF-fake"), result)
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/pdf/... 2>&1 | head -5
```

Expected: compile error.

- [ ] **Step 3: Implement client.go**

```go
// internal/pdf/client.go
package pdf

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}}
}

func (c *Client) RenderPDF(html, css string) ([]byte, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	htmlPart, err := w.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, err
	}
	// inject CSS into HTML
	fullHTML := fmt.Sprintf(`<!DOCTYPE html><html><head><style>%s</style></head><body>%s</body></html>`, css, html)
	if _, err := io.WriteString(htmlPart, fullHTML); err != nil {
		return nil, err
	}
	w.Close()

	resp, err := c.http.Post(c.baseURL+"/forms/chromium/convert/html", w.FormDataContentType(), &body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gotenberg returned %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/pdf/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/pdf/
git commit -m "feat: Gotenberg PDF client"
```

---

## Task 10: MCP server — CV, cover letter, and theme tools

**Files:**
- Create: `internal/mcp/server.go`
- Create: `internal/mcp/cv_tools.go`
- Create: `internal/mcp/coverletter_tools.go`
- Create: `internal/mcp/theme_tools.go`
- Create: `internal/mcp/mcp_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/mcp/mcp_test.go
package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/fs"
	prospectormcp "github.com/oxGrad/spicebag/internal/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (string, *prospectormcp.Server) {
	t.Helper()
	root := t.TempDir()

	// seed test data
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV"))
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.md", "Dear Hiring Manager"))
	themeDir := filepath.Join(root, "themes")
	require.NoError(t, os.MkdirAll(themeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "minimal.css"), []byte("body{}"), 0644))

	dbPath := filepath.Join(root, "prospector.db")
	srv, err := prospectormcp.NewServer(root, dbPath, "http://localhost:3000")
	require.NoError(t, err)
	t.Cleanup(func() { srv.Close() })
	return root, srv
}

func TestListCVsTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "list_cvs", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, result, "cv-backend-2025-01-01.md")
}

func TestReadCVTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "read_cv", map[string]any{"filename": "cv-backend-2025-01-01.md"})
	require.NoError(t, err)
	assert.Contains(t, result, "# Backend CV")
}

func TestWriteCVTool(t *testing.T) {
	root, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "write_cv", map[string]any{
		"filename": "cv-new-2025-06-01.md",
		"content":  "# New CV",
	})
	require.NoError(t, err)

	content, err := fs.ReadCV(root, "cv-new-2025-06-01.md")
	require.NoError(t, err)
	assert.Equal(t, "# New CV", content)
}

func TestListThemesTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "list_themes", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, result, "minimal")
}

// helper to avoid importing mcp package in every test
var _ = mcp.CallToolRequest{}
var _ = json.Marshal
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/mcp/... 2>&1 | head -5
```

Expected: compile error.

- [ ] **Step 3: Implement server.go**

```go
// internal/mcp/server.go
package mcp

import (
	"context"
	"fmt"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	root    string
	store   *db.Store
	gotURL  string
	mcpSrv  *server.MCPServer
}

func NewServer(root, dbPath, gotenbergURL string) (*Server, error) {
	store, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		root:   root,
		store:  store,
		gotURL: gotenbergURL,
		mcpSrv: server.NewMCPServer("prospector", "1.0.0"),
	}

	s.registerCVTools()
	s.registerCoverLetterTools()
	s.registerThemeTools()
	s.registerPDFTools()
	s.registerExperienceTools()
	s.registerApplicationTools()

	return s, nil
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpSrv)
}

func (s *Server) Close() { s.store.Close() }

// Store returns the underlying DB store. Used in tests for seeding data
// without opening a second connection (SQLite allows only one writer).
func (s *Server) Store() *db.Store { return s.store }

// CallTool is used in tests to invoke a tool directly.
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := s.mcpSrv.CallTool(ctx, req)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("tool error: %v", result.Content)
	}
	if len(result.Content) == 0 {
		return "", nil
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("unexpected content type")
	}
	return text.Text, nil
}
```

- [ ] **Step 4: Implement cv_tools.go**

```go
// internal/mcp/cv_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerCVTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("list_cvs", mcp.WithDescription("List base CV files with name and modified date")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			files, err := fs.ListCVs(s.root)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(files)
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcp.NewTool("read_cv",
			mcp.WithDescription("Read a base CV's markdown content"),
			mcp.WithString("filename", mcp.Required(), mcp.Description("CV filename to read")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filename, _ := req.Params.Arguments["filename"].(string)
			content, err := fs.ReadCV(s.root, filename)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(content), nil
		},
	)

	s.mcpSrv.AddTool(
		mcp.NewTool("write_cv",
			mcp.WithDescription("Save a new base CV file"),
			mcp.WithString("filename", mcp.Required(), mcp.Description("Filename e.g. cv-backend-2025-06-01.md")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Full markdown content including frontmatter")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filename, _ := req.Params.Arguments["filename"].(string)
			content, _ := req.Params.Arguments["content"].(string)
			if err := fs.WriteCV(s.root, filename, content); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Saved %s", filename)), nil
		},
	)
}
```

- [ ] **Step 5: Implement coverletter_tools.go**

```go
// internal/mcp/coverletter_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerCoverLetterTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("list_cover_letters", mcp.WithDescription("List standalone cover letter files")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			files, err := fs.ListCoverLetters(s.root)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(files)
			return mcp.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcp.NewTool("read_cover_letter",
			mcp.WithDescription("Read a cover letter's markdown content"),
			mcp.WithString("filename", mcp.Required(), mcp.Description("Cover letter filename to read")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filename, _ := req.Params.Arguments["filename"].(string)
			content, err := fs.ReadCoverLetter(s.root, filename)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(content), nil
		},
	)

	s.mcpSrv.AddTool(
		mcp.NewTool("write_cover_letter",
			mcp.WithDescription("Save a new cover letter file"),
			mcp.WithString("filename", mcp.Required(), mcp.Description("Filename e.g. cl-stripe-2025-06-01.md")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Full markdown content")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filename, _ := req.Params.Arguments["filename"].(string)
			content, _ := req.Params.Arguments["content"].(string)
			if err := fs.WriteCoverLetter(s.root, filename, content); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf("Saved %s", filename)), nil
		},
	)
}
```

- [ ] **Step 6: Implement theme_tools.go**

```go
// internal/mcp/theme_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerThemeTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("list_themes", mcp.WithDescription("List available CSS theme names")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			themes, err := fs.ListThemes(s.root)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(themes)
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/mcp/... -v -run "TestListCVs|TestReadCV|TestWriteCV|TestListThemes"
```

Expected: all 4 tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/mcp/
git commit -m "feat: MCP server with CV, cover letter, and theme tools"
```

---

## Task 11: MCP server — PDF, experience stats, and create_application tools

**Files:**
- Create: `internal/mcp/pdf_tools.go`
- Create: `internal/mcp/experience_tools.go`
- Create: `internal/mcp/application_tools.go`

- [ ] **Step 1: Write failing tests** (add to `internal/mcp/mcp_test.go`)

```go
func TestGetExperienceStatsTool(t *testing.T) {
	_, srv := setup(t)

	// seed via the server's own store — avoids opening a second SQLite connection
	entries := []db.ExperienceEntry{
		{RoleType: "backend", Company: "Acme", StartDate: "2020-01-01", EndDate: "2022-01-01", SyncedFrom: "cv.md"},
	}
	require.NoError(t, srv.Store().UpsertExperience(entries))

	result, err := srv.CallTool(context.Background(), "get_experience_stats", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, result, "backend")
}

func TestCreateApplicationTool(t *testing.T) {
	root, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "create_application", map[string]any{
		"company":              "Stripe",
		"role":                 "Backend Engineer",
		"date":                 "2025-06-01",
		"cv_content":           "# CV",
		"cover_letter_content": "Dear Stripe",
		"job_post_content":     "We are hiring",
		"base_cv_used":         "cv-backend-2025-01-01.md",
	})
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(root, "applications", "stripe", "backend-engineer", "2025-06-01", "cv.md"))
	require.NoError(t, statErr)
}
```

Add this import to the test file's import block:
```go
"github.com/oxGrad/spicebag/internal/db"
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/mcp/... -run "TestGetExperience|TestCreateApplication"
```

Expected: compile error — tools not registered yet.

- [ ] **Step 3: Implement pdf_tools.go**

```go
// internal/mcp/pdf_tools.go
package mcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	pdfpkg "github.com/oxGrad/spicebag/internal/pdf"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerPDFTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("export_pdf",
			mcp.WithDescription("Render a CV or cover letter to PDF via Gotenberg; returns the output file path"),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Relative path to the markdown file under ~/.config/prospector")),
			mcp.WithString("theme", mcp.Required(), mcp.Description("Theme name (without .css extension)")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.Params.Arguments["file_path"].(string)
			theme, _ := req.Params.Arguments["theme"].(string)

			mdBytes, err := os.ReadFile(filepath.Join(s.root, filePath))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("read file: %v", err)), nil
			}

			cssBytes, err := os.ReadFile(filepath.Join(s.root, "themes", theme+".css"))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("read theme: %v", err)), nil
			}

			client := pdfpkg.NewClient(s.gotURL)
			pdfBytes, err := client.RenderPDF(string(mdBytes), string(cssBytes))
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("render PDF: %v", err)), nil
			}

			outPath := filepath.Join(s.root, filePath[:len(filePath)-3]+".pdf")
			if err := os.WriteFile(outPath, pdfBytes, 0644); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("write PDF: %v", err)), nil
			}
			return mcp.NewToolResultText(outPath), nil
		},
	)
}
```

- [ ] **Step 4: Implement experience_tools.go**

```go
// internal/mcp/experience_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerExperienceTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("get_experience_stats", mcp.WithDescription("Return years of experience totals and breakdown by role type")),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			stats, err := s.store.GetExperienceStats()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(stats)
			return mcp.NewToolResultText(string(out)), nil
		},
	)
}
```

- [ ] **Step 5: Implement application_tools.go**

```go
// internal/mcp/application_tools.go
package mcp

import (
	"context"
	"fmt"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerApplicationTools() {
	s.mcpSrv.AddTool(
		mcp.NewTool("create_application",
			mcp.WithDescription("Create full application folder with CV, cover letter, job post, and metadata"),
			mcp.WithString("company", mcp.Required(), mcp.Description("Company name")),
			mcp.WithString("role", mcp.Required(), mcp.Description("Job role title")),
			mcp.WithString("date", mcp.Required(), mcp.Description("Application date YYYY-MM-DD")),
			mcp.WithString("cv_content", mcp.Required(), mcp.Description("Tailored CV markdown content")),
			mcp.WithString("cover_letter_content", mcp.Required(), mcp.Description("Cover letter markdown content")),
			mcp.WithString("job_post_content", mcp.Required(), mcp.Description("Job post content to save")),
			mcp.WithString("base_cv_used", mcp.Description("Base CV filename this was derived from")),
			mcp.WithString("notes", mcp.Description("Optional notes")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			str := func(k string) string { v, _ := req.Params.Arguments[k].(string); return v }

			folderPath, err := fs.CreateApplication(s.root, fs.ApplicationRequest{
				Company:            str("company"),
				Role:               str("role"),
				Date:               str("date"),
				CVContent:          str("cv_content"),
				CoverLetterContent: str("cover_letter_content"),
				JobPostContent:     str("job_post_content"),
				BaseCVUsed:         str("base_cv_used"),
				Notes:              str("notes"),
			})
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			app := db.Application{
				Company:     str("company"),
				Role:        str("role"),
				AppliedDate: str("date"),
				BaseCVUsed:  str("base_cv_used"),
				Notes:       str("notes"),
				FolderPath:  folderPath,
			}
			appID, err := s.store.UpsertApplication(app)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("save to db: %v", err)), nil
			}
			if err := s.store.AddStatusHistory(appID, "applied", ""); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("add status: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Created application at %s", folderPath)), nil
		},
	)
}
```

- [ ] **Step 6: Run all MCP tests**

```bash
go test ./internal/mcp/... -v
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/mcp/pdf_tools.go internal/mcp/experience_tools.go internal/mcp/application_tools.go internal/mcp/mcp_test.go
git commit -m "feat: MCP PDF export, experience stats, and create_application tools"
```

---

## Task 12: CLI commands — init, mcp, up, sync

**Files:**
- Create: `cmd/prospector/cmd_init.go`
- Create: `cmd/prospector/cmd_mcp.go`
- Create: `cmd/prospector/cmd_up.go`
- Create: `cmd/prospector/cmd_sync.go`

These commands wire the internal packages to the CLI. No unit tests for CLI commands — behaviour is verified by running the binary manually.

- [ ] **Step 1: Implement cmd_init.go**

```go
// cmd/prospector/cmd_init.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up ~/.config/prospector and register MCP server with Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()

			dirs := []string{"cv", "cover-letters", "themes", "applications"}
			for _, d := range dirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
					return err
				}
			}

			cfgPath := filepath.Join(root, "config.toml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if err := config.Save(cfgPath, config.Config{DashboardPort: 8080, GotenbergURL: "http://localhost:3000"}); err != nil {
					return err
				}
				fmt.Println("Created", cfgPath)
			}

			dbPath := filepath.Join(root, "prospector.db")
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			store.Close()
			fmt.Println("Initialized database at", dbPath)

			if err := registerMCPServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not register MCP server automatically: %v\n", err)
				fmt.Println("Add this manually to your Claude Code MCP config:")
				printMCPConfig()
			} else {
				fmt.Println("Registered MCP server with Claude Code.")
			}

			fmt.Println("\nNext steps:")
			fmt.Println("  prospector up       # start Gotenberg")
			fmt.Println("  prospector serve    # start dashboard")
			return nil
		},
	}
}

func prospectorRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "prospector")
}

func registerMCPServer() error {
	home, _ := os.UserHomeDir()
	mcpConfigPath := filepath.Join(home, ".claude", "mcp.json")

	var mcpCfg map[string]any
	data, err := os.ReadFile(mcpConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &mcpCfg); err != nil {
			return err
		}
	}
	if mcpCfg == nil {
		mcpCfg = map[string]any{}
	}

	servers, _ := mcpCfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["prospector"] = map[string]any{
		"command": "prospector",
		"args":    []string{"mcp"},
	}
	mcpCfg["mcpServers"] = servers

	out, err := json.MarshalIndent(mcpCfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(mcpConfigPath, out, 0644)
}

func printMCPConfig() {
	fmt.Println(`{
  "mcpServers": {
    "prospector": { "command": "prospector", "args": ["mcp"] }
  }
}`)
}
```

- [ ] **Step 2: Implement cmd_mcp.go**

```go
// cmd/prospector/cmd_mcp.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	prospectormcp "github.com/oxGrad/spicebag/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server (called automatically by Claude Code)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()
			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			srv, err := prospectormcp.NewServer(root, filepath.Join(root, "prospector.db"), cfg.GotenbergURL)
			if err != nil {
				return fmt.Errorf("init MCP server: %w", err)
			}
			defer srv.Close()

			return srv.ServeStdio()
		},
	}
}
```

- [ ] **Step 3: Implement cmd_up.go**

```go
// cmd/prospector/cmd_up.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newUpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Start Gotenberg via docker compose",
		RunE: func(cmd *cobra.Command, args []string) error {
			composePath := filepath.Join(prospectorRoot(), "docker-compose.yml")
			if _, err := os.Stat(composePath); os.IsNotExist(err) {
				return fmt.Errorf("docker-compose.yml not found at %s — run prospector init first", composePath)
			}
			c := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}
```

- [ ] **Step 4: Implement cmd_sync.go**

```go
// cmd/prospector/cmd_sync.go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/oxGrad/spicebag/internal/parser"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Re-parse all base CVs and refresh experience data in SQLite",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()
			store, err := db.Open(filepath.Join(root, "prospector.db"))
			if err != nil {
				return err
			}
			defer store.Close()

			cvFiles, err := fs.ListCVs(root)
			if err != nil {
				return err
			}

			for _, f := range cvFiles {
				content, err := fs.ReadCV(root, f.Name)
				if err != nil {
					fmt.Printf("skip %s: %v\n", f.Name, err)
					continue
				}

				if err := store.DeleteExperienceBySyncedFrom(f.Name); err != nil {
					return err
				}

				entries, err := parser.ParseExperience(content)
				if err != nil {
					fmt.Printf("skip %s (parse error): %v\n", f.Name, err)
					continue
				}

				dbEntries := make([]db.ExperienceEntry, len(entries))
				for i, e := range entries {
					dbEntries[i] = db.ExperienceEntry{
						RoleType:   e.RoleType,
						Company:    e.Company,
						StartDate:  e.StartDate,
						EndDate:    e.EndDate,
						SyncedFrom: f.Name,
					}
				}
				if len(dbEntries) > 0 {
					if err := store.UpsertExperience(dbEntries); err != nil {
						return err
					}
				}
				fmt.Printf("synced %s (%d entries)\n", f.Name, len(dbEntries))
			}
			return nil
		},
	}
}
```

- [ ] **Step 5: Build and smoke test**

```bash
go build ./cmd/prospector/ && ./prospector --help
```

Expected: shows all subcommands: `init`, `mcp`, `up`, `sync`.

```bash
./prospector init
```

Expected: creates `~/.config/prospector/` directories, writes default config, initializes DB, prints next steps.

- [ ] **Step 6: Run all tests**

```bash
go test ./... -v 2>&1 | tail -20
```

Expected: all tests pass, no failures.

- [ ] **Step 7: Commit**

```bash
git add cmd/prospector/
git commit -m "feat: CLI commands init, mcp, up, sync"
```

---

## Task 13: docker-compose.yml and plugin.json

**Files:**
- Create: `docker-compose.yml`
- Create: `plugin.json`

- [ ] **Step 1: Create docker-compose.yml**

```yaml
# docker-compose.yml
services:
  gotenberg:
    image: gotenberg/gotenberg:8
    ports:
      - "3000:3000"
    restart: unless-stopped
```

- [ ] **Step 2: Create plugin.json**

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
  "skills": [
    "skills/customize-cv.md",
    "skills/write-cover-letter.md",
    "skills/apply.md"
  ]
}
```

- [ ] **Step 3: Copy docker-compose.yml in init command**

Open `cmd/prospector/cmd_init.go` and add this block inside the `RunE` function, after the directory creation loop:

```go
// copy docker-compose.yml from binary's embedded assets to ~/.config/prospector/
composeDest := filepath.Join(root, "docker-compose.yml")
if _, err := os.Stat(composeDest); os.IsNotExist(err) {
    src := filepath.Join(execDir(), "docker-compose.yml")
    data, err := os.ReadFile(src)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Warning: could not copy docker-compose.yml: %v\n", err)
    } else {
        os.WriteFile(composeDest, data, 0644)
        fmt.Println("Copied docker-compose.yml to", composeDest)
    }
}
```

Add this helper to `cmd_init.go`:

```go
func execDir() string {
    exe, err := os.Executable()
    if err != nil {
        return "."
    }
    return filepath.Dir(exe)
}
```

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml plugin.json cmd/prospector/cmd_init.go
git commit -m "feat: docker-compose.yml for Gotenberg and plugin.json manifest"
```

---

## Task 14: Final build verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -count=1
```

Expected: all tests pass, no race conditions (add `-race` flag too):

```bash
go test -race ./...
```

- [ ] **Step 2: Build release binary**

```bash
go build -ldflags="-s -w" -o prospector ./cmd/prospector/
```

Expected: binary compiles cleanly.

- [ ] **Step 3: Smoke test MCP server**

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | ./prospector mcp
```

Expected: JSON response listing all 10 tools: `list_cvs`, `read_cv`, `write_cv`, `list_cover_letters`, `read_cover_letter`, `write_cover_letter`, `list_themes`, `export_pdf`, `get_experience_stats`, `create_application`.

- [ ] **Step 4: Commit**

```bash
git add .
git commit -m "chore: verified full build and MCP tool listing"
```

---

## What's next

- **Plan 2:** Dashboard + PDF export — `prospector serve`, HTMX web UI, all pages, theme preview, status tracking, `prospector serve -d`, `prospector stop`
- **Plan 3:** Skills + plugin packaging — `/customize-cv`, `/write-cover-letter`, `/apply` slash command templates
