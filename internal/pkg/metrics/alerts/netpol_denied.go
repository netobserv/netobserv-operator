package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type netpolDenied struct {
	ctx *ruleContext
}

func newNetpolDenied(ctx *ruleContext) HealthRule {
	return &netpolDenied{ctx: ctx}
}

func (r *netpolDenied) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "netpol_denied", "2m")
}

func (r *netpolDenied) GetAnnotations() (map[string]string, error) {
	return map[string]string{
		"summary": "Traffic denied by Network Policies",
		"description": fmt.Sprintf(
			"NetObserv is detecting more than %s%% of denied traffic due to Network Policies%s.",
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(newHealthAnnotation(r.ctx)),
	}, nil
}

func (r *netpolDenied) Build() (*monitoringv1.Rule, error) {
	metric, totalMetric := getMetricsForRule(r.ctx)
	filter := getPromQLFilters(r.ctx, `action="drop"`)
	totalFilter := getPromQLFilters(r.ctx, "")
	metricsRate := promQLRateFromMetric(metric, "", filter, "2m", "")
	totalRate := promQLRateFromMetric(totalMetric, "", totalFilter, "2m", "")
	metricsSumBy := sumBy(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	totalSumBy := sumBy(totalRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	promql := percentagePromQL(metricsSumBy, totalSumBy, r.ctx.alertThreshold, r.ctx.upperThreshold, r.ctx.healthRule.LowVolumeThreshold)
	return createRule(r.ctx, r, promql)
}
