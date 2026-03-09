package alerts

import (
	"fmt"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
)

type promQLRate string

func promQLRateFromMetric(metric, suffix, filters, interval, offset string) promQLRate {
	return promQLRate(fmt.Sprintf("rate(netobserv_%s%s%s[%s]%s)", metric, suffix, filters, interval, offset))
}

func sumBy(promQL promQLRate, groupBy flowslatest.HealthRuleGroupBy, side srcOrDst, extraLabel string) string {
	var nooLabels []string
	var labelsOut []string
	switch groupBy {
	case flowslatest.GroupByNode:
		nooLabels = []string{string(side) + "K8S_HostName"}
		labelsOut = []string{"node"}
	case flowslatest.GroupByNamespace:
		nooLabels = []string{string(side) + "K8S_Namespace"}
		labelsOut = []string{"namespace"}
	case flowslatest.GroupByWorkload:
		nooLabels = []string{string(side) + "K8S_Namespace", string(side) + "K8S_OwnerName", string(side) + "K8S_OwnerType"}
		labelsOut = []string{"namespace", "workload", "kind"}
	}
	if len(labelsOut) > 0 {
		// promQL input is like "rate(netobserv_workload_ingress_bytes_total[1m])"
		// we need to relabel src / dst labels to the same label name in order to allow adding them
		// e.g. of desired output:
		// sum(label_replace(rate(netobserv_workload_ingress_bytes_total[1m]), "namespace", "$1", "SrcK8S_Namespace", "(.*)")) by (namespace)
		replacedLabels := string(promQL)
		for i := range labelsOut {
			in := nooLabels[i]
			out := labelsOut[i]
			replacedLabels = fmt.Sprintf(`label_replace(%s, "%s", "$1", "%s", "(.*)")`, replacedLabels, out, in)
		}
		joinedLabels := strings.Join(labelsOut, ",")
		if extraLabel != "" {
			joinedLabels += "," + extraLabel
		}
		return fmt.Sprintf("sum(%s) by (%s)", replacedLabels, joinedLabels)
	} else if extraLabel != "" {
		return fmt.Sprintf("sum(%s) by (%s)", promQL, extraLabel)
	}
	return fmt.Sprintf("sum(%s)", promQL)
}

func histogramQuantile(promQL promQLRate, groupBy flowslatest.HealthRuleGroupBy, side srcOrDst, quantile string) string {
	sumQL := sumBy(promQL, groupBy, side, "le")
	return fmt.Sprintf("histogram_quantile(%s, %s)", quantile, sumQL)
}

func percentagePromQL(promQLMetricSum, promQLTotalSum string, threshold, upperThreshold, lowVolumeThreshold string) string {
	var lowVolumeThresholdPart string
	if lowVolumeThreshold != "" {
		lowVolumeThresholdPart = " > " + lowVolumeThreshold
	}

	// For recording rules, return only the calculation without comparison
	if threshold == "" {
		return fmt.Sprintf(
			"100 * (%s) / (%s%s)",
			promQLMetricSum,
			promQLTotalSum,
			lowVolumeThresholdPart,
		)
	}

	var upperThresholdPart string
	if upperThreshold != "" {
		upperThresholdPart = " < " + upperThreshold
	}

	// For alert rules, include the threshold comparison
	return fmt.Sprintf(
		"100 * (%s) / (%s%s) > %s%s",
		promQLMetricSum,
		promQLTotalSum,
		lowVolumeThresholdPart,
		threshold,
		upperThresholdPart,
	)
}

func baselineIncreasePromQL(promQLMetric, promQLBaseline string, threshold, upperThreshold string) string {
	// For recording rules, return only the calculation without comparison
	if threshold == "" {
		return fmt.Sprintf(
			"100 * ((%s) - (%s)) / (%s)",
			promQLMetric,
			promQLBaseline,
			promQLBaseline,
		)
	}

	var upperThresholdPart string
	if upperThreshold != "" {
		upperThresholdPart = " < " + upperThreshold
	}

	// For alert rules, include the threshold comparison
	return fmt.Sprintf(
		"100 * ((%s) - (%s)) / (%s) > %s%s",
		promQLMetric,
		promQLBaseline,
		promQLBaseline,
		threshold,
		upperThresholdPart,
	)
}
