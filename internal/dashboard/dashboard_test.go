// internal/dashboard/dashboard_test.go
package dashboard_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/graditya/prospector/internal/config"
	"github.com/graditya/prospector/internal/dashboard"
	"github.com/graditya/prospector/internal/db"
	"github.com/graditya/prospector/internal/fs"
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
