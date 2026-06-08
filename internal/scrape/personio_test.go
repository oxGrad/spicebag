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

func TestPersonioParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/personio.xml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	defer srv.Close()

	jobs, err := Personio{BaseURL: srv.URL, Token: "acme"}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Remote", jobs[0].Location)
	assert.Equal(t, "https://acme.jobs.personio.de/job/9001", jobs[0].URL)
}
