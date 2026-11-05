package ebpf

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func spec(epbf *flowslatest.FlowCollectorEBPF) *flowslatest.FlowCollectorSpec {
	return &flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: *epbf,
		},
	}
}

func TestGetAllowedCapabilities(t *testing.T) {
	// Default
	assert.Equal(t, []v1.Capability{"BPF", "PERFMON", "NET_ADMIN"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{})))

	// Privileged
	assert.Empty(t, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{Privileged: true})))

	// Override
	assert.Equal(t, []v1.Capability{"NET_RAW"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{
		Advanced: &flowslatest.AdvancedAgentConfig{CapOverride: []string{"NET_RAW"}},
	})))

	// Secondary nets
	assert.Equal(t, []v1.Capability{"BPF", "PERFMON", "NET_ADMIN", "SYS_ADMIN"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{
		Features: []flowslatest.AgentFeature{flowslatest.SecondaryNetworks},
	})))

	// Network events
	assert.Equal(t, []v1.Capability{"BPF", "PERFMON", "NET_ADMIN", "DAC_OVERRIDE"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{
		Features: []flowslatest.AgentFeature{flowslatest.NetworkEvents},
	})))

	// UDN
	assert.Equal(t, []v1.Capability{"BPF", "PERFMON", "NET_ADMIN", "DAC_OVERRIDE"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{
		Features: []flowslatest.AgentFeature{flowslatest.UDNMapping},
	})))

	// Many feats
	assert.Equal(t, []v1.Capability{"BPF", "PERFMON", "NET_ADMIN", "SYS_ADMIN", "DAC_OVERRIDE"}, getAllowedCapabilities(spec(&flowslatest.FlowCollectorEBPF{
		Features: []flowslatest.AgentFeature{
			flowslatest.DNSTracking,
			flowslatest.FlowRTT,
			flowslatest.UDNMapping,
			flowslatest.NetworkEvents,
			flowslatest.SecondaryNetworks,
			flowslatest.PacketDrop,
		},
	})))
}
