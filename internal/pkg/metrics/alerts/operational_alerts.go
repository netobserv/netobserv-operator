package alerts

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func alertNoFlows() *monitoringv1.Rule {
	d := monitoringv1.Duration("10m")

	// Not receiving flows
	return &monitoringv1.Rule{
		Alert: string(flowslatest.AlertNoFlows),
		Annotations: map[string]string{
			"description": "NetObserv flowlogs-pipeline is not receiving any flow, this is either a connection issue with the agent, or an agent issue",
			"summary":     "NetObserv flowlogs-pipeline is not receiving any flow",
			"runbook_url": buildRunbookURL(flowslatest.AlertNoFlows),
		},
		Expr:   intstr.FromString("sum(rate(netobserv_ingest_flows_processed[1m])) == 0"),
		For:    &d,
		Labels: buildLabels(flowslatest.AlertNoFlows, "warning", false),
	}
}

func alertLokiError() *monitoringv1.Rule {
	d := monitoringv1.Duration("10m")

	return &monitoringv1.Rule{
		Alert: string(flowslatest.AlertLokiError),
		Annotations: map[string]string{
			"description": "NetObserv flowlogs-pipeline is dropping flows because of Loki errors, Loki may be down or having issues ingesting every flows. Please check Loki and flowlogs-pipeline logs.",
			"summary":     "NetObserv flowlogs-pipeline is dropping flows because of Loki errors",
			"runbook_url": buildRunbookURL(flowslatest.AlertLokiError),
		},
		Expr:   intstr.FromString("sum(rate(netobserv_loki_dropped_entries_total[1m])) > 0"),
		For:    &d,
		Labels: buildLabels(flowslatest.AlertLokiError, "warning", false),
	}
}
