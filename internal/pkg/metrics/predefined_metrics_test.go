package metrics

import (
	"sort"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/test/util"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

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
	assert.Len(res, 3)
	assert.Equal("node_ingress_bytes_total", res[0].Spec.MetricName)
	assert.Equal("Bytes", res[0].Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"}, res[0].Spec.Labels)
	assert.Equal("namespace_flows_total", res[1].Spec.MetricName)
	assert.Empty(res[1].Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, res[1].Spec.Labels)
	assert.Equal("workload_egress_packets_total", res[2].Spec.MetricName)
	assert.Equal("Packets", res[2].Spec.ValueField)
	assert.Equal([]string{"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, res[2].Spec.Labels)
}

func TestGetDefinitionsRemoveZoneCluster(t *testing.T) {
	assert := assert.New(t)

	spec := util.SpecForMetrics("namespace_flows_total", "node_ingress_bytes_total", "workload_egress_packets_total")
	spec.Processor.AddZone = ptr.To(false)
	spec.Processor.MultiClusterDeployment = ptr.To(false)
	res := GetDefinitions(spec, false)
	assert.Len(res, 3)
	assert.Equal("node_ingress_bytes_total", res[0].Spec.MetricName)
	assert.Equal("Bytes", res[0].Spec.ValueField)
	assert.Equal([]string{"SrcK8S_HostName", "DstK8S_HostName"}, res[0].Spec.Labels)
	assert.Equal("namespace_flows_total", res[1].Spec.MetricName)
	assert.Empty(res[1].Spec.ValueField)
	assert.Equal([]string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"}, res[1].Spec.Labels)
	assert.Equal("workload_egress_packets_total", res[2].Spec.MetricName)
	assert.Equal("Packets", res[2].Spec.ValueField)
	assert.Equal([]string{"SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"}, res[2].Spec.Labels)
}
