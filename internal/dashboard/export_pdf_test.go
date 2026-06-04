package dashboard_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/dashboard"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newServerWithGotenberg(t *testing.T, gotenbergURL string) *dashboard.Server {
	t.Helper()
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	mem, err := memory.Open(filepath.Join(root, "memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { mem.Close() })
	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	return dashboard.NewServer(root, store, mem, config.Config{GotenbergURL: gotenbergURL})
}

func mockGotenbergServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write([]byte("%PDF-1.4 mock"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestExportPDFCV(t *testing.T) {
	gotenberg := mockGotenbergServer(t)
	srv := newServerWithGotenberg(t, gotenberg.URL)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Backend Engineer",
		AppliedDate: "2025-01-01", FolderPath: "acme/backend-engineer/2025-01-01",
	})
	require.NoError(t, err)

	// Create cv.html with an h1 to exercise extractH1 and pdfSlug
	docDir := filepath.Join(srv.Root(), "applications", "acme/backend-engineer/2025-01-01")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "cv.html"), []byte("<h1>Gatra Raditya</h1><p>Engineer</p>"), 0o644))

	form := strings.NewReader("doc_type=cv&theme=")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/pdf", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	// Should contain the generated PDF filename derived from h1 + role
	assert.Contains(t, w.Body.String(), "gatra-raditya")
	assert.Contains(t, w.Body.String(), "cv.pdf")
}

func TestExportPDFCL(t *testing.T) {
	gotenberg := mockGotenbergServer(t)
	srv := newServerWithGotenberg(t, gotenberg.URL)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev",
		AppliedDate: "2025-01-01", FolderPath: "acme/dev/2025-01-01",
	})
	require.NoError(t, err)

	docDir := filepath.Join(srv.Root(), "applications", "acme/dev/2025-01-01")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "cover-letter.html"), []byte("<h1>Gatra</h1><p>Dear Acme</p>"), 0o644))

	form := strings.NewReader("doc_type=cl&theme=")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/pdf", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "cover-letter.pdf")
}

func TestExportPDFBase(t *testing.T) {
	gotenberg := mockGotenbergServer(t)
	srv := newServerWithGotenberg(t, gotenberg.URL)
	store := srv.Store()

	// Write a base CV file
	require.NoError(t, os.WriteFile(filepath.Join(srv.Root(), "cv", "base.html"), []byte("<h1>Gatra</h1>"), 0o644))

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev",
		AppliedDate: "2025-01-01", FolderPath: "acme/dev/2025-01-01",
		BaseCVUsed: "base.html",
	})
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(srv.Root(), "applications", "acme/dev/2025-01-01"), 0o755))

	form := strings.NewReader("doc_type=base&theme=")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/pdf", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "base-cv.pdf")
}

func TestExportPDFInvalidDocType(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	form := strings.NewReader("doc_type=invalid")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/pdf", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExportPDFNoBaseCVRecorded(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	form := strings.NewReader("doc_type=base")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/pdf", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExportPDFListBaseSuffix(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	pdfDir := filepath.Join(srv.Root(), "applications", "acme/dev/1", "pdf")
	require.NoError(t, os.MkdirAll(pdfDir, 0o755))
	// Test all inferred doc_type suffixes
	files := []struct{ name, wantType string }{
		{"gatra-dev-base-cv.pdf", "base"},
		{"gatra-dev-cv.pdf", "cv"},
		{"gatra-dev-cover-letter.pdf", "cl"},
		{"unknown.pdf", ""},
	}
	for _, f := range files {
		require.NoError(t, os.WriteFile(filepath.Join(pdfDir, f.name), []byte("%PDF"), 0o644))
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d/pdfs", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	for _, f := range files {
		assert.Contains(t, w.Body.String(), f.name)
		if f.wantType != "" {
			assert.Contains(t, w.Body.String(), f.wantType)
		}
	}
}
