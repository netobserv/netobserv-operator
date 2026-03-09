package otel

import (
	_ "embed"
	"encoding/json"
	"sort"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
)

// openTelemetryDefaultTransformRules defined the default Open Telemetry format
// See https://github.com/rhobs/observability-data-model/blob/main/network-observability.md#format-proposal

//go:embed otel-config.json
var rawOtelConfig []byte
var otelConfig map[string]string
var otelRules []api.GenericTransformRule

func GetOtelConfig() (map[string]string, error) {
	if otelConfig == nil {
		cfg := make(map[string]string)
		err := json.Unmarshal(rawOtelConfig, &cfg)
		if err != nil {
			return cfg, err
		}
		otelConfig = cfg
	}
	return otelConfig, nil
}

func GetOtelTransformRules() ([]api.GenericTransformRule, error) {
	if otelRules == nil {
		cfg, err := GetOtelConfig()
		if err != nil {
			return nil, err
		}
		rules := []api.GenericTransformRule{}
		for k, v := range cfg {
			rules = append(rules, api.GenericTransformRule{
				Input:  k,
				Output: v,
			})
		}
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].Input < rules[j].Input
		})
		otelRules = rules
	}

	return otelRules, nil
}

func GetOtelTransformConfig(rules *[]flowslatest.GenericTransformRule) (*api.TransformGeneric, error) {
	otelRules, err := GetOtelTransformRules()
	if err != nil {
		return nil, err
	}
	transformConfig := api.TransformGeneric{
		Policy: "replace_keys",
		Rules:  otelRules,
	}
	// set custom rules if specified
	if rules != nil {
		transformConfig.Rules = []api.GenericTransformRule{}
		for _, r := range *rules {
			transformConfig.Rules = append(transformConfig.Rules, api.GenericTransformRule{
				Input:      r.Input,
				Output:     r.Output,
				Multiplier: r.Multiplier,
			})
		}
	}

	return &transformConfig, err
}

func GetOtelMetrics(flpMetrics []api.MetricsItem) ([]api.MetricsItem, error) {
	otelRules, err := GetOtelTransformRules()
	if err != nil {
		return nil, err
	}

	var otelMetrics = []api.MetricsItem{}

	for i := range flpMetrics {
		m := flpMetrics[i]

		otelMetrics = append(otelMetrics, api.MetricsItem{
			Name:       convertToOtelLabel(otelRules, m.Name),
			Type:       m.Type,
			Help:       m.Help,
			Filters:    convertToOtelFilters(otelRules, m.Filters),
			ValueKey:   convertToOtelLabel(otelRules, m.ValueKey),
			Labels:     convertToOtelLabels(otelRules, m.Labels),
			Buckets:    m.Buckets,
			ValueScale: m.ValueScale,
		})
	}

	return otelMetrics, nil
}

func convertToOtelLabel(otelRules []api.GenericTransformRule, input string) string {
	for _, tr := range otelRules {
		if tr.Input == input {
			return tr.Output
		}
	}

	return input
}

func convertToOtelFilters(otelRules []api.GenericTransformRule, filters []api.MetricsFilter) []api.MetricsFilter {
	var otelFilters = []api.MetricsFilter{}

	for _, f := range filters {
		otelFilters = append(otelFilters, api.MetricsFilter{
			Key:   convertToOtelLabel(otelRules, f.Key),
			Value: f.Value,
			Type:  f.Type,
		})
	}

	return otelFilters
}

func convertToOtelLabels(otelRules []api.GenericTransformRule, labels []string) []string {
	var otelLabels = []string{}

	for _, l := range labels {
		otelLabels = append(otelLabels, convertToOtelLabel(otelRules, l))
	}

	return otelLabels
}
