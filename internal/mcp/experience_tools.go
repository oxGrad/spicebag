// internal/mcp/experience_tools.go
package mcp

import (
	"context"
	"encoding/json"

	mcplib "github.com/mark3labs/mcp-go/mcp"
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
			out, _ := json.Marshal(stats)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)
}
