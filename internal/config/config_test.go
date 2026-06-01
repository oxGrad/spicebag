package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, err := config.Load(filepath.Join(dir, "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, 8080, cfg.DashboardPort)
	assert.Equal(t, "http://localhost:3000", cfg.GotenbergURL)
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := os.WriteFile(path, []byte("dashboard_port = 9090\ngotenberg_url = \"http://gotenberg:3000\"\n"), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, 9090, cfg.DashboardPort)
	assert.Equal(t, "http://gotenberg:3000", cfg.GotenbergURL)
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	cfg := config.Config{DashboardPort: 7070, GotenbergURL: "http://x:3000"}

	err := config.Save(path, cfg)
	require.NoError(t, err)

	loaded, err := config.Load(path)
	require.NoError(t, err)
	assert.Equal(t, cfg, loaded)
}
