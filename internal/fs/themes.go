package fs

import (
	"os"
	"path/filepath"
	"strings"
)

func themeDir(root string) string { return filepath.Join(root, "themes") }

func ListThemes(root string) ([]string, error) {
	entries, err := os.ReadDir(themeDir(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var themes []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".css" {
			themes = append(themes, strings.TrimSuffix(e.Name(), ".css"))
		}
	}
	return themes, nil
}

func ReadTheme(root, name string) (string, error) {
	data, err := os.ReadFile(filepath.Join(themeDir(root), name+".css"))
	return string(data), err
}
