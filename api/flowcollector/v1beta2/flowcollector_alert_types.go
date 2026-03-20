package v1beta2

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HealthRuleTemplate string
type HealthRuleGroupBy string
type HealthRuleMode string

// Backward compatibility type alias for disableAlerts field
type AlertTemplate = HealthRuleTemplate
type AlertGroupBy = HealthRuleGroupBy

const (
	// Alert-only templates (cannot be used as recording rules)
	AlertNoFlows   AlertTemplate = "NetObservNoFlows"
	AlertLokiError AlertTemplate = "NetObservLokiError"

	// Health rule templates (can be either alerts or recording rules depending on mode)
	HealthRulePacketDropsByKernel      HealthRuleTemplate = "PacketDropsByKernel"
	HealthRulePacketDropsByDevice      HealthRuleTemplate = "PacketDropsByDevice"
	HealthRuleIPsecErrors              HealthRuleTemplate = "IPsecErrors"
	HealthRuleNetpolDenied             HealthRuleTemplate = "NetpolDenied"
	HealthRuleLatencyHighTrend         HealthRuleTemplate = "LatencyHighTrend"
	HealthRuleDNSErrors                HealthRuleTemplate = "DNSErrors"
	HealthRuleDNSNxDomain              HealthRuleTemplate = "DNSNxDomain"
	HealthRuleExternalEgressHighTrend  HealthRuleTemplate = "ExternalEgressHighTrend"
	HealthRuleExternalIngressHighTrend HealthRuleTemplate = "ExternalIngressHighTrend"
	HealthRuleIngress5xxErrors         HealthRuleTemplate = "Ingress5xxErrors"
	HealthRuleIngressHTTPLatencyTrend  HealthRuleTemplate = "IngressHTTPLatencyTrend"

	GroupByNode      HealthRuleGroupBy = "Node"
	GroupByNamespace HealthRuleGroupBy = "Namespace"
	GroupByWorkload  HealthRuleGroupBy = "Workload"

	ModeAlert     HealthRuleMode = "Alert"
	ModeRecording HealthRuleMode = "Recording"
)

type FLPHealthRule struct {
	// Health rule template name.
	// Possible values are: `PacketDropsByKernel`, `PacketDropsByDevice`, `IPsecErrors`, `NetpolDenied`,
	// `LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`.
	// Note: `NetObservNoFlows` and `NetObservLokiError` are alert-only and cannot be used as health rules.
	// More information on health rules: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md
	// +kubebuilder:validation:Enum:="PacketDropsByKernel";"PacketDropsByDevice";"IPsecErrors";"NetpolDenied";"LatencyHighTrend";"DNSErrors";"DNSNxDomain";"ExternalEgressHighTrend";"ExternalIngressHighTrend";"Ingress5xxErrors";"IngressHTTPLatencyTrend"
	// +required
	Template HealthRuleTemplate `json:"template,omitempty"`

	// Mode defines whether this health rule should be generated as an alert or a recording rule.
	// Possible values are: `Alert` (default), `Recording`.
	// Recording rules violations are visible in the Network Health dashboard without generating any Prometheus alert.
	// This provides an alternative way of getting Health information for SRE and cluster admins who may find
	// many new alerts burdensome.
	// +kubebuilder:validation:Enum:="Alert";"Recording"
	// +kubebuilder:default:="Alert"
	// +optional
	Mode HealthRuleMode `json:"mode,omitempty"`

	// A list of variants for this template
	// +required
	Variants []HealthRuleVariant `json:"variants,omitempty"`
}

type HealthRuleVariant struct {
	// The low volume threshold allows to ignore metrics with a too low volume of traffic, in order to improve signal-to-noise.
	// It is provided as an absolute rate (bytes per second or packets per second, depending on the context).
	// When provided, it must be parsable as a float.
	LowVolumeThreshold string `json:"lowVolumeThreshold,omitempty"`

	// Thresholds of the health rule per severity.
	// They are expressed as a percentage of errors above which the alert is triggered. They must be parsable as floats.
	// Required for both alert and recording modes
	// +required
	Thresholds HealthRuleThresholds `json:"thresholds,omitempty"`

	// Optional grouping criteria, possible values are: `Node`, `Namespace`, `Workload`.
	// +kubebuilder:validation:Enum:="";"Node";"Namespace";"Workload"
	// +optional
	GroupBy HealthRuleGroupBy `json:"groupBy,omitempty"`

	// For trending health rules, the time offset for baseline comparison. For example, "1d" means comparing against yesterday. Defaults to 1d.
	TrendOffset *metav1.Duration `json:"trendOffset,omitempty"`

	// For trending health rules, the duration interval for baseline comparison. For example, "2h" means comparing against a 2-hours average. Defaults to 2h.
	TrendDuration *metav1.Duration `json:"trendDuration,omitempty"`

	// Mode overrides the health rule mode for this specific variant.
	// If not specified, inherits from the parent health rule's mode.
	// Possible values are: `Alert`, `Recording`.
	// +kubebuilder:validation:Enum:="Alert";"Recording"
	// +optional
	Mode *HealthRuleMode `json:"mode,omitempty"`
}

type HealthRuleThresholds struct {
	// Threshold for severity `info`. Leave empty to not generate an Info alert.
	Info string `json:"info,omitempty"`

	// Threshold for severity `warning`. Leave empty to not generate a Warning alert.
	Warning string `json:"warning,omitempty"`

	// Threshold for severity `critical`. Leave empty to not generate a Critical alert.
	Critical string `json:"critical,omitempty"`
}

func (s *FlowCollectorSpec) GetIncludeList() []string {
	var list []string
	if s.Processor.Metrics.IncludeList == nil {
		if s.UseLoki() {
			list = DefaultIncludeList
		} else {
			// When loki is disabled, increase what's available through metrics by default, to minimize the loss of information
			list = DefaultIncludeListLokiDisabled
		}
	} else {
		for _, m := range *s.Processor.Metrics.IncludeList {
			list = append(list, string(m))
		}
	}
	if !s.Agent.EBPF.IsPktDropEnabled() && !s.Agent.EBPF.IsNetworkEventsEnabled() {
		list = removeMetricsByPattern(list, "_drop_")
	}
	if !s.Agent.EBPF.IsFlowRTTEnabled() {
		list = removeMetricsByPattern(list, "_rtt_")
	}
	if !s.Agent.EBPF.IsDNSTrackingEnabled() {
		list = removeMetricsByPattern(list, "_dns_")
	}
	if !s.Agent.EBPF.IsNetworkEventsEnabled() {
		list = removeMetricsByPattern(list, "_network_policy_")
	}
	if !s.HasFiltersSampling() {
		list = removeMetricsByPattern(list, "_sampling")
	}
	if !s.Agent.EBPF.IsIPSecEnabled() {
		list = removeMetricsByPattern(list, "_ipsec_")
	}
	return list
}

func removeMetricsByPattern(list []string, search string) []string {
	var filtered []string
	for _, m := range list {
		if !strings.Contains(m, search) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (s *FlowCollectorSpec) GetFLPHealthRules() []FLPHealthRule {
	var rules []FLPHealthRule
	var templates []HealthRuleTemplate // for reproducible ordering

	tplMap := make(map[HealthRuleTemplate]FLPHealthRule)
	for _, group := range DefaultHealthRules {
		if !slices.Contains(s.Processor.Metrics.DisableAlerts, group.Template) {
			tplMap[group.Template] = group
			templates = append(templates, group.Template)
		}
	}
	if s.Processor.Metrics.HealthRules != nil {
		for _, group := range *s.Processor.Metrics.HealthRules {
			if !slices.Contains(s.Processor.Metrics.DisableAlerts, group.Template) {
				// A group defined in FC overrides the default group
				tplMap[group.Template] = group
				if !slices.Contains(templates, group.Template) {
					templates = append(templates, group.Template)
				}
			}
		}
	}

	for _, name := range templates {
		tpl := tplMap[name]
		if ok, _ := tpl.IsAllowed(s); ok {
			rules = append(rules, tpl)
		}
	}

	return rules
}

func (g *FLPHealthRule) IsAllowed(spec *FlowCollectorSpec) (bool, string) {
	switch g.Template {
	case HealthRulePacketDropsByKernel:
		if !spec.Agent.EBPF.IsPktDropEnabled() {
			return false, fmt.Sprintf("HealthRule %s requires the %s agent feature to be enabled", g.Template, PacketDrop)
		}
	case HealthRuleIPsecErrors:
		if !spec.Agent.EBPF.IsIPSecEnabled() {
			return false, fmt.Sprintf("HealthRule %s requires the %s agent feature to be enabled", g.Template, IPSec)
		}
	case HealthRuleDNSErrors, HealthRuleDNSNxDomain:
		if !spec.Agent.EBPF.IsDNSTrackingEnabled() {
			return false, fmt.Sprintf("HealthRule %s requires the %s agent feature to be enabled", g.Template, DNSTracking)
		}
	case HealthRuleLatencyHighTrend:
		if !spec.Agent.EBPF.IsFlowRTTEnabled() {
			return false, fmt.Sprintf("HealthRule %s requires the %s agent feature to be enabled", g.Template, FlowRTT)
		}
	case HealthRuleNetpolDenied:
		if !spec.Agent.EBPF.IsNetworkEventsEnabled() {
			return false, fmt.Sprintf("HealthRule %s requires the %s agent feature to be enabled", g.Template, NetworkEvents)
		}
	case AlertNoFlows, AlertLokiError, HealthRulePacketDropsByDevice, HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend, HealthRuleIngress5xxErrors, HealthRuleIngressHTTPLatencyTrend:
		return true, ""
	}
	return true, ""
}

func (v *HealthRuleVariant) GetTrendParams() (string, string) {
	offset := metav1.Duration{Duration: 24 * time.Hour}
	if v.TrendOffset != nil {
		offset = *v.TrendOffset
	}
	duration := metav1.Duration{Duration: 2 * time.Hour}
	if v.TrendDuration != nil {
		duration = *v.TrendDuration
	}
	return durationToStringTrimmed(&offset), durationToStringTrimmed(&duration)
}

// GetMode returns the effective mode for this variant.
// If the variant has a mode specified, it returns that mode.
// Otherwise, it returns the parent health rule's mode.
func (v *HealthRuleVariant) GetMode(parentMode HealthRuleMode) HealthRuleMode {
	if v.Mode != nil {
		return *v.Mode
	}
	return parentMode
}

var regTrim = regexp.MustCompile("([a-zA-Z])(0[a-zA-Z])+")

func durationToStringTrimmed(d *metav1.Duration) string {
	s := d.Duration.String()
	return regTrim.ReplaceAllString(s, "$1")
}
