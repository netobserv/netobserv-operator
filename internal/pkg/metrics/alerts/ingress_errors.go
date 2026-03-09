package alerts

import (
	"fmt"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type ingressErrors struct {
	ctx *ruleContext
}

func newIngressErrors(ctx *ruleContext) HealthRule {
	return &ingressErrors{ctx: ctx}
}

func (r *ingressErrors) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "ingress_5xx_errors", "2m")
}

func (r *ingressErrors) GetAnnotations() (map[string]string, error) {
	var legend string
	switch r.ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		return nil, fmt.Errorf("Ingress5xxErrors health rule does not support grouping per node")
	case flowslatest.GroupByNamespace:
		// Note: we'll rename exported_namespace to namespace in the PromQL using label_replace
		legend = " [namespace={{ $labels.namespace }}]"
	case flowslatest.GroupByWorkload:
		return nil, fmt.Errorf("Ingress5xxErrors health rule does not support grouping per workload")
	}

	return map[string]string{
		"summary": "Too many ingress 5xx errors",
		"description": fmt.Sprintf(
			"HAProxy is reporting more than %s%% of 5xx HTTP response codes from ingress traffic%s.",
			r.ctx.getLowestThreshold(),
			legend,
		),
		"runbook_url":       buildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(newHealthAnnotation(r.ctx)),
	}, nil
}

func (r *ingressErrors) Build() (*monitoringv1.Rule, error) {
	// No "side" consideration for ingress metrics, so keep only 1 call from the two of them
	if r.ctx.side == asDest {
		return nil, nil
	}

	// Build PromQL with label_replace to rename exported_namespace to namespace
	var errorsQuery, totalQuery string
	if r.ctx.healthRule.GroupBy == flowslatest.GroupByNamespace {
		// Rename exported_namespace to namespace for console plugin compatibility
		errorsQuery = `sum(label_replace(rate(haproxy_server_http_responses_total{code="5xx"}[2m]), "namespace", "$1", "exported_namespace", "(.*)")) by (namespace)`
		totalQuery = `sum(label_replace(rate(haproxy_server_http_responses_total[2m]), "namespace", "$1", "exported_namespace", "(.*)")) by (namespace)`
	} else {
		// Global (no groupBy)
		errorsQuery = `sum(rate(haproxy_server_http_responses_total{code="5xx"}[2m]))`
		totalQuery = `sum(rate(haproxy_server_http_responses_total[2m]))`
	}
	promql := percentagePromQL(errorsQuery, totalQuery, r.ctx.alertThreshold, r.ctx.upperThreshold, r.ctx.healthRule.LowVolumeThreshold)
	return createRule(r.ctx, r, promql)
}
