// internal/mcp/mcp_test.go
package mcp_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
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
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1>"))
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.html", "<p>Dear Hiring Manager</p>"))
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
	assert.Contains(t, result, "cv-backend-2025-01-01.html")
}

func TestReadCVTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "read_cv", map[string]any{"filename": "cv-backend-2025-01-01.html"})
	require.NoError(t, err)
	assert.Contains(t, result, "Backend CV")
}

func TestWriteCVTool(t *testing.T) {
	root, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "write_cv", map[string]any{
		"filename": "cv-new-2025-06-01.html",
		"content":  "<h1>New CV</h1>",
	})
	require.NoError(t, err)

	content, err := fs.ReadCV(root, "cv-new-2025-06-01.html")
	require.NoError(t, err)
	assert.Equal(t, "<h1>New CV</h1>", content)
}

func TestListThemesTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "list_themes", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, result, "minimal")
}

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

// helper to avoid importing mcp package in every test
var _ = mcp.CallToolRequest{}
var _ = json.Marshal
