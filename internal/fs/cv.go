package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type FileInfo struct {
	Name       string
	ModifiedAt time.Time
	Size       int64
}

func cvDir(root string) string { return filepath.Join(root, "cv") }

func ListCVs(root string) ([]FileInfo, error) {
	return listHTMLFiles(cvDir(root))
}

func ReadCV(root, filename string) (string, error) {
	base := cvDir(root)
	resolved := filepath.Join(base, filename)
	if !strings.HasPrefix(resolved, filepath.Clean(base)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid filename: %q", filename)
	}
	data, err := os.ReadFile(resolved)
	return string(data), err
}

func WriteCV(root, filename, content string) error {
	dir := cvDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func listHTMLFiles(dir string) ([]FileInfo, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{Name: e.Name(), ModifiedAt: info.ModTime(), Size: info.Size()})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].ModifiedAt.After(files[j].ModifiedAt) })
	return files, nil
}
