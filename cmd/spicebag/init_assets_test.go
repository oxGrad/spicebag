// cmd/spicebag/init_assets_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractEmbedDir(t *testing.T) {
	dst := t.TempDir()
	err := extractEmbedDir(themesFS, "themes", dst)
	require.NoError(t, err)

	entries, err := os.ReadDir(dst)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	assert.Contains(t, names, "minimal.css")
	assert.Contains(t, names, "modern.css")

	data, err := os.ReadFile(filepath.Join(dst, "minimal.css"))
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestExtractEmbedDirSkipsExisting(t *testing.T) {
	dst := t.TempDir()
	sentinel := filepath.Join(dst, "minimal.css")
	require.NoError(t, os.WriteFile(sentinel, []byte("custom"), 0644))

	err := extractEmbedDir(themesFS, "themes", dst)
	require.NoError(t, err)

	data, err := os.ReadFile(sentinel)
	require.NoError(t, err)
	assert.Equal(t, "custom", string(data), "existing files must not be overwritten")
}
