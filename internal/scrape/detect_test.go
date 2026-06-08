package scrape_test

import (
	"testing"

	"github.com/oxGrad/spicebag/internal/scrape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetect(t *testing.T) {
	cases := []struct {
		url, platform, token string
	}{
		{"https://boards.greenhouse.io/acme", "greenhouse", "acme"},
		{"https://boards.greenhouse.io/acme/jobs/123", "greenhouse", "acme"},
		{"https://jobs.lever.co/acme", "lever", "acme"},
		{"https://jobs.ashbyhq.com/acme", "ashby", "acme"},
		{"https://careers.smartrecruiters.com/Acme", "smartrecruiters", "Acme"},
		{"https://apply.workable.com/acme/", "workable", "acme"},
		{"https://acme.recruitee.com", "recruitee", "acme"},
		{"https://acme.breezy.hr", "breezy", "acme"},
		{"https://acme.bamboohr.com/careers", "bamboohr", "acme"},
		{"https://acme.jobs.personio.de", "personio", "acme"},
		{"https://acme.wd1.myworkdayjobs.com/External", "workday", "acme"},
	}
	for _, c := range cases {
		platform, token, err := scrape.Detect(c.url)
		require.NoError(t, err, c.url)
		assert.Equal(t, c.platform, platform, c.url)
		assert.Equal(t, c.token, token, c.url)
	}
}

func TestDetectUnsupported(t *testing.T) {
	_, _, err := scrape.Detect("https://acme.com/careers")
	assert.Error(t, err)
}

func TestNormalizeURL(t *testing.T) {
	assert.Equal(t,
		"https://boards.greenhouse.io/acme/jobs/42",
		scrape.NormalizeURL("https://Boards.Greenhouse.io/acme/jobs/42/?utm=x#frag"))
}
