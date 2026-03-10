package util //nolint:revive

import (
	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	"k8s.io/utils/ptr"
)

func SpecForMetrics(metrics ...string) *flowslatest.FlowCollectorSpec {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.PacketDrop, flowslatest.UDNMapping, flowslatest.IPSec},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics:                flowslatest.FLPMetrics{},
			AddZone:                ptr.To(true),
			MultiClusterDeployment: ptr.To(true),
		},
	}
	if len(metrics) > 0 {
		var conv []flowslatest.FLPMetric
		for _, m := range metrics {
			conv = append(conv, flowslatest.FLPMetric(m))
		}
		fc.Processor.Metrics.IncludeList = &conv
	}
	return &fc
}
