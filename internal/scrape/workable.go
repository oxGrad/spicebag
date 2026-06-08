package scrape

import (
	"context"
	"fmt"
)

// Workable reads the public API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Workable struct{ BaseURL string }

func (wk Workable) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := wk.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://apply.workable.com/api/v3/accounts/%s/jobs", token)
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			URL      string `json:"url"`
			Location struct {
				LocationStr string `json:"location_str"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location.LocationStr, URL: j.URL})
	}
	return jobs, nil
}
