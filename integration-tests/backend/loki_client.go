package e2etests

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type Resource struct {
	Kind      string
	Name      string
	Namespace string
}

func compareClusterResources(cpu, memory string) bool {
	nodes := getSchedulableLinuxWorkerNodes()
	resourcesMap := getRemainingResourcesNodesMap(nodes)

	var remainingCPU, remainingMemory int64
	for _, node := range nodes {
		remainingCPU += resourcesMap[node.Name].CPU
		remainingMemory += resourcesMap[node.Name].Memory
	}

	requiredCPU, _ := k8sresource.ParseQuantity(cpu)
	requiredMemory, _ := k8sresource.ParseQuantity(memory)
	e2e.Logf("the required cpu is: %d, and the required memory is: %d", requiredCPU.MilliValue(), requiredMemory.MilliValue())
	e2e.Logf("the remaining cpu is: %d, and the remaning memory is: %d", remainingCPU, remainingMemory)
	return remainingCPU > requiredCPU.MilliValue() && remainingMemory > requiredMemory.MilliValue()
}

// ValidateInfraAndResourcesForLoki checks cluster remaning resources and platform type
// supportedPlatforms the platform types which the case can be executed on, if it's empty, then skip this check
func validateInfraAndResourcesForLoki(reqMemory, reqCPU string, supportedPlatforms ...string) bool {
	obj, err := getDynamicResource("infrastructures", "cluster", "")
	o.Expect(err).NotTo(o.HaveOccurred())
	platformType, _ := getNestedField(obj.Object, ".status.platformStatus.type")
	currentPlatform := strings.ToLower(platformType)

	if currentPlatform == "aws" {
		_, getErr := k8sClient.CoreV1().Secrets("kube-system").Get(context.Background(), "aws-creds", metav1.GetOptions{})
		if getErr != nil {
			return false
		}
	}
	if len(supportedPlatforms) > 0 {
		result := contain(supportedPlatforms, currentPlatform) && compareClusterResources(reqCPU, reqMemory)
		return result
	}
	result := compareClusterResources(reqCPU, reqMemory)
	return result
}

type lokiClient struct {
	username        string    // Username for HTTP basic auth.
	password        string    // Password for HTTP basic auth
	address         string    // Server address.
	orgID           string    // adds X-Scope-OrgID to API requests for representing tenant ID. Useful for requesting tenant data when bypassing an auth gateway.
	bearerToken     string    // adds the Authorization header to API requests for authentication purposes.
	bearerTokenFile string    // adds the Authorization header to API requests for authentication purposes.
	retries         int       // How many times to retry each query when getting an error response from Loki.
	queryTags       string    // adds X-Query-Tags header to API requests.
	quiet           bool      // Suppress query metadata.
	startTime       time.Time // Start time for reading logs
	localhost       bool      // whether loki is port-forwarded to localhost, useful for monolithic loki
}

type lokiStream struct {
	App             string `json:"app"`
	DstK8SNamespace string `json:"DstK8S_Namespace"`
	FlowDirection   string `json:"FlowDirection"`
	SrcK8SNamespace string `json:"SrcK8S_Namespace"`
	SrcK8SOwnerName string `json:"SrcK8S_OwnerName"`
	DstK8SOwnerName string `json:"DstK8S_OwnerName"`
	SrcK8SType      string `json:"SrcK8S_Type"`
	DstK8SType      string `json:"DstK8S_Type"`
	SrcK8SZone      string `json:"SrcK8S_Zone"`
	DstK8SZone      string `json:"DstK8S_Zone"`
	K8SClusterName  string `json:"K8S_ClusterName"`
	K8SFlowLayer    string `json:"K8S_FlowLayer"`
	RecordType      string `json:"_RecordType"`
}

type lokiQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Stream lokiStream `json:"stream"`
			Values [][]string `json:"values"`
		} `json:"result"`
		Stats struct {
			Summary struct {
				BytesProcessedPerSecond int     `json:"bytesProcessedPerSecond"`
				LinesProcessedPerSecond int     `json:"linesProcessedPerSecond"`
				TotalBytesProcessed     int     `json:"totalBytesProcessed"`
				TotalLinesProcessed     int     `json:"totalLinesProcessed"`
				ExecTime                float32 `json:"execTime"`
			} `json:"summary"`
			Store struct {
				TotalChunksRef        int `json:"totalChunksRef"`
				TotalChunksDownloaded int `json:"totalChunksDownloaded"`
				ChunksDownloadTime    int `json:"chunksDownloadTime"`
				HeadChunkBytes        int `json:"headChunkBytes"`
				HeadChunkLines        int `json:"headChunkLines"`
				DecompressedBytes     int `json:"decompressedBytes"`
				DecompressedLines     int `json:"decompressedLines"`
				CompressedBytes       int `json:"compressedBytes"`
				TotalDuplicates       int `json:"totalDuplicates"`
			} `json:"store"`
			Ingester struct {
				TotalReached       int `json:"totalReached"`
				TotalChunksMatched int `json:"totalChunksMatched"`
				TotalBatches       int `json:"totalBatches"`
				TotalLinesSent     int `json:"totalLinesSent"`
				HeadChunkBytes     int `json:"headChunkBytes"`
				HeadChunkLines     int `json:"headChunkLines"`
				DecompressedBytes  int `json:"decompressedBytes"`
				DecompressedLines  int `json:"decompressedLines"`
				CompressedBytes    int `json:"compressedBytes"`
				TotalDuplicates    int `json:"totalDuplicates"`
			} `json:"ingester"`
		} `json:"stats"`
	} `json:"data"`
}

// newLokiClient initializes a lokiClient with server address
func newLokiClient(routeAddress string, time time.Time) *lokiClient {
	client := &lokiClient{}
	client.address = routeAddress
	client.retries = 5
	client.quiet = false
	client.startTime = time
	client.localhost = false
	return client
}

// retry sets how many times to retry each query
func (c *lokiClient) retry(retry int) *lokiClient {
	nc := *c
	nc.retries = retry
	return &nc
}

// withToken sets the token used to do query
func (c *lokiClient) withToken(bearerToken string) *lokiClient {
	nc := *c
	nc.bearerToken = bearerToken
	return &nc
}

// buildURL concats a url `http://foo/bar` with a path `/buzz`.
func buildURL(u, p, q string) (string, error) {
	url, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	url.Path = path.Join(url.Path, p)
	url.RawQuery = q
	return url.String(), nil
}

type queryStringBuilder struct {
	values url.Values
}

func newQueryStringBuilder() *queryStringBuilder {
	return &queryStringBuilder{
		values: url.Values{},
	}
}

// encode returns the URL-encoded query string based on key-value
// parameters added to the builder calling Set functions.
func (b *queryStringBuilder) encode() string {
	return b.values.Encode()
}

func (b *queryStringBuilder) setString(name, value string) {
	b.values.Set(name, value)
}

func (b *queryStringBuilder) setInt(name string, value int64) {
	b.setString(name, strconv.FormatInt(value, 10))
}

func (b *queryStringBuilder) setInt32(name string, value int) {
	b.setString(name, strconv.Itoa(value))
}

func (c *lokiClient) getHTTPRequestHeader() (http.Header, error) {
	h := make(http.Header)
	if c.username != "" && c.password != "" {
		h.Set(
			"Authorization",
			"Basic "+base64.StdEncoding.EncodeToString([]byte(c.username+":"+c.password)),
		)
	}
	h.Set("User-Agent", "loki-logcli")

	if c.orgID != "" {
		h.Set("X-Scope-OrgID", c.orgID)
	}

	if c.queryTags != "" {
		h.Set("X-Query-Tags", c.queryTags)
	}

	if (c.username != "" || c.password != "") && (len(c.bearerToken) > 0 || len(c.bearerTokenFile) > 0) {
		return nil, fmt.Errorf("at most one of HTTP basic auth (username/password), bearer-token & bearer-token-file is allowed to be configured")
	}

	if len(c.bearerToken) > 0 && len(c.bearerTokenFile) > 0 {
		return nil, fmt.Errorf("at most one of the options bearer-token & bearer-token-file is allowed to be configured")
	}

	if c.bearerToken != "" {
		h.Set("Authorization", "Bearer "+c.bearerToken)
	}

	if c.bearerTokenFile != "" {
		b, err := os.ReadFile(c.bearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("unable to read authorization credentials file %s: %w", c.bearerTokenFile, err)
		}
		bearerToken := strings.TrimSpace(string(b))
		h.Set("Authorization", "Bearer "+bearerToken)
	}
	return h, nil
}

func (c *lokiClient) doRequest(path, query string, quiet bool, out interface{}) error {
	us, err := buildURL(c.address, path, query)
	if err != nil {
		return err
	}
	if !quiet {
		e2e.Logf("%s", us)
	}

	req, err := http.NewRequest("GET", us, nil)
	if err != nil {
		return err
	}

	h, err := c.getHTTPRequestHeader()
	if err != nil {
		return err
	}
	req.Header = h

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	//  don't use proxy if svc/loki is port-forwarded to localhost
	if !c.localhost {
		tr.Proxy = http.ProxyFromEnvironment
	}

	client := &http.Client{Transport: tr}

	var resp *http.Response
	attempts := c.retries + 1
	success := false

	for attempts > 0 {
		attempts--
		resp, err = client.Do(req)
		if err != nil {
			e2e.Logf("error sending request %v", err)
			continue
		}
		if resp.StatusCode/100 != 2 {
			buf, _ := io.ReadAll(resp.Body) //  nolint
			e2e.Logf("Error response from server: %s (%v) attempts remaining: %d", string(buf), err, attempts)
			// Don't retry on authorization/authentication errors - these are permanent failures
			// Verifies multi-tenancy behavior
			if resp.StatusCode == 401 || resp.StatusCode == 403 {
				if err := resp.Body.Close(); err != nil {
					e2e.Logf("error closing body: %v", err)
				}
				return fmt.Errorf("permission denied: %s", string(buf))
			}
			if err := resp.Body.Close(); err != nil {
				e2e.Logf("error closing body: %v", err)
			}
			continue
		}
		success = true
		break
	}
	if !success {
		return fmt.Errorf("run out of attempts while querying the server")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			e2e.Logf("error closing body: %v", err)
		}
	}()
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *lokiClient) doQuery(path string, query string, quiet bool) (*lokiQueryResponse, error) {
	var err error
	var r lokiQueryResponse

	if err = c.doRequest(path, query, quiet, &r); err != nil {
		return nil, err
	}

	return &r, nil
}

// queryRange uses the /api/v1/query_range endpoint to execute a range query
// logType: application, infrastructure, audit
// queryStr: string to filter logs, for example: "{kubernetes_namespace_name="test"}"
// limit: max log count
// start: Start looking for logs at this absolute time(inclusive), e.g.: time.Now().Add(time.Duration(-1)*time.Hour) means 1 hour ago
// end: Stop looking for logs at this absolute time (exclusive)
// forward: true means scan forwards through logs, false means scan backwards through logs
func (c *lokiClient) queryRange(logType string, queryStr string, limit int, start, end time.Time, forward bool) (*lokiQueryResponse, error) {
	direction := func() string {
		if forward {
			return "FORWARD"
		}
		return "BACKWARD"
	}
	params := newQueryStringBuilder()
	params.setString("query", queryStr)
	params.setInt32("limit", limit)
	params.setInt("start", start.UnixNano())
	params.setInt("end", end.UnixNano())
	params.setString("direction", direction())
	logPath := ""
	if len(logType) > 0 {
		logPath = apiPath + logType + queryRangePath
	} else {
		logPath = queryRangePath
	}

	return c.doQuery(logPath, params.encode(), c.quiet)
}

func (c *lokiClient) searchLogsInLoki(logType, query string) (*lokiQueryResponse, error) {
	e2e.Logf("Loki query is %s", query)
	res, err := c.queryRange(logType, query, 50, c.startTime, time.Now(), false)
	return res, err
}

// Node resource tracking
type NodeInfo struct {
	Name string
}

type NodeResources struct {
	CPU    int64
	Memory int64
}

// getSchedulableLinuxWorkerNodes returns schedulable linux worker nodes
func getSchedulableLinuxWorkerNodes() []NodeInfo {
	nodeList, err := k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{
		LabelSelector: "kubernetes.io/os=linux,node-role.kubernetes.io/worker",
	})
	o.Expect(err).NotTo(o.HaveOccurred())

	var nodes []NodeInfo
	for _, node := range nodeList.Items {
		if node.Spec.Unschedulable {
			continue
		}
		// Check if node is Ready
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				nodes = append(nodes, NodeInfo{Name: node.Name})
				break
			}
		}
	}
	return nodes
}

// getRemainingResourcesNodesMap returns remaining resources per node
func getRemainingResourcesNodesMap(nodes []NodeInfo) map[string]NodeResources {
	rmap := make(map[string]NodeResources)
	requested := getRequestedResourcesNodesMap(nodes)
	allocatable := getAllocatableResourcesNodesMap(nodes)

	for _, node := range nodes {
		rmap[node.Name] = NodeResources{
			CPU:    allocatable[node.Name].CPU - requested[node.Name].CPU,
			Memory: allocatable[node.Name].Memory - requested[node.Name].Memory,
		}
	}
	return rmap
}

// getRequestedResourcesNodesMap returns requested resources per node
func getRequestedResourcesNodesMap(nodes []NodeInfo) map[string]NodeResources {
	rmap := make(map[string]NodeResources)
	podsMap := getPodsNodesMap(nodes)

	for nodeName, pods := range podsMap {
		var totalRequestedCPU, totalRequestedMemory int64
		for _, pod := range pods {
			for _, container := range pod.Containers {
				if container.RequestsCPU != "" {
					cpuQty, _ := k8sresource.ParseQuantity(container.RequestsCPU)
					totalRequestedCPU += cpuQty.MilliValue()
				}
				if container.RequestsMemory != "" {
					memQty, _ := k8sresource.ParseQuantity(container.RequestsMemory)
					totalRequestedMemory += memQty.MilliValue()
				}
			}
		}
		rmap[nodeName] = NodeResources{CPU: totalRequestedCPU, Memory: totalRequestedMemory}
	}
	return rmap
}

// getAllocatableResourcesNodesMap returns allocatable resources per node
func getAllocatableResourcesNodesMap(nodes []NodeInfo) map[string]NodeResources {
	rmap := make(map[string]NodeResources)

	for _, node := range nodes {
		nodeObj, err := k8sClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
		if err != nil {
			continue
		}

		cpuQty := nodeObj.Status.Allocatable[corev1.ResourceCPU]
		memQty := nodeObj.Status.Allocatable[corev1.ResourceMemory]
		rmap[node.Name] = NodeResources{
			CPU:    cpuQty.MilliValue(),
			Memory: memQty.MilliValue(),
		}
	}
	return rmap
}

type PodInfo struct {
	Name       string
	Namespace  string
	NodeName   string
	Phase      string
	Containers []ContainerInfo
}

type ContainerInfo struct {
	RequestsCPU    string
	RequestsMemory string
}

// getPodsNodesMap returns all pods per node
func getPodsNodesMap(nodes []NodeInfo) map[string][]PodInfo {
	podsMap := make(map[string][]PodInfo)
	nodeSet := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		nodeSet[node.Name] = struct{}{}
	}

	// Get all namespaces
	nsList, err := k8sClient.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return podsMap
	}

	// Get pods from each namespace
	for _, nsObj := range nsList.Items {
		ns := nsObj.Name
		podList, err := k8sClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			if _, found := nodeSet[pod.Spec.NodeName]; !found {
				continue
			}
			var containers []ContainerInfo
			for _, c := range pod.Spec.Containers {
				cpuReq := c.Resources.Requests[corev1.ResourceCPU]
				memReq := c.Resources.Requests[corev1.ResourceMemory]
				containers = append(containers, ContainerInfo{
					RequestsCPU:    cpuReq.String(),
					RequestsMemory: memReq.String(),
				})
			}
			podsMap[pod.Spec.NodeName] = append(podsMap[pod.Spec.NodeName], PodInfo{
				Name:       pod.Name,
				Namespace:  ns,
				NodeName:   pod.Spec.NodeName,
				Phase:      string(pod.Status.Phase),
				Containers: containers,
			})
		}
	}

	return podsMap
}
