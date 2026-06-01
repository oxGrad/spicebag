package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ApplicationRequest struct {
	Company            string
	Role               string
	Date               string // YYYY-MM-DD
	CVContent          string
	CoverLetterContent string
	JobPostContent     string
	BaseCVUsed         string
	Notes              string
}

type ApplicationMetadata struct {
	Company     string `yaml:"company"`
	Role        string `yaml:"role"`
	AppliedDate string `yaml:"applied_date"`
	BaseCVUsed  string `yaml:"base_cv_used"`
	Notes       string `yaml:"notes"`
}

func slugify(s string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(s), " ", "-"))
}

func CreateApplication(root string, req ApplicationRequest) (string, error) {
	folderPath := fmt.Sprintf("%s/%s/%s", slugify(req.Company), slugify(req.Role), req.Date)
	dir := filepath.Join(root, "applications", folderPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	files := map[string]string{
		"cv.md":           req.CVContent,
		"cover-letter.md": req.CoverLetterContent,
		"job-post.md":     req.JobPostContent,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			return "", err
		}
	}

	meta := ApplicationMetadata{
		Company:     req.Company,
		Role:        req.Role,
		AppliedDate: req.Date,
		BaseCVUsed:  req.BaseCVUsed,
		Notes:       req.Notes,
	}
	metaBytes, err := yaml.Marshal(meta)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "metadata.yaml"), metaBytes, 0644); err != nil {
		return "", err
	}
	return folderPath, nil
}

func ReadApplicationMetadata(root, folderPath string) (ApplicationMetadata, error) {
	data, err := os.ReadFile(filepath.Join(root, "applications", folderPath, "metadata.yaml"))
	if err != nil {
		return ApplicationMetadata{}, err
	}
	var meta ApplicationMetadata
	return meta, yaml.Unmarshal(data, &meta)
}
