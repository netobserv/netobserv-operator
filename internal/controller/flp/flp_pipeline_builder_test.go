package flp

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/fmstatus"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// This function validate that each stage has its matching parameter
func validatePipelineConfig(t *testing.T, staticCm *corev1.ConfigMap, dynamicCm *corev1.ConfigMap) (*config.Root, string) {
	var cfs config.Root
	err := json.Unmarshal([]byte(staticCm.Data[configFile]), &cfs)
	assert.NoError(t, err)

	var dynCfs config.HotReloadStruct
	err = json.Unmarshal([]byte(dynamicCm.Data[configFile]), &dynCfs)
	assert.NoError(t, err)

	// Rearrange static+dynamic parameters with their correct index
	rebuilt := make([]config.StageParam, len(cfs.Parameters)+len(dynCfs.Parameters))
	cfs.Parameters = append(cfs.Parameters, dynCfs.Parameters...)

	for i, stage := range cfs.Pipeline {
		assert.NotEmpty(t, stage.Name)
		exist := false
		for _, parameter := range cfs.Parameters {
			if stage.Name == parameter.Name {
				rebuilt[i] = parameter
				exist = true
				break
			}
		}
		if !exist {
			for _, parameter := range dynCfs.Parameters {
				if stage.Name == parameter.Name {
					rebuilt[i] = parameter
					exist = true
					break
				}
			}
		}
		assert.True(t, exist, "stage params not found", stage.Name)
	}
	cfs.Parameters = rebuilt
	b, err := json.Marshal(cfs.Pipeline)
	assert.NoError(t, err)
	return &cfs, string(b)
}

func TestPipelineConfig(t *testing.T) {
	assert := assert.New(t)

	// Single config
	ns := "namespace"
	cfg := getConfig()
	cfg.Processor.LogLevel = "info"
	b := monoBuilder(ns, &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	_, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"extract_conntrack","follows":"grpc"},{"name":"enrich","follows":"extract_conntrack"},{"name":"loki","follows":"enrich"},{"name":"prometheus","follows":"enrich"}]`,
		pipeline,
	)

	// Kafka Transformer
	cfg.DeploymentModel = flowslatest.DeploymentModelKafka
	bt := transfBuilder(ns, &cfg)
	scm, _, dcm, err = bt.configMaps()
	assert.NoError(err)
	_, pipeline = validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"kafka-read"},{"name":"extract_conntrack","follows":"kafka-read"},{"name":"enrich","follows":"extract_conntrack"},{"name":"loki","follows":"enrich"},{"name":"prometheus","follows":"enrich"}]`,
		pipeline,
	)
}

func TestPipelineTraceStage(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	_, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"extract_conntrack","follows":"grpc"},{"name":"enrich","follows":"extract_conntrack"},{"name":"loki","follows":"enrich"},{"name":"stdout","follows":"enrich"},{"name":"prometheus","follows":"enrich"}]`,
		pipeline,
	)
}

func getSortedMetricsNames(m []api.MetricsItem) []string {
	ret := []string{}
	for i := range m {
		ret = append(ret, m[i].Name)
	}
	sort.Strings(ret)
	return ret
}

func buildAndGetMetricNames(t *testing.T, cfg *flowslatest.FlowCollectorSpec) ([]string, *config.Root) {
	b := monoBuilder("namespace", cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(t, err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	names := getSortedMetricsNames(cfs.Parameters[5].Encode.Prom.Metrics)
	return names, cfs
}

func TestMergeMetricsConfiguration_Default(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	names := getSortedMetricsNames(cfs.Parameters[5].Encode.Prom.Metrics)
	assert.Equal([]string{
		"namespace_flows_total",
		"namespace_ingress_packets_total",
		"node_egress_bytes_total",
		"node_ingress_bytes_total",
		"node_ingress_packets_total",
		"node_to_node_ingress_flows_total",
		"workload_egress_bytes_total",
		"workload_ingress_bytes_total",
	}, names)
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestMergeMetricsConfiguration_DefaultWithFeatures(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Agent.EBPF.Privileged = true
	cfg.Agent.EBPF.Features = []flowslatest.AgentFeature{flowslatest.DNSTracking, flowslatest.FlowRTT, flowslatest.PacketDrop}

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	names := getSortedMetricsNames(cfs.Parameters[5].Encode.Prom.Metrics)
	assert.Equal([]string{
		"namespace_dns_latency_seconds",
		"namespace_drop_packets_total",
		"namespace_flows_total",
		"namespace_ingress_packets_total",
		"namespace_rtt_seconds",
		"node_drop_packets_total",
		"node_egress_bytes_total",
		"node_ingress_bytes_total",
		"node_ingress_packets_total",
		"node_to_node_ingress_flows_total",
		"workload_egress_bytes_total",
		"workload_ingress_bytes_total",
	}, names)
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestMergeMetricsConfiguration_WithList(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.IncludeList = &[]flowslatest.FLPMetric{"namespace_egress_bytes_total", "namespace_ingress_bytes_total"}

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	names := getSortedMetricsNames(cfs.Parameters[5].Encode.Prom.Metrics)
	assert.Len(names, 2)
	assert.Equal("namespace_egress_bytes_total", names[0])
	assert.Equal("namespace_ingress_bytes_total", names[1])
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestMergeMetricsConfiguration_WithFlowMetrics(t *testing.T) {
	assert := assert.New(t)

	metrics := metricslatest.FlowMetricList{
		Items: []metricslatest.FlowMetric{
			{
				ObjectMeta: v1.ObjectMeta{Name: "te-st"},
				Spec:       metricslatest.FlowMetricSpec{Type: metricslatest.CounterMetric},
			},
		},
	}

	cfg := getConfig()
	cfg.Processor.Metrics.IncludeList = &[]flowslatest.FLPMetric{"namespace_ingress_bytes_total"}
	fmstatus.Reset()

	b := monoBuilderWithMetrics("namespace", &cfg, &metrics)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	names := getSortedMetricsNames(cfs.Parameters[5].Encode.Prom.Metrics)
	assert.Equal([]string{"namespace_ingress_bytes_total", "te_st"}, names)
	assert.Equal("netobserv_te_st", metrics.Items[0].Status.PrometheusName)
}

func TestMergeMetricsConfiguration_EmptyList(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.IncludeList = &[]flowslatest.FLPMetric{}

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, _ := validatePipelineConfig(t, scm, dcm)
	assert.Len(cfs.Parameters, 5)
}

func TestMergeMetricsConfiguration_WithAdditionalList(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.AdditionalIncludeList = &[]flowslatest.FLPMetric{"namespace_egress_bytes_total", "namespace_ingress_bytes_total"}

	names, cfs := buildAndGetMetricNames(t, &cfg)
	// Should have both defaults and additional metrics
	assert.Contains(names, "namespace_egress_bytes_total")
	assert.Contains(names, "namespace_ingress_bytes_total")
	// Should also have some default metrics
	assert.Contains(names, "node_ingress_bytes_total")
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestMergeMetricsConfiguration_WithAdditionalList_Empty(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.AdditionalIncludeList = &[]flowslatest.FLPMetric{}

	names, _ := buildAndGetMetricNames(t, &cfg)
	// Empty list should still have defaults
	assert.NotEmpty(names)
	assert.Contains(names, "node_ingress_bytes_total")
}

func TestMergeMetricsConfiguration_WithAdditionalList_Nil(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.AdditionalIncludeList = nil

	names, _ := buildAndGetMetricNames(t, &cfg)
	// Nil list should still have defaults
	assert.NotEmpty(names)
	assert.Contains(names, "node_ingress_bytes_total")
}

func TestMergeMetricsConfiguration_WithAdditionalList_Dedup(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.AdditionalIncludeList = &[]flowslatest.FLPMetric{
		"node_ingress_bytes_total",     // Already in defaults
		"namespace_egress_bytes_total", // New metric
		"namespace_egress_bytes_total", // Duplicate in additional
		"node_ingress_bytes_total",     // Duplicate overlap with default
	}

	names, cfs := buildAndGetMetricNames(t, &cfg)

	// Count occurrences to verify deduplication
	nodeIngressCount := 0
	namespaceEgressCount := 0
	for _, n := range names {
		if n == "node_ingress_bytes_total" {
			nodeIngressCount++
		}
		if n == "namespace_egress_bytes_total" {
			namespaceEgressCount++
		}
	}

	// Each metric should appear exactly once
	assert.Equal(1, nodeIngressCount, "node_ingress_bytes_total should appear exactly once")
	assert.Equal(1, namespaceEgressCount, "namespace_egress_bytes_total should appear exactly once")
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestMergeMetricsConfiguration_WithBothLists(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Processor.Metrics.IncludeList = &[]flowslatest.FLPMetric{"node_ingress_bytes_total"}
	cfg.Processor.Metrics.AdditionalIncludeList = &[]flowslatest.FLPMetric{"namespace_egress_bytes_total"}

	names, cfs := buildAndGetMetricNames(t, &cfg)
	// When both are set, includeList takes precedence and additionalIncludeList is ignored
	assert.Len(names, 1)
	assert.Equal("node_ingress_bytes_total", names[0])
	assert.Equal("netobserv_", cfs.Parameters[5].Encode.Prom.Prefix)
}

func TestPipelineWithExporter(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Exporters = append(cfg.Exporters, &flowslatest.FlowCollectorExporter{
		Type:  flowslatest.KafkaExporter,
		Kafka: flowslatest.FlowCollectorKafka{Address: "kafka-test", Topic: "topic-test"},
	})

	cfg.Exporters = append(cfg.Exporters, &flowslatest.FlowCollectorExporter{
		Type: flowslatest.IpfixExporter,
		IPFIX: flowslatest.FlowCollectorIPFIXReceiver{
			TargetHost: "ipfix-receiver-test",
			TargetPort: 9999,
			Transport:  "TCP",
		},
	})

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"extract_conntrack","follows":"grpc"},{"name":"enrich","follows":"extract_conntrack"},{"name":"loki","follows":"enrich"},{"name":"stdout","follows":"enrich"},{"name":"prometheus","follows":"enrich"},{"name":"kafka-export-0","follows":"enrich"},{"name":"ipfix-export-1","follows":"enrich"}]`,
		pipeline,
	)

	assert.Equal("kafka-test", cfs.Parameters[6].Encode.Kafka.Address)
	assert.Equal("topic-test", cfs.Parameters[6].Encode.Kafka.Topic)

	assert.Equal("ipfix-receiver-test", cfs.Parameters[7].Write.Ipfix.TargetHost)
	assert.Equal(9999, cfs.Parameters[7].Write.Ipfix.TargetPort)
	assert.Equal("tcp", cfs.Parameters[7].Write.Ipfix.Transport)
}

func TestPipelineWithoutLoki(t *testing.T) {
	assert := assert.New(t)

	cfg := getConfig()
	cfg.Loki.Enable = ptr.To(false)

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	_, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"extract_conntrack","follows":"grpc"},{"name":"enrich","follows":"extract_conntrack"},{"name":"stdout","follows":"enrich"},{"name":"prometheus","follows":"enrich"}]`,
		pipeline,
	)
}

func TestReadMachineNetworks(t *testing.T) {
	cm := corev1.ConfigMap{
		Data: map[string]string{
			"install-config": `
additionalTrustBundlePolicy: Proxyonly
apiVersion: v1
baseDomain: my.openshift.com
compute:
- architecture: amd64
  hyperthreading: Enabled
  name: worker
  platform: {}
  replicas: 3
controlPlane:
  architecture: amd64
  hyperthreading: Enabled
  name: master
  platform: {}
  replicas: 3
metadata:
  creationTimestamp: null
  name: my-cluster
networking:
  clusterNetwork:
  - cidr: 10.128.0.0/14
    hostPrefix: 23
  machineNetwork:
  - cidr: 10.0.0.0/16
  networkType: OVNKubernetes
  serviceNetwork:
  - 172.30.0.0/16
platform:
  aws:
    region: eu-west-3
publish: External`,
		},
	}

	machines, err := readMachineFromConfig(&cm)
	assert.NoError(t, err)

	assert.Equal(t, []string{"10.0.0.0/16"}, machines)
}

func TestPipelineWithSubnetLabels(t *testing.T) {
	assert := assert.New(t)

	cfg := flowslatest.FlowCollectorSpec{
		Processor: flowslatest.FlowCollectorFLP{
			SubnetLabels: flowslatest.SubnetLabels{
				OpenShiftAutoDetect: ptr.To(true),
				CustomLabels: []flowslatest.SubnetLabel{
					{
						Name:  "Foo",
						CIDRs: []string{"8.8.8.8/32"},
					},
				},
			},
		},
		Loki: flowslatest.FlowCollectorLoki{Enable: ptr.To(false)},
	}

	b := monoBuilder("namespace", &cfg)
	b.detectedSubnets = []flowslatest.SubnetLabel{
		{
			Name:  "Pods",
			CIDRs: []string{"10.128.0.0/14"},
		},
	}
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"enrich","follows":"grpc"},{"name":"subnets","follows":"enrich"},{"name":"prometheus","follows":"subnets"}]`,
		pipeline,
	)
	assert.Equal(
		[]api.NetworkTransformSubnetLabel{
			{
				Name:  "Foo",
				CIDRs: []string{"8.8.8.8/32"},
			},
			{
				Name:  "Pods",
				CIDRs: []string{"10.128.0.0/14"},
			},
		},
		cfs.Parameters[2].Transform.Network.SubnetLabels,
	)
}

func TestPipelineWithFilters_WantNamespacesABC(t *testing.T) {
	assert := assert.New(t)

	cfg := flowslatest.FlowCollectorSpec{
		Processor: flowslatest.FlowCollectorFLP{
			Filters: []flowslatest.FLPFilterSet{
				{
					Query:        `namespace="A"`,
					OutputTarget: flowslatest.FLPFilterTargetAll,
				},
				{
					Query:        `namespace="B"`,
					OutputTarget: flowslatest.FLPFilterTargetAll,
				},
				{
					Query:        `namespace="C"`,
					OutputTarget: flowslatest.FLPFilterTargetAll,
				},
			},
		},
	}

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"enrich","follows":"grpc"},{"name":"filters","follows":"enrich"},{"name":"loki","follows":"filters"},{"name":"prometheus","follows":"filters"}]`,
		pipeline,
	)
	assert.Equal(
		[]api.TransformFilterRule{
			{
				Type:           api.KeepEntryQuery,
				KeepEntryQuery: `namespace="A"`,
			},
			{
				Type:           api.KeepEntryQuery,
				KeepEntryQuery: `namespace="B"`,
			},
			{
				Type:           api.KeepEntryQuery,
				KeepEntryQuery: `namespace="C"`,
			},
		},
		cfs.Parameters[2].Transform.Filter.Rules,
	)
}

func TestPipelineWithFilters_DontWantNamespacesABC_LokiOnly(t *testing.T) {
	assert := assert.New(t)

	cfg := flowslatest.FlowCollectorSpec{
		Processor: flowslatest.FlowCollectorFLP{
			Filters: []flowslatest.FLPFilterSet{
				{
					Query:        `namespace!="A" and namespace!="B" and namespace!="C"`,
					OutputTarget: flowslatest.FLPFilterTargetLoki,
				},
			},
		},
	}

	b := monoBuilder("namespace", &cfg)
	scm, _, dcm, err := b.configMaps()
	assert.NoError(err)
	cfs, pipeline := validatePipelineConfig(t, scm, dcm)
	assert.Equal(
		`[{"name":"grpc"},{"name":"enrich","follows":"grpc"},{"name":"filters-loki","follows":"enrich"},{"name":"loki","follows":"filters-loki"},{"name":"prometheus","follows":"enrich"}]`,
		pipeline,
	)
	assert.Equal(
		[]api.TransformFilterRule{
			{
				Type:           api.KeepEntryQuery,
				KeepEntryQuery: `namespace!="A" and namespace!="B" and namespace!="C"`,
			},
		},
		cfs.Parameters[2].Transform.Filter.Rules,
	)
}
