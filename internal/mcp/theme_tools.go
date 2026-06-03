// internal/mcp/theme_tools.go
package mcp

import (
	"context"
	"encoding/json"

	"github.com/oxGrad/spicebag/internal/fs"
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
			if themes == nil {
				themes = []string{}
			}
			out, _ := json.Marshal(themes)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)
}
