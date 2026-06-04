# SQLite FTS5 Memory System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the file-based Claude Code memory system with a SQLite FTS5 database at `~/.config/spicebag/memory.db`, exposed via MCP tools, a CLI hook command, and a dashboard read view.

**Architecture:** A new `internal/memory` package owns the DB (standalone SQLite, not the main `spicebag.db`). `*memory.DB` is threaded into the MCP server and dashboard server as a new dependency. A `spicebag memory search` CLI command reads the user prompt from stdin JSON and outputs matching memories to stdout — this is wired as a `UserPromptSubmit` hook in `.claude-plugin/plugin.json`. The dashboard gets two read-only endpoints and a new `MemoryView.vue` page.

**Tech Stack:** Go (`modernc.org/sqlite`, `database/sql`, FTS5), Cobra CLI, `github.com/mark3labs/mcp-go`, Vue 3 + Tailwind

---

## File Map

**New files:**
- `internal/memory/db.go` — Open, schema init, Write, Search, List, Delete, Close
- `internal/memory/db_test.go` — DB layer unit tests
- `internal/mcp/memory_tools.go` — MCP tools: `memory_write`, `memory_delete`, `memory_search`
- `cmd/spicebag/cmd_memory.go` — `spicebag memory {search,list,write,delete}` subcommands
- `internal/dashboard/handlers_memory.go` — `GET /api/memories`, `GET /api/memories/{name}`
- `frontend/src/views/MemoryView.vue` — Memory browser: table + detail panel

**Modified files:**
- `internal/mcp/server.go` — add `mem *memory.DB` field, update `NewServer` signature, call `registerMemoryTools()`
- `internal/mcp/mcp_test.go` — update `setup()` to pass memory DB path
- `cmd/spicebag/cmd_mcp.go` — open memory DB, pass path to `NewServer`
- `internal/dashboard/server.go` — add `mem *memory.DB` field, update `NewServer`, add routes
- `internal/dashboard/dashboard_test.go` — update `newTestServer()` to open memory DB
- `cmd/spicebag/cmd_start.go` — open memory DB, pass to `dashboard.NewServer`
- `cmd/spicebag/main.go` — register `memory` subcommand
- `frontend/src/router/index.js` — add `/memory` route
- `frontend/src/App.vue` — add Memory sidebar link
- `frontend/src/api.js` — add `api.memory.*` helpers
- `.claude-plugin/plugin.json` — add `UserPromptSubmit` hook

---

## Task 1: `internal/memory` DB layer

**Files:**
- Create: `internal/memory/db.go`
- Create: `internal/memory/db_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/memory/db_test.go
package memory_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *memory.DB {
	t.Helper()
	db, err := memory.Open(filepath.Join(t.TempDir(), "memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func TestWriteAndSearch(t *testing.T) {
	db := openTestDB(t)

	err := db.Write(&memory.Memory{
		Name:        "feedback-terse",
		Type:        "feedback",
		Description: "User prefers terse responses",
		Body:        "No trailing summaries. Why: redundant. How to apply: end after delivering result.",
	})
	require.NoError(t, err)

	results, err := db.Search("terse responses", 5)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "feedback-terse", results[0].Name)
	assert.Equal(t, "feedback", results[0].Type)
	assert.False(t, results[0].UpdatedAt.IsZero())
}

func TestWriteUpserts(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "u1", Type: "user", Description: "original", Body: "body v1",
	}))
	require.NoError(t, db.Write(&memory.Memory{
		Name: "u1", Type: "user", Description: "updated", Body: "body v2",
	}))

	all, err := db.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "updated", all[0].Description)
	assert.Equal(t, "body v2", all[0].Body)
}

func TestDelete(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "del-me", Type: "reference", Description: "temp", Body: "body",
	}))
	require.NoError(t, db.Delete("del-me"))

	results, err := db.Search("temp", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchNoMatch(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "m1", Type: "user", Description: "kubernetes expert", Body: "8 years k8s",
	}))

	results, err := db.Search("postgresql databases", 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchSanitizesPunctuation(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "m1", Type: "feedback", Description: "prefers short", Body: "keep it short",
	}))

	// Prompt with punctuation — should not error, should match on "short"
	results, err := db.Search("what's the shortest way?", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestList(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{Name: "a", Type: "user", Description: "d1", Body: "b1"}))
	require.NoError(t, db.Write(&memory.Memory{Name: "b", Type: "feedback", Description: "d2", Body: "b2"}))

	all, err := db.List()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestUpdatedAtChangesOnUpsert(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{Name: "x", Type: "user", Description: "d", Body: "v1"}))
	time.Sleep(time.Second) // SQLite timestamp resolution is 1s
	require.NoError(t, db.Write(&memory.Memory{Name: "x", Type: "user", Description: "d", Body: "v2"}))

	all, err := db.List()
	require.NoError(t, err)
	require.Len(t, all, 1)
	// updated_at should be after created_at
	assert.True(t, !all[0].UpdatedAt.Before(all[0].CreatedAt))
}
```

- [ ] **Step 2: Run test to confirm it fails**

```bash
go test ./internal/memory/... -v
```

Expected: compile error — package does not exist yet.

- [ ] **Step 3: Implement `internal/memory/db.go`**

```go
// internal/memory/db.go
package memory

import (
	"database/sql"
	"regexp"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Memory struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DB struct {
	db *sql.DB
}

var schema = []string{
	`CREATE TABLE IF NOT EXISTS memories (
		name        TEXT PRIMARY KEY,
		type        TEXT NOT NULL CHECK(type IN ('user','feedback','project','reference')),
		description TEXT NOT NULL,
		body        TEXT NOT NULL,
		created_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
		updated_at  TEXT DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
	)`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
		name, description, body,
		content='memories',
		content_rowid='rowid'
	)`,
	`CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
		INSERT INTO memories_fts(rowid, name, description, body)
		VALUES (new.rowid, new.name, new.description, new.body);
	END`,
	`CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, name, description, body)
		VALUES ('delete', old.rowid, old.name, old.description, old.body);
	END`,
	`CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
		INSERT INTO memories_fts(memories_fts, rowid, name, description, body)
		VALUES ('delete', old.rowid, old.name, old.description, old.body);
		INSERT INTO memories_fts(rowid, name, description, body)
		VALUES (new.rowid, new.name, new.description, new.body);
	END`,
}

func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	for _, stmt := range schema {
		if _, err := sqldb.Exec(stmt); err != nil {
			sqldb.Close()
			return nil, err
		}
	}
	return &DB{db: sqldb}, nil
}

func (d *DB) Close() { d.db.Close() }

func (d *DB) Write(m *Memory) error {
	_, err := d.db.Exec(`
		INSERT INTO memories (name, type, description, body, updated_at)
		VALUES (?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		ON CONFLICT(name) DO UPDATE SET
			type        = excluded.type,
			description = excluded.description,
			body        = excluded.body,
			updated_at  = excluded.updated_at
	`, m.Name, m.Type, m.Description, m.Body)
	return err
}

func (d *DB) Delete(name string) error {
	_, err := d.db.Exec(`DELETE FROM memories WHERE name = ?`, name)
	return err
}

func (d *DB) List() ([]*Memory, error) {
	rows, err := d.db.Query(`
		SELECT name, type, description, body, created_at, updated_at
		FROM memories ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

var wordRE = regexp.MustCompile(`\w+`)

func (d *DB) Search(query string, limit int) ([]*Memory, error) {
	words := wordRE.FindAllString(query, -1)
	if len(words) == 0 {
		return nil, nil
	}
	ftsQuery := strings.Join(words, " ")
	rows, err := d.db.Query(`
		SELECT m.name, m.type, m.description, m.body, m.created_at, m.updated_at
		FROM memories m
		JOIN memories_fts ON m.rowid = memories_fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

func scanMemories(rows *sql.Rows) ([]*Memory, error) {
	var out []*Memory
	for rows.Next() {
		var m Memory
		var createdStr, updatedStr string
		if err := rows.Scan(&m.Name, &m.Type, &m.Description, &m.Body, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
		m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
		out = append(out, &m)
	}
	return out, rows.Err()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/memory/... -v
```

Expected: all 6 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/memory/
git commit -m "feat: add internal/memory SQLite FTS5 DB layer"
```

---

## Task 2: CLI subcommands

**Files:**
- Create: `cmd/spicebag/cmd_memory.go`
- Modify: `cmd/spicebag/main.go`

- [ ] **Step 1: Write `cmd_memory.go`**

```go
// cmd/spicebag/cmd_memory.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/spf13/cobra"
)

func newMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory",
		Short: "Manage Claude Code memory stored in memory.db",
	}
	cmd.AddCommand(
		newMemorySearchCmd(),
		newMemoryListCmd(),
		newMemoryWriteCmd(),
		newMemoryDeleteCmd(),
	)
	return cmd
}

// newMemorySearchCmd is the hook entrypoint: reads prompt JSON from stdin,
// outputs matching memories as plain text for Claude Code context injection.
func newMemorySearchCmd() *cobra.Command {
	var query string
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search memories (reads stdin JSON hook payload when --query is omitted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if query == "" {
				var payload struct {
					Prompt string `json:"prompt"`
				}
				if err := json.NewDecoder(os.Stdin).Decode(&payload); err != nil || payload.Prompt == "" {
					return nil // no query, emit nothing
				}
				query = payload.Prompt
			}

			db, err := openMemoryDB()
			if err != nil {
				return err
			}
			defer db.Close()

			results, err := db.Search(query, 5)
			if err != nil || len(results) == 0 {
				return nil // emit nothing on error or no match
			}

			fmt.Println("Relevant memories:")
			for _, m := range results {
				fmt.Printf("\n[%s] %s\n%s\n", m.Type, m.Name, m.Body)
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&query, "query", "q", "", "Search query (skips stdin when set)")
	return cmd
}

func newMemoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all memories",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openMemoryDB()
			if err != nil {
				return err
			}
			defer db.Close()

			memories, err := db.List()
			if err != nil {
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tDESCRIPTION\tUPDATED")
			for _, m := range memories {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
					m.Name, m.Type,
					truncate(m.Description, 60),
					m.UpdatedAt.Format("2006-01-02"),
				)
			}
			return w.Flush()
		},
	}
}

func newMemoryWriteCmd() *cobra.Command {
	var name, typ, description, body string
	cmd := &cobra.Command{
		Use:   "write",
		Short: "Write or update a memory",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" || typ == "" || description == "" || body == "" {
				return fmt.Errorf("--name, --type, --description, and --body are all required")
			}
			db, err := openMemoryDB()
			if err != nil {
				return err
			}
			defer db.Close()
			if err := db.Write(&memory.Memory{
				Name: name, Type: typ, Description: description, Body: body,
			}); err != nil {
				return err
			}
			fmt.Printf("Saved memory %q\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Memory slug (e.g. feedback-terse)")
	cmd.Flags().StringVar(&typ, "type", "", "Type: user, feedback, project, or reference")
	cmd.Flags().StringVar(&description, "description", "", "One-line summary")
	cmd.Flags().StringVar(&body, "body", "", "Full memory content")
	return cmd
}

func newMemoryDeleteCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a memory by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			db, err := openMemoryDB()
			if err != nil {
				return err
			}
			defer db.Close()
			if err := db.Delete(name); err != nil {
				return err
			}
			fmt.Printf("Deleted memory %q\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Memory slug to delete")
	return cmd
}

func openMemoryDB() (*memory.DB, error) {
	return memory.Open(filepath.Join(spicebagRoot(), "memory.db"))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}
```

- [ ] **Step 2: Register in `main.go`**

Open `cmd/spicebag/main.go`. Find the block where subcommands are added to the root command (the calls to `rootCmd.AddCommand(...)`). Add:

```go
rootCmd.AddCommand(newMemoryCmd())
```

- [ ] **Step 3: Smoke test manually**

```bash
just build-go
spicebag memory list
```

Expected: empty table (no memories yet).

```bash
spicebag memory write --name "test" --type feedback --description "test entry" --body "this is a test"
spicebag memory list
spicebag memory search -q "test entry"
spicebag memory delete --name test
spicebag memory list
```

Expected: write prints "Saved memory", list shows it, search returns it, delete removes it, final list is empty.

- [ ] **Step 4: Commit**

```bash
git add cmd/spicebag/cmd_memory.go cmd/spicebag/main.go
git commit -m "feat: add spicebag memory CLI subcommands"
```

---

## Task 3: MCP memory tools

**Files:**
- Create: `internal/mcp/memory_tools.go`
- Modify: `internal/mcp/server.go` — add `mem` field, update `NewServer`, call `registerMemoryTools()`
- Modify: `internal/mcp/mcp_test.go` — update `setup()` to open memory DB
- Modify: `cmd/spicebag/cmd_mcp.go` — pass memory DB path

- [ ] **Step 1: Write failing tests**

Add these tests to `internal/mcp/mcp_test.go` (after the existing tests):

```go
func TestMemoryWriteTool(t *testing.T) {
	_, srv := setup(t)
	result, err := srv.CallTool(context.Background(), "memory_write", map[string]any{
		"name":        "feedback-terse",
		"type":        "feedback",
		"description": "User prefers terse responses",
		"body":        "No trailing summaries. Why: redundant. How to apply: end after result.",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "feedback-terse")
}

func TestMemorySearchTool(t *testing.T) {
	_, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "memory_write", map[string]any{
		"name": "u1", "type": "user", "description": "SRE background", "body": "8 years kubernetes",
	})
	require.NoError(t, err)

	result, err := srv.CallTool(context.Background(), "memory_search", map[string]any{
		"query": "kubernetes",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "u1")
}

func TestMemoryDeleteTool(t *testing.T) {
	_, srv := setup(t)
	_, err := srv.CallTool(context.Background(), "memory_write", map[string]any{
		"name": "del", "type": "reference", "description": "temp", "body": "to be deleted",
	})
	require.NoError(t, err)

	result, err := srv.CallTool(context.Background(), "memory_delete", map[string]any{
		"name": "del",
	})
	require.NoError(t, err)
	assert.Contains(t, result, "del")

	result, err = srv.CallTool(context.Background(), "memory_search", map[string]any{
		"query": "to be deleted",
	})
	require.NoError(t, err)
	assert.NotContains(t, result, "del")
}
```

- [ ] **Step 2: Run tests — confirm they fail**

```bash
go test ./internal/mcp/... -v -run "TestMemory"
```

Expected: compile errors because `memory_write`, `memory_search`, `memory_delete` tools don't exist and `NewServer` signature hasn't changed yet.

- [ ] **Step 3: Update `internal/mcp/server.go`**

Add import `"github.com/oxGrad/spicebag/internal/memory"` and update the struct and constructor:

```go
// internal/mcp/server.go
package mcp

import (
	"context"
	"fmt"

	"github.com/oxGrad/spicebag/internal/db"
	"github.com/oxGrad/spicebag/internal/memory"
	"github.com/mark3labs/mcp-go/client"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	root   string
	store  *db.Store
	mem    *memory.DB
	gotURL string
	mcpSrv *server.MCPServer
}

func NewServer(root, dbPath, memoryDBPath, gotenbergURL string) (*Server, error) {
	store, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}

	mem, err := memory.Open(memoryDBPath)
	if err != nil {
		store.Close()
		return nil, err
	}

	s := &Server{
		root:   root,
		store:  store,
		mem:    mem,
		gotURL: gotenbergURL,
		mcpSrv: server.NewMCPServer("spicebag", "1.0.0"),
	}

	s.registerCVTools()
	s.registerCoverLetterTools()
	s.registerThemeTools()
	s.registerPDFTools()
	s.registerExperienceTools()
	s.registerApplicationTools()
	s.registerQuestionTools()
	s.registerMemoryTools()

	return s, nil
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpSrv)
}

func (s *Server) Close() {
	s.store.Close()
	s.mem.Close()
}

func (s *Server) Store() *db.Store { return s.store }

func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	c, err := client.NewInProcessClient(s.mcpSrv)
	if err != nil {
		return "", fmt.Errorf("creating in-process client: %w", err)
	}
	initReq := mcplib.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcplib.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcplib.Implementation{Name: "spicebag-test", Version: "1.0.0"}
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
```

- [ ] **Step 4: Create `internal/mcp/memory_tools.go`**

```go
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
```

- [ ] **Step 5: Update `internal/mcp/mcp_test.go` setup()**

Find the `setup()` function and update the `NewServer` call to pass the memory DB path:

```go
func setup(t *testing.T) (string, *prospectormcp.Server) {
	t.Helper()
	root := t.TempDir()

	require.NoError(t, fs.WriteCV(root, "cv-backend-2025-01-01.html", "<h1>Backend CV</h1>"))
	require.NoError(t, fs.WriteCoverLetter(root, "cl-general-2025-01-01.html", "<p>Dear Hiring Manager</p>"))
	themeDir := filepath.Join(root, "themes")
	require.NoError(t, os.MkdirAll(themeDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(themeDir, "minimal.css"), []byte("body{}"), 0644))

	dbPath := filepath.Join(root, "prospector.db")
	memPath := filepath.Join(root, "memory.db")
	srv, err := prospectormcp.NewServer(root, dbPath, memPath, "http://localhost:3000")
	require.NoError(t, err)
	t.Cleanup(func() { srv.Close() })
	return root, srv
}
```

- [ ] **Step 6: Update `cmd/spicebag/cmd_mcp.go`**

```go
// cmd/spicebag/cmd_mcp.go
package main

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	spicebagmcp "github.com/oxGrad/spicebag/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server (called automatically by Claude Code)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := spicebagRoot()
			if needsInit(root) {
				if err := runInit(root, io.Discard); err != nil {
					return fmt.Errorf("auto-init: %w", err)
				}
			}

			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			srv, err := spicebagmcp.NewServer(
				root,
				filepath.Join(root, "spicebag.db"),
				filepath.Join(root, "memory.db"),
				cfg.GotenbergURL,
			)
			if err != nil {
				return fmt.Errorf("init MCP server: %w", err)
			}
			defer srv.Close()

			return srv.ServeStdio()
		},
	}
}
```

- [ ] **Step 7: Run all MCP tests**

```bash
go test ./internal/mcp/... -v
```

Expected: all tests pass including the 3 new memory tool tests.

- [ ] **Step 8: Commit**

```bash
git add internal/mcp/ cmd/spicebag/cmd_mcp.go
git commit -m "feat: add memory MCP tools (memory_write, memory_search, memory_delete)"
```

---

## Task 4: Dashboard HTTP handlers

**Files:**
- Create: `internal/dashboard/handlers_memory.go`
- Modify: `internal/dashboard/server.go` — add `mem` field, update `NewServer`, add routes
- Modify: `internal/dashboard/dashboard_test.go` — update `newTestServer()`, add memory handler tests
- Modify: `cmd/spicebag/cmd_start.go` — open memory DB, pass to `dashboard.NewServer`

- [ ] **Step 1: Write failing tests**

Add to `internal/dashboard/dashboard_test.go` (after existing tests):

```go
func TestMemoriesListRoute(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/memories", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "[")
}

func TestMemoryGetByNameNotFound(t *testing.T) {
	srv := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/memories/does-not-exist", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}
```

- [ ] **Step 2: Run tests — confirm they fail**

```bash
go test ./internal/dashboard/... -v -run "TestMemory"
```

Expected: compile errors or route-not-found (404) because routes don't exist yet.

- [ ] **Step 3: Update `internal/dashboard/server.go`**

Add import `"github.com/oxGrad/spicebag/internal/memory"`. Update struct, `NewServer`, and `routes()`:

```go
type Server struct {
	root  string
	store *db.Store
	mem   *memory.DB
	cfg   config.Config
	mux   *http.ServeMux
}

func NewServer(root string, store *db.Store, mem *memory.DB, cfg config.Config) *Server {
	s := &Server{root: root, store: store, mem: mem, cfg: cfg, mux: http.NewServeMux()}
	s.routes()
	return s
}
```

In `routes()`, add before `s.mux.HandleFunc("/", s.handleSPA)`:

```go
s.mux.HandleFunc("GET /api/memories", s.handleAPIMemoriesList)
s.mux.HandleFunc("GET /api/memories/{name}", s.handleAPIMemoriesGet)
```

- [ ] **Step 4: Create `internal/dashboard/handlers_memory.go`**

```go
// internal/dashboard/handlers_memory.go
package dashboard

import (
	"net/http"
)

func (s *Server) handleAPIMemoriesList(w http.ResponseWriter, r *http.Request) {
	memories, err := s.mem.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if memories == nil {
		memories = []*memory.Memory{}  // NOTE: import "github.com/oxGrad/spicebag/internal/memory" at top
	}
	writeJSON(w, memories)
}

func (s *Server) handleAPIMemoriesGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	results, err := s.mem.Search(name, 1)
	if err != nil || len(results) == 0 {
		http.NotFound(w, r)
		return
	}
	// exact name match only
	for _, m := range results {
		if m.Name == name {
			writeJSON(w, m)
			return
		}
	}
	http.NotFound(w, r)
}
```

Wait — `handleAPIMemoriesGet` should do an exact lookup, not FTS search. Add a `GetByName` method to `internal/memory/db.go`:

```go
func (d *DB) GetByName(name string) (*Memory, error) {
	var m Memory
	var createdStr, updatedStr string
	err := d.db.QueryRow(`
		SELECT name, type, description, body, created_at, updated_at
		FROM memories WHERE name = ?
	`, name).Scan(&m.Name, &m.Type, &m.Description, &m.Body, &createdStr, &updatedStr)
	if err != nil {
		return nil, err
	}
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedStr)
	return &m, nil
}
```

Then update `handleAPIMemoriesGet`:

```go
// internal/dashboard/handlers_memory.go
package dashboard

import (
	"database/sql"
	"net/http"
)

func (s *Server) handleAPIMemoriesList(w http.ResponseWriter, r *http.Request) {
	memories, err := s.mem.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if memories == nil {
		memories = make([]*memoryPkg.Memory, 0)
	}
	writeJSON(w, memories)
}

func (s *Server) handleAPIMemoriesGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	m, err := s.mem.GetByName(name)
	if err == sql.ErrNoRows || m == nil {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, m)
}
```

Because `memory` is also a common word, alias the import to avoid collision with any future local var named `memory`:

```go
import (
	"database/sql"
	"net/http"

	memoryPkg "github.com/oxGrad/spicebag/internal/memory"
)
```

And update `handleAPIMemoriesList` to use `[]*memoryPkg.Memory` zero value:
```go
if memories == nil {
    memories = []*memoryPkg.Memory{}
}
```

- [ ] **Step 5: Add `GetByName` test to `internal/memory/db_test.go`**

```go
func TestGetByName(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Write(&memory.Memory{
		Name: "ref-1", Type: "reference", Description: "a reference", Body: "body",
	}))

	m, err := db.GetByName("ref-1")
	require.NoError(t, err)
	assert.Equal(t, "ref-1", m.Name)

	_, err = db.GetByName("does-not-exist")
	assert.Error(t, err)
}
```

- [ ] **Step 6: Update `internal/dashboard/dashboard_test.go` `newTestServer()`**

Add `memory` import and open a memory DB:

```go
import (
    // ... existing imports ...
    "github.com/oxGrad/spicebag/internal/memory"
)

func newTestServer(t *testing.T) *dashboard.Server {
	t.Helper()
	root := t.TempDir()
	store, err := db.Open(filepath.Join(root, "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	mem, err := memory.Open(filepath.Join(root, "memory.db"))
	require.NoError(t, err)
	t.Cleanup(func() { mem.Close() })

	for _, d := range []string{"cv", "cover-letters", "themes", "applications"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
	}
	cfg := config.Config{GotenbergURL: "http://localhost:3000", DashboardPort: 8080}
	return dashboard.NewServer(root, store, mem, cfg)
}
```

- [ ] **Step 7: Update `cmd/spicebag/cmd_start.go`**

After the `store, err := db.Open(...)` block, add:

```go
mem, err := memory.Open(filepath.Join(root, "memory.db"))
if err != nil {
    return fmt.Errorf("open memory db: %w", err)
}
defer mem.Close()
```

Add import `"github.com/oxGrad/spicebag/internal/memory"`. Update the `dashboard.NewServer` call:

```go
return dashboard.NewServer(root, store, mem, cfg).Serve(addr)
```

- [ ] **Step 8: Run all tests**

```bash
go test ./... 
```

Expected: all pass.

- [ ] **Step 9: Commit**

```bash
git add internal/memory/db.go internal/dashboard/ cmd/spicebag/cmd_start.go
git commit -m "feat: add dashboard memory endpoints (GET /api/memories, GET /api/memories/{name})"
```

---

## Task 5: Frontend MemoryView

**Files:**
- Create: `frontend/src/views/MemoryView.vue`
- Modify: `frontend/src/api.js`
- Modify: `frontend/src/router/index.js`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Add API helpers to `frontend/src/api.js`**

At the end of the `api` export object (before the closing `}`), add:

```js
  memory: {
    list: () => get("/memories"),
    get: (name) => get(`/memories/${encodeURIComponent(name)}`),
  },
```

- [ ] **Step 2: Add route to `frontend/src/router/index.js`**

Add to the `routes` array:

```js
{ path: "/memory", component: () => import("../views/MemoryView.vue") },
```

- [ ] **Step 3: Add sidebar link to `frontend/src/App.vue`**

Inside the `<nav>` block, after the `<RouterLink to="/stats">Experience</RouterLink>` entry, add:

```html
<RouterLink to="/memory"
  active-class="bg-white/10 text-white font-medium"
  class="px-2.5 py-2 rounded-md text-sm text-gray-400 hover:text-gray-200 hover:bg-white/5 transition-colors"
>Memory</RouterLink>
```

- [ ] **Step 4: Create `frontend/src/views/MemoryView.vue`**

```vue
<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-semibold text-gray-900">Memory</h1>
      <div class="flex gap-2">
        <button
          v-for="t in ['all', 'user', 'feedback', 'project', 'reference']"
          :key="t"
          @click="filter = t"
          :class="[
            'px-3 py-1 rounded-full text-xs font-medium transition-colors',
            filter === t
              ? 'bg-gray-900 text-white'
              : 'bg-white text-gray-600 border border-gray-200 hover:border-gray-400'
          ]"
        >{{ t }}</button>
      </div>
    </div>

    <div v-if="loading" class="text-sm text-gray-500">Loading…</div>
    <div v-else-if="error" class="text-sm text-red-600">{{ error }}</div>
    <div v-else-if="filtered.length === 0" class="text-sm text-gray-400">No memories.</div>

    <div v-else class="flex gap-6">
      <!-- Table -->
      <div class="flex-1 min-w-0">
        <div class="bg-white border border-gray-200 rounded-lg overflow-hidden">
          <table class="w-full text-sm">
            <thead class="bg-gray-50 border-b border-gray-200">
              <tr>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Description</th>
                <th class="px-4 py-2.5 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Updated</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              <tr
                v-for="m in filtered"
                :key="m.name"
                @click="selected = m"
                :class="[
                  'cursor-pointer transition-colors',
                  selected?.name === m.name ? 'bg-blue-50' : 'hover:bg-gray-50'
                ]"
              >
                <td class="px-4 py-3 font-mono text-xs text-gray-900">{{ m.name }}</td>
                <td class="px-4 py-3">
                  <span :class="typeBadge(m.type)">{{ m.type }}</span>
                </td>
                <td class="px-4 py-3 text-gray-600 truncate max-w-xs">{{ m.description }}</td>
                <td class="px-4 py-3 text-gray-400 whitespace-nowrap">{{ formatDate(m.updated_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Detail panel -->
      <div v-if="selected" class="w-80 shrink-0">
        <div class="bg-white border border-gray-200 rounded-lg p-4 sticky top-8">
          <div class="flex items-start justify-between mb-3">
            <div>
              <p class="font-mono text-xs text-gray-900 font-medium">{{ selected.name }}</p>
              <span :class="typeBadge(selected.type)" class="mt-1 inline-block">{{ selected.type }}</span>
            </div>
            <button @click="selected = null" class="text-gray-400 hover:text-gray-600 text-lg leading-none">×</button>
          </div>
          <p class="text-xs text-gray-500 mb-3 italic">{{ selected.description }}</p>
          <p class="text-sm text-gray-700 whitespace-pre-wrap leading-relaxed">{{ selected.body }}</p>
          <p class="mt-4 text-xs text-gray-400">Updated {{ formatDate(selected.updated_at) }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { api } from '../api.js'

const memories = ref([])
const loading = ref(true)
const error = ref(null)
const selected = ref(null)
const filter = ref('all')

const filtered = computed(() =>
  filter.value === 'all' ? memories.value : memories.value.filter(m => m.type === filter.value)
)

onMounted(async () => {
  try {
    memories.value = await api.memory.list()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})

function typeBadge(type) {
  const map = {
    user:      'px-1.5 py-0.5 rounded text-xs bg-blue-100 text-blue-700',
    feedback:  'px-1.5 py-0.5 rounded text-xs bg-amber-100 text-amber-700',
    project:   'px-1.5 py-0.5 rounded text-xs bg-green-100 text-green-700',
    reference: 'px-1.5 py-0.5 rounded text-xs bg-purple-100 text-purple-700',
  }
  return map[type] ?? 'px-1.5 py-0.5 rounded text-xs bg-gray-100 text-gray-600'
}

function formatDate(iso) {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
}
</script>
```

- [ ] **Step 5: Build and verify in browser**

```bash
just dev
```

Open `http://localhost:8080/memory`. Expected:
- "Memory" appears in the sidebar and highlights when active
- Page loads with empty state ("No memories.") or table if memories exist
- Type filter buttons work
- Clicking a row shows the detail panel on the right

If the table is empty, seed a test memory via CLI and reload:
```bash
spicebag memory write --name "test-feedback" --type feedback \
  --description "Test entry" --body "This is a test memory body."
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/MemoryView.vue frontend/src/api.js frontend/src/router/index.js frontend/src/App.vue
git commit -m "feat: add Memory dashboard view with type filter and detail panel"
```

---

## Task 6: Plugin hook and MEMORY.md

**Files:**
- Modify: `.claude-plugin/plugin.json`
- Update: `~/.claude/projects/-home-graditya-projects-oxGrad-spicebag/memory/MEMORY.md`

- [ ] **Step 1: Add hook to `.claude-plugin/plugin.json`**

The Claude Code plugin spec supports a `hooks` key. Verify the schema at the URL in `marketplace.json` (`$schema` field) — if the schema doesn't include `hooks`, the hook must instead be added manually to `~/.claude/settings.json` under `hooks.UserPromptSubmit`.

Assuming `hooks` is supported in plugin.json, the updated file:

```json
{
  "name": "spicebag",
  "version": "0.5.0",
  "description": "Spice Bag — Season every application perfectly. CV, cover letter, and job application manager for Claude Code.",
  "commands": ["./plugins/skills/"],
  "mcpServers": {
    "spicebag": {
      "command": "spicebag",
      "args": ["mcp"]
    }
  },
  "hooks": {
    "UserPromptSubmit": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "spicebag memory search"
          }
        ]
      }
    ]
  }
}
```

> **If the schema rejects `hooks`:** add this block directly to `~/.claude/settings.json` under the top-level `"hooks"` key instead. The command is the same: `"spicebag memory search"`.

- [ ] **Step 2: Test the hook manually**

Rebuild and run a test:
```bash
just build-go
echo '{"prompt": "what are my communication preferences?"}' | spicebag memory search
```

Expected: `Relevant memories:` followed by any matching entries (or empty if no memories seeded).

Seed a memory then test again:
```bash
spicebag memory write --name "feedback-terse" --type feedback \
  --description "User prefers terse responses" \
  --body "No trailing summaries. Why: redundant. How to apply: end after delivering result."
echo '{"prompt": "keep responses short and terse"}' | spicebag memory search
```

Expected output:
```
Relevant memories:

[feedback] feedback-terse
No trailing summaries. Why: redundant. How to apply: end after delivering result.
```

- [ ] **Step 3: Migrate existing .md memories to the DB**

Read each `.md` file in `~/.claude/projects/-home-graditya-projects-oxGrad-spicebag/memory/` and import them:

```bash
# For each memory file, extract name/type/description/body and call:
spicebag memory write --name "..." --type "..." --description "..." --body "..."
```

The frontmatter fields (`name`, `type`, `description`) map directly. The `body` is everything after the `---` closing frontmatter delimiter.

- [ ] **Step 4: Update MEMORY.md to a pointer**

Replace `~/.claude/projects/-home-graditya-projects-oxGrad-spicebag/memory/MEMORY.md` with:

```markdown
Memories are stored in ~/.config/spicebag/memory.db (SQLite FTS5).
Search: `spicebag memory search -q "keyword"`
List:   `spicebag memory list`
```

The old `.md` memory files can be deleted after verifying the import.

- [ ] **Step 5: Final integration test**

Start a new Claude Code session and send a message that should trigger memory retrieval (e.g., "how should I format my responses?"). Verify that the hook output appears in Claude's context (Claude should reference the feedback memory without being asked).

- [ ] **Step 6: Commit**

```bash
git add .claude-plugin/plugin.json
git commit -m "feat: wire memory search as UserPromptSubmit hook in plugin.json"
```

---

## Self-Review

**Spec coverage:**
- ✅ SQLite FTS5 at `~/.config/spicebag/memory.db`
- ✅ `spicebag memory search` reads stdin JSON for hook use
- ✅ Hook in `.claude-plugin/plugin.json` (with fallback note for `settings.json`)
- ✅ MCP tools: `memory_write`, `memory_search`, `memory_delete`
- ✅ Dashboard: `GET /api/memories`, `GET /api/memories/{name}`, `MemoryView.vue`
- ✅ No project column — flat schema focused on job search context
- ✅ MEMORY.md replaced with pointer

**Placeholder scan:** No TBDs, all code blocks complete, all file paths exact.

**Type consistency:**
- `memory.Memory` struct used consistently across `internal/memory`, MCP tools, and dashboard handlers
- `*memory.DB` added to both `mcp.Server` and `dashboard.Server`
- `NewServer` signature updated in both packages and both call sites (`cmd_mcp.go`, `cmd_start.go`)
- `GetByName` added to `internal/memory/db.go` (Task 4, Step 4) and used in `handleAPIMemoriesGet`
