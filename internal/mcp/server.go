// internal/mcp/server.go
package mcp

import (
	"context"
	"fmt"

	"github.com/graditya/prospector/internal/db"
	"github.com/mark3labs/mcp-go/client"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	root   string
	store  *db.Store
	gotURL string
	mcpSrv *server.MCPServer
}

func NewServer(root, dbPath, gotenbergURL string) (*Server, error) {
	store, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	s := &Server{
		root:   root,
		store:  store,
		gotURL: gotenbergURL,
		mcpSrv: server.NewMCPServer("spicebag", "1.0.0"),
	}

	s.registerCVTools()
	s.registerCoverLetterTools()
	s.registerThemeTools()
	s.registerPDFTools()
	s.registerExperienceTools()
	s.registerApplicationTools()

	return s, nil
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpSrv)
}

func (s *Server) Close() { s.store.Close() }

// Store returns the underlying DB store. Used in tests for seeding data
// without opening a second connection (SQLite allows only one writer).
func (s *Server) Store() *db.Store { return s.store }

// CallTool is used in tests to invoke a tool directly via the in-process client.
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	c, err := client.NewInProcessClient(s.mcpSrv)
	if err != nil {
		return "", fmt.Errorf("creating in-process client: %w", err)
	}
	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "prospector-test", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, initReq); err != nil {
		return "", fmt.Errorf("initializing client: %w", err)
	}

	req := mcplib.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	result, err := c.CallTool(ctx, req)
	if err != nil {
		return "", err
	}
	if result.IsError {
		return "", fmt.Errorf("tool error: %v", result.Content)
	}
	if len(result.Content) == 0 {
		return "", nil
	}
	text, ok := result.Content[0].(mcplib.TextContent)
	if !ok {
		return "", fmt.Errorf("unexpected content type")
	}
	return text.Text, nil
}
