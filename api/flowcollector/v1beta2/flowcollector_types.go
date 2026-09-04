package v1beta2

import (
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type FlowCollectorDeploymentModel string

const (
	DeploymentModelDirect  FlowCollectorDeploymentModel = "Direct"
	DeploymentModelKafka   FlowCollectorDeploymentModel = "Kafka"
	DeploymentModelService FlowCollectorDeploymentModel = "Service"
)

// Please notice that the FlowCollectorSpec's properties MUST redefine one of the default
// values to force the definition of the section when it is not provided by the manifest.
// This will cause that the remaining default fields will be set according to their definition.
// Otherwise, omitting the sections in the manifest would lead to zero-valued properties.
// This is a workaround for the related issue:
// https://github.com/kubernetes-sigs/controller-tools/issues/622

// Defines the desired state of the FlowCollector resource.
type FlowCollectorSpec struct {
	// Namespace where NetObserv pods are deployed.
	// It is recommended to keep the default setting, to avoid having to make manual adjustments to rights management.
	// The deployed pods require specific cluster role bindings in order to operate.
	// Those bindings are preinstalled for service accounts located in the default namespace ("netobserv").
	// If you configured a different namespace, you must update (or recreate) the cluster role bindings accordingly.
	// More information: https://github.com/netobserv/netobserv-operator/blob/main/README.md#customized-namespace
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Namespace is immutable. If you need to change it, delete and recreate the resource."
	Namespace string `json:"namespace,omitempty"`

	// Agent configuration for flows extraction.
	Agent FlowCollectorAgent `json:"agent,omitempty"`

	// `processor` defines the settings of the component that receives the flows from the agent,
	// enriches them, generates metrics, and forwards them to the Loki persistence layer and/or any available exporter.
	Processor FlowCollectorFLP `json:"processor,omitempty"`

	// `loki`, the flow store, client settings.
	Loki FlowCollectorLoki `json:"loki,omitempty"`

	// `prometheus` defines Prometheus settings, such as querier configuration used to fetch metrics from the Console plugin.
	Prometheus FlowCollectorPrometheus `json:"prometheus,omitempty"`

	// `consolePlugin` defines the settings related to the Web Console.
	ConsolePlugin FlowCollectorConsolePlugin `json:"consolePlugin,omitempty"`

	// `deploymentModel` defines the desired type of deployment for flow processing. Possible values are:<br>
	// - `Service` (default) to make the flow processor listen as a Kubernetes Service, backed by a scalable Deployment.<br>
	// - `Kafka` to make flows sent to a Kafka pipeline before consumption by the processor.<br>
	// - `Direct` to make the flow processor listen directly from the agents using the host network, backed by a DaemonSet. Only recommended on small clusters, below 15 nodes.<br>
	// Kafka can provide better scalability, resiliency, and high availability (for more details, see https://www.redhat.com/en/topics/integration/what-is-apache-kafka).<br>
	// `Direct` is not recommended on large clusters as it is less memory efficient.
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="Service";"Direct";"Kafka"
	// +kubebuilder:default:=Service
	DeploymentModel FlowCollectorDeploymentModel `json:"deploymentModel,omitempty"`

	// Kafka configuration, allowing to use Kafka as a broker as part of the flow collection pipeline. Available when the `spec.deploymentModel` is `Kafka`.
	// +optional
	Kafka FlowCollectorKafka `json:"kafka,omitempty"`

	// `exporters` defines additional optional exporters for custom consumption or storage.
	// +optional
	// +k8s:conversion-gen=false
	Exporters []*FlowCollectorExporter `json:"exporters"`

	// `networkPolicy` defines network policy settings for NetObserv components isolation.
	NetworkPolicy NetworkPolicy `json:"networkPolicy,omitempty"`

	// `execution` defines configuration related to the execution of the flow collection process.
	Execution FlowCollectorExecution `json:"execution,omitempty"`
}

type NetworkPolicy struct {
	// Deploys network policies on the namespaces used by NetObserv operands (main and privileged).
	// These network policies better isolate the NetObserv components to prevent undesired connections from and to them.
	// Because it cannot be tested with all CNIs, this option is only enabled by default when NetObserv runs in a known
	// supported environment, and it is disabled by default otherwise.
	// When disabled, it is highly recommended to create network policies manually, to prevent undesired accesses.
	// This setting is for operands only, and does not control the Operator network policy, which is covered by the `OPERATOR_NETWORK_POLICY`
	// environment variable.
	// If the operator and the operands are deployed in the same namespace, this setting is ignored, and only `OPERATOR_NETWORK_POLICY` controls
	// whether or not any policy is installed.
	// More information: https://github.com/netobserv/netobserv-operator/blob/main/docs/NetworkPolicy.md.
	// +optional
	Enable *bool `json:"enable,omitempty"`

	// `additionalNamespaces` contains additional namespaces allowed to connect to the NetObserv namespace.
	// It provides flexibility in the network policy configuration, but if you need a more specific
	// configuration, you can disable it and install your own instead.
	//+optional
	AdditionalNamespaces []string `json:"additionalNamespaces"`
}

type FlowCollectorAgentType string

const (
	AgentIPFIX FlowCollectorAgentType = "IPFIX"
	AgentEBPF  FlowCollectorAgentType = "eBPF"
)

// `FlowCollectorAgent` is a legacy discriminated union that allowed to select either IPFIX or eBPF. Only eBPF remains available.
// +union
type FlowCollectorAgent struct {
	// `type` selects the flows tracing agent.
	//
	// Deprecated: Previously, this field allowed to select between `eBPF` or `IPFIX`.
	// Only `eBPF` is allowed now, so this field is deprecated and is planned for removal in a future version of the API.
	//
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="eBPF";"IPFIX"
	// +kubebuilder:default:=eBPF
	Type FlowCollectorAgentType `json:"type,omitempty"`

	// `ipfix` describes the settings related to the IPFIX-based flow reporter when `spec.agent.type` is set to `IPFIX`.
	//
	// Deprecated: only `eBPF` remains supported.
	//
	// +optional
	IPFIX FlowCollectorIPFIX `json:"ipfix,omitempty"`

	// `ebpf` describes the settings related to the eBPF-based flow reporter when `spec.agent.type` is set to `eBPF`.
	// +optional
	EBPF FlowCollectorEBPF `json:"ebpf,omitempty"`
}

// `FlowCollectorIPFIX` defines a FlowCollector that uses IPFIX on OVN-Kubernetes to collect the flows information
//
// Deprecated: only `eBPF` agent type remains supported.
type FlowCollectorIPFIX struct {
	//+kubebuilder:validation:Pattern:=^\d+(ns|ms|s|m)?$
	//+kubebuilder:default:="20s"
	// `cacheActiveTimeout` is the max period during which the reporter aggregates flows before sending.
	CacheActiveTimeout string `json:"cacheActiveTimeout,omitempty" mapstructure:"cacheActiveTimeout,omitempty"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=400
	// `cacheMaxFlows` is the max number of flows in an aggregate; when reached, the reporter sends the flows.
	CacheMaxFlows int32 `json:"cacheMaxFlows,omitempty" mapstructure:"cacheMaxFlows,omitempty"`

	//+kubebuilder:validation:Minimum=2
	//+kubebuilder:default:=400
	// `sampling` is the sampling interval on the reporter. 100 means one flow on 100 is sent.
	// To ensure cluster stability, it is not possible to set a value below 2.
	// If you really want to sample every packet, which might impact the cluster stability,
	// refer to `forceSampleAll`. Alternatively, you can use the eBPF Agent instead of IPFIX.
	Sampling int32 `json:"sampling,omitempty" mapstructure:"sampling,omitempty"`

	//+kubebuilder:default:=false
	// `forceSampleAll` allows disabling sampling in the IPFIX-based flow reporter.
	// It is not recommended to sample all the traffic with IPFIX, as it might generate cluster instability.
	// If you REALLY want to do that, set this flag to `true`. Use at your own risk.
	// When it is set to `true`, the value of `sampling` is ignored.
	ForceSampleAll bool `json:"forceSampleAll,omitempty" mapstructure:"-"`

	// `clusterNetworkOperator` defines the settings related to the Cluster Network Operator, when available.
	ClusterNetworkOperator ClusterNetworkOperatorConfig `json:"clusterNetworkOperator,omitempty" mapstructure:"-"`

	// `ovnKubernetes` defines the settings of the OVN-Kubernetes network plugin, when available.
	// This configuration is used when using upstream OVN's IPFIX exports.
	OVNKubernetes OVNKubernetesConfig `json:"ovnKubernetes,omitempty" mapstructure:"-"`
}

// Agent feature, can be one of:<br>
// - `PacketDrop`, to track packet drops.<br>
// - `DNSTracking`, to track specific information on DNS traffic.<br>
// - `FlowRTT`, to track TCP latency.<br>
// - `NetworkEvents`, to track network events.<br>
// - `PacketTranslation`, to enrich flows with packets translation information, such as Service NAT.<br>
// - `EbpfManager`, to enable using eBPF Manager to manage NetObserv eBPF programs.<br>
// - `UDNMapping`, to enable interfaces mapping to UDN.<br>
// - `IPSec`, to track flows between nodes with IPsec encryption.<br>
// - `TLSTracking`, to track TLS usage.<br>
// +kubebuilder:validation:Enum:="PacketDrop";"DNSTracking";"FlowRTT";"NetworkEvents";"PacketTranslation";"EbpfManager";"UDNMapping";"IPSec";"TLSTracking"
type AgentFeature string

const (
	PacketDrop        AgentFeature = "PacketDrop"
	DNSTracking       AgentFeature = "DNSTracking"
	FlowRTT           AgentFeature = "FlowRTT"
	NetworkEvents     AgentFeature = "NetworkEvents"
	PacketTranslation AgentFeature = "PacketTranslation"
	EbpfManager       AgentFeature = "EbpfManager"
	UDNMapping        AgentFeature = "UDNMapping"
	IPSec             AgentFeature = "IPSec"
	TLSTracking       AgentFeature = "TLSTracking"
)

// Name of an eBPF agent alert.
// Possible values are:<br>
// `NetObservDroppedFlows`, which is triggered when the eBPF agent is missing packets or flows, such as when the BPF hashmap is busy or full, or the capacity limiter is being triggered.<br>
// +kubebuilder:validation:Enum:="NetObservDroppedFlows"
type EBPFAgentAlert string

const (
	AlertDroppedFlows EBPFAgentAlert = "NetObservDroppedFlows"
)

// `EBPFMetrics` defines the desired eBPF agent configuration regarding metrics
type EBPFMetrics struct {
	// Metrics server endpoint configuration for the Prometheus scraper.
	// +optional
	Server MetricsServerConfig `json:"server,omitempty"`

	// Set `enable` to `false` to disable eBPF agent metrics collection. It is enabled by default.
	// +optional
	Enable *bool `json:"enable,omitempty"`

	// `disableAlerts` is a list of alerts that should be disabled.
	// Possible values are:<br>
	// `NetObservDroppedFlows`, which is triggered when the eBPF agent is missing packets or flows, such as when the BPF hashmap is busy or full, or the capacity limiter is being triggered.<br>
	// +optional
	DisableAlerts []EBPFAgentAlert `json:"disableAlerts"`
}

// `EBPFFlowFilterRule` defines the desired eBPF agent configuration regarding flow filtering rule.
type EBPFFlowFilterRule struct {
	// `cidr` defines the IP CIDR to filter flows by.
	// Examples: `10.10.10.0/24` or `100:100:100:100::/64`
	CIDR string `json:"cidr,omitempty"`

	// `action` defines the action to perform on the flows that match the filter. The available options are `Accept`, which is the default, and `Reject`.
	// +kubebuilder:validation:Enum:="Accept";"Reject"
	Action string `json:"action,omitempty"`

	// `protocol` optionally defines a protocol to filter flows by. The available options are `TCP`, `UDP`, `ICMP`, `ICMPv6`, and `SCTP`.
	// +kubebuilder:validation:Enum:="TCP";"UDP";"ICMP";"ICMPv6";"SCTP"
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// `direction` optionally defines a direction to filter flows by. The available options are `Ingress` and `Egress`.
	// +kubebuilder:validation:Enum:="Ingress";"Egress"
	// +optional
	Direction string `json:"direction,omitempty"`

	// `tcpFlags` optionally defines TCP flags to filter flows by.
	// In addition to the standard flags (RFC-9293), you can also filter by one of the three following combinations: `SYN-ACK`, `FIN-ACK`, and `RST-ACK`.
	// +kubebuilder:validation:Enum:="SYN";"SYN-ACK";"ACK";"FIN";"RST";"URG";"ECE";"CWR";"FIN-ACK";"RST-ACK"
	// +optional
	TCPFlags string `json:"tcpFlags,omitempty"`

	// `sourcePorts` optionally defines the source ports to filter flows by.
	// To filter a single port, set a single port as an integer value. For example, `sourcePorts: 80`.
	// To filter a range of ports, use a "start-end" range in string format. For example, `sourcePorts: "80-100"`.
	// To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.
	// +optional
	SourcePorts intstr.IntOrString `json:"sourcePorts,omitempty"`

	// `destPorts` optionally defines the destination ports to filter flows by.
	// To filter a single port, set a single port as an integer value. For example, `destPorts: 80`.
	// To filter a range of ports, use a "start-end" range in string format. For example, `destPorts: "80-100"`.
	// To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.
	// +optional
	DestPorts intstr.IntOrString `json:"destPorts,omitempty"`

	// `ports` optionally defines the ports to filter flows by. It is used both for source and destination ports.
	// To filter a single port, set a single port as an integer value. For example, `ports: 80`.
	// To filter a range of ports, use a "start-end" range in string format. For example, `ports: "80-100"`.
	// To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.
	Ports intstr.IntOrString `json:"ports,omitempty"`

	// `peerIP` optionally defines the remote IP address to filter flows by.
	// Example: `10.10.10.10`.
	// +optional
	PeerIP string `json:"peerIP,omitempty"`

	// `peerCIDR` defines the Peer IP CIDR to filter flows by.
	// Examples: `10.10.10.0/24` or `100:100:100:100::/64`
	PeerCIDR string `json:"peerCIDR,omitempty"`

	// `icmpCode`, for Internet Control Message Protocol (ICMP) traffic, optionally defines the ICMP code to filter flows by.
	// +optional
	ICMPCode *int `json:"icmpCode,omitempty"`

	// `icmpType`, for ICMP traffic, optionally defines the ICMP type to filter flows by.
	// +optional
	ICMPType *int `json:"icmpType,omitempty"`

	// `pktDrops` optionally filters only flows containing packet drops.
	// +optional
	PktDrops *bool `json:"pktDrops,omitempty"`

	// `sampling` is the sampling interval for the matched packets, overriding the global sampling defined at `spec.agent.ebpf.sampling`.
	// +optional
	Sampling *uint32 `json:"sampling,omitempty"`
}

// `EBPFFlowFilter` defines the desired eBPF agent configuration regarding flow filtering.
type EBPFFlowFilter struct {
	// Set `enable` to `true` to enable the eBPF flow filtering feature.
	Enable *bool `json:"enable,omitempty"`

	// [Deprecated (*)]. This setting is not used anymore. It is replaced with the `rules` list.
	EBPFFlowFilterRule `json:",inline"`

	// `rules` defines a list of filtering rules on the eBPF Agents.
	// When filtering is enabled, by default, flows that don't match any rule are rejected.
	// To change the default, you can define a rule that accepts everything: `{ action: "Accept", cidr: "0.0.0.0/0" }`, and then refine with rejecting rules.
	// +kubebuilder:validation:MinItems:=1
	// +kubebuilder:validation:MaxItems:=16
	Rules []EBPFFlowFilterRule `json:"rules,omitempty"`
}

// `FlowCollectorEBPF` defines a FlowCollector that uses eBPF to collect the flows information
type FlowCollectorEBPF struct {
	//+kubebuilder:validation:Enum=IfNotPresent;Always;Never
	//+kubebuilder:default:=IfNotPresent
	// `imagePullPolicy` is the Kubernetes pull policy for the image defined above
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	//+kubebuilder:default:={requests:{memory:"50Mi",cpu:"100m"},limits:{memory:"800Mi"}}
	// `resources` are the compute resources required by this container.
	// For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	// Sampling interval of the eBPF probe. 100 means one packet on 100 is sent. 0 or 1 means all packets are sampled.
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=50
	//+optional
	Sampling *int32 `json:"sampling,omitempty"`

	// `cacheActiveTimeout` is the period during which the agent aggregates flows before sending.
	// Increasing `cacheMaxFlows` and `cacheActiveTimeout` can decrease the network traffic overhead and the CPU load,
	// however you can expect higher memory consumption and an increased latency in the flow collection.
	//+kubebuilder:validation:Pattern:=^\d+(ns|ms|s|m)?$
	//+kubebuilder:default:="15s"
	CacheActiveTimeout string `json:"cacheActiveTimeout,omitempty"`

	// `cacheMaxFlows` is the maximum number of flows in an aggregate; when reached, the reporter sends the flows.
	// Increasing `cacheMaxFlows` and `cacheActiveTimeout` can decrease the network traffic overhead and the CPU load,
	// however you can expect higher memory consumption and an increased latency in the flow collection.
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=120000
	CacheMaxFlows int32 `json:"cacheMaxFlows,omitempty"`

	// `interfaces` contains the interface names from where flows are collected. If empty, the agent
	// fetches all the interfaces in the system, excepting the ones listed in `excludeInterfaces`.
	// An entry enclosed by slashes, such as `/br-/`, is matched as a regular expression.
	// Otherwise it is matched as a case-sensitive string.
	//+optional
	Interfaces []string `json:"interfaces"`

	// `excludeInterfaces` contains the interface names that are excluded from flow tracing.
	// An entry enclosed by slashes, such as `/br-/`, is matched as a regular expression.
	// Otherwise it is matched as a case-sensitive string.
	//+kubebuilder:default:=lo;
	//+optional
	ExcludeInterfaces []string `json:"excludeInterfaces"`

	//+kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic
	//+kubebuilder:default:=info
	// `logLevel` defines the log level for the NetObserv eBPF Agent
	LogLevel string `json:"logLevel,omitempty"`

	// Privileged mode for the eBPF Agent container. When set to `true`, the agent is able to capture more traffic, including from secondary interfaces.
	// When ignored or set to `false`, the operator sets granular capabilities (BPF, PERFMON, NET_ADMIN) to the container.
	// Some agent features require the privileged mode, such as packet drops tracking (see `features`) and SR-IOV support.
	// +optional
	Privileged bool `json:"privileged,omitempty"`

	//+kubebuilder:default:=1048576
	// +optional
	// `kafkaBatchSize` limits the maximum size of a request in bytes before being sent to a partition. Ignored when not using Kafka. Default: 1MB.
	KafkaBatchSize int `json:"kafkaBatchSize"`

	// `advanced` allows setting some aspects of the internal configuration of the eBPF agent.
	// This section is aimed mostly for debugging and fine-grained performance optimizations,
	// such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk. You can also
	// override the default Linux capabilities from there.
	// +optional
	Advanced *AdvancedAgentConfig `json:"advanced,omitempty"`

	// List of additional features to enable. They are all disabled by default. Enabling additional features might have performance impacts. Possible values are:<br>
	// - `PacketDrop`: Enable the packets drop flows logging feature. This feature requires mounting
	// the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.<br>
	// - `DNSTracking`: Enable the DNS tracking feature.<br>
	// - `FlowRTT`: Enable flow latency (sRTT) extraction in the eBPF agent from TCP traffic.<br>
	// - `NetworkEvents`: Enable the network events monitoring feature, such as correlating flows and network policies.
	// This feature requires mounting the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.
	// It requires using the OVN-Kubernetes network plugin with the Observability feature.<br>
	// - `PacketTranslation`: Enable enriching flows with packet translation information, such as Service NAT.<br>
	// - `EbpfManager`: Use eBPF Manager to manage NetObserv eBPF programs. Pre-requisite: the eBPF Manager operator (or upstream bpfman operator) must be installed.<br>
	// - `UDNMapping`: Enable interfaces mapping to User Defined Networks (UDN).<br>
	// This feature requires mounting the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.
	// It requires using the OVN-Kubernetes network plugin.<br>
	// - `IPSec`, to track flows between nodes with IPsec encryption.<br>
	// - `TLSTracking`, to track TLS usage.<br>
	// +optional
	Features []AgentFeature `json:"features,omitempty"`

	// `metrics` defines the eBPF agent configuration regarding metrics.
	// +optional
	Metrics EBPFMetrics `json:"metrics,omitempty"`

	// `flowFilter` defines the eBPF agent configuration regarding flow filtering.
	// +optional
	FlowFilter *EBPFFlowFilter `json:"flowFilter,omitempty"`
}

// `FlowCollectorKafka` defines the desired Kafka config of FlowCollector
type FlowCollectorKafka struct {
	// +kubebuilder:default:=""
	// Address of the Kafka server
	Address string `json:"address"`

	// +kubebuilder:default:=""
	// Kafka topic to use. It must exist. NetObserv does not create it.
	Topic string `json:"topic"`

	// +kubebuilder:validation:Enum:="none";"gzip";"snappy";"lz4";"zstd"
	// +kubebuilder:default:="none"
	// Compression codec to use when producing messages to Kafka.
	// Accepted values are: `none` (default), `gzip`, `snappy`, `lz4`, `zstd`.
	// +optional
	Compression string `json:"compression,omitempty"`

	// TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
	// We recommend the use of mTLS for higher security standards.
	// When configuring TLS, the operator watches the certificate secret and copies it to both the netobserv and netobserv-privileged namespaces.
	// In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
	// Refer to the Kafka configuration documentation for more information.
	// +optional
	TLS ClientTLS `json:"tls"`

	// SASL authentication configuration.
	// +optional
	SASL SASLConfig `json:"sasl"`
}

type FlowCollectorIPFIXReceiver struct {
	// +kubebuilder:default:=""
	// Address of the IPFIX external receiver.
	TargetHost string `json:"targetHost"`

	// Port for the IPFIX external receiver.
	// +kubebuilder:default:=4739
	TargetPort int `json:"targetPort"`

	// Transport protocol (`TCP` or `UDP`) to be used for the IPFIX connection, defaults to `TCP`.
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="TCP";"UDP"
	// +optional
	Transport string `json:"transport,omitempty"`

	// EnterpriseID, or Private Enterprise Number (PEN). To date, NetObserv does not own an assigned number,
	// so it is left open for configuration. The PEN is needed to collect non standard data, such as Kubernetes names,
	// RTT, etc.
	// +kubebuilder:default:=2
	EnterpriseID int `json:"enterpriseID"`
}

type FlowCollectorOpenTelemetryLogs struct {
	// Set `enable` to `true` to send logs to an OpenTelemetry receiver.
	// +kubebuilder:default:=true
	Enable *bool `json:"enable,omitempty"`
}

type FlowCollectorOpenTelemetryMetrics struct {
	// Set `enable` to `true` to send metrics to an OpenTelemetry receiver.
	// +kubebuilder:default:=true
	Enable *bool `json:"enable,omitempty"`

	// Specify how often metrics are sent to a collector.
	// +kubebuilder:default:="20s"
	PushTimeInterval *metav1.Duration `json:"pushTimeInterval,omitempty"`
}

type GenericTransformRule struct {
	Input      string `json:"input,omitempty"`
	Output     string `json:"output,omitempty"`
	Multiplier int    `json:"multiplier,omitempty"`
}

type GenericTransform []GenericTransformRule

type FlowCollectorOpenTelemetry struct {
	// Address of the OpenTelemetry receiver.
	// +kubebuilder:default:=""
	TargetHost string `json:"targetHost"`

	// Port for the OpenTelemetry receiver.
	// +kubebuilder:default:=4317
	TargetPort int `json:"targetPort"`

	// Protocol of the OpenTelemetry connection. The available options are `http` and `grpc`.
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="http";"grpc"
	// +optional
	Protocol string `json:"protocol,omitempty"`

	// Headers to add to messages (optional)
	// +optional
	Headers map[string]string `json:"headers,omitempty"`

	// TLS client configuration.
	// +optional
	TLS ClientTLS `json:"tls"`

	// Custom fields mapping to an OpenTelemetry conformant format.
	// By default, NetObserv format proposal is used: https://github.com/rhobs/observability-data-model/blob/main/network-observability.md#format-proposal .
	// As there is currently no accepted standard for L3 or L4 enriched network logs, you can freely override it with your own.
	// +optional
	FieldsMapping *[]GenericTransformRule `json:"fieldsMapping,omitempty"`

	// OpenTelemetry configuration for logs.
	// +optional
	Logs FlowCollectorOpenTelemetryLogs `json:"logs"`

	// OpenTelemetry configuration for metrics.
	// +optional
	Metrics FlowCollectorOpenTelemetryMetrics `json:"metrics"`
}

type TLSConfigType string

const (
	TLSDisabled TLSConfigType = "Disabled"
	TLSProvided TLSConfigType = "Provided"
	TLSAuto     TLSConfigType = "Auto"
	TLSAutoMTLS TLSConfigType = "Auto-mTLS"
)

// `ServerTLS` define the TLS configuration, server side.
type ServerTLS struct {
	// Select the type of TLS configuration:<br>
	// - `Disabled` (default) to not configure TLS for the endpoint.
	// - `Provided` to manually provide cert file and a key file.
	// - `Auto` to use a default certificate, which may vary depending on the Kubernetes vendor.
	// Refer to https://github.com/netobserv/netobserv-operator/blob/main/docs/TLS.md for more information.
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="Disabled";"Provided";"Auto"
	// +kubebuilder:validation:Required
	//+kubebuilder:default:="Disabled"
	Type TLSConfigType `json:"type,omitempty"`

	// TLS configuration when `type` is set to `Provided`.
	// +optional
	Provided *CertificateReference `json:"provided"`

	//+kubebuilder:default:=false
	// `insecureSkipVerify` allows skipping client-side verification of the provided certificate.
	// If set to `true`, the `providedCaFile` field is ignored. For security, this should not be used other than for testing or demo.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// Reference to the CA file when `type` is set to `Provided`.
	// +optional
	ProvidedCAFile *FileReference `json:"providedCaFile,omitempty"`
}

// `ClientServerTLS` define the TLS configuration for both client and server sides
type ClientServerTLS struct {
	// TLS client certificate reference, used for mTLS. Leave unset for simple TLS.
	// +optional
	ClientCert *CertificateReference `json:"clientCert,omitempty"`

	// TLS server certificate reference.
	// +optional
	ServerCert *CertificateReference `json:"serverCert,omitempty"`

	// Reference to the CA file.
	// +optional
	CAFile *FileReference `json:"caFile,omitempty"`
}

// `MetricsServerConfig` define the metrics server endpoint configuration for Prometheus scraper.
type MetricsServerConfig struct {
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	// The metrics server HTTP port.
	Port *int32 `json:"port,omitempty"`

	// TLS configuration.
	// +optional
	TLS ServerTLS `json:"tls"`

	// Prometheus scraping interval, how often metrics are pulled.
	//+kubebuilder:validation:Format=duration
	// +optional
	ScrapeInterval *metav1.Duration `json:"scrapeInterval,omitempty"` // Warning: keep as pointer, else default is ignored
}

// Metric name. More information in https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md.
// +kubebuilder:validation:Enum:="namespace_egress_bytes_total";"namespace_egress_packets_total";"namespace_ingress_bytes_total";"namespace_ingress_packets_total";"namespace_flows_total";"node_egress_bytes_total";"node_egress_packets_total";"node_ingress_bytes_total";"node_ingress_packets_total";"node_flows_total";"workload_egress_bytes_total";"workload_egress_packets_total";"workload_ingress_bytes_total";"workload_ingress_packets_total";"workload_flows_total";"namespace_drop_bytes_total";"namespace_drop_packets_total";"node_drop_bytes_total";"node_drop_packets_total";"workload_drop_bytes_total";"workload_drop_packets_total";"namespace_rtt_seconds";"node_rtt_seconds";"workload_rtt_seconds";"namespace_dns_latency_seconds";"node_dns_latency_seconds";"workload_dns_latency_seconds";"namespace_dns_flows_total";"node_dns_flows_total";"workload_dns_flows_total";"node_network_policy_events_total";"namespace_network_policy_events_total";"workload_network_policy_events_total";"node_ipsec_flows_total";"namespace_ipsec_flows_total";"workload_ipsec_flows_total";"node_tls_flows_total";"namespace_tls_flows_total";"workload_tls_flows_total";"node_to_node_ingress_flows_total"
type FLPMetric string

// `FLPMetrics` define the desired FLP configuration regarding metrics
type FLPMetrics struct {
	// Metrics server endpoint configuration for Prometheus scraper
	// +optional
	Server MetricsServerConfig `json:"server,omitempty"`

	// `includeList` is a list of metric names to specify which ones to generate.
	// The names correspond to the names in Prometheus without the prefix. For example,
	// `namespace_egress_packets_total` shows up as `netobserv_namespace_egress_packets_total` in Prometheus.
	// Note that the more metrics you add, the bigger is the impact on Prometheus workload resources.
	// Metrics enabled by default are:
	// `namespace_flows_total`, `node_ingress_bytes_total`, `node_egress_bytes_total`, `workload_ingress_bytes_total`,
	// `workload_egress_bytes_total`, `namespace_drop_packets_total` (when `PacketDrop` feature is enabled),
	// `namespace_rtt_seconds` (when `FlowRTT` feature is enabled), `namespace_dns_latency_seconds` and `namespace_dns_flows_total` (when `DNSTracking` feature is enabled),
	// `namespace_network_policy_events_total` (when `NetworkEvents` feature is enabled).
	// More information, with full list of available metrics: https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md
	// +optional
	IncludeList *[]FLPMetric `json:"includeList,omitempty"`

	// `additionalIncludeList` is a list of metric names to include in addition to the default metrics.
	// Unlike `includeList`, this appends to the default list rather than replacing it.
	// This field is mutually exclusive with `includeList`. If `includeList` is set, `additionalIncludeList` is ignored.
	// The names correspond to the names in Prometheus without the prefix. For example,
	// `namespace_egress_packets_total` shows up as `netobserv_namespace_egress_packets_total` in Prometheus.
	// Note that the more metrics you add, the bigger is the impact on Prometheus workload resources.
	// More information, with full list of available metrics: https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md
	// +optional
	AdditionalIncludeList *[]FLPMetric `json:"additionalIncludeList,omitempty"`

	// `disableAlerts` is a list of alert groups that should be disabled from the default set of alerts.
	// Possible values are: `NetObservNoFlows`, `NetObservLokiError`, `PacketDropsByKernel`, `PacketDropsByDevice`, `IPsecErrors`, `NetpolDenied`,
	// `LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`, `TLSInsecureVersion`.
	// More information on alerts: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md
	// +optional
	DisableAlerts []HealthRuleTemplate `json:"disableAlerts"`

	// `healthRules` is a list of health rules to be created for Prometheus, organized by templates and variants.
	// Each health rule can be configured to generate either alerts or recording rules based on the mode field.
	// More information on health rules: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md
	// +optional
	HealthRules *[]FLPHealthRule `json:"healthRules"`
}

type FLPLogTypes string

const (
	LogTypeFlows              FLPLogTypes = "Flows"
	LogTypeConversations      FLPLogTypes = "Conversations"
	LogTypeEndedConversations FLPLogTypes = "EndedConversations"
	LogTypeAll                FLPLogTypes = "All"
)

// `FlowCollectorFLP` defines the desired flowlogs-pipeline state of FlowCollector
type FlowCollectorFLP struct {
	//+kubebuilder:validation:Enum=IfNotPresent;Always;Never
	//+kubebuilder:default:=IfNotPresent
	// `imagePullPolicy` is the Kubernetes pull policy for the image defined above
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// `Metrics` define the processor configuration regarding metrics
	Metrics FLPMetrics `json:"metrics,omitempty"`

	//+kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic
	//+kubebuilder:default:=info
	// `logLevel` of the processor runtime
	LogLevel string `json:"logLevel,omitempty"`

	//+kubebuilder:default:={requests:{memory:"100Mi",cpu:"100m"},limits:{memory:"800Mi"}}
	// `resources` are the compute resources required by this container.
	// For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=3
	// `kafkaConsumerReplicas` defines the number of replicas (pods) to start for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
	// This setting is ignored when Kafka is disabled.
	//
	// Deprecated: use `spec.processor.consumerReplicas` instead.
	KafkaConsumerReplicas *int32 `json:"kafkaConsumerReplicas,omitempty"`

	// `kafkaConsumerAutoscaler` is the spec of a horizontal pod autoscaler to set up for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
	// This setting is ignored when Kafka is disabled.
	//
	// Deprecated: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.processor.unmanagedReplicas` to `true`.
	//
	// +optional
	KafkaConsumerAutoscaler FlowCollectorHPA `json:"kafkaConsumerAutoscaler,omitempty"`

	//+kubebuilder:validation:Minimum=0
	// `consumerReplicas` defines the number of replicas (pods) to start for `flowlogs-pipeline`, default is 3.
	// This setting is ignored when `spec.deploymentModel` is `Direct` or when `spec.processor.unmanagedReplicas` is `true`.
	ConsumerReplicas *int32 `json:"consumerReplicas,omitempty"`

	// If `unmanagedReplicas` is `true`, the operator will not reconcile `consumerReplicas`. This is useful when using a pod autoscaler.
	// +optional
	UnmanagedReplicas bool `json:"unmanagedReplicas,omitempty"`

	//+kubebuilder:default:=1000
	// +optional
	// `kafkaConsumerQueueCapacity` defines the capacity of the internal message queue used in the Kafka consumer client. Ignored when not using Kafka.
	KafkaConsumerQueueCapacity int `json:"kafkaConsumerQueueCapacity"`

	//+kubebuilder:default:=10485760
	// +optional
	// `kafkaConsumerBatchSize` indicates to the broker the maximum batch size, in bytes, that the consumer accepts. Ignored when not using Kafka. Default: 10MB.
	KafkaConsumerBatchSize int `json:"kafkaConsumerBatchSize"`

	// `logTypes` defines the desired record types to generate. Possible values are:<br>
	// - `Flows` to export regular network flows. This is the default.<br>
	// - `Conversations` to generate events for started conversations, ended conversations as well as periodic "tick" updates. Note that in this mode, Prometheus metrics are not accurate on long-standing conversations.<br>
	// - `EndedConversations` to generate only ended conversations events. Note that in this mode, Prometheus metrics are not accurate on long-standing conversations.<br>
	// - `All` to generate both network flows and all conversations events. It is not recommended due to the impact on resources footprint.<br>
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:="Flows";"Conversations";"EndedConversations";"All"
	// +kubebuilder:default:=Flows
	LogTypes *FLPLogTypes `json:"logTypes,omitempty"`

	//+kubebuilder:default:=""
	// +optional
	// `clusterName` is the name of the cluster to appear in the flows data. In a multi-cluster context, it makes it possible to identify the flows provenance.
	ClusterName string `json:"clusterName,omitempty"`

	//+kubebuilder:default:=false
	// Set `multiClusterDeployment` to `true` to enable multi clusters feature. This adds `clusterName` label to flows data
	MultiClusterDeployment *bool `json:"multiClusterDeployment,omitempty"`

	//+optional
	// `addZone` allows availability zone awareness by labeling flows with their source and destination zones.
	// This feature requires the "topology.kubernetes.io/zone" label to be set on nodes.
	AddZone *bool `json:"addZone,omitempty"`

	//+optional
	// `subnetLabels` allows to define custom labels on subnets and IPs and, for supported vendors, to enable automatic labeling of recognized subnets, which is used to identify cluster external traffic.
	// When a subnet matches the source or destination IP of a flow, a corresponding field is added: `SrcSubnetLabel` or `DstSubnetLabel`.
	SubnetLabels SubnetLabels `json:"subnetLabels,omitempty"`

	//+optional
	// `deduper` allows you to sample or drop flows identified as duplicates, in order to save on resource usage.
	Deduper *FLPDeduper `json:"deduper,omitempty"`

	// +optional
	// `filters` lets you define custom filters to limit the amount of generated flows.
	// These filters provide more flexibility than the eBPF Agent filters (in `spec.agent.ebpf.flowFilter`), such as allowing to filter by Kubernetes namespace,
	// but with a lesser improvement in performance.
	Filters []FLPFilterSet `json:"filters"`

	// Global configuration managing FlowCollectorSlices custom resources.
	//+optional
	SlicesConfig *SlicesConfig `json:"slicesConfig,omitempty"`

	// Service configuration, only used when `spec.deploymentModel` is `Service`.
	// +optional
	Service *ProcessorServiceConfig `json:"service,omitempty"`

	// `informerCacheProxy` configuration for centralized Kubernetes informers that push cache updates to flowlogs-pipeline processors.
	// This reduces load on the Kubernetes API server by having a single component query the API instead of N FLP processors.
	// When enabled, a dedicated deployment is created that watches Kubernetes resources and pushes updates via gRPC.
	// Benefits: Reduced API server load on large clusters with many FLP replicas.
	// Drawbacks: More complex deployment (additional component), higher resource usage on small clusters.
	// Recommended only for clusters with many FLP replicas (>3) or when API server load is a concern.
	// +optional
	InformerCacheProxy *FlowCollectorInformerCacheProxy `json:"informerCacheProxy,omitempty"`

	// `advanced` allows setting some aspects of the internal configuration of the flow processor.
	// This section is aimed mostly for debugging and fine-grained performance optimizations,
	// such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.
	// +optional
	Advanced *AdvancedProcessorConfig `json:"advanced,omitempty"`
}

// `FlowCollectorInformerCacheProxy` defines the configuration for the informer cache proxy
type FlowCollectorInformerCacheProxy struct {
	// `enabled` controls whether to deploy the informer cache proxy.
	// When `true`, a dedicated deployment watches K8s resources and pushes cache updates via gRPC to FLP processors, reducing API server load.
	// When `false` (default), each FLP processor uses local informers.
	// Enable only on large clusters or when API server load is a concern, as it adds deployment complexity.
	// +kubebuilder:default:=false
	Enabled *bool `json:"enabled,omitempty"`

	// `replicas` defines the number of replicas for the flowlogs-pipeline-informers deployment.
	// For high availability, a minimum of 2 replicas is required when `enabled` is `true`.
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:default:=2
	Replicas *int32 `json:"replicas,omitempty"`

	// `resources` are the compute resources required by the informer cache proxy container.
	// For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +kubebuilder:default:={requests:{memory:"128Mi",cpu:"50m"},limits:{memory:"256Mi",cpu:"200m"}}
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	// `advanced` allows setting some technical parameters of the informer cache proxy component.
	// +optional
	Advanced *AdvancedInformerCacheProxyConfig `json:"advanced,omitempty"`

	// `tls` defines the TLS configuration for the gRPC communication between the informer cache proxy and processors.
	// +optional
	TLS *InformerCacheProxyTLSConfig `json:"tls,omitempty"`
}

// `AdvancedInformerCacheProxyConfig` defines advanced configuration for the informer cache proxy component.
type AdvancedInformerCacheProxyConfig struct {
	// `resyncInterval` defines the interval in seconds to rediscover processors and sync state.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=60
	// +optional
	ResyncInterval *int `json:"resyncInterval,omitempty"`

	// `batchSize` defines the maximum number of cache entries to send in a single update batch.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=100
	// +optional
	BatchSize *int `json:"batchSize,omitempty"`

	// `sendTimeout` defines the timeout in seconds for sending updates to processors.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=10
	// +optional
	SendTimeout *int `json:"sendTimeout,omitempty"`

	// `updateBufferSize` defines the size of the internal update channel buffer.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=100
	// +optional
	UpdateBufferSize *int `json:"updateBufferSize,omitempty"`

	// `processorPort` defines the gRPC port where flowlogs-pipeline processors listen for k8s cache updates.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default:=9402
	// +optional
	ProcessorPort *int32 `json:"processorPort,omitempty"`
}

type FLPDeduperMode string

const (
	FLPDeduperDisabled FLPDeduperMode = "Disabled"
	FLPDeduperDrop     FLPDeduperMode = "Drop"
	FLPDeduperSample   FLPDeduperMode = "Sample"
)

// `FLPDeduper` defines the desired configuration for FLP-based deduper.
type FLPDeduper struct {
	// Set the Processor de-duplication mode. It comes in addition to the Agent-based deduplication, since the Agent cannot de-duplicate same flows reported from different nodes.<br>
	// - Use `Drop` to drop every flow considered as duplicates, allowing saving more on resource usage but potentially losing some information such as the network interfaces used from peer, or network events.<br>
	// - Use `Sample` to randomly keep only one flow on 50, which is the default, among the ones considered as duplicates. This is a compromise between dropping every duplicate or keeping every duplicate. This sampling action comes in addition to the Agent-based sampling. If both Agent and Processor sampling values are `50`, the combined sampling is 1:2500.<br>
	// - Use `Disabled` to turn off Processor-based de-duplication.<br>
	// +kubebuilder:validation:Enum:="Disabled";"Drop";"Sample"
	// +kubebuilder:default:=Disabled
	Mode FLPDeduperMode `json:"mode,omitempty"`

	// `sampling` is the sampling interval when deduper `mode` is `Sample`. For example, a value of `50` means that 1 flow in 50 is sampled.
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=50
	Sampling int32 `json:"sampling,omitempty"`
}

type FLPFilterTarget string

const (
	FLPFilterTargetAll       FLPFilterTarget = ""
	FLPFilterTargetLoki      FLPFilterTarget = "Loki"
	FLPFilterTargetMetrics   FLPFilterTarget = "Metrics"
	FLPFilterTargetExporters FLPFilterTarget = "Exporters"
)

// `FLPFilterSet` defines the desired configuration for FLP-based filtering satisfying all conditions.
type FLPFilterSet struct {
	// A query that selects the network flows to keep. More information about this query language in https://github.com/netobserv/flowlogs-pipeline/blob/main/docs/filtering.md.
	// +optional
	Query string `json:"query"`

	// If specified, these filters target a single output: `Loki`, `Metrics` or `Exporters`. By default, all outputs are targeted.
	// +optional
	// +kubebuilder:validation:Enum:="";"Loki";"Metrics";"Exporters"
	OutputTarget FLPFilterTarget `json:"outputTarget,omitempty"`

	// `sampling` is an optional sampling interval to apply to this filter. For example, a value of `50` means that 1 matching flow in 50 is sampled.
	//+kubebuilder:validation:Minimum=0
	// +optional
	Sampling int32 `json:"sampling,omitempty"`
}

type HPAStatus string

const (
	HPAStatusDisabled HPAStatus = "Disabled"
	HPAStatusEnabled  HPAStatus = "Enabled"
)

type FlowCollectorHPA struct {
	// +kubebuilder:validation:Enum:=Disabled;Enabled
	// +kubebuilder:default:=Disabled
	// `status` describes the desired status regarding deploying an horizontal pod autoscaler.<br>
	// - `Disabled` does not deploy an horizontal pod autoscaler.<br>
	// - `Enabled` deploys an horizontal pod autoscaler.<br>
	Status HPAStatus `json:"status,omitempty"`

	// `minReplicas` is the lower limit for the number of replicas to which the autoscaler
	// can scale down. It defaults to 1 pod. minReplicas is allowed to be 0 if the
	// alpha feature gate HPAScaleToZero is enabled and at least one Object or External
	// metric is configured. Scaling is active as long as at least one metric value is
	// available.
	// +optional
	MinReplicas *int32 `json:"minReplicas,omitempty" protobuf:"varint,2,opt,name=minReplicas"`

	// `maxReplicas` is the upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.
	// +kubebuilder:default:=3
	// +optional
	MaxReplicas int32 `json:"maxReplicas" protobuf:"varint,3,opt,name=maxReplicas"`

	// Metrics used by the pod autoscaler. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/horizontal-pod-autoscaler-v2/
	// +optional
	Metrics []ascv2.MetricSpec `json:"metrics"`
}

type SliceCollectionMode string

const (
	CollectionAlwaysCollect SliceCollectionMode = "AlwaysCollect"
	CollectionAllowList     SliceCollectionMode = "AllowList"
)

type SlicesConfig struct {
	// `enable` determines if the FlowCollectorSlice feature is enabled. If not, all resources of kind FlowCollectorSlice are simply ignored.
	//+kubebuilder:default:=false
	//+kubebuilder:validation:Required
	Enable bool `json:"enable,omitempty"`

	// `collectionMode` determines how the FlowCollectorSlice custom resources impacts the flow collection process:<br>
	// - When set to `AlwaysCollect`, all flows are collected regardless of the presence of FlowCollectorSlice.<br>
	// - When set to `AllowList`, only the flows related to namespaces where a FlowCollectorSlice resource is present, or configured via the global `namespacesAllowList`, are collected.<br>
	//+kubebuilder:validation:Enum=AlwaysCollect;AllowList
	//+kubebuilder:default:="AlwaysCollect"
	CollectionMode SliceCollectionMode `json:"collectionMode,omitempty"`

	// `namespacesAllowList` is a list of namespaces for which flows are always collected, regardless of the presence of FlowCollectorSlice in those namespaces.
	// An entry enclosed by slashes, such as `/openshift-.*/`, is matched as a regular expression.
	// This setting is ignored if `collectionMode` is different from `AllowList`.
	//+kubebuilder:validation:optional
	NamespacesAllowList []string `json:"namespacesAllowList,omitempty"`
}

type LokiAuthToken string

const (
	LokiAuthDisabled         LokiAuthToken = "Disabled"
	LokiAuthUseHostToken     LokiAuthToken = "Host"
	LokiAuthForwardUserToken LokiAuthToken = "Forward"
)

// `LokiManualParams` defines the full connection parameters to Loki.
type LokiManualParams struct {
	//+kubebuilder:default:="http://loki:3100/"
	// `ingesterUrl` is the address of an existing Loki ingester service to push the flows to. When using the Loki Operator,
	// set it to the Loki gateway service with the `network` tenant set in path, for example
	// https://loki-gateway-http.netobserv.svc:8080/api/logs/v1/network.
	IngesterURL string `json:"ingesterUrl,omitempty"`

	//+kubebuilder:default:="http://loki:3100/"
	// `querierUrl` specifies the address of the Loki querier service.
	// When using the Loki Operator, set it to the Loki gateway service with the `network` tenant set in path, for example
	// https://loki-gateway-http.netobserv.svc:8080/api/logs/v1/network.
	QuerierURL string `json:"querierUrl,omitempty"`

	//+kubebuilder:validation:optional
	// `statusUrl` specifies the address of the Loki `/ready`, `/metrics` and `/config` endpoints, in case it is different from the
	// Loki querier URL. If empty, the `querierUrl` value is used.
	// This is useful to show error messages and some context in the frontend.
	// When using the Loki Operator, set it to the Loki HTTP query frontend service, for example
	// https://loki-query-frontend-http.netobserv.svc:3100/.
	// `statusTLS` configuration is used when `statusUrl` is set.
	StatusURL string `json:"statusUrl,omitempty"`

	//+kubebuilder:default:="netobserv"
	// `tenantID` is the Loki `X-Scope-OrgID` that identifies the tenant for each request.
	// When using the Loki Operator, set it to `network`, which corresponds to a special tenant mode.
	TenantID string `json:"tenantID,omitempty"`

	//+kubebuilder:validation:Enum:="Disabled";"Host";"Forward"
	//+kubebuilder:default:="Disabled"
	// `authToken` describes the way to get a token to authenticate to Loki.<br>
	// - `Disabled` does not send any token with the request.<br>
	// - `Forward` forwards the user token for authorization.<br>
	// - `Host` (deprecated) - uses the local pod service account to authenticate to Loki.<br>
	// When using the Loki Operator, this must be set to `Forward`.
	AuthToken LokiAuthToken `json:"authToken,omitempty"`

	// TLS client configuration for Loki URL.
	// +optional
	TLS ClientTLS `json:"tls"`

	// TLS client configuration for Loki status URL.
	// +optional
	StatusTLS ClientTLS `json:"statusTls"`
}

// LokiMicroservicesParams is the configuration for microservices Loki (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#microservices-mode)
type LokiMicroservicesParams struct {
	//+kubebuilder:default:="http://loki-distributor:3100/"
	// `ingesterUrl` is the address of an existing Loki ingester service to push the flows to.
	IngesterURL string `json:"ingesterUrl,omitempty"`

	//+kubebuilder:default:="http://loki-query-frontend:3100/"
	// `querierURL` specifies the address of the Loki querier service.
	QuerierURL string `json:"querierUrl,omitempty"`

	//+kubebuilder:default:="netobserv"
	// `tenantID` is the Loki `X-Scope-OrgID` header that identifies the tenant for each request.
	TenantID string `json:"tenantID,omitempty"`

	// TLS client configuration for Loki URL.
	// +optional
	TLS ClientTLS `json:"tls"`
}

// LokiMonolithParams is the configuration for monolithic Loki (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#monolithic-mode)
type LokiMonolithParams struct {
	// Set `installDemoLoki` to `true` to automatically create Loki deployment, service and storage.
	// This is meant for development and demo use only, and not recommended in production.
	//+kubebuilder:default:=false
	InstallDemoLoki *bool `json:"installDemoLoki,omitempty"`

	//+kubebuilder:default:="http://loki:3100/"
	// `url` is the unique address of an existing Loki service that points to both the ingester and the querier.
	URL string `json:"url,omitempty"`

	//+kubebuilder:default:="netobserv"
	// `tenantID` is the Loki `X-Scope-OrgID` header that identifies the tenant for each request.
	TenantID string `json:"tenantID,omitempty"`

	// TLS client configuration for Loki URL.
	// +optional
	TLS ClientTLS `json:"tls"`
}

// LokiStackRef defines the name and namespace of the LokiStack instance
type LokiStackRef struct {
	// Name of an existing LokiStack resource to use.
	//+kubebuilder:default:="loki"
	//+kubebuilder:validation:Required
	Name string `json:"name,omitempty"`

	// Namespace where this `LokiStack` resource is located. If omitted, it is assumed to be the same as `spec.namespace`.
	// When configuring a different namespace, the operator watches certificate secret and copies it to the netobserv main namespaces.
	// In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
	// Refer to the Loki configuration documentation for more information.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

type LokiMode string

const (
	LokiModeManual        LokiMode = "Manual"
	LokiModeLokiStack     LokiMode = "LokiStack"
	LokiModeMonolithic    LokiMode = "Monolithic"
	LokiModeMicroservices LokiMode = "Microservices"
)

// `FlowCollectorLoki` defines the desired state for FlowCollector's Loki client.
type FlowCollectorLoki struct {
	// Set `enable` to `true` to store flows in Loki.
	// The Console plugin can use either Loki or Prometheus as a data source for metrics (see also `spec.prometheus.querier`), or both.
	// Not all queries are transposable from Loki to Prometheus. Hence, if Loki is disabled, some features of the plugin are disabled as well,
	// such as getting per-pod information or viewing raw flows.
	// If both Prometheus and Loki are enabled, Prometheus takes precedence and Loki is used as a fallback for queries that Prometheus cannot handle.
	// If they are both disabled, the Console plugin is not deployed.
	//+kubebuilder:default:=true
	Enable *bool `json:"enable,omitempty"`

	// `mode` must be set according to the installation mode of Loki:<br>
	// - Use `LokiStack` when Loki is managed using the Loki Operator<br>
	// - Use `Monolithic` when Loki is installed as a monolithic workload<br>
	// - Use `Microservices` when Loki is installed as microservices, but without Loki Operator<br>
	// - Use `Manual` if none of the options above match your setup<br>
	//+unionDiscriminator
	//+kubebuilder:validation:Enum=Manual;LokiStack;Monolithic;Microservices
	//+kubebuilder:default:="Monolithic"
	//+kubebuilder:validation:Required
	Mode LokiMode `json:"mode,omitempty"`

	// Loki configuration for `Manual` mode. This is the most flexible configuration.
	// It is ignored for other modes.
	// +optional
	Manual LokiManualParams `json:"manual,omitempty"`

	// Loki configuration for `Microservices` mode.
	// Use this option when Loki is installed using the microservices deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#microservices-mode).
	// It is ignored for other modes.
	// +optional
	Microservices LokiMicroservicesParams `json:"microservices,omitempty"`

	// Loki configuration for `Monolithic` mode.
	// Use this option when Loki is installed using the monolithic deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#monolithic-mode).
	// It is ignored for other modes.
	// +optional
	Monolithic LokiMonolithParams `json:"monolithic,omitempty"`

	// Loki configuration for `LokiStack` mode. This is useful for an easy Loki Operator configuration.
	// It is ignored for other modes.
	// +optional
	LokiStack LokiStackRef `json:"lokiStack,omitempty"`

	//+kubebuilder:default:="30s"
	// `readTimeout` is the maximum console plugin loki query total time limit.
	// A timeout of zero means no timeout.
	ReadTimeout *metav1.Duration `json:"readTimeout,omitempty"` // Warning: keep as pointer, else default is ignored

	//+kubebuilder:default:="10s"
	// `writeTimeout` is the maximum Loki time connection / request limit.
	// A timeout of zero means no timeout.
	WriteTimeout *metav1.Duration `json:"writeTimeout,omitempty"` // Warning: keep as pointer, else default is ignored

	//+kubebuilder:default:="1s"
	// `writeBatchWait` is the maximum time to wait before sending a Loki batch.
	WriteBatchWait *metav1.Duration `json:"writeBatchWait,omitempty"` // Warning: keep as pointer, else default is ignored

	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:default:=10485760
	// `writeBatchSize` is the maximum batch size (in bytes) of Loki logs to accumulate before sending.
	WriteBatchSize int64 `json:"writeBatchSize,omitempty"`

	// `advanced` allows setting some aspects of the internal configuration of the Loki clients.
	// This section is aimed mostly for debugging and fine-grained performance optimizations.
	// +optional
	Advanced *AdvancedLokiConfig `json:"advanced,omitempty"`
}

// `PrometheusQuerierManual` defines the full connection parameters to Prometheus.
type PrometheusQuerierManual struct {
	//+kubebuilder:default:="http://prometheus:9090"
	// `url` is the address of an existing Prometheus service to use for querying metrics.
	URL string `json:"url,omitempty"`

	// TLS client configuration for Prometheus URL.
	// +optional
	TLS ClientTLS `json:"tls"`

	// Set `true` to forward logged in user token in queries to Prometheus
	// +optional
	ForwardUserToken bool `json:"forwardUserToken"`

	// AlertManager configuration. This is used in the console to query silenced alerts, for displaying health information.
	// +optional
	AlertManager *AlertManagerQuerierManual `json:"alertManager"`
}

// `AlertManagerQuerierManual` defines the full connection parameters to Prometheus AlertManager.
type AlertManagerQuerierManual struct {
	// `url` is the address of an existing Prometheus AlertManager service to use for querying alerts.
	// +optional
	URL string `json:"url,omitempty"`

	// TLS client configuration for Prometheus AlertManager URL.
	// +optional
	TLS ClientTLS `json:"tls"`
}

type PrometheusMode string

const (
	PromModeAuto   PrometheusMode = "Auto"
	PromModeManual PrometheusMode = "Manual"
)

// `FlowCollectorPrometheus` defines the desired state for usage of Prometheus.
type FlowCollectorPrometheus struct {
	// Prometheus querying configuration, such as client settings, used in the Console plugin.
	Querier PrometheusQuerier `json:"querier,omitempty"`
}

// `PrometheusQuerier` defines the desired state for querying Prometheus (client...)
type PrometheusQuerier struct {
	// When `enable` is `true`, the Console plugin queries flow metrics from Prometheus instead of Loki whenever possible.
	// It is enabled by default: set it to `false` to disable this feature.
	// The Console plugin can use either Loki or Prometheus as a data source for metrics (see also `spec.loki`), or both.
	// Not all queries are transposable from Loki to Prometheus. Hence, if Loki is disabled, some features of the plugin are disabled as well,
	// such as getting per-pod information or viewing raw flows.
	// If both Prometheus and Loki are enabled, Prometheus takes precedence and Loki is used as a fallback for queries that Prometheus cannot handle.
	// If they are both disabled, the Console plugin is not deployed.
	//+kubebuilder:default:=true
	Enable *bool `json:"enable,omitempty"`

	// `mode` must be set according to the type of Prometheus installation that stores NetObserv metrics:<br>
	// - Use `Auto` to try configuring automatically for known vendors.<br>
	// - Use `Manual` for a manual setup.<br>
	//+unionDiscriminator
	//+kubebuilder:validation:Enum=Manual;Auto
	//+kubebuilder:default:="Auto"
	//+kubebuilder:validation:Required
	Mode PrometheusMode `json:"mode,omitempty"`

	// Prometheus configuration for `Manual` mode.
	// +optional
	Manual PrometheusQuerierManual `json:"manual,omitempty"`

	//+kubebuilder:default:="30s"
	// `timeout` is the read timeout for console plugin queries to Prometheus.
	// A timeout of zero means no timeout.
	Timeout *metav1.Duration `json:"timeout,omitempty"` // Warning: keep as pointer, else default is ignored
}

// FlowCollectorConsolePlugin defines the desired ConsolePlugin state of FlowCollector.
type FlowCollectorConsolePlugin struct {
	//+kubebuilder:default:=true
	// Enables the console plugin deployment.
	Enable *bool `json:"enable,omitempty"`

	// Deploy as a standalone console. Supported vendors may use a plugin system instead.
	Standalone *bool `json:"standalone,omitempty"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=1
	// `replicas` defines the number of replicas (pods) to start.
	Replicas *int32 `json:"replicas,omitempty"`

	// If `unmanagedReplicas` is `true`, the operator will not reconcile `replicas`. This is useful when using a pod autoscaler.
	// +optional
	UnmanagedReplicas bool `json:"unmanagedReplicas,omitempty"`

	//+kubebuilder:validation:Enum=IfNotPresent;Always;Never
	//+kubebuilder:default:=IfNotPresent
	// `imagePullPolicy` is the Kubernetes pull policy for the image defined above.
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	//+kubebuilder:default:={requests:{memory:"50Mi",cpu:"100m"},limits:{memory:"100Mi"}}
	// `resources`, in terms of compute resources, required by this container.
	// For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`

	//+kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal;panic
	//+kubebuilder:default:=info
	// `logLevel` for the web console backend.
	LogLevel string `json:"logLevel,omitempty"`

	// `autoscaler`: spec of a horizontal pod autoscaler to set up for the web console Deployment.
	//
	// Deprecated: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.consolePlugin.unmanagedReplicas` to `true`.
	//
	// +optional
	Autoscaler FlowCollectorHPA `json:"autoscaler,omitempty"`

	//+kubebuilder:default:={enable:true}
	// `portNaming` defines the configuration of the port-to-service name translation.
	PortNaming ConsolePluginPortConfig `json:"portNaming,omitempty"`

	//+kubebuilder:default:={{name:"Applications",filter:{"flow_layer":"\"app\""},default:true},{name:"Infrastructure",filter:{"flow_layer":"\"infra\""}},{name:"Pods network",filter:{"src_kind":"\"Pod\"","dst_kind":"\"Pod\""},default:true},{name:"Services network",filter:{"dst_kind":"\"Service\""}},{name:"External ingress",filter:{"src_subnet_label":"\"\",EXT:"}},{name:"External egress",filter:{"dst_subnet_label":"\"\",EXT:"}}}
	// +optional
	// `quickFilters` configures quick filter presets for the web console.
	// Filters for external traffic assume the subnet labels are configured to distinguish internal and external traffic (see `spec.processor.subnetLabels`).
	QuickFilters []QuickFilter `json:"quickFilters"`

	// `advanced` allows setting some aspects of the internal configuration of the console plugin.
	// This section is aimed mostly for debugging and fine-grained performance optimizations,
	// such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.
	// +optional
	Advanced *AdvancedPluginConfig `json:"advanced,omitempty"`
}

// Configuration of the port to service name translation feature of the console plugin
type ConsolePluginPortConfig struct {
	//+kubebuilder:default:=true
	// Enable the console plugin port-to-service name translation
	Enable *bool `json:"enable,omitempty"`

	// `portNames` defines additional port names to use in the console,
	// for example, `portNames: {"3100": "loki"}`.
	// +optional
	PortNames map[string]string `json:"portNames" yaml:"portNames"`
}

// `QuickFilter` defines preset configuration for Console's quick filters
type QuickFilter struct {
	// Name of the filter, that is displayed in the Console
	// +kubebuilder:MinLength:=1
	Name string `json:"name"`
	// `filter` is a set of keys and values to be set when this filter is selected. Each key can relate to a list of values using a coma-separated string,
	// for example, `filter: {"src_namespace": "namespace1,namespace2"}`.
	// +kubebuilder:MinProperties:=1
	Filter map[string]string `json:"filter"`
	// `default` defines whether this filter should be active by default or not
	// +optional
	Default bool `json:"default,omitempty"`
}

// `ClusterNetworkOperatorConfig` defines the desired configuration related to the Cluster Network Configuration
type ClusterNetworkOperatorConfig struct {
	// Namespace  where the config map is going to be deployed.
	Namespace string `json:"namespace,omitempty"`
}

// `OVNKubernetesConfig` defines the desired configuration related to the OVN-Kubernetes network plugin, when Cluster Network Operator isn't installed.
type OVNKubernetesConfig struct {
	//+kubebuilder:default:=ovn-kubernetes
	// Namespace where OVN-Kubernetes pods are deployed.
	Namespace string `json:"namespace,omitempty"`

	//+kubebuilder:default:=ovnkube-node
	// `daemonSetName` defines the name of the DaemonSet controlling the OVN-Kubernetes pods.
	DaemonSetName string `json:"daemonSetName,omitempty"`

	//+kubebuilder:default:=ovnkube-node
	// `containerName` defines the name of the container to configure for IPFIX.
	ContainerName string `json:"containerName,omitempty"`
}

type MountableType string

const (
	RefTypeSecret    MountableType = "secret"
	RefTypeConfigMap MountableType = "configmap"
)

type FileReference struct {
	//+kubebuilder:validation:Enum=configmap;secret
	// Type for the file reference: `configmap` or `secret`.
	Type MountableType `json:"type,omitempty"`

	// Name of the config map or secret containing the file.
	Name string `json:"name,omitempty"`

	// Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
	// If the namespace is different, the config map or the secret is copied so that it can be mounted as required.
	// +optional
	//+kubebuilder:default:=""
	Namespace string `json:"namespace,omitempty"`

	// File name within the config map or secret.
	File string `json:"file,omitempty"`
}

type CertificateReference struct {
	//+kubebuilder:validation:Enum=configmap;secret
	// Type for the certificate reference: `configmap` or `secret`.
	Type MountableType `json:"type,omitempty"`

	// Name of the config map or secret containing certificates.
	Name string `json:"name,omitempty"`

	// Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
	// If the namespace is different, the config map or the secret is copied so that it can be mounted as required.
	// +optional
	//+kubebuilder:default:=""
	Namespace string `json:"namespace,omitempty"`

	// `certFile` defines the path to the certificate file name within the config map or secret.
	CertFile string `json:"certFile,omitempty"`

	// `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.
	// +optional
	CertKey string `json:"certKey,omitempty"`
}

// `ClientTLS` defines TLS client configuration
type ClientTLS struct {
	//+kubebuilder:default:=false
	// Enable TLS
	Enable bool `json:"enable,omitempty"`

	//+kubebuilder:default:=false
	// `insecureSkipVerify` allows skipping client-side verification of the server certificate.
	// If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// `caCert` defines the reference of the certificate for the Certificate Authority.
	CACert CertificateReference `json:"caCert,omitempty"`

	// `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.
	// +optional
	UserCert CertificateReference `json:"userCert,omitempty"`
}

type SASLType string

const (
	SASLDisabled    SASLType = "Disabled"
	SASLPlain       SASLType = "Plain"
	SASLScramSHA512 SASLType = "ScramSHA512"
)

// `SASLConfig` defines SASL configuration
type SASLConfig struct {
	//+kubebuilder:validation:Enum=Disabled;Plain;ScramSHA512
	//+kubebuilder:default:=Disabled
	// Type of SASL authentication to use, or `Disabled` if SASL is not used
	Type SASLType `json:"type,omitempty"`

	// Reference to the secret or config map containing the client ID
	ClientIDReference FileReference `json:"clientIDReference,omitempty"`

	// Reference to the secret or config map containing the client secret
	ClientSecretReference FileReference `json:"clientSecretReference,omitempty"`
}

// `SchedulingConfig` defines the scheduling configuration for NetObserv pods.
type SchedulingConfig struct {
	// `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
	// For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.
	//+optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// `nodeSelector` allows scheduling of pods only onto nodes that have each of the specified labels.
	// For documentation, refer to https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.
	// +optional
	// +mapType=atomic
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// If specified, indicates the pod's priority. For documentation, refer to https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#how-to-use-priority-and-preemption.
	// If not specified, default priority is used, or zero if there is no default.
	// +optional
	PriorityClassName string `json:"priorityClassName,omitempty"`
}

// `AdvancedAgentConfig` allows tweaking some aspects of the internal configuration of the agent.
// They are aimed mostly for debugging. Set these values at your own risk.
type AdvancedAgentConfig struct {
	// `env` allows passing custom environment variables to underlying components. Useful for passing
	// some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
	// publicly exposed as part of the FlowCollector descriptor, as they are only useful
	// in edge debug or support scenarios.
	//+optional
	Env map[string]string `json:"env,omitempty"`

	// scheduling controls how the pods are scheduled on nodes.
	// +optional
	Scheduling *SchedulingConfig `json:"scheduling,omitempty"`

	// Linux capabilities override, when not running as privileged. Default capabilities are BPF, PERFMON and NET_ADMIN.
	// +optional
	CapOverride []string `json:"capOverride,omitempty"`
}

// `AdvancedProcessorConfig` allows tweaking some aspects of the internal configuration of the processor.
// They are aimed mostly for debugging. Set these values at your own risk.
type AdvancedProcessorConfig struct {
	// `env` allows passing custom environment variables to underlying components. Useful for passing
	// some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
	// publicly exposed as part of the FlowCollector descriptor, as they are only useful
	// in edge debug or support scenarios.
	//+optional
	Env map[string]string `json:"env,omitempty"`

	//+kubebuilder:validation:Minimum=1025
	//+kubebuilder:validation:Maximum=65535
	//+kubebuilder:default:=2055
	//+optional
	// Port of the flow collector (host port).
	// By convention, some values are forbidden. It must be greater than 1024 and different from
	// 4500, 4789 and 6081.
	Port *int32 `json:"port,omitempty"`

	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	//+kubebuilder:default:=8080
	//+optional
	// `healthPort` is a collector HTTP port in the Pod that exposes the health check API.
	HealthPort *int32 `json:"healthPort,omitempty"`

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=65535
	//+optional
	// `profilePort` allows setting up a Go pprof profiler listening to this port.
	// This is for debugging purpose only. This port should not be exposed, you can
	// access it through local port-forwarding.
	ProfilePort *int32 `json:"profilePort,omitempty"`

	//+kubebuilder:default:=true
	//+optional
	// `enableKubeProbes` is a flag to enable or disable Kubernetes liveness and readiness probes.
	EnableKubeProbes *bool `json:"enableKubeProbes,omitempty"`

	//+kubebuilder:default:=true
	//+optional
	// `dropUnusedFields`.
	//
	// Deprecated: this setting is not used anymore.
	DropUnusedFields *bool `json:"dropUnusedFields,omitempty"`

	//+kubebuilder:default:="30s"
	//+optional
	// `conversationHeartbeatInterval` is the time to wait between "tick" events of a conversation.
	ConversationHeartbeatInterval *metav1.Duration `json:"conversationHeartbeatInterval,omitempty"`

	//+kubebuilder:default:="10s"
	//+optional
	// `conversationEndTimeout` is the time to wait after a network flow is received, to consider the conversation ended.
	// This delay is ignored when a FIN packet is collected for TCP flows (see `conversationTerminatingTimeout` instead).
	ConversationEndTimeout *metav1.Duration `json:"conversationEndTimeout,omitempty"`

	//+kubebuilder:default:="5s"
	//+optional
	// `conversationTerminatingTimeout` is the time to wait from detected FIN flag to end a conversation. Only relevant for TCP flows.
	ConversationTerminatingTimeout *metav1.Duration `json:"conversationTerminatingTimeout,omitempty"`

	// `scheduling` controls how the pods are scheduled on nodes.
	// +optional
	Scheduling *SchedulingConfig `json:"scheduling,omitempty"`

	// Defines secondary networks to be checked for resources identification.
	// To guarantee a correct identification, indexed values must form an unique identifier across the cluster.
	// If the same index is used by several resources, those resources might be incorrectly labeled.
	// If not provided and `spec.agent.ebpf.privileged` is `true`, secondary networks are detected automatically.
	// +optional
	SecondaryNetworks []SecondaryNetwork `json:"secondaryNetworks,omitempty"`
}

// Field to index for secondary network pod identification, can be any of: `MAC`, `IP`, `Interface`.
// +kubebuilder:validation:Enum:="MAC";"IP";"Interface"
type SecondaryNetworkIndex string

const (
	SecondaryNetworkIndexByMAC       SecondaryNetworkIndex = "MAC"
	SecondaryNetworkIndexByIP        SecondaryNetworkIndex = "IP"
	SecondaryNetworkIndexByInterface SecondaryNetworkIndex = "Interface"
)

type SecondaryNetwork struct {
	// Deprecated: `name` is unused.
	// +optional
	Name string `json:"name,omitempty"`

	// `index` is a list of fields to use for indexing the pods. They should form a unique Pod identifier across the cluster.
	// Can be any of: `MAC`, `IP`, `Interface`.
	// Fields absent from the 'k8s.v1.cni.cncf.io/network-status' annotation must not be added to the index.
	// +kubebuilder:validation:Required
	Index []SecondaryNetworkIndex `json:"index,omitempty"`
}

// `AdvancedLokiConfig` allows tweaking some aspects of the Loki clients.
// They are aimed mostly for debugging. Set these values at your own risk.
type AdvancedLokiConfig struct {
	//+kubebuilder:default:="1s"
	//+optional
	// `writeMinBackoff` is the initial backoff time for Loki client connection between retries.
	WriteMinBackoff *metav1.Duration `json:"writeMinBackoff,omitempty"` // Warning: keep as pointer, else default is ignored

	//+kubebuilder:default:="5s"
	//+optional
	// `writeMaxBackoff` is the maximum backoff time for Loki client connection between retries.
	WriteMaxBackoff *metav1.Duration `json:"writeMaxBackoff,omitempty"` // Warning: keep as pointer, else default is ignored

	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default:=2
	//+optional
	// `writeMaxRetries` is the maximum number of retries for Loki client connections.
	WriteMaxRetries *int32 `json:"writeMaxRetries,omitempty"`

	//+kubebuilder:default:={"app":"netobserv-flowcollector"}
	//+optional
	// `staticLabels` is a map of common labels to set on each flow in Loki storage.
	StaticLabels map[string]string `json:"staticLabels,omitempty"`

	//+optional
	// `excludeLabels` is a list of fields to be excluded from the list of Loki labels.
	ExcludeLabels []string `json:"excludeLabels,omitempty"`
}

// `AdvancedPluginConfig` allows tweaking some aspects of the internal configuration of the web console.
// They are aimed mostly for debugging. Set these values at your own risk.
type AdvancedPluginConfig struct {
	// `env` allows passing custom environment variables to underlying components. Useful for passing
	// some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
	// publicly exposed as part of the FlowCollector descriptor, as they are only useful
	// in edge debug or support scenarios.
	//+optional
	Env map[string]string `json:"env,omitempty"`

	// `args` allows passing custom arguments to underlying components. Useful for overriding
	// some parameters, such as a URL or a configuration path, that should not be
	// publicly exposed as part of the FlowCollector descriptor, as they are only useful
	// in edge debug or support scenarios.
	//+optional
	Args []string `json:"args,omitempty"`

	//+kubebuilder:default:=true
	//+optional
	// `register` allows, when set to `true`, to automatically register the console plugin when possible, depending on the vendor.
	// It requires `spec.consolePlugin.standalone` to be `false`.
	Register *bool `json:"register,omitempty"`

	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=65535
	//+kubebuilder:default:=9001
	//+optional
	// `port` is the plugin service port. Do not use 9002, which is reserved for metrics.
	Port *int32 `json:"port,omitempty"`

	// `scheduling` controls how the pods are scheduled on nodes.
	// +optional
	Scheduling *SchedulingConfig `json:"scheduling,omitempty"`
}

// `SubnetLabels` allows you to define custom labels on subnets and IPs and, for supported vendors, to enable automatic labeling of recognized subnets, which is used to identify cluster external traffic.
type SubnetLabels struct {
	// `openShiftAutoDetect` allows, when set to `true`, to detect automatically the machines, pods and services subnets based on
	// vendor-specific configuration. Indirectly, this is a way to accurately detect
	// external traffic: flows that are not labeled for those subnets are external to the cluster.
	//+optional
	//
	// Deprecated: use `autoDetect` instead.
	OpenShiftAutoDetect *bool `json:"openShiftAutoDetect,omitempty"`

	// `autoDetect` allows, when set to `true`, to detect automatically the machines, pods and services subnets based on
	// vendor-specific configuration. It requires a vendor-specific implementation. Indirectly, this is a way to accurately detect
	// external traffic: flows that are not labeled for those subnets are external to the cluster. Enabled by default.
	//+optional
	AutoDetect *bool `json:"autoDetect,omitempty"`

	// `customLabels` allows you to customize subnets and IPs labeling, such as to identify cluster external workloads or web services.
	// External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/>
	// If `autoDetect` is disabled or your Kubernetes vendor has no auto-detection implemented, it is recommended to manually configure labels for the cluster subnets, to distinguish internal traffic from external traffic.<br/>
	// If `autoDetect` is enabled, `customLabels` overrides the detected subnets when they overlap.<br/>
	//+optional
	CustomLabels []SubnetLabel `json:"customLabels,omitempty"`
}

// SubnetLabel allows to label subnets and IPs, such as to identify cluster-external workloads or web services.
type SubnetLabel struct {
	// List of CIDRs, such as `["1.2.3.4/32"]`.
	//+required
	CIDRs []string `json:"cidrs,omitempty"` // Note, starting with k8s 1.31 / ocp 4.16 there's a new way to validate CIDR such as `+kubebuilder:validation:XValidation:rule="isCIDR(self)",message="field should be in CIDR notation format"`. But older versions would reject the CRD so we cannot implement it now to maintain compatibility.

	// Label name, used to flag matching flows.
	// External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/>
	// +kubebuilder:validation:Pattern:="^[a-zA-Z_:-][a-zA-Z0-9_:-]*$"
	//+required
	Name string `json:"name,omitempty"`
}

type ProcessorServiceConfig struct {
	// Select the type of TLS configuration:<br>
	// - `Disabled` to not configure TLS for the endpoint. Disabling TLS results in a less secure deployment model.<br>
	// - `Provided` to manually provide the key and certificate references.<br>
	// - `Auto` (default) to enable automatically based on the running environment.<br>
	// - `Auto-mTLS` to preconfigure mTLS.<br>
	// See also: https://github.com/netobserv/netobserv-operator/blob/main/docs/TLS.md.
	// +kubebuilder:validation:Enum:="Disabled";"Provided";"Auto";"Auto-mTLS"
	// +kubebuilder:validation:Required
	// +kubebuilder:default:="Auto"
	TLSType TLSConfigType `json:"tlsType,omitempty"`

	// TLS or mTLS configuration when `type` is set to `Provided`.
	// +optional
	ProvidedCertificates *ClientServerTLS `json:"providedCertificates,omitempty"`
}

// `InformerCacheProxyTLSConfig` defines the TLS configuration for gRPC communication between the informer cache proxy and processors
type InformerCacheProxyTLSConfig struct {
	// Select the type of TLS configuration:<br>
	// - `Disabled` to not configure TLS for the k8scache endpoint. Disabling TLS results in a less secure deployment model.<br>
	// - `Provided` to manually provide cert/key references for mTLS.<br>
	// - `Auto` (default) to use a default certificate, which may vary depending on the Kubernetes vendor.<br>
	// - `Auto-mTLS` to preconfigure mTLS with cert-manager.<br>
	// See also: https://github.com/netobserv/netobserv-operator/blob/main/docs/TLS.md.
	// +kubebuilder:validation:Enum:="Disabled";"Provided";"Auto";"Auto-mTLS"
	// +kubebuilder:validation:Required
	// +kubebuilder:default:="Auto"
	Type TLSConfigType `json:"type,omitempty"`

	// mTLS configuration when `type` is set to `Provided`.
	// `serverCert` is required. `clientCert` is optional; if provided, mTLS is enabled.
	// +optional
	ProvidedCertificates *ClientServerTLS `json:"providedCertificates,omitempty"`
}

// Add more exporter types below
type ExporterType string

const (
	KafkaExporter         ExporterType = "Kafka"
	IpfixExporter         ExporterType = "IPFIX"
	OpenTelemetryExporter ExporterType = "OpenTelemetry"
)

// `FlowCollectorExporter` defines an additional exporter to send enriched flows to.
type FlowCollectorExporter struct {
	// `type` selects the type of exporters. The available options are `Kafka`, `IPFIX`, and `OpenTelemetry`.
	// +unionDiscriminator
	// +kubebuilder:validation:Enum:="Kafka";"IPFIX";"OpenTelemetry"
	// +kubebuilder:validation:Required
	Type ExporterType `json:"type"`

	// Kafka configuration, such as the address and topic, to send enriched flows to.
	// +optional
	Kafka FlowCollectorKafka `json:"kafka,omitempty"`

	// IPFIX configuration, such as the IP address and port to send enriched IPFIX flows to.
	// +optional
	IPFIX FlowCollectorIPFIXReceiver `json:"ipfix,omitempty"`

	// OpenTelemetry configuration, such as the IP address and port to send enriched logs or metrics to.
	// +optional
	OpenTelemetry FlowCollectorOpenTelemetry `json:"openTelemetry,omitempty"`
}

type ExecutionMode string

const (
	Running ExecutionMode = "Running"
	OnHold  ExecutionMode = "OnHold"
)

// `FlowCollectorExecution` defines the flow collection process execution desired state.
type FlowCollectorExecution struct {
	// `mode` is the flow collection process execution desired mode: `Running` or `OnHold`.
	// When `OnHold`, the operator deletes all managed services and workloads, with the exception
	// of the static console plugin, and the operator itself.
	// It allows to use minimal cluster resources without losing configuration.
	// +kubebuilder:validation:Enum:="";"Running";"OnHold"
	// +kubebuilder:default:=Running
	// +optional
	Mode ExecutionMode `json:"mode"`
}

// `FlowCollectorComponentStatus` represents the status of a single operator component.
type FlowCollectorComponentStatus struct {
	// `state` reports the overall health of the component.
	// +kubebuilder:validation:Enum:="Ready";"InProgress";"Failure";"Degraded";"Unknown";"Unused"
	State string `json:"state"`

	// `reason` is a one-word CamelCase reason for the component's current state.
	// +optional
	Reason string `json:"reason,omitempty"`

	// `message` is a human-readable description of the component's current state.
	// +optional
	Message string `json:"message,omitempty"`

	// `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).
	// +optional
	DesiredReplicas *int32 `json:"desiredReplicas,omitempty"`

	// `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).
	// +optional
	ReadyReplicas *int32 `json:"readyReplicas,omitempty"`

	// `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).
	// +optional
	UnhealthyPodCount int32 `json:"unhealthyPodCount,omitempty"`

	// `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").
	// +optional
	PodIssues string `json:"podIssues,omitempty"`
}

// `FlowCollectorExporterStatus` represents the status of a configured exporter.
type FlowCollectorExporterStatus struct {
	// `name` is a generated identifier for this exporter (e.g., "kafka-export-0"), derived from its type and position in spec.exporters.
	Name string `json:"name"`

	// `type` is the exporter type (Kafka, IPFIX, OpenTelemetry).
	// +kubebuilder:validation:Enum:="Kafka";"IPFIX";"OpenTelemetry"
	Type string `json:"type"`

	// `state` reports the health of this exporter.
	// +kubebuilder:validation:Enum:="Ready";"InProgress";"Failure";"Degraded";"Unknown"
	State string `json:"state"`

	// `reason` is a one-word CamelCase reason for the exporter's current state.
	// +optional
	Reason string `json:"reason,omitempty"`

	// `message` is a human-readable description of the exporter's current state.
	// +optional
	Message string `json:"message,omitempty"`
}

// `FlowCollectorComponentsStatus` groups the status of operator-managed components.
type FlowCollectorComponentsStatus struct {
	// `agent` reports the status of the eBPF agent component.
	// +optional
	Agent *FlowCollectorComponentStatus `json:"agent,omitempty"`

	// `processor` reports the status of the flowlogs-pipeline component.
	// +optional
	Processor *FlowCollectorComponentStatus `json:"processor,omitempty"`

	// `plugin` reports the status of the console plugin component.
	// +optional
	Plugin *FlowCollectorComponentStatus `json:"plugin,omitempty"`
}

// `FlowCollectorIntegrationsStatus` groups the status of external integrations.
type FlowCollectorIntegrationsStatus struct {
	// `loki` reports the status of the Loki integration.
	// +optional
	Loki *FlowCollectorComponentStatus `json:"loki,omitempty"`

	// `monitoring` reports the status of monitoring (dashboards, ServiceMonitor, etc.).
	// +optional
	Monitoring *FlowCollectorComponentStatus `json:"monitoring,omitempty"`

	// `exporters` reports the status of configured exporters.
	// +optional
	Exporters []FlowCollectorExporterStatus `json:"exporters,omitempty"`
}

// `FlowCollectorStatus` defines the observed state of FlowCollector
type FlowCollectorStatus struct {
	// `conditions` represents the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`

	// `components` reports the status of operator-managed components (agent, processor, plugin).
	// +optional
	Components *FlowCollectorComponentsStatus `json:"components,omitempty"`

	// `integrations` reports the status of external integrations (Loki, monitoring, exporters).
	// +optional
	Integrations *FlowCollectorIntegrationsStatus `json:"integrations,omitempty"`

	// Namespace where console plugin and flowlogs-pipeline have been deployed.
	//
	// Deprecated: annotations are used instead
	Namespace string `json:"namespace,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Agent",type="string",JSONPath=`.status.components.agent.state`
// +kubebuilder:printcolumn:name="Processor",type="string",JSONPath=`.status.components.processor.state`
// +kubebuilder:printcolumn:name="Web Console",type="string",JSONPath=`.status.components.plugin.state`
// +kubebuilder:printcolumn:name="Sampling",type="string",JSONPath=`.spec.agent.ebpf.sampling`
// +kubebuilder:printcolumn:name="Deployment Model",type="string",JSONPath=`.spec.deploymentModel`
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Warnings",type="string",JSONPath=`.status.conditions[?(@.type=="ConfigurationIssue")].message`
// +kubebuilder:storageversion
// `FlowCollector` is the schema for the network flows collection API, which pilots and configures the underlying deployments.
type FlowCollector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FlowCollectorSpec   `json:"spec,omitempty"`
	Status FlowCollectorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// `FlowCollectorList` contains a list of FlowCollector
type FlowCollectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowCollector `json:"items"`
}
