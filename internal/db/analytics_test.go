package db_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationsPerMonthEmpty(t *testing.T) {
	store := openTestStore(t)
	months, err := store.ApplicationsPerMonth("", "")
	require.NoError(t, err)
	assert.Empty(t, months)
}

func TestApplicationsPerMonth(t *testing.T) {
	store := openTestStore(t)

	for _, app := range []db.Application{
		{Company: "A", Role: "Eng", AppliedDate: "2025-01-15", FolderPath: "a/eng/2025-01-15"},
		{Company: "B", Role: "Eng", AppliedDate: "2025-01-20", FolderPath: "b/eng/2025-01-20"},
		{Company: "C", Role: "Dev", AppliedDate: "2025-02-10", FolderPath: "c/dev/2025-02-10"},
	} {
		_, err := store.UpsertApplication(app)
		require.NoError(t, err)
	}

	months, err := store.ApplicationsPerMonth("", "")
	require.NoError(t, err)
	require.Len(t, months, 2)
	// Chronological order
	assert.Equal(t, "2025-01", months[0].Month)
	assert.Equal(t, 2, months[0].Count)
	assert.Equal(t, "2025-02", months[1].Month)
	assert.Equal(t, 1, months[1].Count)
}

func TestSourceStatsEmpty(t *testing.T) {
	store := openTestStore(t)
	stats, err := store.SourceStats("", "")
	require.NoError(t, err)
	assert.Empty(t, stats)
}

func TestSourceStats(t *testing.T) {
	store := openTestStore(t)

	id1, err := store.UpsertApplication(db.Application{
		Company: "A", Role: "Eng", AppliedDate: "2025-01-01", FolderPath: "a/eng/1", Source: "LinkedIn",
	})
	require.NoError(t, err)
	require.NoError(t, store.AddStatusHistory(id1, "applied", ""))
	require.NoError(t, store.AddStatusHistory(id1, "interview", ""))

	id2, err := store.UpsertApplication(db.Application{
		Company: "B", Role: "Eng", AppliedDate: "2025-01-02", FolderPath: "b/eng/2", Source: "LinkedIn",
	})
	require.NoError(t, err)
	require.NoError(t, store.AddStatusHistory(id2, "applied", ""))

	id3, err := store.UpsertApplication(db.Application{
		Company: "C", Role: "Dev", AppliedDate: "2025-01-03", FolderPath: "c/dev/3", Source: "Referral",
	})
	require.NoError(t, err)
	require.NoError(t, store.AddStatusHistory(id3, "offer", ""))

	stats, err := store.SourceStats("", "")
	require.NoError(t, err)
	require.Len(t, stats, 2)

	// LinkedIn has most: 2 total, 1 responded (interview)
	var linkedin, referral db.SourceStat
	for _, s := range stats {
		switch s.Source {
		case "LinkedIn":
			linkedin = s
		case "Referral":
			referral = s
		}
	}

	assert.Equal(t, 2, linkedin.Total)
	assert.Equal(t, 1, linkedin.Responded)
	assert.Equal(t, 1, linkedin.Interviewed)
	assert.Equal(t, 0, linkedin.Offered)

	assert.Equal(t, 1, referral.Total)
	assert.Equal(t, 1, referral.Responded)
	assert.Equal(t, 1, referral.Interviewed)
	assert.Equal(t, 1, referral.Offered)
	assert.InDelta(t, 0.0, referral.DropRate, 0.01)
}

func TestUpdateApplicationSource(t *testing.T) {
	store := openTestStore(t)
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	require.NoError(t, store.UpdateApplicationSource(id, "LinkedIn"))

	app, err := store.GetApplicationByID(id)
	require.NoError(t, err)
	assert.Equal(t, "LinkedIn", app.Source)
}
