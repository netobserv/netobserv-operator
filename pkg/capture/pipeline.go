package capture

import (
	"encoding/json"
	"regexp"
	"sort"

	flpapi "github.com/netobserv/flowlogs-pipeline/pkg/api"
	flpconfig "github.com/netobserv/flowlogs-pipeline/pkg/config"
)

var bareQueryValue = regexp.MustCompile(`((?:[A-Za-z_][A-Za-z0-9_>]*\s*)(?:!=|=~|!~|=)\s*)([A-Za-z_][A-Za-z0-9_.-]*)`)

// AddSubnetLabels adds the same subnet enrichment used by the CLI --get-subnets option.
func AddSubnetLabels(config *flpconfig.Root, subnets map[string][]string) {
	if config == nil || len(subnets) == 0 {
		return
	}
	for i := range config.Parameters {
		parameter := &config.Parameters[i]
		if parameter.Name != "enrich" || parameter.Transform == nil || parameter.Transform.Network == nil {
			continue
		}
		network := parameter.Transform.Network
		network.Rules = append(network.Rules,
			flpapi.NetworkTransformRule{Type: flpapi.NetworkAddSubnetLabel, AddSubnetLabel: &flpapi.NetworkAddSubnetLabelRule{Input: "SrcAddr", Output: "SrcSubnetLabel"}},
			flpapi.NetworkTransformRule{Type: flpapi.NetworkAddSubnetLabel, AddSubnetLabel: &flpapi.NetworkAddSubnetLabelRule{Input: "DstAddr", Output: "DstSubnetLabel"}},
		)
		categories := make([]string, 0, len(subnets))
		for category := range subnets {
			categories = append(categories, category)
		}
		sort.Strings(categories)
		network.SubnetLabels = make([]flpapi.NetworkTransformSubnetLabel, 0, len(categories))
		for _, category := range categories {
			network.SubnetLabels = append(network.SubnetLabels, flpapi.NetworkTransformSubnetLabel{Name: category, CIDRs: subnets[category]})
		}
		return
	}
}

// AddFilterQueries adds Flowlogs-Pipeline keep-entry queries after enrichment.
// It works for both collector and metrics pipelines by routing the first stage
// after enrichment through the shared filter stage.
func AddFilterQueries(config *flpconfig.Root, queries []string) {
	if config == nil || len(queries) == 0 {
		return
	}
	rules := make([]flpapi.TransformFilterRule, 0, len(queries))
	for _, query := range queries {
		rules = append(rules, flpapi.TransformFilterRule{Type: flpapi.KeepEntryQuery, KeepEntryQuery: normalizeQuery(query)})
	}
	for i := range config.Parameters {
		if config.Parameters[i].Name == "filter" && config.Parameters[i].Transform != nil && config.Parameters[i].Transform.Filter != nil {
			config.Parameters[i].Transform.Filter.Rules = append(config.Parameters[i].Transform.Filter.Rules, rules...)
			return
		}
	}
	config.Parameters = append(config.Parameters, flpconfig.StageParam{
		Name:      "filter",
		Transform: &flpconfig.Transform{Type: "filter", Filter: &flpapi.TransformFilter{Rules: rules}},
	})
	for i := range config.Pipeline {
		if config.Pipeline[i].Follows == "enrich" {
			config.Pipeline[i].Follows = "filter"
		}
	}
	filter := flpconfig.Stage{Name: "filter", Follows: "enrich"}
	insertAt := len(config.Pipeline)
	for i := range config.Pipeline {
		if config.Pipeline[i].Name != "enrich" {
			insertAt = i
			break
		}
	}
	config.Pipeline = append(config.Pipeline, flpconfig.Stage{})
	copy(config.Pipeline[insertAt+1:], config.Pipeline[insertAt:])
	config.Pipeline[insertAt] = filter
}

// normalizeQuery quotes simple string values so queries assembled by clients as
// K8S_FlowLayer=app remain valid Flowlogs-Pipeline expressions. Numeric values
// and already quoted values are left untouched.
func normalizeQuery(query string) string {
	return bareQueryValue.ReplaceAllString(query, `${1}"${2}"`)
}

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
	metrics := make(flpapi.MetricsItems, 0, 36)
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
		counterMetric(scope.prefix+"_tls_flows_total", "", appendCopy(labels, "TLSVersion", "TLSCipherSuite", "TLSGroup"), flowFilter("TLSTypes", "", flpapi.MetricFilterPresence)),
		counterMetric(scope.prefix+"_ipsec_flows_total", "", appendCopy(labels, "IPSecStatus"), flowFilter("IPSecStatus", "", flpapi.MetricFilterPresence)),
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
