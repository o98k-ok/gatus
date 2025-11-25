package endpoint

// ConditionResult result of a Condition
type ConditionResult struct {
	// Condition that was evaluated
	Condition string `json:"condition"`

	// Success whether the condition was met (successful) or not (failed)
	Success bool `json:"success"`

	// Value stores the resolved value from the left side of the condition
	// This is populated for all conditions, enabling metric tracking without special handling
	Value string `json:"value,omitempty"`
}
