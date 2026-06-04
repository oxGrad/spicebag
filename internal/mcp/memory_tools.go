// internal/mcp/memory_tools.go
package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oxGrad/spicebag/internal/memory"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

func (s *Server) registerMemoryTools() {
	s.mcpSrv.AddTool(
		mcplib.NewTool("memory_write",
			mcplib.WithDescription("Write or update a Claude Code memory entry in memory.db"),
			mcplib.WithString("name", mcplib.Required(), mcplib.Description("Kebab-case slug, e.g. feedback-terse")),
			mcplib.WithString("type", mcplib.Required(), mcplib.Description("One of: user, feedback, project, reference")),
			mcplib.WithString("description", mcplib.Required(), mcplib.Description("One-line summary used for relevance matching")),
			mcplib.WithString("body", mcplib.Required(), mcplib.Description("Full memory content")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			m := &memory.Memory{
				Name:        req.GetString("name", ""),
				Type:        req.GetString("type", ""),
				Description: req.GetString("description", ""),
				Body:        req.GetString("body", ""),
			}
			if m.Name == "" || m.Type == "" || m.Description == "" || m.Body == "" {
				return mcplib.NewToolResultError("name, type, description, and body are all required"), nil
			}
			if err := s.mem.Write(m); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("write memory: %v", err)), nil
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Saved memory %q", m.Name)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("memory_search",
			mcplib.WithDescription("Search Claude Code memories by keyword. Returns up to 5 relevant entries."),
			mcplib.WithString("query", mcplib.Required(), mcplib.Description("Keywords to search for")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			query := req.GetString("query", "")
			if query == "" {
				return mcplib.NewToolResultError("query is required"), nil
			}
			results, err := s.mem.Search(query, 5)
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("search: %v", err)), nil
			}
			if len(results) == 0 {
				return mcplib.NewToolResultText("No memories matched."), nil
			}
			out, _ := json.Marshal(results)
			return mcplib.NewToolResultText(string(out)), nil
		},
	)

	s.mcpSrv.AddTool(
		mcplib.NewTool("memory_delete",
			mcplib.WithDescription("Delete a Claude Code memory entry by name"),
			mcplib.WithString("name", mcplib.Required(), mcplib.Description("Memory slug to delete")),
		),
		func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
			name := req.GetString("name", "")
			if name == "" {
				return mcplib.NewToolResultError("name is required"), nil
			}
			if err := s.mem.Delete(name); err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("delete: %v", err)), nil
			}
			return mcplib.NewToolResultText(fmt.Sprintf("Deleted memory %q", name)), nil
		},
	)
}
