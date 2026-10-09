package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type NWEvents string

const (
	AllowRelated NWEvents = "allow-related"
	Drop         NWEvents = "drop"
)

// returns ture/false if flowcollector API exists.
func isFlowCollectorAPIExists() (bool, error) {
	_, err := getDynamicResource("crd", "flowcollectors.flows.netobserv.io", "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Verify flow records from logs
func verifyFlowRecordFromLogs(podLog string) {
	re := regexp.MustCompile("{\"AgentIP\":.*")
	flowRecords := re.FindAllString(podLog, -3)
	partialFlowRegex := regexp.MustCompile("DstMac\":\"00:00:00:00:00:00")
	for _, flow := range flowRecords {
		// skip assertions and log Partial flows
		if partialFlowRegex.Match([]byte(flow)) {
			e2e.Logf("Found partial flows %s", flow)
		} else {
			o.Expect(flow).Should(o.And(
				o.MatchRegexp("Bytes.:[0-9]+"),
				o.MatchRegexp("TimeFlowEndMs.:[1-9][0-9]+"),
				o.MatchRegexp("TimeFlowStartMs.:[1-9][0-9]+"),
				o.MatchRegexp("TimeReceived.:[1-9][0-9]+")), flow)
		}
	}
}

// Get flow recrods from loki
func getFlowRecords(lokiValues [][]string, stream lokiStream) ([]FlowRecord, error) {
	flowRecords := []FlowRecord{}
	for _, values := range lokiValues {
		timestamp, _ := strconv.ParseInt(values[0], 10, 64)
		var flowlog Flowlog
		err := json.Unmarshal([]byte(values[1]), &flowlog)
		if err != nil {
			return []FlowRecord{}, err
		}
		// FLP strips fields configured as Loki labels from the JSON body;
		// merge them back from the stream labels so assertions can access them.
		if flowlog.SrcK8SType == "" && stream.SrcK8SType != "" {
			flowlog.SrcK8SType = stream.SrcK8SType
		}
		if flowlog.DstK8SType == "" && stream.DstK8SType != "" {
			flowlog.DstK8SType = stream.DstK8SType
		}
		if flowlog.SrcK8SZone == "" && stream.SrcK8SZone != "" {
			flowlog.SrcK8SZone = stream.SrcK8SZone
		}
		if flowlog.DstK8SZone == "" && stream.DstK8SZone != "" {
			flowlog.DstK8SZone = stream.DstK8SZone
		}
		if flowlog.K8SClusterName == "" && stream.K8SClusterName != "" {
			flowlog.K8SClusterName = stream.K8SClusterName
		}
		if flowlog.RecordType == "" && stream.RecordType != "" {
			flowlog.RecordType = stream.RecordType
		}
		flowRecord := FlowRecord{
			Timestamp: timestamp,
			Flowlog:   flowlog,
		}
		flowRecords = append(flowRecords, flowRecord)
	}
	e2e.Logf("Number of flow records found %d", len(flowRecords))
	return flowRecords, nil
}

// Get flow records from IPFIX collector HTTP API
func getIPFIXFlowRecordsFromAPI(namespace, podName string) ([]FlowRecord, error) {
	flowRecords := []FlowRecord{}

	// Query the collector HTTP API using pod exec with specific container
	cmd := "curl -s http://localhost:8080/records?format=json"
	output, err := execCommandInSpecificPod(namespace, podName, cmd, "ipfix-collector")
	if err != nil {
		return flowRecords, fmt.Errorf("failed to query collector API: %w", err)
	}

	// Parse JSON response: {"flowRecords":[{"data":"key: value\n..."}, ...]}
	var response struct {
		FlowRecords []struct {
			Data string `json:"data"`
		} `json:"flowRecords"`
	}

	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return flowRecords, fmt.Errorf("failed to parse collector response: %w", err)
	}

	// Convert each flow record
	for _, record := range response.FlowRecords {
		// Parse the YAML-like data string into a map
		dataFields := parseIPFIXDataString(record.Data)
		flowlog := convertIPFIXToFlowlog(dataFields)
		flowRecord := FlowRecord{
			Timestamp: time.Now().Unix(),
			Flowlog:   flowlog,
		}
		flowRecords = append(flowRecords, flowRecord)
	}

	e2e.Logf("Found %d IPFIX flow records from collector API", len(flowRecords))
	return flowRecords, nil
}

// Parse IPFIX data string format: "    key: value \n    key2: value2 \n ..."
func parseIPFIXDataString(data string) map[string]any {
	fields := make(map[string]any)
	lines := strings.SplitSeq(data, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])

		// Try to parse as number
		if intVal, err := strconv.Atoi(valueStr); err == nil {
			fields[key] = float64(intVal)
		} else if floatVal, err := strconv.ParseFloat(valueStr, 64); err == nil {
			fields[key] = floatVal
		} else {
			fields[key] = valueStr
		}
	}

	return fields
}

// Convert IPFIX data fields to Flowlog struct
func convertIPFIXToFlowlog(dataFields map[string]interface{}) Flowlog {
	flowlog := Flowlog{}

	// Helper to safely get string values
	getString := func(key string) string {
		if val, ok := dataFields[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
		return ""
	}

	getInt := func(key string) int {
		if val, ok := dataFields[key]; ok {
			switch v := val.(type) {
			case float64:
				return int(v)
			case int:
				return v
			case int64:
				return int(v)
			}
		}
		return 0
	}

	// Map IPFIX fields to Flowlog fields
	flowlog.SrcAddr = getString("sourceIPv4Address")
	if flowlog.SrcAddr == "" {
		flowlog.SrcAddr = getString("sourceIPv6Address")
	}
	flowlog.DstAddr = getString("destinationIPv4Address")
	if flowlog.DstAddr == "" {
		flowlog.DstAddr = getString("destinationIPv6Address")
	}
	flowlog.SrcPort = getInt("sourceTransportPort")
	flowlog.DstPort = getInt("destinationTransportPort")
	flowlog.Proto = getInt("protocolIdentifier")
	flowlog.Bytes = getInt("octetDeltaCount")
	if flowlog.Bytes == 0 {
		flowlog.Bytes = getInt("octetTotalCount")
	}
	flowlog.Packets = getInt("packetDeltaCount")
	if flowlog.Packets == 0 {
		flowlog.Packets = getInt("packetTotalCount")
	}
	flowlog.TimeFlowStartMs = getInt("flowStartMilliseconds")
	flowlog.TimeFlowEndMs = getInt("flowEndMilliseconds")
	flowlog.TimeReceived = int(time.Now().Unix())

	// RFC 5477 sampling fields - the key fields for NETOBSERV-2706!
	flowlog.Sampling = getInt("samplingPacketInterval")
	// Also check samplingProbability and convert if present
	if samplingProb, ok := dataFields["samplingProbability"].(float64); ok && samplingProb > 0 {
		flowlog.Sampling = int(1.0 / samplingProb)
	}

	return flowlog
}

// Verify some key and deterministic flow recrods fields and their values
func (flowlog *Flowlog) verifyFlowRecord() {
	flow := fmt.Sprintf("Flow log is: %+v\n", flowlog)
	o.Expect(flowlog.AgentIP).To(o.Equal(flowlog.DstK8SHostIP), flow)
	o.Expect(flowlog.Bytes).Should(o.BeNumerically(">", 0), flow)
	now := time.Now()
	compareTime := now.Add(time.Duration(-2) * time.Hour)
	compareTimeMs := compareTime.UnixMilli()
	o.Expect(flowlog.TimeFlowEndMs).Should(o.BeNumerically(">", compareTimeMs), flow)
	o.Expect(flowlog.TimeFlowStartMs).Should(o.BeNumerically(">", compareTimeMs), flow)
	o.Expect(flowlog.TimeReceived).Should(o.BeNumerically(">", compareTime.Unix()), flow)
}

// Verify IPFIX-specific fields are present and valid
func (flowlog *Flowlog) verifyIPFIXFields() {
	flow := fmt.Sprintf("IPFIX Flow log: %+v\n", flowlog)

	// Basic flow verification (IPFIX flows don't have AgentIP/K8S enrichment)
	o.Expect(flowlog.Bytes).Should(o.BeNumerically(">", 0), flow)
	now := time.Now()
	compareTime := now.Add(time.Duration(-2) * time.Hour)
	compareTimeMs := compareTime.UnixMilli()
	o.Expect(flowlog.TimeFlowEndMs).Should(o.BeNumerically(">", compareTimeMs), flow)
	o.Expect(flowlog.TimeFlowStartMs).Should(o.BeNumerically(">", compareTimeMs), flow)
	o.Expect(flowlog.TimeReceived).Should(o.BeNumerically(">", compareTime.Unix()), flow)

	// Verify IPFIX standard fields are present and valid
	o.Expect(flowlog.SrcAddr).NotTo(o.BeEmpty(), flow)
	o.Expect(flowlog.DstAddr).NotTo(o.BeEmpty(), flow)
	o.Expect(flowlog.SrcPort).Should(o.BeNumerically(">=", 0), flow)
	o.Expect(flowlog.DstPort).Should(o.BeNumerically(">=", 0), flow)
	o.Expect(flowlog.Proto).Should(o.BeNumerically(">", 0), flow)
	o.Expect(flowlog.Packets).Should(o.BeNumerically(">", 0), flow)
	o.Expect(flowlog.Sampling).Should(o.BeNumerically(">=", 0), flow)
}

func (lokilabels Lokilabels) getLokiQueryLabels() string {
	label := reflect.ValueOf(&lokilabels).Elem()
	labelType := label.Type()
	var lokiQuery = "{"
	for i := 0; i < label.NumField(); i++ {
		if label.Field(i).Interface() != "" {
			field := labelType.Field(i)

			// Get the label name from loki tag, or use field name as fallback
			lokiTag := field.Tag.Get("loki")
			if lokiTag == "-" {
				continue
			}
			labelName := field.Name
			if lokiTag != "" {
				labelName = lokiTag
			}

			// Handle FlowDirection special case: only include if value is 0, 1, or 2
			if field.Name == "FlowDirection" {
				if label.Field(i).Interface() != "0" && label.Field(i).Interface() != "1" && label.Field(i).Interface() != "2" {
					continue
				}
			}

			lokiQuery += fmt.Sprintf("%s=\"%s\", ", labelName, label.Field(i).Interface())
		}
	}
	lokiQuery = strings.TrimSuffix(lokiQuery, ", ")
	lokiQuery += "}"

	return lokiQuery
}

func (lokilabels Lokilabels) getLokiJSONfilterQuery(parameters ...string) string {
	lokiQuery := lokilabels.getLokiQueryLabels()
	if len(parameters) != 0 {
		lokiQuery += " | json"
		for _, p := range parameters {
			if strings.Contains(p, "Flags") {
				lokiQuery += fmt.Sprintf(" %s | json", p)
			} else {
				lokiQuery += fmt.Sprintf(" | %s", p)
			}
		}
	}
	e2e.Logf("Loki query is %s", lokiQuery)
	return lokiQuery
}

func (lokilabels Lokilabels) getLokiRegexFilterQuery(parameters ...string) string {
	lokiQuery := lokilabels.getLokiQueryLabels()
	if len(parameters) != 0 {
		for _, p := range parameters {
			lokiQuery += fmt.Sprintf(" |~ `%s`", p)
		}
	}
	e2e.Logf("Loki query is %s", lokiQuery)
	return lokiQuery
}

func (lokilabels Lokilabels) getLokiQuery(filterType string, parameters ...string) string {
	var lokiQuery string
	switch filterType {
	case "JSON":
		lokiQuery = lokilabels.getLokiJSONfilterQuery(parameters...)
	case "REGEX":
		lokiQuery = lokilabels.getLokiRegexFilterQuery(parameters...)
	default:
		panic("loki filter is not supported yet")
	}
	return lokiQuery
}

func (lokilabels Lokilabels) GetMonolithicLokiFlowLogs(lokiRoute string, startTime time.Time, parameters ...string) ([]FlowRecord, error) {
	return lokilabels.getMonolithicLokiFlowLogs(lokiRoute, startTime, "JSON", parameters...)
}

// GetMonolithicLokiFlowLogsRegex queries monolithic Loki using raw-line regex filters.
func (lokilabels Lokilabels) GetMonolithicLokiFlowLogsRegex(lokiRoute string, startTime time.Time, parameters ...string) ([]FlowRecord, error) {
	return lokilabels.getMonolithicLokiFlowLogs(lokiRoute, startTime, "REGEX", parameters...)
}

func (lokilabels Lokilabels) getMonolithicLokiFlowLogs(lokiRoute string, startTime time.Time, filterType string, parameters ...string) ([]FlowRecord, error) {
	// Expose Loki service via an OpenShift Route so queries work from outside the cluster
	namespace, _ := parseMonolithicLokiURL(lokiRoute)
	if namespace != "" {
		lokiRoute = "http://" + createLokiRoute(namespace)
		defer deleteLokiRoute(namespace)
	}

	lc := newLokiClient(lokiRoute, startTime).retry(5)
	lc.quiet = false
	lokiQuery := lokilabels.getLokiQuery(filterType, parameters...)
	flowRecords := []FlowRecord{}
	var res *lokiQueryResponse
	err := wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		var qErr error
		res, qErr = lc.searchLogsInLoki("", lokiQuery)
		if qErr != nil {
			e2e.Logf("\ngot error %v when getting logs for query: %s\n", qErr, lokiQuery)
			return false, nil
		}

		if !lokilabels.AllowEmpty && len(res.Data.Result) == 0 {
			e2e.Logf("waiting for non-empty results for query: %s", lokiQuery)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return flowRecords, err
	}

	for _, result := range res.Data.Result {
		records, recErr := getFlowRecords(result.Values, result.Stream)
		if recErr != nil {
			return []FlowRecord{}, recErr
		}
		flowRecords = append(flowRecords, records...)
	}

	return flowRecords, err
}

// parseMonolithicLokiURL extracts namespace and port from a cluster-internal Loki URL
// e.g. "http://loki.e2e-netobserv-uqmm.svc:3100/" returns ("e2e-netobserv-uqmm", "3100")
func parseMonolithicLokiURL(lokiURL string) (namespace, port string) {
	re := regexp.MustCompile(`://loki\.([^.]+)\.svc:(\d+)`)
	matches := re.FindStringSubmatch(lokiURL)
	if len(matches) == 3 {
		return matches[1], matches[2]
	}
	return "", ""
}

// createLokiRoute creates an OpenShift Route for the demoLoki service if one doesn't already exist.
// It also creates a NetworkPolicy to allow ingress from the OpenShift router.
func createLokiRoute(namespace string) string {
	_, err := routeV1Client.RouteV1().Routes(namespace).Get(context.Background(), "loki", metav1.GetOptions{})
	if err == nil {
		return getRouteAddress(namespace, "loki")
	}

	// Allow ingress from the OpenShift router to reach the Loki pod
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-from-openshift-ingress-to-loki",
			Namespace: namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "loki"},
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"policy-group.network.openshift.io/ingress": "",
						},
					},
				}},
			}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
		},
	}
	_, err = k8sClient.NetworkingV1().NetworkPolicies(namespace).Create(context.Background(), np, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	route := &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: namespace,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: "loki",
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromInt32(3100),
			},
		},
	}
	_, err = routeV1Client.RouteV1().Routes(namespace).Create(context.Background(), route, metav1.CreateOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())
	return getRouteAddress(namespace, "loki")
}

func deleteLokiRoute(namespace string) {
	_ = routeV1Client.RouteV1().Routes(namespace).Delete(context.Background(), "loki", metav1.DeleteOptions{})
	_ = k8sClient.NetworkingV1().NetworkPolicies(namespace).Delete(context.Background(), "allow-from-openshift-ingress-to-loki", metav1.DeleteOptions{})
}

// TODO: add argument for condition to be matched.
// Get flows from Loki logs
func (lokilabels Lokilabels) getLokiFlowLogs(token, lokiRoute string, startTime time.Time, parameters ...string) ([]FlowRecord, error) {
	lc := newLokiClient(lokiRoute, startTime).withToken(token).retry(5)
	tenantID := "network"
	lokiQuery := lokilabels.getLokiQuery("JSON", parameters...)
	flowRecords := []FlowRecord{}
	var res *lokiQueryResponse
	err := wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		var qErr error
		res, qErr = lc.searchLogsInLoki(tenantID, lokiQuery)
		if qErr != nil {
			e2e.Logf("\ngot error %v when getting %s logs for query: %s\n", qErr, tenantID, lokiQuery)
			// Don't retry on permission errors
			if strings.Contains(qErr.Error(), "permission") {
				return false, qErr
			}
			return false, nil
		}

		if !lokilabels.AllowEmpty && len(res.Data.Result) == 0 {
			e2e.Logf("waiting for non-empty results for query: %s", lokiQuery)
			return false, nil
		}

		return true, nil
	})

	if err != nil {
		return flowRecords, err
	}

	for _, result := range res.Data.Result {
		records, recErr := getFlowRecords(result.Values, result.Stream)
		if recErr != nil {
			return []FlowRecord{}, recErr
		}
		flowRecords = append(flowRecords, records...)
	}

	return flowRecords, err
}

// Verify loki flow records and if it was written in the last 5 minutes
// func verifyLokilogsTime(token, lokiRoute string, startTime time.Time) error {
// 	lc := newLokiClient(lokiRoute, startTime).withToken(token).retry(5)
// 	res, err := lc.searchLogsInLoki("network", "{app=\"netobserv-flowcollector\", FlowDirection=\"0\"}")
//
// 	if err != nil {
// 		return err
// 	}
// 	if len(res.Data.Result) == 0 {
// 		return errors.New("network logs not found")
// 	}
// 	flowRecords := []FlowRecord{}
//
// 	for _, result := range res.Data.Result {
// 		flowRecords, err = getFlowRecords(result.Values, result.Stream)
// 		if err != nil {
// 			return err
// 		}
// 	}
//
// 	for _, r := range flowRecords {
// 		r.Flowlog.verifyFlowRecord()
// 	}
// 	return nil
// }

// Verify monolithic loki flow records and if it was written in the last 5 minutes
func verifyMonolithicLokilogsTime(lokiRoute string, startTime time.Time) error {
	// Expose Loki service via an OpenShift Route so queries work from outside the cluster
	namespace, _ := parseMonolithicLokiURL(lokiRoute)
	if namespace != "" {
		lokiRoute = "http://" + createLokiRoute(namespace)
		defer deleteLokiRoute(namespace)
	}

	lc := newLokiClient(lokiRoute, startTime).retry(5)
	lc.quiet = false

	var res *lokiQueryResponse
	flowRecords := []FlowRecord{}

	err := wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		var qErr error
		res, qErr = lc.searchLogsInLoki("", "{app=\"netobserv-flowcollector\", FlowDirection=\"0\"}")
		if qErr != nil {
			e2e.Logf("\ngot error %v when getting logs\n", qErr)
			return false, nil
		}
		if len(res.Data.Result) == 0 {
			e2e.Logf("network logs not found yet, will retry")
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return err
	}

	for _, result := range res.Data.Result {
		records, recErr := getFlowRecords(result.Values, result.Stream)
		if recErr != nil {
			return recErr
		}
		flowRecords = append(flowRecords, records...)
	}

	for _, r := range flowRecords {
		r.Flowlog.verifyFlowRecord()
	}
	return nil
}

// Verify some key and deterministic conversation record fields and their values
func (flowlog *Flowlog) verifyConversationRecord() {
	conversationRecord := fmt.Sprintf("Conversation record in error: %+v\n", flowlog)
	o.Expect(flowlog.Bytes).Should(o.BeNumerically(">", 0), conversationRecord)
	now := time.Now()
	compareTime := now.Add(time.Duration(-2) * time.Hour)
	compareTimeMs := compareTime.UnixMilli()
	o.Expect(flowlog.TimeFlowEndMs).Should(o.BeNumerically(">", compareTimeMs), conversationRecord)
	o.Expect(flowlog.TimeFlowStartMs).Should(o.BeNumerically(">", compareTimeMs), conversationRecord)
	o.Expect(flowlog.HashID).NotTo(o.BeEmpty(), conversationRecord)
	o.Expect(flowlog.NumFlowLogs).Should(o.BeNumerically(">", 0), conversationRecord)
}

// Verify loki conversation records and if it was written in the last 5 minutes
func verifyConversationRecordTime(record []FlowRecord) {
	for _, r := range record {
		r.Flowlog.verifyConversationRecord()
	}
}

// Verify flow correctness based on number of bytes
func verifyFlowCorrectness(objectSize string, flowRecords []FlowRecord) {
	var multiplier int
	switch unit := objectSize[len(objectSize)-1:]; unit {
	case "K":
		multiplier = 1024
	case "M":
		multiplier = 1024 * 1024
	case "G":
		multiplier = 1024 * 1024 * 1024
	default:
		panic("invalid object size unit")
	}
	nObject, _ := strconv.Atoi(objectSize[0 : len(objectSize)-1])
	// minBytes is the size of the object fetched
	minBytes := nObject * multiplier
	// maxBytes is the minBytes +2% tolerance
	maxBytes := int(float64(minBytes) + (float64(minBytes) * 0.02))
	var errFlows float64
	nflows := float64(len(flowRecords))

	for _, r := range flowRecords {
		// occurs very rarely but sometimes >= comparison can be flaky
		// when eBPF-agent evicts packets sooner,
		// currently it configured to be 15seconds.
		if r.Flowlog.Bytes <= minBytes {
			errFlows++
		}
		if r.Flowlog.Bytes >= maxBytes {
			errFlows++
		}
		r.Flowlog.verifyFlowRecord()
	}
	// allow only 10% of flows to have Bytes violating minBytes and maxBytes.
	tolerance := math.Ceil(nflows * 0.10)
	o.Expect(errFlows).Should(o.BeNumerically("<=", tolerance))
}

// Verify Network Events feature flows
func verifyNetworkEvents(flowRecords []FlowRecord, action NWEvents, policytype, direction string) {
	nNWEventsLogs := 0
	for _, flow := range flowRecords {
		nwevent := flow.Flowlog.NetworkEvents
		if len(nwevent) >= 1 {
			e2e.Logf("found nwevent %v", nwevent)
			// usually for our scenario we expect only one nw event
			// but there could be more than 1.
			o.Expect(NWEvents(nwevent[0].Action)).Should(o.Equal(action))
			o.Expect(nwevent[0].Type).Should(o.Equal(policytype))
			o.Expect(nwevent[0].Direction).Should(o.Equal(direction))
			nNWEventsLogs++
		} else {
			e2e.Logf("nwevent missing %v", flow.Flowlog)
		}
		if action == Drop {
			o.Expect(flow.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			o.Expect(flow.Flowlog.PktDropLatestState).Should(o.Equal("TCP_INVALID_STATE"))
			o.Expect(flow.Flowlog.PktDropLatestDropCause).Should(o.ContainSubstring("NetworkEvent_"))
		}
	}
	o.Expect(nNWEventsLogs).Should(o.BeNumerically(">=", 1), "Found no logs with Network Events")
}
