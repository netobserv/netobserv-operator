package status

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestGetResolvedProfileSummary(t *testing.T) {
	info := getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{})
	assert.Equal(t, "Current profile: (none) {spec.agent.ebpf.sampling=50, spec.agent.ebpf.cacheMaxFlows=120000, spec.agent.ebpf.features=[], spec.loki.enable=true}", info)

	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileLowResources,
	})
	assert.Equal(t, "Current profile: Low {spec.agent.ebpf.sampling=400, spec.agent.ebpf.cacheMaxFlows=50000, spec.agent.ebpf.features=[], spec.loki.enable=false}", info)

	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileLowResources,
		Loki:    flowslatest.FlowCollectorLoki{Enable: ptr.To(true)},
	})
	assert.Equal(t, "Current profile: Low {spec.agent.ebpf.sampling=400, spec.agent.ebpf.cacheMaxFlows=50000, spec.agent.ebpf.features=[], spec.loki.enable=true (override; default: false)}", info)

	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileDefault,
	})
	assert.Equal(t, "Current profile: Default {spec.agent.ebpf.sampling=50, spec.agent.ebpf.cacheMaxFlows=120000, spec.agent.ebpf.features=[], spec.loki.enable=true}", info)

	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileHighResources,
	})
	assert.Equal(t, "Current profile: High {spec.agent.ebpf.sampling=1, spec.agent.ebpf.cacheMaxFlows=120000, spec.agent.ebpf.features=[DNSTracking,FlowRTT,PacketDrop,SecondaryNetworks,TLSTracking,UDNMapping], spec.loki.enable=true}", info)

	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileHighResources,
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Sampling: ptr.To(int32(10)),
				Features: []flowslatest.AgentFeature{flowslatest.DNSTracking, flowslatest.IPSec},
			},
		}})
	assert.Equal(t, "Current profile: High {spec.agent.ebpf.sampling=10 (override; default: 1), spec.agent.ebpf.cacheMaxFlows=120000, spec.agent.ebpf.features=[DNSTracking,IPSec] (override; default: [DNSTracking,FlowRTT,PacketDrop,SecondaryNetworks,TLSTracking,UDNMapping]), spec.loki.enable=true}", info)

	// Some edge cases
	info = getResolvedProfileSummary(&flowslatest.FlowCollectorSpec{
		Profile: flowslatest.ProfileHighResources,
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				CacheMaxFlows: ptr.To(int32(0)),             // counts as default => expect 120000
				Sampling:      ptr.To(int32(0)),             // does *not* count as default => expect 0 (even though it's equivalent to 1 for the agent)
				Features:      []flowslatest.AgentFeature{}, // does *not* count as default => expect no features
			},
		},
		Loki: flowslatest.FlowCollectorLoki{Enable: ptr.To(false)},
	})
	assert.Equal(t, "Current profile: High {spec.agent.ebpf.sampling=0 (override; default: 1), spec.agent.ebpf.cacheMaxFlows=120000, spec.agent.ebpf.features=[] (override; default: [DNSTracking,FlowRTT,PacketDrop,SecondaryNetworks,TLSTracking,UDNMapping]), spec.loki.enable=false (override; default: true)}", info)
}
