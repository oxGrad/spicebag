// cmd/prospector/cmd_sync.go
package main

import (
	"fmt"
	"path/filepath"

	"github.com/graditya/prospector/internal/db"
	"github.com/graditya/prospector/internal/fs"
	"github.com/graditya/prospector/internal/parser"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Re-parse all base CVs and refresh experience data in SQLite",
		RunE: func(cmd *cobra.Command, args []string) error {
			root := prospectorRoot()
			store, err := db.Open(filepath.Join(root, "prospector.db"))
			if err != nil {
				return err
			}
			defer store.Close()

			cvFiles, err := fs.ListCVs(root)
			if err != nil {
				return err
			}

			for _, f := range cvFiles {
				content, err := fs.ReadCV(root, f.Name)
				if err != nil {
					fmt.Printf("skip %s: %v\n", f.Name, err)
					continue
				}

				if err := store.DeleteExperienceBySyncedFrom(f.Name); err != nil {
					return err
				}

				entries, err := parser.ParseExperience(content)
				if err != nil {
					fmt.Printf("skip %s (parse error): %v\n", f.Name, err)
					continue
				}

				dbEntries := make([]db.ExperienceEntry, len(entries))
				for i, e := range entries {
					dbEntries[i] = db.ExperienceEntry{
						RoleType:   e.RoleType,
						Company:    e.Company,
						StartDate:  e.StartDate,
						EndDate:    e.EndDate,
						SyncedFrom: f.Name,
					}
				}
				if len(dbEntries) > 0 {
					if err := store.UpsertExperience(dbEntries); err != nil {
						return err
					}
				}
				fmt.Printf("synced %s (%d entries)\n", f.Name, len(dbEntries))
			}
			return nil
		},
	}
}
