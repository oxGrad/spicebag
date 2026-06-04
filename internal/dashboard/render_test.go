package dashboard_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── App doc render ───────────────────────────────────────────────────────────

func TestRenderAppCV(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	docDir := filepath.Join(srv.Root(), "applications", "acme/dev/1")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "cv.html"), []byte("<h1>Gatra</h1>"), 0o644))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/render/app/%d/cv", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Gatra")
}

func TestRenderAppCVNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/render/app/9999/cv", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRenderAppCL(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	docDir := filepath.Join(srv.Root(), "applications", "acme/dev/1")
	require.NoError(t, os.MkdirAll(docDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docDir, "cover-letter.html"), []byte("<p>Dear Hiring Manager</p>"), 0o644))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/render/app/%d/cl", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Dear Hiring Manager")
}

func TestRenderAppCLMissingFile(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)
	// No cover-letter.html created
	require.NoError(t, os.MkdirAll(filepath.Join(srv.Root(), "applications", "acme/dev/1"), 0o755))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/render/app/%d/cl", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── PDF file listing ─────────────────────────────────────────────────────────

func TestListPDFsNoPdfDir(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d/pdfs", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "[]")
}

func TestListPDFsWithFiles(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	pdfDir := filepath.Join(srv.Root(), "applications", "acme/dev/1", "pdf")
	require.NoError(t, os.MkdirAll(pdfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pdfDir, "gatra-dev-cv.pdf"), []byte("%PDF"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(pdfDir, "gatra-dev-cover-letter.pdf"), []byte("%PDF"), 0o644))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d/pdfs", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "gatra-dev-cv.pdf")
	assert.Contains(t, w.Body.String(), `"doc_type":"cv"`)
	assert.Contains(t, w.Body.String(), "gatra-dev-cover-letter.pdf")
	assert.Contains(t, w.Body.String(), `"doc_type":"cl"`)
}

func TestListPDFsAppNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/apps/9999/pdfs", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── PDF file serving ─────────────────────────────────────────────────────────

func TestRenderAppPDF(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	pdfDir := filepath.Join(srv.Root(), "applications", "acme/dev/1", "pdf")
	require.NoError(t, os.MkdirAll(pdfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(pdfDir, "gatra-dev-cv.pdf"), []byte("%PDF-1.4 content"), 0o644))

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/render/app/%d/pdf/gatra-dev-cv.pdf", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
}
