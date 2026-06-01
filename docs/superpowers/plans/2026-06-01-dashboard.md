# Prospector Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Prospector web dashboard — Go HTTP server with HTMX+Tailwind UI covering applications tracking, CV library, cover letter library, experience stats, themes, and PDF export — plus `prospector serve`, `prospector serve -d`, and `prospector stop` CLI commands.

**Architecture:** `internal/dashboard` package serves all pages using `html/template` with templates embedded in the binary via `embed.FS`. Tailwind CSS (with typography plugin) and HTMX 1.9 are loaded from CDN — no build step. The package is layered on top of the existing `db`, `fs`, `pdf`, `config`, and `parser` packages. `prospector serve -d` re-execs the binary without the flag and detaches from the terminal; PID is written to `~/.config/prospector/prospector.pid`. `prospector stop` sends SIGTERM to that PID.

**Tech Stack:** `html/template` + `embed.FS` (stdlib), `github.com/yuin/goldmark` (markdown→HTML for preview and export), Tailwind CSS CDN with typography plugin, HTMX 1.9 CDN

---

## File Map

```
internal/dashboard/
  server.go                    ← Server struct, NewServer, Serve, ServeHTTP, render/renderPartial helpers
  markdown.go                  ← RenderMarkdown(md string) template.HTML  (goldmark)
  handlers_apps.go             ← GET /  (list), GET /apps/{id} (detail), POST /apps/{id}/status
  handlers_cv.go               ← GET /cv, GET /cv/{name}, POST /export (scoped to cv/)
  handlers_cl.go               ← GET /cl, GET /cl/{name}, POST /export (scoped to cover-letters/)
  handlers_stats.go            ← GET /stats, POST /stats/sync
  handlers_themes.go           ← GET /themes, GET /themes/{name}/preview, POST /themes/upload
  dashboard_test.go            ← httptest-based route tests for all handlers
  templates/
    layout.html                ← nav sidebar + {{block "content" .}}
    apps_list.html             ← {{define "content"}}: applications table
    app_detail.html            ← {{define "content"}}: detail view + HTMX status form
    status_history.html        ← {{define "status_history"}}: HTMX partial for status list
    cv_list.html               ← {{define "content"}}: CV file list
    cv_view.html               ← {{define "content"}}: rendered CV + theme picker + export
    cl_list.html               ← {{define "content"}}: cover letter file list
    cl_view.html               ← {{define "content"}}: rendered CL + theme picker + export
    stats.html                 ← {{define "content"}}: stats table + sync button
    stats_content.html         ← {{define "stats_content"}}: HTMX partial for stats rows
    themes.html                ← {{define "content"}}: theme list + preview + upload form

internal/db/
  applications.go              ← add ApplicationWithStatus, GetApplicationByID, ListApplicationsWithStatus

cmd/prospector/
  cmd_serve.go                 ← newServeCmd() (foreground + -d flag) + newStopCmd()
  main.go                      ← add newServeCmd(), newStopCmd() to AddCommand
```

---

## Deviations from spec

- **Preview in same page, not modal:** CV and cover letter view pages render the markdown preview inline with theme CSS injected via a `<style>` tag rather than in a separate preview frame. Simpler and visually identical.
- **Sync runs in-process:** `POST /stats/sync` runs the sync logic directly in the handler (same logic as the `sync` CLI command) rather than shelling out to `prospector sync`. Avoids process management complexity.

---

## Task 1: DB additions for dashboard

**Files:**
- Modify: `internal/db/applications.go`
- Modify: `internal/db/db_test.go`

- [ ] **Step 1: Write failing tests** (add to end of `internal/db/db_test.go`)

```go
func TestListApplicationsWithStatus(t *testing.T) {
	store := openTestStore(t)

	app := db.Application{
		Company: "Stripe", Role: "Engineer",
		AppliedDate: "2025-01-01", FolderPath: "stripe/engineer/2025-01-01",
	}
	id, err := store.UpsertApplication(app)
	require.NoError(t, err)

	require.NoError(t, store.AddStatusHistory(id, "applied", ""))
	require.NoError(t, store.AddStatusHistory(id, "interview", "phone screen"))

	apps, err := store.ListApplicationsWithStatus()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, "interview", apps[0].CurrentStatus)
	assert.Equal(t, "Stripe", apps[0].Company)
}

func TestGetApplicationByID(t *testing.T) {
	store := openTestStore(t)

	app := db.Application{
		Company: "Acme", Role: "Dev",
		AppliedDate: "2025-06-01", FolderPath: "acme/dev/2025-06-01",
	}
	id, err := store.UpsertApplication(app)
	require.NoError(t, err)

	got, err := store.GetApplicationByID(id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, "Acme", got.Company)
}

func TestGetApplicationByIDNotFound(t *testing.T) {
	store := openTestStore(t)
	_, err := store.GetApplicationByID(9999)
	assert.Error(t, err)
}
```

Note: `openTestStore` is already defined earlier in `db_test.go`. Check what it looks like if unsure:
```bash
grep -n "openTestStore" internal/db/db_test.go
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/db/... -run "TestListApplicationsWithStatus|TestGetApplicationByID"
```

Expected: compile error — types/methods don't exist yet.

- [ ] **Step 3: Add types and methods to `internal/db/applications.go`**

Append to the end of `internal/db/applications.go`:

```go
// ApplicationWithStatus is an Application plus its most recent status.
type ApplicationWithStatus struct {
	Application
	CurrentStatus string
}

// ListApplicationsWithStatus returns all applications with their latest status.
func (s *Store) ListApplicationsWithStatus() ([]ApplicationWithStatus, error) {
	rows, err := s.db.Query(`
		SELECT a.id, a.company, a.role, a.applied_date, a.base_cv_used, a.notes, a.folder_path,
		       COALESCE(
		         (SELECT status FROM application_status_history
		          WHERE application_id = a.id
		          ORDER BY changed_at DESC LIMIT 1),
		         'unknown'
		       ) AS current_status
		FROM applications a
		ORDER BY a.applied_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []ApplicationWithStatus
	for rows.Next() {
		var a ApplicationWithStatus
		if err := rows.Scan(
			&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath,
			&a.CurrentStatus,
		); err != nil {
			return nil, err
		}
		apps = append(apps, a)
	}
	return apps, rows.Err()
}

// GetApplicationByID returns a single application by its primary key.
func (s *Store) GetApplicationByID(id int64) (Application, error) {
	var a Application
	err := s.db.QueryRow(
		`SELECT id, company, role, applied_date, base_cv_used, notes, folder_path FROM applications WHERE id = ?`,
		id,
	).Scan(&a.ID, &a.Company, &a.Role, &a.AppliedDate, &a.BaseCVUsed, &a.Notes, &a.FolderPath)
	return a, err
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/db/... -v -run "TestListApplicationsWithStatus|TestGetApplicationByID"
```

Expected: 3 tests pass.

- [ ] **Step 5: Run full db test suite to confirm no regressions**

```bash
go test ./internal/db/... -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/db/applications.go internal/db/db_test.go
git commit -m "feat: add GetApplicationByID and ListApplicationsWithStatus for dashboard"
```

---

## Task 2: Dashboard package foundation + markdown renderer

**Files:**
- Create: `internal/dashboard/server.go`
- Create: `internal/dashboard/markdown.go`
- Create: `internal/dashboard/templates/layout.html`
- Create: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Add goldmark dependency**

```bash
go get github.com/yuin/goldmark@latest
go mod tidy
```

Expected: `go.mod` updated with `github.com/yuin/goldmark`.

- [ ] **Step 2: Write failing test** (`internal/dashboard/dashboard_test.go`)

```go
// internal/dashboard/dashboard_test.go
package dashboard_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/dashboard"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *dashboard.Server {
	t.Helper()
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	// create required dirs
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0755)
	}
	cfg := config.Config{GotenbergURL: "http://localhost:3000", DashboardPort: 8080}
	return dashboard.NewServer(root, store, cfg)
}

func TestRootReturns200(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
```

- [ ] **Step 3: Run to confirm it fails**

```bash
go test ./internal/dashboard/... 2>&1 | head -5
```

Expected: compile error — package doesn't exist.

- [ ] **Step 4: Create `internal/dashboard/markdown.go`**

```go
// internal/dashboard/markdown.go
package dashboard

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

// RenderMarkdown converts markdown to safe HTML for embedding in templates.
func RenderMarkdown(md string) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(md), &buf); err != nil {
		return template.HTML("<pre>" + template.HTMLEscapeString(md) + "</pre>")
	}
	return template.HTML(buf.String())
}
```

- [ ] **Step 5: Create `internal/dashboard/templates/layout.html`**

```html
{{define "layout"}}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Prospector</title>
  <script src="https://cdn.tailwindcss.com?plugins=typography"></script>
  <script src="https://unpkg.com/htmx.org@1.9.12"></script>
</head>
<body class="bg-gray-50 text-gray-900">
  <div class="flex min-h-screen">
    <nav class="w-52 bg-gray-900 text-white flex flex-col gap-1 p-4 fixed h-full shrink-0">
      <div class="text-lg font-bold mb-5 px-2">Prospector</div>
      <a href="/"       class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Applications</a>
      <a href="/cv"     class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">CV Library</a>
      <a href="/cl"     class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Cover Letters</a>
      <a href="/stats"  class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Experience</a>
      <a href="/themes" class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Themes</a>
    </nav>
    <main class="ml-52 flex-1 p-8 min-w-0">
      {{block "content" .}}{{end}}
    </main>
  </div>
</body>
</html>
{{end}}
```

- [ ] **Step 6: Create `internal/dashboard/server.go`**

```go
// internal/dashboard/server.go
package dashboard

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
)

//go:embed templates
var templateFS embed.FS

// Server holds all dependencies and the HTTP mux.
type Server struct {
	root  string
	store *db.Store
	cfg   config.Config
	mux   *http.ServeMux
}

// NewServer creates a Server and registers all routes.
func NewServer(root string, store *db.Store, cfg config.Config) *Server {
	s := &Server{root: root, store: store, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Serve starts the HTTP server on addr (e.g. ":8080").
func (s *Server) Serve(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

// ServeHTTP implements http.Handler so the server can be used with httptest.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// routes registers all URL patterns.
func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.handleAppsList)
	s.mux.HandleFunc("GET /apps/{id}", s.handleAppDetail)
	s.mux.HandleFunc("POST /apps/{id}/status", s.handleAppStatusUpdate)

	s.mux.HandleFunc("GET /cv", s.handleCVList)
	s.mux.HandleFunc("GET /cv/{name}", s.handleCVView)

	s.mux.HandleFunc("GET /cl", s.handleCLList)
	s.mux.HandleFunc("GET /cl/{name}", s.handleCLView)

	s.mux.HandleFunc("POST /export", s.handleExport)

	s.mux.HandleFunc("GET /stats", s.handleStats)
	s.mux.HandleFunc("POST /stats/sync", s.handleStatsSync)

	s.mux.HandleFunc("GET /themes", s.handleThemesList)
	s.mux.HandleFunc("GET /themes/{name}/preview", s.handleThemePreview)
	s.mux.HandleFunc("POST /themes/upload", s.handleThemeUpload)
}

// render parses layout.html + the named page template and executes "layout".
func (s *Server) render(w http.ResponseWriter, page string, data any) {
	t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/"+page)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
	}
}

// renderPartial parses a single partial template and executes it by name.
// The partial file must contain exactly one {{define "name"}} block.
func (s *Server) renderPartial(w http.ResponseWriter, partial, name string, data any) {
	t, err := template.ParseFS(templateFS, "templates/"+partial)
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("renderPartial %s: %v", partial, err)
	}
}

// parseID extracts an integer path parameter by name from the URL pattern.
// Uses Go 1.22 PathValue.
func parseID(r *http.Request, param string) (int64, bool) {
	s := r.PathValue(param)
	if s == "" {
		return 0, false
	}
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err == nil
}

// statusBadgeClass returns a Tailwind class string for a given status value.
func statusBadgeClass(status string) string {
	switch strings.ToLower(status) {
	case "offer":
		return "bg-green-100 text-green-800"
	case "interview", "assessment":
		return "bg-yellow-100 text-yellow-800"
	case "rejected", "withdrawn", "ghosted":
		return "bg-red-100 text-red-800"
	default:
		return "bg-blue-100 text-blue-800"
	}
}
```

Note: `parseID` uses `fmt.Sscanf` — add `"fmt"` to the imports.

Full import block for server.go:

```go
import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
)
```

- [ ] **Step 7: Add stub handlers** so the package compiles (real implementations come in Tasks 3–7)

Create `internal/dashboard/handlers_apps.go`:

```go
// internal/dashboard/handlers_apps.go
package dashboard

import "net/http"

func (s *Server) handleAppsList(w http.ResponseWriter, r *http.Request) {
	s.render(w, "apps_list.html", nil)
}

func (s *Server) handleAppDetail(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

func (s *Server) handleAppStatusUpdate(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
```

Create `internal/dashboard/handlers_cv.go`:

```go
// internal/dashboard/handlers_cv.go
package dashboard

import "net/http"

func (s *Server) handleCVList(w http.ResponseWriter, r *http.Request)  { http.NotFound(w, r) }
func (s *Server) handleCVView(w http.ResponseWriter, r *http.Request)   { http.NotFound(w, r) }
```

Create `internal/dashboard/handlers_cl.go`:

```go
// internal/dashboard/handlers_cl.go
package dashboard

import "net/http"

func (s *Server) handleCLList(w http.ResponseWriter, r *http.Request)  { http.NotFound(w, r) }
func (s *Server) handleCLView(w http.ResponseWriter, r *http.Request)   { http.NotFound(w, r) }
```

Create `internal/dashboard/handlers_stats.go`:

```go
// internal/dashboard/handlers_stats.go
package dashboard

import "net/http"

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request)     { http.NotFound(w, r) }
func (s *Server) handleStatsSync(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) }
```

Create `internal/dashboard/handlers_themes.go`:

```go
// internal/dashboard/handlers_themes.go
package dashboard

import "net/http"

func (s *Server) handleThemesList(w http.ResponseWriter, r *http.Request)         { http.NotFound(w, r) }
func (s *Server) handleThemePreview(w http.ResponseWriter, r *http.Request)       { http.NotFound(w, r) }
func (s *Server) handleThemeUpload(w http.ResponseWriter, r *http.Request)        { http.NotFound(w, r) }
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request)             { http.NotFound(w, r) }
```

Create minimal `internal/dashboard/templates/apps_list.html`:

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-6">Applications</h1>
<p class="text-gray-500">No applications yet.</p>
{{end}}
```

- [ ] **Step 8: Run test**

```bash
go test ./internal/dashboard/... -v -run TestRootReturns200
```

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/dashboard/ go.mod go.sum
git commit -m "feat: dashboard package foundation with server, markdown renderer, and layout"
```

---

## Task 3: Applications handlers

**Files:**
- Modify: `internal/dashboard/handlers_apps.go`
- Create: `internal/dashboard/templates/apps_list.html`
- Create: `internal/dashboard/templates/app_detail.html`
- Create: `internal/dashboard/templates/status_history.html`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests** (add to `dashboard_test.go`)

```go
func TestAppsListRoute(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Applications")
}

func TestAppDetailRoute(t *testing.T) {
	srv := newTestServer(t)
	// seed an application
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Stripe", Role: "Engineer",
		AppliedDate: "2025-01-01", FolderPath: "stripe/engineer/2025-01-01",
	})
	require.NoError(t, err)
	store.AddStatusHistory(id, "applied", "")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/apps/%d", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Stripe")
}

func TestAppStatusUpdate(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, _ := store.UpsertApplication(db.Application{
		Company: "X", Role: "Y", AppliedDate: "2025-01-01", FolderPath: "x/y/2025-01-01",
	})
	store.AddStatusHistory(id, "applied", "")

	form := strings.NewReader("status=interview&notes=phone+screen")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/apps/%d/status", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "interview")
}
```

Add these imports to the test file import block:
```go
"fmt"
"strings"
"github.com/oxGrad/spicebag/internal/db"
```

Also add a `Store()` accessor to `Server` in `server.go`:
```go
// Store returns the underlying DB store (used in tests).
func (s *Server) Store() *db.Store { return s.store }
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/dashboard/... -run "TestAppsListRoute|TestAppDetailRoute|TestAppStatusUpdate" 2>&1 | head -10
```

Expected: compilation error (Store method missing) or tests fail (404s from stubs).

- [ ] **Step 3: Create `templates/status_history.html`**

```html
{{define "status_history"}}
<div id="status-history" class="space-y-2">
  {{range .}}
  <div class="flex items-center gap-3 text-sm">
    <span class="text-gray-400 w-36 shrink-0">{{.ChangedAt.Format "2006-01-02 15:04"}}</span>
    <span class="px-2 py-0.5 rounded text-xs font-semibold bg-blue-100 text-blue-800">{{.Status}}</span>
    {{if .Notes}}<span class="text-gray-600">{{.Notes}}</span>{{end}}
  </div>
  {{else}}
  <p class="text-sm text-gray-400">No status history.</p>
  {{end}}
</div>
{{end}}
```

- [ ] **Step 4: Create `templates/apps_list.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-6">Applications</h1>
<div class="bg-white rounded-lg shadow overflow-hidden">
  <table class="w-full text-sm">
    <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
      <tr>
        <th class="text-left px-4 py-3">Company</th>
        <th class="text-left px-4 py-3">Role</th>
        <th class="text-left px-4 py-3">Applied</th>
        <th class="text-left px-4 py-3">Status</th>
      </tr>
    </thead>
    <tbody class="divide-y divide-gray-100">
      {{range .Apps}}
      <tr class="hover:bg-gray-50 cursor-pointer" onclick="location.href='/apps/{{.ID}}'">
        <td class="px-4 py-3 font-medium">{{.Company}}</td>
        <td class="px-4 py-3 text-gray-600">{{.Role}}</td>
        <td class="px-4 py-3 text-gray-500">{{.AppliedDate}}</td>
        <td class="px-4 py-3">
          <span class="px-2 py-0.5 rounded text-xs font-semibold {{.BadgeClass}}">{{.CurrentStatus}}</span>
        </td>
      </tr>
      {{else}}
      <tr><td colspan="4" class="px-4 py-8 text-center text-gray-400">No applications yet. Use <code>/apply</code> in Claude Code to create one.</td></tr>
      {{end}}
    </tbody>
  </table>
</div>
{{end}}
```

- [ ] **Step 5: Create `templates/app_detail.html`**

```html
{{define "content"}}
<div class="mb-4">
  <a href="/" class="text-blue-600 hover:underline text-sm">← Applications</a>
</div>
<div class="mb-6">
  <h1 class="text-2xl font-bold">{{.App.Company}}</h1>
  <p class="text-gray-500">{{.App.Role}} · Applied {{.App.AppliedDate}}</p>
</div>

<div class="grid grid-cols-2 gap-6">
  <div class="bg-white rounded-lg shadow p-5">
    <h2 class="font-semibold mb-3">Status History</h2>
    {{template "status_history" .History}}
    <form class="mt-4 flex gap-2 items-end"
          hx-post="/apps/{{.App.ID}}/status"
          hx-target="#status-history"
          hx-swap="outerHTML">
      <div class="flex flex-col gap-1">
        <label class="text-xs text-gray-500">New Status</label>
        <select name="status" class="border rounded px-2 py-1.5 text-sm">
          {{range .ValidStatuses}}
          <option value="{{.}}">{{.}}</option>
          {{end}}
        </select>
      </div>
      <div class="flex flex-col gap-1">
        <label class="text-xs text-gray-500">Notes (optional)</label>
        <input type="text" name="notes" class="border rounded px-2 py-1.5 text-sm w-40">
      </div>
      <button type="submit" class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Update</button>
    </form>
  </div>

  <div class="bg-white rounded-lg shadow p-5">
    <h2 class="font-semibold mb-3">Files</h2>
    {{if .App.FolderPath}}
    <div class="space-y-2 text-sm">
      <a href="/cv-file/{{.App.FolderPath}}/cv.md" class="block text-blue-600 hover:underline">View CV →</a>
      <a href="/cv-file/{{.App.FolderPath}}/cover-letter.md" class="block text-blue-600 hover:underline">View Cover Letter →</a>
    </div>
    {{else}}
    <p class="text-gray-400 text-sm">No files linked.</p>
    {{end}}
    {{if .App.Notes}}
    <div class="mt-4">
      <h3 class="text-xs text-gray-500 mb-1">Notes</h3>
      <p class="text-sm text-gray-700">{{.App.Notes}}</p>
    </div>
    {{end}}
  </div>
</div>
{{end}}
```

Note: `app_detail.html` uses `{{template "status_history" .History}}` — both templates must be parsed together in the detail handler.

- [ ] **Step 6: Implement `handlers_apps.go`**

```go
// internal/dashboard/handlers_apps.go
package dashboard

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
)

var validStatuses = []string{"applied", "assessment", "interview", "offer", "rejected", "withdrawn", "ghosted"}

type appsListData struct {
	Apps []appsListRow
}

type appsListRow struct {
	db.ApplicationWithStatus
	BadgeClass string
}

func (s *Server) handleAppsList(w http.ResponseWriter, r *http.Request) {
	apps, err := s.store.ListApplicationsWithStatus()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows := make([]appsListRow, len(apps))
	for i, a := range apps {
		rows[i] = appsListRow{a, statusBadgeClass(a.CurrentStatus)}
	}
	s.render(w, "apps_list.html", appsListData{Apps: rows})
}

type appDetailData struct {
	App          db.Application
	History      []db.StatusHistoryEntry
	ValidStatuses []string
}

func (s *Server) handleAppDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	app, err := s.store.GetApplicationByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	history, err := s.store.GetStatusHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// app_detail.html includes {{template "status_history" .History}}, so parse both files.
	t, err := template.ParseFS(templateFS,
		"templates/layout.html",
		"templates/app_detail.html",
		"templates/status_history.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(w, "layout", appDetailData{App: app, History: history, ValidStatuses: validStatuses})
}

func (s *Server) handleAppStatusUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(r, "id")
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	status := r.FormValue("status")
	notes := r.FormValue("notes")

	if err := s.store.AddStatusHistory(id, status, notes); err != nil {
		http.Error(w, fmt.Sprintf("update status: %v", err), http.StatusInternalServerError)
		return
	}

	history, err := s.store.GetStatusHistory(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPartial(w, "status_history.html", "status_history", history)
}
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/dashboard/... -v -run "TestAppsListRoute|TestAppDetailRoute|TestAppStatusUpdate"
```

Expected: all 3 pass.

- [ ] **Step 8: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: applications list, detail, and status update handlers"
```

---

## Task 4: CV library handlers

**Files:**
- Modify: `internal/dashboard/handlers_cv.go`
- Create: `internal/dashboard/templates/cv_list.html`
- Create: `internal/dashboard/templates/cv_view.html`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests** (add to `dashboard_test.go`)

```go
func TestCVListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV"))

	req := httptest.NewRequest(http.MethodGet, "/cv", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cv-backend-2025-01-01.md")
}

func TestCVViewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV\n\nContent here."))

	req := httptest.NewRequest(http.MethodGet, "/cv/cv-backend-2025-01-01.md", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Backend CV")
}
```

Add to server.go: `func (s *Server) Root() string { return s.root }`

Add to test imports: `"github.com/oxGrad/spicebag/internal/fs"`

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/dashboard/... -run "TestCVListRoute|TestCVViewRoute" 2>&1 | head -5
```

Expected: fail (404 from stubs or missing Root() method).

- [ ] **Step 3: Create `templates/cv_list.html`**

```html
{{define "content"}}
<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold">CV Library</h1>
</div>
<div class="bg-white rounded-lg shadow divide-y">
  {{range .Files}}
  <div class="flex items-center justify-between px-4 py-3 hover:bg-gray-50">
    <div>
      <a href="/cv/{{.Name}}" class="font-medium text-blue-600 hover:underline">{{.Name}}</a>
      <span class="text-xs text-gray-400 ml-3">{{.ModifiedAt.Format "2006-01-02 15:04"}}</span>
    </div>
    <a href="/cv/{{.Name}}" class="text-sm text-gray-500 hover:text-gray-700">View →</a>
  </div>
  {{else}}
  <div class="px-4 py-8 text-center text-gray-400">
    No CVs yet. Use <code>/customize-cv</code> in Claude Code to create one.
  </div>
  {{end}}
</div>
{{end}}
```

- [ ] **Step 4: Create `templates/cv_view.html`**

```html
{{define "content"}}
<div class="mb-4">
  <a href="/cv" class="text-blue-600 hover:underline text-sm">← CV Library</a>
</div>
<div class="flex items-start justify-between mb-4 gap-4">
  <h1 class="text-xl font-bold">{{.Name}}</h1>
  <div class="flex gap-2 shrink-0">
    <form method="get" class="flex gap-2 items-center">
      <select name="theme" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        {{range .Themes}}
        <option value="{{.}}" {{if eq . $.SelectedTheme}}selected{{end}}>{{.}}</option>
        {{end}}
      </select>
      <button type="submit" class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50">Preview</button>
    </form>
    <form method="post" action="/export">
      <input type="hidden" name="file_path" value="cv/{{.Name}}">
      <input type="hidden" name="theme" value="{{.SelectedTheme}}">
      <button type="submit" class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">Export PDF</button>
    </form>
  </div>
</div>
{{.ThemeStyle}}
<article class="bg-white rounded-lg shadow p-8 prose prose-sm max-w-none" id="preview">
  {{.Content}}
</article>
{{end}}
```

- [ ] **Step 5: Implement `handlers_cv.go`**

```go
// internal/dashboard/handlers_cv.go
package dashboard

import (
	"html/template"
	"net/http"

	"github.com/oxGrad/spicebag/internal/fs"
)

type cvListData struct {
	Files []fs.FileInfo
}

type cvViewData struct {
	Name          string
	Content       template.HTML
	Themes        []string
	SelectedTheme string
	ThemeStyle    template.HTML
}

func (s *Server) handleCVList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCVs(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "cv_list.html", cvListData{Files: files})
}

func (s *Server) handleCVView(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	content, err := fs.ReadCV(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	themes, _ := fs.ListThemes(s.root)
	selectedTheme := r.URL.Query().Get("theme")

	var themeStyle template.HTML
	if selectedTheme != "" {
		if css, err := fs.ReadTheme(s.root, selectedTheme); err == nil {
			themeStyle = template.HTML("<style>" + css + "</style>")
		}
	}

	s.render(w, "cv_view.html", cvViewData{
		Name:          name,
		Content:       RenderMarkdown(content),
		Themes:        themes,
		SelectedTheme: selectedTheme,
		ThemeStyle:    themeStyle,
	})
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/dashboard/... -v -run "TestCVListRoute|TestCVViewRoute"
```

Expected: both pass.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: CV library list and view handlers with theme preview"
```

---

## Task 5: Cover letter library handlers

**Files:**
- Modify: `internal/dashboard/handlers_cl.go`
- Create: `internal/dashboard/templates/cl_list.html`
- Create: `internal/dashboard/templates/cl_view.html`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests** (add to `dashboard_test.go`)

```go
func TestCLListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.md", "Dear Hiring Manager"))

	req := httptest.NewRequest(http.MethodGet, "/cl", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cl-general-2025-01-01.md")
}

func TestCLViewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.md", "Dear Hiring Manager\n\nI am excited..."))

	req := httptest.NewRequest(http.MethodGet, "/cl/cl-general-2025-01-01.md", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Hiring Manager")
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/dashboard/... -run "TestCLListRoute|TestCLViewRoute" 2>&1 | head -5
```

Expected: fail (404 from stubs).

- [ ] **Step 3: Create `templates/cl_list.html`**

```html
{{define "content"}}
<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold">Cover Letters</h1>
</div>
<div class="bg-white rounded-lg shadow divide-y">
  {{range .Files}}
  <div class="flex items-center justify-between px-4 py-3 hover:bg-gray-50">
    <div>
      <a href="/cl/{{.Name}}" class="font-medium text-blue-600 hover:underline">{{.Name}}</a>
      <span class="text-xs text-gray-400 ml-3">{{.ModifiedAt.Format "2006-01-02 15:04"}}</span>
    </div>
    <a href="/cl/{{.Name}}" class="text-sm text-gray-500 hover:text-gray-700">View →</a>
  </div>
  {{else}}
  <div class="px-4 py-8 text-center text-gray-400">
    No cover letters yet. Use <code>/write-cover-letter</code> in Claude Code to create one.
  </div>
  {{end}}
</div>
{{end}}
```

- [ ] **Step 4: Create `templates/cl_view.html`**

```html
{{define "content"}}
<div class="mb-4">
  <a href="/cl" class="text-blue-600 hover:underline text-sm">← Cover Letters</a>
</div>
<div class="flex items-start justify-between mb-4 gap-4">
  <h1 class="text-xl font-bold">{{.Name}}</h1>
  <div class="flex gap-2 shrink-0">
    <form method="get" class="flex gap-2 items-center">
      <select name="theme" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        {{range .Themes}}
        <option value="{{.}}" {{if eq . $.SelectedTheme}}selected{{end}}>{{.}}</option>
        {{end}}
      </select>
      <button type="submit" class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50">Preview</button>
    </form>
    <form method="post" action="/export">
      <input type="hidden" name="file_path" value="cover-letters/{{.Name}}">
      <input type="hidden" name="theme" value="{{.SelectedTheme}}">
      <button type="submit" class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">Export PDF</button>
    </form>
  </div>
</div>
{{.ThemeStyle}}
<article class="bg-white rounded-lg shadow p-8 prose prose-sm max-w-none" id="preview">
  {{.Content}}
</article>
{{end}}
```

- [ ] **Step 5: Implement `handlers_cl.go`**

```go
// internal/dashboard/handlers_cl.go
package dashboard

import (
	"html/template"
	"net/http"

	"github.com/oxGrad/spicebag/internal/fs"
)

type clListData struct {
	Files []fs.FileInfo
}

type clViewData struct {
	Name          string
	Content       template.HTML
	Themes        []string
	SelectedTheme string
	ThemeStyle    template.HTML
}

func (s *Server) handleCLList(w http.ResponseWriter, r *http.Request) {
	files, err := fs.ListCoverLetters(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.render(w, "cl_list.html", clListData{Files: files})
}

func (s *Server) handleCLView(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	content, err := fs.ReadCoverLetter(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	themes, _ := fs.ListThemes(s.root)
	selectedTheme := r.URL.Query().Get("theme")

	var themeStyle template.HTML
	if selectedTheme != "" {
		if css, err := fs.ReadTheme(s.root, selectedTheme); err == nil {
			themeStyle = template.HTML("<style>" + css + "</style>")
		}
	}

	s.render(w, "cl_view.html", clViewData{
		Name:          name,
		Content:       RenderMarkdown(content),
		Themes:        themes,
		SelectedTheme: selectedTheme,
		ThemeStyle:    themeStyle,
	})
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/dashboard/... -v -run "TestCLListRoute|TestCLViewRoute"
```

Expected: both pass.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: cover letter library list and view handlers with theme preview"
```

---

## Task 6: Experience stats + sync handler

**Files:**
- Modify: `internal/dashboard/handlers_stats.go`
- Create: `internal/dashboard/templates/stats.html`
- Create: `internal/dashboard/templates/stats_content.html`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests** (add to `dashboard_test.go`)

```go
func TestStatsRoute(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	require.NoError(t, store.UpsertExperience([]db.ExperienceEntry{
		{RoleType: "backend", Company: "Acme", StartDate: "2020-01-01", EndDate: "2022-01-01", SyncedFrom: "cv.md"},
	}))

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "backend")
}

func TestStatsSyncRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	// write a CV with frontmatter so sync has something to parse
	cvContent := "---\nexperience:\n  - role_type: devops\n    company: FooCo\n    start: \"2021-01-01\"\n    end: \"2023-01-01\"\n---\n# CV\n"
	require.NoError(t, fs.WriteCV(root, "cv-devops-2025-01-01.md", cvContent))

	req := httptest.NewRequest(http.MethodPost, "/stats/sync", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "devops")
}
```

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/dashboard/... -run "TestStatsRoute|TestStatsSyncRoute" 2>&1 | head -5
```

Expected: fail (404 from stubs).

- [ ] **Step 3: Create `templates/stats_content.html`** (HTMX partial)

```html
{{define "stats_content"}}
<div id="stats-content">
  <div class="bg-white rounded-lg shadow overflow-hidden mb-6">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
        <tr>
          <th class="text-left px-4 py-3">Role Type</th>
          <th class="text-right px-4 py-3">Years</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        {{range $role, $years := .ByRole}}
        <tr>
          <td class="px-4 py-3 font-medium">{{$role}}</td>
          <td class="px-4 py-3 text-right text-gray-600">{{printf "%.1f" $years}}</td>
        </tr>
        {{else}}
        <tr><td colspan="2" class="px-4 py-8 text-center text-gray-400">No experience data. Run sync after adding CVs with frontmatter.</td></tr>
        {{end}}
      </tbody>
      {{if .ByRole}}
      <tfoot class="bg-gray-50 font-semibold">
        <tr>
          <td class="px-4 py-3">Total</td>
          <td class="px-4 py-3 text-right">{{printf "%.1f" .Total}}</td>
        </tr>
      </tfoot>
      {{end}}
    </table>
  </div>
</div>
{{end}}
```

- [ ] **Step 4: Create `templates/stats.html`**

```html
{{define "content"}}
<div class="flex items-center justify-between mb-6">
  <h1 class="text-2xl font-bold">Experience Stats</h1>
  <button hx-post="/stats/sync"
          hx-target="#stats-content"
          hx-swap="outerHTML"
          class="bg-blue-600 text-white rounded px-4 py-2 text-sm hover:bg-blue-700 flex items-center gap-2">
    <span class="htmx-indicator" id="sync-spinner">⟳</span>
    Sync from CVs
  </button>
</div>
{{template "stats_content" .Stats}}
<p class="text-xs text-gray-400 mt-2">Years calculated from CV frontmatter experience entries. Run Sync after editing CVs.</p>
{{end}}
```

- [ ] **Step 5: Implement `handlers_stats.go`**

```go
// internal/dashboard/handlers_stats.go
package dashboard

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/oxGrad/spicebag/internal/parser"
)

type statsPageData struct {
	Stats db.ExperienceStats
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.GetExperienceStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// stats.html embeds {{template "stats_content" .Stats}} so parse both files.
	t, err := template.ParseFS(templateFS, "templates/layout.html", "templates/stats.html", "templates/stats_content.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.ExecuteTemplate(w, "layout", statsPageData{Stats: stats})
}

func (s *Server) handleStatsSync(w http.ResponseWriter, r *http.Request) {
	cvFiles, err := fs.ListCVs(s.root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, f := range cvFiles {
		content, err := fs.ReadCV(s.root, f.Name)
		if err != nil {
			continue
		}
		if err := s.store.DeleteExperienceBySyncedFrom(f.Name); err != nil {
			http.Error(w, fmt.Sprintf("delete old entries: %v", err), http.StatusInternalServerError)
			return
		}
		entries, err := parser.ParseExperience(content)
		if err != nil {
			continue
		}
		dbEntries := make([]db.ExperienceEntry, len(entries))
		for i, e := range entries {
			dbEntries[i] = db.ExperienceEntry{
				RoleType: e.RoleType, Company: e.Company,
				StartDate: e.StartDate, EndDate: e.EndDate, SyncedFrom: f.Name,
			}
		}
		if len(dbEntries) > 0 {
			s.store.UpsertExperience(dbEntries)
		}
	}

	stats, err := s.store.GetExperienceStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderPartial(w, "stats_content.html", "stats_content", stats)
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/dashboard/... -v -run "TestStatsRoute|TestStatsSyncRoute"
```

Expected: both pass.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: experience stats page with in-process sync"
```

---

## Task 7: Themes handler + PDF export handler

**Files:**
- Modify: `internal/dashboard/handlers_themes.go`
- Create: `internal/dashboard/templates/themes.html`
- Modify: `internal/dashboard/dashboard_test.go`

The `handleExport` function is also implemented here since it shares the `pdf` dependency.

- [ ] **Step 1: Write failing tests** (add to `dashboard_test.go`)

```go
func TestThemesListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	os.WriteFile(filepath.Join(root, "themes", "minimal.css"), []byte("body { font-family: serif; }"), 0644)

	req := httptest.NewRequest(http.MethodGet, "/themes", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "minimal")
}

func TestThemePreviewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	os.WriteFile(filepath.Join(root, "themes", "minimal.css"), []byte("body { color: red; }"), 0644)
	require.NoError(t, fs.WriteCV(root, "cv-test.md", "# Hello World"))

	req := httptest.NewRequest(http.MethodGet, "/themes/minimal/preview?cv=cv-test.md", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Hello World")
}

func TestThemeUploadRoute(t *testing.T) {
	srv := newTestServer(t)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("theme", "custom.css")
	fw.Write([]byte("body { background: blue; }"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/themes/upload", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusSeeOther, w.Code)
}

func TestExportRoute(t *testing.T) {
	// mock Gotenberg
	gotenberg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-fake"))
	}))
	defer gotenberg.Close()

	root := t.TempDir()
	store, _ := db.Open(filepath.Join(root, "test.db"))
	defer store.Close()
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0755)
	}
	cfg := config.Config{GotenbergURL: gotenberg.URL}
	srv := dashboard.NewServer(root, store, cfg)

	require.NoError(t, fs.WriteCV(root, "cv-test.md", "# Hello"))

	form := strings.NewReader("file_path=cv%2Fcv-test.md&theme=")
	req := httptest.NewRequest(http.MethodPost, "/export", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}
```

Add to test imports: `"bytes"`, `"mime/multipart"`.

- [ ] **Step 2: Run to confirm they fail**

```bash
go test ./internal/dashboard/... -run "TestThemesListRoute|TestThemePreviewRoute|TestThemeUploadRoute|TestExportRoute" 2>&1 | head -5
```

Expected: fail (404 from stubs).

- [ ] **Step 3: Create `templates/themes.html`**

```html
{{define "content"}}
<h1 class="text-2xl font-bold mb-6">Themes</h1>

<div class="grid grid-cols-2 gap-6">
  <div>
    <h2 class="font-semibold mb-3">Available Themes</h2>
    <div class="bg-white rounded-lg shadow divide-y mb-4">
      {{range .Themes}}
      <div class="flex items-center justify-between px-4 py-3 hover:bg-gray-50">
        <span class="font-medium">{{.}}</span>
        <a href="/themes/{{.}}/preview" class="text-sm text-blue-600 hover:underline">Preview →</a>
      </div>
      {{else}}
      <div class="px-4 py-8 text-center text-gray-400">No themes yet. Upload a CSS file below.</div>
      {{end}}
    </div>

    <h2 class="font-semibold mb-3">Upload New Theme</h2>
    <form method="post" action="/themes/upload" enctype="multipart/form-data"
          class="bg-white rounded-lg shadow p-4 flex gap-3 items-end">
      <div class="flex flex-col gap-1">
        <label class="text-xs text-gray-500">CSS File</label>
        <input type="file" name="theme" accept=".css" class="text-sm">
      </div>
      <button type="submit" class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">Upload</button>
    </form>
  </div>

  {{if .PreviewTheme}}
  <div>
    <h2 class="font-semibold mb-3">Preview: {{.PreviewTheme}}</h2>
    <div class="bg-white rounded-lg shadow p-6">
      {{.PreviewStyle}}
      <article class="prose prose-sm max-w-none">{{.PreviewContent}}</article>
    </div>
  </div>
  {{end}}
</div>
{{end}}
```

- [ ] **Step 4: Implement `handlers_themes.go`**

```go
// internal/dashboard/handlers_themes.go
package dashboard

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/oxGrad/spicebag/internal/pdf"
)

type themesPageData struct {
	Themes         []string
	PreviewTheme   string
	PreviewStyle   template.HTML
	PreviewContent template.HTML
}

func (s *Server) handleThemesList(w http.ResponseWriter, r *http.Request) {
	themes, _ := fs.ListThemes(s.root)
	s.render(w, "themes.html", themesPageData{Themes: themes})
}

func (s *Server) handleThemePreview(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	css, err := fs.ReadTheme(s.root, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// render preview with the first available CV, or sample text
	previewMD := "# Sample Heading\n\nThis is how your theme looks with **bold**, *italic*, and regular text.\n\n## Section\n\nMore content here."
	cvParam := r.URL.Query().Get("cv")
	if cvParam != "" {
		if content, err := fs.ReadCV(s.root, cvParam); err == nil {
			previewMD = content
		}
	}

	themes, _ := fs.ListThemes(s.root)
	s.render(w, "themes.html", themesPageData{
		Themes:         themes,
		PreviewTheme:   name,
		PreviewStyle:   template.HTML("<style>" + css + "</style>"),
		PreviewContent: RenderMarkdown(previewMD),
	})
}

func (s *Server) handleThemeUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(2 << 20); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("theme")
	if err != nil {
		http.Error(w, "missing file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	name := header.Filename
	if !strings.HasSuffix(name, ".css") {
		http.Error(w, "file must be a .css file", http.StatusBadRequest)
		return
	}

	themeDir := filepath.Join(s.root, "themes")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dest := filepath.Join(themeDir, name)
	f, err := os.Create(dest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if _, err := io.Copy(f, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/themes", http.StatusSeeOther)
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	filePath := r.FormValue("file_path")
	theme := r.FormValue("theme")

	if filePath == "" {
		http.Error(w, "file_path is required", http.StatusBadRequest)
		return
	}

	// read the markdown file
	mdBytes, err := os.ReadFile(filepath.Join(s.root, filePath))
	if err != nil {
		http.Error(w, fmt.Sprintf("read file: %v", err), http.StatusNotFound)
		return
	}

	// render markdown to HTML
	htmlContent := string(RenderMarkdown(string(mdBytes)))

	// read theme CSS (optional)
	var css string
	if theme != "" {
		cssBytes, err := fs.ReadTheme(s.root, theme)
		if err == nil {
			css = cssBytes
		}
	}

	client := pdf.NewClient(s.cfg.GotenbergURL)
	pdfBytes, err := client.RenderPDF(htmlContent, css)
	if err != nil {
		http.Error(w, fmt.Sprintf("render PDF: %v", err), http.StatusInternalServerError)
		return
	}

	// derive filename for download
	base := filepath.Base(filePath)
	pdfName := strings.TrimSuffix(base, filepath.Ext(base)) + ".pdf"

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, pdfName))
	w.Write(pdfBytes)
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/dashboard/... -v -run "TestThemesListRoute|TestThemePreviewRoute|TestThemeUploadRoute|TestExportRoute"
```

Expected: all 4 pass.

- [ ] **Step 6: Run all dashboard tests**

```bash
go test ./internal/dashboard/... -v
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/
git commit -m "feat: themes list/preview/upload and PDF export handler"
```

---

## Task 8: serve and stop CLI commands

**Files:**
- Create: `cmd/prospector/cmd_serve.go`
- Modify: `cmd/prospector/main.go`

No unit tests for CLI commands — verified by building and running.

- [ ] **Step 1: Create `cmd/prospector/cmd_serve.go`**

```go
// cmd/prospector/cmd_serve.go
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/dashboard"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)

func newServeCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the web dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()
			pidPath := filepath.Join(root, "prospector.pid")
			logPath := filepath.Join(root, "prospector.log")

			if daemon {
				logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return fmt.Errorf("open log: %w", err)
				}

				child := exec.Command(os.Args[0], "serve")
				child.Stdout = logFile
				child.Stderr = logFile
				child.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // detach from terminal
				if err := child.Start(); err != nil {
					return fmt.Errorf("start background server: %w", err)
				}
				logFile.Close()

				pidStr := strconv.Itoa(child.Process.Pid) + "\n"
				if err := os.WriteFile(pidPath, []byte(pidStr), 0644); err != nil {
					return fmt.Errorf("write PID: %w", err)
				}

				fmt.Printf("Dashboard started in background (PID %d)\n", child.Process.Pid)
				fmt.Printf("Log:  %s\n", logPath)
				fmt.Printf("Stop: prospector stop\n")
				return nil
			}

			// foreground mode
			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			store, err := db.Open(filepath.Join(root, "prospector.db"))
			if err != nil {
				return fmt.Errorf("open db: %w", err)
			}
			defer store.Close()

			addr := fmt.Sprintf(":%d", cfg.DashboardPort)
			fmt.Printf("Dashboard running at http://localhost%s\n", addr)
			return dashboard.NewServer(root, store, cfg).Serve(addr)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "Run in background")
	return cmd
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background dashboard server",
		RunE: func(cmd *cobra.Command, args []string) error {
			pidPath := filepath.Join(prospectorRoot(), "prospector.pid")
			data, err := os.ReadFile(pidPath)
			if os.IsNotExist(err) {
				return fmt.Errorf("prospector is not running (no PID file at %s)", pidPath)
			}
			if err != nil {
				return err
			}

			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil {
				return fmt.Errorf("invalid PID in %s", pidPath)
			}

			proc, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("process not found: %w", err)
			}
			if err := proc.Signal(syscall.SIGTERM); err != nil {
				return fmt.Errorf("stop process: %w", err)
			}

			os.Remove(pidPath)
			fmt.Printf("Stopped prospector dashboard (PID %d)\n", pid)
			return nil
		},
	}
}
```

- [ ] **Step 2: Update `cmd/prospector/main.go`** — add `newServeCmd()` and `newStopCmd()`

Read the current `main.go` first, then change the `AddCommand` call to:

```go
root.AddCommand(
    newInitCmd(),
    newMCPCmd(),
    newUpCmd(),
    newSyncCmd(),
    newServeCmd(),
    newStopCmd(),
)
```

- [ ] **Step 3: Build**

```bash
go build ./cmd/prospector/ && ./prospector --help
```

Expected: `serve` and `stop` appear in the command list.

```bash
./prospector serve --help
```

Expected: shows `-d, --daemon` flag.

- [ ] **Step 4: Run full test suite**

```bash
go test ./... -count=1
```

Expected: all tests pass across all packages.

- [ ] **Step 5: Build release binary**

```bash
go build -ldflags="-s -w" -o prospector ./cmd/prospector/
rm prospector
```

Expected: compiles cleanly, binary removed.

- [ ] **Step 6: Commit**

```bash
git add cmd/prospector/cmd_serve.go cmd/prospector/main.go
git commit -m "feat: prospector serve (foreground + -d background) and stop commands"
```

---

## What's next

- **Plan 3:** Skills + plugin packaging — `/customize-cv`, `/write-cover-letter`, `/apply` slash command templates for the Claude Code plugin marketplace.
