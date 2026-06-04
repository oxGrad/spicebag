// cmd/spicebag/cmd_mcp.go
package main

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/oxGrad/spicebag/internal/config"
	spicebagmcp "github.com/oxGrad/spicebag/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
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

	cmd.AddCommand(&cobra.Command{
		Use:   "tools",
		Short: "List all registered MCP tools",
		RunE: func(c *cobra.Command, args []string) error {
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
			srv, err := spicebagmcp.NewServer(root,
				filepath.Join(root, "spicebag.db"),
				filepath.Join(root, "memory.db"),
				cfg.GotenbergURL,
			)
			if err != nil {
				return fmt.Errorf("init MCP server: %w", err)
			}
			defer srv.Close()

			tools, err := srv.ListTools(context.Background())
			if err != nil {
				return err
			}
			fmt.Printf("%-35s %s\n", "TOOL", "DESCRIPTION")
			fmt.Printf("%-35s %s\n", "----", "-----------")
			for _, t := range tools {
				fmt.Printf("%-35s %s\n", t.Name, t.Description)
			}
			return nil
		},
	})

	return cmd
}
