package scrape

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL strips query/fragment, trailing slash, lowercases host.
// Kept identical in spirit to db.NormalizeJobURL so linkage agrees.
func NormalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	u.Path = strings.TrimRight(u.Path, "/")
	return u.Scheme + "://" + u.Host + u.Path
}

// Detect identifies the ATS platform and extracts the company token from a
// careers URL. Returns an error for unsupported hosts.
func Detect(raw string) (platform, token string, err error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", "", fmt.Errorf("invalid URL: %q", raw)
	}
	host := strings.ToLower(u.Host)
	segs := pathSegments(u.Path)
	first := ""
	if len(segs) > 0 {
		first = segs[0]
	}
	sub := subdomain(host)

	switch {
	case host == "boards.greenhouse.io" && first != "":
		return "greenhouse", first, nil
	case host == "jobs.lever.co" && first != "":
		return "lever", first, nil
	case host == "jobs.ashbyhq.com" && first != "":
		return "ashby", first, nil
	case host == "careers.smartrecruiters.com" && first != "":
		return "smartrecruiters", first, nil
	case host == "apply.workable.com" && first != "":
		return "workable", first, nil
	case strings.HasSuffix(host, ".recruitee.com") && sub != "":
		return "recruitee", sub, nil
	case strings.HasSuffix(host, ".breezy.hr") && sub != "":
		return "breezy", sub, nil
	case strings.HasSuffix(host, ".bamboohr.com") && sub != "":
		return "bamboohr", sub, nil
	case strings.HasSuffix(host, ".jobs.personio.de") && sub != "":
		return "personio", sub, nil
	case strings.HasSuffix(host, ".myworkdayjobs.com") && sub != "":
		return "workday", sub, nil
	}
	return "", "", fmt.Errorf("unsupported ATS host: %q", host)
}

func pathSegments(p string) []string {
	var out []string
	for _, s := range strings.Split(p, "/") {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// subdomain returns the left-most label (e.g. "acme" from "acme.recruitee.com",
// "acme" from "acme.wd1.myworkdayjobs.com", "acme" from "acme.jobs.personio.de").
func subdomain(host string) string {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}
