package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/fs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCVs(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1>"))
	require.NoError(t, fs.WriteCV(root, "cv-devops-2025-01-01.html", "<h1>DevOps CV</h1>"))

	files, err := fs.ListCVs(root)
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestReadCV(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1><p>Content here</p>"))

	content, err := fs.ReadCV(root, "cv-backend-2025-01-01.html")
	require.NoError(t, err)
	assert.Equal(t, "<h1>Backend CV</h1><p>Content here</p>", content)
}

func TestWriteCVCreatesDir(t *testing.T) {
	root := t.TempDir()
	err := fs.WriteCV(root, "cv-new-2025-01-01.html", "<p>content</p>")
	require.NoError(t, err)

	content, err := fs.ReadCV(root, "cv-new-2025-01-01.html")
	require.NoError(t, err)
	assert.Equal(t, "<p>content</p>", content)
}

func TestListCoverLetters(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.html", "<p>Dear Hiring Manager</p>"))

	files, err := fs.ListCoverLetters(root)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, "cl-general-2025-01-01.html", files[0].Name)
}

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

	cv, err := os.ReadFile(filepath.Join(root, "applications", folderPath, "cv.html"))
	require.NoError(t, err)
	assert.Equal(t, "# CV", string(cv))

	meta, err := fs.ReadApplicationMetadata(root, folderPath)
	require.NoError(t, err)
	assert.Equal(t, "Stripe", meta.Company)
}

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

func TestListThemes(t *testing.T) {
	root := t.TempDir()
	themeDir := filepath.Join(root, "themes")
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "minimal.css"), []byte("body{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "modern.css"), []byte("body{}"), 0o644))

	themes, err := fs.ListThemes(root)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"minimal", "modern"}, themes)
}
