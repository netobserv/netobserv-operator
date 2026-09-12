package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type kernelDrops struct {
	ctx *ruleContext
}

func newKernelDrops(ctx *ruleContext) HealthRule {
	return &kernelDrops{ctx: ctx}
}

func (r *kernelDrops) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "packet_drops_kernel", "2m")
}

func (r *kernelDrops) GetAnnotations() (map[string]string, error) {
	return map[string]string{
		"summary": "Too many packets dropped by the kernel",
		"description": fmt.Sprintf(
			"NetObserv is detecting more than %s%% of packets dropped by the kernel%s.",
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(newHealthAnnotation(r.ctx)),
	}, nil
}

func (r *kernelDrops) Build() (*monitoringv1.Rule, error) {
	metric, totalMetric := getMetricsForRule(r.ctx)
	filter := getPromQLFilters(r.ctx, "")
	metricsRate := promQLRateFromMetric(metric, "", filter, "2m", "")
	totalRate := promQLRateFromMetric(totalMetric, "", filter, "2m", "")
	metricsSumBy := sumBy(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	totalSumBy := sumBy(totalRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	promql := percentagePromQL(metricsSumBy, totalSumBy, r.ctx.alertThreshold, r.ctx.upperThreshold, r.ctx.healthRule.LowVolumeThreshold)
	return createRule(r.ctx, r, promql)
}
