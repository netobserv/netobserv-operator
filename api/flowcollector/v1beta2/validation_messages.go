package v1beta2

// Validation warning messages
const (
	// MetricsIncludeListWarning is shown when both includeList and additionalIncludeList are set
	MetricsIncludeListWarning = "Both spec.processor.metrics.includeList and spec.processor.metrics.additionalIncludeList are set. " +
		"When includeList is set, it replaces the default metrics entirely, and additionalIncludeList is ignored. " +
		"Use includeList to override defaults, or use additionalIncludeList alone to append to defaults."
)
