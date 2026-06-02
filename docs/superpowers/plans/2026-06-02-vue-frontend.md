# Vue Frontend Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the Go HTML template + htmx dashboard with a Vue 3 SPA embedded in the Go binary, using HTML fragments as the document format and iframes for isolated document preview.

**Architecture:** Vue 3 + Vite lives in `frontend/`; `vite build` writes compiled assets to `internal/dashboard/ui/` which Go embeds into the binary. The Go backend exposes a JSON API at `/api/` and render endpoints at `/render/cv/{name}` and `/render/cl/{name}` that return full standalone HTML pages (fragment + theme CSS injected) for display in iframes. CVs and cover letters are stored as `.html` fragments (no `<!DOCTYPE>`, no `<head>`, just body content) instead of markdown.

**Tech Stack:** Vue 3, Vue Router 4, Vite, Tailwind CSS (PostCSS), Go `//go:embed`, existing Go domain packages (db, fs, pdf, config, mcp)

---

## File Map

### New files
- `frontend/` — Vue project root
- `frontend/index.html` — Vite entry HTML
- `frontend/vite.config.js` — Vite config (outDir → `../internal/dashboard/ui`)
- `frontend/tailwind.config.js`
- `frontend/postcss.config.js`
- `frontend/package.json`
- `frontend/src/main.js`
- `frontend/src/App.vue` — shell with sidebar nav + RouterView
- `frontend/src/router/index.js` — Vue Router config
- `frontend/src/api.js` — fetch wrapper for all `/api/` calls
- `frontend/src/views/AppsView.vue`
- `frontend/src/views/AppDetailView.vue`
- `frontend/src/views/CVListView.vue`
- `frontend/src/views/CVView.vue` — iframe preview
- `frontend/src/views/CLListView.vue`
- `frontend/src/views/CLView.vue` — iframe preview
- `frontend/src/views/ThemesView.vue`
- `frontend/src/views/StatsView.vue`
- `frontend/src/components/GotenbergWidget.vue`
- `internal/dashboard/ui/` — **generated, gitignored** (Vite output)
- `internal/dashboard/render.go` — `renderDocument(fragment, css)` helper + render endpoint handlers

### Modified files
- `internal/fs/cv.go` — list/read/write `.html` files instead of `.md`
- `internal/fs/coverletter.go` — same
- `internal/dashboard/server.go` — add `/api/` + `/render/` routes, SPA catch-all, embed `ui/`
- `internal/dashboard/handlers_apps.go` — return JSON
- `internal/dashboard/handlers_cv.go` — return JSON; remove `scopeCSS`, remove markdown rendering
- `internal/dashboard/handlers_cl.go` — return JSON
- `internal/dashboard/handlers_themes.go` — return JSON; export handler reads HTML fragment
- `internal/dashboard/handlers_stats.go` — return JSON; remove sync-from-CV (no more parser dependency)
- `internal/dashboard/handlers_gotenberg.go` — return JSON
- `internal/dashboard/dashboard_test.go` — update for JSON API
- `.goreleaser.yaml` — add `npm ci && npm run build` hook
- `.gitignore` — add `internal/dashboard/ui/`
- `justfile` — add `build-frontend` recipe
- `plugins/skills/apply.md` — generate HTML fragments
- `plugins/skills/customize-cv.md` — generate HTML fragments
- `plugins/skills/write-cover-letter.md` — generate HTML fragments

### Deleted files
- `internal/dashboard/markdown.go`
- `internal/dashboard/templates/` — all `.html` templates
- `internal/parser/parser.go` (and test) — no longer needed

---

## Task 1: Frontend toolchain — Vite + Vue 3 + Tailwind, embedded in Go

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.js`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Create: `frontend/index.html`
- Create: `frontend/src/main.js`
- Create: `frontend/src/App.vue`
- Modify: `.gitignore`
- Modify: `justfile`
- Modify: `internal/dashboard/server.go`

- [ ] **Step 1: Create `frontend/package.json`**

```json
{
  "name": "spicebag-ui",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "vue": "^3.5.0",
    "vue-router": "^4.4.0"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.0.0",
    "autoprefixer": "^10.4.0",
    "postcss": "^8.4.0",
    "tailwindcss": "^3.4.0",
    "vite": "^6.0.0"
  }
}
```

- [ ] **Step 2: Create `frontend/vite.config.js`**

```js
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  build: {
    outDir: '../internal/dashboard/ui',
    emptyOutDir: true,
  },
})
```

- [ ] **Step 3: Create `frontend/tailwind.config.js`**

```js
export default {
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: { extend: {} },
  plugins: [],
}
```

- [ ] **Step 4: Create `frontend/postcss.config.js`**

```js
export default {
  plugins: { tailwindcss: {}, autoprefixer: {} },
}
```

- [ ] **Step 5: Create `frontend/index.html`**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Spice Bag</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.js"></script>
  </body>
</html>
```

- [ ] **Step 6: Create `frontend/src/main.js`**

```js
import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/:pathMatch(.*)*', component: () => import('./views/AppsView.vue') }],
})

createApp(App).use(router).mount('#app')
```

- [ ] **Step 7: Create `frontend/src/style.css`**

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

- [ ] **Step 8: Create `frontend/src/App.vue` (minimal shell)**

```vue
<template>
  <div class="flex min-h-screen bg-gray-50 text-gray-900">
    <nav class="w-52 bg-gray-900 text-white flex flex-col gap-1 p-4 fixed h-full shrink-0">
      <div class="text-lg font-bold mb-5 px-2">🌶️ Spice Bag</div>
      <RouterLink to="/"       class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Applications</RouterLink>
      <RouterLink to="/cv"     class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">CV Library</RouterLink>
      <RouterLink to="/cl"     class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Cover Letters</RouterLink>
      <RouterLink to="/stats"  class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Experience</RouterLink>
      <RouterLink to="/themes" class="px-2 py-1.5 rounded hover:bg-gray-700 text-sm">Themes</RouterLink>
    </nav>
    <main class="ml-52 flex-1 p-8 min-w-0">
      <RouterView />
    </main>
  </div>
</template>
```

- [ ] **Step 9: Install dependencies**

```bash
cd frontend && npm install
```

- [ ] **Step 10: Run test build**

```bash
cd frontend && npm run build
```

Expected: `internal/dashboard/ui/` created with `index.html`, `assets/` directory.

- [ ] **Step 11: Add `internal/dashboard/ui/` to `.gitignore`**

Add to `.gitignore`:
```
internal/dashboard/ui/
```

- [ ] **Step 12: Update `justfile` to add `build-frontend` recipe**

```just
build-frontend:
  cd frontend && npm ci && npm run build

build: build-frontend
  go build -o spicebag ./cmd/spicebag/

run: build
  ./spicebag start

test:
  go test ./...

clean:
  rm -f ./spicebag
  rm -rf internal/dashboard/ui/
```

- [ ] **Step 13: Add Go embed + SPA handler to `internal/dashboard/server.go`**

Add to imports and struct:
```go
import "embed"

//go:embed ui
var uiFS embed.FS
```

Add `handleSPA` method (serves `ui/index.html` for all unmatched routes):
```go
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
    // Try to serve the exact file first
    path := "ui" + r.URL.Path
    f, err := uiFS.Open(path)
    if err == nil {
        f.Close()
        http.FileServer(http.FS(uiFS)).ServeHTTP(w, r)
        return
    }
    // Fall back to index.html for SPA routing
    data, _ := uiFS.ReadFile("ui/index.html")
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.Write(data)
}
```

Register as catch-all at the end of `routes()`:
```go
s.mux.HandleFunc("GET /", s.handleSPA)
```

- [ ] **Step 14: Build Go with embedded UI**

```bash
just build
```

Expected: builds without errors. `./spicebag start` serves the Vue placeholder at `http://localhost:8080`.

- [ ] **Step 15: Commit**

```bash
git add frontend/ internal/dashboard/server.go .gitignore justfile
git commit -m "feat: add Vue 3 + Vite frontend scaffold embedded in Go binary"
```

---

## Task 2: fs layer — HTML fragment support

**Files:**
- Modify: `internal/fs/cv.go`
- Modify: `internal/fs/coverletter.go`
- Modify: `internal/fs/fs_test.go`

- [ ] **Step 1: Write failing tests for HTML file listing**

In `internal/fs/fs_test.go`, add:
```go
func TestListCVsHTML(t *testing.T) {
    root := t.TempDir()
    os.MkdirAll(filepath.Join(root, "cv"), 0755)
    os.WriteFile(filepath.Join(root, "cv", "base.html"), []byte("<h1>Test</h1>"), 0644)
    os.WriteFile(filepath.Join(root, "cv", "old.md"), []byte("# old"), 0644) // should NOT appear

    files, err := fs.ListCVs(root)
    require.NoError(t, err)
    require.Len(t, files, 1)
    assert.Equal(t, "base.html", files[0].Name)
}

func TestWriteAndReadCVHTML(t *testing.T) {
    root := t.TempDir()
    content := "<h1>Gatra Raditya</h1><p>Engineer</p>"
    err := fs.WriteCV(root, "base.html", content)
    require.NoError(t, err)

    got, err := fs.ReadCV(root, "base.html")
    require.NoError(t, err)
    assert.Equal(t, content, got)
}
```

- [ ] **Step 2: Run tests to see them fail**

```bash
go test ./internal/fs/... -run TestListCVsHTML -v
```

Expected: FAIL — `ListCVs` still lists `.md` files, returns 0 results.

- [ ] **Step 3: Update `internal/fs/cv.go` — list `.html` files**

Replace `listMarkdownFiles` call and the `listMarkdownFiles` function:

```go
func ListCVs(root string) ([]FileInfo, error) {
    return listHTMLFiles(cvDir(root))
}

func listHTMLFiles(dir string) ([]FileInfo, error) {
    if err := os.MkdirAll(dir, 0755); err != nil {
        return nil, err
    }
    entries, err := os.ReadDir(dir)
    if err != nil {
        return nil, err
    }
    var files []FileInfo
    for _, e := range entries {
        if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
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

Remove `listMarkdownFiles` from `cv.go` (it is no longer used after Task 2 Step 5 below).

- [ ] **Step 4: Update `internal/fs/coverletter.go` — use `listHTMLFiles`**

```go
func ListCoverLetters(root string) ([]FileInfo, error) {
    return listHTMLFiles(coverLetterDir(root))
}
```

`listHTMLFiles` is defined in `cv.go` (same package), so no duplication.

- [ ] **Step 5: Remove `listMarkdownFiles` from `cv.go`**

Delete the `listMarkdownFiles` function entirely — it is replaced by `listHTMLFiles`.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/fs/... -v
```

Expected: all pass including new HTML tests.

- [ ] **Step 7: Commit**

```bash
git add internal/fs/
git commit -m "feat: fs layer lists and reads .html fragments instead of .md"
```

---

## Task 3: Render endpoints

These serve standalone HTML pages (`fragment + theme CSS`) for display in iframes and for Gotenberg PDF export.

**Files:**
- Create: `internal/dashboard/render.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/handlers_themes.go` (export handler reads HTML fragment)
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests for render endpoints**

In `internal/dashboard/dashboard_test.go`, add:
```go
func TestRenderCV(t *testing.T) {
    srv := newTestServer(t)
    fragment := "<h1>Gatra Raditya</h1><p>Engineer</p>"
    require.NoError(t, fs.WriteCV(srv.Root(), "base.html", fragment))

    req := httptest.NewRequest(http.MethodGet, "/render/cv/base.html", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "<h1>Gatra Raditya</h1>")
    assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

func TestRenderCVWithTheme(t *testing.T) {
    srv := newTestServer(t)
    require.NoError(t, fs.WriteCV(srv.Root(), "base.html", "<h1>Test</h1>"))
    os.WriteFile(filepath.Join(srv.Root(), "themes", "minimal.css"), []byte("body{color:red}"), 0644)

    req := httptest.NewRequest(http.MethodGet, "/render/cv/base.html?theme=minimal", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "body{color:red}")
}
```

- [ ] **Step 2: Run tests to see them fail**

```bash
go test ./internal/dashboard/... -run TestRenderCV -v
```

Expected: FAIL — routes not registered yet.

- [ ] **Step 3: Create `internal/dashboard/render.go`**

```go
package dashboard

import (
    "fmt"
    "net/http"

    "github.com/oxGrad/spicebag/internal/fs"
)

// writeDocument writes a full standalone HTML page wrapping fragment with optional CSS.
func writeDocument(w http.ResponseWriter, fragment, css string) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><style>%s</style></head>
<body>%s</body>
</html>`, css, fragment)
}

func (s *Server) handleRenderCV(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")
    fragment, err := fs.ReadCV(s.root, name)
    if err != nil {
        http.NotFound(w, r)
        return
    }
    css := s.themeCSS(r.URL.Query().Get("theme"))
    writeDocument(w, fragment, css)
}

func (s *Server) handleRenderCL(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")
    fragment, err := fs.ReadCoverLetter(s.root, name)
    if err != nil {
        http.NotFound(w, r)
        return
    }
    css := s.themeCSS(r.URL.Query().Get("theme"))
    writeDocument(w, fragment, css)
}

// themeCSS returns the CSS for a named theme, or "" if name is empty or not found.
func (s *Server) themeCSS(name string) string {
    if name == "" {
        return ""
    }
    css, _ := fs.ReadTheme(s.root, name)
    return css
}
```

- [ ] **Step 4: Register render routes in `internal/dashboard/server.go`**

Add to `routes()` before the SPA catch-all:
```go
s.mux.HandleFunc("GET /render/cv/{name}", s.handleRenderCV)
s.mux.HandleFunc("GET /render/cl/{name}", s.handleRenderCL)
```

- [ ] **Step 5: Update `handleExport` in `handlers_themes.go` to read HTML fragment directly**

Replace the markdown read + render block:
```go
// read the HTML fragment
fragment, err := func() (string, error) {
    rootClean := filepath.Clean(s.root)
    resolved := filepath.Join(rootClean, filePath)
    if !strings.HasPrefix(resolved, rootClean+string(os.PathSeparator)) {
        return "", fmt.Errorf("invalid file_path")
    }
    data, err := os.ReadFile(resolved)
    return string(data), err
}()
if err != nil {
    http.Error(w, fmt.Sprintf("read file: %v", err), http.StatusNotFound)
    return
}

css := s.themeCSS(theme)
client := pdf.NewClient(s.cfg.GotenbergURL)
pdfBytes, err := client.RenderPDF(fragment, css)
```

Also remove the `RenderMarkdown` call and the markdown import from `handlers_themes.go`.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/dashboard/... -v
```

Expected: all pass including new render tests.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/render.go internal/dashboard/server.go internal/dashboard/handlers_themes.go internal/dashboard/dashboard_test.go
git commit -m "feat: add render endpoints for iframe document preview"
```

---

## Task 4: JSON API — Applications

**Files:**
- Modify: `internal/dashboard/handlers_apps.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing JSON API tests**

In `dashboard_test.go`, add:
```go
func TestAPIAppsList(t *testing.T) {
    srv := newTestServer(t)
    req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
    assert.Contains(t, w.Body.String(), "[")
}

func TestAPIAppDetail(t *testing.T) {
    srv := newTestServer(t)
    id, _ := srv.Store().UpsertApplication(db.Application{
        Company: "Stripe", Role: "SRE", AppliedDate: "2025-01-01", FolderPath: "stripe/sre",
    })
    req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d", id), nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "Stripe")
}
```

- [ ] **Step 2: Run to see fail**

```bash
go test ./internal/dashboard/... -run TestAPIApps -v
```

Expected: FAIL — `/api/apps` not registered.

- [ ] **Step 3: Rewrite `internal/dashboard/handlers_apps.go` for JSON**

```go
package dashboard

import (
    "encoding/json"
    "fmt"
    "net/http"
    "slices"

    "github.com/oxGrad/spicebag/internal/db"
)

var validStatuses = []string{"applied", "assessment", "interview", "offer", "rejected", "withdrawn", "ghosted"}

func isValidStatus(s string) bool { return slices.Contains(validStatuses, s) }

func writeJSON(w http.ResponseWriter, v any) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(v)
}

func (s *Server) handleAPIAppsList(w http.ResponseWriter, r *http.Request) {
    apps, err := s.store.ListApplicationsWithStatus()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, apps)
}

type appDetailResponse struct {
    App           db.Application          `json:"app"`
    History       []db.StatusHistoryEntry `json:"history"`
    ValidStatuses []string                `json:"valid_statuses"`
}

func (s *Server) handleAPIAppDetail(w http.ResponseWriter, r *http.Request) {
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
    writeJSON(w, appDetailResponse{App: app, History: history, ValidStatuses: validStatuses})
}

func (s *Server) handleAPIAppStatusUpdate(w http.ResponseWriter, r *http.Request) {
    id, ok := parseID(r, "id")
    if !ok {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }
    status := r.FormValue("status")
    notes := r.FormValue("notes")
    if !isValidStatus(status) {
        http.Error(w, "invalid status", http.StatusBadRequest)
        return
    }
    if err := s.store.AddStatusHistory(id, status, notes); err != nil {
        http.Error(w, fmt.Sprintf("update status: %v", err), http.StatusInternalServerError)
        return
    }
    history, err := s.store.GetStatusHistory(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, history)
}
```

- [ ] **Step 4: Register API routes in `server.go`**

Replace old app routes with:
```go
s.mux.HandleFunc("GET /api/apps", s.handleAPIAppsList)
s.mux.HandleFunc("GET /api/apps/{id}", s.handleAPIAppDetail)
s.mux.HandleFunc("POST /api/apps/{id}/status", s.handleAPIAppStatusUpdate)
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/dashboard/... -v
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add internal/dashboard/handlers_apps.go internal/dashboard/server.go internal/dashboard/dashboard_test.go
git commit -m "feat: replace app HTML handlers with JSON API"
```

---

## Task 5: JSON API — CVs, Cover Letters, Themes

**Files:**
- Modify: `internal/dashboard/handlers_cv.go`
- Modify: `internal/dashboard/handlers_cl.go`
- Modify: `internal/dashboard/handlers_themes.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestAPICVList(t *testing.T) {
    srv := newTestServer(t)
    fs.WriteCV(srv.Root(), "base.html", "<h1>Test</h1>")
    req := httptest.NewRequest(http.MethodGet, "/api/cv", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "base.html")
}

func TestAPIThemesList(t *testing.T) {
    srv := newTestServer(t)
    os.WriteFile(filepath.Join(srv.Root(), "themes", "minimal.css"), []byte("body{}"), 0644)
    req := httptest.NewRequest(http.MethodGet, "/api/themes", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "minimal")
}
```

- [ ] **Step 2: Run to see fail**

```bash
go test ./internal/dashboard/... -run TestAPICV -v
```

Expected: FAIL.

- [ ] **Step 3: Rewrite `internal/dashboard/handlers_cv.go`**

```go
package dashboard

import (
    "net/http"
    "github.com/oxGrad/spicebag/internal/fs"
)

func (s *Server) handleAPICVList(w http.ResponseWriter, r *http.Request) {
    files, err := fs.ListCVs(s.root)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, files)
}
```

- [ ] **Step 4: Rewrite `internal/dashboard/handlers_cl.go`**

```go
package dashboard

import (
    "net/http"
    "github.com/oxGrad/spicebag/internal/fs"
)

func (s *Server) handleAPICLList(w http.ResponseWriter, r *http.Request) {
    files, err := fs.ListCoverLetters(s.root)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, files)
}
```

- [ ] **Step 5: Rewrite `internal/dashboard/handlers_themes.go` for JSON list + keep upload + export**

Keep `handleThemeUpload` and `handleExport` as-is (from Task 3). Replace `handleThemesList` and `handleThemePreview` with:

```go
func (s *Server) handleAPIThemesList(w http.ResponseWriter, r *http.Request) {
    themes, _ := fs.ListThemes(s.root)
    if themes == nil {
        themes = []string{}
    }
    writeJSON(w, themes)
}
```

Remove `handleThemePreview` (theme preview is now done via the render endpoint with `?theme=x` in the Vue iframe).

- [ ] **Step 6: Update routes in `server.go`**

Replace old CV/CL/theme routes:
```go
s.mux.HandleFunc("GET /api/cv", s.handleAPICVList)
s.mux.HandleFunc("GET /api/cl", s.handleAPICLList)
s.mux.HandleFunc("GET /api/themes", s.handleAPIThemesList)
s.mux.HandleFunc("POST /api/themes/upload", s.handleThemeUpload)
s.mux.HandleFunc("POST /api/export", s.handleExport)
```

Remove old routes: `GET /cv`, `GET /cv/{name}`, `GET /cl`, `GET /cl/{name}`, `GET /themes`, `GET /themes/{name}/preview`, `POST /themes/upload`, `POST /export`.

- [ ] **Step 7: Run tests**

```bash
go test ./internal/dashboard/... -v
```

Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/dashboard/handlers_cv.go internal/dashboard/handlers_cl.go internal/dashboard/handlers_themes.go internal/dashboard/server.go internal/dashboard/dashboard_test.go
git commit -m "feat: replace CV/CL/theme HTML handlers with JSON API"
```

---

## Task 6: JSON API — Stats and Gotenberg

**Files:**
- Modify: `internal/dashboard/handlers_stats.go`
- Modify: `internal/dashboard/handlers_gotenberg.go`
- Modify: `internal/dashboard/server.go`
- Modify: `internal/dashboard/dashboard_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestAPIStatsReturnsJSON(t *testing.T) {
    srv := newTestServer(t)
    req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestAPIGotenbergStatus(t *testing.T) {
    srv := newTestServer(t)
    req := httptest.NewRequest(http.MethodGet, "/api/gotenberg/status", nil)
    w := httptest.NewRecorder()
    srv.ServeHTTP(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
    assert.Contains(t, w.Body.String(), "running")
}
```

- [ ] **Step 2: Run to see fail**

```bash
go test ./internal/dashboard/... -run TestAPIStats -v
```

Expected: FAIL.

- [ ] **Step 3: Rewrite `internal/dashboard/handlers_stats.go`**

Remove the sync-from-CV feature (experience is managed via MCP tools). Keep stats read:

```go
package dashboard

import (
    "net/http"
)

func (s *Server) handleAPIStats(w http.ResponseWriter, r *http.Request) {
    stats, err := s.store.GetExperienceStats()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    writeJSON(w, stats)
}
```

- [ ] **Step 4: Rewrite Gotenberg handlers for JSON**

Replace `handleGotenbergStatus`, `handleGotenbergStart`, `handleGotenbergStop` in `handlers_gotenberg.go`:

```go
type gotenbergStatusJSON struct {
    Running bool   `json:"running"`
    Err     string `json:"error,omitempty"`
}

func (s *Server) handleAPIGotenbergStatus(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, gotenbergStatusJSON{Running: s.checkGotenberg()})
}

func (s *Server) handleAPIGotenbergStart(w http.ResponseWriter, r *http.Request) {
    resp := gotenbergStatusJSON{}
    if err := s.startGotenberg(); err != nil {
        resp.Err = err.Error()
        resp.Running = s.checkGotenberg()
    } else {
        resp.Running = s.waitForGotenberg(5)
    }
    writeJSON(w, resp)
}

func (s *Server) handleAPIGotenbergStop(w http.ResponseWriter, r *http.Request) {
    resp := gotenbergStatusJSON{}
    if err := s.stopGotenberg(); err != nil {
        resp.Err = err.Error()
    }
    resp.Running = s.checkGotenberg()
    writeJSON(w, resp)
}
```

Remove `gotenbergStatusData` struct and `renderPartial` calls.

- [ ] **Step 5: Update routes in `server.go`**

```go
s.mux.HandleFunc("GET /api/stats", s.handleAPIStats)
s.mux.HandleFunc("GET /api/gotenberg/status", s.handleAPIGotenbergStatus)
s.mux.HandleFunc("POST /api/gotenberg/start", s.handleAPIGotenbergStart)
s.mux.HandleFunc("POST /api/gotenberg/stop", s.handleAPIGotenbergStop)
```

Remove old routes: `GET /stats`, `POST /stats/sync`, `GET /gotenberg/status`, `POST /gotenberg/start`, `POST /gotenberg/stop`.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/dashboard/... -v
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/dashboard/handlers_stats.go internal/dashboard/handlers_gotenberg.go internal/dashboard/server.go internal/dashboard/dashboard_test.go
git commit -m "feat: replace stats/gotenberg HTML handlers with JSON API"
```

---

## Task 7: Vue — api.js + Router

**Files:**
- Create: `frontend/src/api.js`
- Modify: `frontend/src/router/index.js` (replace stub in `main.js`)
- Modify: `frontend/src/main.js`

- [ ] **Step 1: Create `frontend/src/api.js`**

```js
const base = '/api'

async function get(path) {
  const res = await fetch(base + path)
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}`)
  return res.json()
}

async function post(path, body) {
  const isFormData = body instanceof FormData
  const res = await fetch(base + path, {
    method: 'POST',
    headers: isFormData ? {} : { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: isFormData ? body : new URLSearchParams(body),
  })
  if (!res.ok) throw new Error(`POST ${path} → ${res.status}`)
  return res.json()
}

export const api = {
  apps: {
    list: ()              => get('/apps'),
    get:  (id)            => get(`/apps/${id}`),
    updateStatus: (id, status, notes) =>
      post(`/apps/${id}/status`, { status, notes: notes ?? '' }),
  },
  cv: {
    list: () => get('/cv'),
  },
  cl: {
    list: () => get('/cl'),
  },
  themes: {
    list:   ()     => get('/themes'),
    upload: (file) => {
      const fd = new FormData()
      fd.append('theme', file)
      return post('/themes/upload', fd)
    },
  },
  stats: {
    get: () => get('/stats'),
  },
  gotenberg: {
    status: ()  => get('/gotenberg/status'),
    start:  ()  => post('/gotenberg/start', {}),
    stop:   ()  => post('/gotenberg/stop', {}),
  },
  export: (filePath, theme) =>
    fetch('/api/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({ file_path: filePath, theme }),
    }),
}
```

- [ ] **Step 2: Create `frontend/src/router/index.js`**

```js
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/',           component: () => import('../views/AppsView.vue') },
  { path: '/apps/:id',   component: () => import('../views/AppDetailView.vue') },
  { path: '/cv',         component: () => import('../views/CVListView.vue') },
  { path: '/cv/:name',   component: () => import('../views/CVView.vue') },
  { path: '/cl',         component: () => import('../views/CLListView.vue') },
  { path: '/cl/:name',   component: () => import('../views/CLView.vue') },
  { path: '/themes',     component: () => import('../views/ThemesView.vue') },
  { path: '/stats',      component: () => import('../views/StatsView.vue') },
]

export default createRouter({ history: createWebHistory(), routes })
```

- [ ] **Step 3: Update `frontend/src/main.js` to use router file**

```js
import { createApp } from 'vue'
import router from './router/index.js'
import App from './App.vue'
import './style.css'

createApp(App).use(router).mount('#app')
```

- [ ] **Step 4: Build to verify no errors**

```bash
cd frontend && npm run build
```

Expected: builds successfully.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/api.js frontend/src/router/ frontend/src/main.js
git commit -m "feat: add Vue Router and API client"
```

---

## Task 8: Vue — Applications page

**Files:**
- Create: `frontend/src/views/AppsView.vue`

- [ ] **Step 1: Create `frontend/src/views/AppsView.vue`**

```vue
<template>
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
        <tr v-if="apps.length === 0">
          <td colspan="4" class="px-4 py-8 text-center text-gray-400">
            No applications yet. Use <code>/apply</code> in Claude Code to create one.
          </td>
        </tr>
        <tr
          v-for="app in apps"
          :key="app.ID"
          class="hover:bg-gray-50 cursor-pointer"
          @click="$router.push(`/apps/${app.ID}`)"
        >
          <td class="px-4 py-3 font-medium">{{ app.Company }}</td>
          <td class="px-4 py-3 text-gray-600">{{ app.Role }}</td>
          <td class="px-4 py-3 text-gray-500">{{ app.AppliedDate }}</td>
          <td class="px-4 py-3">
            <span class="px-2 py-0.5 rounded text-xs font-semibold" :class="badgeClass(app.CurrentStatus)">
              {{ app.CurrentStatus }}
            </span>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const apps = ref([])

onMounted(async () => {
  apps.value = await api.apps.list()
})

function badgeClass(status) {
  const map = {
    offer: 'bg-green-100 text-green-800',
    interview: 'bg-yellow-100 text-yellow-800',
    assessment: 'bg-yellow-100 text-yellow-800',
    rejected: 'bg-red-100 text-red-800',
    withdrawn: 'bg-red-100 text-red-800',
    ghosted: 'bg-red-100 text-red-800',
  }
  return map[status?.toLowerCase()] ?? 'bg-blue-100 text-blue-800'
}
</script>
```

- [ ] **Step 2: Build and verify**

```bash
cd frontend && npm run build
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/AppsView.vue
git commit -m "feat: Vue Applications list page"
```

---

## Task 9: Vue — App detail page

**Files:**
- Create: `frontend/src/views/AppDetailView.vue`

- [ ] **Step 1: Create `frontend/src/views/AppDetailView.vue`**

```vue
<template>
  <div class="mb-4">
    <RouterLink to="/" class="text-blue-600 hover:underline text-sm">← Applications</RouterLink>
  </div>
  <div v-if="detail" class="mb-6">
    <h1 class="text-2xl font-bold">{{ detail.app.Company }}</h1>
    <p class="text-gray-500">{{ detail.app.Role }} · Applied {{ detail.app.AppliedDate }}</p>
  </div>
  <div v-if="detail" class="grid grid-cols-2 gap-6">
    <div class="bg-white rounded-lg shadow p-5">
      <h2 class="font-semibold mb-3">Status History</h2>
      <div class="space-y-2 mb-4">
        <div v-for="entry in detail.history" :key="entry.ID" class="flex gap-3 text-sm">
          <span class="text-gray-400 text-xs w-24 shrink-0">{{ entry.ChangedAt }}</span>
          <span class="px-2 py-0.5 rounded text-xs font-semibold" :class="badgeClass(entry.Status)">{{ entry.Status }}</span>
          <span v-if="entry.Notes" class="text-gray-500">{{ entry.Notes }}</span>
        </div>
      </div>
      <form class="flex gap-2 items-end" @submit.prevent="submitStatus">
        <div class="flex flex-col gap-1">
          <label class="text-xs text-gray-500">New Status</label>
          <select v-model="newStatus" class="border rounded px-2 py-1.5 text-sm">
            <option v-for="s in detail.valid_statuses" :key="s" :value="s">{{ s }}</option>
          </select>
        </div>
        <div class="flex flex-col gap-1">
          <label class="text-xs text-gray-500">Notes (optional)</label>
          <input v-model="newNotes" type="text" class="border rounded px-2 py-1.5 text-sm w-40">
        </div>
        <button type="submit" class="bg-blue-600 text-white px-3 py-1.5 rounded text-sm">Update</button>
      </form>
    </div>
    <div class="bg-white rounded-lg shadow p-5">
      <h2 class="font-semibold mb-3">Notes</h2>
      <p v-if="detail.app.Notes" class="text-sm text-gray-700">{{ detail.app.Notes }}</p>
      <p v-else class="text-gray-400 text-sm">No notes.</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'

const route = useRoute()
const detail = ref(null)
const newStatus = ref('')
const newNotes = ref('')

onMounted(async () => {
  detail.value = await api.apps.get(route.params.id)
  newStatus.value = detail.value.valid_statuses[0]
})

async function submitStatus() {
  const updated = await api.apps.updateStatus(route.params.id, newStatus.value, newNotes.value)
  detail.value.history = updated
  newNotes.value = ''
}

function badgeClass(status) {
  const map = {
    offer: 'bg-green-100 text-green-800',
    interview: 'bg-yellow-100 text-yellow-800',
    assessment: 'bg-yellow-100 text-yellow-800',
    rejected: 'bg-red-100 text-red-800',
    withdrawn: 'bg-red-100 text-red-800',
    ghosted: 'bg-red-100 text-red-800',
  }
  return map[status?.toLowerCase()] ?? 'bg-blue-100 text-blue-800'
}
</script>
```

- [ ] **Step 2: Build and verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/AppDetailView.vue
git commit -m "feat: Vue App detail page with status history"
```

---

## Task 10: Vue — CV library + viewer with iframe

**Files:**
- Create: `frontend/src/views/CVListView.vue`
- Create: `frontend/src/views/CVView.vue`

- [ ] **Step 1: Create `frontend/src/views/CVListView.vue`**

```vue
<template>
  <h1 class="text-2xl font-bold mb-6">CV Library</h1>
  <div class="bg-white rounded-lg shadow divide-y">
    <div v-if="files.length === 0" class="px-4 py-8 text-center text-gray-400">
      No CVs yet. Use <code>/seed-cv</code> in Claude Code to import one.
    </div>
    <div
      v-for="f in files"
      :key="f.Name"
      class="flex items-center justify-between px-4 py-3 hover:bg-gray-50 cursor-pointer"
      @click="$router.push(`/cv/${f.Name}`)"
    >
      <span class="font-medium">{{ f.Name }}</span>
      <span class="text-xs text-gray-400">{{ formatDate(f.ModifiedAt) }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const files = ref([])

onMounted(async () => { files.value = await api.cv.list() })

function formatDate(iso) {
  return new Date(iso).toLocaleDateString()
}
</script>
```

- [ ] **Step 2: Create `frontend/src/views/CVView.vue`**

```vue
<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/cv" class="text-blue-600 hover:underline text-sm">← CV Library</RouterLink>
    <div class="flex gap-2 items-center">
      <select v-model="selectedTheme" @change="onThemeChange" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
      </select>
      <GotenbergWidget :file-path="`cv/${name}`" :theme="selectedTheme" />
    </div>
  </div>
  <h1 class="text-xl font-bold mb-4">{{ name }}</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 80vh;">
    <iframe
      :src="renderSrc"
      class="w-full h-full border-0"
      title="CV Preview"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import GotenbergWidget from '../components/GotenbergWidget.vue'

const route = useRoute()
const name = computed(() => route.params.name)
const themes = ref([])
const selectedTheme = ref('')

const renderSrc = computed(() => {
  const t = selectedTheme.value ? `?theme=${encodeURIComponent(selectedTheme.value)}` : ''
  return `/render/cv/${name.value}${t}`
})

onMounted(async () => {
  themes.value = await api.themes.list()
})

function onThemeChange() {
  // renderSrc computed property automatically updates the iframe src
}
</script>
```

- [ ] **Step 3: Build and verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/CVListView.vue frontend/src/views/CVView.vue
git commit -m "feat: Vue CV library and viewer with iframe preview"
```

---

## Task 11: Vue — Cover letters + viewer

**Files:**
- Create: `frontend/src/views/CLListView.vue`
- Create: `frontend/src/views/CLView.vue`
- Create: `frontend/src/components/GotenbergWidget.vue`

- [ ] **Step 1: Create `frontend/src/components/GotenbergWidget.vue`**

This component shows Gotenberg status and the Export PDF button. Used in both CVView and CLView.

```vue
<template>
  <div class="flex gap-2 items-center">
    <button
      v-if="!running"
      @click="start"
      :disabled="starting"
      class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50 disabled:opacity-50"
    >
      {{ starting ? 'Starting…' : 'Start Gotenberg' }}
    </button>
    <button
      v-if="running"
      @click="exportPDF"
      class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700"
    >
      Export PDF
    </button>
    <button
      v-if="running"
      @click="stop"
      class="border rounded px-3 py-1.5 text-sm bg-white hover:bg-gray-50 text-gray-500"
    >
      Stop
    </button>
    <span v-if="err" class="text-xs text-red-500">{{ err }}</span>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const props = defineProps({
  filePath: { type: String, required: true },
  theme:    { type: String, default: '' },
})

const running  = ref(false)
const starting = ref(false)
const err      = ref('')

onMounted(async () => {
  const s = await api.gotenberg.status()
  running.value = s.running
})

async function start() {
  starting.value = true
  err.value = ''
  const s = await api.gotenberg.start()
  running.value = s.running
  if (s.error) err.value = s.error
  starting.value = false
}

async function stop() {
  const s = await api.gotenberg.stop()
  running.value = s.running
}

async function exportPDF() {
  err.value = ''
  const res = await api.export(props.filePath, props.theme)
  if (!res.ok) { err.value = 'Export failed'; return }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = props.filePath.split('/').pop().replace('.html', '.pdf')
  a.click()
  URL.revokeObjectURL(url)
}
</script>
```

- [ ] **Step 2: Create `frontend/src/views/CLListView.vue`**

```vue
<template>
  <h1 class="text-2xl font-bold mb-6">Cover Letters</h1>
  <div class="bg-white rounded-lg shadow divide-y">
    <div v-if="files.length === 0" class="px-4 py-8 text-center text-gray-400">
      No cover letters yet. Use <code>/apply</code> in Claude Code to create one.
    </div>
    <div
      v-for="f in files"
      :key="f.Name"
      class="flex items-center justify-between px-4 py-3 hover:bg-gray-50 cursor-pointer"
      @click="$router.push(`/cl/${f.Name}`)"
    >
      <span class="font-medium">{{ f.Name }}</span>
      <span class="text-xs text-gray-400">{{ new Date(f.ModifiedAt).toLocaleDateString() }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const files = ref([])
onMounted(async () => { files.value = await api.cl.list() })
</script>
```

- [ ] **Step 3: Create `frontend/src/views/CLView.vue`**

```vue
<template>
  <div class="mb-4 flex items-center justify-between">
    <RouterLink to="/cl" class="text-blue-600 hover:underline text-sm">← Cover Letters</RouterLink>
    <div class="flex gap-2 items-center">
      <select v-model="selectedTheme" class="border rounded px-2 py-1.5 text-sm bg-white">
        <option value="">No theme</option>
        <option v-for="t in themes" :key="t" :value="t">{{ t }}</option>
      </select>
      <GotenbergWidget :file-path="`cover-letters/${name}`" :theme="selectedTheme" />
    </div>
  </div>
  <h1 class="text-xl font-bold mb-4">{{ name }}</h1>
  <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 80vh;">
    <iframe :src="renderSrc" class="w-full h-full border-0" title="Cover Letter Preview" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api.js'
import GotenbergWidget from '../components/GotenbergWidget.vue'

const route = useRoute()
const name = computed(() => route.params.name)
const themes = ref([])
const selectedTheme = ref('')

const renderSrc = computed(() => {
  const t = selectedTheme.value ? `?theme=${encodeURIComponent(selectedTheme.value)}` : ''
  return `/render/cl/${name.value}${t}`
})

onMounted(async () => { themes.value = await api.themes.list() })
</script>
```

- [ ] **Step 4: Build and verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/CLListView.vue frontend/src/views/CLView.vue frontend/src/components/GotenbergWidget.vue
git commit -m "feat: Vue cover letter viewer and Gotenberg widget component"
```

---

## Task 12: Vue — Themes page

**Files:**
- Create: `frontend/src/views/ThemesView.vue`

- [ ] **Step 1: Create `frontend/src/views/ThemesView.vue`**

```vue
<template>
  <h1 class="text-2xl font-bold mb-6">Themes</h1>
  <div class="grid grid-cols-2 gap-6">
    <div>
      <h2 class="font-semibold mb-3">Available Themes</h2>
      <div class="bg-white rounded-lg shadow divide-y mb-6">
        <div v-if="themes.length === 0" class="px-4 py-8 text-center text-gray-400">No themes yet.</div>
        <div
          v-for="t in themes"
          :key="t"
          class="flex items-center justify-between px-4 py-3 hover:bg-gray-50"
        >
          <span class="font-medium">{{ t }}</span>
          <button @click="previewTheme = t" class="text-sm text-blue-600 hover:underline">Preview →</button>
        </div>
      </div>

      <h2 class="font-semibold mb-3">Upload New Theme</h2>
      <div class="bg-white rounded-lg shadow p-4 flex gap-3 items-end">
        <div class="flex flex-col gap-1">
          <label class="text-xs text-gray-500">CSS File</label>
          <input type="file" accept=".css" @change="onFileChange" class="text-sm">
        </div>
        <button
          @click="upload"
          :disabled="!selectedFile"
          class="bg-blue-600 text-white rounded px-3 py-1.5 text-sm hover:bg-blue-700 disabled:opacity-50"
        >Upload</button>
      </div>
    </div>

    <div v-if="previewTheme">
      <h2 class="font-semibold mb-3">Preview: {{ previewTheme }}</h2>
      <div class="bg-white rounded-lg shadow overflow-hidden" style="height: 60vh;">
        <iframe
          :src="`/render/cv/base.html?theme=${encodeURIComponent(previewTheme)}`"
          class="w-full h-full border-0"
          title="Theme Preview"
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const themes       = ref([])
const previewTheme = ref('')
const selectedFile = ref(null)

onMounted(async () => { themes.value = await api.themes.list() })

function onFileChange(e) { selectedFile.value = e.target.files[0] ?? null }

async function upload() {
  if (!selectedFile.value) return
  await api.themes.upload(selectedFile.value)
  themes.value = await api.themes.list()
  selectedFile.value = null
}
</script>
```

- [ ] **Step 2: Build and verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/ThemesView.vue
git commit -m "feat: Vue Themes page with upload and iframe preview"
```

---

## Task 13: Vue — Stats page

**Files:**
- Create: `frontend/src/views/StatsView.vue`

- [ ] **Step 1: Create `frontend/src/views/StatsView.vue`**

```vue
<template>
  <h1 class="text-2xl font-bold mb-6">Experience Stats</h1>
  <div v-if="stats" class="bg-white rounded-lg shadow overflow-hidden">
    <table class="w-full text-sm">
      <thead class="bg-gray-100 text-gray-600 uppercase text-xs">
        <tr>
          <th class="text-left px-4 py-3">Role Type</th>
          <th class="text-left px-4 py-3">Years</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-gray-100">
        <tr v-if="!stats.Entries || stats.Entries.length === 0">
          <td colspan="2" class="px-4 py-8 text-center text-gray-400">
            No experience data. Use <code>add_experience</code> MCP tool to add entries.
          </td>
        </tr>
        <tr v-for="entry in stats.Entries" :key="entry.RoleType" class="hover:bg-gray-50">
          <td class="px-4 py-3 font-medium">{{ entry.RoleType }}</td>
          <td class="px-4 py-3 text-gray-600">{{ entry.Years }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <p class="text-xs text-gray-400 mt-2">Experience is managed via the <code>add_experience</code> MCP tool in Claude Code.</p>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api.js'

const stats = ref(null)
onMounted(async () => { stats.value = await api.stats.get() })
</script>
```

- [ ] **Step 2: Build and verify**

```bash
cd frontend && npm run build
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/StatsView.vue
git commit -m "feat: Vue Stats page"
```

---

## Task 14: Cleanup — remove Go templates, markdown, parser

**Files:**
- Delete: `internal/dashboard/templates/` (all files)
- Delete: `internal/dashboard/markdown.go`
- Delete: `internal/parser/parser.go`
- Delete: `internal/parser/parser_test.go`
- Modify: `internal/dashboard/server.go` — remove `templateFS`, `render`, `renderPartial`
- Modify: `internal/dashboard/handlers_cv.go` — remove `scopeCSS`, unused imports
- Modify: `internal/dashboard/handlers_gotenberg.go` — remove `gotenbergStatusData`

- [ ] **Step 1: Delete Go templates directory**

```bash
rm -rf internal/dashboard/templates/
```

- [ ] **Step 2: Delete `internal/dashboard/markdown.go`**

```bash
rm internal/dashboard/markdown.go
```

- [ ] **Step 3: Delete parser package**

```bash
rm internal/parser/parser.go internal/parser/parser_test.go
rmdir internal/parser
```

- [ ] **Step 4: Remove `templateFS`, `render`, `renderPartial` from `server.go`**

Remove:
```go
//go:embed templates
var templateFS embed.FS
```

Remove the `render` and `renderPartial` methods from `server.go`.

Keep `uiFS` and `handleSPA`.

- [ ] **Step 5: Remove `scopeCSS` and dead imports from `handlers_cv.go`**

`handlers_cv.go` now only contains `handleAPICVList`. Remove `scopeCSS`, `cssRuleRe`, and all unused imports (`regexp`, `strings`, `html/template`).

- [ ] **Step 6: Build to verify no broken references**

```bash
go build ./...
```

Fix any remaining compilation errors from dead references.

- [ ] **Step 7: Run tests**

```bash
go test ./...
```

Expected: all pass. `internal/parser` package is gone so any test importing it will need updating — check `dashboard_test.go` for any remaining `parser` imports.

- [ ] **Step 8: Commit**

```bash
git add -A
git commit -m "chore: remove Go HTML templates, markdown renderer, and parser package"
```

---

## Task 15: Update skills for HTML fragments

**Files:**
- Modify: `plugins/skills/apply.md`
- Modify: `plugins/skills/customize-cv.md`
- Modify: `plugins/skills/write-cover-letter.md`

- [ ] **Step 1: Update `plugins/skills/apply.md`**

Change step 6 to produce HTML fragments instead of markdown. Replace the tailored CV and cover letter instructions:

In the "Write both documents in one pass" step, replace:
```
- **Tailored CV**: adjust emphasis and wording for this role (keep all facts accurate)
- **Cover letter**: company-specific opener, concrete experience references, under 350 words
```

With:
```
- **Tailored CV**: HTML fragment (no DOCTYPE/html/head/body tags — just semantic body content). Use `<h1>` for name, `<h2>` for sections, `<h3>` for job titles, `<ul>/<li>` for bullets. Adjust emphasis for this role; keep all facts accurate.
- **Cover letter**: HTML fragment. Use `<header>` with `<h1>` for name and `<div class="contact">` for details, `<div class="date-block">` for date, `<p class="salutation">` for greeting, `<p>` for body paragraphs, `<div class="closing">` with `<div class="sign-off">` and `<div class="name">`. Under 350 words of body text.
```

Change step 7 `create_application`:
- `cv_content`: the tailored CV HTML fragment
- `cover_letter_content`: the cover letter HTML fragment

Change filenames in step 7:
- `cv_content` saved as `cv.html`
- `cover_letter_content` saved as `cover-letter.html`

- [ ] **Step 2: Update `plugins/skills/customize-cv.md`**

Change step 8 `write_cv` to use filename `cv-{role-type}-{YYYY-MM-DD}.html` (not `.md`).

Change step 3 `read_cv` — note that CVs are now HTML fragments.

Change step 6 rewrite instruction:
```
Rewrite as an HTML fragment (no DOCTYPE/html/head/body — just semantic body content). Use <h1> for name, <h2> for section headings, <h3> for job titles/companies, <ul>/<li> for bullets. Keep all factual data accurate.
```

- [ ] **Step 3: Update `plugins/skills/write-cover-letter.md`**

Read this file first, then change it to produce an HTML fragment cover letter in the same format described in Task 15 Step 1.

- [ ] **Step 4: Commit**

```bash
git add plugins/skills/
git commit -m "feat: update skills to generate HTML fragments instead of markdown"
```

---

## Task 16: Goreleaser — add frontend build hook

**Files:**
- Modify: `.goreleaser.yaml`

- [ ] **Step 1: Add `npm ci && npm run build` to `before.hooks` in `.goreleaser.yaml`**

```yaml
before:
  hooks:
    - go mod tidy
    - sh scripts/set-plugin-version.sh {{ .Version }}
    - bash -c "cd frontend && npm ci && npm run build"
```

- [ ] **Step 2: Verify goreleaser config parses**

```bash
goreleaser check
```

Expected: no errors. (If goreleaser isn't installed locally, skip and rely on CI.)

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "chore: add frontend build step to goreleaser hooks"
```

---

## Self-Review

**Spec coverage:**
- ✅ Vue 3 SPA replacing htmx templates
- ✅ HTML fragments as document format
- ✅ Render endpoints for iframe-isolated document preview
- ✅ iframe-based preview (no CSS bleed)
- ✅ Theme switching via iframe `src` URL change
- ✅ Gotenberg widget for start/stop/export
- ✅ All existing pages ported (apps, CV, CL, themes, stats)
- ✅ Go binary embeds built Vue assets
- ✅ Skills updated to generate HTML fragments
- ✅ Goreleaser builds frontend before Go binary

**Removed features (intentional):**
- Stats "sync from CV" — experience is now managed via MCP tools directly. The parser package is deleted.
- `handleThemePreview` Go endpoint — replaced by iframe render with `?theme=x`

**Placeholder scan:** None found.

**Type consistency:**
- `api.js` `post('/apps/{id}/status')` matches `handleAPIAppStatusUpdate` which reads `r.FormValue("status")` and `r.FormValue("notes")` ✅
- `api.export` returns a `Response` (not parsed JSON) — `GotenbergWidget` calls `res.blob()` directly ✅
- `GotenbergWidget` emits nothing; parent components read `selectedTheme` from their own state ✅
- `writeJSON` helper defined in `handlers_apps.go` — used by all handler files in same package ✅
