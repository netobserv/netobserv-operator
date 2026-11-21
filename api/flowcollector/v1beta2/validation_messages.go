package v1beta2

// Validation warning messages
const (
	// MetricsIncludeListWarning is shown when both includeList and additionalIncludeList are set
	MetricsIncludeListWarning = "Both spec.processor.metrics.includeList and spec.processor.metrics.additionalIncludeList are set. " +
		"When includeList is set, it replaces the default metrics entirely, and additionalIncludeList is ignored. " +
		"Use includeList to override defaults, or use additionalIncludeList alone to append to defaults."

	// MetricsAuthWithoutTLSWarning is shown when a metrics endpoint requires authentication but isn't encrypted.
	// It takes the metrics server path as a single argument, repeated.
	MetricsAuthWithoutTLSWarning = "%[1]s.auth.type is set to Kubernetes while %[1]s.tls.type is Disabled, " +
		"so the Prometheus scraper sends its bearer token over an unencrypted connection. " +
		"Set %[1]s.tls.type to Auto (on OpenShift) or Provided to encrypt that endpoint."

	// MetricsAuthScraperRBACWarning is shown when metrics authentication is enabled outside of OpenShift,
	// where the scraper isn't granted the /metrics non-resource URL out of the box.
	// It takes the metrics server path as a single argument.
	MetricsAuthScraperRBACWarning = "%s.auth.type is set to Kubernetes: the service account of your Prometheus " +
		"must be granted \"get\" on the \"/metrics\" non-resource URL, else scraping fails with 403. " +
		"The netobserv-metrics-scraper cluster role is provided for that purpose, but isn't bound to anything by default."

	// MetricsAuthWithoutAgentMetricsWarning is shown when agent metrics authentication is configured
	// while agent metrics are disabled altogether.
	MetricsAuthWithoutAgentMetricsWarning = "spec.agent.ebpf.metrics.server.auth.type is set to Kubernetes, " +
		"but spec.agent.ebpf.metrics.enable is false, so it has no effect."

	// MetricsAuthEnvOverrideWarning is shown when the agent metrics auth is overridden via advanced env.
	MetricsAuthEnvOverrideWarning = "METRICS_AUTH_TYPE is overridden in spec.agent.ebpf.advanced.env. " +
		"The operator still configures the ServiceMonitor and the RBAC from spec.agent.ebpf.metrics.server.auth, " +
		"so an inconsistent override breaks metrics scraping."
)
