/*
 * Copyright (C) 2022 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package api

type TransformNetwork struct {
	Rules         NetworkTransformRules         `yaml:"rules" json:"rules" doc:"list of transform rules, each includes:"`
	KubeConfig    NetworkTransformKubeConfig    `yaml:"kubeConfig,omitempty" json:"kubeConfig,omitempty" doc:"global configuration related to Kubernetes (optional)"`
	ServicesFile  string                        `yaml:"servicesFile,omitempty" json:"servicesFile,omitempty" doc:"path to services file (optional, default: /etc/services)"`
	ProtocolsFile string                        `yaml:"protocolsFile,omitempty" json:"protocolsFile,omitempty" doc:"path to protocols file (optional, default: /etc/protocols)"`
	SubnetLabels  []NetworkTransformSubnetLabel `yaml:"subnetLabels,omitempty" json:"subnetLabels,omitempty" doc:"configure subnet and IPs custom labels"`
	DirectionInfo NetworkTransformDirectionInfo `yaml:"directionInfo,omitempty" json:"directionInfo,omitempty" doc:"information to reinterpret flow direction (optional, to use with reinterpret_direction rule)"`
}

func (tn *TransformNetwork) Preprocess() {
	for i := range tn.Rules {
		tn.Rules[i].preprocess()
	}
}

func (tn *TransformNetwork) GetServiceFiles() (string, string) {
	p := tn.ProtocolsFile
	if p == "" {
		p = "/etc/protocols"
	}
	s := tn.ServicesFile
	if s == "" {
		s = "/etc/services"
	}
	return p, s
}

const (
	OVN = "ovn"
)

type NetworkTransformKubeConfig struct {
	ConfigPath        string             `yaml:"configPath,omitempty" json:"configPath,omitempty" doc:"path to kubeconfig file (optional)"`
	SecondaryNetworks []SecondaryNetwork `yaml:"secondaryNetworks,omitempty" json:"secondaryNetworks,omitempty" doc:"configuration for secondary networks"`
	ManagedCNI        []string           `yaml:"managedCNI,omitempty" json:"managedCNI,omitempty" doc:"a list of CNI (network plugins) to manage, for detecting additional interfaces. Currently supported: ovn"`
	TrackedKinds      []string           `yaml:"trackedKinds,omitempty" json:"trackedKinds,omitempty" doc:"list of Kubernetes resource kinds to track for ownership chain (e.g., Deployment, Gateway, VirtualMachine). If a resource's owner is in this list, FLP will continue tracking up the ownership chain."`
}

type TransformNetworkOperationEnum string

const (
	// For doc generation, enum definitions must match format `Constant Type = "value" // doc`
	NetworkAddSubnet            TransformNetworkOperationEnum = "add_subnet"            // add output subnet field from input field and prefix length from parameters field
	NetworkAddLocation          TransformNetworkOperationEnum = "add_location"          // add output location fields from input
	NetworkAddService           TransformNetworkOperationEnum = "add_service"           // add output network service field from input port and parameters protocol field
	NetworkAddKubernetes        TransformNetworkOperationEnum = "add_kubernetes"        // add output kubernetes fields from input
	NetworkAddKubernetesInfra   TransformNetworkOperationEnum = "add_kubernetes_infra"  // add output kubernetes isInfra field from input
	NetworkReinterpretDirection TransformNetworkOperationEnum = "reinterpret_direction" // reinterpret flow direction at the node level (instead of net interface), to ease the deduplication process
	NetworkAddSubnetLabel       TransformNetworkOperationEnum = "add_subnet_label"      // categorize IPs based on known subnets configuration
	NetworkDecodeTCPFlags       TransformNetworkOperationEnum = "decode_tcp_flags"      // decode bitwise TCP flags into a string
)

type NetworkTransformRule struct {
	Type            TransformNetworkOperationEnum `yaml:"type,omitempty" json:"type,omitempty" doc:"(enum) one of the following:"`
	KubernetesInfra *K8sInfraRule                 `yaml:"kubernetes_infra,omitempty" json:"kubernetes_infra,omitempty" doc:"Kubernetes infra rule configuration"`
	Kubernetes      *K8sRule                      `yaml:"kubernetes,omitempty" json:"kubernetes,omitempty" doc:"Kubernetes rule configuration"`
	AddSubnet       *NetworkAddSubnetRule         `yaml:"add_subnet,omitempty" json:"add_subnet,omitempty" doc:"Add subnet rule configuration"`
	AddLocation     *NetworkAddLocationRule       `yaml:"add_location,omitempty" json:"add_location,omitempty" doc:"Add location rule configuration"`
	AddSubnetLabel  *NetworkAddSubnetLabelRule    `yaml:"add_subnet_label,omitempty" json:"add_subnet_label,omitempty" doc:"Add subnet label rule configuration"`
	AddService      *NetworkAddServiceRule        `yaml:"add_service,omitempty" json:"add_service,omitempty" doc:"Add service rule configuration"`
	DecodeTCPFlags  *NetworkGenericRule           `yaml:"decode_tcp_flags,omitempty" json:"decode_tcp_flags,omitempty" doc:"Decode bitwise TCP flags into a string"`
}

func (r *NetworkTransformRule) preprocess() {
	if r.Kubernetes != nil {
		r.Kubernetes.preprocess()
	}
}

type K8sInfraRule struct {
	NamespaceNameFields []K8sReference `yaml:"namespaceNameFields,omitempty" json:"namespaceNameFields,omitempty" doc:"entries for namespace and name input fields"`
	Output              string         `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
	InfraPrefixes       []string       `yaml:"infra_prefixes,omitempty" json:"infra_prefixes,omitempty" doc:"Namespace prefixes that will be tagged as infra"`
	InfraRefs           []K8sReference `yaml:"infra_refs,omitempty" json:"infra_refs,omitempty" doc:"Additional object references to be tagged as infra"`
}

type K8sReference struct {
	Name      string `yaml:"name,omitempty" json:"name,omitempty" doc:"name of the object"`
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" doc:"namespace of the object"`
}

type K8sRule struct {
	IPField                  string              `yaml:"ipField,omitempty" json:"ipField,omitempty" doc:"entry IP input field"`
	InterfacesField          string              `yaml:"interfacesField,omitempty" json:"interfacesField,omitempty" doc:"entry Interfaces input field"`
	UDNsField                string              `yaml:"udnsField,omitempty" json:"udnsField,omitempty" doc:"entry UDNs input field"`
	MACField                 string              `yaml:"macField,omitempty" json:"macField,omitempty" doc:"entry MAC input field"`
	Output                   string              `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
	Assignee                 string              `yaml:"assignee,omitempty" json:"assignee,omitempty" doc:"value needs to assign to output field"`
	LabelsPrefix             string              `yaml:"labels_prefix,omitempty" json:"labels_prefix,omitempty" doc:"labels prefix to use to copy input labels, if empty labels will not be copied"`
	LabelInclusions          []string            `yaml:"label_inclusions,omitempty" json:"label_inclusions,omitempty" doc:"labels to include, if empty all labels will be included. Only used if labels_prefix is specified"`
	LabelExclusions          []string            `yaml:"label_exclusions,omitempty" json:"label_exclusions,omitempty" doc:"labels to exclude, if empty no labels will be excluded. Only used if labels_prefix is specified"`
	LabelValueMaxLength      *int                `yaml:"label_value_max_length,omitempty" json:"label_value_max_length,omitempty" doc:"label value max length, if specified, will trim label values to this length"`
	AnnotationsPrefix        string              `yaml:"annotations_prefix,omitempty" json:"annotations_prefix,omitempty" doc:"annotations prefix to use to copy input annotations, if empty annotations will not be copied"`
	AnnotationInclusions     []string            `yaml:"annotation_inclusions,omitempty" json:"annotation_inclusions,omitempty" doc:"annotations to include, if empty all annotations will be included. Only used if annotations_prefix is specified"`
	AnnotationExclusions     []string            `yaml:"annotation_exclusions,omitempty" json:"annotation_exclusions,omitempty" doc:"annotations to exclude, if empty no annotations will be excluded. Only used if annotations_prefix is specified"`
	AnnotationValueMaxLength *int                `yaml:"annotation_value_max_length,omitempty" json:"annotation_value_max_length,omitempty" doc:"annotation value max length, if specified, will trim annotation values to this length"`
	AddZone                  bool                `yaml:"add_zone,omitempty" json:"add_zone,omitempty" doc:"if true the rule will add the zone"`
	OutputKeys               K8SOutputKeys       `yaml:"-" json:"-"`
	LabelInclusionsMap       map[string]struct{} `yaml:"-" json:"-"`
	LabelExclusionsMap       map[string]struct{} `yaml:"-" json:"-"`
	AnnotationInclusionsMap  map[string]struct{} `yaml:"-" json:"-"`
	AnnotationExclusionsMap  map[string]struct{} `yaml:"-" json:"-"`
}

type K8SOutputKeys struct {
	Namespace   string
	Name        string
	Kind        string
	OwnerName   string
	OwnerKind   string
	NetworkName string
	HostIP      string
	HostName    string
	Zone        string
}

func (r *K8sRule) preprocess() {
	if r.Assignee == "otel" {
		// NOTE: Some of these fields are taken from opentelemetry specs.
		// See https://opentelemetry.io/docs/specs/semconv/resource/k8s/
		// Other fields (not specified in the specs) are named similarly
		r.OutputKeys = K8SOutputKeys{
			Namespace:   r.Output + "k8s.namespace.name",
			Name:        r.Output + "k8s.name",
			Kind:        r.Output + "k8s.type",
			OwnerName:   r.Output + "k8s.owner.name",
			OwnerKind:   r.Output + "k8s.owner.type",
			NetworkName: r.Output + "k8s.net.name",
			HostIP:      r.Output + "k8s.host.ip",
			HostName:    r.Output + "k8s.host.name",
			Zone:        r.Output + "k8s.zone",
		}
	} else {
		r.OutputKeys = K8SOutputKeys{
			Namespace:   r.Output + "_Namespace",
			Name:        r.Output + "_Name",
			Kind:        r.Output + "_Type",
			OwnerName:   r.Output + "_OwnerName",
			OwnerKind:   r.Output + "_OwnerType",
			NetworkName: r.Output + "_NetworkName",
			HostIP:      r.Output + "_HostIP",
			HostName:    r.Output + "_HostName",
			Zone:        r.Output + "_Zone",
		}
	}

	r.LabelInclusionsMap = buildStringMap(r.LabelInclusions)
	r.LabelExclusionsMap = buildStringMap(r.LabelExclusions)
	r.AnnotationInclusionsMap = buildStringMap(r.AnnotationInclusions)
	r.AnnotationExclusionsMap = buildStringMap(r.AnnotationExclusions)
}

type SecondaryNetwork struct {
	Name  string         `yaml:"name,omitempty" json:"name,omitempty" doc:"deprecated / unused"`
	Index map[string]any `yaml:"index,omitempty" json:"index,omitempty" doc:"fields to use for indexing, must be any combination of 'mac', 'ip', 'interface', or 'udn'"`
}

type NetworkGenericRule struct {
	Input  string `yaml:"input,omitempty" json:"input,omitempty" doc:"entry input field"`
	Output string `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
}

type NetworkAddSubnetRule struct {
	Input      string `yaml:"input,omitempty" json:"input,omitempty" doc:"entry input field"`
	Output     string `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
	SubnetMask string `yaml:"subnet_mask,omitempty" json:"subnet_mask,omitempty" doc:"subnet mask field"`
}

type NetworkAddLocationRule struct {
	Input    string `yaml:"input,omitempty" json:"input,omitempty" doc:"entry input field"`
	Output   string `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
	FilePath string `yaml:"file_path,omitempty" json:"file_path,omitempty" doc:"path of the location DB file (zip archive), from ip2location.com (Lite DB9); leave unset to try downloading the file at startup"`
}

type NetworkAddSubnetLabelRule struct {
	Input  string `yaml:"input,omitempty" json:"input,omitempty" doc:"entry input field"`
	Output string `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
}

type NetworkAddServiceRule struct {
	Input    string `yaml:"input,omitempty" json:"input,omitempty" doc:"entry input field"`
	Output   string `yaml:"output,omitempty" json:"output,omitempty" doc:"entry output field"`
	Protocol string `yaml:"protocol,omitempty" json:"protocol,omitempty" doc:"entry protocol field"`
}

type NetworkTransformDirectionInfo struct {
	ReporterIPField    string `yaml:"reporterIPField,omitempty" json:"reporterIPField,omitempty" doc:"field providing the reporter (agent) host IP"`
	SrcHostField       string `yaml:"srcHostField,omitempty" json:"srcHostField,omitempty" doc:"source host field"`
	DstHostField       string `yaml:"dstHostField,omitempty" json:"dstHostField,omitempty" doc:"destination host field"`
	FlowDirectionField string `yaml:"flowDirectionField,omitempty" json:"flowDirectionField,omitempty" doc:"field providing the flow direction in the input entries; it will be rewritten"`
	IfDirectionField   string `yaml:"ifDirectionField,omitempty" json:"ifDirectionField,omitempty" doc:"interface-level field for flow direction, to create in output"`
}

type NetworkTransformRules []NetworkTransformRule

type NetworkTransformSubnetLabel struct {
	CIDRs []string `yaml:"cidrs,omitempty" json:"cidrs,omitempty" doc:"list of CIDRs to match a label"`
	Name  string   `yaml:"name,omitempty" json:"name,omitempty" doc:"name of the label"`
}

func buildStringMap(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, item := range items {
		m[item] = struct{}{}
	}
	return m
}
