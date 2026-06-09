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

func TestWorkableParse(t *testing.T) {
	data, err := os.ReadFile("testdata/workable.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Workable{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Remote, Indonesia", jobs[0].Location)
	assert.Equal(t, "https://apply.workable.com/acme/j/ABC123", jobs[0].URL)
}
