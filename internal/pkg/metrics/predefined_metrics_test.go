package metrics

import (
	"sort"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/pkg/test/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func findMetric(metrics []metricslatest.FlowMetric, name string) *metricslatest.FlowMetric {
	for i := range metrics {
		if metrics[i].Spec.MetricName == name {
			return &metrics[i]
		}
	}
	return nil
}

func TestIncludeExclude(t *testing.T) {
	assert := assert.New(t)

	// IgnoreTags set, Include list unset => resolving ignore tags
	res := GetAsIncludeList([]string{"egress", "packets", "flows"}, nil)
	sort.Slice(*res, func(i, j int) bool { return (*res)[i] < (*res)[j] })
	assert.Equal([]flowslatest.FLPMetric{
		"namespace_dns_latency_seconds",
		"namespace_drop_bytes_total",
		"namespace_ingress_bytes_total",
		"namespace_ipsec_flows_total",
		"namespace_network_policy_events_total",
		"namespace_rtt_seconds",
		"namespace_sampling",
		"node_dns_latency_seconds",
		"node_drop_bytes_total",
		"node_ingress_bytes_total",
		"node_ipsec_flows_total",
		"node_network_policy_events_total",
		"node_rtt_seconds",
		"node_sampling",
		"node_to_node_ingress_flows_total",
		"workload_dns_latency_seconds",
		"workload_drop_bytes_total",
		"workload_ingress_bytes_total",
		"workload_ipsec_flows_total",
		"workload_network_policy_events_total",
		"workload_rtt_seconds",
		"workload_sampling",
	}, *res)

	// IgnoreTags set, Include list set => keep include list
	res = GetAsIncludeList([]string{"egress", "packets"}, &[]flowslatest.FLPMetric{"namespace_flows_total"})
	assert.Equal([]flowslatest.FLPMetric{"namespace_flows_total"}, *res)

	// IgnoreTags set as defaults, Include list unset => use default include list
	res = GetAsIncludeList([]string{"egress", "packets", "nodes-flows", "namespaces-flows", "workloads-flows", "namespaces"}, nil)
	assert.Nil(res)

	// IgnoreTags set as defaults, Include list set => use include list
	res = GetAsIncludeList([]string{"egress", "packets", "nodes-flows", "namespaces-flows", "workloads-flows", "namespaces"}, &[]flowslatest.FLPMetric{"namespace_flows_total"})
	assert.Equal([]flowslatest.FLPMetric{"namespace_flows_total"}, *res)
}

func TestGetDefinitions(t *testing.T) {
	assert := assert.New(t)

	res := GetDefinitions(util.SpecForMetrics("namespace_flows_total", "node_ingress_bytes_total", "workload_egress_packets_total"), false)

	nodeIngress := findMetric(res, "node_ingress_bytes_total")
	namespaceFlows := findMetric(res, "namespace_flows_total")
	workloadEgress := findMetric(res, "workload_egress_packets_total")

	assert.NotNil(nodeIngress, "node_ingress_bytes_total should be present")
	assert.Equal("Bytes", nodeIngress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"}, nodeIngress.Spec.Labels)

	assert.NotNil(namespaceFlows, "namespace_flows_total should be present")
	assert.Empty(namespaceFlows.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, namespaceFlows.Spec.Labels)

	assert.NotNil(workloadEgress, "workload_egress_packets_total should be present")
	assert.Equal("Packets", workloadEgress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_NetworkName", "DstK8S_NetworkName", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, workloadEgress.Spec.Labels)
}

func TestGetDefinitionsRemoveZoneCluster(t *testing.T) {
	assert := assert.New(t)

	spec := util.SpecForMetrics("namespace_flows_total", "node_ingress_bytes_total", "workload_egress_packets_total")
	spec.Processor.AddZone = ptr.To(false)
	spec.Processor.MultiClusterDeployment = ptr.To(false)
	res := GetDefinitions(spec, false)

	nodeIngress := findMetric(res, "node_ingress_bytes_total")
	assert.NotNil(nodeIngress)
	assert.Equal("Bytes", nodeIngress.Spec.ValueField)
	assert.Equal([]string{"SrcK8S_HostName", "DstK8S_HostName"}, nodeIngress.Spec.Labels)

	namespaceFlows := findMetric(res, "namespace_flows_total")
	assert.NotNil(namespaceFlows)
	assert.Empty(namespaceFlows.Spec.ValueField)
	assert.Equal([]string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, namespaceFlows.Spec.Labels)

	workloadEgress := findMetric(res, "workload_egress_packets_total")
	assert.NotNil(workloadEgress)
	assert.Equal("Packets", workloadEgress.Spec.ValueField)
	assert.Equal([]string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_NetworkName", "DstK8S_NetworkName", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, workloadEgress.Spec.Labels)
}

func TestGetDefinitionsRemoveNetworkLabels(t *testing.T) {
	assert := assert.New(t)

	spec := util.SpecForMetrics("workload_ingress_bytes_total")
	spec.Agent.EBPF.Features = []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.PacketDrop}
	spec.Processor.Advanced = nil
	res := GetDefinitions(spec, false)

	workloadIngress := findMetric(res, "workload_ingress_bytes_total")
	assert.NotNil(workloadIngress)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, workloadIngress.Spec.Labels)
}

func TestGetDefinitionsNodeMetrics(t *testing.T) {
	assert := assert.New(t)

	res := GetDefinitions(util.SpecForMetrics("node_ingress_bytes_total", "node_egress_packets_total", "node_flows_total"), false)

	nodeIngress := findMetric(res, "node_ingress_bytes_total")
	assert.NotNil(nodeIngress)
	assert.Equal("Bytes", nodeIngress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"}, nodeIngress.Spec.Labels)

	nodeEgress := findMetric(res, "node_egress_packets_total")
	assert.NotNil(nodeEgress)
	assert.Equal("Packets", nodeEgress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"}, nodeEgress.Spec.Labels)

	nodeFlows := findMetric(res, "node_flows_total")
	assert.NotNil(nodeFlows)
	assert.Empty(nodeFlows.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"}, nodeFlows.Spec.Labels)
}

func TestGetDefinitionsNamespaceMetrics(t *testing.T) {
	assert := assert.New(t)

	res := GetDefinitions(util.SpecForMetrics("namespace_ingress_bytes_total", "namespace_egress_packets_total", "namespace_flows_total"), false)

	nsIngress := findMetric(res, "namespace_ingress_bytes_total")
	assert.NotNil(nsIngress)
	assert.Equal("Bytes", nsIngress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, nsIngress.Spec.Labels)

	nsEgress := findMetric(res, "namespace_egress_packets_total")
	assert.NotNil(nsEgress)
	assert.Equal("Packets", nsEgress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, nsEgress.Spec.Labels)

	nsFlows := findMetric(res, "namespace_flows_total")
	assert.NotNil(nsFlows)
	assert.Empty(nsFlows.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, nsFlows.Spec.Labels)
}

func TestGetDefinitionsWorkloadMetrics(t *testing.T) {
	assert := assert.New(t)

	res := GetDefinitions(util.SpecForMetrics("workload_ingress_bytes_total", "workload_egress_packets_total", "workload_flows_total"), false)

	wlIngress := findMetric(res, "workload_ingress_bytes_total")
	assert.NotNil(wlIngress)
	assert.Equal("Bytes", wlIngress.Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_NetworkName", "DstK8S_NetworkName", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, wlIngress.Spec.Labels)

	wlEgress := findMetric(res, "workload_egress_packets_total")
	assert.NotNil(wlEgress)
	assert.Equal("Packets", wlEgress.Spec.ValueField)

	wlFlows := findMetric(res, "workload_flows_total")
	assert.NotNil(wlFlows)
	assert.Empty(wlFlows.Spec.ValueField)
}

func TestGetDefinitionsAllMetricTypesForGroup(t *testing.T) {
	assert := assert.New(t)

	// Test all metric types for a single group (node)
	res := GetDefinitions(util.SpecForMetrics("node_ingress_bytes_total", "node_rtt_seconds", "node_drop_packets_total", "node_dns_latency_seconds", "node_ipsec_flows_total"), false)

	metricNames := make(map[string]bool)
	for _, m := range res {
		metricNames[m.Spec.MetricName] = true
	}
	assert.Contains(metricNames, "node_ingress_bytes_total")
	assert.Contains(metricNames, "node_rtt_seconds")
	assert.Contains(metricNames, "node_drop_packets_total")
	assert.Contains(metricNames, "node_dns_latency_seconds")
	assert.Contains(metricNames, "node_ipsec_flows_total")

	// Verify RTT has correct configuration
	for _, m := range res {
		if m.Spec.MetricName == "node_rtt_seconds" {
			assert.Equal("TimeFlowRttNs", m.Spec.ValueField)
			assert.Equal("1000000000", m.Spec.Divider)
			assert.Len(m.Spec.Filters, 1)
		}
	}
}

func TestGetDefinitionsMixedGroups(t *testing.T) {
	assert := assert.New(t)

	// Test requesting metrics from different groups
	res := GetDefinitions(util.SpecForMetrics("node_ingress_bytes_total", "namespace_flows_total", "workload_egress_packets_total"), false)

	metricNames := make(map[string]bool)
	for _, m := range res {
		metricNames[m.Spec.MetricName] = true
	}
	assert.True(metricNames["node_ingress_bytes_total"])
	assert.True(metricNames["namespace_flows_total"])
	assert.True(metricNames["workload_egress_packets_total"])
}

func TestGetDefinitionsRemoveZoneLabels(t *testing.T) {
	assert := assert.New(t)

	spec := util.SpecForMetrics("node_ingress_bytes_total", "namespace_flows_total")
	spec.Processor.AddZone = ptr.To(false)
	res := GetDefinitions(spec, false)

	// All metrics should have zone labels removed
	for _, m := range res {
		assert.NotContains(m.Spec.Labels, "SrcK8S_Zone")
		assert.NotContains(m.Spec.Labels, "DstK8S_Zone")
	}
}

func TestGetDefinitionsRemoveMultiClusterLabels(t *testing.T) {
	assert := assert.New(t)

	spec := util.SpecForMetrics("node_ingress_bytes_total", "namespace_flows_total")
	spec.Processor.MultiClusterDeployment = ptr.To(false)
	res := GetDefinitions(spec, false)

	// All metrics should have cluster label removed
	for _, m := range res {
		assert.NotContains(m.Spec.Labels, "K8S_ClusterName")
	}
}

func TestGetDefinitionsNoDuplicates(t *testing.T) {
	assert := assert.New(t)

	// Specify metrics that are already in the default list
	// node_ingress_bytes_total and node_egress_bytes_total are both in defaults
	spec := util.SpecForMetrics("node_ingress_bytes_total", "node_egress_bytes_total", "workload_ingress_bytes_total")
	res := GetDefinitions(spec, false)

	// Count occurrences of each metric to ensure no duplicates
	metricCounts := make(map[string]int)
	for _, m := range res {
		metricCounts[m.Spec.MetricName]++
	}

	// Each metric should appear exactly once
	assert.Equal(1, metricCounts["node_ingress_bytes_total"], "node_ingress_bytes_total should appear only once")
	assert.Equal(1, metricCounts["node_egress_bytes_total"], "node_egress_bytes_total should appear only once")
	assert.Equal(1, metricCounts["workload_ingress_bytes_total"], "workload_ingress_bytes_total should appear only once")

	// Verify all metrics are unique
	for metricName, count := range metricCounts {
		assert.Equal(1, count, "Metric %s should appear exactly once, but appears %d times", metricName, count)
	}
}
