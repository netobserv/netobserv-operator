package alerts

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type externalTrend struct {
	ctx     *ruleContext
	ingress bool
}

func newExternalTrend(ctx *ruleContext, ingress bool) HealthRule {
	return &externalTrend{ctx: ctx, ingress: ingress}
}

func (r *externalTrend) RecordingName() string {
	name := "external_egress"
	if r.ingress {
		name = "external_ingress"
	}
	return buildRecordingRuleName(r.ctx, name, "2m")
}

func (r *externalTrend) GetAnnotations() (map[string]string, error) {
	direction := "egress"
	trafficLinkFilter := `dst_subnet_label="",EXT:`
	if r.ingress {
		direction = "ingress"
		trafficLinkFilter = `src_subnet_label="",EXT:`
	}
	offset, _ := r.ctx.healthRule.GetTrendParams()
	healthAnnot := newHealthAnnotation(r.ctx)
	healthAnnot.CloseOpenScale(r.ctx, 5)
	healthAnnot.TrafficLink = &trafficLink{
		BackAndForth:      true,
		ExtraFilter:       trafficLinkFilter,
		FilterDestination: r.ctx.side == asDest,
	}

	return map[string]string{
		"summary": fmt.Sprintf("External %s traffic increase", direction),
		"description": fmt.Sprintf(
			"NetObserv is detecting external %s traffic increased by more than %s%%%s, compared to baseline (offset: %s).",
			direction,
			r.ctx.getLowestThreshold(),
			getAlertLegend(r.ctx),
			offset,
		),
		"runbook_url":       BuildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(healthAnnot),
	}, nil
}

func (r *externalTrend) Build() (*monitoringv1.Rule, error) {
	// Don't create ingress for asSource, or egress for asDestination
	if r.ctx.side == asSource && r.ingress {
		return nil, nil
	} else if r.ctx.side == asDest && !r.ingress {
		return nil, nil
	}
	filterForExternal := `DstSubnetLabel=~"|EXT:.*",DstK8S_Namespace="",DstK8S_OwnerName=""`
	if r.ingress {
		filterForExternal = `SrcSubnetLabel=~"|EXT:.*",SrcK8S_Namespace="",SrcK8S_OwnerName=""`
	}
	offset, duration := r.ctx.healthRule.GetTrendParams()

	metric, baseline := getMetricsForRule(r.ctx)
	filter := getPromQLFilters(r.ctx, filterForExternal)
	metricsRate := promQLRateFromMetric(metric, "", filter, "2m", "")
	baselineRate := promQLRateFromMetric(baseline, "", filter, duration, " offset "+offset)
	metricsSumBy := sumBy(metricsRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	baselineSumBy := sumBy(baselineRate, r.ctx.healthRule.GroupBy, r.ctx.side, "")
	promql := baselineIncreasePromQL(metricsSumBy, baselineSumBy, r.ctx.alertThreshold, r.ctx.upperThreshold)
	return createRule(r.ctx, r, promql)
}
