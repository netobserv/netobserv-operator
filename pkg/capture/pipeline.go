package capture

import (
	"encoding/json"

	flpapi "github.com/netobserv/flowlogs-pipeline/pkg/api"
	flpconfig "github.com/netobserv/flowlogs-pipeline/pkg/config"
)

// CollectorPipelineConfig returns the typed FLP configuration used by flow and
// packet captures. Keeping this here makes the CLI and FlowCapture use exactly
// the same enrichment and collector connection settings.
func CollectorPipelineConfig(targetHost string) *flpconfig.Root {
	return &flpconfig.Root{
		LogLevel:        "trace",
		MetricsSettings: flpconfig.MetricsSettings{DisableGlobalServer: true},
		Parameters: []flpconfig.StageParam{
			{Name: "enrich", Transform: &flpconfig.Transform{Type: "network", Network: &flpapi.TransformNetwork{
				Rules: []flpapi.NetworkTransformRule{
					kubernetesRule("SrcAddr", "SrcMac", "SrcK8S"),
					kubernetesRule("DstAddr", "DstMac", "DstK8S"),
					{Type: flpapi.NetworkReinterpretDirection},
				},
				DirectionInfo: flowDirectionInfo(),
			}}},
			{Name: "send", Write: &flpconfig.Write{Type: "grpc", GRPC: &flpapi.WriteGRPC{TargetHost: targetHost, TargetPort: 9999}}},
		},
		Pipeline: []flpconfig.Stage{{Name: "enrich", Follows: "preset-ingester"}, {Name: "send", Follows: "enrich"}},
	}
}

// MetricsPipelineConfig returns the typed FLP configuration used by metrics
// captures. Metric families are generated from the three supported scopes
// instead of being maintained as a large JSON resource.
func MetricsPipelineConfig() *flpconfig.Root {
	return &flpconfig.Root{
		LogLevel: "trace",
		MetricsSettings: flpconfig.MetricsSettings{
			PromConnectionInfo: flpapi.PromConnectionInfo{Port: 9401},
			Prefix:             "on_demand_netobserv_",
			NoPanic:            true,
		},
		Parameters: []flpconfig.StageParam{
			{Name: "enrich", Transform: &flpconfig.Transform{Type: "network", Network: &flpapi.TransformNetwork{
				Rules: []flpapi.NetworkTransformRule{
					kubernetesRule("SrcAddr", "", "SrcK8S"),
					kubernetesRule("DstAddr", "", "DstK8S"),
					{Type: flpapi.NetworkReinterpretDirection},
					{Type: flpapi.NetworkAddKubernetesInfra, KubernetesInfra: &flpapi.K8sInfraRule{
						NamespaceNameFields: []flpapi.K8sReference{{Name: "SrcK8S_Name", Namespace: "SrcK8S_Namespace"}, {Name: "DstK8S_Name", Namespace: "DstK8S_Namespace"}},
						Output:              "K8S_FlowLayer", InfraPrefixes: []string{"netobserv", "openshift"},
						InfraRefs: []flpapi.K8sReference{{Name: "kubernetes", Namespace: "default"}, {Name: "openshift", Namespace: "default"}},
					}},
				},
				DirectionInfo: flowDirectionInfo(),
			}}},
			{Name: "prometheus", Encode: &flpconfig.Encode{Type: "prom", Prom: &flpapi.PromEncode{Metrics: captureMetrics(), Prefix: "on_demand_netobserv_"}}},
		},
		Pipeline: []flpconfig.Stage{{Name: "enrich", Follows: "preset-ingester"}, {Name: "prometheus", Follows: "enrich"}},
	}
}

// PipelineConfigJSON serializes a typed FLP configuration for FLP_CONFIG.
func PipelineConfigJSON(config *flpconfig.Root) (string, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func kubernetesRule(ipField, macField, output string) flpapi.NetworkTransformRule {
	return flpapi.NetworkTransformRule{Type: flpapi.NetworkAddKubernetes, Kubernetes: &flpapi.K8sRule{
		AddZone: true, IPField: ipField, InterfacesField: fieldIfSet(macField, "Interfaces"), UDNsField: fieldIfSet(macField, "Udns"), MACField: macField, Output: output,
	}}
}

func fieldIfSet(value, field string) string {
	if value == "" {
		return ""
	}
	return field
}

func flowDirectionInfo() flpapi.NetworkTransformDirectionInfo {
	return flpapi.NetworkTransformDirectionInfo{ReporterIPField: "AgentIP", SrcHostField: "SrcK8S_HostIP", DstHostField: "DstK8S_HostIP", FlowDirectionField: "FlowDirection"}
}

func captureMetrics() flpapi.MetricsItems {
	metrics := make(flpapi.MetricsItems, 0, 30)
	for _, scope := range []metricScope{
		{prefix: "node", labels: []string{"SrcK8S_HostName", "DstK8S_HostName"}},
		{prefix: "namespace", labels: []string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}},
		{prefix: "workload", labels: []string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}},
	} {
		metrics = append(metrics, scopedMetrics(scope)...)
	}
	return metrics
}

type metricScope struct {
	prefix string
	labels []string
}

func scopedMetrics(scope metricScope) flpapi.MetricsItems {
	labels := scope.labels
	return flpapi.MetricsItems{
		counterMetric(scope.prefix+"_egress_bytes_total", "Bytes", labels, flowFilter("FlowDirection", "1|2", flpapi.MetricFilterRegex)),
		counterMetric(scope.prefix+"_ingress_bytes_total", "Bytes", labels, flowFilter("FlowDirection", "0|2", flpapi.MetricFilterRegex)),
		counterMetric(scope.prefix+"_egress_packets_total", "Packets", labels, flowFilter("FlowDirection", "1|2", flpapi.MetricFilterRegex)),
		counterMetric(scope.prefix+"_ingress_packets_total", "Packets", labels, flowFilter("FlowDirection", "0|2", flpapi.MetricFilterRegex)),
		counterMetric(scope.prefix+"_flows_total", "", labels),
		histogramMetric(scope.prefix+"_rtt_seconds", "TimeFlowRttNs", labels, 1_000_000_000, flowFilter("TimeFlowRttNs", "", flpapi.MetricFilterPresence)),
		counterMetric(scope.prefix+"_drop_packets_total", "PktDropPackets", appendCopy(labels, "PktDropLatestState", "PktDropLatestDropCause"), flowFilter("PktDropPackets", "", flpapi.MetricFilterPresence)),
		counterMetric(scope.prefix+"_drop_bytes_total", "PktDropBytes", appendCopy(labels, "PktDropLatestState", "PktDropLatestDropCause"), flowFilter("PktDropBytes", "", flpapi.MetricFilterPresence)),
		histogramMetric(scope.prefix+"_dns_latency_seconds", "DnsLatencyMs", appendCopy(labels, "DnsFlagsResponseCode"), 1000, flowFilter("DnsId", "", flpapi.MetricFilterPresence)),
		policyMetric(scope.prefix+"_network_policy_events_total", labels),
	}
}

func counterMetric(name, valueKey string, labels []string, filters ...flpapi.MetricsFilter) flpapi.MetricsItem {
	return flpapi.MetricsItem{Name: name, Type: flpapi.MetricCounter, Filters: filters, ValueKey: valueKey, Labels: appendCopy(labels)}
}

func histogramMetric(name, valueKey string, labels []string, scale float64, filters ...flpapi.MetricsFilter) flpapi.MetricsItem {
	return flpapi.MetricsItem{Name: name, Type: flpapi.MetricHistogram, Filters: filters, ValueKey: valueKey, Labels: appendCopy(labels), Buckets: []float64{0.005, 0.01, 0.02, 0.03, 0.04, 0.05, 0.075, 0.1, 0.25, 1}, ValueScale: scale}
}

func policyMetric(name string, labels []string) flpapi.MetricsItem {
	return flpapi.MetricsItem{
		Name: name, Type: flpapi.MetricCounter,
		Filters: []flpapi.MetricsFilter{{Key: "NetworkEvents>Feature", Value: "acl"}},
		Labels:  appendCopy(labels, "NetworkEvents>Type", "NetworkEvents>Namespace", "NetworkEvents>Name", "NetworkEvents>Action", "NetworkEvents>Direction"),
		Remap:   map[string]string{"NetworkEvents>Action": "action", "NetworkEvents>Direction": "direction", "NetworkEvents>Name": "name", "NetworkEvents>Namespace": "namespace", "NetworkEvents>Type": "type"},
		Flatten: []string{"NetworkEvents"},
	}
}

func flowFilter(key, value string, filterType flpapi.MetricFilterEnum) flpapi.MetricsFilter {
	return flpapi.MetricsFilter{Key: key, Value: value, Type: filterType}
}

func appendCopy(values []string, extra ...string) []string {
	result := append([]string(nil), values...)
	return append(result, extra...)
}
