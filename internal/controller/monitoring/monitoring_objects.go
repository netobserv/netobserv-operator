package monitoring

import (
	"regexp"
	"strings"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/pkg/dashboards"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	downstreamLabelKey    = "openshift.io/cluster-monitoring"
	downstreamLabelValue  = "true"
	dashboardCMNamespace  = "openshift-config-managed"
	dashboardCMAnnotation = "console.openshift.io/dashboard"

	flowDashboardCMFile = "netobserv-flow-metrics.json"

	healthDashboardCMName = "grafana-dashboard-netobserv-health"
	healthDashboardCMFile = "netobserv-health-metrics.json"
)

var k8sInvalidChar = regexp.MustCompile(`[^a-z0-9\-]`)

func buildNamespace(ns string, isDownstream bool) *corev1.Namespace {
	labels := map[string]string{}
	if isDownstream {
		labels[downstreamLabelKey] = downstreamLabelValue
	}
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   ns,
			Labels: labels,
		},
	}
}

func buildFlowMetricsDashboards(metrics []metricslatest.FlowMetric) []*corev1.ConfigMap {
	var cms []*corev1.ConfigMap
	dash := dashboards.CreateFlowMetricsDashboards(metrics)

	for name, json := range dash {
		k8sName := "netobserv-" + k8sInvalidChar.ReplaceAllString(strings.ToLower(name), "-")
		configMap := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      k8sName,
				Namespace: dashboardCMNamespace,
				Labels: map[string]string{
					dashboardCMAnnotation: "true",
				},
			},
			Data: map[string]string{
				flowDashboardCMFile: json,
			},
		}
		cms = append(cms, &configMap)
	}
	return cms
}

func buildHealthDashboard(namespace, nsFlowsMetric string) (*corev1.ConfigMap, bool, error) {
	dashboard, err := dashboards.CreateHealthDashboard(namespace, nsFlowsMetric)
	if err != nil {
		return nil, false, err
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      healthDashboardCMName,
			Namespace: dashboardCMNamespace,
			Labels: map[string]string{
				dashboardCMAnnotation: "true",
			},
		},
		Data: map[string]string{
			healthDashboardCMFile: dashboard,
		},
	}
	return &configMap, len(dashboard) == 0, nil
}
