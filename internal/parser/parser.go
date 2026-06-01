// internal/parser/parser.go
package parser

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type ExperienceEntry struct {
	RoleType  string `yaml:"role_type"`
	Company   string `yaml:"company"`
	StartDate string `yaml:"start"`
	EndDate   string `yaml:"end"`
}

type frontmatter struct {
	Experience []ExperienceEntry `yaml:"experience"`
}

func ParseExperience(markdown string) ([]ExperienceEntry, error) {
	if !strings.HasPrefix(markdown, "---") {
		return nil, nil
	}
	// extract content between first --- and second ---
	rest := strings.TrimPrefix(markdown, "---")
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return nil, nil
	}
	yamlBlock := strings.TrimSpace(rest[:end])

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, err
	}
	return fm.Experience, nil
}
