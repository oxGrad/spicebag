package scrape

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAshbyParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/ashby.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Ashby{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Site Reliability Engineer", jobs[0].Title)
	assert.Equal(t, "Remote - Worldwide", jobs[0].Location)
	assert.Equal(t, "https://jobs.ashbyhq.com/acme/111", jobs[0].URL)
}
