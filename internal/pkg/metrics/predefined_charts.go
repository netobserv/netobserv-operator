package metrics

import (
	"fmt"
	"strings"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
)

const (
	mainDashboard = "Main"
)

func trafficCharts(group, vt, dir string) []metricslatest.Chart {
	sectionName := "Traffic rates per " + strings.TrimSuffix(group, "s")
	var unit metricslatest.Unit
	switch vt {
	case tagBytes:
		unit = metricslatest.UnitBPS
	case tagPackets:
		unit = metricslatest.UnitPPS
	}

	totalSingleStat := metricslatest.Chart{
		Type:          metricslatest.ChartTypeSingleStat,
		SectionName:   "",
		DashboardName: mainDashboard,
		Title:         fmt.Sprintf("Total %s traffic", dir),
		Unit:          unit,
		Queries:       []metricslatest.Query{{PromQL: "sum(rate($METRIC[2m]))"}},
	}
	charts := []metricslatest.Chart{totalSingleStat}

	return append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeStackArea,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         fmt.Sprintf("Top %s traffic", dir),
		Unit:          unit,
		Queries:       []metricslatest.Query{{PromQL: "sum(rate($METRIC{$FILTERS}[2m])) by ($LABELS)", Legend: "$LEGEND"}},
	}, group, string(unit))...)
}

func rttCharts(group string) []metricslatest.Chart {
	sectionName := "TCP latencies"
	charts := []metricslatest.Chart{{
		Type:          metricslatest.ChartTypeSingleStat,
		SectionName:   "",
		DashboardName: mainDashboard,
		Title:         "TCP latency",
		Unit:          metricslatest.UnitSeconds,
		Queries: []metricslatest.Query{
			{
				PromQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket[2m])) by (le)) > 0",
				Legend: "p99",
			},
		},
	}}
	charts = append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeLine,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "Top P50 sRTT",
		Unit:          metricslatest.UnitSeconds,
		Queries: []metricslatest.Query{
			{
				PromQL: "histogram_quantile(0.5, sum(rate($METRIC_bucket{$FILTERS}[2m])) by (le,$LABELS))*1000 > 0",
				Legend: "$LEGEND",
			},
		},
	}, group, "ms")...)
	charts = append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeLine,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "Top P99 sRTT",
		Unit:          metricslatest.UnitSeconds,
		Queries: []metricslatest.Query{
			{
				PromQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket{$FILTERS}[2m])) by (le,$LABELS))*1000 > 0",
				Legend: "$LEGEND",
			},
		},
	}, group, "ms")...)

	return charts
}

func dropCharts(group string, unit metricslatest.Unit) []metricslatest.Chart {
	sectionName := "Byte and packet drops"
	var charts []metricslatest.Chart
	if unit == "pps" {
		charts = append(charts, metricslatest.Chart{
			Type:          metricslatest.ChartTypeSingleStat,
			SectionName:   "",
			DashboardName: mainDashboard,
			Title:         "Drops",
			Unit:          unit,
			Queries:       []metricslatest.Query{{PromQL: "sum(rate($METRIC[2m]))"}},
		})
	}
	return append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeStackArea,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "Top drops",
		Unit:          unit,
		Queries:       []metricslatest.Query{{PromQL: "sum(rate($METRIC{$FILTERS}[2m])) by ($LABELS)", Legend: "$LEGEND"}},
	}, group, string(unit))...)
}

func dnsCharts(group string) []metricslatest.Chart {
	sectionName := "DNS"
	charts := []metricslatest.Chart{
		{
			Type:          metricslatest.ChartTypeSingleStat,
			SectionName:   "",
			DashboardName: mainDashboard,
			Title:         "DNS latency",
			Unit:          metricslatest.UnitSeconds,
			Queries: []metricslatest.Query{
				{
					PromQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket[2m])) by (le)) > 0",
					Legend: "p99",
				},
			},
		},
		{
			Type:          metricslatest.ChartTypeSingleStat,
			SectionName:   "",
			DashboardName: mainDashboard,
			Title:         "DNS error rate",
			Queries:       []metricslatest.Query{{PromQL: `sum(rate($METRIC_count{DnsFlagsResponseCode!="NoError"}[2m]))`}},
		},
	}
	charts = append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeLine,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "Top P50 DNS latency",
		Unit:          metricslatest.UnitSeconds,
		Queries: []metricslatest.Query{
			{
				PromQL: "histogram_quantile(0.5, sum(rate($METRIC_bucket{$FILTERS}[2m])) by (le,$LABELS))*1000 > 0",
				Legend: "$LEGEND",
			},
		},
	}, group, "ms")...)
	charts = append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeLine,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "Top P99 DNS latency",
		Unit:          metricslatest.UnitSeconds,
		Queries: []metricslatest.Query{
			{
				PromQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket{$FILTERS}[2m])) by (le,$LABELS))*1000 > 0",
				Legend: "$LEGEND",
			},
		},
	}, group, "ms")...)

	return append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeStackArea,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "DNS error rate",
		Queries: []metricslatest.Query{{
			PromQL: `sum(rate($METRIC_count{DnsFlagsResponseCode!="NoError",$FILTERS}[2m])) by (DnsFlagsResponseCode,$LABELS)`,
			Legend: "$LEGEND, {{ DnsFlagsResponseCode }}",
		}},
	}, group, "")...)
}

func netpolCharts(group string) []metricslatest.Chart {
	sectionName := "Network Policy"
	charts := []metricslatest.Chart{
		{
			Type:          metricslatest.ChartTypeSingleStat,
			SectionName:   "",
			DashboardName: mainDashboard,
			Title:         "Policy drop rate",
			Queries:       []metricslatest.Query{{PromQL: `sum(rate($METRIC{action="drop"}[2m]))`}},
		},
		{
			Type:          metricslatest.ChartTypeSingleStat,
			SectionName:   "",
			DashboardName: mainDashboard,
			Title:         "Policy allow rate",
			Queries:       []metricslatest.Query{{PromQL: `sum(rate($METRIC{action=~"allow.*"}[2m]))`}},
		},
	}

	charts = append(charts,
		chartVariantsFor(&metricslatest.Chart{
			Type:          metricslatest.ChartTypeStackArea,
			SectionName:   sectionName,
			DashboardName: mainDashboard,
			Title:         "Drop rate",
			Queries: []metricslatest.Query{{
				PromQL: `sum(rate($METRIC{action="drop",$FILTERS}[2m])) by (type,direction,$LABELS)`,
				Legend: "$LEGEND, {{ type }}, {{ direction }}",
			}},
		}, group, "")...)
	return append(charts,
		chartVariantsFor(&metricslatest.Chart{
			Type:          metricslatest.ChartTypeStackArea,
			SectionName:   sectionName,
			DashboardName: mainDashboard,
			Title:         "Allow rate",
			Queries: []metricslatest.Query{{
				PromQL: `sum(rate($METRIC{action=~"allow.*",$FILTERS}[2m])) by (type,direction,$LABELS)`,
				Legend: "$LEGEND, {{ type }}, {{ direction }}",
			}},
		}, group, "")...)
}

func ipsecStatusChart(group string) []metricslatest.Chart {
	sectionName := "IPsec"
	charts := []metricslatest.Chart{{
		Type:          metricslatest.ChartTypeSingleStat,
		SectionName:   "",
		DashboardName: mainDashboard,
		Title:         "IPsec encrypted traffic",
		Unit:          metricslatest.UnitPercent,
		Queries: []metricslatest.Query{
			{
				PromQL: `sum(rate(netobserv_node_ipsec_flows_total{IPSecStatus="success"}[2m])) / sum(rate(netobserv_node_to_node_ingress_flows_total[2m]))`,
			},
		},
	}}
	charts = append(charts, chartVariantsFor(&metricslatest.Chart{
		Type:          metricslatest.ChartTypeLine,
		SectionName:   sectionName,
		DashboardName: mainDashboard,
		Title:         "IPsec flows rate",
		Queries: []metricslatest.Query{{
			PromQL: `sum(rate($METRIC[2m])) by (IPSecStatus)`,
			Legend: "{{ IPSecStatus }}",
		}},
	}, group, "")...)

	return charts
}

func chartVariantsFor(chart *metricslatest.Chart, group, unit string) []metricslatest.Chart {
	var additionalCharts []metricslatest.Chart
	if group == tagWorkloads {
		sectionNameNamespace := strings.Replace(chart.SectionName, "per workload", "per namespace", 1)
		nsInfra := chartVariantFor(chart, tagNamespaces, "infra", unit)
		nsInfra.SectionName = sectionNameNamespace
		nsApp := chartVariantFor(chart, tagNamespaces, "app", unit)
		nsApp.SectionName = sectionNameNamespace
		additionalCharts = []metricslatest.Chart{nsInfra, nsApp}
	}
	switch group {
	case tagNodes:
		return []metricslatest.Chart{
			chartVariantFor(chart, group, "", unit),
		}
	case tagNamespaces, tagWorkloads:
		return append(additionalCharts, []metricslatest.Chart{
			chartVariantFor(chart, group, "infra", unit),
			chartVariantFor(chart, group, "app", unit),
		}...)
	}
	return nil
}

func chartVariantFor(c *metricslatest.Chart, group, layer, unit string) metricslatest.Chart {
	chart := *c
	var flowLayerFilter, labels, legend string
	chart.Title += " per "
	if layer != "" {
		chart.Title += layer + " "
		flowLayerFilter = `K8S_FlowLayer="` + layer + `",`
	}
	var orFilters []string
	switch group {
	case tagNodes:
		chart.Title += "node"
		labels = "SrcK8S_HostName,DstK8S_HostName"
		legend = "source:{{SrcK8S_HostName}}, dest:{{DstK8S_HostName}}"
	case tagNamespaces:
		chart.Title += "namespace"
		labels = "SrcK8S_Namespace,DstK8S_Namespace"
		legend = "source:{{SrcK8S_Namespace}}, dest:{{DstK8S_Namespace}}"
		// orFilters aim to eliminate node-to-node traffic when looking at namespace-based metrics
		orFilters = []string{
			flowLayerFilter + `SrcK8S_Namespace!=""`,
			flowLayerFilter + `DstK8S_Namespace!=""`,
		}
	case tagWorkloads:
		chart.Title += "workload"
		labels = "SrcK8S_Namespace,SrcK8S_OwnerName,DstK8S_Namespace,DstK8S_OwnerName"
		legend = "source:{{SrcK8S_OwnerName}}/{{SrcK8S_Namespace}}, dest:{{DstK8S_OwnerName}}/{{DstK8S_Namespace}}"
		// orFilters aim to eliminate node-to-node traffic when looking at workload-based metrics
		orFilters = []string{
			flowLayerFilter + `SrcK8S_Namespace!=""`,
			flowLayerFilter + `DstK8S_Namespace!=""`,
		}
	}
	if unit != "" {
		chart.Title += " (" + unit + ")"
	}
	queriesReplaceAll(&chart, labels, legend, orFilters)
	return chart
}

func queriesReplaceAll(c *metricslatest.Chart, labels, legend string, orFilters []string) {
	var queries []metricslatest.Query
	for _, q := range c.Queries {
		q.PromQL = strings.ReplaceAll(q.PromQL, "$LABELS", labels)
		q.Legend = strings.ReplaceAll(q.Legend, "$LEGEND", legend)
		if len(orFilters) == 0 {
			q.PromQL = strings.ReplaceAll(q.PromQL, "$FILTERS", "")
		} else {
			var parts []string
			for _, filter := range orFilters {
				parts = append(parts, "("+strings.ReplaceAll(q.PromQL, "$FILTERS", filter)+")")
			}
			q.PromQL = strings.Join(parts, " or ")
		}
		queries = append(queries, q)
	}
	c.Queries = queries
}
