package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Cover letters ────────────────────────────────────────────────────────────

func TestListCoverLettersTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "list_cover_letters", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, result, "cl-general-2025-01-01.html")
}

func TestReadCoverLetterTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "read_cover_letter", map[string]any{
		"filename": "cl-general-2025-01-01.html",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Dear Hiring Manager")
}

func TestWriteCoverLetterTool(t *testing.T) {
	_, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "write_cover_letter", map[string]any{
		"filename": "cl-acme-2025-01-01.html",
		"content":  "<p>To Acme Corp</p>",
	})
	require.NoError(t, err)

	result, err := srv.CallTool(context.Background(), "read_cover_letter", map[string]any{
		"filename": "cl-acme-2025-01-01.html",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "To Acme Corp")
}

// ─── Experience ───────────────────────────────────────────────────────────────

func TestAddExperienceTool(t *testing.T) {
	_, srv := setup(t)

	entries := `[{"role_type":"DevOps","company":"Acme","start_date":"2022-01-01","end_date":"2024-06-01"},{"role_type":"Software Engineering","company":"Beta","start_date":"2024-07-01","end_date":""}]`
	result, err := srv.CallTool(context.Background(), "add_experience", map[string]any{
		"synced_from": "base.html",
		"entries":     entries,
	})
	require.NoError(t, err)
	assert.Contains(t, result, "2")

	stats, err := srv.CallTool(context.Background(), "get_experience_stats", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, stats, "DevOps")
}

func TestAddExperienceReplacesExisting(t *testing.T) {
	_, srv := setup(t)

	first := `[{"role_type":"DevOps","company":"OldCo","start_date":"2020-01-01","end_date":"2022-01-01"}]`
	_, err := srv.CallTool(context.Background(), "add_experience", map[string]any{
		"synced_from": "base.html",
		"entries":     first,
	})
	require.NoError(t, err)

	second := `[{"role_type":"Software Engineering","company":"NewCo","start_date":"2022-02-01","end_date":""}]`
	_, err = srv.CallTool(context.Background(), "add_experience", map[string]any{
		"synced_from": "base.html",
		"entries":     second,
	})
	require.NoError(t, err)

	stats, err := srv.CallTool(context.Background(), "get_experience_stats", map[string]any{})
	require.NoError(t, err)
	// OldCo (DevOps) was replaced; only NewCo (Software Engineering) remains
	assert.NotContains(t, stats, "DevOps")
	assert.Contains(t, stats, "Software Engineering")
}

// ─── Questions ────────────────────────────────────────────────────────────────

func TestListApplicationQuestionsTool(t *testing.T) {
	_, srv := setup(t)

	id, err := srv.Store().UpsertApplication(db.Application{
		Company: "Stripe", Role: "Eng", AppliedDate: "2025-01-01", FolderPath: "stripe/eng/1",
	})
	require.NoError(t, err)
	_, err = srv.Store().AddQuestion(id, "Tell me about yourself", 0)
	require.NoError(t, err)

	result, err := srv.CallTool(context.Background(), "list_application_questions", map[string]any{
		"application_id": float64(id),
	})
	require.NoError(t, err)
	assert.Contains(t, result, "Tell me about yourself")
}

func TestWriteApplicationAnswersTool(t *testing.T) {
	_, srv := setup(t)

	id, err := srv.Store().UpsertApplication(db.Application{
		Company: "Stripe", Role: "Eng", AppliedDate: "2025-01-01", FolderPath: "stripe/eng/1",
	})
	require.NoError(t, err)
	q, err := srv.Store().AddQuestion(id, "Why this role?", 0)
	require.NoError(t, err)

	answersJSON, _ := json.Marshal([]map[string]any{
		{"question_id": q.ID, "answer": "Because I love the mission"},
	})
	result, err := srv.CallTool(context.Background(), "write_application_answers", map[string]any{
		"application_id": float64(id),
		"answers":        string(answersJSON),
	})
	require.NoError(t, err)
	assert.Contains(t, result, fmt.Sprintf("Saved 1 answers for application %d", id))
}
