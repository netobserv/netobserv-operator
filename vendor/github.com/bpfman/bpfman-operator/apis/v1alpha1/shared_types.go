/*
Copyright 2022.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DefaultAttachPriority is the default priority used when attaching BPF
// programs (XDP, TC, TCX) to network interfaces. Priority determines execution
// order relative to other programs at the same attachment point, where lower
// values indicate higher precedence. Valid range is 0-1000.
const DefaultAttachPriority int32 = 1000

type InterfaceDiscovery struct {
	// interfaceAutoDiscovery is an optional field. When enabled, the agent
	// monitors the creation and deletion of interfaces and automatically
	// attached eBPF programs to the newly discovered interfaces.
	// CAUTION: This has the potential to attach a given eBPF program to a large
	// number of interfaces. Use with caution.
	// +optional
	// +kubebuilder:default:=false
	InterfaceAutoDiscovery *bool `json:"interfaceAutoDiscovery,omitempty"`

	// excludeInterfaces is an optional field that contains a list of interface
	// names that are excluded from interface discovery. The interface names in
	// the list are case-sensitive. By default, the list contains the loopback
	// interface, "lo". This field is only taken into consideration if
	// interfaceAutoDiscovery is set to true.
	// +optional
	// +kubebuilder:default:={"lo"}
	ExcludeInterfaces []string `json:"excludeInterfaces,omitempty"`

	// allowedInterfaces is an optional field that contains a list of interface
	// names that are allowed to be discovered. If empty, the agent will fetch all
	// the interfaces in the system, excepting the ones listed in
	// excludeInterfaces. if non-empty, only entries in the list will be considered
	// for discovery. If an entry enclosed by slashes, such as `/br-/` or
	// `/veth*/`, then the entry is considered as a regular expression for
	// matching. Otherwise, the interface names in the list are case-sensitive.
	// This field is only taken into consideration if interfaceAutoDiscovery is set
	// to true.
	// +optional
	AllowedInterfaces []string `json:"allowedInterfaces,omitempty"`
}

// InterfaceSelector describes the set of interfaces to attach a program to.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type InterfaceSelector struct {
	// interfacesDiscoveryConfig is an optional field that is used to control if
	// and how to automatically discover interfaces. If the agent should
	// automatically discover and attach eBPF programs to interfaces, use the
	// fields under interfacesDiscoveryConfig to control what is allow and excluded
	// from discovery.
	// +optional
	InterfacesDiscoveryConfig *InterfaceDiscovery `json:"interfacesDiscoveryConfig,omitempty"`

	// interfaces is an optional field and is a list of network interface names to
	// attach the eBPF program. The interface names in the list are case-sensitive.
	// +optional
	Interfaces []string `json:"interfaces,omitempty"`

	// primaryNodeInterface is and optional field and indicates to attach the eBPF
	// program to the primary interface on the Kubernetes node. Only 'true' is
	// accepted.
	// +optional
	PrimaryNodeInterface *bool `json:"primaryNodeInterface,omitempty"`
}

// ClContainerSelector identifies a set of containers.
type ClContainerSelector struct {
	// namespace is an optional field and indicates the target Kubernetes
	// namespace. If not provided, all Kubernetes namespaces are included.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// pods is a required field and indicates the target pods. To select all pods
	// use the standard metav1.LabelSelector semantics and make it empty.
	// +required
	Pods metav1.LabelSelector `json:"pods"`

	// containerNames is an optional field and is a list of container names in a
	// pod to attach the eBPF program. If no names are specified, all containers
	// in the pod are selected.
	// +optional
	ContainerNames []string `json:"containerNames,omitempty"`
}

// ContainerSelector identifies a set of containers. It is different from ClContainerSelector
// in that "Namespace" was removed. Namespace scoped programs can only attach to the namespace
// they are created in.
type ContainerSelector struct {
	// pods is a required field and indicates the target pods. To select all pods
	// use the standard metav1.LabelSelector semantics and make it empty.
	// +required
	Pods metav1.LabelSelector `json:"pods"`

	// containerNames is an optional field and is a list of container names in a
	// pod to attach the eBPF program. If no names are  specified, all containers
	// in the pod are selected.
	// +optional
	ContainerNames []string `json:"containerNames,omitempty"`
}

// ClNetworkNamespaceSelector identifies a network namespace for network-related
// program types in the cluster-scoped ClusterBpfApplication object.
type ClNetworkNamespaceSelector struct {
	// namespace is an optional field and indicates the target network namespace.
	// If not provided, the default network namespace is used.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// pods is a required field and indicates the target pods. To select all pods
	// use the standard metav1.LabelSelector semantics and make it empty.
	// +required
	Pods metav1.LabelSelector `json:"pods"`
}

// NetworkNamespaceSelector identifies a network namespace for network-related
// program types in the namespace-scoped BpfApplication object.
type NetworkNamespaceSelector struct {
	// pods is a required field and indicates the target pods. To select all pods
	// use the standard metav1.LabelSelector semantics and make it empty.
	// +required
	Pods metav1.LabelSelector `json:"pods"`
}

// BpfAppCommon defines the common attributes for all BpfApp programs
type BpfAppCommon struct {
	// nodeSelector is a required field and allows the user to specify which
	// Kubernetes nodes to deploy the eBPF programs. To select all nodes use
	// standard metav1.LabelSelector semantics and make it empty.
	// +required
	NodeSelector metav1.LabelSelector `json:"nodeSelector"`

	// globalData is an optional field that allows the user to set global variables
	// when the program is loaded. This allows the same compiled bytecode to be
	// deployed by different BPF Applications to behave differently based on
	// globalData configuration values.  It uses an array of raw bytes. This is a
	// very low level primitive. The caller is responsible for formatting the byte
	// string appropriately considering such things as size, endianness, alignment
	// and packing of data structures.
	// +optional
	GlobalData map[string][]byte `json:"globalData,omitempty"`

	// bytecode is a required field and configures where the eBPF program's
	// bytecode should be loaded from. The image must contain one or more
	// eBPF programs.
	// +required
	ByteCode ByteCodeSelector `json:"byteCode"`

	// mapOwnerSelector is an optional field used to share maps across
	// applications. eBPF programs loaded with the same ClusterBpfApplication or
	// BpfApplication instance do not need to use this field. This label selector
	// allows maps from a different ClusterBpfApplication or BpfApplication
	// instance to be used by this instance.
	// TODO: mapOwnerSelector is currently not supported due to recent code rework.
	// +optional
	MapOwnerSelector *metav1.LabelSelector `json:"mapOwnerSelector,omitempty"`
}

// status reflects the status of a BPF Application and indicates if all the
// eBPF programs for a given instance loaded successfully or not.
type BpfAppStatus struct {
	// conditions contains the summary state for all eBPF programs defined in the
	// BPF Application instance for all the Kubernetes nodes in the cluster.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// AttachInfoStateCommon reflects the status for one attach point for a given bpf
// application program
type AttachInfoStateCommon struct {
	// shouldAttach reflects whether the attachment should exist.
	// +required
	ShouldAttach bool `json:"shouldAttach"`
	// uuid is an Unique identifier for the attach point assigned by bpfman agent.
	// +required
	UUID string `json:"uuid"`
	// linkId is an identifier for the link assigned by bpfman. This field is
	// empty until the program is successfully attached and bpfman returns the
	// id.
	// +optional
	LinkId *uint32 `json:"linkId,omitempty"`
	// linkStatus reflects whether the attachment has been reconciled
	// successfully, and if not, why.
	// +required
	LinkStatus LinkStatus `json:"linkStatus"`
}

type BpfProgramStateCommon struct {
	// name is the name of the function that is the entry point for the eBPF
	// program
	// +required
	Name string `json:"name"`
	// programLinkStatus reflects whether all links requested for the program
	// are in the correct state.
	// +required
	ProgramLinkStatus ProgramLinkStatus `json:"programLinkStatus"`
	// programId is the id of the program in the kernel.  Not set until the
	// program is loaded.
	// +optional
	ProgramId *uint32 `json:"programId,omitempty"`
}

// PullPolicy describes a policy for if/when to pull a container image
// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
type PullPolicy string

const (
	// PullAlways means that bpfman always attempts to pull the latest bytecode image. Container will fail If the pull fails.
	PullAlways PullPolicy = "Always"
	// PullNever means that bpfman never pulls an image, but only uses a local image. Container will fail if the image isn't present
	PullNever PullPolicy = "Never"
	// PullIfNotPresent means that bpfman pulls if the image isn't present on disk. Container will fail if the image isn't present and the pull fails.
	PullIfNotPresent PullPolicy = "IfNotPresent"
)

// ByteCodeSelector defines the various ways to reference BPF bytecode objects.
// +kubebuilder:validation:MaxProperties=1
// +kubebuilder:validation:MinProperties=1
type ByteCodeSelector struct {
	// image is an optional field and used to specify details on how to retrieve an
	// eBPF program packaged in a OCI container image from a given registry.
	// +optional
	Image *ByteCodeImage `json:"image,omitempty"`

	// path is an optional field and used to specify a bytecode object file via
	// filepath on a Kubernetes node.
	// +optional
	// +kubebuilder:validation:Pattern=`^(/[^/\0]+)+/?$`
	Path *string `json:"path,omitempty"`
}

// ByteCodeImage defines how to specify a bytecode container image.
type ByteCodeImage struct {
	// url is a required field and is a valid container image URL used to reference
	// a remote bytecode image. url must not be an empty string, must not exceed
	// 525 characters in length and must be a valid URL.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength:=525
	// +kubebuilder:validation:Pattern=`[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}`
	Url string `json:"url"`

	// pullPolicy is an optional field that describes a policy for if/when to pull
	// a bytecode image. Defaults to IfNotPresent. Allowed values are:
	//   Always, IfNotPresent and Never
	//
	// When set to Always, the given image will be pulled even if the image is
	// already present on the node.
	//
	// When set to IfNotPresent, the given image will only be pulled if it is not
	// present on the node.
	//
	// When set to Never, the given image will never be pulled and must be
	// loaded on the node by some other means.
	// +optional
	// +kubebuilder:default:=IfNotPresent
	ImagePullPolicy PullPolicy `json:"imagePullPolicy,omitempty"`

	// imagePullSecret is an optional field and indicates the secret which contains
	// the credentials to access the image repository.
	// +optional
	ImagePullSecret *ImagePullSecretSelector `json:"imagePullSecret,omitempty"`
}

// ImagePullSecretSelector defines the name and namespace of an image pull secret.
type ImagePullSecretSelector struct {
	// name is a required field and is the name of the secret which contains the
	// credentials to access the image repository.
	// +required
	Name string `json:"name"`

	// namespace is a required field and is the namespace of the secret which
	// contains the credentials to access the image repository.
	// +required
	Namespace string `json:"namespace"`
}

// -----------------------------------------------------------------------------
// Status Conditions - BPF Programs
// -----------------------------------------------------------------------------

// BpfApplicationConditionType is a condition type to indicate the status of a BPF
// application at the cluster level.
type BpfApplicationConditionType string

const (
	// BpfAppCondPending indicates that bpfman has not yet completed reconciling
	// the Bpf Application on all nodes in the cluster.
	BpfAppCondPending BpfApplicationConditionType = "Pending"

	// BpfAppCondSuccess indicates that the BPF Application has been
	// successfully loaded and attached as requested on all nodes in the
	// cluster.
	BpfAppCondSuccess BpfApplicationConditionType = "Success"

	// BpfAppCondError indicates that an error has occurred on one or more nodes
	// while attempting to apply the configuration described in the CRD.
	BpfAppCondError BpfApplicationConditionType = "Error"

	// BpfAppCondDeleteError indicates that the BPF Application was marked for
	// deletion, but deletion was unsuccessful on one or more nodes.
	BpfAppCondDeleteError BpfApplicationConditionType = "DeleteError"
)

// Condition is a helper method to promote any given BpfApplicationConditionType
// to a full metav1.Condition in an opinionated fashion.
//
// TODO: this was created in the early days to provide at least SOME status
// information to the user, but the hardcoded messages need to be replaced in
// the future with dynamic and situation-aware messages later.
//
// See: https://github.com/bpfman/bpfman/issues/430
func (b BpfApplicationConditionType) Condition(message string) metav1.Condition {
	cond := metav1.Condition{}

	switch b {
	case BpfAppCondPending:
		if len(message) == 0 {
			message = "Waiting for Bpf Application Object to be reconciled on all nodes"
		}
		condType := string(BpfAppCondPending)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: message,
		}
	case BpfAppCondError:
		if len(message) == 0 {
			message = "An error has occurred on one or more nodes"
		}
		condType := string(BpfAppCondError)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Error",
			Message: message,
		}
	case BpfAppCondSuccess:
		if len(message) == 0 {
			message = "BPF application configuration successfully applied on all nodes"
		}
		condType := string(BpfAppCondSuccess)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: message,
		}
	case BpfAppCondDeleteError:
		if len(message) == 0 {
			message = "Deletion failed on one or more nodes"
		}
		condType := string(BpfAppCondDeleteError)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "DeleteError",
			Message: message,
		}
	}

	return cond
}

// BpfApplicationStateConditionType is used to indicate the status of a BPF
// application on a given node.
type BpfApplicationStateConditionType string

const (
	// BpfAppStateCondPending indicates that bpfman has not yet completed
	// reconciling the Bpf Application on the given node.
	BpfAppStateCondPending BpfApplicationStateConditionType = "Pending"

	// BpfAppStateCondSuccess indicates that the BPF Application has been
	// successfully loaded and attached as requested on the given node.
	BpfAppStateCondSuccess BpfApplicationStateConditionType = "Success"

	// BpfAppStateCondError indicates that an error has occurred on the given
	// node while attempting to apply the configuration described in the CRD.
	BpfAppStateCondError BpfApplicationStateConditionType = "Error"

	// BpfAppStateCondError indicates that an error has occurred on the given
	// node while attempting to apply the configuration described in the CRD.
	BpfAppStateCondProgramListChangedError BpfApplicationStateConditionType = "ProgramListChangedError"

	// BpfAppStateCondUnloadError indicates that the BPF Application was marked
	// for deletion, but unloading one or more programs was unsuccessful on the
	// given node.
	BpfAppStateCondUnloadError BpfApplicationStateConditionType = "UnloadError"

	// BpfAppStateCondUnloaded indicates that the BPF Application was marked
	// for deletion, and has been successfully unloaded.
	BpfAppStateCondUnloaded BpfApplicationStateConditionType = "Unloaded"
)

// Condition is a helper method to promote any given
// BpfApplicationStateConditionType to a full metav1.Condition in an opinionated
// fashion.
func (b BpfApplicationStateConditionType) Condition() metav1.Condition {
	cond := metav1.Condition{}

	switch b {
	case BpfAppStateCondPending:
		condType := string(BpfAppStateCondPending)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Pending",
			Message: "Not yet complete",
		}
	case BpfAppStateCondSuccess:
		condType := string(BpfAppStateCondSuccess)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Success",
			Message: "The BPF application has been successfully loaded and attached",
		}
	case BpfAppStateCondError:
		condType := string(BpfAppStateCondError)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Error",
			Message: "An error has occurred",
		}
	case BpfAppStateCondUnloadError:
		condType := string(BpfAppStateCondUnloadError)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Unload Error",
			Message: "Unload failed for one or more programs",
		}
	case BpfAppStateCondUnloaded:
		condType := string(BpfAppStateCondUnloaded)
		cond = metav1.Condition{
			Type:    condType,
			Status:  metav1.ConditionTrue,
			Reason:  "Unloaded",
			Message: "The application has been successfully unloaded",
		}
	}
	return cond
}

type AppLoadStatus string

const (
	// The initial load condition
	AppLoadNotLoaded AppLoadStatus = "NotLoaded"
	// All programs for app have been loaded
	AppLoadSuccess AppLoadStatus = "LoadSuccess"
	// One or more programs for app has not been loaded
	AppLoadError AppLoadStatus = "LoadError"
	// All programs for app have been unloaded
	AppUnLoadSuccess AppLoadStatus = "UnloadSuccess"
	// One or more programs for app has not been unloaded
	AppUnloadError AppLoadStatus = "UnloadError"
	// The app is not selected to run on the node
	NotSelected AppLoadStatus = "NotSelected"
	// The program list has changed which is not allowed
	ProgListChangedError AppLoadStatus = "ProgramListChangedError"
)

type ProgramLinkStatus string

const (
	// The initial program attach state
	ProgAttachPending ProgramLinkStatus = "Pending"
	// All attachments for program are in the correct state
	ProgAttachSuccess ProgramLinkStatus = "Success"
	// One or more attachments for program are not in the correct state
	ProgAttachError ProgramLinkStatus = "Error"
	// There was an error updating the attach info
	UpdateAttachInfoError ProgramLinkStatus = "UpdateAttachInfoError"
)

type LinkStatus string

const (
	// Attach point is attached
	ApAttachAttached LinkStatus = "Attached"
	// Attach point is not attached
	ApAttachNotAttached LinkStatus = "NotAttached"
	// An attach was attempted, but there was an error
	ApAttachError LinkStatus = "AttachError"
	// A detach was attempted, but there was an error
	ApDetachError LinkStatus = "DetachError"
)
