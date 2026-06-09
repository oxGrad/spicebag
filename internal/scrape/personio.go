package scrape

import (
	"context"
	"encoding/xml"
	"fmt"
)

// Personio reads the public jobs XML feed.
// BaseURL is overridable in tests; empty means the real endpoint.
// Token is overridable in tests; empty means it is derived from the parameter.
type Personio struct {
	BaseURL string
	Token   string
}

func (p Personio) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	if p.Token != "" {
		token = p.Token
	}
	base := p.BaseURL
	if base == "" {
		base = fmt.Sprintf("https://%s.jobs.personio.de/xml", token)
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Positions []struct {
			ID     string `xml:"id"`
			Name   string `xml:"name"`
			Office string `xml:"office"`
		} `xml:"position"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	jobs := make([]Job, 0, len(doc.Positions))
	for _, j := range doc.Positions {
		url := fmt.Sprintf("https://%s.jobs.personio.de/job/%s", token, j.ID)
		jobs = append(jobs, Job{Title: j.Name, Location: j.Office, URL: url})
	}
	return jobs, nil
}
