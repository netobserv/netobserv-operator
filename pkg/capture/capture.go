// Package capture contains lifecycle-neutral capture concepts shared by the
// CLI and operator. Kubernetes clients and reconciliation remain outside this
// package.
package capture

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	flpconfig "github.com/netobserv/flowlogs-pipeline/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type Type string

const CaptureCompletedAnnotation = "flows.netobserv.io/capture-completed"

const (
	Flows   Type = "flows"
	Packets Type = "packets"
	Metrics Type = "metrics"
)

func (t Type) Valid() bool        { return t == Flows || t == Packets || t == Metrics }
func (t Type) HasCollector() bool { return t == Flows || t == Packets }
func (t Type) Singular() string   { return strings.TrimSuffix(string(t), "s") }

// Feature is an optional eBPF agent capability. The values intentionally match
// FlowCollector's public feature names so captures can use the same API vocabulary.
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
)

func (f Feature) Valid() bool {
	switch f {
	case PacketDrop, DNSTracking, FlowRTT, NetworkEvents, PacketTranslation,
		EbpfManager, UDNMapping, IPSec, TLSTracking:
		return true
	default:
		return false
	}
}

// ValidateFeatures checks feature combinations that the underlying agent
// cannot run for a capture type. Packet capture mode is a separate PCA mode
// and does not support the flow-oriented feature environment variables.
func ValidateFeatures(kind Type, features []Feature) error {
	if kind != Packets || len(features) == 0 {
		return nil
	}
	return fmt.Errorf("packet captures do not support features: %s", strings.Join(func() []string {
		names := make([]string, len(features))
		for i, feature := range features {
			names[i] = string(feature)
		}
		return names
	}(), ", "))
}

// FeaturesRequirePrivilege reports whether the feature set needs the agent
// container to run privileged.
func FeaturesRequirePrivilege(features []Feature) bool {
	for _, feature := range features {
		switch feature {
		case PacketDrop, NetworkEvents, UDNMapping:
			return true
		case DNSTracking, FlowRTT, PacketTranslation, EbpfManager, IPSec, TLSTracking:
			// These features do not require privileged mode.
		}
	}
	return false
}

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

// NormalizeFilters applies the defaults used by the eBPF agent and by the
// CLI filter builder. Keeping them here makes FlowCapture and CLI-created
// captures produce the same agent configuration.
func NormalizeFilters(filters []FlowFilter) []FlowFilter {
	normalized := make([]FlowFilter, len(filters))
	copy(normalized, filters)
	for i := range normalized {
		if normalized[i].IPCIDR == "" {
			normalized[i].IPCIDR = "0.0.0.0/0"
		}
		if normalized[i].Action == "" {
			normalized[i].Action = "Accept"
		}
	}
	return normalized
}

type Names struct {
	AgentDaemonSet   string
	CollectorPod     string
	CollectorService string
}

func NewNames(name string) Names {
	return Names{AgentDaemonSet: name + "-agent", CollectorPod: name + "-collector", CollectorService: name + "-collector"}
}

// CollectorCommand builds the command used by a capture collector. Keeping
// this small command contract here prevents the CLI and FlowCapture controller
// from drifting apart as capture limits evolve.
func CollectorCommand(kind Type, namespace, logLevel, maxTime string, maxBytes int64, options string) []string {
	args := []string{"/network-observability-cli", "get-" + string(kind), "--loglevel", logLevel, "--maxtime", maxTime, "--namespace", namespace}
	if kind.HasCollector() && maxBytes > 0 {
		args = append(args, "--maxbytes", strconv.FormatInt(maxBytes, 10))
	}
	if options != "" {
		args = append(args, "--options", options)
	}
	return args
}

// CollectorPodOptions describes only the immutable shape of a collector pod.
// The caller owns execution, waiting, TLS setup and cleanup.
type CollectorPodOptions struct {
	Name, Namespace, ServiceAccount, Image string
	Command                                []string
}

func BuildCollectorService(name, namespace string, selector map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       corev1.ServiceSpec{Selector: selector, Ports: []corev1.ServicePort{{Name: "collector", Protocol: corev1.ProtocolTCP, Port: 9999, TargetPort: intstr.FromInt32(9999)}}},
	}
}

func BuildCollectorPod(options *CollectorPodOptions) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: options.Name, Namespace: options.Namespace, Labels: map[string]string{"run": options.Name}},
		Spec: corev1.PodSpec{
			ServiceAccountName: options.ServiceAccount,
			RestartPolicy:      corev1.RestartPolicyNever,
			Volumes:            []corev1.Volume{{Name: "output", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
			Containers: []corev1.Container{{
				Name: options.Name, Image: options.Image, Command: options.Command,
				VolumeMounts: []corev1.VolumeMount{{Name: "output", MountPath: "/output"}},
			}},
		},
	}
}

func BuildServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "ServiceAccount"}, ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
}

// AgentOptions is the common, lifecycle-neutral part of an agent DaemonSet.
type AgentOptions struct {
	Name, Namespace, ServiceAccount, Image, TargetHost string
	NodeSelector                                       map[string]string
	Sampling                                           int
	Interfaces, ExcludeInterfaces                      []string
	Features                                           []Feature
	Filters                                            []FlowFilter
	Type                                               Type
	MetricsPort                                        int32
	Labels                                             map[string]string
	ContainerName                                      string
	PipelineConfig                                     *flpconfig.Root
}

func BuildAgentDaemonSet(options *AgentOptions) *appsv1.DaemonSet {
	labels := options.Labels
	if labels == nil {
		labels = map[string]string{"flowcapture": options.Name, "component": "agent"}
	}
	containerName := options.ContainerName
	if containerName == "" {
		containerName = "agent"
	}
	envs := agentEnvironment(options)
	if options.PipelineConfig != nil {
		pipelineConfig, err := PipelineConfigJSON(options.PipelineConfig)
		if err != nil {
			panic("capture pipeline configuration cannot be serialized: " + err.Error())
		}
		envs = append(envs, corev1.EnvVar{Name: "FLP_CONFIG", Value: pipelineConfig})
	}
	for _, feature := range options.Features {
		if envName, ok := FeatureEnvName(feature); ok {
			envs = append(envs, corev1.EnvVar{Name: envName, Value: "true"})
		}
	}
	privileged := FeaturesRequirePrivilege(options.Features)
	filters, _ := json.Marshal(NormalizeFilters(options.Filters))
	envs = append(envs, corev1.EnvVar{Name: "FLOW_FILTER_RULES", Value: string(filters)})
	volumes, mounts := agentVolumes(options.Type)
	container := corev1.Container{Name: containerName, Image: options.Image, ImagePullPolicy: corev1.PullAlways, Env: envs, VolumeMounts: mounts,
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: boolPtr(false), Privileged: boolPtr(privileged), ReadOnlyRootFilesystem: boolPtr(true), RunAsUser: int64Ptr(0), Capabilities: &corev1.Capabilities{Add: []corev1.Capability{"BPF", "PERFMON", "NET_ADMIN"}, Drop: []corev1.Capability{"ALL"}}}}
	if options.Type == Metrics {
		port := options.MetricsPort
		if port == 0 {
			port = 9401
		}
		container.Ports = []corev1.ContainerPort{{Name: "prometheus", ContainerPort: port, Protocol: corev1.ProtocolTCP}}
	}
	return &appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"}, ObjectMeta: metav1.ObjectMeta{Name: options.Name, Namespace: options.Namespace, Labels: labels}, Spec: appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{
			ServiceAccountName: options.ServiceAccount, HostNetwork: true, DNSPolicy: corev1.DNSClusterFirstWithHostNet, NodeSelector: options.NodeSelector,
			Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}}, Volumes: volumes, Containers: []corev1.Container{container},
		}},
	}}
}

func agentEnvironment(options *AgentOptions) []corev1.EnvVar {
	envs := []corev1.EnvVar{{Name: "EXPORT", Value: "direct-flp"}, {Name: "TARGET_HOST", Value: options.TargetHost}, {Name: "TARGET_PORT", Value: "9999"}, {Name: "METRICS_ENABLE", Value: "false"}, {Name: "LOG_LEVEL", Value: "info"}, {Name: "SAMPLING", Value: strconv.Itoa(options.Sampling)}, {Name: "INTERFACES", Value: strings.Join(options.Interfaces, ",")}, {Name: "EXCLUDE_INTERFACES", Value: strings.Join(options.ExcludeInterfaces, ",")}, {Name: "TC_ATTACH_MODE", Value: "any"}}
	if options.Type == Flows {
		envs = append(envs, corev1.EnvVar{Name: "CACHE_ACTIVE_TIMEOUT", Value: "1s"}, corev1.EnvVar{Name: "PREFERRED_INTERFACE_FOR_MAC_PREFIX", Value: "0a:58=eth0"})
	}
	if options.Type == Packets {
		envs = append(envs, corev1.EnvVar{Name: "CACHE_ACTIVE_TIMEOUT", Value: "1s"}, corev1.EnvVar{Name: "ENABLE_PCA", Value: "true"}, corev1.EnvVar{Name: "ENABLE_OPENSSL_TRACKING", Value: "false"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_MIN_BYTES", Value: "0"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_PREVIEW_BYTES", Value: "256"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_PROCESS_ALLOWLIST", Value: ""})
	}
	return envs
}

func agentVolumes(kind Type) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{{Name: "bpf-kernel-debug", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: "/sys/kernel/debug", Type: hostPathType(corev1.HostPathDirectory)}}}}
	mounts := []corev1.VolumeMount{{Name: "bpf-kernel-debug", MountPath: "/sys/kernel/debug", MountPropagation: mountPropagation(corev1.MountPropagationHostToContainer)}}
	if kind == Flows {
		for _, volume := range []struct {
			name, path string
			typeValue  corev1.HostPathType
		}{{"var-run-ovn", "/var/run/ovn-ic", corev1.HostPathDirectoryOrCreate}, {"var-run-ovs", "/var/run/openvswitch", corev1.HostPathDirectoryOrCreate}, {"var-run-netns", "/var/run/netns", corev1.HostPathDirectory}} {
			volumes = append(volumes, corev1.Volume{Name: volume.name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: volume.path, Type: hostPathType(volume.typeValue)}}})
			mounts = append(mounts, corev1.VolumeMount{Name: volume.name, MountPath: volume.path, MountPropagation: mountPropagation(corev1.MountPropagationHostToContainer)})
		}
	}
	if kind == Metrics {
		for _, volume := range []struct{ name, path string }{{"var-run-ovn", "/var/run/ovn-ic"}, {"var-run-ovs", "/var/run/openvswitch"}} {
			volumes = append(volumes, corev1.Volume{Name: volume.name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: volume.path, Type: hostPathType(corev1.HostPathDirectoryOrCreate)}}})
			mounts = append(mounts, corev1.VolumeMount{Name: volume.name, MountPath: volume.path, MountPropagation: mountPropagation(corev1.MountPropagationHostToContainer)})
		}
	}
	return volumes, mounts
}

// FeatureEnvName returns the eBPF agent environment variable for a feature.
func FeatureEnvName(feature Feature) (string, bool) {
	switch feature {
	case PacketDrop:
		return "ENABLE_PKT_DROPS", true
	case DNSTracking:
		return "ENABLE_DNS_TRACKING", true
	case FlowRTT:
		return "ENABLE_RTT", true
	case NetworkEvents:
		return "ENABLE_NETWORK_EVENTS_MONITORING", true
	case PacketTranslation:
		return "ENABLE_PKT_TRANSLATION", true
	case EbpfManager:
		return "EBPF_PROGRAM_MANAGER_MODE", true
	case UDNMapping:
		return "ENABLE_UDN_MAPPING", true
	case IPSec:
		return "ENABLE_IPSEC_TRACKING", true
	case TLSTracking:
		return "ENABLE_TLS_TRACKING", true
	default:
		return "", false
	}
}

func boolPtr(value bool) *bool                                                        { return &value }
func int64Ptr(value int64) *int64                                                     { return &value }
func hostPathType(value corev1.HostPathType) *corev1.HostPathType                     { return &value }
func mountPropagation(value corev1.MountPropagationMode) *corev1.MountPropagationMode { return &value }
