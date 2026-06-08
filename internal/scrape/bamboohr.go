package scrape

import (
	"context"
	"fmt"
)

// BambooHR's list endpoint omits absolute URLs, so we build them from the token.
// Token is set by the test for URL construction; in production FetchJobs receives
// the token as its argument.
type BambooHR struct {
	BaseURL string
	Token   string // test override for URL construction
}

func (b BambooHR) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	if b.Token != "" {
		token = b.Token
	}
	base := b.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.bamboohr.com/careers/list", token)
	}
	var resp struct {
		Result []struct {
			ID             string `json:"id"`
			JobOpeningName string `json:"jobOpeningName"`
			Location       struct {
				City string `json:"city"`
			} `json:"location"`
		} `json:"result"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Result))
	for _, j := range resp.Result {
		url := fmt.Sprintf("https://%s.bamboohr.com/careers/%s", token, j.ID)
		jobs = append(jobs, Job{Title: j.JobOpeningName, Location: j.Location.City, URL: url})
	}
	return jobs, nil
}
