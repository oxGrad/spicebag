// handlers_test.go covers questions, sources, analytics, and app-source handlers.
package dashboard_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Questions ────────────────────────────────────────────────────────────────

func TestQuestionsListEmpty(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/apps/%d/questions", id), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "[]", strings.TrimSpace(w.Body.String()))
}

func TestQuestionsAdd(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	form := strings.NewReader("question=Tell+me+about+yourself&position=0")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/questions", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Tell me about yourself")
}

func TestQuestionsAddMissingQuestion(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, _ := store.UpsertApplication(db.Application{
		Company: "X", Role: "Y", AppliedDate: "2025-01-01", FolderPath: "x/y/1",
	})

	form := strings.NewReader("question=")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/questions", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQuestionUpdateAnswer(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)
	q, err := store.AddQuestion(id, "Why us?", 0)
	require.NoError(t, err)

	form := strings.NewReader("answer=Because+I+love+the+mission")
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/apps/%d/questions/%d", id, q.ID), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestQuestionUpdateBullets(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)
	q, err := store.AddQuestion(id, "Strengths?", 0)
	require.NoError(t, err)

	body, _ := json.Marshal([]string{"bullet a", "bullet b"})
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/apps/%d/questions/%d/bullets", id, q.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestQuestionUpdateBulletsInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, _ := store.UpsertApplication(db.Application{
		Company: "X", Role: "Y", AppliedDate: "2025-01-01", FolderPath: "x/y/1",
	})
	q, _ := store.AddQuestion(id, "Q?", 0)

	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/apps/%d/questions/%d/bullets", id, q.ID), strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQuestionDelete(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)
	q, err := store.AddQuestion(id, "To delete", 0)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/apps/%d/questions/%d", id, q.ID), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestQuestionsBulkAnswer(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)
	q1, _ := store.AddQuestion(id, "Q1", 0)
	q2, _ := store.AddQuestion(id, "Q2", 1)

	entries := []map[string]any{
		{"question_id": q1.ID, "answer": "Ans1"},
		{"question_id": q2.ID, "answer": "Ans2"},
	}
	body, _ := json.Marshal(entries)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/questions/answers", id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestQuestionsBulkAnswerInvalidJSON(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, _ := store.UpsertApplication(db.Application{
		Company: "X", Role: "Y", AppliedDate: "2025-01-01", FolderPath: "x/y/1",
	})

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/questions/answers", id), strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Sources ──────────────────────────────────────────────────────────────────

func TestSourcesList(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/sources", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "LinkedIn")
}

func TestSourcesCreate(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("name=AngelList")
	req := httptest.NewRequest(http.MethodPost, "/api/sources", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "AngelList")
}

func TestSourcesCreateMissingName(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("name=")
	req := httptest.NewRequest(http.MethodPost, "/api/sources", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSourcesDelete(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	src, err := store.AddSource("AngelList")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/sources/%d", src.ID), nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestSourcesDeleteInvalidID(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodDelete, "/api/sources/not-a-number", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Analytics ────────────────────────────────────────────────────────────────

func TestAnalyticsEmpty(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/analytics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "per_period")
	assert.Contains(t, w.Body.String(), "source_stats")
}

func TestAnalyticsWithData(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	id, err := store.UpsertApplication(db.Application{
		Company: "Stripe", Role: "Eng", AppliedDate: "2025-01-10", FolderPath: "stripe/eng/1", Source: "LinkedIn",
	})
	require.NoError(t, err)
	require.NoError(t, store.AddStatusHistory(id, "applied", ""))

	req := httptest.NewRequest(http.MethodGet, "/api/analytics", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "2025-01")
	assert.Contains(t, w.Body.String(), "LinkedIn")
}

// ─── App source update ────────────────────────────────────────────────────────

func TestAppSourceUpdate(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()
	id, err := store.UpsertApplication(db.Application{
		Company: "Acme", Role: "Dev", AppliedDate: "2025-01-01", FolderPath: "acme/dev/1",
	})
	require.NoError(t, err)

	form := strings.NewReader("source=Greenhouse")
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/apps/%d/source", id), form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

// ─── CV usages ────────────────────────────────────────────────────────────────

func TestCVUsages(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	_, err := store.UpsertApplication(db.Application{
		Company: "Stripe", Role: "Eng", AppliedDate: "2025-01-01",
		FolderPath: "stripe/eng/1", BaseCVUsed: "base.html",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/cv/base.html/usages", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Stripe")
}
