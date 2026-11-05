package v1beta2

import (
	"time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
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
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
				},
				{
					Thresholds: HealthRuleThresholds{
						Info:    "100",
						Warning: "500",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
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
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
				},
				{
					Thresholds: HealthRuleThresholds{
						Info:    "100",
						Warning: "500",
					},
					GroupBy:       GroupByNamespace,
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
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
					TrendOffset:   &v1.Duration{Duration: 24 * time.Hour},
					TrendDuration: &v1.Duration{Duration: time.Hour},
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
)
