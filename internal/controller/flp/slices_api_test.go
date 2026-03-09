package flp

import (
	"encoding/json"
	"testing"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	"github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/fmstatus"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/slicesstatus"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
)

var (
	adminSubnets = []flowslatest.SubnetLabel{
		{
			Name:  "admin",
			CIDRs: []string{"10.0.0.0/16"},
		},
	}
	autoSubnets = []flowslatest.SubnetLabel{
		{
			Name:  "test",
			CIDRs: []string{"1.2.3.4/32", "10.0.0.0/8"},
		},
	}
	slicez = []sliceslatest.FlowCollectorSlice{
		{
			ObjectMeta: v1.ObjectMeta{Name: "a", Namespace: "ns-a"},
		},
		{
			ObjectMeta: v1.ObjectMeta{Name: "b1", Namespace: "ns-b"},
			Spec: sliceslatest.FlowCollectorSliceSpec{
				SubnetLabels: []sliceslatest.SubnetLabel{
					{
						Name:  "my-override",
						CIDRs: []string{"10.10.0.0/16"},
					},
					{
						Name:  "my-label",
						CIDRs: []string{"100.0.0.0/24"},
					},
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{Name: "b2", Namespace: "ns-b"},
			Spec: sliceslatest.FlowCollectorSliceSpec{
				SubnetLabels: []sliceslatest.SubnetLabel{
					{
						Name:  "skipped-overlap",
						CIDRs: []string{"10.0.0.0/24"},
					},
					{
						Name:  "partial-overlap",
						CIDRs: []string{"100.0.0.0/23"},
					},
				},
			},
		},
	}
)

func getConfiguredFiltersAndSubnets(cm *corev1.ConfigMap) ([]api.TransformFilterRule, []api.NetworkTransformSubnetLabel) {
	var cfs config.Root
	err := json.Unmarshal([]byte(cm.Data[configFile]), &cfs)
	if err != nil {
		return nil, nil
	}
	var filters []api.TransformFilterRule
	var subnetLabels []api.NetworkTransformSubnetLabel
	for _, stage := range cfs.Parameters {
		if stage.Transform != nil && stage.Name == "subnets" {
			subnetLabels = stage.Transform.Network.SubnetLabels
		}
		if stage.Transform != nil && stage.Name == "filters" {
			filters = stage.Transform.Filter.Rules
		}
	}
	return filters, subnetLabels
}

func defaultBuilderWithSlices(cfg *flowslatest.SlicesConfig) (monolithBuilder, error) {
	fc := getConfig()
	fc.Processor.SlicesConfig = cfg
	fc.Processor.SubnetLabels.CustomLabels = adminSubnets
	info := reconcilers.Common{Namespace: "namespace", Loki: &helper.LokiConfig{}, ClusterInfo: &cluster.Info{}}
	return newMonolithBuilder(info.NewInstance(image, status.Instance{}), &fc, &v1alpha1.FlowMetricList{}, slicez, autoSubnets)
}

func TestSlicesDisabled(t *testing.T) {
	fmstatus.Reset()
	slicesstatus.Reset(&sliceslatest.FlowCollectorSliceList{})
	b, err := defaultBuilderWithSlices(&flowslatest.SlicesConfig{Enable: false})
	assert.NoError(t, err)
	_, _, cm, err := b.configMaps()
	assert.NoError(t, err)
	filters, subnets := getConfiguredFiltersAndSubnets(cm)
	assert.Nil(t, filters)
	assert.Equal(t, []api.NetworkTransformSubnetLabel{
		{
			Name:  "admin",
			CIDRs: []string{"10.0.0.0/16"},
		},
		{
			Name:  "test",
			CIDRs: []string{"1.2.3.4/32", "10.0.0.0/8"},
		},
	}, subnets)
	for _, slice := range slicez {
		assert.Nil(t, slicesstatus.GetReadyCondition(&slice))
		assert.Nil(t, slicesstatus.GetSubnetWarningCondition(&slice))
		assert.Equal(t, 0, slice.Status.SubnetLabelsConfigured)
		assert.Equal(t, "", slice.Status.FilterApplied)
	}
}

func TestSlicesEnablesCollectAll(t *testing.T) {
	fmstatus.Reset()
	slicesstatus.Reset(&sliceslatest.FlowCollectorSliceList{})
	b, err := defaultBuilderWithSlices(&flowslatest.SlicesConfig{
		Enable:              true,
		CollectionMode:      flowslatest.CollectionAlwaysCollect,
		NamespacesAllowList: []string{"should-be-ignored"},
	})
	assert.NoError(t, err)
	_, _, cm, err := b.configMaps()
	assert.NoError(t, err)
	filters, subnets := getConfiguredFiltersAndSubnets(cm)
	assert.Nil(t, filters)
	assert.Equal(t, []api.NetworkTransformSubnetLabel{
		{
			Name:  "admin",
			CIDRs: []string{"10.0.0.0/16"},
		},
		{
			Name:  "my-override",
			CIDRs: []string{"10.10.0.0/16"},
		},
		{
			Name:  "my-label",
			CIDRs: []string{"100.0.0.0/24"},
		},
		{
			Name:  "partial-overlap",
			CIDRs: []string{"100.0.0.0/23"},
		},
		{
			Name:  "test",
			CIDRs: []string{"1.2.3.4/32", "10.0.0.0/8"},
		},
	}, subnets)
	// Slice 0
	ready := slicesstatus.GetReadyCondition(&slicez[0])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	assert.Nil(t, slicesstatus.GetSubnetWarningCondition(&slicez[0]))
	assert.Equal(t, 0, slicez[0].Status.SubnetLabelsConfigured)
	assert.Equal(t, "", slicez[0].Status.FilterApplied)
	// Slice 1
	ready = slicesstatus.GetReadyCondition(&slicez[1])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	assert.Nil(t, slicesstatus.GetSubnetWarningCondition(&slicez[1]))
	assert.Equal(t, 2, slicez[1].Status.SubnetLabelsConfigured)
	assert.Equal(t, "", slicez[1].Status.FilterApplied)
	// Slice 2
	ready = slicesstatus.GetReadyCondition(&slicez[2])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	warnings := slicesstatus.GetSubnetWarningCondition(&slicez[2])
	assert.NotNil(t, warnings)
	assert.Equal(t, `CIDR for 'skipped-overlap' (10.0.0.0/24) is fully overlapped by config (admin: 10.0.0.0/16) and will be ignored; CIDR for 'partial-overlap' (100.0.0.0/23) overlaps with config (ns-b/b1: 100.0.0.0/24)`, warnings.Message)
	assert.Equal(t, 1, slicez[2].Status.SubnetLabelsConfigured)
	assert.Equal(t, "", slicez[2].Status.FilterApplied)
}

func TestSlicesEnablesWhitelist(t *testing.T) {
	fmstatus.Reset()
	slicesstatus.Reset(&sliceslatest.FlowCollectorSliceList{})
	b, err := defaultBuilderWithSlices(&flowslatest.SlicesConfig{
		Enable:              true,
		CollectionMode:      flowslatest.CollectionAllowList,
		NamespacesAllowList: []string{"should-be-filtered", "/should-.*/"},
	})
	assert.NoError(t, err)
	_, _, cm, err := b.configMaps()
	assert.NoError(t, err)
	filters, subnets := getConfiguredFiltersAndSubnets(cm)
	assert.Equal(t, []api.TransformFilterRule{
		{
			Type:           api.KeepEntryQuery,
			KeepEntryQuery: `SrcK8S_Namespace="should-be-filtered" or DstK8S_Namespace="should-be-filtered"`,
		},
		{
			Type:           api.KeepEntryQuery,
			KeepEntryQuery: `SrcK8S_Namespace=~"should-.*" or DstK8S_Namespace=~"should-.*"`,
		},
		{
			Type:           api.KeepEntryQuery,
			KeepEntryQuery: `SrcK8S_Namespace="ns-a" or DstK8S_Namespace="ns-a"`,
		},
		{
			Type:           api.KeepEntryQuery,
			KeepEntryQuery: `SrcK8S_Namespace="ns-b" or DstK8S_Namespace="ns-b"`,
		},
	}, filters)
	assert.Equal(t, []api.NetworkTransformSubnetLabel{
		{
			Name:  "admin",
			CIDRs: []string{"10.0.0.0/16"},
		},
		{
			Name:  "my-override",
			CIDRs: []string{"10.10.0.0/16"},
		},
		{
			Name:  "my-label",
			CIDRs: []string{"100.0.0.0/24"},
		},
		{
			Name:  "partial-overlap",
			CIDRs: []string{"100.0.0.0/23"},
		},
		{
			Name:  "test",
			CIDRs: []string{"1.2.3.4/32", "10.0.0.0/8"},
		},
	}, subnets)
	// Slice 0
	ready := slicesstatus.GetReadyCondition(&slicez[0])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	assert.Nil(t, slicesstatus.GetSubnetWarningCondition(&slicez[0]))
	assert.Equal(t, 0, slicez[0].Status.SubnetLabelsConfigured)
	assert.Equal(t, `SrcK8S_Namespace="ns-a" or DstK8S_Namespace="ns-a"`, slicez[0].Status.FilterApplied)
	// Slice 1
	ready = slicesstatus.GetReadyCondition(&slicez[1])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	assert.Nil(t, slicesstatus.GetSubnetWarningCondition(&slicez[1]))
	assert.Equal(t, 2, slicez[1].Status.SubnetLabelsConfigured)
	assert.Equal(t, `SrcK8S_Namespace="ns-b" or DstK8S_Namespace="ns-b"`, slicez[1].Status.FilterApplied)
	// Slice 2
	ready = slicesstatus.GetReadyCondition(&slicez[2])
	assert.NotNil(t, ready)
	assert.Equal(t, v1.ConditionTrue, ready.Status)
	warnings := slicesstatus.GetSubnetWarningCondition(&slicez[2])
	assert.NotNil(t, warnings)
	assert.Equal(t, 1, slicez[2].Status.SubnetLabelsConfigured)
	assert.Equal(t, `(skipped, not needed)`, slicez[2].Status.FilterApplied)
}
