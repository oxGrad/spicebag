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
