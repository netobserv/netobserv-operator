package metrics

import (
	"fmt"
	"slices"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
)

const (
	tagNamespaces = "namespaces"
	tagNodes      = "nodes"
	tagWorkloads  = "workloads"
	tagBytes      = "bytes"
	tagPackets    = "packets"
)

var (
	latencyBuckets = []string{".005", ".01", ".02", ".03", ".04", ".05", ".075", ".1", ".25", "1"}
	mapLabels      = map[string][]string{
		tagNodes:      {"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_HostName", "DstK8S_HostName"},
		tagNamespaces: {"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace", "K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel"},
		tagWorkloads: {"K8S_ClusterName", "SrcK8S_Zone", "DstK8S_Zone", "SrcK8S_Namespace", "DstK8S_Namespace",
			"K8S_FlowLayer", "SrcSubnetLabel", "DstSubnetLabel", "SrcK8S_NetworkName", "DstK8S_NetworkName", "SrcASN", "DstASN",
			"SrcK8S_OwnerName", "DstK8S_OwnerName", "SrcK8S_OwnerType", "DstK8S_OwnerType", "SrcK8S_Type", "DstK8S_Type"},
	}
	mapValueFields = map[string]string{
		tagBytes:   "Bytes",
		tagPackets: "Packets",
	}
	predefinedMetrics []metricslatest.FlowMetricSpec
)

func init() {
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		// Bytes / packets metrics
		for _, vt := range []string{tagBytes, tagPackets} {
			valueField := mapValueFields[vt]
			for _, dir := range []metricslatest.FlowDirection{metricslatest.Egress, metricslatest.Ingress} {
				lowDir := strings.ToLower(string(dir))
				predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
					MetricName: fmt.Sprintf("%s_%s_%s_total", groupTrimmed, lowDir, vt),
					Type:       metricslatest.CounterMetric,
					Help:       fmt.Sprintf("Total %s per %s in %s direction", vt, groupTrimmed, lowDir),
					ValueField: valueField,
					Direction:  dir,
					Labels:     labels,
					Charts:     trafficCharts(group, vt, lowDir),
				})
			}
		}
		// Sampling
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_sampling", groupTrimmed),
			Type:       metricslatest.GaugeMetric,
			Help:       fmt.Sprintf("Sampling per %s", groupTrimmed),
			ValueField: "Sampling",
			Labels:     labels,
		})
		// Flows metrics
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_flows_total", groupTrimmed),
			Type:       "counter",
			Help:       fmt.Sprintf("Total flows per %s", groupTrimmed),
			Labels:     labels,
		})
	}
	// RTT metrics
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_rtt_seconds", groupTrimmed),
			Type:       metricslatest.HistogramMetric,
			Help:       fmt.Sprintf("Round-trip time latency in seconds per %s", groupTrimmed),
			ValueField: "TimeFlowRttNs",
			Filters: []metricslatest.MetricFilter{
				{Field: "TimeFlowRttNs", MatchType: metricslatest.MatchPresence},
			},
			Labels:  labels,
			Divider: "1000000000", // ns => s
			Buckets: latencyBuckets,
			Charts:  rttCharts(group),
		})
	}
	// Drops metrics
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		dropLabels := labels
		dropLabels = append(dropLabels, "PktDropLatestState", "PktDropLatestDropCause")
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_drop_packets_total", groupTrimmed),
			Type:       metricslatest.CounterMetric,
			Help:       fmt.Sprintf("Total dropped packets per %s", groupTrimmed),
			ValueField: "PktDropPackets",
			Filters: []metricslatest.MetricFilter{
				{Field: "PktDropPackets", MatchType: metricslatest.MatchPresence},
			},
			Labels: dropLabels,
			Charts: dropCharts(group, "pps"),
		})
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_drop_bytes_total", groupTrimmed),
			Type:       metricslatest.CounterMetric,
			Help:       fmt.Sprintf("Total dropped bytes per %s", groupTrimmed),
			ValueField: "PktDropBytes",
			Filters: []metricslatest.MetricFilter{
				{Field: "PktDropBytes", MatchType: metricslatest.MatchPresence},
			},
			Labels: dropLabels,
			Charts: dropCharts(group, "Bps"),
		})
	}
	// DNS metrics
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_dns_latency_seconds", groupTrimmed),
			Type:       metricslatest.HistogramMetric,
			Help:       fmt.Sprintf("DNS latency in seconds per %s", groupTrimmed),
			ValueField: "DnsLatencyMs",
			Filters: []metricslatest.MetricFilter{
				{Field: "DnsId", MatchType: metricslatest.MatchPresence},
			},
			Labels:  labels,
			Divider: "1000", // ms => s
			Buckets: latencyBuckets,
			Charts:  dnsLatencyCharts(group),
		})
		dnsLabels := labels
		dnsLabels = append(dnsLabels, "DnsFlagsResponseCode")
		dnsLabels = append(dnsLabels, "DnsName")
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_dns_flows_total", groupTrimmed),
			Type:       metricslatest.CounterMetric,
			Help:       fmt.Sprintf("DNS flows per %s", groupTrimmed),
			Filters: []metricslatest.MetricFilter{
				{Field: "DnsId", MatchType: metricslatest.MatchPresence},
			},
			Labels: dnsLabels,
			Charts: dnsCharts(group),
		})
	}

	// Netpol metrics
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		netpolLabels := labels
		netpolLabels = append(netpolLabels, "NetworkEvents>Type", "NetworkEvents>Namespace", "NetworkEvents>Name", "NetworkEvents>Action", "NetworkEvents>Direction")
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_network_policy_events_total", groupTrimmed),
			Type:       "counter",
			Help:       fmt.Sprintf("Total network policy events per %s", groupTrimmed),
			Labels:     netpolLabels,
			Filters:    []metricslatest.MetricFilter{{Field: "NetworkEvents>Feature", Value: "acl"}},
			Flatten:    []string{"NetworkEvents"},
			Remap: map[string]metricslatest.Label{
				"NetworkEvents>Type":      "type",
				"NetworkEvents>Namespace": "namespace",
				"NetworkEvents>Name":      "name",
				"NetworkEvents>Action":    "action",
				"NetworkEvents>Direction": "direction",
			},
			Charts: netpolCharts(group),
		})
	}

	// TLS
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		tlsLabels := labels
		tlsLabels = append(tlsLabels, "TLSVersion", "TLSCipherSuite", "TLSGroup")
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_tls_flows_total", groupTrimmed),
			Type:       metricslatest.CounterMetric,
			Help:       fmt.Sprintf("Total TLS flows per %s", groupTrimmed),
			Filters:    []metricslatest.MetricFilter{{Field: "TLSTypes", MatchType: metricslatest.MatchPresence}},
			Labels:     tlsLabels,
			Charts:     tlsStatusChart(),
		})
	}

	// IPSEC
	for _, group := range []string{tagNodes, tagNamespaces, tagWorkloads} {
		groupTrimmed := strings.TrimSuffix(group, "s")
		labels := mapLabels[group]
		ipsecLabels := labels
		ipsecLabels = append(ipsecLabels, "IPSecStatus")
		predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
			MetricName: fmt.Sprintf("%s_ipsec_flows_total", groupTrimmed),
			Type:       metricslatest.CounterMetric,
			Help:       fmt.Sprintf("Total IPsec encrypted flows per %s", groupTrimmed),
			Filters:    []metricslatest.MetricFilter{{Field: "IPSecStatus", MatchType: metricslatest.MatchPresence}},
			Labels:     ipsecLabels,
			Charts:     ipsecStatusChart(group),
		})
	}

	// Cross-nodes metric
	predefinedMetrics = append(predefinedMetrics, metricslatest.FlowMetricSpec{
		MetricName: "node_to_node_ingress_flows_total",
		Type:       metricslatest.CounterMetric,
		Help:       "Total ingress flows between nodes",
		Labels:     mapLabels[tagNodes],
		Filters: []metricslatest.MetricFilter{
			{Field: "FlowDirection", Value: "2", MatchType: metricslatest.MatchNotEqual},
			{Field: "SrcK8S_HostName", MatchType: metricslatest.MatchPresence},
			{Field: "DstK8S_HostName", MatchType: metricslatest.MatchPresence},
		},
	})
}

func GetAllNames() []string {
	names := []string{}
	for i := range predefinedMetrics {
		names = append(names, predefinedMetrics[i].MetricName)
	}
	return names
}

func getUpdatedDefsFromNames(names []string, labelsToRemove []string, filterRecordType *metricslatest.MetricFilter) []metricslatest.FlowMetric {
	ret := []metricslatest.FlowMetric{}
	for i := range predefinedMetrics {
		if slices.Contains(names, predefinedMetrics[i].MetricName) {
			spec := predefinedMetrics[i]
			spec.Labels = removeLabels(spec.Labels, labelsToRemove)
			if filterRecordType != nil {
				spec.Filters = append(spec.Filters, *filterRecordType)
			}
			// Do not display charts for pps when same metric exists as bps, to avoid overloading the dashboard
			if strings.Contains(predefinedMetrics[i].MetricName, "_packets_") {
				nameWithBytes := strings.Replace(predefinedMetrics[i].MetricName, "_packets_", "_bytes_", 1)
				if slices.Contains(names, nameWithBytes) {
					spec.Charts = nil
				}
				nameWithBytes = strings.Replace(nameWithBytes, "namespace_", "workload_", 1)
				if slices.Contains(names, nameWithBytes) {
					spec.Charts = nil
				}
			}
			ret = append(ret, metricslatest.FlowMetric{Spec: spec})
		}
	}
	return ret
}

func removeLabels(initial []string, toRemove []string) []string {
	var labels []string
	for _, lbl := range initial {
		if !slices.Contains(toRemove, lbl) {
			labels = append(labels, lbl)
		}
	}
	return labels
}

func GetDefinitions(fc *flowslatest.FlowCollectorSpec, allMetrics bool) []metricslatest.FlowMetric {
	var names []string
	if allMetrics {
		names = GetAllNames()
	} else {
		names = fc.GetIncludeList()
	}

	var labelsToRemove []string
	if !fc.Processor.IsZoneEnabled() {
		labelsToRemove = append(labelsToRemove, "SrcK8S_Zone", "DstK8S_Zone")
	}
	if !fc.Processor.IsMultiClusterEnabled() {
		labelsToRemove = append(labelsToRemove, "K8S_ClusterName")
	}
	if !fc.Agent.EBPF.IsUDNMappingEnabled() && len(fc.GetSecondaryIndexes()) == 0 {
		labelsToRemove = append(labelsToRemove, "SrcK8S_NetworkName", "DstK8S_NetworkName")
	}
	if !fc.Processor.IsBgpEnrichmentEnabled() {
		labelsToRemove = append(labelsToRemove, "SrcASN", "DstASN")
	}

	var filterRecordType *metricslatest.MetricFilter
	if fc.Processor.LogTypes != nil {
		switch *fc.Processor.LogTypes {
		case flowslatest.LogTypeFlows, flowslatest.LogTypeEndedConversations:
			// no special filter needed here, since only one kind of record is ever emitted
		case flowslatest.LogTypeConversations:
			// Records can be 'newConnection', 'heartbeat' or 'endConnection'. Only 'endConnection' gives a somewhat accurate count.
			filterRecordType = &metricslatest.MetricFilter{
				Field: "_RecordType",
				Value: "endConnection",
			}
		case flowslatest.LogTypeAll:
			// Records can be 'flowLog', 'newConnection', 'heartbeat' or 'endConnection'. 'flowLog' is the most accurate one.
			filterRecordType = &metricslatest.MetricFilter{
				Field: "_RecordType",
				Value: "flowLog",
			}
		}
	}

	return getUpdatedDefsFromNames(names, labelsToRemove, filterRecordType)
}

func MergePredefined(fm []metricslatest.FlowMetric, fc *flowslatest.FlowCollectorSpec) []metricslatest.FlowMetric {
	predefined := GetDefinitions(fc, false)
	return append(predefined, fm...)
}
