package scrape

import "context"

type WeWorkRemotely struct{ BaseURL string }

func (w WeWorkRemotely) Name() string                              { return "weworkremotely" }
func (w WeWorkRemotely) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
