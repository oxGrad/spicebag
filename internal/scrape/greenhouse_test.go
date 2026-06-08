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

func TestGreenhouseParse(t *testing.T) {
	data, err := os.ReadFile("testdata/greenhouse.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	gh := Greenhouse{BaseURL: srv.URL}
	jobs, err := gh.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Senior SRE", jobs[0].Title)
	assert.Equal(t, "Remote - Anywhere", jobs[0].Location)
	assert.Equal(t, "https://boards.greenhouse.io/acme/jobs/42", jobs[0].URL)
}
