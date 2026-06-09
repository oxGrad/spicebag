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

func TestLeverParse(t *testing.T) {
	data, err := os.ReadFile("testdata/lever.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	lever := Lever{BaseURL: srv.URL}
	jobs, err := lever.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Remote (APAC)", jobs[0].Location)
	assert.Equal(t, "https://jobs.lever.co/acme/abc-123", jobs[0].URL)
}
