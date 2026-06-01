// cmd/spicebag/cmd_init.go
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	"github.com/oxGrad/spicebag/internal/db"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up ~/.config/spicebag and register MCP server with Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runInit(spicebagRoot(), os.Stdout); err != nil {
				return err
			}
			fmt.Println("\nRun `spicebag start` to open the dashboard.")
			return nil
		},
	}
}

// needsInit reports whether the config directory has not yet been initialised.
func needsInit(root string) bool {
	_, err := os.Stat(filepath.Join(root, "config.toml"))
	return os.IsNotExist(err)
}

// runInit performs first-time setup. Informational output is written to w;
// pass io.Discard for silent operation (e.g. auto-init from mcp).
func runInit(root string, w io.Writer) error {
	dirs := []string{"cv", "cover-letters", "themes", "applications"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return err
		}
	}

	cfgPath := filepath.Join(root, "config.toml")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		if err := config.Save(cfgPath, config.Config{DashboardPort: 8080, GotenbergURL: "http://localhost:3000"}); err != nil {
			return err
		}
		fmt.Fprintln(w, "Created", cfgPath)
	}

	dbPath := filepath.Join(root, "spicebag.db")
	store, err := db.Open(dbPath)
	if err != nil {
		return err
	}
	store.Close()
	fmt.Fprintln(w, "Initialized database at", dbPath)

	if err := extractEmbedDir(themesFS, "themes", filepath.Join(root, "themes")); err != nil {
		fmt.Fprintf(w, "Warning: could not extract default themes: %v\n", err)
	} else {
		fmt.Fprintln(w, "Extracted default themes to", filepath.Join(root, "themes"))
	}

	home, _ := os.UserHomeDir()
	commandsDir := filepath.Join(home, ".claude", "commands", "spicebag")
	if err := extractEmbedDir(skillsFS, "skills", commandsDir); err != nil {
		fmt.Fprintf(w, "Warning: could not install skills: %v\n", err)
	} else {
		fmt.Fprintln(w, "Installed skills to", commandsDir)
	}

	composeDest := filepath.Join(root, "docker-compose.yml")
	if _, err := os.Stat(composeDest); os.IsNotExist(err) {
		src := filepath.Join(execDir(), "docker-compose.yml")
		data, err := os.ReadFile(src)
		if err != nil {
			fmt.Fprintf(w, "Warning: could not copy docker-compose.yml: %v\n", err)
		} else {
			if err := os.WriteFile(composeDest, data, 0o644); err != nil {
				fmt.Fprintf(w, "Warning: could not write docker-compose.yml: %v\n", err)
			} else {
				fmt.Fprintln(w, "Copied docker-compose.yml to", composeDest)
			}
		}
	}

	if err := registerMCPServer(); err != nil {
		fmt.Fprintf(w, "Warning: could not register MCP server automatically: %v\n", err)
		fmt.Fprintln(w, `Add manually to ~/.claude/mcp.json: {"mcpServers":{"spicebag":{"command":"spicebag","args":["mcp"]}}}`)
	} else {
		fmt.Fprintln(w, "Registered MCP server with Claude Code.")
	}

	return nil
}

func spicebagRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "spicebag")
}

func execDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func registerMCPServer() error {
	home, _ := os.UserHomeDir()
	mcpConfigPath := filepath.Join(home, ".claude", "mcp.json")

	var mcpCfg map[string]any
	data, err := os.ReadFile(mcpConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &mcpCfg); err != nil {
			return err
		}
	}
	if mcpCfg == nil {
		mcpCfg = map[string]any{}
	}

	servers, _ := mcpCfg["mcpServers"].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	servers["spicebag"] = map[string]any{
		"command": "spicebag",
		"args":    []string{"mcp"},
	}
	mcpCfg["mcpServers"] = servers

	out, err := json.MarshalIndent(mcpCfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(mcpConfigPath, out, 0o644)
}

func extractEmbedDir(fsys embed.FS, src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(fsys, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, rel)
		if _, err := os.Stat(destPath); err == nil {
			return nil // skip existing files
		}
		data, err := fsys.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
}
