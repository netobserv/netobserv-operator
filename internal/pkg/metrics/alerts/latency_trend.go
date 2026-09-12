package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type latencyTrend struct {
	ctx *ruleContext
}

func newLatencyTrend(ctx *ruleContext) HealthRule {
	return &latencyTrend{ctx: ctx}
}

func (r *latencyTrend) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "tcp_latency_increase_p90", "2m")
}

func (r *latencyTrend) GetAnnotations() (map[string]string, error) {
	offset, _ := r.ctx.healthRule.GetTrendParams()
	healthAnnot := newHealthAnnotation(r.ctx)
	healthAnnot.CloseOpenScale(r.ctx, 5)

	return map[string]string{
		"summary": "TCP latency increase",
		"description": fmt.Sprintf(
			"NetObserv is detecting TCP latency increased by more than %s%%%s, compared to baseline (offset: %s).",
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
			offset,
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(healthAnnot),
	}, nil
}

func (r *latencyTrend) Build() (*monitoringv1.Rule, error) {
	offset, duration := r.ctx.healthRule.GetTrendParams()

	metric, baseline := getMetricsForRule(r.ctx)
	filter := getPromQLFilters(r.ctx, "")
	metricsRate := promQLRateFromMetric(metric, "_bucket", filter, "2m", "")
	baselineRate := promQLRateFromMetric(baseline, "_bucket", filter, duration, " offset "+offset)
	metricQuantile := histogramQuantile(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "0.9")
	baselineQuantile := histogramQuantile(baselineRate, r.ctx.healthRule.GroupBy, r.ctx.side, "0.9")
	promql := baselineIncreasePromQL(metricQuantile, baselineQuantile, r.ctx.alertThreshold, r.ctx.upperThreshold)
	return createRule(r.ctx, r, promql)
}
