package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=odc;odcs
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.captureType`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Items",type=integer,JSONPath=`.status.itemsCollected`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// OnDemandCapture is the Schema for the ondemandcaptures API
type OnDemandCapture struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OnDemandCaptureSpec   `json:"spec,omitempty"`
	Status OnDemandCaptureStatus `json:"status,omitempty"`
}

// +kubebuilder:validation:Enum=flows;packets;metrics
// CaptureType defines the type of capture to perform
type CaptureType string

const (
	CaptureTypeFlows   CaptureType = "flows"
	CaptureTypePackets CaptureType = "packets"
	CaptureTypeMetrics CaptureType = "metrics"
)

// +kubebuilder:validation:Enum=Pending;Running;Completed;Failed
// OnDemandCapturePhase defines the phase of an OnDemandCapture
type OnDemandCapturePhase string

const (
	OnDemandCapturePending   OnDemandCapturePhase = "Pending"
	OnDemandCaptureRunning   OnDemandCapturePhase = "Running"
	OnDemandCaptureCompleted OnDemandCapturePhase = "Completed"
	OnDemandCaptureFailed    OnDemandCapturePhase = "Failed"
)

// OnDemandCaptureSpec defines the desired state of OnDemandCapture
type OnDemandCaptureSpec struct {
	// +optional
	// args contains additional native CLI capture arguments, passed without a shell.
	// They follow structured settings and may override them. Connection, output,
	// and lifecycle options are managed by the operator and cannot be supplied.
	Args []string `json:"args,omitempty"`

	// +kubebuilder:validation:Required
	// captureType is the type of capture to perform: flows, packets, or metrics
	CaptureType CaptureType `json:"captureType"`

	// +optional
	// interfaces is a list of network interfaces to capture from
	Interfaces []string `json:"interfaces,omitempty"`

	// +optional
	// filters allows filtering the captured traffic
	Filters *CaptureFilters `json:"filters,omitempty"`

	// +optional
	// +kubebuilder:validation:Type=string
	// +kubebuilder:validation:Pattern="^([0-9]+h)?([0-9]+m)?([0-9]+s)?$"
	// duration is how long to capture (e.g. "5m", "1h30m")
	Duration *metav1.Duration `json:"duration,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// maxBytes is the maximum number of bytes to capture
	MaxBytes *int64 `json:"maxBytes,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=0
	// ttlSecondsAfterFinished is the retention period after completion or failure.
	// Defaults to 86400 (24 hours) when omitted. Zero requests immediate cleanup.
	// Expiry deletes the CR and its resources, including undownloaded capture files.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// +optional
	// flowConfig is configuration specific to flow capture
	FlowConfig *FlowCaptureConfig `json:"flowConfig,omitempty"`

	// +optional
	// packetConfig is configuration specific to packet capture
	PacketConfig *PacketCaptureConfig `json:"packetConfig,omitempty"`

	// +optional
	// metricsConfig is configuration specific to metrics capture
	MetricsConfig *MetricsCaptureConfig `json:"metricsConfig,omitempty"`

	// +optional
	// resources specifies the CPU and memory limits for the capture pod
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +optional
	// nodeSelector restricts which nodes the eBPF agents are deployed to.
	// When omitted, agents are deployed to all nodes.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

// CaptureFilters allows filtering captured traffic
type CaptureFilters struct {
	CaptureFilterRule `json:",inline"`

	// +optional
	// or adds alternative filter groups, equivalent to the CLI or separator.
	Or []CaptureFilterRule `json:"or,omitempty"`

	// +optional
	// namespaces is a list of namespaces to capture traffic from
	Namespaces []string `json:"namespaces,omitempty"`

	// +optional
	// podLabelSelector filters pods by label selectors
	PodLabelSelector map[string]string `json:"podLabelSelector,omitempty"`
}

// CaptureFilterRule uses the same filter names and values as the native CLI.
type CaptureFilterRule struct {
	// +optional
	// Ingress or Egress direction.
	Direction string `json:"direction,omitempty"`
	// +optional
	// IP CIDR filter.
	CIDR string `json:"cidr,omitempty"`
	// +optional
	// TCP, UDP, SCTP, ICMP, or ICMPv6.
	Protocol string `json:"protocol,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// Source port.
	SPort *int32 `json:"sport,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// Destination port.
	DPort *int32 `json:"dport,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=65535
	// Source or destination port.
	Port *int32 `json:"port,omitempty"`
	// +optional
	// Source port range in CLI syntax.
	SPortRange string `json:"sport_range,omitempty"`
	// +optional
	// Destination port range in CLI syntax.
	DPortRange string `json:"dport_range,omitempty"`
	// +optional
	// Source or destination port range in CLI syntax.
	PortRange string `json:"port_range,omitempty"`
	// +optional
	// Comma-separated source ports.
	Sports string `json:"sports,omitempty"`
	// +optional
	// Comma-separated destination ports.
	Dports string `json:"dports,omitempty"`
	// +optional
	// Comma-separated source or destination ports.
	Ports string `json:"ports,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=255
	// ICMP type, including zero.
	ICMPType *int32 `json:"icmp_type,omitempty"`
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=255
	// ICMP code, including zero.
	ICMPCode *int32 `json:"icmp_code,omitempty"`
	// +optional
	// Peer IP address.
	PeerIP string `json:"peer_ip,omitempty"`
	// +optional
	// Peer CIDR filter.
	PeerCIDR string `json:"peer_cidr,omitempty"`
	// +optional
	// Accept or Reject action.
	Action string `json:"action,omitempty"`
	// +optional
	// TCP flags in CLI syntax.
	TCPFlags string `json:"tcp_flags,omitempty"`
	// +optional
	// Capture only dropped traffic and enable packet-drop tracking.
	Drops bool `json:"drops,omitempty"`
}

// FlowCaptureConfig is configuration for flow capture
type FlowCaptureConfig struct {
	// +optional
	// enableDNS enables DNS latency tracking
	EnableDNS bool `json:"enableDNS,omitempty"`

	// +optional
	// enableRTT enables TCP round-trip time tracking
	EnableRTT bool `json:"enableRTT,omitempty"`

	// +optional
	// enablePacketDrops enables packet drop tracking
	EnablePacketDrops bool `json:"enablePacketDrops,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// sampling is the sampling rate (e.g. 50 means 1 in 50 packets)
	Sampling int32 `json:"sampling,omitempty"`
}

// PacketCaptureConfig is configuration for packet capture
type PacketCaptureConfig struct {
	// +optional
	// protocol filters packets by protocol (TCP, UDP, ICMP, etc.)
	Protocol string `json:"protocol,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// port filters packets by destination port
	Port int32 `json:"port,omitempty"`
}

// MetricsCaptureConfig is configuration for metrics capture
type MetricsCaptureConfig struct {
	// +optional
	// enablePacketDrops enables packet drop metrics collection
	EnablePacketDrops bool `json:"enablePacketDrops,omitempty"`

	// +optional
	// enableDNS enables DNS metrics collection
	EnableDNS bool `json:"enableDNS,omitempty"`

	// +optional
	// enableRTT enables latency metrics collection
	EnableRTT bool `json:"enableRTT,omitempty"`
}

// OnDemandCaptureStatus defines the observed state of OnDemandCapture
type OnDemandCaptureStatus struct {
	// +optional
	// phase represents the current phase of the capture
	Phase OnDemandCapturePhase `json:"phase,omitempty"`

	// +optional
	// startTime is when the capture started
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// +optional
	// completionTime is when the capture completed
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// +optional
	// itemsCollected is the number of items collected
	ItemsCollected int64 `json:"itemsCollected,omitempty"`

	// +optional
	// bytesCollected is the total number of bytes collected
	BytesCollected int64 `json:"bytesCollected,omitempty"`

	// +optional
	// podName is the name of the capture pod
	PodName string `json:"podName,omitempty"`

	// +optional
	// podNamespace is the namespace of the capture pod
	PodNamespace string `json:"podNamespace,omitempty"`

	// +optional
	// podIP is the IP address of the collector pod
	PodIP string `json:"podIP,omitempty"`

	// +optional
	// daemonSetName is the name of the temporary eBPF agent DaemonSet
	DaemonSetName string `json:"daemonSetName,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// fileServerPort is the port where the file server is running
	FileServerPort int32 `json:"fileServerPort,omitempty"`

	// +optional
	// conditions represent the latest available observations of the OnDemandCapture's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// +optional
	// error is a message describing any error that occurred
	Error string `json:"error,omitempty"`
}

// +kubebuilder:object:root=true
// OnDemandCaptureList contains a list of OnDemandCapture
type OnDemandCaptureList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OnDemandCapture `json:"items"`
}
