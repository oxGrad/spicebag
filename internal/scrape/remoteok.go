package scrape

import "context"

type RemoteOK struct{ BaseURL string }

func (r RemoteOK) Name() string                              { return "remoteok" }
func (r RemoteOK) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
