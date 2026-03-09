package alerts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	healthAnnotationKey = "netobserv_io_network_health"
	runbookURLBase      = "https://github.com/openshift/runbooks/blob/master/alerts/network-observability-operator"
)

type HealthRule interface {
	RecordingName() string
	GetAnnotations() (map[string]string, error)
	Build() (*monitoringv1.Rule, error)
}

func encodeHealthAnnotation(hann *healthAnnotation) string {
	// The health annotation contains json-encoded information used in console plugin display
	bAnnot, _ := json.Marshal(hann)
	return string(bAnnot)
}

func BuildMonitoringRules(ctx context.Context, fc *flowslatest.FlowCollectorSpec) []monitoringv1.Rule {
	log := log.FromContext(ctx)
	rules := []monitoringv1.Rule{}

	healthRules, err := BuildHealthRules(fc)
	if err != nil {
		log.Error(err, "Can't build some health rules")
		// do not return: other rules might have been created
	}
	for _, hr := range healthRules {
		if mr, err := hr.Build(); err != nil {
			log.Error(err, "Can't build a monitoring rule")
		} else if mr != nil {
			rules = append(rules, *mr)
		}
	}

	if !slices.Contains(fc.Processor.Metrics.DisableAlerts, flowslatest.AlertNoFlows) {
		r := alertNoFlows()
		rules = append(rules, *r)
	}
	if !slices.Contains(fc.Processor.Metrics.DisableAlerts, flowslatest.AlertLokiError) {
		r := alertLokiError()
		rules = append(rules, *r)
	}

	return rules
}

func BuildHealthRules(fc *flowslatest.FlowCollectorSpec) ([]HealthRule, error) {
	var rules []HealthRule
	var errs []error
	healthRules := fc.GetFLPHealthRules()
	metrics := fc.GetIncludeList()
	for _, healthRule := range healthRules {
		if ok, _ := healthRule.IsAllowed(fc); !ok {
			continue
		}
		for _, variant := range healthRule.Variants {
			// Get effective mode: variant.Mode if specified, otherwise healthRule.Mode
			effectiveMode := variant.GetMode(healthRule.Mode)
			if r, err := buildHealthRulesForVariant(healthRule.Template, effectiveMode, &variant, metrics); err != nil {
				errs = append(errs, err)
			} else if len(r) > 0 {
				rules = append(rules, r...)
			}
		}
	}
	return rules, errors.Join(errs...)
}

func buildHealthRulesForVariant(template flowslatest.HealthRuleTemplate, mode flowslatest.HealthRuleMode, healthRule *flowslatest.HealthRuleVariant, enabledMetrics []string) ([]HealthRule, error) {
	var allContexts []ruleContext
	var upperThreshold string
	sides := []srcOrDst{asSource, asDest}
	if healthRule.GroupBy == "" {
		// No side for global group
		sides = []srcOrDst{""}
	}

	// For recording rules, we only generate one rule (not per severity)
	// because the recording rule just calculates the percentage value
	if mode == flowslatest.ModeRecording {
		for _, side := range sides {
			allContexts = append(allContexts, ruleContext{
				template:       template,
				healthRule:     healthRule,
				mode:           mode,
				enabledMetrics: enabledMetrics,
				side:           side,
				recordingThresholds: &recordingThresholds{
					Info:     healthRule.Thresholds.Info,
					Warning:  healthRule.Thresholds.Warning,
					Critical: healthRule.Thresholds.Critical,
				},
			})
		}
	} else {
		// For alert rules, create up to 3 rules, one per severity, with non-overlapping thresholds
		thresholds := []struct {
			s string
			t string
		}{
			{s: "critical", t: healthRule.Thresholds.Critical},
			{s: "warning", t: healthRule.Thresholds.Warning},
			{s: "info", t: healthRule.Thresholds.Info},
		}
		for _, st := range thresholds {
			if st.t != "" {
				for _, side := range sides {
					allContexts = append(allContexts, ruleContext{
						template:       template,
						healthRule:     healthRule,
						mode:           mode,
						enabledMetrics: enabledMetrics,
						side:           side,
						severity:       st.s,
						alertThreshold: st.t,
						upperThreshold: upperThreshold,
						duration:       monitoringv1.Duration("5m"),
					})
				}
				upperThreshold = st.t
			}
		}
	}
	var rules []HealthRule
	for i := range allContexts {
		if r := allContexts[i].toRule(); r != nil {
			rules = append(rules, r)
		} else {
			return nil, fmt.Errorf("no rule for template: %s", allContexts[i].template)
		}
	}
	return rules, nil
}

func (ctx *ruleContext) toRule() HealthRule {
	switch ctx.template {
	case flowslatest.HealthRulePacketDropsByDevice:
		return newDeviceDrops(ctx)
	case flowslatest.HealthRulePacketDropsByKernel:
		return newKernelDrops(ctx)
	case flowslatest.HealthRuleIPsecErrors:
		return newIPsecErrors(ctx)
	case flowslatest.HealthRuleDNSErrors:
		return newDNSErrors(ctx)
	case flowslatest.HealthRuleDNSNxDomain:
		return newDNSNxDomain(ctx)
	case flowslatest.HealthRuleNetpolDenied:
		return newNetpolDenied(ctx)
	case flowslatest.HealthRuleLatencyHighTrend:
		return newLatencyTrend(ctx)
	case flowslatest.HealthRuleExternalEgressHighTrend:
		return newExternalTrend(ctx, false)
	case flowslatest.HealthRuleExternalIngressHighTrend:
		return newExternalTrend(ctx, true)
	case flowslatest.HealthRuleIngress5xxErrors:
		return newIngressErrors(ctx)
	case flowslatest.HealthRuleIngressHTTPLatencyTrend:
		return newIngressHTTPLatencyTrend(ctx)
	case flowslatest.AlertLokiError, flowslatest.AlertNoFlows:
		// ?
	}
	return nil
}

func buildLabels(template flowslatest.HealthRuleTemplate, severity string, forHealth bool) map[string]string {
	m := map[string]string{
		"template": string(template),
		"app":      "netobserv", // means that the rule is created by netobserv
	}
	if severity != "" { // should be always true for alerts, false for recordings
		m["severity"] = severity
	}
	if forHealth {
		m["netobserv"] = "true" // means that the rule should be fetched by netobserv console plugin for health
	}
	return m
}

// buildRunbookURL constructs the runbook URL for a given template
func buildRunbookURL(template flowslatest.AlertTemplate) string {
	// Template names are already in the correct format (e.g., "DNSErrors", "NetObservNoFlows")
	// They match the runbook filename without extension
	return fmt.Sprintf("%s/%s.md", runbookURLBase, template)
}

func createRule(ctx *ruleContext, r HealthRule, promQL string) (*monitoringv1.Rule, error) {
	// Generate recording rule
	if ctx.mode == flowslatest.ModeRecording {
		return &monitoringv1.Rule{
			Record: r.RecordingName(),
			// Note: Recording rules cannot have annotations in Prometheus
			Expr:   intstr.FromString(promQL),
			Labels: buildLabels(ctx.template, "", true),
		}, nil
	}

	// Generate alert rule
	ann, err := r.GetAnnotations()
	if err != nil {
		return nil, err
	}

	var gr string
	if ctx.healthRule.GroupBy != "" {
		gr = "Per" + string(ctx.side) + string(ctx.healthRule.GroupBy)
	}
	return &monitoringv1.Rule{
		Alert:       fmt.Sprintf("%s_%s%s", ctx.template, gr, strings.ToUpper(ctx.severity[:1])+ctx.severity[1:]),
		Annotations: ann,
		Expr:        intstr.FromString(promQL),
		For:         &ctx.duration,
		Labels:      buildLabels(ctx.template, ctx.severity, true),
	}, nil
}
