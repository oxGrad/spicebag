package scrape

import "context"

// Remotive reads the public remote jobs API.
// BaseURL is overridable in tests; empty means the real endpoint.
type Remotive struct{ BaseURL string }

func (r Remotive) Name() string { return "remotive" }

func (r Remotive) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := r.BaseURL
	if base == "" {
		base = "https://remotive.com/api/remote-jobs"
	}
	var resp struct {
		Jobs []struct {
			Title    string `json:"title"`
			URL      string `json:"url"`
			Company  string `json:"company_name"`
			Location string `json:"candidate_required_location"`
		} `json:"jobs"`
	}
	if err := httpGetJSON(ctx, base, &resp); err != nil {
		return nil, err
	}
	out := make([]BoardJob, 0, len(resp.Jobs))
	for _, j := range resp.Jobs {
		out = append(out, BoardJob{CompanyName: j.Company, Title: j.Title, Location: j.Location, URL: j.URL})
	}
	return out, nil
}
