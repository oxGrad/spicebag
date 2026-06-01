// cmd/prospector/cmd_init.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/graditya/prospector/internal/config"
	"github.com/graditya/prospector/internal/db"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Set up ~/.config/prospector and register MCP server with Claude Code",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()

			dirs := []string{"cv", "cover-letters", "themes", "applications"}
			for _, d := range dirs {
				if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
					return err
				}
			}

			cfgPath := filepath.Join(root, "config.toml")
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if err := config.Save(cfgPath, config.Config{DashboardPort: 8080, GotenbergURL: "http://localhost:3000"}); err != nil {
					return err
				}
				fmt.Println("Created", cfgPath)
			}

			dbPath := filepath.Join(root, "prospector.db")
			store, err := db.Open(dbPath)
			if err != nil {
				return err
			}
			store.Close()
			fmt.Println("Initialized database at", dbPath)

			// copy docker-compose.yml from next to binary to ~/.config/prospector/
			composeDest := filepath.Join(root, "docker-compose.yml")
			if _, err := os.Stat(composeDest); os.IsNotExist(err) {
				src := filepath.Join(execDir(), "docker-compose.yml")
				data, err := os.ReadFile(src)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not copy docker-compose.yml: %v\n", err)
				} else {
					if err := os.WriteFile(composeDest, data, 0644); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: could not write docker-compose.yml: %v\n", err)
					} else {
						fmt.Println("Copied docker-compose.yml to", composeDest)
					}
				}
			}

			if err := registerMCPServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not register MCP server automatically: %v\n", err)
				fmt.Println("Add this manually to your Claude Code MCP config:")
				printMCPConfig()
			} else {
				fmt.Println("Registered MCP server with Claude Code.")
			}

			fmt.Println("\nNext steps:")
			fmt.Println("  prospector up       # start Gotenberg")
			fmt.Println("  prospector serve    # start dashboard")
			return nil
		},
	}
}

func prospectorRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "prospector")
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
	servers["prospector"] = map[string]any{
		"command": "prospector",
		"args":    []string{"mcp"},
	}
	mcpCfg["mcpServers"] = servers

	out, err := json.MarshalIndent(mcpCfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(mcpConfigPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(mcpConfigPath, out, 0644)
}

func printMCPConfig() {
	fmt.Println(`{
  "mcpServers": {
    "prospector": { "command": "prospector", "args": ["mcp"] }
  }
}`)
}
