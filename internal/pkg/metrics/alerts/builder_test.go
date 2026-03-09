package alerts

import (
	"net/http"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
)

func TestBuildLabelFilter(t *testing.T) {
	// Test GroupByNode with source side
	ctx := &ruleContext{
		healthRule: &flowslatest.HealthRuleVariant{
			GroupBy: flowslatest.GroupByNode,
		},
		side: asSource,
	}
	filter := getPromQLFilters(ctx, "")
	assert.Equal(t, `{SrcK8S_HostName!=""}`, filter)

	// Test GroupByNode with destination side
	ctx.side = asDest
	filter = getPromQLFilters(ctx, "")
	assert.Equal(t, `{DstK8S_HostName!=""}`, filter)

	// Test GroupByNamespace
	ctx.healthRule.GroupBy = flowslatest.GroupByNamespace
	ctx.side = asSource
	filter = getPromQLFilters(ctx, "")
	assert.Equal(t, `{SrcK8S_Namespace!=""}`, filter)

	// Test GroupByWorkload
	ctx.healthRule.GroupBy = flowslatest.GroupByWorkload
	ctx.side = asDest
	filter = getPromQLFilters(ctx, "")
	assert.Equal(t, `{DstK8S_Namespace!="",DstK8S_OwnerName!="",DstK8S_OwnerType!=""}`, filter)

	// Test with additional filter
	ctx.healthRule.GroupBy = flowslatest.GroupByNamespace
	ctx.side = asSource
	filter = getPromQLFilters(ctx, `DnsFlagsResponseCode!="NoError"`)
	assert.Equal(t, `{SrcK8S_Namespace!="",DnsFlagsResponseCode!="NoError"}`, filter)

	// Test with action filter (netpol)
	ctx.healthRule.GroupBy = flowslatest.GroupByWorkload
	ctx.side = asDest
	filter = getPromQLFilters(ctx, `action="drop"`)
	assert.Equal(t, `{DstK8S_Namespace!="",DstK8S_OwnerName!="",DstK8S_OwnerType!="",action="drop"}`, filter)

	// Test no grouping (global)
	ctx.healthRule.GroupBy = ""
	ctx.side = ""
	filter = getPromQLFilters(ctx, "")
	assert.Equal(t, "", filter)

	// Test no grouping with additional filter
	filter = getPromQLFilters(ctx, `DnsFlagsResponseCode!="NoError"`)
	assert.Equal(t, `{DnsFlagsResponseCode!="NoError"}`, filter)
}

func TestRecordingRuleNames(t *testing.T) {
	tests := []struct {
		name     string
		template flowslatest.HealthRuleTemplate
		groupBy  flowslatest.HealthRuleGroupBy
		side     srcOrDst
		expected string
	}{
		// DNS Errors
		{
			name:     "DNSErrors no grouping",
			template: flowslatest.HealthRuleDNSErrors,
			groupBy:  "",
			side:     asDest,
			expected: "netobserv:health:dns_errors:rate2m",
		},
		{
			name:     "DNSErrors by Namespace",
			template: flowslatest.HealthRuleDNSErrors,
			groupBy:  flowslatest.GroupByNamespace,
			side:     asDest,
			expected: "netobserv:health:dns_errors:namespace:dst:rate2m",
		},
		{
			name:     "DNSErrors by Node",
			template: flowslatest.HealthRuleDNSErrors,
			groupBy:  flowslatest.GroupByNode,
			side:     asDest,
			expected: "netobserv:health:dns_errors:node:dst:rate2m",
		},
		{
			name:     "DNSErrors by Workload",
			template: flowslatest.HealthRuleDNSErrors,
			groupBy:  flowslatest.GroupByWorkload,
			side:     asDest,
			expected: "netobserv:health:dns_errors:workload:dst:rate2m",
		},

		// Packet Drops By Kernel
		{
			name:     "PacketDropsByKernel no grouping",
			template: flowslatest.HealthRulePacketDropsByKernel,
			groupBy:  "",
			side:     asSource,
			expected: "netobserv:health:packet_drops_kernel:rate2m",
		},
		{
			name:     "PacketDropsByKernel by Namespace src",
			template: flowslatest.HealthRulePacketDropsByKernel,
			groupBy:  flowslatest.GroupByNamespace,
			side:     asSource,
			expected: "netobserv:health:packet_drops_kernel:namespace:src:rate2m",
		},
		{
			name:     "PacketDropsByKernel by Namespace dst",
			template: flowslatest.HealthRulePacketDropsByKernel,
			groupBy:  flowslatest.GroupByNamespace,
			side:     asDest,
			expected: "netobserv:health:packet_drops_kernel:namespace:dst:rate2m",
		},

		// IPsec Errors
		{
			name:     "IPsecErrors no grouping",
			template: flowslatest.HealthRuleIPsecErrors,
			groupBy:  "",
			side:     asSource,
			expected: "netobserv:health:ipsec_errors:rate2m",
		},
		{
			name:     "IPsecErrors by Node",
			template: flowslatest.HealthRuleIPsecErrors,
			groupBy:  flowslatest.GroupByNode,
			side:     asSource,
			expected: "netobserv:health:ipsec_errors:node:src:rate2m",
		},

		// Netpol Denied
		{
			name:     "NetpolDenied no grouping",
			template: flowslatest.HealthRuleNetpolDenied,
			groupBy:  "",
			side:     asSource,
			expected: "netobserv:health:netpol_denied:rate2m",
		},
		{
			name:     "NetpolDenied by Workload",
			template: flowslatest.HealthRuleNetpolDenied,
			groupBy:  flowslatest.GroupByWorkload,
			side:     asDest,
			expected: "netobserv:health:netpol_denied:workload:dst:rate2m",
		},

		// Latency High Trend
		{
			name:     "LatencyHighTrend no grouping",
			template: flowslatest.HealthRuleLatencyHighTrend,
			groupBy:  "",
			side:     asSource,
			expected: "netobserv:health:tcp_latency_increase_p90:rate2m",
		},
		{
			name:     "LatencyHighTrend by Node",
			template: flowslatest.HealthRuleLatencyHighTrend,
			groupBy:  flowslatest.GroupByNode,
			side:     asDest,
			expected: "netobserv:health:tcp_latency_increase_p90:node:dst:rate2m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &ruleContext{
				template: tt.template,
				healthRule: &flowslatest.HealthRuleVariant{
					GroupBy: tt.groupBy,
				},
				mode: flowslatest.ModeRecording,
				side: tt.side,
			}

			rule := ctx.toRule()
			assert.NotNil(t, rule)
			mr, err := rule.Build()
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, mr.Record)
		})
	}
}

func TestHealthAnnotationMetadata(t *testing.T) {
	tests := []struct {
		name                    string
		groupBy                 flowslatest.HealthRuleGroupBy
		expectedNodeLabels      []string
		expectedNamespaceLabels []string
		expectedWorkloadLabels  []string
		expectedKindLabels      []string
	}{
		{
			name:                    "Global (no grouping)",
			groupBy:                 "",
			expectedNodeLabels:      nil,
			expectedNamespaceLabels: nil,
			expectedWorkloadLabels:  nil,
		},
		{
			name:                    "GroupBy Node",
			groupBy:                 flowslatest.GroupByNode,
			expectedNodeLabels:      []string{"node"},
			expectedNamespaceLabels: nil,
			expectedWorkloadLabels:  nil,
		},
		{
			name:                    "GroupBy Namespace",
			groupBy:                 flowslatest.GroupByNamespace,
			expectedNodeLabels:      nil,
			expectedNamespaceLabels: []string{"namespace"},
			expectedWorkloadLabels:  nil,
		},
		{
			name:                    "GroupBy Workload",
			groupBy:                 flowslatest.GroupByWorkload,
			expectedNodeLabels:      nil,
			expectedNamespaceLabels: []string{"namespace"},
			expectedWorkloadLabels:  []string{"workload"},
			expectedKindLabels:      []string{"kind"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hCtx := &ruleContext{
				template: flowslatest.HealthRulePacketDropsByKernel,
				healthRule: &flowslatest.HealthRuleVariant{
					GroupBy: tt.groupBy,
					Thresholds: flowslatest.HealthRuleThresholds{
						Critical: "10",
					},
				},
				alertThreshold: "10",
				side:           asSource,
			}

			// Build the health annotation
			ha := newHealthAnnotation(hCtx)
			assert.NotNil(t, ha)

			assert.Equal(t, tt.expectedNodeLabels, ha.NodeLabels, "Expected nodeLabels=%s in annotation for %s", tt.expectedNodeLabels, tt.name)
			assert.Equal(t, tt.expectedNamespaceLabels, ha.NamespaceLabels, "Expected namespaceLabels=%s in annotation for %s", tt.expectedNamespaceLabels, tt.name)
			assert.Equal(t, tt.expectedWorkloadLabels, ha.WorkloadLabels, "Expected workloadLabels=%s in annotation for %s", tt.expectedWorkloadLabels, tt.name)
			assert.Equal(t, tt.expectedKindLabels, ha.KindLabels, "Expected kindLabels=%s in annotation for %s", tt.expectedKindLabels, tt.name)
		})
	}
}

func TestBuildRunbookURL(t *testing.T) {
	tests := []struct {
		template flowslatest.AlertTemplate
		expected string
	}{
		// Health Rule templates
		{
			template: "PacketDropsByKernel",
			expected: runbookURLBase + "/PacketDropsByKernel.md",
		},
		{
			template: "PacketDropsByDevice",
			expected: runbookURLBase + "/PacketDropsByDevice.md",
		},
		{
			template: "IPsecErrors",
			expected: runbookURLBase + "/IPsecErrors.md",
		},
		{
			template: "NetpolDenied",
			expected: runbookURLBase + "/NetpolDenied.md",
		},
		{
			template: "LatencyHighTrend",
			expected: runbookURLBase + "/LatencyHighTrend.md",
		},
		{
			template: "DNSErrors",
			expected: runbookURLBase + "/DNSErrors.md",
		},
		{
			template: "DNSNxDomain",
			expected: runbookURLBase + "/DNSNxDomain.md",
		},
		{
			template: "ExternalEgressHighTrend",
			expected: runbookURLBase + "/ExternalEgressHighTrend.md",
		},
		{
			template: "ExternalIngressHighTrend",
			expected: runbookURLBase + "/ExternalIngressHighTrend.md",
		},
		// Operator alert templates
		{
			template: "NetObservNoFlows",
			expected: runbookURLBase + "/NetObservNoFlows.md",
		},
		{
			template: "NetObservLokiError",
			expected: runbookURLBase + "/NetObservLokiError.md",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.template), func(t *testing.T) {
			url := buildRunbookURL(tt.template)
			assert.Equal(t, tt.expected, url)
		})
	}
}

func TestRunbookURLsExist(t *testing.T) {
	// Test all templates to ensure their runbook URLs exist
	templates := []flowslatest.AlertTemplate{
		// Health Rule templates
		"PacketDropsByKernel",
		"PacketDropsByDevice",
		"IPsecErrors",
		"NetpolDenied",
		"LatencyHighTrend",
		"DNSErrors",
		"DNSNxDomain",
		"ExternalEgressHighTrend",
		"ExternalIngressHighTrend",
		// Operator alert templates
		"NetObservNoFlows",
		"NetObservLokiError",
	}

	for _, template := range templates {
		t.Run(string(template), func(t *testing.T) {
			url := buildRunbookURL(template)
			resp, err := http.Get(url)
			assert.NoError(t, err, "Failed to fetch runbook URL: %s", url)
			if err == nil {
				defer resp.Body.Close()
				assert.NotEqual(t, http.StatusNotFound, resp.StatusCode,
					"Runbook URL returns 404: %s", url)
			}
		})
	}
}
