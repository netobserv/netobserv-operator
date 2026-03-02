package otel

import (
	"testing"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/metrics"
	"github.com/netobserv/netobserv-operator/internal/pkg/test/util"
	"github.com/stretchr/testify/assert"
)

func TestOtelTransformConfig(t *testing.T) {
	m, err := GetOtelTransformConfig(nil)
	assert.Equal(t, err, nil)
	assert.Equal(t, api.TransformGenericOperationEnum("replace_keys"), m.Policy)
	assert.Equal(t, []api.GenericTransformRule{
		{Input: "Bytes", Output: "bytes", Multiplier: 0},
		{Input: "DnsErrno", Output: "dns.errno", Multiplier: 0},
		{Input: "DnsFlags", Output: "dns.flags", Multiplier: 0},
		{Input: "DnsFlagsResponseCode", Output: "dns.responsecode", Multiplier: 0},
		{Input: "DnsId", Output: "dns.id", Multiplier: 0},
		{Input: "DnsLatencyMs", Output: "dns.latency", Multiplier: 0},
		{Input: "Dscp", Output: "dscp", Multiplier: 0},
		{Input: "DstAddr", Output: "destination.address", Multiplier: 0},
		{Input: "DstK8S_HostIP", Output: "destination.k8s.host.address", Multiplier: 0},
		{Input: "DstK8S_HostName", Output: "destination.k8s.host.name", Multiplier: 0},
		{Input: "DstK8S_Name", Output: "destination.k8s.name", Multiplier: 0},
		{Input: "DstK8S_Namespace", Output: "destination.k8s.namespace.name", Multiplier: 0},
		{Input: "DstK8S_NetworkName", Output: "destination.network.name", Multiplier: 0},
		{Input: "DstK8S_OwnerName", Output: "destination.k8s.owner.name", Multiplier: 0},
		{Input: "DstK8S_OwnerType", Output: "destination.k8s.owner.kind", Multiplier: 0},
		{Input: "DstK8S_Type", Output: "destination.k8s.kind", Multiplier: 0},
		{Input: "DstK8S_Zone", Output: "destination.zone", Multiplier: 0},
		{Input: "DstMac", Output: "destination.mac", Multiplier: 0},
		{Input: "DstPort", Output: "destination.port", Multiplier: 0},
		{Input: "DstSubnetLabel", Output: "destination.subnet.label", Multiplier: 0},
		{Input: "Flags", Output: "tcp.flags", Multiplier: 0},
		{Input: "FlowDirection", Output: "host.direction", Multiplier: 0},
		{Input: "IPSecStatus", Output: "ipsec.status", Multiplier: 0},
		{Input: "IcmpCode", Output: "icmp.code", Multiplier: 0},
		{Input: "IcmpType", Output: "icmp.type", Multiplier: 0},
		{Input: "IfDirections", Output: "interface.directions", Multiplier: 0},
		{Input: "Interfaces", Output: "interface.names", Multiplier: 0},
		{Input: "K8S_ClusterName", Output: "k8s.cluster.name", Multiplier: 0},
		{Input: "K8S_FlowLayer", Output: "k8s.layer", Multiplier: 0},
		{Input: "Packets", Output: "packets", Multiplier: 0},
		{Input: "PktDropBytes", Output: "drops.bytes", Multiplier: 0},
		{Input: "PktDropLatestDropCause", Output: "drops.latestcause", Multiplier: 0},
		{Input: "PktDropLatestFlags", Output: "drops.latestflags", Multiplier: 0},
		{Input: "PktDropLatestState", Output: "drops.lateststate", Multiplier: 0},
		{Input: "PktDropPackets", Output: "drops.packets", Multiplier: 0},
		{Input: "Proto", Output: "protocol", Multiplier: 0},
		{Input: "SrcAddr", Output: "source.address", Multiplier: 0},
		{Input: "SrcK8S_HostIP", Output: "source.k8s.host.address", Multiplier: 0},
		{Input: "SrcK8S_HostName", Output: "source.k8s.host.name", Multiplier: 0},
		{Input: "SrcK8S_Name", Output: "source.k8s.name", Multiplier: 0},
		{Input: "SrcK8S_Namespace", Output: "source.k8s.namespace.name", Multiplier: 0},
		{Input: "SrcK8S_NetworkName", Output: "source.network.name", Multiplier: 0},
		{Input: "SrcK8S_OwnerName", Output: "source.k8s.owner.name", Multiplier: 0},
		{Input: "SrcK8S_OwnerType", Output: "source.k8s.owner.kind", Multiplier: 0},
		{Input: "SrcK8S_Type", Output: "source.k8s.kind", Multiplier: 0},
		{Input: "SrcK8S_Zone", Output: "source.zone", Multiplier: 0},
		{Input: "SrcMac", Output: "source.mac", Multiplier: 0},
		{Input: "SrcPort", Output: "source.port", Multiplier: 0},
		{Input: "SrcSubnetLabel", Output: "source.subnet.label", Multiplier: 0},
		{Input: "TimeFlowEndMs", Output: "timeflowend", Multiplier: 0},
		{Input: "TimeFlowRttNs", Output: "tcp.rtt", Multiplier: 0},
		{Input: "TimeFlowStartMs", Output: "timeflowstart", Multiplier: 0},
		{Input: "TimeReceived", Output: "timereceived", Multiplier: 0},
	}, m.Rules)

	// Make sure default metric labels are all covered
	defs := metrics.GetDefinitions(util.SpecForMetrics(), false)
	for _, metric := range defs {
		for _, label := range metric.Spec.Labels {
			assert.True(t, fieldFound(label, m.Rules), "missing label '%s' found in metric '%s'", label, metric.Spec.MetricName)
		}
		for _, filter := range metric.Spec.Filters {
			assert.True(t, fieldFound(filter.Field, m.Rules), "missing label '%s' found in filters for metric '%s'", filter.Field, metric.Spec.MetricName)
		}
		if metric.Spec.ValueField != "" {
			assert.True(t, fieldFound(metric.Spec.ValueField, m.Rules), "missing '%s' used as ValueField for metric '%s'", metric.Spec.ValueField, metric.Spec.MetricName)
		}
	}

	// override with custom rules
	m, err = GetOtelTransformConfig(&[]flowslatest.GenericTransformRule{{
		Input:      "Test",
		Output:     "outputTest",
		Multiplier: 1234,
	}})
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(m.Rules))
	assert.Equal(t, "Test", m.Rules[0].Input)
	assert.Equal(t, "outputTest", m.Rules[0].Output)
	assert.Equal(t, 1234, m.Rules[0].Multiplier)
}

func fieldFound(name string, rules []api.GenericTransformRule) bool {
	for _, r := range rules {
		if name == r.Input {
			return true
		}
	}
	return false
}

func TestOtelMetrics(t *testing.T) {
	metrics, err := GetOtelMetrics([]api.MetricsItem{{
		Name: "SrcK8S_Name",
		Type: "counter",
		Filters: []api.MetricsFilter{{
			Key:   "Proto",
			Value: "6",
			Type:  "equal",
		}},
		ValueKey: "Bytes",
		Labels:   []string{"SrcK8S_Name"},
	}})
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(metrics))
	assert.Equal(t, api.MetricsItem{
		Name: "source.k8s.name",
		Type: "counter",
		Filters: []api.MetricsFilter{
			{
				Key:   "protocol",
				Value: "6",
				Type:  "equal",
			},
		},
		ValueKey: "bytes",
		Labels:   []string{"source.k8s.name"},
	}, metrics[0])
}
