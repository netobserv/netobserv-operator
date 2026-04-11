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

type XdpProgramInfo struct {
	// links is an optional field and is the list of attachment points to which the
	// XDP program should be attached. The XDP program is loaded in kernel memory
	// when the BPF Application CRD is created and the selected Kubernetes nodes
	// are active. The XDP program will not be triggered until the program has also
	// been attached to an attachment point described in this list. Items may be
	// added or removed from the list at any point, causing the XDP program to be
	// attached or detached.
	//
	// The attachment point for a XDP program is a network interface (or device).
	// The interface can be specified by name, by allowing bpfman to discover each
	// interface, or by setting the primaryNodeInterface flag, which instructs
	// bpfman to use the primary interface of a Kubernetes node.
	// +optional
	Links []XdpAttachInfo `json:"links,omitempty"`
}

type XdpAttachInfo struct {
	// interfaceSelector is a required field and is used to determine the network
	// interface (or interfaces) the XDP program is attached. Interface list is set
	// by providing a list of interface names, enabling auto discovery, or setting
	// the primaryNodeInterface flag, but only one option is allowed.
	// +required
	InterfaceSelector InterfaceSelector `json:"interfaceSelector"`

	// networkNamespaces is a required field that identifies the set of network
	// namespaces in which to attach the eBPF program.
	// +required
	NetworkNamespaces NetworkNamespaceSelector `json:"networkNamespaces"`

	// priority is an optional field and determines the execution order of the XDP
	// program relative to other XDP programs attached to the same attachment
	// point. It must be a value between 0 and 1000, where lower values indicate
	// higher precedence. For XDP programs on the same attachment point with the
	// same priority, the most recently attached program has a lower precedence.
	// If not provided, priority will default to 1000.
	// +optional
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	Priority *int32 `json:"priority,omitempty"`

	// proceedOn is an optional field and allows the user to call other XDP
	// programs in a chain, or not call the next program in a chain based on the
	// exit code of an XDP program. Allowed values, which are the possible exit
	// codes from an XDP eBPF program, are:
	//   Aborted, Drop, Pass, TX, ReDirect,DispatcherReturn
	//
	// Multiple values are supported. Default is Pass and DispatcherReturn. So
	// using the default values, if an XDP program returns Pass, the next XDP
	// program in the chain will be called. If an XDP program returns Drop, the
	// next XDP program in the chain will NOT be called.
	// +optional
	// +kubebuilder:default:={Pass,DispatcherReturn}
	ProceedOn []XdpProceedOnValue `json:"proceedOn,omitempty"`
}

type XdpProgramInfoState struct {
	// links is a list of attachment points for the XDP program. Each entry in the
	// list includes a linkStatus, which indicates if the attachment was successful
	// or not on this node, a linkId, which is the kernel ID for the link if
	// successfully attached, and other attachment specific data.
	// +optional
	Links []XdpAttachInfoState `json:"links,omitempty"`
}

type XdpAttachInfoState struct {
	AttachInfoStateCommon `json:",inline"`

	// interfaceName is the name of the interface the XDP program should be
	// attached.
	// +required
	InterfaceName string `json:"interfaceName"`

	// netnsPath is the path to the network namespace inside of which the XDP
	// program should be attached.
	// +required
	NetnsPath string `json:"netnsPath"`

	// priority is the provisioned priority of the XDP program in relation to other
	// programs of the same type with the same attach point. It is a value from 0
	// to 1000, where lower values have higher precedence.
	// +required
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	Priority int32 `json:"priority"`

	// proceedOn is the provisioned list of proceedOn values. proceedOn allows the
	// user to call other TC programs in a chain, or not call the next program in a
	// chain based on the exit code of a TC program .Multiple values are supported.
	// +required
	ProceedOn []XdpProceedOnValue `json:"proceedOn"`
}
