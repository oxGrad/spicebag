package scrape

import (
	"context"
	"fmt"
	"strings"
)

// SmartRecruiters reads the public job board API.
// BaseURL is overridable in tests; empty means the real endpoint.
type SmartRecruiters struct{ BaseURL string }

func (s SmartRecruiters) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := s.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.smartrecruiters.com/v1/companies/%s/postings", token)
	}
	var resp struct {
		Content []struct {
			Name     string `json:"name"`
			Ref      string `json:"ref"`
			Location struct {
				City    string `json:"city"`
				Country string `json:"country"`
			} `json:"location"`
		} `json:"content"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Content))
	for _, j := range resp.Content {
		var parts []string
		if j.Location.City != "" {
			parts = append(parts, j.Location.City)
		}
		if j.Location.Country != "" {
			parts = append(parts, j.Location.Country)
		}
		jobs = append(jobs, Job{Title: j.Name, Location: strings.Join(parts, ", "), URL: j.Ref})
	}
	return jobs, nil
}
