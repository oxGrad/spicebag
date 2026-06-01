package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "prospector",
		Short: "CV, cover letter, and job application manager for Claude Code",
	}

	root.AddCommand(
		newInitCmd(),
		newMCPCmd(),
		newUpCmd(),
		newSyncCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{Use: "init", Short: "Set up ~/.config/prospector"}
}

func newMCPCmd() *cobra.Command {
	return &cobra.Command{Use: "mcp", Short: "Start MCP server"}
}

func newUpCmd() *cobra.Command {
	return &cobra.Command{Use: "up", Short: "Start Gotenberg"}
}

func newSyncCmd() *cobra.Command {
	return &cobra.Command{Use: "sync", Short: "Sync experience data"}
}
