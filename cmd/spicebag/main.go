package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:     "spicebag",
		Short:   "Season every application perfectly — CV and cover letter manager for Claude Code",
		Version: Version,
	}

	root.AddCommand(
		newInitCmd(),
		newMCPCmd(),
		newStartCmd(),
		newStopCmd(),
		newMemoryCmd(),
	)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
