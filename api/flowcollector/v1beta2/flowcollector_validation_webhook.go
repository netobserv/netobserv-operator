package v1beta2

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/netobserv/flowlogs-pipeline/pkg/dsl"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type NetworkType string

const (
	OpenShiftSDN  NetworkType = "OpenShiftSDN"
	OVNKubernetes NetworkType = "OVNKubernetes"
	Kindnet       NetworkType = "Kindnet"
)

type clusterInfo interface {
	IsOpenShift() bool
	IsOpenShiftVersionAtLeast(v string) (bool, string, error)
	GetNbNodes() (uint16, error)
	GetCNI() (NetworkType, error)
}

var (
	log                    = logf.Log.WithName("flowcollector-resource")
	CurrentClusterInfo     clusterInfo
	needPrivileged         = []AgentFeature{UDNMapping, NetworkEvents}
	neededOpenShiftVersion = map[AgentFeature]string{
		PacketDrop:    "4.14.0",
		UDNMapping:    "4.18.0",
		NetworkEvents: "4.19.0",
		EbpfManager:   "4.19.0",
	}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *FlowCollector) ValidateCreate(ctx context.Context, fc *FlowCollector) (admission.Warnings, error) {
	log.Info("validate create", "name", r.Name)
	return r.Validate(ctx, fc)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *FlowCollector) ValidateUpdate(ctx context.Context, _, fc *FlowCollector) (admission.Warnings, error) {
	log.Info("validate update", "name", r.Name)
	return r.Validate(ctx, fc)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *FlowCollector) ValidateDelete(_ context.Context, _ *FlowCollector) (admission.Warnings, error) {
	log.Info("validate delete", "name", r.Name)
	return nil, nil
}

func (r *FlowCollector) Validate(_ context.Context, fc *FlowCollector) (admission.Warnings, error) {
	v := validator{fc: &fc.Spec}
	v.validateDeploymentModel()
	v.validateNetPol()
	v.validateAgent()
	v.validateFLP()
	v.warnLogLevels()
	v.warnLokiDemo()
	return v.warnings, errors.Join(v.errors...)
}

type validator struct {
	fc       *FlowCollectorSpec
	warnings admission.Warnings
	errors   []error
}

func (v *validator) warnLogLevels() {
	if v.fc.Agent.EBPF.LogLevel == "debug" || v.fc.Agent.EBPF.LogLevel == "trace" {
		v.warnings = append(v.warnings, fmt.Sprintf("The log level for the eBPF agent is %s, which impacts performance and resource footprint.", v.fc.Agent.EBPF.LogLevel))
	}
	if v.fc.Processor.LogLevel == "debug" || v.fc.Processor.LogLevel == "trace" {
		v.warnings = append(v.warnings, fmt.Sprintf("The log level for the processor (flowlogs-pipeline) is %s, which impacts performance and resource footprint.", v.fc.Processor.LogLevel))
	}
}

func (v *validator) warnLokiDemo() {
	if v.fc.Loki.Mode == LokiModeMonolithic && v.fc.Loki.Monolithic.InstallDemoLoki != nil && *v.fc.Loki.Monolithic.InstallDemoLoki {
		v.warnings = append(v.warnings, "InstallDemoLoki option is enabled. This is useful for development and demo purposes but should not be used in production.")
	}
}

func (v *validator) validateDeploymentModel() {
	if CurrentClusterInfo != nil {
		n, err := CurrentClusterInfo.GetNbNodes()
		if err != nil {
			v.warnings = append(v.warnings, fmt.Sprintf("Could not get the number of nodes, cannot validate the deployment model: %s", err.Error()))
		} else if n >= 15 && v.fc.DeploymentModel == DeploymentModelDirect {
			v.warnings = append(v.warnings, fmt.Sprintf(`The number of nodes is bigger than the recommendation for deployment model "Direct" (%d >= 15), meaning that "flowlogs-pipeline" uses a lot more memory and bandwidth than necessary; it is recommended to use a different deployment model ("Service" or "Kafka").`, n))
		}
	} else {
		v.warnings = append(v.warnings, "Unknown environment, cannot validate the deployment model")
	}
}

func (v *validator) validateNetPol() {
	if CurrentClusterInfo != nil {
		cni, err := CurrentClusterInfo.GetCNI()
		if err != nil {
			v.warnings = append(v.warnings, fmt.Sprintf("Could not detect CNI: %s", err.Error()))
		} else if cni == OpenShiftSDN && v.fc.NetworkPolicy.Enable != nil && *v.fc.NetworkPolicy.Enable {
			v.warnings = append(v.warnings, "OpenShiftSDN detected with unsupported setting: spec.networkPolicy.enable; this setting will be ignored; to remove this warning set spec.networkPolicy.enable to false.")
		} else if cni == "" && v.fc.DeployNetworkPolicy(false) {
			v.warnings = append(v.warnings, "Network policy is enabled via spec.networkPolicy.enable, despite running on an unknown CNI: this configuration has not been tested; to remove this warning set spec.networkPolicy.enable to false.")
		}
	} else {
		v.warnings = append(v.warnings, "Unknown environment, cannot detect the CNI in use")
	}
}

func (v *validator) validateAgent() {
	for feat, minVersion := range neededOpenShiftVersion {
		if slices.Contains(v.fc.Agent.EBPF.Features, feat) {
			if CurrentClusterInfo != nil && CurrentClusterInfo.IsOpenShift() {
				// Make sure required version of ocp is installed
				ok, actual, err := CurrentClusterInfo.IsOpenShiftVersionAtLeast(minVersion)
				if err != nil {
					v.warnings = append(v.warnings, fmt.Sprintf("Could not detect OpenShift cluster version: %s", err.Error()))
				} else if !ok {
					v.warnings = append(v.warnings, fmt.Sprintf("The %s feature requires OpenShift %s or above (version detected: %s)", feat, minVersion, actual))
				}
			} else {
				v.warnings = append(v.warnings, fmt.Sprintf("Unknown environment, cannot detect if the feature %s is supported", feat))
			}
		}
	}
	if !v.fc.Agent.EBPF.Privileged {
		for _, feat := range needPrivileged {
			if slices.Contains(v.fc.Agent.EBPF.Features, feat) {
				v.warnings = append(v.warnings, fmt.Sprintf("The %s feature requires eBPF Agent to run in privileged mode, which is currently disabled in spec.agent.ebpf.privileged", feat))
			}
		}
	}

	if slices.Contains(v.fc.Agent.EBPF.Features, PacketDrop) &&
		!v.fc.Agent.EBPF.Privileged &&
		!slices.Contains(v.fc.Agent.EBPF.Features, EbpfManager) {
		v.warnings = append(v.warnings, "The PacketDrop feature requires eBPF Agent to run in privileged mode, which is currently disabled in spec.agent.ebpf.privileged, or to use with eBPF Manager")
	}
	if v.fc.Agent.EBPF.FlowFilter != nil && v.fc.Agent.EBPF.FlowFilter.Enable != nil && *v.fc.Agent.EBPF.FlowFilter.Enable {
		m := make(map[string]bool)
		for i := range v.fc.Agent.EBPF.FlowFilter.Rules {
			rule := v.fc.Agent.EBPF.FlowFilter.Rules[i]
			key := rule.CIDR + "-" + rule.PeerCIDR
			if found := m[key]; found {
				v.errors = append(v.errors, fmt.Errorf("flow filter rule CIDR and PeerCIDR %s already exists", key))
				break
			}
			m[key] = true
			v.validateAgentFilter(&rule)
		}
		v.validateAgentFilter(&v.fc.Agent.EBPF.FlowFilter.EBPFFlowFilterRule)
	}
}

func (v *validator) validateAgentFilter(f *EBPFFlowFilterRule) {
	if f.CIDR != "" {
		if _, _, err := net.ParseCIDR(f.CIDR); err != nil {
			v.errors = append(v.errors, err)
		}
	}
	hasPorts := f.Ports.IntVal > 0 || f.Ports.StrVal != ""
	if hasPorts {
		if err := validateFilterPortConfig(f.Ports); err != nil {
			v.errors = append(v.errors, err)
		}
	}
	hasSrcPorts := f.SourcePorts.IntVal > 0 || f.SourcePorts.StrVal != ""
	if hasSrcPorts {
		if err := validateFilterPortConfig(f.SourcePorts); err != nil {
			v.errors = append(v.errors, err)
		}
	}
	hasDstPorts := f.DestPorts.IntVal > 0 || f.DestPorts.StrVal != ""
	if hasDstPorts {
		if err := validateFilterPortConfig(f.DestPorts); err != nil {
			v.errors = append(v.errors, err)
		}
	}
	if hasPorts && hasSrcPorts {
		v.errors = append(v.errors, errors.New("cannot configure agent filter with ports and sourcePorts, they are mutually exclusive"))
	}
	if hasPorts && hasDstPorts {
		v.errors = append(v.errors, errors.New("cannot configure agent filter with ports and destPorts, they are mutually exclusive"))
	}
}

func validateFilterPortConfig(value intstr.IntOrString) error {
	if value.Type == intstr.Int {
		return nil
	}
	sVal := value.String()
	if strings.Contains(sVal, "-") {
		ps := strings.SplitN(sVal, "-", 2)
		if len(ps) != 2 {
			return fmt.Errorf("invalid ports range: expected two integers separated by '-' but found %s", sVal)
		}
		start, err := validatePortString(ps[0])
		if err != nil {
			return fmt.Errorf("start port in range: %w", err)
		}
		end, err := validatePortString(ps[1])
		if err != nil {
			return fmt.Errorf("end port in range: %w", err)
		}
		if start >= end {
			return fmt.Errorf("invalid port range: start is greater or equal to end")
		}
		return nil
	} else if strings.Contains(sVal, ",") {
		ps := strings.Split(sVal, ",")
		if len(ps) != 2 {
			return fmt.Errorf("invalid ports couple: expected two integers separated by ',' but found %s", sVal)
		}
		_, err := validatePortString(ps[0])
		if err != nil {
			return fmt.Errorf("first port: %w", err)
		}
		_, err = validatePortString(ps[1])
		if err != nil {
			return fmt.Errorf("second port: %w", err)
		}
		return nil
	}
	// Should be a single port then
	_, err := validatePortString(sVal)
	if err != nil {
		return err
	}
	return nil
}

func validatePortString(s string) (uint16, error) {
	p, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port number %w", err)
	}
	if p == 0 {
		return 0, fmt.Errorf("invalid port 0")
	}
	return uint16(p), nil
}

func (v *validator) validateFLP() {
	v.validateScheduling()
	v.validateFLPLogTypes()
	v.validateFLPFilters()
	v.validateFLPAlerts()
	v.validateFLPMetricsForAlerts()
	v.validateFLPMetricsIncludeLists()
	v.validateFLPTLS()
}

func (v *validator) validateScheduling() {
	if v.fc.DeploymentModel == DeploymentModelDirect {
		// In direct mode, agent and FLP scheduling should be consistent, to ensure the 1-1 relation
		var agent, flp *SchedulingConfig
		if v.fc.Agent.EBPF.Advanced != nil {
			agent = v.fc.Agent.EBPF.Advanced.Scheduling
		}
		if v.fc.Processor.Advanced != nil {
			flp = v.fc.Processor.Advanced.Scheduling
		}
		if !reflect.DeepEqual(agent, flp) {
			v.warnings = append(v.warnings, "Mismatch detected between spec.agent.ebpf.advanced.scheduling and spec.processor.advanced.scheduling. In Direct mode, it can lead to inconsistent pod scheduling that would result in errors in the flow collection process.")
		}
	}
}

func (v *validator) validateFLPLogTypes() {
	if v.fc.Processor.HasConntrack() {
		if *v.fc.Processor.LogTypes == LogTypeAll {
			v.warnings = append(v.warnings, "Enabling all log types (in spec.processor.logTypes) has a high impact on resources footprint")
		}
		if !v.fc.UseLoki() {
			v.errors = append(v.errors, errors.New("enabling conversation tracking without Loki is not allowed, as it generates extra processing for no benefit"))
		}
		if v.fc.DeploymentModel == DeploymentModelService {
			v.errors = append(v.errors, errors.New("cannot enable conversation tracking when spec.deploymentModel is Service: you must disable it, or change the deployment model"))
		}
	}
}

func (v *validator) validateFLPFilters() {
	for i, filter := range v.fc.Processor.Filters {
		if _, err := dsl.Parse(filter.Query); err != nil {
			v.errors = append(v.errors, fmt.Errorf("cannot parse spec.processor.filters[%d].query: %w", i, err))
		}
	}
}

func (v *validator) validateFLPAlerts() {
	if v.fc.Processor.Metrics.HealthRules != nil {
		for i, alert := range *v.fc.Processor.Metrics.HealthRules {
			if _, msg := alert.IsAllowed(v.fc); len(msg) > 0 {
				v.warnings = append(v.warnings, msg)
			}
			for j, variant := range alert.Variants {
				// Check allowed groups
				if !v.isFLPHealthRuleGroupBySupported(alert.Template, &variant) {
					v.errors = append(
						v.errors,
						fmt.Errorf(
							`%s alert template does not support grouping by %s, in spec.processor.metrics.healthRules[%d].variants[%d]`,
							alert.Template, variant.GroupBy, i, j,
						),
					)
				}
				// Check that at least one threshold is provided (required for both alert and recording modes)
				if variant.Thresholds.Critical == "" && variant.Thresholds.Warning == "" && variant.Thresholds.Info == "" {
					v.errors = append(
						v.errors,
						fmt.Errorf(
							`at least one threshold (critical, warning, or info) must be provided in spec.processor.metrics.healthRules[%d].variants[%d]`,
							i, j,
						),
					)
				}
				lastThreshold := float64(-1)
				thresholds := []struct {
					s string
					t string
				}{
					{s: "critical", t: variant.Thresholds.Critical},
					{s: "warning", t: variant.Thresholds.Warning},
					{s: "info", t: variant.Thresholds.Info},
				}
				for _, st := range thresholds {
					if st.t != "" {
						val, err := strconv.ParseFloat(st.t, 64)
						if err != nil {
							v.errors = append(
								v.errors,
								fmt.Errorf(`cannot parse %s threshold as float in spec.processor.metrics.healthRules[%d].variants[%d]: "%s"`, st.s, i, j, st.t),
							)
						} else if val < 0 {
							v.errors = append(
								v.errors,
								fmt.Errorf(`%s threshold must be positive in spec.processor.metrics.healthRules[%d].variants[%d]: "%s"`, st.s, i, j, st.t),
							)
						} else if lastThreshold > 0 && val > lastThreshold {
							v.errors = append(
								v.errors,
								fmt.Errorf(`%s threshold must be lower than %.0f, which is defined for a higher severity, in spec.processor.metrics.healthRules[%d].variants[%d]: "%s"`, st.s, lastThreshold, i, j, st.t),
							)
						}
						lastThreshold = val
					}
				}
				if variant.LowVolumeThreshold != "" {
					_, err := strconv.ParseFloat(variant.LowVolumeThreshold, 64)
					if err != nil {
						v.errors = append(
							v.errors,
							fmt.Errorf(`cannot parse lowVolumeThreshold as float in spec.processor.metrics.healthRules[%d].variants[%d]: "%s"`, i, j, variant.LowVolumeThreshold),
						)
					}
				}
				// Validate variant mode
				v.validateVariantMode(alert.Template, &variant, i, j)
			}
		}
	}
}

func (v *validator) validateVariantMode(template HealthRuleTemplate, variant *HealthRuleVariant, ruleIndex, variantIndex int) {
	// Validate that variant mode (if specified) is not Recording for alert-only templates
	// Note: AlertNoFlows and AlertLokiError are handled separately and not part of healthRules,
	// but we keep this check for defensive programming in case that changes
	if variant.Mode != nil && *variant.Mode == ModeRecording {
		if template == AlertNoFlows || template == AlertLokiError {
			v.errors = append(
				v.errors,
				fmt.Errorf(
					`alert-only template %s cannot have variant with mode Recording in spec.processor.metrics.healthRules[%d].variants[%d]`,
					template, ruleIndex, variantIndex,
				),
			)
		}
	}
}

func (v *validator) isFLPHealthRuleGroupBySupported(template HealthRuleTemplate, variant *HealthRuleVariant) bool {
	switch template {
	case HealthRulePacketDropsByDevice:
		return variant.GroupBy != GroupByWorkload
	case HealthRuleIPsecErrors:
		return variant.GroupBy != GroupByWorkload && variant.GroupBy != GroupByNamespace
	case HealthRuleIngress5xxErrors, HealthRuleIngressHTTPLatencyTrend:
		return variant.GroupBy != GroupByNode && variant.GroupBy != GroupByWorkload
	case HealthRulePacketDropsByKernel, HealthRuleDNSErrors, HealthRuleDNSNxDomain, HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend, HealthRuleLatencyHighTrend, HealthRuleNetpolDenied:
		return true
	case AlertLokiError, AlertNoFlows: // not applicable
		return false
	}
	return true
}

func (v *validator) validateFLPMetricsForAlerts() {
	metrics := v.fc.GetIncludeList()
	alerts := v.fc.GetFLPHealthRules()
	for _, g := range alerts {
		for _, a := range g.Variants {
			reqMetrics1, reqMetrics2 := GetElligibleMetricsForAlert(g.Template, &a)
			// At least one metric from reqMetrics1 should be present, same for reqMetrics2
			if len(reqMetrics1) > 0 {
				if GetFirstRequiredMetrics(reqMetrics1, metrics) == "" {
					v.warnings = append(
						v.warnings,
						fmt.Sprintf("HealthRule %s/%s requires enabling at least one metric from this list: %s", g.Template, a.GroupBy, strings.Join(reqMetrics1, ", ")),
					)
				}
			}
			if len(reqMetrics2) > 0 {
				if GetFirstRequiredMetrics(reqMetrics2, metrics) == "" {
					v.warnings = append(
						v.warnings,
						fmt.Sprintf("HealthRule %s/%s requires enabling at least one metric from this list: %s", g.Template, a.GroupBy, strings.Join(reqMetrics2, ", ")),
					)
				}
			}
		}
	}
}

func (v *validator) validateFLPMetricsIncludeLists() {
	if v.fc.Processor.Metrics.IncludeList != nil && v.fc.Processor.Metrics.AdditionalIncludeList != nil {
		v.warnings = append(
			v.warnings,
			MetricsIncludeListWarning,
		)
	}
}

func (v *validator) validateFLPTLS() {
	if v.fc.DeploymentModel == DeploymentModelService && v.fc.Processor.Service != nil && v.fc.Processor.Service.TLSType == TLSProvided {
		if v.fc.Processor.Service.ProvidedCertificates == nil {
			v.errors = append(
				v.errors,
				errors.New("missing configuration in spec.processor.providedCertificates despite spec.processor.tlsType being set to Provided"),
			)
		} else if v.fc.Processor.Service.ProvidedCertificates.CAFile == nil {
			v.errors = append(
				v.errors,
				errors.New("missing configuration in spec.processor.providedCertificates.caFile despite spec.processor.tlsType being set to Provided"),
			)
		} else if v.fc.Processor.Service.ProvidedCertificates.ServerCert == nil {
			v.errors = append(
				v.errors,
				errors.New("missing configuration in spec.processor.providedCertificates.serverCert despite spec.processor.tlsType being set to Provided"),
			)
		}
	}
}

func GetFirstRequiredMetrics(anyRequired, actual []string) string {
	for _, m := range anyRequired {
		if slices.Contains(actual, m) {
			return m
		}
	}
	return ""
}

func GetElligibleMetricsForAlert(template HealthRuleTemplate, alertDef *HealthRuleVariant) ([]string, []string) {
	var metricPatterns, totalMetricPatterns []string
	switch template {
	case HealthRulePacketDropsByKernel:
		metricPatterns = []string{"%s_drop_packets_total"}
		totalMetricPatterns = []string{"%s_ingress_packets_total", "%s_egress_packets_total"}
	case HealthRuleIPsecErrors:
		return []string{"node_ipsec_flows_total"}, []string{"node_to_node_ingress_flows_total"}
	case HealthRuleDNSErrors, HealthRuleDNSNxDomain:
		metricPatterns = []string{`%s_dns_latency_seconds`}
		totalMetricPatterns = []string{"%s_dns_latency_seconds"}
	case HealthRuleExternalEgressHighTrend:
		metricPatterns = []string{`%s_egress_bytes_total`}
		totalMetricPatterns = []string{`%s_egress_bytes_total`}
	case HealthRuleExternalIngressHighTrend:
		metricPatterns = []string{`%s_ingress_bytes_total`}
		totalMetricPatterns = []string{`%s_ingress_bytes_total`}
	case HealthRuleLatencyHighTrend:
		metricPatterns = []string{`%s_rtt_seconds`}
		totalMetricPatterns = []string{`%s_rtt_seconds`}
	case HealthRuleNetpolDenied:
		metricPatterns = []string{`%s_network_policy_events_total`}
		totalMetricPatterns = []string{"%s_flows_total"}
	case AlertNoFlows, AlertLokiError, HealthRulePacketDropsByDevice, HealthRuleIngress5xxErrors, HealthRuleIngressHTTPLatencyTrend:
		// nothing - these rules don't use NetObserv metrics
		return nil, nil
	}
	var gr []string
	switch alertDef.GroupBy {
	case GroupByNode:
		gr = []string{"node"}
	case GroupByNamespace:
		gr = []string{"namespace", "workload"}
	case GroupByWorkload:
		gr = []string{"workload"}
	default: // global => any of the metric can work
		gr = []string{"namespace", "workload", "node"}
	}
	var metrics, totalMetrics []string
	for _, p := range metricPatterns {
		for _, g := range gr {
			metrics = append(metrics, fmt.Sprintf(p, g))
		}
	}
	for _, p := range totalMetricPatterns {
		for _, g := range gr {
			totalMetrics = append(totalMetrics, fmt.Sprintf(p, g))
		}
	}
	return metrics, totalMetrics
}

// Alias for backward compatibility
var GetElligibleMetricsForHealthRule = GetElligibleMetricsForAlert
