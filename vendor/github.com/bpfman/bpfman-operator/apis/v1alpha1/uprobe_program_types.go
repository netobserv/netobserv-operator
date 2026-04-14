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

type UprobeProgramInfo struct {
	// links is an optional field and is the list of attachment points to which the
	// UProbe or URetProbe program should be attached. The eBPF program is loaded
	// in kernel memory when the BPF Application CRD is created and the selected
	// Kubernetes nodes are active. The eBPF program will not be triggered until
	// the program has also been attached to an attachment point described in this
	// list. Items may be added or removed from the list at any point, causing the
	// eBPF program to be attached or detached.
	//
	// The attachment point for a UProbe and URetProbe program is a user-space
	// binary or function. By default, the eBPF program is triggered at the entry
	// of the attachment point, but the attachment point can be adjusted using an
	// optional function name and/or offset. Optionally, the eBPF program can be
	// installed in a set of containers or limited to a specified PID.
	// +optional
	Links []UprobeAttachInfo `json:"links,omitempty"`
}

type UprobeAttachInfo struct {
	// function is an optional field and specifies the name of a user-space function
	// to attach the UProbe or URetProbe program. If not provided, the eBPF program
	// will be triggered on the entry of the target. function must not be an empty
	// string, must not exceed 64 characters in length, must start with alpha
	// characters and must only contain alphanumeric characters.
	// +optional
	// +kubebuilder:validation:Pattern="^[a-zA-Z][a-zA-Z0-9_]+."
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=64
	Function string `json:"function,omitempty"`

	// offset is an optional field and the value is added to the address of the
	// attachment point function. If not provided, offset defaults to 0.
	// +optional
	// +kubebuilder:default:=0
	Offset int64 `json:"offset"`

	// target is a required field and is the user-space library name or the
	// absolute path to a binary or library.
	// +required
	Target string `json:"target"`

	// pid is an optional field and if provided, limits the execution of the UProbe
	// or URetProbe to the provided process identification number (PID). If pid is
	// not provided, the UProbe or URetProbe executes for all PIDs.
	// +optional
	Pid *int32 `json:"pid,omitempty"`

	// containers is an optional field that identifies the set of containers in
	// which to attach the UProbe or URetProbe program. If containers is not
	// specified, the eBPF program will be attached in the bpfman container.
	// uprobe.
	Containers ContainerSelector `json:"containers"`
}

type UprobeProgramInfoState struct {
	// links is a list of attachment points for the UProbe program. Each entry in
	// the list includes a linkStatus, which indicates if the attachment was
	// successful or not on this node, a linkId, which is the kernel ID for the
	// link if successfully attached, and other attachment specific data.
	// +optional
	Links []UprobeAttachInfoState `json:"links,omitempty"`
}

type UprobeAttachInfoState struct {
	AttachInfoStateCommon `json:",inline"`

	// function is the provisioned name of the user-space function the UProbe
	// program should be attached.
	// +optional
	Function string `json:"function,omitempty"`

	// offset is the provisioned offset, whose value is added to the address of the
	// attachment point function.
	// +optional
	// +kubebuilder:default:=0
	Offset int64 `json:"offset"`

	// target is the provisioned user-space library name or the absolute path to a
	// binary or library.
	// +required
	Target string `json:"target"`

	// pid is the provisioned pid. If set, pid limits the execution of the UProbe
	// or URetProbe to the provided process identification number (PID). If pid is
	// not provided, the UProbe or URetProbe executes for all PIDs.
	// +optional
	Pid *int32 `json:"pid,omitempty"`

	// If containers is provisioned in the BpfApplication instance, containerPid is
	// the derived PID of the container the UProbe or URetProbe this attachment
	// point is attached.
	// +optional
	ContainerPid int32 `json:"containerPid,omitempty"`
}
