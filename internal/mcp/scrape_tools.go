// internal/mcp/scrape_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/scrape"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerScrapeTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"get_scrape_preferences",
			mcplib.WithDescription("Return the user's saved job-scraping companies, target roles, and location preferences (home timezone + notes) used to judge job fit."),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			companies, err := s.store.ListScrapeCompanies()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			roles, err := s.store.ListScrapeRoles()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			prefs, err := s.store.GetScrapePrefs()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			if companies == nil {
				companies = []db.ScrapeCompany{}
			}
			if roles == nil {
				roles = []db.ScrapeRole{}
			}
			payload := map[string]any{
				"companies":      companies,
				"roles":          roles,
				"home_timezone":  prefs.HomeTimezone,
				"location_notes": prefs.LocationNotes,
			}
			b, _ := json.Marshal(payload)
			return mcplib.NewToolResultText(string(b)), nil
		},
	)

	s.registerFetchATSJobs()
	s.registerSaveScrapedJobs()
}

func (s *Server) registerFetchATSJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"fetch_ats_jobs",
			mcplib.WithDescription("Fetch current job listings from all registered ATS companies. Returns a compact list of {company_id, company, title, location, url} plus per-company errors. Records each company's scrape status. Apply timezone/region/role judgment to the returned list, then call save_scraped_jobs with the matches."),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			companies, err := s.store.ListScrapeCompanies()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			reg := scrape.Registry()

			type outJob struct {
				CompanyID int64  `json:"company_id"`
				Company   string `json:"company"`
				Title     string `json:"title"`
				Location  string `json:"location"`
				URL       string `json:"url"`
			}
			type outErr struct {
				Company string `json:"company"`
				Error   string `json:"error"`
			}
			var jobs []outJob
			var errs []outErr

			for i, c := range companies {
				if i > 0 {
					time.Sleep(jitter())
				}
				now := time.Now().Format("2006-01-02 15:04:05")
				var adapter scrape.Adapter
				if c.ATSPlatform == "workday" {
					adapter = scrape.Workday{CareersURL: c.CareersURL}
				} else {
					a, ok := reg[c.ATSPlatform]
					if !ok {
						msg := "Unsupported platform: " + c.ATSPlatform
						s.store.UpdateScrapeCompanyStatus(c.ID, now, "error", msg, 0) //nolint:errcheck
						errs = append(errs, outErr{Company: c.Name, Error: msg})
						continue
					}
					adapter = a
				}
				fetched, ferr := adapter.FetchJobs(ctx, c.ATSToken)
				if ferr != nil {
					msg := scrape.ClassifyError(c.ATSPlatform, ferr)
					s.store.UpdateScrapeCompanyStatus(c.ID, now, "error", msg, 0) //nolint:errcheck
					errs = append(errs, outErr{Company: c.Name, Error: msg})
					continue
				}
				kept := 0
				for _, j := range fetched {
					if !scrape.HasRemoteSignal(j.Location) {
						continue
					}
					jobs = append(jobs, outJob{
						CompanyID: c.ID, Company: c.Name,
						Title: j.Title, Location: j.Location, URL: j.URL,
					})
					kept++
				}
				s.store.UpdateScrapeCompanyStatus(c.ID, now, "ok", "", kept) //nolint:errcheck
			}

			payload := map[string]any{"jobs": jobs, "errors": errs}
			b, _ := json.Marshal(payload)
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}

// jitter returns a small randomized delay between company fetches.
func jitter() time.Duration {
	return time.Duration(300+rand.Intn(700)) * time.Millisecond
}

func (s *Server) registerSaveScrapedJobs() {
	s.mcpSrv.AddTool(
		mcplib.NewTool(
			"save_scraped_jobs",
			mcplib.WithDescription("Save matched jobs (those that pass the user's timezone/region/role rule). Jobs whose URL already exists are ignored. Returns counts of new vs already-seen."),
			mcplib.WithString("jobs", mcplib.Required(),
				mcplib.Description(`JSON array of {"company_id": <id>, "title": "...", "location": "...", "url": "...", "match_reason": "..."}`)),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			jobsJSON := req.GetString("jobs", "")
			var entries []struct {
				CompanyID   int64  `json:"company_id"`
				Title       string `json:"title"`
				Location    string `json:"location"`
				URL         string `json:"url"`
				MatchReason string `json:"match_reason"`
			}
			if err := json.Unmarshal([]byte(jobsJSON), &entries); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("invalid jobs JSON: %v", err)), nil
			}
			var jobs []db.ScrapedJob
			for _, e := range entries {
				jobs = append(jobs, db.ScrapedJob{
					CompanyID:   e.CompanyID,
					CompanyName: s.companyNameByID(e.CompanyID),
					Title:       e.Title,
					Location:    e.Location,
					URL:         e.URL,
					MatchReason: e.MatchReason,
				})
			}
			added, err := s.store.SaveScrapedJobs(jobs)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			b, _ := json.Marshal(map[string]any{"new": added, "already_seen": len(jobs) - added})
			return mcplib.NewToolResultText(string(b)), nil
		},
	)
}

func (s *Server) companyNameByID(id int64) string {
	companies, err := s.store.ListScrapeCompanies()
	if err != nil {
		return ""
	}
	for _, c := range companies {
		if c.ID == id {
			return c.Name
		}
	}
	return ""
}
