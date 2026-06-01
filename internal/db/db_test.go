package db_test

import (
	"path/filepath"
	"testing"

	"github.com/graditya/prospector/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := db.Open(path)
	require.NoError(t, err)
	defer store.Close()
	assert.NotNil(t, store)
}

func TestOpenRunsMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := db.Open(path)
	require.NoError(t, err)
	defer store.Close()

	tables := []string{"experience", "applications", "application_status_history"}
	for _, table := range tables {
		var name string
		err := store.DB().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		require.NoError(t, err, "table %s missing", table)
		assert.Equal(t, table, name)
	}
}

func TestExperience(t *testing.T) {
	store := openTestStore(t)

	entries := []db.ExperienceEntry{
		{RoleType: "backend", Company: "Acme", StartDate: "2018-01-01", EndDate: "2020-06-01", SyncedFrom: "cv-backend.md"},
		{RoleType: "devops", Company: "FooCo", StartDate: "2022-03-01", EndDate: "", SyncedFrom: "cv-backend.md"},
	}

	err := store.UpsertExperience(entries)
	require.NoError(t, err)

	loaded, err := store.ListExperience()
	require.NoError(t, err)
	require.Len(t, loaded, 2)
	assert.Equal(t, "backend", loaded[0].RoleType)
	assert.Equal(t, "Acme", loaded[0].Company)

	err = store.DeleteExperienceBySyncedFrom("cv-backend.md")
	require.NoError(t, err)

	loaded, err = store.ListExperience()
	require.NoError(t, err)
	assert.Len(t, loaded, 0)
}

// helper used by all db tests
func openTestStore(t *testing.T) *db.Store {
	t.Helper()
	store, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestApplications(t *testing.T) {
	store := openTestStore(t)

	app := db.Application{
		Company:     "Stripe",
		Role:        "Backend Engineer",
		AppliedDate: "2025-05-20",
		BaseCVUsed:  "cv-backend-2025-01.md",
		FolderPath:  "stripe/backend-engineer/2025-05-20",
	}

	id, err := store.UpsertApplication(app)
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	apps, err := store.ListApplications()
	require.NoError(t, err)
	require.Len(t, apps, 1)
	assert.Equal(t, "Stripe", apps[0].Company)

	err = store.AddStatusHistory(id, "applied", "")
	require.NoError(t, err)
	err = store.AddStatusHistory(id, "interview", "phone screen")
	require.NoError(t, err)

	history, err := store.GetStatusHistory(id)
	require.NoError(t, err)
	require.Len(t, history, 2)
	assert.Equal(t, "applied", history[0].Status)
	assert.Equal(t, "interview", history[1].Status)
}

func TestExperienceStats(t *testing.T) {
	store := openTestStore(t)
	entries := []db.ExperienceEntry{
		{RoleType: "backend", Company: "A", StartDate: "2018-01-01", EndDate: "2020-01-01", SyncedFrom: "cv.md"},
		{RoleType: "backend", Company: "B", StartDate: "2022-01-01", EndDate: "2023-01-01", SyncedFrom: "cv.md"},
		{RoleType: "devops", Company: "C", StartDate: "2020-01-01", EndDate: "2022-01-01", SyncedFrom: "cv.md"},
	}
	require.NoError(t, store.UpsertExperience(entries))

	stats, err := store.GetExperienceStats()
	require.NoError(t, err)
	assert.InDelta(t, 3.0, stats.ByRole["backend"], 0.1)
	assert.InDelta(t, 2.0, stats.ByRole["devops"], 0.1)
	assert.InDelta(t, 5.0, stats.Total, 0.1)
}
