// internal/mcp/scrape_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/oxGrad/spicebag/internal/db"
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

// Placeholder stubs replaced by Tasks 10–11.
func (s *Server) registerFetchATSJobs()    {}
func (s *Server) registerSaveScrapedJobs() {}
