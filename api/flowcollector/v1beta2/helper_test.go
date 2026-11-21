package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsMetricsAuthEnabled(t *testing.T) {
	assert := assert.New(t)

	for _, tc := range []struct {
		name     string
		authType MetricsAuthType
		expected bool
	}{
		// Unset is treated as disabled, so that resources created before this field existed keep
		// their previous behaviour even when they bypass the API server defaulting.
		{name: "unset", authType: "", expected: false},
		{name: "disabled", authType: MetricsAuthDisabled, expected: false},
		{name: "kubernetes", authType: MetricsAuthKubernetes, expected: true},
	} {
		t.Run(tc.name, func(_ *testing.T) {
			spec := FlowCollectorSpec{}
			spec.Processor.Metrics.Server.Auth.Type = tc.authType
			spec.Agent.EBPF.Metrics.Server.Auth.Type = tc.authType

			assert.Equal(tc.expected, spec.Processor.Metrics.Server.IsMetricsAuthEnabled(), tc.name)
			assert.Equal(tc.expected, spec.Agent.EBPF.Metrics.Server.IsMetricsAuthEnabled(), tc.name)
		})
	}
}

// The two metrics servers share the same type but are configured independently.
func TestIsMetricsAuthEnabled_Independent(t *testing.T) {
	assert := assert.New(t)

	spec := FlowCollectorSpec{}
	spec.Processor.Metrics.Server.Auth.Type = MetricsAuthKubernetes
	spec.Agent.EBPF.Metrics.Server.Auth.Type = MetricsAuthDisabled

	assert.True(spec.Processor.Metrics.Server.IsMetricsAuthEnabled())
	assert.False(spec.Agent.EBPF.Metrics.Server.IsMetricsAuthEnabled())
}
