package v1beta2

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	// See spec.profile
	HighProfileFeatures = []AgentFeature{
		DNSTracking,
		FlowRTT,
		PacketDrop,
		SecondaryNetworks,
		TLSTracking,
		UDNMapping,
	}

	// Note that we set default in-code rather than in CRD, in order to keep track of value being unset or set intentionnally in FlowCollector
	DefaultIncludeList = []string{
		"node_ingress_bytes_total",
		"node_egress_bytes_total",
		"node_ingress_packets_total",
		"node_drop_packets_total",
		"workload_sampling",
		"workload_ingress_bytes_total",
		"workload_egress_bytes_total",
		"namespace_flows_total",
		"namespace_ingress_packets_total",
		"namespace_drop_packets_total",
		"namespace_rtt_seconds",
		"namespace_dns_latency_seconds",
		"namespace_dns_flows_total",
		"namespace_network_policy_events_total",
		"namespace_tls_flows_total",
		"node_ipsec_flows_total",
		"node_to_node_ingress_flows_total",
	}
	// More metrics enabled when Loki is disabled, to avoid loss of information
	DefaultIncludeListLokiDisabled = []string{
		"node_ingress_bytes_total",
		"node_egress_bytes_total",
		"node_ingress_packets_total",
		"node_drop_packets_total",
		"workload_ingress_bytes_total",
		"workload_egress_bytes_total",
		"workload_sampling",
		"workload_ingress_packets_total",
		"workload_egress_packets_total",
		"workload_flows_total",
		"workload_drop_bytes_total",
		"workload_drop_packets_total",
		"workload_rtt_seconds",
		"workload_dns_latency_seconds",
		"workload_dns_flows_total",
		"workload_tls_flows_total",
		"namespace_network_policy_events_total",
		"node_ipsec_flows_total",
		"node_to_node_ingress_flows_total",
	}
	DefaultHealthRules = []FLPHealthRule{
		{
			Template: HealthRulePacketDropsByKernel,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "10",
						Warning: "20",
					},
					LowVolumeThreshold: "5",
					GroupBy:            GroupByNamespace,
				},
				{
					Mode: ptr.To(ModeRecording),
					Thresholds: HealthRuleThresholds{
						Info:    "5",
						Warning: "15",
					},
					GroupBy: GroupByNode,
				},
			},
		},
		{
			Template: HealthRulePacketDropsByDevice,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "5",
						Warning: "10",
					},
					GroupBy: GroupByNode,
				},
			},
		},
		{
			Template: HealthRuleIPsecErrors,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Warning: "2",
					},
				},
				{
					Thresholds: HealthRuleThresholds{
						Warning: "2",
					},
					GroupBy: GroupByNode,
				},
			},
		},
		{
			Template: HealthRuleDNSErrors,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Warning: "5",
					},
				},
				{
					Thresholds: HealthRuleThresholds{
						Info:    "5",
						Warning: "10",
					},
					GroupBy: GroupByNamespace,
				},
			},
		},
		{
			Template: HealthRuleDNSNxDomain,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "10",
						Warning: "80",
					},
					GroupBy: GroupByNamespace,
				},
			},
		},
		{
			Template: HealthRuleNetpolDenied,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "5",
						Warning: "10",
					},
					GroupBy: GroupByNamespace,
				},
			},
		},
		{
			Template: HealthRuleLatencyHighTrend,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info: "100",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
			},
		},
		{
			Template: HealthRuleExternalEgressHighTrend,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Warning: "200",
					},
					GroupBy:       GroupByNode,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
				{
					Thresholds: HealthRuleThresholds{
						Info:    "100",
						Warning: "500",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
			},
		},
		{
			Template: HealthRuleExternalIngressHighTrend,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Warning: "200",
					},
					GroupBy:       GroupByNode,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
				{
					Thresholds: HealthRuleThresholds{
						Info:    "100",
						Warning: "500",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
			},
		},
		{
			Template: HealthRuleIngress5xxErrors,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "5",
						Warning: "10",
					},
					GroupBy: GroupByNamespace,
				},
			},
		},
		{
			Template: HealthRuleIngressHTTPLatencyTrend,
			Mode:     ModeRecording,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Info:    "100",
						Warning: "200",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &metav1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &metav1.Duration{Duration: time.Hour},
				},
			},
		},
		{
			Template: HealthRuleTLSInsecureVersion,
			Variants: []HealthRuleVariant{
				{
					Thresholds: HealthRuleThresholds{
						Warning: "5",
					},
					GroupBy: GroupByNamespace,
				},
			},
		},
	}

	// DefaultFlowCollector returns a default FlowCollectorSpec. It's used instead of kubebuilder defaults for several reasons:
	//   - with kubebuilder defaults, the CR is automatically expanded with the defaults. A minimalist CR instantly becomes huge once it's applied on the cluster,
	//     with a lot of fields that are not relevant (e.g. Kafka config is visible despite not using Kafka). It becomes super noisy.
	//   - there are already in-code defaults for some settings that have contextual defaults. E.g. netpol installed by default only when ovn-k is found, etc.
	//     Having everything in-code makes the whole more consistent.
	//   - in-code allows to change the defaults on upgrade when needed, which API-based default doesn't allow
	// Make sure every added field has the "omitempty" json tag (see helper.EnsureDefaults)
	DefaultFlowCollector = &FlowCollectorSpec{
		DeploymentModel: DeploymentModelService,
		Agent: FlowCollectorAgent{
			EBPF: FlowCollectorEBPF{
				LogLevel: "info",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("50Mi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("800Mi"),
					},
				},
				CacheActiveTimeout: "15s",
				ExcludeInterfaces:  []string{"lo"},
				KafkaBatchSize:     1048576,
			},
		},
		Processor: FlowCollectorFLP{
			LogLevel: "info",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("100Mi"),
					corev1.ResourceCPU:    resource.MustParse("100m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("800Mi"),
				},
			},
			KafkaConsumerQueueCapacity: 1000,
			KafkaConsumerBatchSize:     10485760,
			LogTypes:                   ptr.To(LogTypeFlows),
			Service: &ProcessorServiceConfig{
				TLSType: TLSAuto,
			},
		},
		ConsolePlugin: FlowCollectorConsolePlugin{
			Enable:   ptr.To(true),
			LogLevel: "info",
			Replicas: ptr.To(int32(1)),
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("50Mi"),
					corev1.ResourceCPU:    resource.MustParse("100m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: resource.MustParse("100Mi"),
				},
			},
			PortNaming: ConsolePluginPortConfig{
				Enable: ptr.To(true),
			},
			QuickFilters: []QuickFilter{
				{
					Name:    "Applications",
					Filter:  map[string]string{"flow_layer": `"app"`},
					Default: true,
				},
				{
					Name:   "Infrastructure",
					Filter: map[string]string{"flow_layer": `"infra"`},
				},
				{
					Name:    "Pods network",
					Filter:  map[string]string{"src_kind": `"Pod"`, "dst_kind": `"Pod"`},
					Default: true,
				},
				{
					Name:   "Services network",
					Filter: map[string]string{"dst_kind": `"Service"`},
				},
				{
					Name:   "External ingress",
					Filter: map[string]string{"src_subnet_label": `"",EXT:`},
				},
				{
					Name:   "External egress",
					Filter: map[string]string{"dst_subnet_label": `"",EXT:`},
				},
			},
		},
		Loki: FlowCollectorLoki{
			ReadTimeout:    &metav1.Duration{Duration: 30 * time.Second},
			WriteTimeout:   &metav1.Duration{Duration: 10 * time.Second},
			WriteBatchWait: &metav1.Duration{Duration: 1 * time.Second},
			WriteBatchSize: 10485760,
			Mode:           LokiModeMonolithic,
			Monolithic: LokiMonolithParams{
				URL:      "http://loki:3100/",
				TenantID: "netobserv",
			},
			Microservices: LokiMicroservicesParams{
				IngesterURL: "http://loki-distributor:3100/",
				QuerierURL:  "http://loki-query-frontend:3100/",
				TenantID:    "netobserv",
			},
			LokiStack: LokiStackRef{
				Name: "loki",
			},
			Manual: LokiManualParams{
				IngesterURL: "http://loki:3100/",
				QuerierURL:  "http://loki:3100/",
				TenantID:    "netobserv",
			},
		},
		Prometheus: FlowCollectorPrometheus{
			Querier: PrometheusQuerier{
				Enable:  ptr.To(true),
				Mode:    PromModeAuto,
				Timeout: &metav1.Duration{Duration: 30 * time.Second},
				Manual:  PrometheusQuerierManual{URL: "http://prometheus:9090"},
			},
		},
		Kafka: FlowCollectorKafka{
			Compression: "none",
		},
		NetworkPolicy: NetworkPolicy{},
		Execution:     FlowCollectorExecution{},
	}
)
