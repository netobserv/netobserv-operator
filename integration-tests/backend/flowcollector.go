package e2etests

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Flowcollector struct to handle Flowcollector resources
type Flowcollector struct {
	Namespace                         string
	ProcessorKind                     string
	MultiClusterDeployment            string
	AddZone                           string
	LogType                           string
	FLPFilters                        string
	DeploymentModel                   string
	LokiEnable                        string
	LokiMode                          string
	LokiURL                           string
	LokiTLSCertName                   string
	LokiStatusTLSEnable               string
	LokiStatusURL                     string
	LokiStatusTLSCertName             string
	LokiStatusTLSUserCertName         string
	LokiNamespace                     string
	MonolithicLokiURL                 string
	KafkaAddress                      string
	KafkaTLSEnable                    string
	KafkaClusterName                  string
	KafkaTopic                        string
	KafkaUser                         string
	KafkaNamespace                    string
	FLPMetricServerTLSType            string
	EBPFMetricServerTLSType           string
	EBPFCacheActiveTimeout            string
	EBPFPrivileged                    string
	EBPFFilterEnable                  string
	EBPFFilterRules                   string
	Sampling                          string
	EBPFMetrics                       string
	EBPFeatures                       []string
	CacheMaxFlows                     string
	PluginEnable                      string
	NetworkPolicyEnable               string
	NetworkPolicyAdditionalNamespaces []string
	Exporters                         []string
	SecondaryNetworks                 string
	CollectionMode                    string
	SlicesEnable                      string
	NamespacesAllow                   []string
	ServiceTLSType                    string
	ServiceCASecretName               string
	ServiceServerCertSecretName       string
	ServiceClientCertSecretName       string
	Template                          string
}

type Flowlog struct {
	// Source
	SrcPort         int
	SrcK8SType      string
	SrcK8SName      string
	SrcK8SHostName  string
	SrcK8SOwnerType string
	SrcAddr         string
	SrcMac          string
	SrcSubnetLabel  string
	// Destination
	DstPort         int
	DstK8SType      string
	DstK8SName      string
	DstK8SHostName  string
	DstK8SOwnerType string
	DstAddr         string
	DstMac          string
	DstK8SHostIP    string
	DstSubnetLabel  string
	// Protocol
	Proto    int
	IcmpCode int
	IcmpType int
	Dscp     int
	Flags    []string
	// Time
	TimeReceived    int
	TimeFlowEndMs   int
	TimeFlowStartMs int
	// Interface
	IfDirection  int
	IfDirections []int
	Interfaces   []string
	Etype        int
	// Others
	Packets        int
	Bytes          int
	Duplicate      bool
	AgentIP        string
	Sampling       int
	HashID         string `json:"_HashId,omitempty"`
	IsFirst        bool   `json:"_IsFirst,omitempty"`
	RecordType     string `json:"_RecordType,omitempty"`
	NumFlowLogs    int    `json:"numFlowLogs,omitempty"`
	K8SClusterName string `json:"K8S_ClusterName,omitempty"`
	// Zone
	SrcK8SZone string `json:"SrcK8S_Zone,omitempty"`
	DstK8SZone string `json:"DstK8S_Zone,omitempty"`
	// DNS
	DNSLatencyMs         int    `json:"DnsLatencyMs,omitempty"`
	DNSFlagsResponseCode string `json:"DnsFlagsResponseCode,omitempty"`
	// Packet Drop
	PktDropBytes           int    `json:"PktDropBytes,omitempty"`
	PktDropPackets         int    `json:"PktDropPackets,omitempty"`
	PktDropLatestState     string `json:"PktDropLatestState,omitempty"`
	PktDropLatestDropCause string `json:"PktDropLatestDropCause,omitempty"`
	// RTT
	TimeFlowRttNs int `json:"TimeFlowRttNs,omitempty"`
	// Packet Translation
	XlatDstAddr         string `json:"XlatDstAddr,omitempty"`
	XlatDstK8SName      string `json:"XlatDstK8S_Name,omitempty"`
	XlatDstK8SNamespace string `json:"XlatDstK8S_Namespace,omitempty"`
	XlatDstK8SType      string `json:"XlatDstK8S_Type,omitempty"`
	XlatDstPort         int    `json:"XlatDstPort,omitempty"`
	XlatSrcAddr         string `json:"XlatSrcAddr,omitempty"`
	XlatSrcK8SName      string `json:"XlatSrcK8S_Name,omitempty"`
	XlatSrcK8SNamespace string `json:"XlatSrcK8S_Namespace,omitempty"`
	ZoneID              int    `json:"ZoneId,omitempty"`
	// Network Events
	NetworkEvents []NetworkEvent `json:"NetworkEvents,omitempty"`
	// Secondary Network
	SrcK8SNetworkName string `json:"SrcK8S_NetworkName,omitempty"`
	DstK8SNetworkName string `json:"DstK8S_NetworkName,omitempty"`
	// UDN
	Udns []string `json:"Udns,omitempty"`
	// IPSec
	IPSecStatus string `json:"IPSecStatus,omitempty"`
	// TLS
	TLSVersion     string   `json:"TLSVersion,omitempty"`
	TLSTypes       []string `json:"TLSTypes,omitempty"`
	TLSCurve       string   `json:"TLSCurve,omitempty"`
	TLSCipherSuite string   `json:"TLSCipherSuite,omitempty"`
}

type NetworkEvent struct {
	Action    string `json:"Action,omitempty"`
	Type      string `json:"Type,omitempty"`
	Name      string `json:"Name,omitempty"`
	Namespace string `json:"Namespace,omitempty"`
	Direction string `json:"Direction,omitempty"`
	Feature   string `json:"Feature,omitempty"`
}

type FlowRecord struct {
	Timestamp int64
	Flowlog   Flowlog
}

type Lokilabels struct {
	App             string
	SrcK8SNamespace string
	DstK8SNamespace string
	RecordType      string
	FlowDirection   string
	SrcK8SOwnerName string
	DstK8SOwnerName string
	K8SClusterName  string
	SrcK8SType      string
	DstK8SType      string
	Interfaces      string
}

// create flowcollector CRD for a given manifest file
func (flow Flowcollector) CreateFlowcollector(oc *exutil.CLI) {
	parameters := []string{"--ignore-unknown-parameters=true", "-f", flow.Template, "-p"}

	flowCollector := reflect.ValueOf(&flow).Elem()

	for i := 0; i < flowCollector.NumField(); i++ {
		if flowCollector.Field(i).Interface() != "" {
			if flowCollector.Type().Field(i).Name != "Template" {
				parameters = append(parameters, fmt.Sprintf("%s=%s", flowCollector.Type().Field(i).Name, flowCollector.Field(i).Interface()))
			}
		}
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, flow.Namespace, parameters...)
	flow.WaitForFlowcollectorReady(oc)
}

// delete flowcollector CRD from a cluster
func (flow *Flowcollector) DeleteFlowcollector(oc *exutil.CLI) error {
	return oc.AsAdmin().WithoutNamespace().Run("delete").Args("flowcollector", "cluster").Execute()
}

func (flow *Flowcollector) WaitForFlowcollectorReady(oc *exutil.CLI) {
	// check FLP status
	switch flow.DeploymentModel {
	case "Kafka":
		waitUntilDeploymentReady(oc, "flowlogs-pipeline-transformer", flow.Namespace)
	case "Service":
		waitUntilDeploymentReady(oc, "flowlogs-pipeline", flow.Namespace)
	default:
		waitUntilDaemonSetReady(oc, "flowlogs-pipeline", flow.Namespace)
	}
	// check ebpf-agent status
	waitUntilDaemonSetReady(oc, "netobserv-ebpf-agent", flow.Namespace+"-privileged")

	// check plugin status - only wait if Loki is enabled and plugin not explicitly disabled
	if flow.PluginEnable != "false" && flow.LokiEnable != "false" {
		waitUntilDeploymentReady(oc, "netobserv-plugin", flow.Namespace)
	}

	compat_otp.AssertAllPodsToBeReady(oc, flow.Namespace)
	compat_otp.AssertAllPodsToBeReady(oc, flow.Namespace+"-privileged")
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {

		status, err := oc.AsAdmin().Run("get").Args("flowcollector", "-o", "jsonpath='{.items[*].status.conditions[0].reason}'").Output()

		if err != nil {
			return false, err
		}
		if strings.Contains(status, "Ready") {
			return true, nil
		}

		msg, err := oc.AsAdmin().Run("get").Args("flowcollector", "-o", "jsonpath='{.items[*].status.conditions[0].message}'").Output()
		e2e.Logf("flowcollector status is %s due to %s", status, msg)
		if err != nil {
			return false, err
		}

		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, "Flowcollector did not become Ready")
}
