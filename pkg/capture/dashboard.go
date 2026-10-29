package capture

import (
	"github.com/netobserv/netobserv-operator/pkg/dashboards"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BuildMetricsDashboard returns the CLI on-demand metrics dashboard as a
// typed ConfigMap generated from the shared dashboard model.
func BuildMetricsDashboard() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsResourceName,
			Namespace: "openshift-config-managed",
			Labels:    map[string]string{"console.openshift.io/dashboard": "true"},
		},
		Data: map[string]string{"on-demand-netobserv-flow-metrics.json": dashboards.CreateOnDemandDashboard()},
	}
}
