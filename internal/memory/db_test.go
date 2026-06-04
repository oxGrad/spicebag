// internal/memory/db_test.go
package memory_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *memory.DB {
	t.Helper()
	db, err := memory.Open(filepath.Join(t.TempDir(), "memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestWriteAndSearch(t *testing.T) {
	db := openTestDB(t)

	err := db.Write(&memory.Memory{
		Name:        "feedback-terse",
		Type:        "feedback",
		Description: "User prefers terse responses",
		Body:        "No trailing summaries. Why: redundant. How to apply: end after delivering result.",
	})
	require.NoError(t, err)

	results, err := db.Search("terse responses", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "feedback-terse", results[0].Name)
	assert.Equal(t, "feedback", results[0].Type)
	assert.False(t, results[0].UpdatedAt.IsZero())
}

func TestWriteUpserts(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "u1", Type: "user", Description: "original", Body: "body v1",
	}))
	require.NoError(t, db.Write(&memory.Memory{
		Name: "u1", Type: "user", Description: "updated", Body: "body v2",
	}))

	all, err := db.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "updated", all[0].Description)
	assert.Equal(t, "body v2", all[0].Body)
}

func TestDelete(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "del-me", Type: "reference", Description: "temp", Body: "body",
	}))
	require.NoError(t, db.Delete("del-me"))

	results, err := db.Search("temp", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchNoMatch(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "m1", Type: "user", Description: "kubernetes expert", Body: "8 years k8s",
	}))

	results, err := db.Search("postgresql databases", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSanitizesPunctuation(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "m1", Type: "feedback", Description: "prefers short", Body: "keep it short",
	}))

	results, err := db.Search("what's the shortest way?", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestList(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{Name: "a", Type: "user", Description: "d1", Body: "b1"}))
	require.NoError(t, db.Write(&memory.Memory{Name: "b", Type: "feedback", Description: "d2", Body: "b2"}))

	all, err := db.List()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestGetByName(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "ref-1", Type: "reference", Description: "a reference", Body: "body",
	}))

	m, err := db.GetByName("ref-1")
	require.NoError(t, err)
	assert.Equal(t, "ref-1", m.Name)

	_, err = db.GetByName("does-not-exist")
	assert.Error(t, err)
}

func TestUpdatedAtChangesOnUpsert(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{Name: "x", Type: "user", Description: "d", Body: "v1"}))
	time.Sleep(time.Second)
	require.NoError(t, db.Write(&memory.Memory{Name: "x", Type: "user", Description: "d", Body: "v2"}))

	all, err := db.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.True(t, !all[0].UpdatedAt.Before(all[0].CreatedAt))
}
