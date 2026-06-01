package fs

import (
	"os"
	"path/filepath"
)

func coverLetterDir(root string) string { return filepath.Join(root, "cover-letters") }

func ListCoverLetters(root string) ([]FileInfo, error) {
	return listMarkdownFiles(coverLetterDir(root))
}

func ReadCoverLetter(root, filename string) (string, error) {
	data, err := os.ReadFile(filepath.Join(coverLetterDir(root), filename))
	return string(data), err
}

func WriteCoverLetter(root, filename, content string) error {
	dir := coverLetterDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}
