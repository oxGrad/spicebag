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

func TestRemoteOKParse(t *testing.T) {
	data, err := os.ReadFile("testdata/remoteok.json")
	require.NoError(t, err)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := RemoteOK{BaseURL: srv.URL}.FetchJobs(context.Background())
	require.NoError(t, err)
	require.Len(t, jobs, 2)
	assert.Equal(t, "DevOps Engineer", jobs[0].Title)
	assert.Equal(t, "Acme Corp", jobs[0].CompanyName)
	assert.Equal(t, "Worldwide", jobs[0].Location)
	assert.Equal(t, "https://remoteok.com/remote-jobs/2001", jobs[0].URL)
}
