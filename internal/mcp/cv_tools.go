// internal/mcp/cv_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/graditya/prospector/internal/fs"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerCVTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("list_cvs", mcplib.WithDescription("List base CV files with name and modified date")),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			files, err := fs.ListCVs(s.root)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(files)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("read_cv",
			mcplib.WithDescription("Read a base CV's markdown content"),
			mcplib.WithString("filename", mcplib.Required(), mcplib.Description("CV filename to read")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filename := req.GetString("filename", "")
			content, err := fs.ReadCV(s.root, filename)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(content), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("write_cv",
			mcplib.WithDescription("Save a new base CV file"),
			mcplib.WithString("filename", mcplib.Required(), mcplib.Description("Filename e.g. cv-backend-2025-06-01.md")),
			mcplib.WithString("content", mcplib.Required(), mcplib.Description("Full markdown content including frontmatter")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filename := req.GetString("filename", "")
			content := req.GetString("content", "")
			if err := fs.WriteCV(s.root, filename, content); err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Saved %s", filename)), nil
		},
	)
}
