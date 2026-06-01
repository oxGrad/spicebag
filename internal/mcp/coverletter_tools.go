// internal/mcp/coverletter_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/graditya/prospector/internal/fs"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerCoverLetterTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("list_cover_letters", mcplib.WithDescription("List standalone cover letter files")),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			files, err := fs.ListCoverLetters(s.root)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			out, _ := json.Marshal(files)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("read_cover_letter",
			mcplib.WithDescription("Read a cover letter's markdown content"),
			mcplib.WithString("filename", mcplib.Required(), mcplib.Description("Cover letter filename to read")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filename := req.GetString("filename", "")
			content, err := fs.ReadCoverLetter(s.root, filename)
			if err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(content), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("write_cover_letter",
			mcplib.WithDescription("Save a new cover letter file"),
			mcplib.WithString("filename", mcplib.Required(), mcplib.Description("Filename e.g. cl-stripe-2025-06-01.md")),
			mcplib.WithString("content", mcplib.Required(), mcplib.Description("Full markdown content")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			filename := req.GetString("filename", "")
			content := req.GetString("content", "")
			if err := fs.WriteCoverLetter(s.root, filename, content); err != nil {
				return mcplib.NewToolResultError(err.Error()), nil
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Saved %s", filename)), nil
		},
	)
}
