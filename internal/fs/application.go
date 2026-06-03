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
	JobURL             string
	JobSummary         string
}

type ApplicationMetadata struct {
	Company     string `yaml:"company"`
	Role        string `yaml:"role"`
	AppliedDate string `yaml:"applied_date"`
	BaseCVUsed  string `yaml:"base_cv_used"`
	Notes       string `yaml:"notes"`
	JobURL      string `yaml:"job_url,omitempty"`
	JobSummary  string `yaml:"job_summary,omitempty"`
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	// strip any character that is not a-z, 0-9, or -
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func CreateApplication(root string, req ApplicationRequest) (string, error) {
	folderPath := filepath.Join(slugify(req.Company), slugify(req.Role), req.Date)
	dir := filepath.Join(root, "applications", folderPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	files := map[string]string{
		"cv.html":           req.CVContent,
		"cover-letter.html": req.CoverLetterContent,
		"job-post.md":       req.JobPostContent,
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
		JobURL:      req.JobURL,
		JobSummary:  req.JobSummary,
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
	base := filepath.Join(root, "applications")
	resolved := filepath.Join(base, folderPath, "metadata.yaml")
	if !strings.HasPrefix(resolved, filepath.Clean(base)+string(os.PathSeparator)) {
		return ApplicationMetadata{}, fmt.Errorf("invalid folderPath: %q", folderPath)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return ApplicationMetadata{}, err
	}
	var meta ApplicationMetadata
	return meta, yaml.Unmarshal(data, &meta)
}
