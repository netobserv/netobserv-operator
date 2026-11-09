package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type OutputType string

const (
	Flows   OutputType = "flows"
	Packets OutputType = "packets"
	Metrics OutputType = "metrics"
)

func (f Feature) Valid() bool {
	switch f {
	case PacketDrop, DNSTracking, FlowRTT, NetworkEvents, PacketTranslation,
		UDNMapping, IPSec, TLSTracking, SecondaryNetworks:
		return true
	case EbpfManager:
		// EbpfManager is supported by FlowCollector, not FlowCapture.
		return false
	default:
		return false
	}
}

// +kubebuilder:validation:Enum=PacketDrop;DNSTracking;FlowRTT;NetworkEvents;PacketTranslation;UDNMapping;IPSec;TLSTracking;SecondaryNetworks
type Feature string

const (
	PacketDrop        Feature = "PacketDrop"
	DNSTracking       Feature = "DNSTracking"
	FlowRTT           Feature = "FlowRTT"
	NetworkEvents     Feature = "NetworkEvents"
	PacketTranslation Feature = "PacketTranslation"
	EbpfManager       Feature = "EbpfManager"
	UDNMapping        Feature = "UDNMapping"
	IPSec             Feature = "IPSec"
	TLSTracking       Feature = "TLSTracking"
	SecondaryNetworks Feature = "SecondaryNetworks"
)

// FlowFilter is shared with the eBPF agent's FlowCollector filter vocabulary.
type FlowFilter struct {
	Direction            string `json:"direction,omitempty"`
	IPCIDR               string `json:"ip_cidr,omitempty"`
	Protocol             string `json:"protocol,omitempty"`
	SourcePort           int32  `json:"source_port,omitempty"`
	DestinationPort      int32  `json:"destination_port,omitempty"`
	Port                 int32  `json:"port,omitempty"`
	SourcePortRange      string `json:"source_port_range,omitempty"`
	SourcePorts          string `json:"source_ports,omitempty"`
	DestinationPortRange string `json:"destination_port_range,omitempty"`
	DestinationPorts     string `json:"destination_ports,omitempty"`
	PortRange            string `json:"port_range,omitempty"`
	Ports                string `json:"ports,omitempty"`
	ICMPType             int    `json:"icmp_type,omitempty"`
	ICMPCode             int    `json:"icmp_code,omitempty"`
	PeerIP               string `json:"peer_ip,omitempty"`
	PeerCIDR             string `json:"peer_cidr,omitempty"`
	Action               string `json:"action,omitempty"`
	TCPFlags             string `json:"tcp_flags,omitempty"`
	Drops                bool   `json:"drops,omitempty"`
	Sample               uint32 `json:"sample,omitempty"`
}

// TLSPlaintextSpec enables TLS plaintext capture through libssl uprobes.
// It is only valid for packet captures and requires additional host access.
type TLSPlaintextSpec struct {
	// MinBytes drops plaintext events shorter than this size. The default is 0.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MinBytes int32 `json:"minBytes,omitempty"`
	// PreviewBytes limits the plaintext preview attached to exported events. 0 means unlimited.
	// The default is 256 bytes.
	// +kubebuilder:validation:Minimum=0
	// +optional
	PreviewBytes int32 `json:"previewBytes,omitempty"`
	// ProcessAllowlist limits uprobes to these process names. An empty list hooks eligible
	// processes on each selected node.
	// +optional
	ProcessAllowlist []string `json:"processAllowlist,omitempty"`
}

type FlowCaptureSpec struct {
	// OutputType selects the data captured by the eBPF agents.
	// +kubebuilder:validation:Enum=flows;packets;metrics
	OutputType OutputType `json:"outputType"`
	// MaxTime limits the capture lifetime. The default is one hour.
	// +optional
	MaxTime *metav1.Duration `json:"maxTime,omitempty"`
	// MaxBytes limits flow and packet captures. It is ignored for metrics and defaults to unlimited.
	// +kubebuilder:validation:Minimum=0
	// +optional
	MaxBytes int64 `json:"maxBytes,omitempty"`
	// Sampling is the eBPF sampling interval. 1 captures every packet. The default is 1.
	// +kubebuilder:validation:Minimum=1
	// +optional
	Sampling int32 `json:"sampling,omitempty"`
	// +optional
	Interfaces []string `json:"interfaces,omitempty"`
	// +optional
	ExcludeInterfaces []string `json:"excludeInterfaces,omitempty"`
	// Features enables optional eBPF features. Values use the same vocabulary as
	// FlowCollector.spec.agent.ebpf.features.
	// +optional
	Features []Feature `json:"features,omitempty"`
	// Filters are applied by the eBPF agent.
	// +optional
	Filters []FlowFilter `json:"filters,omitempty"`
	// Queries are Flowlogs-Pipeline keep-entry expressions applied after enrichment.
	// They use the same syntax as the CLI --query option and are evaluated in order.
	// +optional
	Queries []string `json:"queries,omitempty"`
	// GetSubnets labels flows with the detected machine, pod, and service CIDRs.
	// This is currently supported on OpenShift clusters.
	// +optional
	GetSubnets bool `json:"getSubnets,omitempty"`
	// TLSPlaintext enables TLS plaintext capture for packet captures.
	// +optional
	TLSPlaintext *TLSPlaintextSpec `json:"tlsPlaintext,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

func (t OutputType) Valid() bool { return t == Flows || t == Packets || t == Metrics }

type FlowCaptureStatus struct {
	// Phase is the current lifecycle phase of the capture.
	// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
	// +optional
	Phase string `json:"phase,omitempty"`
	// Reason identifies the event that produced the current phase.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message contains additional human-readable status details.
	// +optional
	Message string `json:"message,omitempty"`
	// CompletionTime is set when the capture is marked complete; it does not guarantee that the collector Pod has exited.
	// For managed captures, the collector Pod and its output remain available until the FlowCapture is deleted.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`
	// Conditions records the lifecycle transitions of the capture in order,
	// including Starting, Running, Completed, or Failed.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// FlowCapture requests a short-lived capture of flows, packets, or metrics.
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=fcap
// +kubebuilder:subresource:status
type FlowCapture struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FlowCaptureSpec   `json:"spec,omitempty"`
	Status            FlowCaptureStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type FlowCaptureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FlowCapture `json:"items"`
}
