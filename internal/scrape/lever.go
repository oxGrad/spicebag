package scrape

import (
	"context"
	"fmt"
)

// Lever reads the public postings API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Lever struct{ BaseURL string }

func (l Lever) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := l.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json", token)
	}
	var resp []struct {
		Text       string `json:"text"`
		HostedURL  string `json:"hostedUrl"`
		Categories struct {
			Location string `json:"location"`
		} `json:"categories"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp))
	for _, j := range resp {
		jobs = append(jobs, Job{Title: j.Text, Location: j.Categories.Location, URL: j.HostedURL})
	}
	return jobs, nil
}
