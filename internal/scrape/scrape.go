package scrape

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Job is a single vacancy as returned by an adapter.
type Job struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	URL      string `json:"url"`
}

// Adapter fetches jobs for one ATS platform given a company token.
type Adapter interface {
	FetchJobs(ctx context.Context, token string) ([]Job, error)
}

// userAgent is a realistic desktop Chrome UA so public endpoints don't reject us.
const userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// httpGetJSON performs a GET and decodes the JSON body into out.
func httpGetJSON(ctx context.Context, url string, out any) error {
	body, err := httpGet(ctx, url)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}
	return nil
}

// httpGet performs a GET and returns the raw body.
func httpGet(ctx context.Context, url string) ([]byte, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json, text/xml, */*")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrBlocked
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: status %d", ErrUnexpectedFormat, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// Sentinel errors for classification into user-facing messages.
var (
	ErrNotFound         = errors.New("not found")
	ErrUnexpectedFormat = errors.New("unexpected format")
	ErrNetwork          = errors.New("network error")
	ErrBlocked          = errors.New("blocked")
)

// Registry maps platform names to adapters. Each adapter task adds its entry.
// Workday is intentionally absent — it is special-cased in fetch_ats_jobs
// because it needs the company's CareersURL.
func Registry() map[string]Adapter {
	return map[string]Adapter{
		"greenhouse":     Greenhouse{},
		"lever":          Lever{},
		"ashby":          Ashby{},
		"smartrecruiters": SmartRecruiters{},
	}
}

// ClassifyError maps a fetch error to a human-readable, platform-aware message.
func ClassifyError(platform string, err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrNotFound):
		return fmt.Sprintf("Company not found on %s — the token may have changed", platform)
	case errors.Is(err, ErrBlocked):
		return "Request was blocked — try again later"
	case errors.Is(err, ErrNetwork):
		return fmt.Sprintf("Couldn't reach %s (network/timeout)", platform)
	case errors.Is(err, ErrUnexpectedFormat):
		return fmt.Sprintf("Couldn't parse %s response — format may have changed", platform)
	default:
		return err.Error()
	}
}
