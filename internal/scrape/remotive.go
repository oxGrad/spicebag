package scrape

import "context"

type Remotive struct{ BaseURL string }

func (r Remotive) Name() string                              { return "remotive" }
func (r Remotive) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
