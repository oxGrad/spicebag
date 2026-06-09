package scrape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkdayEndpointFromCareersURL(t *testing.T) {
	ep, err := workdayEndpoint("https://acme.wd1.myworkdayjobs.com/External")
	require.NoError(t, err)
	assert.Equal(t, "https://acme.wd1.myworkdayjobs.com/wday/cxs/acme/External/jobs", ep)
}

func TestWorkdayEndpointInvalid(t *testing.T) {
	_, err := workdayEndpoint("https://acme.com/careers")
	assert.Error(t, err)
}
