package dashboard_test

import (
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

func TestScrapeCompanyAddDetectsPlatform(t *testing.T) {
	srv := newTestServer(t)

	form := strings.NewReader("careers_url=https://boards.greenhouse.io/acme")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "greenhouse")

	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/companies", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "acme")
}

func TestScrapeCompanyAddRejectsUnsupported(t *testing.T) {
	srv := newTestServer(t)
	form := strings.NewReader("careers_url=https://acme.com/careers")
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/companies", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestScrapeRoleAddAndPrefs(t *testing.T) {
	srv := newTestServer(t)

	roleReq := httptest.NewRequest(http.MethodPost, "/api/scrape/roles", strings.NewReader("keyword=SRE"))
	roleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rw := httptest.NewRecorder()
	srv.ServeHTTP(rw, roleReq)
	assert.Equal(t, http.StatusCreated, rw.Code)

	prefReq := httptest.NewRequest(http.MethodPost, "/api/scrape/prefs",
		strings.NewReader("home_timezone=UTC%2B7&location_notes=APAC"))
	prefReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	pw := httptest.NewRecorder()
	srv.ServeHTTP(pw, prefReq)
	assert.Equal(t, http.StatusNoContent, pw.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/scrape/prefs", nil)
	gw := httptest.NewRecorder()
	srv.ServeHTTP(gw, getReq)
	assert.Contains(t, gw.Body.String(), "UTC+7")
}

func TestScrapeSkillAddListDelete(t *testing.T) {
	srv := newTestServer(t)

	// Add
	req := httptest.NewRequest(http.MethodPost, "/api/scrape/skills",
		strings.NewReader("keyword=Go"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Go")

	// List
	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/skills", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), "Go")

	// Extract ID and delete
	var skills []struct{ ID int64 `json:"id"` }
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &skills))
	require.Len(t, skills, 1)

	req3 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/scrape/skills/%d", skills[0].ID), nil)
	w3 := httptest.NewRecorder()
	srv.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusNoContent, w3.Code)
}

func TestBoardToggleAndList(t *testing.T) {
	srv := newTestServer(t)

	// list returns all 4 seeded boards
	req := httptest.NewRequest(http.MethodGet, "/api/scrape/boards", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	var boards []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &boards))
	require.Len(t, boards, 4)

	// find remotive id
	var remotiveID float64
	for _, b := range boards {
		if b["name"] == "remotive" {
			remotiveID = b["id"].(float64)
			assert.Equal(t, true, b["enabled"])
		}
	}
	require.NotZero(t, remotiveID)

	// toggle off
	form := strings.NewReader("enabled=0")
	toggleReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/scrape/boards/%d/toggle", int(remotiveID)), form)
	toggleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tw := httptest.NewRecorder()
	srv.ServeHTTP(tw, toggleReq)
	assert.Equal(t, http.StatusNoContent, tw.Code)

	// verify disabled
	req2 := httptest.NewRequest(http.MethodGet, "/api/scrape/boards", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	var boards2 []map[string]any
	json.Unmarshal(w2.Body.Bytes(), &boards2)
	for _, b := range boards2 {
		if b["name"] == "remotive" {
			assert.Equal(t, false, b["enabled"])
		}
	}
}

func TestBoardJobsListAndStatus(t *testing.T) {
	srv := newTestServer(t)
	store := srv.Store()

	store.SaveBoardJobs([]db.BoardJob{
		{SourceBoard: "remotive", CompanyName: "Acme", Title: "SRE", Location: "Worldwide", URL: "https://remotive.com/1", MatchReason: "worldwide"},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/board-jobs?status=new", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "SRE")

	var jobs []map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jobs))
	id := int(jobs[0]["id"].(float64))

	form := strings.NewReader("status=dismissed")
	statusReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/board-jobs/%d/status", id), form)
	statusReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	sw := httptest.NewRecorder()
	srv.ServeHTTP(sw, statusReq)
	assert.Equal(t, http.StatusNoContent, sw.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/api/board-jobs?status=new", nil)
	w2 := httptest.NewRecorder()
	srv.ServeHTTP(w2, req2)
	assert.Equal(t, "[]", strings.TrimSpace(w2.Body.String()))
}
