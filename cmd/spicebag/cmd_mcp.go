// cmd/spicebag/cmd_mcp.go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/graditya/prospector/internal/config"
	prospectormcp "github.com/graditya/prospector/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start MCP server (called automatically by Claude Code)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := spicebagRoot()
			cfgPath := filepath.Join(root, "config.toml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			srv, err := prospectormcp.NewServer(root, filepath.Join(root, "spicebag.db"), cfg.GotenbergURL)
			if err != nil {
				return fmt.Errorf("init MCP server: %w", err)
			}
			defer srv.Close()

			return srv.ServeStdio()
		},
	}
}
