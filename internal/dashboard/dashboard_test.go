// internal/dashboard/dashboard_test.go
package dashboard_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/dashboard"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *dashboard.Server {
	t.Helper()
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
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

func TestAppsListRoute(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/apps", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "[")
}

func TestAppDetailRoute(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Stripe", Role: "Engineer",
		AppliedDate: "2025-01-01", FolderPath: "stripe/engineer/2025-01-01",
	})
	require.NoError(t, err)
	store.AddStatusHistory(id, "applied", "")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d", id), nil)
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
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/status", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "interview")
}

func TestCVListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1>"))

	req := httptest.NewRequest(http.MethodGet, "/cv", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cv-backend-2025-01-01.html")
}

func TestCVViewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1><p>Content here.</p>"))

	req := httptest.NewRequest(http.MethodGet, "/cv/cv-backend-2025-01-01.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cv-backend-2025-01-01.html")
}

func TestCLListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.html", "<p>Dear Hiring Manager</p>"))

	req := httptest.NewRequest(http.MethodGet, "/cl", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cl-general-2025-01-01.html")
}

func TestCLViewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.html", "<p>Dear Hiring Manager</p><p>I am excited...</p>"))

	req := httptest.NewRequest(http.MethodGet, "/cl/cl-general-2025-01-01.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cl-general-2025-01-01.html")
}

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
	cvContent := "---\nexperience:\n  - role_type: devops\n    company: FooCo\n    start: \"2021-01-01\"\n    end: \"2023-01-01\"\n---\n<h1>CV</h1>\n"
	require.NoError(t, fs.WriteCV(root, "cv-devops-2025-01-01.html", cvContent))

	req := httptest.NewRequest(http.MethodPost, "/stats/sync", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "devops")
}

func TestThemesListRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	os.WriteFile(filepath.Join(root, "themes", "minimal.css"), []byte("body { font-family: serif; }"), 0o644)

	req := httptest.NewRequest(http.MethodGet, "/themes", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "minimal")
}

func TestThemePreviewRoute(t *testing.T) {
	srv := newTestServer(t)
	root := srv.Root()
	os.WriteFile(filepath.Join(root, "themes", "minimal.css"), []byte("body { color: red; }"), 0o644)
	require.NoError(t, fs.WriteCV(root, "cv-test.html", "<h1>Hello World</h1>"))

	req := httptest.NewRequest(http.MethodGet, "/themes/minimal/preview?cv=cv-test.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "minimal")
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
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	cfg := config.Config{GotenbergURL: gotenberg.URL}
	srv := dashboard.NewServer(root, store, cfg)

	require.NoError(t, fs.WriteCV(root, "cv-test.html", "<h1>Hello</h1>"))

	form := strings.NewReader("file_path=cv%2Fcv-test.html&theme=")
	req := httptest.NewRequest(http.MethodPost, "/export", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}

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
	srv.SetGotenbergRunners(func() error { return nil }, func() error { return nil }) // no-op: skip docker

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
	srv.SetGotenbergRunners(func() error { return nil }, func() error { return nil })

	form := strings.NewReader("file_path=cover-letters%2Fcl.md&theme=")
	req := httptest.NewRequest(http.MethodPost, "/gotenberg/stop", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gotenberg stopped")
	assert.Contains(t, w.Body.String(), "disabled")
}

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

func TestRenderCL(t *testing.T) {
	srv := newTestServer(t)
	fragment := "<p>Dear Hiring Team,</p>"
	require.NoError(t, fs.WriteCoverLetter(srv.Root(), "cl.html", fragment))

	req := httptest.NewRequest(http.MethodGet, "/render/cl/cl.html", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "<p>Dear Hiring Team,</p>")
	assert.Contains(t, w.Body.String(), "<!DOCTYPE html>")
}

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
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "Stripe")
}

func TestAPIAppStatusUpdate(t *testing.T) {
	srv := newTestServer(t)
	id, _ := srv.Store().UpsertApplication(db.Application{
		Company: "X", Role: "Y", AppliedDate: "2025-01-01", FolderPath: "x/y",
	})
	srv.Store().AddStatusHistory(id, "applied", "")

	form := strings.NewReader("status=interview&notes=phone+screen")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/status", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "interview")
}
