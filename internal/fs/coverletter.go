package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func coverLetterDir(root string) string { return filepath.Join(root, "cover-letters") }

func ListCoverLetters(root string) ([]FileInfo, error) {
	return listHTMLFiles(coverLetterDir(root))
}

func ReadCoverLetter(root, filename string) (string, error) {
	base := coverLetterDir(root)
	resolved := filepath.Join(base, filename)
	if !strings.HasPrefix(resolved, filepath.Clean(base)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid filename: %q", filename)
	}
	data, err := os.ReadFile(resolved)
	return string(data), err
}

func WriteCoverLetter(root, filename, content string) error {
	dir := coverLetterDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}
