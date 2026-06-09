package scrape

import (
	"context"
	"encoding/json"
	"fmt"
)

// RemoteOK reads the public API. The first array element is a legal notice and is skipped.
// BaseURL is overridable in tests; empty means the real endpoint.
type RemoteOK struct{ BaseURL string }

func (r RemoteOK) Name() string { return "remoteok" }

func (r RemoteOK) FetchJobs(ctx context.Context) ([]BoardJob, error) {
	base := r.BaseURL
	if base == "" {
		base = "https://remoteok.com/api"
	}
	body, err := httpGet(ctx, base)
	if err != nil {
		return nil, err
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	var out []BoardJob
	// raw[0] is always the legal notice; raw[1:] may be empty
	for _, item := range raw[1:] { // first element is a legal notice object
		var j struct {
			Position string `json:"position"`
			Company  string `json:"company"`
			Location string `json:"location"`
			URL      string `json:"url"`
		}
		if err := json.Unmarshal(item, &j); err != nil || j.URL == "" {
			continue
		}
		out = append(out, BoardJob{CompanyName: j.Company, Title: j.Position, Location: j.Location, URL: j.URL})
	}
	return out, nil
}
