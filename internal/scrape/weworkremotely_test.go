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

func TestWeWorkRemotelyParse(t *testing.T) {
	data, err := os.ReadFile("testdata/weworkremotely.xml")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := WeWorkRemotely{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "Senior DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://weworkremotely.com/remote-jobs/view/3001", jobs[0].URL)
	assert.Equal(t, "Beta Corp", jobs[1].CompanyName)
}
