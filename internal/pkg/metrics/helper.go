package metrics

import (
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
)

func GetFilters(fm *metricslatest.FlowMetricSpec) []metricslatest.MetricFilter {
	var filters []metricslatest.MetricFilter
	if fm.Direction == metricslatest.Egress {
		filters = append(filters, metricslatest.MetricFilter{
			Field:     "FlowDirection",
			MatchType: metricslatest.MatchNotEqual,
			Value:     "0", // 1 or 2
		})
	} else if fm.Direction == metricslatest.Ingress {
		filters = append(filters, metricslatest.MetricFilter{
			Field:     "FlowDirection",
			MatchType: metricslatest.MatchNotEqual,
			Value:     "1", // 0 or 2
		})
	}
	return append(fm.Filters, filters...)
}
