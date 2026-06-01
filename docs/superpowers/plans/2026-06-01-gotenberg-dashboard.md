# Gotenberg Dashboard Control Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Gotenberg start/stop/healthcheck from the `spicebag up` CLI command into the web dashboard, showing a contextual status widget with a disabled Export button when Gotenberg is not running.

**Architecture:** Three new HTTP handlers (`/gotenberg/status`, `/gotenberg/start`, `/gotenberg/stop`) live in `handlers_gotenberg.go`. A self-contained htmx partial template (`gotenberg_status.html`) renders the status dot, start/stop button, and export button atomically — the export button is disabled with a tooltip when Gotenberg is stopped. CV view and CL view load this widget via `hx-get` on page load, replacing their static export forms.

**Tech Stack:** Go `net/http`, `os/exec` (docker compose), `html/template`, htmx 1.9 (already in layout.html), Tailwind CSS (already in layout.html).

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| Delete | `cmd/spicebag/cmd_up.go` | Gone — Gotenberg managed from dashboard |
| Modify | `cmd/spicebag/main.go` | Remove `newUpCmd()` registration |
| Create | `internal/dashboard/handlers_gotenberg.go` | `gotenbergStatusData`, `checkGotenberg()`, `defaultRunCompose()`, `SetComposeRunner()`, `handleGotenbergStatus`, `handleGotenbergStart`, `handleGotenbergStop` |
| Create | `internal/dashboard/templates/gotenberg_status.html` | `gotenberg-status` htmx partial: status dot, start/stop button, export button |
| Modify | `internal/dashboard/server.go` | Add `runCompose` field to `Server`; register 3 new routes; wire default runner in `NewServer` |
| Modify | `internal/dashboard/templates/cv_view.html` | Replace static export form with htmx status widget div |
| Modify | `internal/dashboard/templates/cl_view.html` | Same as cv_view |
| Modify | `internal/dashboard/dashboard_test.go` | Add 4 Gotenberg handler tests |

---

## Task 1: Remove `spicebag up` CLI command

**Files:**
- Delete: `cmd/spicebag/cmd_up.go`
- Modify: `cmd/spicebag/main.go`

- [ ] **Step 1: Delete cmd_up.go**

```bash
rm cmd/spicebag/cmd_up.go
```

- [ ] **Step 2: Remove newUpCmd from main.go**

Open `cmd/spicebag/main.go`. Remove the `newUpCmd(),` line from the `root.AddCommand(...)` call. The result:

```go
root.AddCommand(
    newInitCmd(),
    newMCPCmd(),
    newSyncCmd(),
    newServeCmd(),
    newStopCmd(),
)
```

- [ ] **Step 3: Verify build**

```bash
go build ./cmd/spicebag/
```

Expected: no output, exit 0.

- [ ] **Step 4: Commit**

```bash
git add cmd/spicebag/cmd_up.go cmd/spicebag/main.go
git commit -m "feat: remove spicebag up — Gotenberg managed from dashboard"
```

---

## Task 2: Gotenberg status endpoint (TDD)

**Files:**
- Create: `internal/dashboard/handlers_gotenberg.go`
- Create: `internal/dashboard/templates/gotenberg_status.html`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write the two failing status tests**

Add to the bottom of `internal/dashboard/dashboard_test.go`:

```go
func TestGotenbergStatusRunning(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mock.Close()

	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	defer store.Close()
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	srv := dashboard.NewServer(root, store, config.Config{GotenbergURL: mock.URL})

	req := httptest.NewRequest(http.MethodGet, "/gotenberg/status?file_path=cv%2Ftest.md&theme=", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gotenberg running")
	assert.Contains(t, w.Body.String(), "Export PDF")
	// export button must NOT be disabled when running
	assert.NotContains(t, w.Body.String(), `disabled`)
}

func TestGotenbergStatusStopped(t *testing.T) {
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	defer store.Close()
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	// point at a port with nothing listening
	srv := dashboard.NewServer(root, store, config.Config{GotenbergURL: "http://localhost:19999"})

	req := httptest.NewRequest(http.MethodGet, "/gotenberg/status?file_path=cv%2Ftest.md&theme=", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gotenberg stopped")
	assert.Contains(t, w.Body.String(), `disabled`)
	assert.Contains(t, w.Body.String(), "Start Gotenberg to export PDF")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/dashboard/ -run "TestGotenbergStatus" -v
```

Expected: FAIL — `no pattern matching routes` or similar (route not registered yet).

- [ ] **Step 3: Add runCompose field to Server**

In `internal/dashboard/server.go`, update the `Server` struct and `NewServer`:

```go
// Server holds all dependencies and the HTTP mux.
type Server struct {
	root       string
	store      *db.Store
	cfg        config.Config
	mux        *http.ServeMux
	runCompose func(args ...string) error
}

// NewServer creates a Server and registers all routes.
func NewServer(root string, store *db.Store, cfg config.Config) *Server {
	s := &Server{root: root, store: store, cfg: cfg, mux: http.NewServeMux()}
	s.runCompose = s.defaultRunCompose
	s.routes()
	return s
}
```

- [ ] **Step 4: Register Gotenberg routes in server.go**

In the `routes()` method, add at the bottom:

```go
s.mux.HandleFunc("GET /gotenberg/status", s.handleGotenbergStatus)
s.mux.HandleFunc("POST /gotenberg/start", s.handleGotenbergStart)
s.mux.HandleFunc("POST /gotenberg/stop", s.handleGotenbergStop)
```

- [ ] **Step 5: Create handlers_gotenberg.go**

Create `internal/dashboard/handlers_gotenberg.go`:

```go
// internal/dashboard/handlers_gotenberg.go
package dashboard

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type gotenbergStatusData struct {
	Running  bool
	Err      string
	FilePath string
	Theme    string
}

// SetComposeRunner replaces the docker compose executor — used in tests.
func (s *Server) SetComposeRunner(fn func(args ...string) error) {
	s.runCompose = fn
}

func (s *Server) checkGotenberg() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(s.cfg.GotenbergURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func (s *Server) defaultRunCompose(args ...string) error {
	composePath := filepath.Join(s.root, "docker-compose.yml")
	baseArgs := append([]string{"compose", "-f", composePath}, args...)
	cmd := exec.Command("docker", baseArgs...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *Server) handleGotenbergStatus(w http.ResponseWriter, r *http.Request) {
	data := gotenbergStatusData{
		Running:  s.checkGotenberg(),
		FilePath: r.URL.Query().Get("file_path"),
		Theme:    r.URL.Query().Get("theme"),
	}
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}

func (s *Server) handleGotenbergStart(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	data := gotenbergStatusData{
		FilePath: r.FormValue("file_path"),
		Theme:    r.FormValue("theme"),
	}
	if err := s.runCompose("up", "-d"); err != nil {
		data.Err = err.Error()
	}
	data.Running = s.checkGotenberg()
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}

func (s *Server) handleGotenbergStop(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "parse form: "+err.Error(), http.StatusBadRequest)
		return
	}
	data := gotenbergStatusData{
		FilePath: r.FormValue("file_path"),
		Theme:    r.FormValue("theme"),
	}
	if err := s.runCompose("stop", "gotenberg"); err != nil {
		data.Err = err.Error()
	}
	data.Running = s.checkGotenberg()
	s.renderPartial(w, "gotenberg_status.html", "gotenberg-status", data)
}
```

- [ ] **Step 6: Create gotenberg_status.html partial**

Create `internal/dashboard/templates/gotenberg_status.html`:

```html
{{define "gotenberg-status"}}
<div id="gotenberg-status" class="flex flex-col gap-2">
  {{if .Running}}
  <div class="flex items-center gap-2 text-sm text-green-700">
    <span class="w-2 h-2 rounded-full bg-green-500 inline-block shrink-0"></span>
    Gotenberg running
    <form hx-post="/gotenberg/stop"
          hx-target="#gotenberg-status"
          hx-swap="outerHTML"
          class="inline">
      <input type="hidden" name="file_path" value="{{.FilePath}}">
      <input type="hidden" name="theme" value="{{.Theme}}">
      <button type="submit"
              class="ml-1 text-xs border border-gray-300 rounded px-2 py-0.5 bg-white hover:bg-gray-50 text-gray-700">
        Stop
      </button>
    </form>
  </div>
  <form method="post" action="/export">
    <input type="hidden" name="file_path" value="{{.FilePath}}">
    <input type="hidden" name="theme" value="{{.Theme}}">
    <button type="submit"
            class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">
      Export PDF
    </button>
  </form>
  {{else}}
  <div class="flex items-center gap-2 text-sm text-red-700">
    <span class="w-2 h-2 rounded-full bg-red-500 inline-block shrink-0"></span>
    Gotenberg stopped
    <form hx-post="/gotenberg/start"
          hx-target="#gotenberg-status"
          hx-swap="outerHTML"
          class="inline">
      <input type="hidden" name="file_path" value="{{.FilePath}}">
      <input type="hidden" name="theme" value="{{.Theme}}">
      <button type="submit"
              class="ml-1 text-xs border border-gray-300 rounded px-2 py-0.5 bg-white hover:bg-gray-50 text-gray-700">
        Start
      </button>
    </form>
  </div>
  <button type="button"
          disabled
          title="Start Gotenberg to export PDF"
          class="bg-blue-300 text-white rounded px-3 py-1.5 text-sm cursor-not-allowed w-fit">
    Export PDF
  </button>
  {{if .Err}}
  <p class="text-xs text-red-600">{{.Err}}</p>
  {{end}}
  {{end}}
</div>
{{end}}
```

- [ ] **Step 7: Run status tests**

```bash
go test ./internal/dashboard/ -run "TestGotenbergStatus" -v
```

Expected: both PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/dashboard/handlers_gotenberg.go \
        internal/dashboard/templates/gotenberg_status.html \
        internal/dashboard/server.go \
        internal/dashboard/dashboard_test.go
git commit -m "feat: Gotenberg status endpoint and partial template"
```

---

## Task 3: Gotenberg start/stop endpoints (TDD)

**Files:**
- Modify: `internal/dashboard/dashboard_test.go`
- (handlers_gotenberg.go already has the implementation from Task 2)

- [ ] **Step 1: Write the two failing start/stop tests**

Add to the bottom of `internal/dashboard/dashboard_test.go`:

```go
func TestGotenbergStart(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer mock.Close()

	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	defer store.Close()
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	srv := dashboard.NewServer(root, store, config.Config{GotenbergURL: mock.URL})
	srv.SetComposeRunner(func(args ...string) error { return nil }) // no-op: skip docker

	form := strings.NewReader("file_path=cv%2Ftest.md&theme=minimal")
	req := httptest.NewRequest(http.MethodPost, "/gotenberg/start", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gotenberg running")
	assert.NotContains(t, w.Body.String(), "disabled")
}

func TestGotenbergStop(t *testing.T) {
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	defer store.Close()
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	// nothing listening on 19999 → healthcheck returns false → "stopped"
	srv := dashboard.NewServer(root, store, config.Config{GotenbergURL: "http://localhost:19999"})
	srv.SetComposeRunner(func(args ...string) error { return nil })

	form := strings.NewReader("file_path=cover-letters%2Fcl.md&theme=")
	req := httptest.NewRequest(http.MethodPost, "/gotenberg/stop", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gotenberg stopped")
	assert.Contains(t, w.Body.String(), "disabled")
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/dashboard/ -run "TestGotenbergStart|TestGotenbergStop" -v
```

Expected: FAIL — `SetComposeRunner` undefined (method not exported yet).

Wait — `SetComposeRunner` was already written in Task 2 Step 5. The tests should now compile. If they fail for another reason, check that `SetComposeRunner` is exported correctly in `handlers_gotenberg.go`.

- [ ] **Step 3: Run the tests**

```bash
go test ./internal/dashboard/ -run "TestGotenbergStart|TestGotenbergStop" -v
```

Expected: both PASS (implementation is already in place from Task 2).

- [ ] **Step 4: Run the full test suite**

```bash
go test ./internal/dashboard/ -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/dashboard/dashboard_test.go
git commit -m "test: Gotenberg start/stop handler tests"
```

---

## Task 4: Wire status widget into CV view and CL view

**Files:**
- Modify: `internal/dashboard/templates/cv_view.html`
- Modify: `internal/dashboard/templates/cl_view.html`

- [ ] **Step 1: Update cv_view.html**

In `internal/dashboard/templates/cv_view.html`, replace the static export form:

```html
    <form method="post" action="/export">
      <input type="hidden" name="file_path" value="cv/{{.Name}}">
      <input type="hidden" name="theme" value="{{.SelectedTheme}}">
      <button type="submit" class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">Export PDF</button>
    </form>
```

with the htmx status widget loader:

```html
    <div hx-get="/gotenberg/status?file_path=cv/{{.Name}}&theme={{.SelectedTheme}}"
         hx-trigger="load"
         hx-swap="outerHTML">
      <button type="button" disabled
              class="bg-blue-300 text-white rounded px-3 py-1.5 text-sm cursor-not-allowed">
        Export PDF
      </button>
    </div>
```

- [ ] **Step 2: Update cl_view.html**

In `internal/dashboard/templates/cl_view.html`, replace the static export form:

```html
    <form method="post" action="/export">
      <input type="hidden" name="file_path" value="cover-letters/{{.Name}}">
      <input type="hidden" name="theme" value="{{.SelectedTheme}}">
      <button type="submit" class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700">Export PDF</button>
    </form>
```

with:

```html
    <div hx-get="/gotenberg/status?file_path=cover-letters/{{.Name}}&theme={{.SelectedTheme}}"
         hx-trigger="load"
         hx-swap="outerHTML">
      <button type="button" disabled
              class="bg-blue-300 text-white rounded px-3 py-1.5 text-sm cursor-not-allowed">
        Export PDF
      </button>
    </div>
```

- [ ] **Step 3: Run the full test suite**

```bash
go test ./...
```

Expected: all tests PASS. (Existing CV/CL view tests check page content but not the export widget — the widget is now loaded via a separate htmx request, so those tests are unaffected.)

- [ ] **Step 4: Build**

```bash
go build ./cmd/spicebag/
```

Expected: exit 0.

- [ ] **Step 5: Commit**

```bash
git add internal/dashboard/templates/cv_view.html \
        internal/dashboard/templates/cl_view.html
git commit -m "feat: replace static export button with Gotenberg status widget"
```
