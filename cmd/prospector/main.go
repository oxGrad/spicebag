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
