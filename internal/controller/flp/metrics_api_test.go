package flp

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/fmstatus"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
)

func getConfiguredMetrics(cm *corev1.ConfigMap) (api.MetricsItems, error) {
	var cfs config.Root
	err := json.Unmarshal([]byte(cm.Data[configFile]), &cfs)
	if err != nil {
		return nil, err
	}
	for _, stage := range cfs.Parameters {
		if stage.Encode != nil && stage.Encode.Type == "prom" {
			return stage.Encode.Prom.Metrics, nil
		}
	}
	return nil, errors.New("prom encode stage not found")
}

func defaultBuilderWithMetrics(metrics *metricslatest.FlowMetricList) (monolithBuilder, error) {
	cfg := getConfig()
	loki := helper.NewLokiConfig(&cfg.Loki, "any")
	info := reconcilers.Common{Namespace: "namespace", Loki: &loki, ClusterInfo: &cluster.Info{}}
	return newMonolithBuilder(info.NewInstance(image, status.Instance{}), &cfg, metrics, nil, nil)
}

func metric(metrics api.MetricsItems, name string) *api.MetricsItem {
	for i := range metrics {
		if metrics[i].Name == name {
			return &metrics[i]
		}
	}
	return nil
}

func TestFlowMetricToFLP(t *testing.T) {
	assert := assert.New(t)

	fmstatus.Reset()
	b, err := defaultBuilderWithMetrics(&metricslatest.FlowMetricList{
		Items: []metricslatest.FlowMetric{
			{Spec: metricslatest.FlowMetricSpec{
				MetricName: "m_1",
				Type:       metricslatest.CounterMetric,
				ValueField: "val",
				Labels:     []string{"by_field"},
				Filters:    []metricslatest.MetricFilter{{Field: "f", Value: "v", MatchType: metricslatest.MatchEqual}},
			}},
			{Spec: metricslatest.FlowMetricSpec{
				MetricName: "m_2",
				Type:       metricslatest.HistogramMetric,
				Labels:     []string{"by_field"},
				Direction:  metricslatest.Egress,
				Filters: []metricslatest.MetricFilter{
					{Field: "f", Value: "v", MatchType: metricslatest.MatchRegex},
					{Field: "f2", MatchType: metricslatest.MatchAbsence},
				},
				Buckets: []string{"1", "5", "10", "50", "100"},
			}},
		},
	})
	assert.NoError(err)
	_, _, cm, err := b.configMaps()
	assert.NoError(err)
	items, err := getConfiguredMetrics(cm)
	assert.NoError(err)
	names := getSortedMetricsNames(items)
	assert.Equal([]string{
		"m_1",
		"m_2",
		"namespace_flows_total",
		"namespace_ingress_packets_total",
		"node_egress_bytes_total",
		"node_ingress_bytes_total",
		"node_ingress_packets_total",
		"node_to_node_ingress_flows_total",
		"workload_egress_bytes_total",
		"workload_ingress_bytes_total",
	}, names)

	m1 := metric(items, "m_1")
	assert.Equal(api.MetricsItem{
		Name: "m_1",
		Type: "counter",
		Filters: []api.MetricsFilter{
			{Key: "f", Value: "v", Type: api.MetricFilterEqual},
		},
		ValueKey: "val",
		Labels:   []string{"by_field"},
	}, *m1)
	m2 := metric(items, "m_2")
	assert.Equal(api.MetricsItem{
		Name: "m_2",
		Type: "histogram",
		Filters: []api.MetricsFilter{
			{Key: "f", Value: "v", Type: api.MetricFilterRegex},
			{Key: "f2", Type: api.MetricFilterAbsence},
			{Key: "FlowDirection", Value: "0", Type: api.MetricFilterNotEqual},
		},
		Labels:  []string{"by_field"},
		Buckets: []float64{1, 5, 10, 50, 100},
	}, *m2)
}
