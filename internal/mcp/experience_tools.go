// internal/mcp/experience_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/oxGrad/spicebag/internal/db"
)

func (s *Server) registerExperienceTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("get_experience_stats",
			mcplib.WithDescription("Return years of experience totals and breakdown by role type"),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			stats, err := s.store.GetExperienceStats()
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			out, err := json.Marshal(stats)
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("marshal stats: %v", err)), nil
			}
			return mcplib.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("add_experience",
			mcplib.WithDescription("Sync experience entries from a CV into the database. Replaces all previous entries for the same synced_from source atomically."),
			mcplib.WithString("synced_from", mcplib.Required(), mcplib.Description("Identifier for the source (e.g. CV filename like 'base.html')")),
			mcplib.WithString("entries", mcplib.Required(), mcplib.Description(`JSON array of experience entries: [{"role_type":"DevOps","company":"Acme","start_date":"2022-01","end_date":"2024-06"},...]  — end_date is empty string for current roles`)),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			syncedFrom := req.GetString("synced_from", "")
			if syncedFrom == "" {
				return mcplib.NewToolResultError("synced_from is required"), nil
			}

			var raw []struct {
				RoleType  string `json:"role_type"`
				Company   string `json:"company"`
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
			}
			if err := json.Unmarshal([]byte(req.GetString("entries", "[]")), &raw); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("invalid entries JSON: %v", err)), nil
			}

			if err := s.store.DeleteExperienceBySyncedFrom(syncedFrom); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("clear existing: %v", err)), nil
			}

			entries := make([]db.ExperienceEntry, len(raw))
			for i, r := range raw {
				entries[i] = db.ExperienceEntry{
					RoleType:   r.RoleType,
					Company:    r.Company,
					StartDate:  r.StartDate,
					EndDate:    r.EndDate,
					SyncedFrom: syncedFrom,
				}
			}
			if err := s.store.UpsertExperience(entries); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("insert entries: %v", err)), nil
			}

			return mcplib.NewToolResultText(fmt.Sprintf("Synced %d experience entries from %q", len(entries), syncedFrom)), nil
		},
	)
}
