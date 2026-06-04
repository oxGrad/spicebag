package db_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeApp(t *testing.T) (*db.Store, int64) {
	t.Helper()
	store := openTestStore(t)
	id, err := store.UpsertApplication(db.Application{
		Company: "TestCo", Role: "Dev",
		AppliedDate: "2025-01-01", FolderPath: "testco/dev/2025-01-01",
	})
	require.NoError(t, err)
	return store, id
}

func TestAddAndListQuestions(t *testing.T) {
	store, appID := makeApp(t)

	q, err := store.AddQuestion(appID, "Tell me about yourself", 0)
	require.NoError(t, err)
	assert.Greater(t, q.ID, int64(0))
	assert.Equal(t, "Tell me about yourself", q.Question)
	assert.Equal(t, appID, q.ApplicationID)
	assert.Empty(t, q.UserBullets)

	q2, err := store.AddQuestion(appID, "Why this role?", 1)
	require.NoError(t, err)

	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	require.Len(t, qs, 2)
	assert.Equal(t, q.ID, qs[0].ID)
	assert.Equal(t, q2.ID, qs[1].ID)
}

func TestListQuestionsEmpty(t *testing.T) {
	store, appID := makeApp(t)
	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	assert.Empty(t, qs)
}

func TestUpdateAnswer(t *testing.T) {
	store, appID := makeApp(t)
	q, err := store.AddQuestion(appID, "Describe a challenge", 0)
	require.NoError(t, err)

	require.NoError(t, store.UpdateAnswer(q.ID, "I once fixed a production bug at 2am"))

	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	assert.Equal(t, "I once fixed a production bug at 2am", qs[0].Answer)
}

func TestUpdateBullets(t *testing.T) {
	store, appID := makeApp(t)
	q, err := store.AddQuestion(appID, "Strengths?", 0)
	require.NoError(t, err)

	bullets := []string{"Fast learner", "Team player", "Strong communicator"}
	require.NoError(t, store.UpdateBullets(q.ID, bullets))

	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	assert.Equal(t, bullets, qs[0].UserBullets)
}

func TestDeleteQuestion(t *testing.T) {
	store, appID := makeApp(t)
	q, err := store.AddQuestion(appID, "To delete", 0)
	require.NoError(t, err)

	require.NoError(t, store.DeleteQuestion(q.ID))

	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	assert.Empty(t, qs)
}

func TestBulkUpdateAnswers(t *testing.T) {
	store, appID := makeApp(t)
	q1, err := store.AddQuestion(appID, "Q1", 0)
	require.NoError(t, err)
	q2, err := store.AddQuestion(appID, "Q2", 1)
	require.NoError(t, err)

	pairs := []struct {
		QuestionID int64
		Answer     string
	}{
		{q1.ID, "Answer one"},
		{q2.ID, "Answer two"},
	}
	require.NoError(t, store.BulkUpdateAnswers(pairs))

	qs, err := store.ListQuestions(appID)
	require.NoError(t, err)
	require.Len(t, qs, 2)
	assert.Equal(t, "Answer one", qs[0].Answer)
	assert.Equal(t, "Answer two", qs[1].Answer)
}
