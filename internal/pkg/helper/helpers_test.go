package helper

import (
	"testing"
	"time"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

func TestExtractVersion(t *testing.T) {
	assert := assert.New(t)

	v := ExtractVersion("quay.io/netobserv/flowlogs-pipeline:v0.1.0")
	assert.Equal("v0.1.0", v)
}

func TestExtractUnknownVersion(t *testing.T) {
	assert := assert.New(t)

	v := ExtractVersion("flowlogs-pipeline")
	assert.Equal("unknown", v)
}

func TestIsSubset(t *testing.T) {
	assert.True(t, IsSubSet(
		map[string]string{"a": "b", "c": "d", "e": "f"},
		map[string]string{"a": "b", "c": "d", "e": "f"}))
	assert.True(t, IsSubSet(
		map[string]string{"a": "b", "c": "d", "e": "f"},
		map[string]string{"a": "b", "e": "f"}))
	assert.False(t, IsSubSet(
		map[string]string{"a": "b", "c": "d", "e": "f"},
		map[string]string{"a": "b", "e": "xxx"}))
	assert.False(t, IsSubSet(
		map[string]string{"a": "b", "c": "d", "e": "f"},
		map[string]string{"a": "b", "z": "d"}))
	assert.False(t, IsSubSet(
		map[string]string{"a": "b", "c": "d", "e": "f"},
		map[string]string{"a": "b", "c": "d", "e": "f", "g": "h"}))
}

func TestRemoveAllStrings(t *testing.T) {
	assert := assert.New(t)

	s := RemoveAllStrings([]string{"one", "two", "three", "four", "three"}, "three")
	assert.Equal([]string{"one", "two", "four"}, s)

	s = RemoveAllStrings(s, "five")
	assert.Equal([]string{"one", "two", "four"}, s)
}

func TestKeySorted(t *testing.T) {
	set := map[string]string{
		"b": "1",
		"c": "2",
		"a": "3",
		"d": "4",
	}
	assert.Equal(t,
		[][2]string{{"a", "3"}, {"b", "1"}, {"c", "2"}, {"d", "4"}},
		KeySorted(set))
}

func TestMaxLabelLengt_Cut(t *testing.T) {
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
		MaxLabelLength("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde_CUT_HERE"))
}

func TestMaxLabelLengt_NoCut(t *testing.T) {
	assert.Equal(t, "0123456789", MaxLabelLength("0123456789"))
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde",
		MaxLabelLength("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcde"))
}

func TestUnstructuredDuration(t *testing.T) {

	t.Run("nil input", func(t *testing.T) {
		var d *metav1.Duration
		got := UnstructuredDuration(d)
		want := ""

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("valid input", func(t *testing.T) {
		d := &metav1.Duration{Duration: time.Minute}
		want := "1m0s"
		got := UnstructuredDuration(d)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestCRDDefault(t *testing.T) {
	labels := map[string]string{"app": "netobserv-flowcollector"}
	labelBytes, _ := yaml.Marshal(labels)

	SetCRD(&v1.CustomResourceDefinition{
		Spec: v1.CustomResourceDefinitionSpec{
			Versions: []v1.CustomResourceDefinitionVersion{
				{
					Name: "testAPI",
					Schema: &v1.CustomResourceValidation{
						OpenAPIV3Schema: &v1.JSONSchemaProps{
							Properties: map[string]v1.JSONSchemaProps{
								"spec": {
									Properties: map[string]v1.JSONSchemaProps{
										"processor": {
											Properties: map[string]v1.JSONSchemaProps{
												"debug": {
													Properties: map[string]v1.JSONSchemaProps{
														"conversationEndTimeout": {
															Default: &v1.JSON{
																Raw: []byte("10s"),
															},
														},
														"lokiMaxRetries": {
															Default: &v1.JSON{
																Raw: []byte("2"),
															},
														},
														"lokiStaticLabels": {
															Default: &v1.JSON{
																Raw: labelBytes,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	// conversationEndTimeout
	assert.Equal(t, "10s", GetFieldDefaultString([]string{"spec", "processor", "debug"}, "conversationEndTimeout"))
	assert.Equal(t, true, IsDefaultValue([]string{"spec", "processor", "debug"}, "conversationEndTimeout", "10s"))
	assert.Equal(t, false, IsDefaultValue([]string{"spec", "processor", "debug"}, "conversationEndTimeout", "1s"))
	// lokiMaxRetries
	assert.Equal(t, "2", GetFieldDefaultString([]string{"spec", "processor", "debug"}, "lokiMaxRetries"))
	assert.Equal(t, true, IsDefaultValue([]string{"spec", "processor", "debug"}, "lokiMaxRetries", "2"))
	assert.Equal(t, false, IsDefaultValue([]string{"spec", "processor", "debug"}, "lokiMaxRetries", "12"))
	// lokiStaticLabels
	assert.Equal(t, "app: netobserv-flowcollector\n", GetFieldDefaultString([]string{"spec", "processor", "debug"}, "lokiStaticLabels"))
}

func TestPrometheusMetricName(t *testing.T) {
	name := PrometheusMetricName("a-metric-name:unit")
	assert.Equal(t, "a_metric_name:unit", name)
}

func TestEnsureDefaults(t *testing.T) {
	fc := flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{},
	}
	err := EnsureDefaults(&fc)
	require.NoError(t, err)
	assert.Equal(t, flowslatest.DefaultFlowCollector, &fc.Spec)
}

func TestEnsureDefaults_Overrides(t *testing.T) {
	fc := flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "test",
			Agent: flowslatest.FlowCollectorAgent{
				EBPF: flowslatest.FlowCollectorEBPF{
					Sampling:           ptr.To(int32(100)),
					CacheActiveTimeout: "20s",
				},
			},
		},
	}
	err := EnsureDefaults(&fc)
	require.NoError(t, err)
	assert.Equal(t, "test", fc.Spec.Namespace)
	assert.EqualValues(t, 100, *fc.Spec.Agent.EBPF.Sampling)
	assert.Equal(t, "20s", fc.Spec.Agent.EBPF.CacheActiveTimeout)
	assert.Equal(t, 1048576, fc.Spec.Agent.EBPF.KafkaBatchSize)
	assert.Equal(t, 10485760, fc.Spec.Processor.KafkaConsumerBatchSize)
}
