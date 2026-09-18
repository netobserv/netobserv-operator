package e2etests

import (
	"context"
	"fmt"
	"reflect"
	"time"

	o "github.com/onsi/gomega"

	"golang.org/x/mod/semver"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Flowcollector struct to handle Flowcollector resources
type Flowcollector struct {
	Namespace                         string
	ProcessorKind                     string
	MultiClusterDeployment            string
	AddZone                           string
	BgpEnrichment                     string
	LogType                           string
	FLPFilters                        string
	DeploymentModel                   string
	LokiEnable                        string
	LokiMode                          string
	LokiNamespace                     string
	MonolithicLokiURL                 string
	InstallDemoLoki                   string
	KafkaAddress                      string
	KafkaTLSEnable                    string
	KafkaClusterName                  string
	KafkaTopic                        string
	KafkaUser                         string
	KafkaCompression                  string
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
	SrcK8SType      string `json:"SrcK8S_Type,omitempty"`
	SrcK8SName      string `json:"SrcK8S_Name,omitempty"`
	SrcK8SHostName  string `json:"SrcK8S_HostName,omitempty"`
	SrcK8SOwnerType string `json:"SrcK8S_OwnerType,omitempty"`
	SrcAddr         string
	SrcMac          string
	SrcSubnetLabel  string
	// Destination
	DstPort         int
	DstK8SType      string `json:"DstK8S_Type,omitempty"`
	DstK8SName      string `json:"DstK8S_Name,omitempty"`
	DstK8SHostName  string `json:"DstK8S_HostName,omitempty"`
	DstK8SOwnerType string `json:"DstK8S_OwnerType,omitempty"`
	DstAddr         string
	DstMac          string
	DstK8SHostIP    string `json:"DstK8S_HostIP,omitempty"`
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
	// BGP ASN
	SrcASN string `json:"SrcASN,omitempty"`
	DstASN string `json:"DstASN,omitempty"`
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
	TLSGroup       string   `json:"TLSGroup,omitempty"`
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
	App             string `loki:"app"`
	SrcK8SNamespace string `loki:"SrcK8S_Namespace"`
	DstK8SNamespace string `loki:"DstK8S_Namespace"`
	RecordType      string `loki:"_RecordType"`
	FlowDirection   string `loki:"FlowDirection"`
	SrcK8SOwnerName string `loki:"SrcK8S_OwnerName"`
	DstK8SOwnerName string `loki:"DstK8S_OwnerName"`
	K8SClusterName  string `loki:"K8S_ClusterName"`
	K8SFlowLayer    string `loki:"K8S_FlowLayer"`
	SrcK8SType      string `loki:"SrcK8S_Type"`
	DstK8SType      string `loki:"DstK8S_Type"`
	Interfaces      string `loki:"Interfaces"`
	// When set, AllowEmpty is for tests expecting 0 flows (e.g. multi-tenancy, filtering)
	AllowEmpty bool `loki:"-"`
}

// create flowcollector CRD for a given manifest file
func (flow Flowcollector) CreateFlowcollector() {
	parameters := []string{"--ignore-unknown-parameters=true", "-f", flow.Template, "-p"}

	flowCollector := reflect.ValueOf(&flow).Elem()

	for i := 0; i < flowCollector.NumField(); i++ {
		if flowCollector.Field(i).Interface() != "" {
			if flowCollector.Type().Field(i).Name != "Template" {
				parameters = append(parameters, fmt.Sprintf("%s=%s", flowCollector.Type().Field(i).Name, flowCollector.Field(i).Interface()))
			}
		}
	}

	err := applyNsResourceFromTemplateByAdmin(flow.Namespace, parameters...)
	o.Expect(err).NotTo(o.HaveOccurred())

	flow.createRoleBindings()

	flow.WaitForFlowcollectorReady()
}

// createSecretWatcherRB creates a secret-watcher RoleBinding in the given
// namespace so the operator can watch secrets there. Call this when deploying
// external resources (Kafka, LokiStack) so the RoleBinding is ready before
// the FlowCollector CR is created.
func createSecretWatcherRB(namespace string) {
	_, err := k8sClient.RbacV1().RoleBindings(namespace).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "secret-watcher", Namespace: namespace},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "netobserv-secret-watcher"},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      "netobserv-controller-manager",
			Namespace: netobservNS,
		}},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return
	}
	o.Expect(err).NotTo(o.HaveOccurred())
}

func (flow Flowcollector) createRoleBindings() {
	operatorSA := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      "netobserv-controller-manager",
		Namespace: netobservNS,
	}

	componentCRBs := []struct {
		name        string
		clusterRole string
		sa          string
	}{
		{"netobserv-informers-flp-" + flow.Namespace, "netobserv-informers", "flowlogs-pipeline"},
		{"netobserv-informers-flpinformers-" + flow.Namespace, "netobserv-informers", "flowlogs-pipeline-informers"},
		{"netobserv-hostnetwork-flp-" + flow.Namespace, "netobserv-hostnetwork", "flowlogs-pipeline"},
		{"netobserv-loki-writer-flp-" + flow.Namespace, "netobserv-loki-writer", "flowlogs-pipeline"},
		{"netobserv-informers-flptransfo-" + flow.Namespace, "netobserv-informers", "flowlogs-pipeline-transformer"},
		{"netobserv-loki-writer-flptransfo-" + flow.Namespace, "netobserv-loki-writer", "flowlogs-pipeline-transformer"},
		{"netobserv-token-review-plugin-" + flow.Namespace, "netobserv-token-review", "netobserv-plugin"},
	}
	for _, crb := range componentCRBs {
		_, err := k8sClient.RbacV1().ClusterRoleBindings().Create(context.Background(), &rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: crb.name},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: crb.clusterRole},
			Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: crb.sa, Namespace: flow.Namespace}},
		}, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			continue
		}
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	privNS := flow.Namespace + "-privileged"
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 120*time.Second, false, func(ctx context.Context) (bool, error) {
		_, nsErr := k8sClient.CoreV1().Namespaces().Get(ctx, privNS, metav1.GetOptions{})
		if nsErr == nil {
			return true, nil
		}
		if apierrors.IsNotFound(nsErr) {
			return false, nil
		}
		return false, nsErr
	})
	o.Expect(err).NotTo(o.HaveOccurred(), "timed out waiting for namespace %s to be created", privNS)

	secretRBs := []struct {
		name        string
		namespace   string
		clusterRole string
	}{
		{"secret-creator", flow.Namespace, "netobserv-secret-creator"},
		{"secret-creator", privNS, "netobserv-secret-creator"},
	}
	for _, rb := range secretRBs {
		_, err := k8sClient.RbacV1().RoleBindings(rb.namespace).Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: rb.name, Namespace: rb.namespace},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: rb.clusterRole},
			Subjects:   []rbacv1.Subject{operatorSA},
		}, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			continue
		}
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	// flp-informers needs leases access for leader election
	_, err = k8sClient.RbacV1().Roles(flow.Namespace).Create(context.Background(), &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: "flp-informers-leases", Namespace: flow.Namespace},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		}},
	}, metav1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		o.Expect(err).NotTo(o.HaveOccurred())
	}
	_, err = k8sClient.RbacV1().RoleBindings(flow.Namespace).Create(context.Background(), &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "flp-informers-leases", Namespace: flow.Namespace},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: "flp-informers-leases"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "flowlogs-pipeline-informers", Namespace: flow.Namespace}},
	}, metav1.CreateOptions{})
	if !apierrors.IsAlreadyExists(err) {
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

// delete flowcollector CRD from a cluster and wait for privileged namespace to be fully removed
func (flow *Flowcollector) DeleteFlowcollector() error {
	err := deleteDynamicResource("flowcollector", "cluster", "")
	if err != nil {
		return err
	}
	privNS := flow.Namespace + "-privileged"
	return Resource{"namespace", privNS, ""}.WaitUntilResourceIsGone()
}

func (flow *Flowcollector) WaitForFlowcollectorReady() {
	ctx := context.Background()

	// check FLP status
	switch flow.DeploymentModel {
	case "Kafka":
		waitUntilDeploymentReady("flowlogs-pipeline-transformer", flow.Namespace)
	case "Direct":
		waitUntilDaemonSetReady("flowlogs-pipeline", flow.Namespace)
	default:
		waitUntilDeploymentReady("flowlogs-pipeline", flow.Namespace)
	}
	// check informers deployment - only available in version >= 2.0
	csvVersion, err := getCSVVersion(netobservNS)
	if err != nil {
		e2e.Logf("Could not get CSV version, skipping informers check: %v", err)
	} else if semver.Compare(semver.Canonical("v"+csvVersion), "v2.0.0") >= 0 {
		waitUntilDeploymentReady("flowlogs-pipeline-informers", flow.Namespace)
	} else {
		e2e.Logf("Skipping informers check, CSV version %s is below 2.0", csvVersion)
	}

	// check ebpf-agent status
	waitUntilDaemonSetReady("netobserv-ebpf-agent", flow.Namespace+"-privileged")

	// check plugin status - only wait if Loki is enabled and plugin not explicitly disabled
	if flow.PluginEnable != "false" && flow.LokiEnable != "false" {
		waitUntilDeploymentReady("netobserv-plugin", flow.Namespace)
	}

	// Wait for all pods to be ready
	WaitForAllPodsReady(flow.Namespace)
	WaitForAllPodsReady(flow.Namespace + "-privileged")

	// Wait for FlowCollector status to be Ready
	err = wait.PollUntilContextTimeout(ctx, 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		obj, getErr := getDynamicResource("flowcollector", "cluster", "")
		if getErr != nil {
			return false, nil
		}
		condStatus, reason, message := getConditionStatus(obj, "Ready")
		if condStatus == "True" {
			return true, nil
		}
		e2e.Logf("flowcollector Ready condition status=%s: %s,%s", condStatus, reason, message)
		return false, nil
	})
	assertWaitPollNoErr(err, "Flowcollector did not become Ready")
}
