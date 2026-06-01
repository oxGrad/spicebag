// internal/mcp/theme_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/graditya/prospector/internal/fs"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerThemeTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("list_themes", mcplib.WithDescription("List available CSS theme names")),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			themes, err := fs.ListThemes(s.root)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(themes)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)
}
