package endpoint

import (
	"errors"
	"strings"
)

// Metric defines which condition value to track as a metric over time
type Metric struct {
	// Name is the display name for this metric
	Name string `yaml:"name" json:"name"`

	// Value is the pattern to match against conditions
	// Example: "[BODY].cpu_usage" will match any condition containing this pattern
	Value string `yaml:"value" json:"value"`

	// Unit is the measurement unit for display purposes
	// Examples: "percent", "ms", "count", "MB"
	Unit string `yaml:"unit,omitempty" json:"unit,omitempty"`
}

// Validate checks if the metric configuration is valid
func (m *Metric) Validate() error {
	if m.Name == "" {
		return errors.New("metric name cannot be empty")
	}
	if m.Value == "" {
		return errors.New("metric value pattern cannot be empty")
	}
	if !strings.HasPrefix(m.Value, "[") {
		return errors.New("metric value must be a placeholder pattern (e.g., [BODY].field)")
	}
	return nil
}
