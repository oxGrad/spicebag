package scrape

import (
	"context"
	"fmt"
)

// Ashby reads the public job board API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Ashby struct{ BaseURL string }

func (a Ashby) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := a.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.ashbyhq.com/posting-api/job-board/%s", token)
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			Location string `json:"location"`
			JobURL   string `json:"jobUrl"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location, URL: j.JobURL})
	}
	return jobs, nil
}
