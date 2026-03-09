package alerts

import (
	"math"
	"strconv"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

type srcOrDst string

const (
	asSource srcOrDst = "Src"
	asDest   srcOrDst = "Dst"
)

type ruleContext struct {
	template            flowslatest.HealthRuleTemplate
	healthRule          *flowslatest.HealthRuleVariant
	mode                flowslatest.HealthRuleMode
	enabledMetrics      []string
	side                srcOrDst
	severity            string
	alertThreshold      string
	recordingThresholds *recordingThresholds
	upperThreshold      string
	duration            monitoringv1.Duration
}

type healthAnnotation struct {
	AlertThreshold      string               `json:"alertThreshold,omitempty"`
	RecordingThresholds *recordingThresholds `json:"recordingThresholds,omitempty"`
	UpperBound          string               `json:"upperBound,omitempty"`
	Unit                string               `json:"unit,omitempty"`
	NodeLabels          []string             `json:"nodeLabels,omitempty"`
	NamespaceLabels     []string             `json:"namespaceLabels,omitempty"`
	WorkloadLabels      []string             `json:"workloadLabels,omitempty"`
	KindLabels          []string             `json:"kindLabels,omitempty"`
	TrafficLink         *trafficLink         `json:"trafficLink,omitempty"`
}

type recordingThresholds struct {
	Info     string `json:"info,omitempty"`
	Warning  string `json:"warning,omitempty"`
	Critical string `json:"critical,omitempty"`
}

type trafficLink struct {
	ExtraFilter       string `json:"extraFilter"`
	BackAndForth      bool   `json:"backAndForth"`
	FilterDestination bool   `json:"filterDestination"`
}

func (ctx *ruleContext) getLowestThreshold() string {
	if ctx.alertThreshold != "" {
		return ctx.alertThreshold
	}
	if ctx.recordingThresholds != nil {
		if ctx.recordingThresholds.Info != "" {
			return ctx.recordingThresholds.Info
		}
		if ctx.recordingThresholds.Warning != "" {
			return ctx.recordingThresholds.Warning
		}
		return ctx.recordingThresholds.Critical
	}
	return ""
}

func (ctx *ruleContext) getHighestThreshold() string {
	if ctx.alertThreshold != "" {
		return ctx.alertThreshold
	}
	if ctx.recordingThresholds != nil {
		if ctx.recordingThresholds.Critical != "" {
			return ctx.recordingThresholds.Critical
		}
		if ctx.recordingThresholds.Warning != "" {
			return ctx.recordingThresholds.Warning
		}
		return ctx.recordingThresholds.Info
	}
	return ""
}

func newHealthAnnotation(ctx *ruleContext) *healthAnnotation {
	// The health annotation contains json-encoded information used in console plugin display
	annotation := healthAnnotation{
		AlertThreshold:      ctx.alertThreshold,
		RecordingThresholds: ctx.recordingThresholds,
		Unit:                "%",
	}
	switch ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		annotation.NodeLabels = []string{"node"}
	case flowslatest.GroupByNamespace:
		annotation.NamespaceLabels = []string{"namespace"}
	case flowslatest.GroupByWorkload:
		annotation.NamespaceLabels = []string{"namespace"}
		annotation.WorkloadLabels = []string{"workload"}
		annotation.KindLabels = []string{"kind"}
	}
	return &annotation
}

func (h *healthAnnotation) CloseOpenScale(ctx *ruleContext, factor float64) {
	// trending comparison are on an open scale; but in the health page, we need a closed scale to compute the score
	// let's set an upper bound to max(5*threshold, 100) so score can be computed after clamping
	th := ctx.getHighestThreshold()
	val, err := strconv.ParseFloat(th, 64)
	if err != nil {
		val = 100
	}
	h.UpperBound = strconv.Itoa(int(math.Max(val*factor, 100)))
}

// buildRecordingRuleName builds recording rule name following the convention:
// netobserv:health:<template>:<groupby>:<side>:rate2m
func buildRecordingRuleName(ctx *ruleContext, prefix, rateInterval string) string {
	if ctx.mode != flowslatest.ModeRecording {
		return ""
	}
	parts := []string{"netobserv", "health", prefix}

	// Add groupBy if present
	if ctx.healthRule.GroupBy != "" {
		parts = append(parts, strings.ToLower(string(ctx.healthRule.GroupBy)))
	}

	// Add side if groupBy is present (side is only relevant with groupBy)
	if ctx.healthRule.GroupBy != "" && ctx.side != "" {
		parts = append(parts, strings.ToLower(string(ctx.side)))
	}

	// Add rate interval (rate2m for 2m window)
	parts = append(parts, "rate"+rateInterval)

	return strings.Join(parts, ":")
}

func getAlertLegend(ctx *ruleContext) string {
	var sideText string
	switch ctx.side {
	case asSource:
		sideText = "source "
	case asDest:
		sideText = "dest. "
	}
	switch ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		return " [" + sideText + "node={{ $labels.node }}]"
	case flowslatest.GroupByNamespace:
		return " [" + sideText + "namespace={{ $labels.namespace }}]"
	case flowslatest.GroupByWorkload:
		return " [" + sideText + "workload={{ $labels.workload }} ({{ $labels.kind }})]"
	}
	return ""
}

func getPromQLFilters(ctx *ruleContext, additionalFilter string) string {
	var filters []string

	// Build label matchers to filter out metrics where K8s labels don't exist or are empty
	// This prevents alerts from firing with empty namespace/workload/node labels
	switch ctx.healthRule.GroupBy {
	case flowslatest.GroupByNode:
		filters = append(filters, string(ctx.side)+`K8S_HostName!=""`)
	case flowslatest.GroupByNamespace:
		filters = append(filters, string(ctx.side)+`K8S_Namespace!=""`)
	case flowslatest.GroupByWorkload:
		filters = append(filters, string(ctx.side)+`K8S_Namespace!=""`)
		filters = append(filters, string(ctx.side)+`K8S_OwnerName!=""`)
		filters = append(filters, string(ctx.side)+`K8S_OwnerType!=""`)
	}

	// Add additional business logic filters
	if additionalFilter != "" {
		filters = append(filters, additionalFilter)
	}

	if len(filters) == 0 {
		return ""
	}
	return "{" + strings.Join(filters, ",") + "}"
}

func getMetricsForRule(ctx *ruleContext) (string, string) {
	var reqMetric1, reqMetric2 string
	reqMetrics1, reqMetrics2 := flowslatest.GetElligibleMetricsForAlert(ctx.template, ctx.healthRule)
	if len(reqMetrics1) > 0 {
		reqMetric1 = flowslatest.GetFirstRequiredMetrics(reqMetrics1, ctx.enabledMetrics)
	}
	if len(reqMetrics2) > 0 {
		reqMetric2 = flowslatest.GetFirstRequiredMetrics(reqMetrics2, ctx.enabledMetrics)
	}
	return reqMetric1, reqMetric2
}
