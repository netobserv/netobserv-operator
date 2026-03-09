package alerts

import (
	"context"
	"slices"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
)

func allTemplates() []flowslatest.HealthRuleTemplate {
	return []flowslatest.HealthRuleTemplate{
		flowslatest.AlertLokiError,
		flowslatest.AlertNoFlows,
		flowslatest.HealthRulePacketDropsByKernel,
		flowslatest.HealthRulePacketDropsByDevice,
		flowslatest.HealthRuleDNSErrors,
		flowslatest.HealthRuleDNSNxDomain,
		flowslatest.HealthRuleIPsecErrors,
		flowslatest.HealthRuleLatencyHighTrend,
		flowslatest.HealthRuleNetpolDenied,
		flowslatest.HealthRuleExternalEgressHighTrend,
		flowslatest.HealthRuleExternalIngressHighTrend,
		flowslatest.HealthRuleIngress5xxErrors,
		flowslatest.HealthRuleIngressHTTPLatencyTrend,
	}
}

func allTemplatesBut(tpls ...flowslatest.AlertTemplate) []flowslatest.AlertTemplate {
	var ret []flowslatest.AlertTemplate
	for _, tpl := range allTemplates() {
		if !slices.Contains(tpls, tpl) {
			ret = append(ret, tpl)
		}
	}
	return ret
}

func TestBuildRules_DefaultWithDisabled(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: []flowslatest.AlertTemplate{
					flowslatest.AlertLokiError,
					flowslatest.HealthRulePacketDropsByDevice,
					flowslatest.HealthRuleExternalEgressHighTrend,
					flowslatest.HealthRuleExternalIngressHighTrend,
					flowslatest.HealthRuleIngress5xxErrors,
					flowslatest.HealthRuleIngressHTTPLatencyTrend,
				},
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Len(t, rules, 1)
	assert.Contains(t, rules[0].Annotations["description"], "NetObserv flowlogs-pipeline is not receiving any flow")
}

func allNames(rules []monitoringv1.Rule) []string {
	var names []string
	for _, r := range rules {
		if r.Alert != "" {
			names = append(names, r.Alert)
		} else {
			names = append(names, r.Record)
		}
	}
	return names
}

func TestBuildRules_DefaultWithFeaturesAndDisabled(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features: []flowslatest.AgentFeature{
					flowslatest.FlowRTT,
					flowslatest.DNSTracking,
					flowslatest.IPSec,
					flowslatest.NetworkEvents,
					flowslatest.PacketDrop,
				},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: []flowslatest.AlertTemplate{
					flowslatest.AlertLokiError,
					flowslatest.HealthRuleExternalEgressHighTrend,
					flowslatest.HealthRuleExternalIngressHighTrend,
				},
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Equal(t, []string{
		"netobserv:health:packet_drops_kernel:namespace:src:rate2m",
		"netobserv:health:packet_drops_kernel:namespace:dst:rate2m",
		"netobserv:health:packet_drops_kernel:node:src:rate2m",
		"netobserv:health:packet_drops_kernel:node:dst:rate2m",
		"PacketDropsByDevice_PerNodeWarning",
		"PacketDropsByDevice_PerNodeInfo",
		"IPsecErrors_Warning",
		"IPsecErrors_PerSrcNodeWarning",
		"IPsecErrors_PerDstNodeWarning",
		"DNSErrors_Warning",
		"DNSErrors_PerDstNamespaceWarning",
		"DNSErrors_PerDstNamespaceInfo",
		"netobserv:health:dns_nxdomain:namespace:dst:rate2m",
		"netobserv:health:netpol_denied:namespace:src:rate2m",
		"netobserv:health:netpol_denied:namespace:dst:rate2m",
		"netobserv:health:tcp_latency_increase_p90:namespace:src:rate2m",
		"netobserv:health:tcp_latency_increase_p90:namespace:dst:rate2m",
		"netobserv:health:ingress_5xx_errors:namespace:src:rate2m",
		"netobserv:health:ingress_http_latency_increase_avg:namespace:src:rate2m",
		"NetObservNoFlows",
	}, allNames(rules))
	r := findRule("PacketDropsByDevice_PerNodeWarning", rules)
	assert.NotNil(t, r)
	assert.Equal(t, "node-exporter is reporting more than 10% of dropped packets [node={{ $labels.instance }}].", r.Annotations["description"])
	assert.Equal(t, `{"alertThreshold":"10","unit":"%","nodeLabels":["instance"]}`, r.Annotations["netobserv_io_network_health"])
	r = findRule("IPsecErrors_Warning", rules)
	assert.NotNil(t, r)
	assert.Equal(t, "NetObserv is detecting more than 2% of IPsec errors.", r.Annotations["description"])
	assert.Equal(t, `{"alertThreshold":"2","unit":"%"}`, r.Annotations["netobserv_io_network_health"])
	r = findRule("DNSErrors_PerDstNamespaceWarning", rules)
	assert.NotNil(t, r)
	assert.Equal(t, "NetObserv is detecting more than 10% of DNS errors [dest. namespace={{ $labels.namespace }}] (other than NX_DOMAIN).", r.Annotations["description"])
	assert.Contains(t, r.Annotations["netobserv_io_network_health"], `{"alertThreshold":"10","unit":"%","namespaceLabels":["namespace"]`)
	r = findRule("NetObservNoFlows", rules)
	assert.NotNil(t, r)
	assert.Contains(t, r.Annotations["description"], "NetObserv flowlogs-pipeline is not receiving any flow")
}

func findRule(name string, rules []monitoringv1.Rule) *monitoringv1.Rule {
	for i := range rules {
		if rules[i].Alert == name {
			return &rules[i]
		}
	}
	return nil
}

func TestBuildRules_DefaultWithFeaturesAndAllDisabled(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.IPSec, flowslatest.NetworkEvents, flowslatest.PacketDrop},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: allTemplates(),
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Empty(t, rules)
}

func TestBuildRules_Overidden(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.IPSec, flowslatest.NetworkEvents, flowslatest.PacketDrop},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: allTemplatesBut(flowslatest.HealthRulePacketDropsByKernel),
				HealthRules: &[]flowslatest.FLPHealthRule{
					{
						Template: flowslatest.HealthRulePacketDropsByKernel,
						Variants: []flowslatest.HealthRuleVariant{
							{
								Thresholds: flowslatest.HealthRuleThresholds{
									Critical: "50",
								},
								GroupBy: flowslatest.GroupByWorkload,
							},
						},
					},
				},
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Len(t, rules, 2)
	assert.Contains(t, rules[0].Annotations["description"], "NetObserv is detecting more than 50% of packets dropped by the kernel [source workload={{ $labels.workload }} ({{ $labels.kind }})]")
	assert.Contains(t, rules[1].Annotations["description"], "NetObserv is detecting more than 50% of packets dropped by the kernel [dest. workload={{ $labels.workload }} ({{ $labels.kind }})]")
}

func TestBuildRules_Global(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.IPSec, flowslatest.NetworkEvents, flowslatest.PacketDrop},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: allTemplatesBut(flowslatest.HealthRulePacketDropsByKernel),
				HealthRules: &[]flowslatest.FLPHealthRule{
					{
						Template: flowslatest.HealthRulePacketDropsByKernel,
						Variants: []flowslatest.HealthRuleVariant{
							{
								Thresholds: flowslatest.HealthRuleThresholds{
									Critical: "50",
								},
							},
						},
					},
				},
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Len(t, rules, 1)
	assert.Contains(t, rules[0].Annotations["description"], "NetObserv is detecting more than 50% of packets dropped by the kernel.")
	assert.Equal(t, "100 * (sum(rate(netobserv_namespace_drop_packets_total[2m]))) / (sum(rate(netobserv_namespace_ingress_packets_total[2m]))) > 50", rules[0].Expr.StrVal)
}

func TestBuildRules_DisableTakesPrecedence(t *testing.T) {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.IPSec, flowslatest.NetworkEvents, flowslatest.PacketDrop},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{
				DisableAlerts: allTemplates(),
				HealthRules: &[]flowslatest.FLPHealthRule{
					{
						Template: flowslatest.HealthRulePacketDropsByKernel,
						Variants: []flowslatest.HealthRuleVariant{
							{
								Thresholds: flowslatest.HealthRuleThresholds{
									Critical: "50",
								},
								GroupBy: flowslatest.GroupByWorkload,
							},
						},
					},
				},
			},
		},
	}
	rules := BuildMonitoringRules(context.Background(), &fc)
	assert.Empty(t, rules)
}

func TestLatencyPromql(t *testing.T) {
	variant := flowslatest.HealthRuleVariant{
		GroupBy: flowslatest.GroupByNamespace,
		Thresholds: flowslatest.HealthRuleThresholds{
			Info: "100",
		},
	}
	rules, err := buildHealthRulesForVariant(flowslatest.HealthRuleLatencyHighTrend, flowslatest.ModeAlert, &variant, []string{"namespace_rtt_seconds"})
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
	anns, err := rules[0].GetAnnotations()
	assert.NoError(t, err)
	assert.Contains(t, anns["description"], "NetObserv is detecting TCP latency increased by more than 100% [source namespace={{ $labels.namespace }}], compared to baseline (offset: 24h).")
	assert.Equal(t, `{"alertThreshold":"100","upperBound":"500","unit":"%","namespaceLabels":["namespace"]}`, anns["netobserv_io_network_health"])

	mr, err := rules[0].Build()
	assert.NoError(t, err)
	// The pattern is:
	// 100 * (<current latency> - <past latency>) / <past latency>
	assert.Equal(t,
		`100 * `+
			`((histogram_quantile(0.9, `+
			`sum(label_replace(rate(netobserv_namespace_rtt_seconds_bucket{SrcK8S_Namespace!=""}[2m]), "namespace", "$1", "SrcK8S_Namespace", "(.*)")) by (namespace,le)))`+
			` - (histogram_quantile(0.9, `+
			`sum(label_replace(rate(netobserv_namespace_rtt_seconds_bucket{SrcK8S_Namespace!=""}[2h] offset 24h), "namespace", "$1", "SrcK8S_Namespace", "(.*)")) by (namespace,le))))`+
			` / (histogram_quantile(0.9, `+
			`sum(label_replace(rate(netobserv_namespace_rtt_seconds_bucket{SrcK8S_Namespace!=""}[2h] offset 24h), "namespace", "$1", "SrcK8S_Namespace", "(.*)")) by (namespace,le)))`+
			` > 100`,
		mr.Expr.StrVal,
	)
}

func TestAllAlertsHaveRunbookURL(t *testing.T) {
	// Create a FlowCollector with all features enabled
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features: []flowslatest.AgentFeature{
					flowslatest.FlowRTT,
					flowslatest.DNSTracking,
					flowslatest.IPSec,
					flowslatest.NetworkEvents,
					flowslatest.PacketDrop,
				},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics: flowslatest.FLPMetrics{},
		},
	}

	rules := BuildMonitoringRules(context.Background(), &fc)

	// Verify all rules have a runbook_url annotation
	for _, rule := range rules {
		if rule.Alert != "" {
			url := rule.Annotations["runbook_url"]
			assert.Contains(t, url, runbookURLBase+"/", "Alert %s has invalid runbook_url: %s", rule.Alert, url)
			assert.Contains(t, url, ".md", "Alert %s runbook_url doesn't end with .md: %s", rule.Alert, url)
		}
	}
}
