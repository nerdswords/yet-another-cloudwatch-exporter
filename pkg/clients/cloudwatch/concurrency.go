package cloudwatch

// ConcurrencyConfig configures how concurrency should be limited in a Cloudwatch API client. It allows
// one to pick between different limiter implementations: a single limit limiter, or one with a different limit per
// API call.
type ConcurrencyConfig struct {
	// PerAPIEnabled configures wether to have a limit per API call.
	PerAPIEnabled bool

	// SingleLimit configures the concurrency limit when using a single limiter for api calls.
	SingleLimit int

	// ListMetrics limits the number for ListMetrics API concurrent API calls.
	ListMetrics int

	// GetMetricData limits the number for GetMetricData API concurrent API calls.
	GetMetricData int

	// GetMetricStatistics limits the number for GetMetricStatistics API concurrent API calls.
	GetMetricStatistics int
}

// NewLimiter creates a new ConcurrencyLimiter, according to the ConcurrencyConfig.
func (cfg ConcurrencyConfig) NewLimiter() ConcurrencyLimiter {
	if cfg.PerAPIEnabled {
		return NewPerAPICallLimiter(cfg.ListMetrics, cfg.GetMetricData, cfg.GetMetricStatistics)
	} else {
		return NewSingleLimiter(cfg.SingleLimit)
	}
}

// perAPICallLimiter is a ConcurrencyLimiter that keeps a different concurrency limiter per different API call. This allows
// a more granular control of concurrency, allowing us to take advantage of different api limits. For example, ListMetrics
// has a limit of 25 TPS, while GetMetricData has none.
type perAPICallLimiter map[string]chan struct{}

// NewPerAPICallLimiter creates a new PerAPICallLimiter.
func NewPerAPICallLimiter(listMetrics, getMetricData, getMetricStatistics int) ConcurrencyLimiter {
	return perAPICallLimiter(map[string]chan struct{}{
		listMetricsCall:         make(chan struct{}, listMetrics),
		getMetricDataCall:       make(chan struct{}, getMetricData),
		getMetricStatisticsCall: make(chan struct{}, getMetricStatistics),
	})
}

func (l perAPICallLimiter) Acquire(op string) {
	l[op] <- struct{}{}
}

func (l perAPICallLimiter) Release(op string) {
	<-l[op]
}

// singleLimiter is the current implementation of ConcurrencyLimiter, which has a single limit for all different API calls.
type singleLimiter struct {
	sem chan struct{}
}

// NewSingleLimiter creates a new SingleLimiter.
func NewSingleLimiter(limit int) ConcurrencyLimiter {
	return &singleLimiter{sem: make(chan struct{}, limit)}
}

func (sl *singleLimiter) Acquire(_ string) {
	sl.sem <- struct{}{}
}

func (sl *singleLimiter) Release(_ string) {
	<-sl.sem
}
