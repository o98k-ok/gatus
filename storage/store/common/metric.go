package common

import "errors"

// MetricHistory contains historical values for a metric
type MetricHistory struct {
	Timestamps []int64
	Values     []string
}

var (
	// ErrMetricNotConfigured is returned when an endpoint has no metric configured
	ErrMetricNotConfigured = errors.New("endpoint has no metric configured")
)
