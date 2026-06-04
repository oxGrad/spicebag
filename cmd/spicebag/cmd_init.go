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
		Short: "Set up ~/.config/spicebag with default config, database, and themes",
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

	home, _ := os.UserHomeDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := ensureMemoryHook(settingsPath); err != nil {
		fmt.Fprintf(w, "Warning: could not install memory hook in %s: %v\n", settingsPath, err)
	} else {
		fmt.Fprintln(w, "Installed memory search hook in", settingsPath,
			"(fires on every prompt — silent when no memories match)")
	}

	return nil
}

const memoryHookCommand = "spicebag memory search"

// ensureMemoryHook reads a Claude settings JSON file (creating it if absent)
// and appends the spicebag memory search hook to hooks.UserPromptSubmit if it
// is not already present. The operation is idempotent.
func ensureMemoryHook(settingsPath string) error {
	var settings map[string]any

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		settings = map[string]any{}
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			return fmt.Errorf("parse %s: %w", settingsPath, err)
		}
	}

	// Navigate / create hooks.UserPromptSubmit
	hooksRaw := settings["hooks"]
	hooksMap, ok := hooksRaw.(map[string]any)
	if !ok {
		hooksMap = map[string]any{}
	}

	upsRaw := hooksMap["UserPromptSubmit"]
	ups, ok := upsRaw.([]any)
	if !ok {
		ups = []any{}
	}

	// Check if our hook is already present anywhere in the list.
	for _, entryRaw := range ups {
		entry, ok := entryRaw.(map[string]any)
		if !ok {
			continue
		}
		hooksListRaw := entry["hooks"]
		hooksList, ok := hooksListRaw.([]any)
		if !ok {
			continue
		}
		for _, hRaw := range hooksList {
			h, ok := hRaw.(map[string]any)
			if !ok {
				continue
			}
			if cmd, _ := h["command"].(string); cmd == memoryHookCommand {
				return nil // already installed
			}
		}
	}

	// Append the new hook entry.
	newEntry := map[string]any{
		"matcher": "",
		"hooks": []any{
			map[string]any{
				"type":    "command",
				"command": memoryHookCommand,
			},
		},
	}
	ups = append(ups, newEntry)
	hooksMap["UserPromptSubmit"] = ups
	settings["hooks"] = hooksMap

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0o644)
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
