package dashboards

import (
	"testing"

	metricslatest "github.com/netobserv/network-observability-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/network-observability-operator/internal/pkg/metrics"
	"github.com/netobserv/network-observability-operator/internal/pkg/test/util"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateFlowMetricsDashboard_All(t *testing.T) {
	assert := assert.New(t)

	defs := metrics.GetDefinitions(util.SpecForMetrics(), true)
	js := CreateFlowMetricsDashboards(defs)

	d, err := FromBytes([]byte(js["Main"]))
	assert.NoError(err)

	assert.Equal("NetObserv / Main", d.Title)

	assert.Equal([]string{"", "Traffic rates per node", "Traffic rates per namespace", "Traffic rates per workload", "TCP latencies", "Byte and packet drops", "DNS", "Network Policy", "IPsec"}, d.Titles())

	assert.Len(d.Rows[0].Panels, 8)
	assert.Len(d.Rows[1].Panels, 2)

	p := d.FindPanel("Top egress traffic per node")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal("topk(7, sum(rate(netobserv_node_egress_bytes_total{}[2m])) by (SrcK8S_HostName,DstK8S_HostName))", p.Targets[0].Expr)

	p = d.FindPanel("Top P50 DNS latency per node (ms)")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal("topk(7, histogram_quantile(0.5, sum(rate(netobserv_node_dns_latency_seconds_bucket{}[2m])) by (le,SrcK8S_HostName,DstK8S_HostName))*1000 > 0)", p.Targets[0].Expr)

	p = d.FindPanel("Top P99 DNS latency per node (ms)")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal("topk(7, histogram_quantile(0.99, sum(rate(netobserv_node_dns_latency_seconds_bucket{}[2m])) by (le,SrcK8S_HostName,DstK8S_HostName))*1000 > 0)", p.Targets[0].Expr)

	p = d.FindPanel("Top ingress traffic per app namespace")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal(
		`topk(7, (sum(rate(netobserv_namespace_ingress_bytes_total{K8S_FlowLayer="app",SrcK8S_Namespace!=""}[2m])) by (SrcK8S_Namespace,DstK8S_Namespace))`+
			` or (sum(rate(netobserv_namespace_ingress_bytes_total{K8S_FlowLayer="app",DstK8S_Namespace!=""}[2m])) by (SrcK8S_Namespace,DstK8S_Namespace)))`,
		p.Targets[0].Expr,
	)
	p = d.FindPanel("Top ingress traffic per infra namespace")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal(
		`topk(7, (sum(rate(netobserv_namespace_ingress_bytes_total{K8S_FlowLayer="infra",SrcK8S_Namespace!=""}[2m])) by (SrcK8S_Namespace,DstK8S_Namespace))`+
			` or (sum(rate(netobserv_namespace_ingress_bytes_total{K8S_FlowLayer="infra",DstK8S_Namespace!=""}[2m])) by (SrcK8S_Namespace,DstK8S_Namespace)))`,
		p.Targets[0].Expr,
	)

	p = d.FindPanel("Top P50 sRTT per infra namespace (ms)")
	assert.NotNil(p)
	assert.Len(p.Targets, 1)
	assert.Equal(
		`topk(7, (histogram_quantile(0.5, sum(rate(netobserv_namespace_rtt_seconds_bucket{K8S_FlowLayer="infra",SrcK8S_Namespace!=""}[2m])) by (le,SrcK8S_Namespace,DstK8S_Namespace))*1000 > 0)`+
			` or (histogram_quantile(0.5, sum(rate(netobserv_namespace_rtt_seconds_bucket{K8S_FlowLayer="infra",DstK8S_Namespace!=""}[2m])) by (le,SrcK8S_Namespace,DstK8S_Namespace))*1000 > 0))`,
		p.Targets[0].Expr,
	)
}

func TestCreateFlowMetricsDashboard_OnlyNodeIngressBytes(t *testing.T) {
	assert := assert.New(t)

	defs := metrics.GetDefinitions(util.SpecForMetrics("node_ingress_bytes_total"), false)
	js := CreateFlowMetricsDashboards(defs)

	d, err := FromBytes([]byte(js["Main"]))
	assert.NoError(err)

	assert.Equal("NetObserv / Main", d.Title)
	assert.Equal([]string{"", "Traffic rates per node"}, d.Titles())

	topRow := d.FindRow("")
	assert.Equal([]string{"Total ingress traffic"}, topRow.Titles())

	trafficRow := d.FindRow("Traffic rates per node")
	assert.Equal([]string{"Top ingress traffic per node (Bps)"}, trafficRow.Titles())
}

func TestCreateFlowMetricsDashboard_DefaultList(t *testing.T) {
	assert := assert.New(t)

	defs := metrics.GetDefinitions(util.SpecForMetrics(), false)
	js := CreateFlowMetricsDashboards(defs)

	d, err := FromBytes([]byte(js["Main"]))
	assert.NoError(err)

	assert.Equal("NetObserv / Main", d.Title)
	assert.Equal([]string{
		"",
		"Traffic rates per node",
		"Traffic rates per namespace",
		"Traffic rates per workload",
		"TCP latencies",
		"Byte and packet drops",
		"DNS",
		"IPsec",
	}, d.Titles())

	topRow := d.FindRow("")
	assert.Equal([]string{
		"Total egress traffic",
		"Total ingress traffic",
		"TCP latency, p99",
		"Drops",
		"DNS latency, p99",
		"DNS error rate",
		"IPsec encrypted traffic",
	}, topRow.Titles())

	trafficRow := d.FindRow("Traffic rates per node")
	assert.Equal([]string{
		"Top egress traffic per node (Bps)",
		"Top ingress traffic per node (Bps)",
	}, trafficRow.Titles())

	trafficRow = d.FindRow("Traffic rates per namespace")
	assert.Equal([]string{
		"Top egress traffic per infra namespace (Bps)",
		"Top egress traffic per app namespace (Bps)",
		"Top ingress traffic per infra namespace (Bps)",
		"Top ingress traffic per app namespace (Bps)",
	}, trafficRow.Titles())

	trafficRow = d.FindRow("Traffic rates per workload")
	assert.Equal([]string{
		"Top egress traffic per infra workload (Bps)",
		"Top egress traffic per app workload (Bps)",
		"Top ingress traffic per infra workload (Bps)",
		"Top ingress traffic per app workload (Bps)",
	}, trafficRow.Titles())

	rttRow := d.FindRow("TCP latencies")
	assert.Equal([]string{
		"Top P50 sRTT per infra namespace (ms)",
		"Top P50 sRTT per app namespace (ms)",
		"Top P99 sRTT per infra namespace (ms)",
		"Top P99 sRTT per app namespace (ms)",
	}, rttRow.Titles())

	dropsRow := d.FindRow("Byte and packet drops")
	assert.Equal([]string{
		"Top drops per node (pps)",
		"Top drops per infra namespace (pps)",
		"Top drops per app namespace (pps)",
	}, dropsRow.Titles())

	dnsRow := d.FindRow("DNS")
	assert.Equal([]string{
		"Top P50 DNS latency per infra namespace (ms)",
		"Top P50 DNS latency per app namespace (ms)",
		"Top P99 DNS latency per infra namespace (ms)",
		"Top P99 DNS latency per app namespace (ms)",
		"DNS error rate per infra namespace",
		"DNS error rate per app namespace",
	}, dnsRow.Titles())
}

func TestCreateHealthDashboard_Default(t *testing.T) {
	assert := assert.New(t)

	js, err := CreateHealthDashboard("netobserv", "netobserv_namespace_flows_total")
	assert.NoError(err)

	d, err := FromBytes([]byte(js))
	assert.NoError(err)

	assert.Equal("NetObserv / Health", d.Title)
	assert.Equal([]string{"", "Flowlogs-pipeline statistics", "eBPF agent statistics", "Operator statistics", "Resource usage"}, d.Titles())

	// First row
	row := 0
	assert.Len(d.Rows[row].Panels, 4)
	assert.Equal("Flows per second", d.Rows[row].Panels[0].Title)
	assert.Len(d.Rows[row].Panels[0].Targets, 1)
	assert.Contains(d.Rows[row].Panels[0].Targets[0].Expr, "netobserv_ingest_flows_processed")
}

func TestCreateCustomDashboard(t *testing.T) {
	assert := assert.New(t)

	js := CreateFlowMetricsDashboards([]metricslatest.FlowMetric{
		{
			Spec: metricslatest.FlowMetricSpec{
				MetricName: "my_metric",
				Charts: []metricslatest.Chart{
					{
						DashboardName: "Main",
						SectionName:   "My section",
						Title:         "My chart",
						Unit:          metricslatest.UnitBPS,
						Type:          metricslatest.ChartTypeSingleStat,
						Queries: []metricslatest.Query{
							{
								PromQL: `sum(rate($METRIC{label="foo"}[5m]))`,
								Legend: "",
							},
						},
					},
					{
						DashboardName: "Main",
						SectionName:   "My next section",
						Title:         "My next chart",
						Unit:          metricslatest.UnitBPS,
						Type:          metricslatest.ChartTypeLine,
						Queries: []metricslatest.Query{
							{
								PromQL: `sum(rate($METRIC{label="foo"}[5m])) by (lbl1,lbl2)`,
								Legend: "{{lbl1}}: {{lbl2}}",
								Top:    10,
							},
						},
					},
				},
			},
		},
		{
			Spec: metricslatest.FlowMetricSpec{
				MetricName: "my_metric",
				Charts: []metricslatest.Chart{
					{
						DashboardName: "My other dashboard",
						SectionName:   "Other section",
						Title:         "Other chart",
						Unit:          metricslatest.UnitBPS,
						Type:          metricslatest.ChartTypeLine,
						Queries: []metricslatest.Query{
							{
								PromQL: `sum(rate($METRIC{label="foo"}[5m])) by (lbl1,lbl2)`,
								Legend: "{{lbl1}}: {{lbl2}}",
							},
						},
					},
				},
			},
		},
	})

	d, err := FromBytes([]byte(js["Main"]))
	assert.NoError(err)

	assert.Equal("NetObserv / Main", d.Title)
	assert.Equal([]string{"My section", "My next section"}, d.Titles())

	r1 := d.FindRow("My section")
	assert.Equal([]string{"My chart"}, r1.Titles())
	assert.Equal(Panel{
		Title:  "My chart",
		Type:   "singlestat",
		Span:   3,
		Format: "Bps",
		Targets: []Target{
			{
				Expr:         "sum(rate(netobserv_my_metric{label=\"foo\"}[5m]))",
				LegendFormat: "",
			},
		},
	}, r1.Panels[0])

	r2 := d.FindRow("My next section")
	assert.Equal([]string{"My next chart"}, r2.Titles())
	assert.Equal(Panel{
		Title:  "My next chart",
		Type:   "graph",
		Span:   4,
		Format: "Bps",
		Targets: []Target{
			{
				Expr:         "topk(10, sum(rate(netobserv_my_metric{label=\"foo\"}[5m])) by (lbl1,lbl2))",
				LegendFormat: "{{lbl1}}: {{lbl2}}",
			},
		},
	}, r2.Panels[0])

	d, err = FromBytes([]byte(js["My other dashboard"]))
	assert.NoError(err)

	assert.Equal("NetObserv / My other dashboard", d.Title)
	assert.Equal([]string{"Other section"}, d.Titles())

	r1 = d.FindRow("Other section")
	assert.Equal([]string{"Other chart"}, r1.Titles())
	assert.Equal(Panel{
		Title:  "Other chart",
		Type:   "graph",
		Span:   4,
		Format: "Bps",
		Targets: []Target{
			{
				Expr:         "topk(7, sum(rate(netobserv_my_metric{label=\"foo\"}[5m])) by (lbl1,lbl2))",
				LegendFormat: "{{lbl1}}: {{lbl2}}",
			},
		},
	}, r1.Panels[0])
}

func TestSortedCharts(t *testing.T) {
	assert := assert.New(t)

	var quickChart = func(title string) metricslatest.Chart {
		return metricslatest.Chart{
			DashboardName: "Main",
			SectionName:   "S0",
			Title:         title,
			Type:          metricslatest.ChartTypeSingleStat,
			Queries:       []metricslatest.Query{{PromQL: `(query)`, Legend: ""}},
		}
	}

	js := CreateFlowMetricsDashboards([]metricslatest.FlowMetric{
		{
			ObjectMeta: v1.ObjectMeta{Name: "z"},
			Spec: metricslatest.FlowMetricSpec{
				Charts: []metricslatest.Chart{
					quickChart("C0"),
					quickChart("C1"),
				},
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{Name: "a"},
			Spec: metricslatest.FlowMetricSpec{
				Charts: []metricslatest.Chart{
					quickChart("C2"),
				},
			},
		},
	})

	d, err := FromBytes([]byte(js["Main"]))
	assert.NoError(err)

	r := d.FindRow("S0")
	assert.Equal([]string{"C2", "C0", "C1"}, r.Titles())
}
