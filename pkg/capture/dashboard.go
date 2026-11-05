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
		Data: map[string]string{"on-demand-netobserv-flow-metrics.json": dashboards.CreateDashboard("NetObserv / On Demand", dashboards.OnDemandRows())},
	}
}

// BuildFlowCaptureDashboard returns the shared dashboard for operator-managed
// FlowCaptures. It is intentionally independent from the capture type: health
// metrics are exposed by every capture agent, while traffic panels populate
// when the capture type produces Prometheus metrics.
func BuildFlowCaptureDashboard() *corev1.ConfigMap {
	dashboard := dashboards.Dashboard{
		Title: "NetObserv / FlowCaptures",
		Variables: []dashboards.Variable{{
			Name:       "flowcapture",
			Label:      "FlowCapture",
			Type:       "query",
			Query:      `label_values(up{flowcapture!=""}, flowcapture)`,
			Datasource: "prometheus",
			Refresh:    1,
			Multi:      true,
			IncludeAll: true,
			AllValue:   ".*",
		}},
		Rows: append([]*dashboards.Row{
			dashboards.NewRow("Capture health", false, "250px", []dashboards.Panel{
				dashboards.NewPanel("Agent targets up", dashboards.ChartTypeSingleStat, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(up{flowcapture=~"$flowcapture"})`, "agents")),
				dashboards.NewPanel("Agent errors", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(rate(flowcapture_agent_errors_total{flowcapture=~"$flowcapture"}[1m])) by (flowcapture,component,error)`, "{{flowcapture}} {{component}} {{error}}")),
				dashboards.NewPanel("Evicted flows", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(rate(flowcapture_agent_evicted_flows_total{flowcapture=~"$flowcapture"}[1m])) by (flowcapture,source,reason)`, "{{flowcapture}} {{source}} {{reason}}")),
				dashboards.NewPanel("Agent buffer size", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(flowcapture_agent_buffer_size{flowcapture=~"$flowcapture"}) by (flowcapture,name)`, "{{flowcapture}} {{name}}")),
			}),
			dashboards.NewRow("Metrics capture activity", false, "250px", []dashboards.Panel{
				dashboards.NewPanel("Processed flows", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(rate(on_demand_netobserv_metrics_processed{flowcapture=~"$flowcapture"}[1m])) by (flowcapture)`, "{{flowcapture}}")),
				dashboards.NewPanel("Reported metrics", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(rate(on_demand_netobserv_encode_prom_metrics_reported{flowcapture=~"$flowcapture"}[1m])) by (flowcapture)`, "{{flowcapture}}")),
				dashboards.NewPanel("Dropped metrics", dashboards.ChartTypeLine, dashboards.Unit(""), 4,
					dashboards.NewTarget(`sum(rate(on_demand_netobserv_metrics_dropped{flowcapture=~"$flowcapture"}[1m])) by (flowcapture)`, "{{flowcapture}}")),
			}),
		}, dashboards.OnDemandRows()...),
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ConfigMap"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "netobserv-flowcaptures",
			Namespace: "openshift-config-managed",
			Labels:    map[string]string{"console.openshift.io/dashboard": "true"},
			Annotations: map[string]string{
				"flows.netobserv.io/managed": "flowcapture",
			},
		},
		Data: map[string]string{"flowcaptures.json": dashboard.ToGrafanaJSON()},
	}
}
