package scrape

import "context"

type Jobicy struct{ BaseURL string }

func (j Jobicy) Name() string                              { return "jobicy" }
func (j Jobicy) FetchJobs(_ context.Context) ([]BoardJob, error) { return nil, nil }
