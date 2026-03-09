package dashboards

import (
	"fmt"
	"sort"
	"strings"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
)

type chart struct {
	metricslatest.Chart
	mptr *metricslatest.FlowMetric
}

func createSingleStatPanels(c *chart) []Panel {
	var panels []Panel
	for _, q := range c.Queries {
		title := c.Title
		if q.Legend != "" {
			title += ", " + q.Legend
		}
		query := strings.ReplaceAll(q.PromQL, "$METRIC", "netobserv_"+c.mptr.Spec.MetricName)
		newPanel := NewPanel(title, metricslatest.ChartTypeSingleStat, c.Unit, 3, NewTarget(query, ""))
		panels = append(panels, newPanel)
	}
	return panels
}

func createGraphPanel(c *chart) Panel {
	var targets []Target
	for _, q := range c.Queries {
		top := 7
		if q.Top > 0 {
			top = q.Top
		}
		query := strings.ReplaceAll(q.PromQL, "$METRIC", "netobserv_"+c.mptr.Spec.MetricName)
		query = fmt.Sprintf("topk(%d, %s)", top, query)
		targets = append(targets, NewTarget(query, q.Legend))
	}
	return NewPanel(c.Title, c.Type, c.Unit, 4, targets...)
}

func rearrangeRows(rows []*Row, mapTopPanels, mapBodyPanels map[string][]Panel) {
	for i, row := range rows {
		topPanels := mapTopPanels[row.Title]
		bodyPanels := mapBodyPanels[row.Title]
		// Most of the time, panels are correctly arranged within a section.
		// Excepted when there are 4 panels (or 3*rows+1), it shows 3 on first row then 1 on the second row
		// We'll change that to 2 + 2
		count := len(bodyPanels)
		if count > 3 && count%3 == 1 {
			// Set Span=6 (half page) for the two last panels
			bodyPanels[count-1].Span = 6
			bodyPanels[count-2].Span = 6
		}
		rows[i].Panels = topPanels
		rows[i].Panels = append(rows[i].Panels, bodyPanels...)
		if rows[i].Title == "" && len(rows[i].Panels) > 8 {
			// When top row has many panels, create a collapsable section
			rows[i].Title = "Overview"
		}
	}
}

func createFlowMetricsDashboard(dashboardName string, charts []chart) string {
	mapRows := make(map[string]*Row)
	mapTopPanels := make(map[string][]Panel)
	mapBodyPanels := make(map[string][]Panel)
	var orderedRows []*Row
	chartsDedupMap := make(map[string]any)
	for i := range charts {
		chart := charts[i]
		// A chart might be provided by several metrics, e.g. Total ingress bps can be provided by node_ingress_bytes_total and namespace_ingress_bytes_total
		// Dedup them, assuming they have the same title+unit
		dedupKey := chart.Title + "/" + string(chart.Unit)
		if _, exists := chartsDedupMap[dedupKey]; exists {
			continue
		}
		chartsDedupMap[dedupKey] = true

		if chart.Type == metricslatest.ChartTypeSingleStat {
			mapTopPanels[chart.SectionName] = append(mapTopPanels[chart.SectionName], createSingleStatPanels(&chart)...)
		} else {
			mapBodyPanels[chart.SectionName] = append(mapBodyPanels[chart.SectionName], createGraphPanel(&chart))
		}

		if _, exists := mapRows[chart.SectionName]; !exists {
			row := NewRow(chart.SectionName, false, "250px", nil)
			mapRows[chart.SectionName] = row
			orderedRows = append(orderedRows, row)
		}
	}

	rearrangeRows(orderedRows, mapTopPanels, mapBodyPanels)
	d := Dashboard{Rows: orderedRows, Title: "NetObserv / " + dashboardName}
	return d.ToGrafanaJSON()
}

func CreateFlowMetricsDashboards(metrics []metricslatest.FlowMetric) map[string]string {
	dashboardsJSON := make(map[string]string)
	chartsPerDashboard := make(map[string][]chart)
	// Sort alphabetically to enforce consistent ordering
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].Name < metrics[j].Name })
	for i := range metrics {
		metric := &metrics[i]
		for j := range metric.Spec.Charts {
			c := chart{
				Chart: metric.Spec.Charts[j],
				mptr:  metric,
			}
			chartsPerDashboard[c.DashboardName] = append(chartsPerDashboard[c.DashboardName], c)
		}
	}
	for name, charts := range chartsPerDashboard {
		dashboardsJSON[name] = createFlowMetricsDashboard(name, charts)
	}
	return dashboardsJSON
}
