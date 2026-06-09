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

func TestJobicyParse(t *testing.T) {
	data, err := os.ReadFile("testdata/jobicy.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Jobicy{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Platform Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://jobicy.com/jobs/4001-platform-engineer", jobs[0].URL)
}
