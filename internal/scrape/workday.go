package scrape

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Workday struct {
	CareersURL string // full URL needed to derive the CXS endpoint
}

// workdayEndpoint derives the CXS jobs API endpoint from a Workday careers URL.
// https://acme.wd1.myworkdayjobs.com/External
//
//	-> https://acme.wd1.myworkdayjobs.com/wday/cxs/acme/External/jobs
func workdayEndpoint(careersURL string) (string, error) {
	u, err := url.Parse(careersURL)
	if err != nil || !strings.HasSuffix(strings.ToLower(u.Host), ".myworkdayjobs.com") {
		return "", fmt.Errorf("%w: not a Workday URL", ErrUnexpectedFormat)
	}
	tenant := strings.Split(u.Host, ".")[0]
	segs := pathSegments(u.Path)
	if tenant == "" || len(segs) == 0 {
		return "", fmt.Errorf("%w: cannot derive Workday site", ErrUnexpectedFormat)
	}
	site := segs[len(segs)-1]
	return fmt.Sprintf("https://%s/wday/cxs/%s/%s/jobs", u.Host, tenant, site), nil
}

func (wd Workday) FetchJobs(ctx context.Context, token string) ([]Job, error) {
	endpoint, err := workdayEndpoint(wd.CareersURL)
	if err != nil {
		return nil, err
	}

	path, found := launcher.LookPath()
	if !found {
		return nil, fmt.Errorf("%w: chrome not found for Workday", ErrNetwork)
	}
	u, err := launcher.New().Bin(path).Headless(true).NoSandbox(true).Launch()
	if err != nil {
		return nil, fmt.Errorf("%w: launch chrome: %v", ErrNetwork, err)
	}
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("%w: connect chrome: %v", ErrNetwork, err)
	}
	defer browser.Close()

	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		return nil, fmt.Errorf("%w: open page: %v", ErrNetwork, err)
	}
	defer page.Close()

	js := fmt.Sprintf(`async () => {
		const res = await fetch(%q, {
			method: 'POST',
			headers: {'Content-Type': 'application/json'},
			body: JSON.stringify({appliedFacets:{}, limit:100, offset:0, searchText:''})
		});
		return await res.text();
	}`, endpoint)

	time.Sleep(jitterScrape())
	res, err := page.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("%w: workday fetch: %v", ErrBlocked, err)
	}

	var payload struct {
		JobPostings []struct {
			Title         string `json:"title"`
			LocationsText string `json:"locationsText"`
			ExternalPath  string `json:"externalPath"`
		} `json:"jobPostings"`
	}
	if err := json.Unmarshal([]byte(res.Value.Str()), &payload); err != nil {
		return nil, fmt.Errorf("parse: %w", ErrUnexpectedFormat)
	}

	host := ""
	if pu, perr := url.Parse(wd.CareersURL); perr == nil {
		host = pu.Scheme + "://" + pu.Host
	}
	jobs := make([]Job, 0, len(payload.JobPostings))
	for _, j := range payload.JobPostings {
		jobs = append(jobs, Job{
			Title:    j.Title,
			Location: j.LocationsText,
			URL:      host + j.ExternalPath,
		})
	}
	return jobs, nil
}

// jitterScrape is a local delay before the Workday call.
func jitterScrape() time.Duration { return 800 * time.Millisecond }
