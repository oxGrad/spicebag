package scrape

import (
	"context"
	"fmt"
)

// Greenhouse reads the public boards API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Greenhouse struct{ BaseURL string }

func (g Greenhouse) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := g.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://boards-api.greenhouse.io/v1/boards/%s/jobs", token)
	}
	var resp struct {
		Jobs []struct {
			Title       string `json:"title"`
			AbsoluteURL string `json:"absolute_url"`
			Location    struct {
				Name string `json:"name"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location.Name, URL: j.AbsoluteURL})
	}
	return jobs, nil
}
