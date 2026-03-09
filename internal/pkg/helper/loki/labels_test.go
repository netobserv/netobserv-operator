package loki

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestDefaultLokiLabels(t *testing.T) {
	labels, err := GetLabels(&flowslatest.FlowCollectorSpec{})
	assert.Equal(t, err, nil)
	assert.Equal(t, []string{
		"SrcK8S_Namespace",
		"SrcK8S_OwnerName",
		"SrcK8S_Type",
		"DstK8S_Namespace",
		"DstK8S_OwnerName",
		"DstK8S_Type",
		"K8S_FlowLayer",
		"FlowDirection",
	}, labels)
}

func TestAllLokiLabels(t *testing.T) {
	outputRecordTypes := flowslatest.LogTypeAll
	labels, err := GetLabels(&flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Features: []flowslatest.AgentFeature{flowslatest.UDNMapping},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			LogTypes:               &outputRecordTypes,
			MultiClusterDeployment: ptr.To(true),
			AddZone:                ptr.To(true),
		},
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, []string{
		"SrcK8S_Namespace",
		"SrcK8S_OwnerName",
		"SrcK8S_Type",
		"DstK8S_Namespace",
		"DstK8S_OwnerName",
		"DstK8S_Type",
		"K8S_FlowLayer",
		"FlowDirection",
		"_RecordType",
		"K8S_ClusterName",
		"SrcK8S_Zone",
		"DstK8S_Zone",
		"UdnId",
	}, labels)
}

func TestExcludedLokiLabels(t *testing.T) {
	labels, err := GetLabels(&flowslatest.FlowCollectorSpec{
		Loki: flowslatest.FlowCollectorLoki{
			Advanced: &flowslatest.AdvancedLokiConfig{
				ExcludeLabels: []string{"SrcK8S_OwnerName", "DstK8S_OwnerName"},
			},
		},
	})
	assert.Equal(t, err, nil)
	assert.Equal(t, []string{
		"SrcK8S_Namespace",
		"SrcK8S_Type",
		"DstK8S_Namespace",
		"DstK8S_Type",
		"K8S_FlowLayer",
		"FlowDirection",
	}, labels)
}
