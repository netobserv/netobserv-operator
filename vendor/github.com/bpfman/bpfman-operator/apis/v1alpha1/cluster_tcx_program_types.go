/*
Copyright 2024 The bpfman Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// All fields are required unless explicitly marked optional
package v1alpha1

// ClTcxProgramInfo defines the tcx program details
type ClTcxProgramInfo struct {
	// links is an optional field and is the list of attachment points to which the
	// TCX program should be attached. The TCX program is loaded in kernel memory
	// when the BPF Application CRD is created and the selected Kubernetes nodes
	// are active. The TCX program will not be triggered until the program has also
	// been attached to an attachment point described in this list. Items may be
	// added or removed from the list at any point, causing the TCX program to be
	// attached or detached.
	//
	// The attachment point for a TCX program is a network interface (or device).
	// The interface can be specified by name, by allowing bpfman to discover each
	// interface, or by setting the primaryNodeInterface flag, which instructs
	// bpfman to use the primary interface of a Kubernetes node. Optionally, the
	// TCX program can also be installed into a set of network namespaces.
	// +optional
	Links []ClTcxAttachInfo `json:"links,omitempty"`
}

type ClTcxAttachInfo struct {
	// interfaceSelector is a required field and is used to determine the network
	// interface (or interfaces) the TCX program is attached. Interface list is set
	// by providing a list of interface names, enabling auto discovery, or setting
	// the primaryNodeInterface flag, but only one option is allowed.
	// +required
	InterfaceSelector InterfaceSelector `json:"interfaceSelector"`

	// networkNamespaces is an optional field that identifies the set of network
	// namespaces in which to attach the eBPF program. If networkNamespaces is not
	// specified, the eBPF program will be attached in the root network namespace.
	// +optional
	NetworkNamespaces *ClNetworkNamespaceSelector `json:"networkNamespaces,omitempty"`

	// direction is a required field and specifies the direction of traffic.
	// Allowed values are:
	//    Ingress, Egress
	//
	// When set to Ingress, the TC program is triggered when packets are received
	// by the interface.
	//
	// When set to Egress, the TC program is triggered when packets are to be
	// transmitted by the interface.
	// +required
	// +kubebuilder:validation:Enum=Ingress;Egress
	Direction TCDirectionType `json:"direction"`

	// priority is an optional field and determines the execution order of the TCX
	// program relative to other TCX programs attached to the same attachment
	// point. It must be a value between 0 and 1000, where lower values indicate
	// higher precedence. For TCX programs on the same attachment point with the
	// same direction and priority, the most recently attached program has a lower
	// precedence. If not provided, priority will default to 1000.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	Priority *int32 `json:"priority,omitempty"`
}

type ClTcxProgramInfoState struct {
	// links is a list of attachment points for the TCX program. Each entry in the
	// list includes a linkStatus, which indicates if the attachment was successful
	// or not on this node, a linkId, which is the kernel ID for the link if
	// successfully attached, and other attachment specific data.
	// +optional
	Links []ClTcxAttachInfoState `json:"links,omitempty"`
}

type ClTcxAttachInfoState struct {
	AttachInfoStateCommon `json:",inline"`

	// interfaceName is the name of the interface the TCX program should be
	// attached.
	// +required
	InterfaceName string `json:"interfaceName"`

	// netnsPath is the optional path to the network namespace inside of which the
	// TCX program should be attached.
	// +optional
	NetnsPath string `json:"netnsPath,omitempty"`

	// direction is the provisioned direction of traffic, Ingress or Egress, the TC
	// program should be attached for a given network device.
	// +required
	// +kubebuilder:validation:Enum=Ingress;Egress
	Direction TCDirectionType `json:"direction"`

	// priority is the provisioned priority of the TCX program in relation to other
	// programs of the same type with the same attach point. It is a value from 0
	// to 1000, where lower values have higher precedence.
	// +required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	Priority int32 `json:"priority"`
}
