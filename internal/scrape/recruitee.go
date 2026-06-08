package scrape

import (
	"context"
	"fmt"
)

type Recruitee struct{ BaseURL string }

func (rc Recruitee) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	base := rc.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.recruitee.com/api/offers/", token)
	}
	var resp struct {
		Offers []struct {
			Title      string `json:"title"`
			Location   string `json:"location"`
			CareersURL string `json:"careers_url"`
		} `json:"offers"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	jobs := make([]Job, 0, len(resp.Offers))
	for _, j := range resp.Offers {
		jobs = append(jobs, Job{Title: j.Title, Location: j.Location, URL: j.CareersURL})
	}
	return jobs, nil
}
