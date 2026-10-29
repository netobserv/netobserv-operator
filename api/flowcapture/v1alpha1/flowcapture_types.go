package v1alpha1

import (
	sharedcapture "github.com/netobserv/netobserv-operator/pkg/capture"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CaptureType string

const (
	Flows   CaptureType = "flows"
	Packets CaptureType = "packets"
	Metrics CaptureType = "metrics"
)

// +kubebuilder:validation:Enum=PacketDrop;DNSTracking;FlowRTT;NetworkEvents;PacketTranslation;EbpfManager;UDNMapping;IPSec;TLSTracking
type Feature string

const (
	PacketDrop        Feature = Feature(sharedcapture.PacketDrop)
	DNSTracking       Feature = Feature(sharedcapture.DNSTracking)
	FlowRTT           Feature = Feature(sharedcapture.FlowRTT)
	NetworkEvents     Feature = Feature(sharedcapture.NetworkEvents)
	PacketTranslation Feature = Feature(sharedcapture.PacketTranslation)
	EbpfManager       Feature = Feature(sharedcapture.EbpfManager)
	UDNMapping        Feature = Feature(sharedcapture.UDNMapping)
	IPSec             Feature = Feature(sharedcapture.IPSec)
	TLSTracking       Feature = Feature(sharedcapture.TLSTracking)
)

type FlowCaptureSpec struct {
	// Type selects the data captured by the eBPF agents.
	// +kubebuilder:validation:Enum=flows;packets;metrics
	Type CaptureType `json:"type"`
	// MaxTime is passed to the collector and limits the capture lifetime.
	// +optional
	MaxTime *metav1.Duration `json:"maxTime,omitempty"`
	// MaxBytes limits flow and packet captures. It is ignored for metrics.
	// +optional
	MaxBytes int64 `json:"maxBytes,omitempty"`
	// Sampling is the eBPF sampling interval. 1 captures every flow.
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
	Filters []sharedcapture.FlowFilter `json:"filters,omitempty"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

func (t CaptureType) Valid() bool { return t == Flows || t == Packets || t == Metrics }

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
	// CompletionTime is set when the collector exits.
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
