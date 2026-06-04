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
