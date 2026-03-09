package alerts

import (
	"fmt"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type deviceDrops struct {
	ctx *ruleContext
}

func newDeviceDrops(ctx *ruleContext) HealthRule {
	return &deviceDrops{ctx: ctx}
}

func (r *deviceDrops) RecordingName() string {
	return buildRecordingRuleName(r.ctx, "packet_drops_device", "2m")
}

func (r *deviceDrops) GetAnnotations() (map[string]string, error) {
	healthAnnot := newHealthAnnotation(r.ctx)
	var legend string
	switch r.ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		healthAnnot.NodeLabels = []string{"instance"}
		legend = " [node={{ $labels.instance }}]"
	case flowslatest.GroupByNamespace:
		return nil, fmt.Errorf("PacketDropsByDevice health rule does not support grouping per namespace")
	case flowslatest.GroupByWorkload:
		return nil, fmt.Errorf("PacketDropsByDevice health rule does not support grouping per workload")
	}

	return map[string]string{
		"summary": "Too many drops from device",
		"description": fmt.Sprintf(
			"node-exporter is reporting more than %s%% of dropped packets%s.",
			r.ctx.getLowestThreshold(),
			legend,
		),
		"runbook_url":       buildRunbookURL(r.ctx.template),
		healthAnnotationKey: encodeHealthAnnotation(healthAnnot),
	}, nil
}

func (r *deviceDrops) Build() (*monitoringv1.Rule, error) {
	// No "side" consideration on netdev metrics, so keep only 1 call from the two of them
	if r.ctx.side == asDest {
		return nil, nil
	}
	// Override (unset) side
	r.ctx.side = ""

	var byLabels string
	switch r.ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		byLabels = " by (instance)"
	case flowslatest.GroupByNamespace:
		return nil, fmt.Errorf("PacketDropsByDevice health rule does not support grouping per namespace")
	case flowslatest.GroupByWorkload:
		return nil, fmt.Errorf("PacketDropsByDevice health rule does not support grouping per workload")
	}

	promql := percentagePromQL(
		fmt.Sprintf("sum(rate(node_network_receive_drop_total[2m]))%s + sum(rate(node_network_transmit_drop_total[2m]))%s", byLabels, byLabels),
		fmt.Sprintf("sum(rate(node_network_receive_packets_total[2m]))%s + sum(rate(node_network_transmit_packets_total[2m]))%s", byLabels, byLabels),
		r.ctx.alertThreshold,
		r.ctx.upperThreshold,
		r.ctx.healthRule.LowVolumeThreshold,
	)

	return createRule(r.ctx, r, promql)
}
