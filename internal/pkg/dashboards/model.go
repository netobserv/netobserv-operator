package dashboards

import (
	"encoding/json"
	"fmt"
	"strings"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
)

type Dashboard struct {
	Rows  []*Row
	Title string
}

type Row struct {
	Title    string
	Collapse bool
	Height   string
	Metric   string // TODO: remove
	Panels   []Panel
}

func NewRow(title string, collapse bool, height string, panels []Panel) *Row {
	return &Row{
		Title:    title,
		Collapse: collapse,
		Height:   height,
		Panels:   panels,
	}
}

type Panel struct {
	Title   string
	Type    metricslatest.ChartType
	Targets []Target
	Span    int
	Unit    metricslatest.Unit
	Format  string // only used for unmarshalling grafana json in tests
}

func NewPanel(title string, t metricslatest.ChartType, unit metricslatest.Unit, span int, targets ...Target) Panel {
	return Panel{
		Title:   title,
		Type:    t,
		Unit:    unit,
		Span:    span,
		Targets: targets,
	}
}

type Target struct {
	Expr         string
	LegendFormat string
}

func NewTarget(expr, legend string) Target {
	return Target{
		Expr:         expr,
		LegendFormat: legend,
	}
}

var formatCleaner = strings.NewReplacer(
	"\"", "\\\"",
	"\t", "",
	"\n", "",
)

func FromBytes(b []byte) (*Dashboard, error) {
	var d Dashboard
	err := json.Unmarshal(b, &d)
	return &d, err
}

func (d *Dashboard) Titles() []string {
	var titles []string
	for _, r := range d.Rows {
		titles = append(titles, r.Title)
	}
	return titles
}

func (d *Dashboard) FindRow(titleSubstr string) *Row {
	for _, r := range d.Rows {
		if strings.Contains(r.Title, titleSubstr) {
			return r
		}
	}
	return nil
}

func (d *Dashboard) FindPanel(titleSubstr string) *Panel {
	return d.FindNthPanel(titleSubstr, 1)
}

func (d *Dashboard) FindNthPanel(titleSubstr string, n int) *Panel {
	for _, r := range d.Rows {
		for _, p := range r.Panels {
			if strings.Contains(p.Title, titleSubstr) {
				if n <= 1 {
					return &p
				}
				n--
			}
		}
	}
	return nil
}

func (d *Dashboard) ToGrafanaJSON() string {
	// return empty if dashboard doesn't contains rows
	if len(d.Rows) == 0 {
		return ""
	}

	var rows []string
	for _, ri := range d.Rows {
		rows = append(rows, ri.ToGrafanaJSON())
	}

	rowsStr := strings.Join(rows, ",")
	return fmt.Sprintf(`
	{
		"__inputs": [],
		"__requires": [],
		"annotations": {
			"list": []
		},
		"editable": false,
		"gnetId": null,
		"graphTooltip": 0,
		"hideControls": false,
		"id": null,
		"links": [],
		"rows": [%s],
		"refresh": "",
		"schemaVersion": 16,
		"style": "dark",
		"tags": [
			"networking-mixin"
		],
		"templating": {
			"list": []
		},
		"time": {
			"from": "now",
			"to": "now"
		},
		"timepicker": {
			"refresh_intervals": [
				"5s",
				"10s",
				"30s",
				"1m",
				"5m",
				"15m",
				"30m",
				"1h",
				"2h",
				"1d"
			],
			"time_options": [
				"5m",
				"15m",
				"1h",
				"6h",
				"12h",
				"24h",
				"2d",
				"7d",
				"30d"
			]
		},
		"timezone": "browser",
		"title": "%s",
		"version": 0
	}
	`, rowsStr, d.Title)
}

func (r *Row) Titles() []string {
	var titles []string
	for _, p := range r.Panels {
		titles = append(titles, p.Title)
	}
	return titles
}

func (r *Row) ToGrafanaJSON() string {
	var panels []string
	for _, panel := range r.Panels {
		panels = append(panels, panel.ToGrafanaJSON())
	}
	return fmt.Sprintf(`
	{
		"collapse": %t,
		"editable": true,
		"height": "%s",
		"panels": [%s],
		"showTitle": %t,
		"title": "%s"
	}
	`, r.Collapse, r.Height, strings.Join(panels, ","), r.Title != "", r.Title)
}

func (p *Panel) ToGrafanaJSON() string {
	var targets []string
	for _, target := range p.Targets {
		targets = append(targets, target.ToGrafanaJSON())
	}
	unit := string(p.Unit)
	if unit == "" {
		unit = "short"
	}
	var singleStatFormat string
	//nolint:exhaustive
	switch p.Unit {
	case metricslatest.UnitSeconds:
		singleStatFormat = "s"
	case metricslatest.UnitPercent:
		singleStatFormat = "percentunit"
		unit = "short"
	default:
		singleStatFormat = unit
	}
	var t string
	stacked := false
	switch p.Type {
	case metricslatest.ChartTypeSingleStat:
		t = "singlestat"
	case metricslatest.ChartTypeLine:
		t = "graph"
	case metricslatest.ChartTypeStackArea:
		t = "graph"
		stacked = true
	}
	return fmt.Sprintf(`
	{
		"aliasColors": {},
		"bars": false,
		"dashLength": 10,
		"dashes": false,
		"datasource": "prometheus",
		"fill": 1,
		"fillGradient": 0,
		"format": "%s",
		"gridPos": {},
		"id": 1,
		"legend": {
			"alignAsTable": false,
			"avg": false,
			"current": false,
			"max": false,
			"min": false,
			"rightSide": false,
			"show": true,
			"sideWidth": null,
			"total": false,
			"values": false
		},
		"lines": true,
		"linewidth": 1,
		"links": [],
		"nullPointMode": "null",
		"percentage": false,
		"pointradius": 5,
		"points": false,
		"renderer": "flot",
		"repeat": null,
		"seriesOverrides": [],
		"spaceLength": 10,
		"span": %d,
		"stack": %t,
		"steppedLine": false,
		"targets": [%s],
		"thresholds": [],
		"timeFrom": null,
		"timeShift": null,
		"title": "%s",
		"tooltip": {
			"shared": true,
			"sort": 0,
			"value_type": "individual"
		},
		"type": "%s",
		"xaxis": {
			"buckets": null,
			"mode": "time",
			"name": null,
			"show": true,
			"values": []
		},
		"yaxes": [
			{
				"format": "%s",
				"label": null,
				"logBase": 1,
				"max": null,
				"min": null,
				"show": true
			}
		]
	}
	`, singleStatFormat, p.Span, stacked, strings.Join(targets, ","), p.Title, t, unit)
}

func (t *Target) ToGrafanaJSON() string {
	expr := formatCleaner.Replace(t.Expr)
	return fmt.Sprintf(`
	{
		"expr": "%s",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "%s",
		"refId": "A"
	}
	`, expr, t.LegendFormat)
}
