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

func TestFetchATSJobsRecordsStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("network test")
	}
	_, srv := setup(t)
	store := srv.Store()

	c, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "does-not-exist-xyz", CareersURL: "u",
	})
	require.NoError(t, err)

	out, err := srv.CallTool(context.Background(), "fetch_ats_jobs", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Jobs   []map[string]any `json:"jobs"`
		Errors []map[string]any `json:"errors"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	list, _ := store.ListScrapeCompanies()
	require.Len(t, list, 1)
	assert.NotEqual(t, "never", list[0].LastScrapeStatus)
	_ = c
}

func TestSaveScrapedJobs(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})

	jobsJSON := fmt.Sprintf(`[
		{"company_id":%d,"title":"SRE","location":"Remote","url":"https://j/1","match_reason":"worldwide"},
		{"company_id":%d,"title":"DevOps","location":"Remote APAC","url":"https://j/2","match_reason":"APAC includes UTC+7"}
	]`, c.ID, c.ID)

	out, err := srv.CallTool(context.Background(), "save_scraped_jobs", map[string]any{"jobs": jobsJSON})
	require.NoError(t, err)
	assert.Contains(t, out, "2")

	jobs, _ := store.ListScrapedJobs("new")
	assert.Len(t, jobs, 2)
}

func TestCreateApplicationLinksScrapedJob(t *testing.T) {
	_, srv := setup(t)
	store := srv.Store()
	c, _ := store.AddScrapeCompany(db.ScrapeCompany{Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u"})
	store.SaveScrapedJobs([]db.ScrapedJob{
		{CompanyID: c.ID, CompanyName: "Acme", Title: "SRE", URL: "https://boards.greenhouse.io/acme/jobs/42"},
	})

	_, err := srv.CallTool(context.Background(), "create_application", map[string]any{
		"company":              "Acme",
		"role":                 "SRE",
		"date":                 "2026-06-08",
		"cv_content":           "<h1>CV</h1>",
		"cover_letter_content": "<p>CL</p>",
		"job_post_content":     "JD",
		"job_url":              "https://boards.greenhouse.io/acme/jobs/42?utm_source=x",
	})
	require.NoError(t, err)

	newJobs, _ := store.ListScrapedJobs("new")
	assert.Len(t, newJobs, 0)
	applied, _ := store.ListScrapedJobs("applied")
	require.Len(t, applied, 1)
	assert.True(t, applied[0].AppliedApplicationID.Valid)
}

func TestGetScrapePreferences(t *testing.T) {
	_, srv := setup(t) // helper in internal/mcp/mcp_test.go: returns (root, *Server)
	store := srv.Store()

	_, err := store.AddScrapeCompany(db.ScrapeCompany{
		Name: "Acme", ATSPlatform: "greenhouse", ATSToken: "acme", CareersURL: "u",
	})
	require.NoError(t, err)
	_, err = store.AddScrapeRole("SRE")
	require.NoError(t, err)
	require.NoError(t, store.UpdateScrapePrefs(db.ScrapePrefs{HomeTimezone: "UTC+7", LocationNotes: "APAC"}))

	out, err := srv.CallTool(context.Background(), "get_scrape_preferences", map[string]any{})
	require.NoError(t, err)

	var got struct {
		Companies []db.ScrapeCompany `json:"companies"`
		Roles     []db.ScrapeRole    `json:"roles"`
		HomeTZ    string             `json:"home_timezone"`
		Notes     string             `json:"location_notes"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	assert.Len(t, got.Companies, 1)
	assert.Len(t, got.Roles, 1)
	assert.Equal(t, "UTC+7", got.HomeTZ)
	assert.Equal(t, "APAC", got.Notes)
}
