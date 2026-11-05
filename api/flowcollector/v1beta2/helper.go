package v1beta2

import (
	"slices"
	"strconv"
	"strings"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
)

func (spec *FlowCollectorSpec) OnHold() bool {
	return spec.Execution.Mode == OnHold
}

func (spec *FlowCollectorSpec) GetSampling() int {
	if spec.Agent.EBPF.Sampling == nil {
		// Profile-dependent default
		switch spec.Profile {
		case ProfileLowResources:
			return 400
		case ProfileHighResources:
			return 1
		case ProfileDefault:
		}
		return 50
	}
	return int(*spec.Agent.EBPF.Sampling)
}

func (spec *FlowCollectorSpec) GetCacheMaxFlows() int {
	if spec.Agent.EBPF.CacheMaxFlows == nil || *spec.Agent.EBPF.CacheMaxFlows == 0 {
		// Profile-dependent default
		switch spec.Profile {
		case ProfileLowResources:
			return 50_000
		case ProfileHighResources, ProfileDefault:
		}
		return 120_000
	}
	return int(*spec.Agent.EBPF.CacheMaxFlows)
}

func (spec *FlowCollectorSpec) UseKafka() bool {
	return spec.DeploymentModel == DeploymentModelKafka
}

func (spec *FlowCollectorSpec) HasKafkaExporter() bool {
	for _, ex := range spec.Exporters {
		if ex.Type == KafkaExporter {
			return true
		}
	}
	return false
}

func (cfg *SASLConfig) UseSASL() bool {
	return cfg.Type == SASLPlain || cfg.Type == SASLScramSHA512
}

func (spec *FlowCollectorSpec) UseLoki() bool {
	// nil should fallback to default value, which is "true", unless LowProfile is set
	if spec.Loki.Enable == nil {
		return spec.Profile != ProfileLowResources
	}
	return *spec.Loki.Enable
}

func (spec *FlowCollectorSpec) UseLokiDev() bool {
	return spec.UseLoki() && spec.Loki.Mode == LokiModeMonolithic && spec.Loki.Monolithic.InstallDemoLoki != nil && *spec.Loki.Monolithic.InstallDemoLoki
}

func (spec *FlowCollectorSpec) UsePrometheus() bool {
	// nil should fallback to default value, which is "true"
	return spec.Prometheus.Querier.Enable == nil || *spec.Prometheus.Querier.Enable
}

func (spec *FlowCollectorSpec) UseWebConsole() bool {
	return (spec.UseLoki() || spec.UsePrometheus()) &&
		// nil should fallback to default value, which is "true"
		(spec.ConsolePlugin.Enable == nil || *spec.ConsolePlugin.Enable)
}

func (spec *FlowCollectorSpec) UseStandaloneConsole(hasPluginAPI bool) bool {
	// defaults to true if there's no plugin API, false otherwise
	return (spec.ConsolePlugin.Standalone != nil && *spec.ConsolePlugin.Standalone ||
		spec.ConsolePlugin.Standalone == nil && !hasPluginAPI)
}

// NeedsConsolePluginDeployment is true when the console plugin Deployment (and related objects)
// should be reconciled. When false, the console reconciler only tears down or marks unused and
// does not need a resolved plugin image. hasPluginAPI is whether the ConsolePlugin CRD exists.
func (spec *FlowCollectorSpec) NeedsConsolePluginDeployment(hasPluginAPI bool) bool {
	return spec.UseWebConsole() &&
		(hasPluginAPI || spec.UseStandaloneConsole(hasPluginAPI)) &&
		!spec.OnHold()
}

func (spec *FlowCollectorSpec) UseHostNetwork() bool {
	return spec.DeploymentModel == DeploymentModelDirect
}

func (spec *FlowCollectorSpec) IsAgentFeatureEnabled(feature AgentFeature) bool {
	if spec.Agent.EBPF.Features == nil && spec.Profile == ProfileHighResources {
		return slices.Contains(HighProfileFeatures, feature)
	}
	return slices.Contains(spec.Agent.EBPF.Features, feature)
}

func (spec *FlowCollectorSpec) IsPktDropEnabled() bool {
	return spec.IsAgentFeatureEnabled(PacketDrop)
}

func (spec *FlowCollectorSpec) IsDNSTrackingEnabled() bool {
	return spec.IsAgentFeatureEnabled(DNSTracking)
}

// GetDNSTrackingPorts returns the DNS tracking ports as a comma-separated string for the eBPF agent.
func (spec *FlowCollectorEBPF) GetDNSTrackingPorts(vendor constants.Vendor) string {
	// Default is vendor-specific
	if len(spec.DNSTrackingPorts) == 0 {
		switch vendor {
		case constants.VendorOpenShift, constants.VendorOpenShiftDownstream:
			return "53,5353"
		default:
			return "53"
		}
	}
	// Convert []int32 to comma-separated string
	ports := make([]string, len(spec.DNSTrackingPorts))
	for i, port := range spec.DNSTrackingPorts {
		ports[i] = strconv.Itoa(int(port))
	}
	return strings.Join(ports, ",")
}

func (spec *FlowCollectorSpec) IsFlowRTTEnabled() bool {
	return spec.IsAgentFeatureEnabled(FlowRTT)
}

func (spec *FlowCollectorSpec) IsNetworkEventsEnabled() bool {
	return spec.IsAgentFeatureEnabled(NetworkEvents)
}

func (spec *FlowCollectorSpec) IsPacketTranslationEnabled() bool {
	return spec.IsAgentFeatureEnabled(PacketTranslation)
}

func (spec *FlowCollectorSpec) IsEbpfManagerEnabled() bool {
	return spec.IsAgentFeatureEnabled(EbpfManager)
}

func (spec *FlowCollectorSpec) IsUDNMappingEnabled() bool {
	return spec.IsAgentFeatureEnabled(UDNMapping)
}

func (spec *FlowCollectorSpec) IsIPSecEnabled() bool {
	return spec.IsAgentFeatureEnabled(IPSec)
}

func (spec *FlowCollectorSpec) IsTLSTrackingEnabled() bool {
	return spec.IsAgentFeatureEnabled(TLSTracking)
}

func (spec *FlowCollectorSpec) IsSecondaryNetworksEnabled() bool {
	// For backward compatibility, always enable SecondaryNetworks in priviliged mode
	return spec.IsAgentFeatureEnabled(SecondaryNetworks) || spec.Agent.EBPF.Privileged
}

func (spec *FlowCollectorEBPF) IsEBPFMetricsEnabled() bool {
	return spec.Metrics.Enable == nil || *spec.Metrics.Enable
}

func (spec *FlowCollectorEBPF) IsEBPFFlowFilterEnabled() bool {
	return spec.FlowFilter != nil && spec.FlowFilter.Enable != nil && *spec.FlowFilter.Enable
}

func (spec *FlowCollectorSpec) HasFiltersSampling() bool {
	if spec.Agent.EBPF.FlowFilter != nil {
		for i := range spec.Agent.EBPF.FlowFilter.Rules {
			if spec.Agent.EBPF.FlowFilter.Rules[i].Sampling != nil && *spec.Agent.EBPF.FlowFilter.Rules[i].Sampling > 1 {
				return true
			}
		}
	}
	for _, rule := range spec.Processor.Filters {
		if rule.Sampling > 1 {
			return true
		}
	}
	return false
}

func (spec *FlowCollectorFLP) HasConntrack() bool {
	return spec != nil && spec.LogTypes != nil && *spec.LogTypes != LogTypeFlows
}

func (spec *FlowCollectorFLP) IsMultiClusterEnabled() bool {
	return spec != nil && spec.MultiClusterDeployment != nil && *spec.MultiClusterDeployment
}

func (spec *FlowCollectorFLP) IsZoneEnabled() bool {
	return spec != nil && spec.AddZone != nil && *spec.AddZone
}

func (spec *FlowCollectorFLP) IsSubnetLabelsEnabled() bool {
	return spec.HasAutoDetectNetworks() || len(spec.SubnetLabels.CustomLabels) > 0
}

func (spec *FlowCollectorFLP) IsBgpEnrichmentEnabled() bool {
	return spec != nil && spec.BgpEnrichment != nil && *spec.BgpEnrichment
}

func (spec *FlowCollectorSpec) GetSecondaryIndexes() []SecondaryNetwork {
	if spec.Processor.Advanced != nil && len(spec.Processor.Advanced.SecondaryNetworks) > 0 {
		return spec.Processor.Advanced.SecondaryNetworks
	}
	if spec.IsSecondaryNetworksEnabled() {
		// Turn-on auto-detection in FLP by interface+MAC or interface+IP
		return []SecondaryNetwork{
			{Index: []SecondaryNetworkIndex{SecondaryNetworkIndexByInterface, SecondaryNetworkIndexByIP}},
			{Index: []SecondaryNetworkIndex{SecondaryNetworkIndexByInterface, SecondaryNetworkIndexByMAC}},
		}
	}
	return nil
}

func (spec *FlowCollectorFLP) HasAutoDetectNetworks() bool {
	if spec.SubnetLabels.AutoDetect != nil {
		return *spec.SubnetLabels.AutoDetect
	}
	if spec.SubnetLabels.OpenShiftAutoDetect != nil {
		return *spec.SubnetLabels.OpenShiftAutoDetect
	}
	return true
}

func (spec *FlowCollectorFLP) HasFLPDeduper() bool {
	return spec.Deduper != nil && spec.Deduper.Mode != "" && spec.Deduper.Mode != FLPDeduperDisabled
}

func (spec *FlowCollectorEBPF) GetMetricsPort() int32 {
	port := int32(constants.EBPFMetricPort)
	if spec.Metrics.Server.Port != nil {
		port = *spec.Metrics.Server.Port
	}
	return port
}

func (spec *FlowCollectorFLP) GetMetricsPort() int32 {
	port := int32(constants.FLPMetricsPort)
	if spec.Metrics.Server.Port != nil {
		port = *spec.Metrics.Server.Port
	}
	return port
}

func ShouldInstallNetworkPolicy(config *bool, cni NetworkType) bool {
	if cni == OpenShiftSDN {
		return false
	}
	if config != nil {
		return *config
	}
	// Default true only for recognized CNIs
	return cni != ""
}

func (spec *FlowCollectorFLP) GetFLPReplicas() int32 {
	if spec.ConsumerReplicas != nil {
		return *spec.ConsumerReplicas
	} else if spec.KafkaConsumerReplicas != nil {
		return *spec.KafkaConsumerReplicas
	}
	return 3
}

func (spec *FlowCollectorHPA) IsHPAEnabled() bool {
	return spec != nil && spec.Status == HPAStatusEnabled
}

func (spec *FlowCollectorFLP) IsUnmanagedFLPReplicas() bool {
	if spec.UnmanagedReplicas {
		return true
	}
	return spec.KafkaConsumerAutoscaler.IsHPAEnabled()
}

func (spec *FlowCollectorFLP) IsInformerCacheProxyEnabled() bool {
	return spec.InformerCacheProxy != nil && spec.InformerCacheProxy.Enabled != nil && *spec.InformerCacheProxy.Enabled
}

// DefaultK8sCachePort is the default gRPC port where FLP processors listen for k8scache updates
const DefaultK8sCachePort int32 = 9402

// GetK8sCachePort returns the gRPC port where FLP processors listen for k8scache updates.
// If spec.processor.informerCacheProxy.advanced.processorPort is configured, it returns that value.
// Otherwise, it returns the default port (9402).
func (spec *FlowCollectorFLP) GetK8sCachePort() int32 {
	if spec.InformerCacheProxy != nil &&
		spec.InformerCacheProxy.Advanced != nil &&
		spec.InformerCacheProxy.Advanced.ProcessorPort != nil {
		return *spec.InformerCacheProxy.Advanced.ProcessorPort
	}
	return DefaultK8sCachePort
}

func (spec *FlowCollectorInformerCacheProxy) GetTLSType() TLSConfigType {
	if spec == nil || spec.TLS == nil {
		return TLSAuto
	}
	return spec.TLS.Type
}

func (spec *FlowCollectorInformerCacheProxy) UsesOpenShiftServiceCA(isOpenShift bool) bool {
	if !isOpenShift {
		return false
	}
	tlsType := spec.GetTLSType()
	return tlsType == TLSAuto
}

func (spec *FlowCollectorConsolePlugin) IsUnmanagedConsolePluginReplicas() bool {
	if spec.UnmanagedReplicas {
		return true
	}
	return spec.Autoscaler.IsHPAEnabled()
}

func (spec *FlowCollectorSpec) IsSliceEnabled() bool {
	return spec.Processor.SlicesConfig != nil && spec.Processor.SlicesConfig.Enable
}

func IsEnvEnabled(vars map[string]string, key string) bool {
	env := vars[key]
	// Use ParseBool to allow common variants ("true", "True", "1"...) and ignore non-bools
	b, err := strconv.ParseBool(env)
	return err == nil && b
}
