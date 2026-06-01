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

	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.md", "# Backend CV"))
	require.NoError(t, fs.WriteCV(root, "cv-devops-2025-01-01.md", "# DevOps CV"))

	files, err := fs.ListCVs(root)
	require.NoError(t, err)
	require.Len(t, files, 2)
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
	require.NoError(t, os.MkdirAll(themeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "minimal.css"), []byte("body{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "modern.css"), []byte("body{}"), 0o644))

	themes, err := fs.ListThemes(root)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"minimal", "modern"}, themes)
}
