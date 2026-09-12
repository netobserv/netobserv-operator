package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type dnsNxDomain struct {
	ctx *ruleContext
}

func newDNSNxDomain(ctx *ruleContext) HealthRule {
	return &dnsNxDomain{ctx: ctx}
}

func (r *dnsNxDomain) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "dns_nxdomain", "2m")
}

func (r *dnsNxDomain) GetAnnotations() (map[string]string, error) {
	healthAnnot := newHealthAnnotation(r.ctx)
	healthAnnot.TrafficLink = &trafficLink{
		BackAndForth:      false,
		ExtraFilter:       `dns_flag_response_code="NXDomain"`,
		FilterDestination: true,
	}

	return map[string]string{
		"summary": "Too many DNS NX_DOMAIN errors",
		"description": fmt.Sprintf(
			"NetObserv is detecting more than %s%% of DNS NX_DOMAIN errors%s.",
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(healthAnnot),
	}, nil
}

func (r *dnsNxDomain) Build() (*monitoringv1.Rule, error) {
	// DNS errors are in return traffic only
	if r.ctx.side == asSource {
		return nil, nil
	}

	metric, totalMetric := getMetricsForRule(r.ctx)
	filter := getPromQLFilters(r.ctx, `DnsFlagsResponseCode="NXDomain"`)
	totalFilter := getPromQLFilters(r.ctx, "")
	metricsRate := promQLRateFromMetric(metric, "", filter, "2m", "")
	totalRate := promQLRateFromMetric(totalMetric, "", totalFilter, "2m", "")
	metricsSumBy := sumBy(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	totalSumBy := sumBy(totalRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	promql := percentagePromQL(metricsSumBy, totalSumBy, r.ctx.alertThreshold, r.ctx.upperThreshold, r.ctx.healthRule.LowVolumeThreshold)
	return createRule(r.ctx, r, promql)
}
