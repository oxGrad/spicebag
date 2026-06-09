package scrape

import "context"

// Jobicy reads the public remote jobs API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Jobicy struct{ BaseURL string }

func (j Jobicy) Name() string { return "jobicy" }

func (j Jobicy) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := j.BaseURL
	if base == "" {
		base = "https://jobicy.com/api/v2/remote-jobs"
	}
	var resp struct {
		Jobs []struct {
			Title   string `json:"jobTitle"`
			Company string `json:"companyName"`
			Geo     string `json:"jobGeo"`
			URL     string `json:"url"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	out := make([]BoardJob, 0, len(resp.Jobs))
	for _, job := range resp.Jobs {
		out = append(out, BoardJob{CompanyName: job.Company, Title: job.Title, Location: job.Geo, URL: job.URL})
	}
	return out, nil
}
