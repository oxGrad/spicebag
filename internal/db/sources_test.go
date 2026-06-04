package db_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddAndListSources(t *testing.T) {
	store := openTestStore(t)

	// Seeded sources are already present; add custom ones.
	src, err := store.AddSource("AngelList")
	require.NoError(t, err)
	assert.Greater(t, src.ID, int64(0))
	assert.Equal(t, "AngelList", src.Name)

	sources, err := store.ListSources()
	require.NoError(t, err)
	// Seeds + 1 new
	assert.Greater(t, len(sources), 1)

	var found bool
	for _, s := range sources {
		if s.Name == "AngelList" {
			found = true
		}
	}
	assert.True(t, found, "AngelList should be in the list")
}

func TestListSourcesHasSeeds(t *testing.T) {
	store := openTestStore(t)
	sources, err := store.ListSources()
	require.NoError(t, err)
	// Migration seeds 8 sources.
	assert.GreaterOrEqual(t, len(sources), 8)
}

func TestDeleteSource(t *testing.T) {
	store := openTestStore(t)

	src, err := store.AddSource("AngelList")
	require.NoError(t, err)

	require.NoError(t, store.DeleteSource(src.ID))

	sources, err := store.ListSources()
	require.NoError(t, err)
	for _, s := range sources {
		assert.NotEqual(t, "AngelList", s.Name)
	}
}
