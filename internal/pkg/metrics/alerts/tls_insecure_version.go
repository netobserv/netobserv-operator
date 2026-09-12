package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type tlsInsecureVersion struct {
	ctx *ruleContext
}

func newTLSInsecureVersion(ctx *ruleContext) HealthRule {
	return &tlsInsecureVersion{ctx: ctx}
}

func (r *tlsInsecureVersion) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "tls_insecure_version", "2m")
}

func (r *tlsInsecureVersion) GetAnnotations() (map[string]string, error) {
	healthAnnot := newHealthAnnotation(r.ctx)
	healthAnnot.TrafficLink = &trafficLink{
		BackAndForth:      true,
		ExtraFilter:       `tls_version="TLS 1.0,TLS 1.1,SSL 2.0,SSL 3.0"`,
		FilterDestination: false,
	}

	return map[string]string{
		"summary": "Insecure TLS version detected",
		"description": fmt.Sprintf(
			"NetObserv is detecting more than %s%% of traffic using insecure TLS versions (TLS 1.0, TLS 1.1, or SSL)%s.",
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(healthAnnot),
	}, nil
}

func (r *tlsInsecureVersion) Build() (*monitoringv1.Rule, error) {
	metric, totalMetric := getMetricsForRule(r.ctx)
	// Filter for insecure TLS versions: TLS 1.0, TLS 1.1, or SSL variants
	filter := getPromQLFilters(r.ctx, `TLSVersion=~"TLS 1\\.0|TLS 1\\.1|SSL.*"`)
	totalFilter := getPromQLFilters(r.ctx, `TLSVersion!=""`)
	metricsRate := promQLRateFromMetric(metric, "", filter, "2m", "")
	totalRate := promQLRateFromMetric(totalMetric, "", totalFilter, "2m", "")
	metricsSumBy := sumBy(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	totalSumBy := sumBy(totalRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	promql := percentagePromQL(metricsSumBy, totalSumBy, r.ctx.alertThreshold, r.ctx.upperThreshold, r.ctx.healthRule.LowVolumeThreshold)
	return createRule(r.ctx, r, promql)
}
