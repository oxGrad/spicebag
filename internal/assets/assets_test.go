package assets_test

import (
	"io/fs"
	"testing"

	"github.com/graditya/prospector/internal/assets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThemesEmbedded(t *testing.T) {
	for _, name := range []string{"minimal.css", "modern.css"} {
		data, err := fs.ReadFile(assets.ThemesFS, "themes/"+name)
		require.NoError(t, err, "theme %s must be embedded", name)
		assert.NotEmpty(t, data, "theme %s must not be empty", name)
	}
}

func TestSkillsEmbedded(t *testing.T) {
	for _, name := range []string{"customize-cv.md", "write-cover-letter.md", "apply.md"} {
		data, err := fs.ReadFile(assets.SkillsFS, "skills/"+name)
		require.NoError(t, err, "skill %s must be embedded", name)
		assert.Contains(t, string(data), "name:", "skill %s must have frontmatter", name)
	}
}
