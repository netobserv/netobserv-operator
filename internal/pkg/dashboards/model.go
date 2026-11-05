package dashboards

import "encoding/json"

// This package adapts the FlowMetrics API to the reusable dashboard model.
// Keeping the adapter internal prevents the public model from depending on
// operator API packages (and therefore avoids an import cycle with capture).

import publicdash "github.com/netobserv/netobserv-operator/pkg/dashboards"
import metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"

type Dashboard = publicdash.Dashboard
type Row = publicdash.Row
type Panel = publicdash.Panel
type Target = publicdash.Target

func NewRow(title string, collapse bool, height string, panels []Panel) *Row {
	return publicdash.NewRow(title, collapse, height, panels)
}

func NewPanel(title string, chartType metricslatest.ChartType, unit metricslatest.Unit, span int, targets ...Target) Panel {
	return publicdash.NewPanel(title, publicdash.ChartType(chartType), publicdash.Unit(unit), span, targets...)
}

var NewTarget = publicdash.NewTarget

func FromBytes(data []byte) (*Dashboard, error) {
	var dashboard Dashboard
	if err := json.Unmarshal(data, &dashboard); err != nil {
		return nil, err
	}
	return &dashboard, nil
}
