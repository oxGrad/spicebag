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

func TestRecruiteeParse(t *testing.T) {
	data, _ := os.ReadFile("testdata/recruitee.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer srv.Close()

	jobs, err := Recruitee{BaseURL: srv.URL}.FetchJobs(context.Background(), "acme")
	require.NoError(t, err)
	require.Len(t, jobs, 1)
	assert.Equal(t, "Cloud Engineer", jobs[0].Title)
	assert.Equal(t, "Remote - Asia", jobs[0].Location)
	assert.Equal(t, "https://acme.recruitee.com/o/cloud-engineer", jobs[0].URL)
}
