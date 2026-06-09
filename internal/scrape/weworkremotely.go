package scrape

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"
)

// WeWorkRemotely reads the public RSS feed.
// BaseURL is overridable in tests; empty means the real endpoint.
// Item titles have the format "Company Name: Job Title".
type WeWorkRemotely struct{ BaseURL string }

func (w WeWorkRemotely) Name() string { return "weworkremotely" }

func (w WeWorkRemotely) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := w.BaseURL
	if base == "" {
		base = "https://weworkremotely.com/remote-jobs.rss"
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Items []struct {
			Title  string `xml:"title"`
			Link   string `xml:"link"`
			Region string `xml:"region"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(body, &doc); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	var out []BoardJob
	for _, item := range doc.Items {
		company, title := splitWWRTitle(item.Title)
		if title == "" {
			continue
		}
		out = append(out, BoardJob{CompanyName: company, Title: title, Location: item.Region, URL: item.Link})
	}
	return out, nil
}

// splitWWRTitle splits "Company Name: Job Title" into its two parts.
func splitWWRTitle(raw string) (company, title string) {
	company, title, ok := strings.Cut(raw, ": ")
	if !ok {
		return "", strings.TrimSpace(raw)
	}
	return strings.TrimSpace(company), strings.TrimSpace(title)
}
