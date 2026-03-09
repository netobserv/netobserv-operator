package alerts

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
)

func TestSumBy(t *testing.T) {
	// Test GroupByNode with source side - filters are now added via buildLabelFilter before promQLRateFromMetric
	// sumBy should only handle label replacement and aggregation
	pql := sumBy(`rate(my_metric{SrcK8S_HostName!=""}[1m])`, flowslatest.GroupByNode, asSource, "")
	assert.Equal(t,
		`sum(label_replace(rate(my_metric{SrcK8S_HostName!=""}[1m]), "node", "$1", "SrcK8S_HostName", "(.*)")) by (node)`,
		pql,
	)
	assert.Contains(t, pql, `SrcK8S_HostName!=""`, "should preserve filters from input")

	// Test GroupByWorkload with destination side - filters come from buildLabelFilter
	pql = sumBy(`rate(my_metric{DstK8S_Namespace!="",DstK8S_OwnerName!="",DstK8S_OwnerType!=""}[1m])`, flowslatest.GroupByWorkload, asDest, "")
	assert.Equal(t,
		`sum(label_replace(label_replace(label_replace(rate(my_metric{DstK8S_Namespace!="",DstK8S_OwnerName!="",DstK8S_OwnerType!=""}[1m]), "namespace", "$1", "DstK8S_Namespace", "(.*)"), "workload", "$1", "DstK8S_OwnerName", "(.*)"), "kind", "$1", "DstK8S_OwnerType", "(.*)")) by (namespace,workload,kind)`,
		pql,
	)
	assert.Contains(t, pql, `DstK8S_Namespace!=""`, "should preserve DstK8S_Namespace filter from input")
	assert.Contains(t, pql, `DstK8S_OwnerName!=""`, "should preserve DstK8S_OwnerName filter from input")
	assert.Contains(t, pql, `DstK8S_OwnerType!=""`, "should preserve DstK8S_OwnerType filter from input")

	// Test GroupByNamespace - filters from buildLabelFilter
	pql = sumBy(`rate(my_metric{DstK8S_Namespace!=""}[1m])`, flowslatest.GroupByNamespace, asDest, "")
	assert.Equal(t,
		`sum(label_replace(rate(my_metric{DstK8S_Namespace!=""}[1m]), "namespace", "$1", "DstK8S_Namespace", "(.*)")) by (namespace)`,
		pql,
	)
	assert.Contains(t, pql, `DstK8S_Namespace!=""`, "should preserve namespace filter from input")

	// Test no grouping - should NOT add any filters
	pql = sumBy("rate(my_metric[1m])", "", "", "")
	assert.Equal(t, `sum(rate(my_metric[1m]))`, pql)
	assert.NotContains(t, pql, "K8S_", "should not add K8s label filters when not grouping")

	// Test with existing label selector (like DNS errors) - filters are merged in buildLabelFilter
	pql = sumBy(`rate(my_metric{DstK8S_Namespace!="",DnsFlagsResponseCode!="NoError"}[1m])`, flowslatest.GroupByNamespace, asDest, "")
	assert.Equal(t,
		`sum(label_replace(rate(my_metric{DstK8S_Namespace!="",DnsFlagsResponseCode!="NoError"}[1m]), "namespace", "$1", "DstK8S_Namespace", "(.*)")) by (namespace)`,
		pql,
	)
	assert.Contains(t, pql, `DstK8S_Namespace!=""`, "should preserve namespace filter")
	assert.Contains(t, pql, `DnsFlagsResponseCode!="NoError"`, "should preserve business logic filter")
}

func TestPercentagePromQL(t *testing.T) {
	// Test alert mode (isRecording = false)
	pql := percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "10", "", "")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m]))) > 10", pql)

	pql = percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "10", "20", "")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m]))) > 10 < 20", pql)

	pql = percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "10", "20", "2")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m])) > 2) > 10 < 20", pql)

	// Test recording mode (isRecording = true) - no threshold comparisons
	pql = percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "", "", "")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m])))", pql)
	assert.NotContains(t, pql, ">", "recording rules should not have threshold comparisons")

	pql = percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "", "20", "")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m])))", pql)
	assert.NotContains(t, pql, ">", "recording rules should not have threshold comparisons")
	assert.NotContains(t, pql, "<", "recording rules should not have threshold comparisons")

	pql = percentagePromQL("sum(rate(my_metric[1m]))", "sum(rate(my_total[1m]))", "", "20", "2")
	assert.Equal(t, "100 * (sum(rate(my_metric[1m]))) / (sum(rate(my_total[1m])) > 2)", pql)
	assert.Contains(t, pql, "> 2", "recording rules should keep lowVolumeThreshold filter")
	assert.NotContains(t, pql, "> 10", "recording rules should not have threshold comparison")
}
