package scrape

import (
	"context"
	"fmt"
)

// Breezy reads the public job board API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Breezy struct{ BaseURL string }

func (b Breezy) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := b.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.breezy.hr/json", token)
	}
	var resp []struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Location struct {
			Name string `json:"name"`
		} `json:"location"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp))
	for _, j := range resp {
		jobs = append(jobs, Job{Title: j.Name, Location: j.Location.Name, URL: j.URL})
	}
	return jobs, nil
}
