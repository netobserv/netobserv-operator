package helper

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
)

func serviceMonitorWithToken(token string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				Port:            "prometheus",
				Interval:        "15s",
				BearerTokenFile: token,
			}},
		},
	}
}

// Turning metrics authentication off clears bearerTokenFile. DeepDerivative ignores the fields that
// are zero-valued in the desired object, so this case would go unnoticed and the live ServiceMonitor
// would keep scraping with a token that the endpoint no longer expects.
func TestServiceMonitorChanged_BearerTokenRemoved(t *testing.T) {
	assert := assert.New(t)

	old := serviceMonitorWithToken("/var/run/secrets/kubernetes.io/serviceaccount/token")
	n := serviceMonitorWithToken("")

	report := NewChangeReport("")
	assert.True(ServiceMonitorChanged(old, n, &report))
}

func TestServiceMonitorChanged_BearerTokenAdded(t *testing.T) {
	assert := assert.New(t)

	old := serviceMonitorWithToken("")
	n := serviceMonitorWithToken("/var/run/secrets/kubernetes.io/serviceaccount/token")

	report := NewChangeReport("")
	assert.True(ServiceMonitorChanged(old, n, &report))
}

// Guards against the exact endpoints comparison causing an endless update loop.
func TestServiceMonitorChanged_Unchanged(t *testing.T) {
	assert := assert.New(t)

	for _, token := range []string{"", "/var/run/secrets/kubernetes.io/serviceaccount/token"} {
		report := NewChangeReport("")
		assert.False(ServiceMonitorChanged(serviceMonitorWithToken(token), serviceMonitorWithToken(token), &report))
	}
}
