package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// prometheusQueryResult the response of querying prometheus APIs
type prometheusQueryResult struct {
	Data struct {
		Result     []metric `json:"result"`
		ResultType string   `json:"resultType"`
	} `json:"data"`
	Status string `json:"status"`
}

// metric the prometheus metric
type metric struct {
	Metric struct {
		Name          string `json:"__name__"`
		Cluster       string `json:"cluster,omitempty"`
		Container     string `json:"container,omitempty"`
		ContainerName string `json:"containername,omitempty"`
		Endpoint      string `json:"endpoint,omitempty"`
		Instance      string `json:"instance,omitempty"`
		Job           string `json:"job,omitempty"`
		Namespace     string `json:"namespace,omitempty"`
		Path          string `json:"path,omitempty"`
		Pod           string `json:"pod,omitempty"`
		PodName       string `json:"podname,omitempty"`
		Service       string `json:"service,omitempty"`
	} `json:"metric"`
	Value []interface{} `json:"value"`
}

func getMetric(oc *exutil.CLI, query string) ([]metric, error) {
	bearerToken := getSAToken(oc, "prometheus-k8s", "openshift-monitoring")
	promRoute := "https://" + getRouteAddress(oc, "openshift-monitoring", "prometheus-k8s")
	res, err := queryPrometheus(promRoute, query, bearerToken)
	if err != nil {
		return []metric{}, err
	}
	attempts := 10
	for len(res.Data.Result) == 0 && attempts > 0 {
		time.Sleep(5 * time.Second)
		res, err = queryPrometheus(promRoute, query, bearerToken)
		if err != nil {
			return []metric{}, err
		}
		attempts--
	}
	errMsg := fmt.Sprintf("0 results returned for query %s", query)
	o.Expect(len(res.Data.Result)).Should(o.BeNumerically(">=", 1), errMsg)
	return res.Data.Result, nil
}

// queryPrometheus returns the promtheus metrics which match the query string
// path: the api path, for example: /api/v1/query?
// query: the metric or alert you want to search
// action: it can be "GET", "get", "Get", "POST", "post", "Post"
func queryPrometheus(promRoute string, query string, bearerToken string) (*prometheusQueryResult, error) {
	path := "/api/v1/query"
	action := "GET"

	h := make(http.Header)
	h.Add("Content-Type", "application/json")
	h.Add("Authorization", "Bearer "+bearerToken)

	params := url.Values{}
	if len(query) > 0 {
		params.Add("query", query)
	}

	var p prometheusQueryResult
	resp, err := doHTTPRequest(h, promRoute, path, params.Encode(), action, false, 5, nil, 200)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(resp, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// return the first metric value
func popMetricValue(metrics []metric) float64 {
	valInterface := metrics[0].Value[1]
	val, _ := valInterface.(string)
	value, err := strconv.ParseFloat(val, 64)
	o.Expect(err).NotTo(o.HaveOccurred())
	return value
}

// polls any prometheus metrics
func pollMetrics(oc *exutil.CLI, promQuery string) float64 {
	var metricsVal float64
	e2e.Logf("Query is %s", promQuery)
	err := wait.PollUntilContextTimeout(context.Background(), 60*time.Second, 600*time.Second, false, func(context.Context) (bool, error) {
		metrics, err := getMetric(oc, promQuery)
		if err != nil {
			return false, err
		}
		metricsVal = popMetricValue(metrics)
		if metricsVal <= 0 {
			e2e.Logf("%s did not return metrics value > 0, will try again", promQuery)
		}
		return metricsVal > 0, nil
	})

	msg := fmt.Sprintf("%s did not return valid metrics in 600 seconds", promQuery)
	compat_otp.AssertWaitPollNoErr(err, msg)
	return metricsVal
}

// verify FLP metrics
func verifyFLPMetrics(oc *exutil.CLI) {
	query := "sum(netobserv_ingest_flows_processed)"
	pollMetrics(oc, query)
	query = "sum(netobserv_loki_sent_entries_total)"
	pollMetrics(oc, query)
}

// verify eBPF metrics
func verifyEBPFMetrics(oc *exutil.CLI) {
	query := "sum(netobserv_agent_exported_batch_total)"
	pollMetrics(oc, query)
	query = "sum(netobserv_agent_evictions_total)"
	pollMetrics(oc, query)
}

// verify eBPF filter metrics
func verifyEBPFFilterMetrics(oc *exutil.CLI, reason string) {
	query := fmt.Sprintf(`100 * sum(rate(netobserv_agent_filtered_flows_total{reason="%s"}[1m])) / sum(rate(netobserv_agent_filtered_flows_total[1m]))`, reason)
	metrics := pollMetrics(oc, query)
	switch reason {
	case "FilterAccept":
		// Expected to be around 1
		o.Expect(metrics).Should(o.BeNumerically("~", 0.01, 2), "Accept metrics are beyond threshold values")
	case "FilterNoMatch":
		// Expected to be around 100
		o.Expect(metrics).Should(o.BeNumerically("~", 98, 100), "NoMatch metrics are beyond threshold values")
	case "FilterReject":
		// Expected to be around 2
		o.Expect(metrics).Should(o.BeNumerically("~", 1, 5), "Reject metrics are beyond threshold values")
	}
}

// verify eBPF feature metrics
func verifyEBPFFeatureMetrics(oc *exutil.CLI, feature string) {
	query := fmt.Sprintf(`sum(rate(netobserv_agent_buffer_size{name="%s"}[1m]))`, feature)
	metrics := pollMetrics(oc, query)
	// making sure it's simply greater than 0 because we don't know the deteministic value to expect.
	o.Expect(metrics).Should(o.BeNumerically(">", 0), fmt.Sprintf("%s metrics is 0", feature))
}

func getMetricsScheme(oc *exutil.CLI, servicemonitor string, namespace string) (string, error) {
	out, err := oc.AsAdmin().Run("get").Args("servicemonitor", servicemonitor, "-n", namespace, "-o", "jsonpath='{.spec.endpoints[].scheme}'").Output()
	return out, err
}

func getMetricsServerName(oc *exutil.CLI, servicemonitor string, namespace string) (string, error) {
	out, err := oc.AsAdmin().Run("get").Args("servicemonitor", servicemonitor, "-n", namespace, "-o", "jsonpath='{.spec.endpoints[].tlsConfig.serverName}'").Output()
	return out, err
}
