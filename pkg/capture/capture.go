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
	agentconfig "github.com/netobserv/netobserv-ebpf-agent/pkg/config"
	flowcapturev1alpha1 "github.com/netobserv/netobserv-operator/api/flowcapture/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
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
type Feature = flowcapturev1alpha1.Feature

const (
	PacketDrop        = flowcapturev1alpha1.PacketDrop
	DNSTracking       = flowcapturev1alpha1.DNSTracking
	FlowRTT           = flowcapturev1alpha1.FlowRTT
	NetworkEvents     = flowcapturev1alpha1.NetworkEvents
	PacketTranslation = flowcapturev1alpha1.PacketTranslation
	EbpfManager       = flowcapturev1alpha1.EbpfManager
	UDNMapping        = flowcapturev1alpha1.UDNMapping
	IPSec             = flowcapturev1alpha1.IPSec
	TLSTracking       = flowcapturev1alpha1.TLSTracking
	SecondaryNetworks = flowcapturev1alpha1.SecondaryNetworks
)

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

// FeaturesRequirePrivilege is kept for compatibility with callers of the shared
// package. Features use granular capabilities and never require a privileged
// container; TLS plaintext capture is handled separately by TLSPlaintextOptions.
func FeaturesRequirePrivilege(_ []Feature) bool {
	return false
}

type FlowFilter = flowcapturev1alpha1.FlowFilter

// NormalizeFilters applies the defaults used by the eBPF agent and by the
// CLI filter builder. Keeping them here makes FlowCapture and CLI-created
// captures produce the same agent configuration.
func NormalizeFilters(filters []FlowFilter) []agentconfig.FlowFilter {
	normalized := make([]agentconfig.FlowFilter, len(filters))
	for i := range normalized {
		source := filters[i]
		normalized[i] = agentconfig.FlowFilter{
			Direction: source.Direction, IPCIDR: source.IPCIDR, Protocol: source.Protocol,
			SourcePort: source.SourcePort, DestinationPort: source.DestinationPort, Port: source.Port,
			SourcePortRange: source.SourcePortRange, SourcePorts: source.SourcePorts,
			DestinationPortRange: source.DestinationPortRange, DestinationPorts: source.DestinationPorts,
			PortRange: source.PortRange, Ports: source.Ports, ICMPType: source.ICMPType,
			ICMPCode: source.ICMPCode, PeerIP: source.PeerIP, PeerCIDR: source.PeerCIDR,
			Action: source.Action, TCPFlags: source.TCPFlags, Drops: source.Drops, Sample: source.Sample,
		}
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
	ActiveDeadlineSeconds                  *int64
}

func BuildCollectorService(name, namespace string, selector map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: selector},
		Spec: corev1.ServiceSpec{Selector: selector, Ports: []corev1.ServicePort{
			{Name: "collector", Protocol: corev1.ProtocolTCP, Port: 9999, TargetPort: intstr.FromInt32(9999)},
			{Name: "prometheus", Protocol: corev1.ProtocolTCP, Port: 9401, TargetPort: intstr.FromInt32(9401)},
		}},
	}
}

func BuildCollectorPod(options *CollectorPodOptions) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: options.Name, Namespace: options.Namespace, Labels: map[string]string{"run": options.Name}},
		Spec: corev1.PodSpec{
			ServiceAccountName:    options.ServiceAccount,
			RestartPolicy:         corev1.RestartPolicyNever,
			ActiveDeadlineSeconds: options.ActiveDeadlineSeconds,
			Volumes:               []corev1.Volume{{Name: "output", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
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
	TLSPlaintext                                       *TLSPlaintextOptions
}

// TLSPlaintextOptions controls the packet agent's libssl uprobes.
type TLSPlaintextOptions struct {
	MinBytes         int32
	PreviewBytes     int32
	ProcessAllowlist []string
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
	metricsPort := options.MetricsPort
	if metricsPort == 0 {
		metricsPort = 9400
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
	filters, _ := json.Marshal(NormalizeFilters(options.Filters))
	envs = append(envs, corev1.EnvVar{Name: "FLOW_FILTER_RULES", Value: string(filters)})
	volumes, mounts := agentVolumes(options.Features)
	plaintext := options.Type == Packets && options.TLSPlaintext != nil
	if plaintext {
		addTLSPlaintextVolumes(&volumes, &mounts)
	}
	container := corev1.Container{Name: containerName, Image: options.Image, ImagePullPolicy: corev1.PullAlways, Env: envs, VolumeMounts: mounts,
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false), Privileged: ptr.To(false), ReadOnlyRootFilesystem: ptr.To(!plaintext), RunAsUser: int64Ptr(0), Capabilities: &corev1.Capabilities{Add: agentCapabilities(options.Features, plaintext), Drop: []corev1.Capability{"ALL"}}}}
	if options.Type == Metrics {
		port := options.MetricsPort
		if port == 0 {
			port = 9401
		}
		container.Ports = []corev1.ContainerPort{{Name: "prometheus", ContainerPort: port, Protocol: corev1.ProtocolTCP}}
	} else {
		container.Ports = []corev1.ContainerPort{{Name: "health", ContainerPort: metricsPort, Protocol: corev1.ProtocolTCP}}
	}
	return &appsv1.DaemonSet{TypeMeta: metav1.TypeMeta{APIVersion: "apps/v1", Kind: "DaemonSet"}, ObjectMeta: metav1.ObjectMeta{Name: options.Name, Namespace: options.Namespace, Labels: labels}, Spec: appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: labels},
		Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{
			ServiceAccountName: options.ServiceAccount, HostNetwork: true, HostPID: plaintext, DNSPolicy: corev1.DNSClusterFirstWithHostNet, NodeSelector: options.NodeSelector,
			Tolerations: []corev1.Toleration{{Operator: corev1.TolerationOpExists}}, Volumes: volumes, Containers: []corev1.Container{container},
		}},
	}}
}

func agentEnvironment(options *AgentOptions) []corev1.EnvVar {
	metricsPort := options.MetricsPort
	if metricsPort == 0 {
		metricsPort = 9400
	}
	envs := []corev1.EnvVar{{Name: "EXPORT", Value: "direct-flp"}, {Name: "TARGET_HOST", Value: options.TargetHost}, {Name: "TARGET_PORT", Value: "9999"}, {Name: "METRICS_ENABLE", Value: "true"}, {Name: "METRICS_SERVER_PORT", Value: strconv.FormatInt(int64(metricsPort), 10)}, {Name: "METRICS_PREFIX", Value: "flowcapture_agent_"}, {Name: "LOG_LEVEL", Value: "info"}, {Name: "SAMPLING", Value: strconv.Itoa(options.Sampling)}, {Name: "INTERFACES", Value: strings.Join(options.Interfaces, ",")}, {Name: "EXCLUDE_INTERFACES", Value: strings.Join(options.ExcludeInterfaces, ",")}, {Name: "TC_ATTACH_MODE", Value: "any"}}
	if options.Type == Flows {
		envs = append(envs, corev1.EnvVar{Name: "CACHE_ACTIVE_TIMEOUT", Value: "1s"}, corev1.EnvVar{Name: "PREFERRED_INTERFACE_FOR_MAC_PREFIX", Value: "0a:58=eth0"})
	}
	if options.Type == Packets {
		envs = append(envs, corev1.EnvVar{Name: "CACHE_ACTIVE_TIMEOUT", Value: "1s"}, corev1.EnvVar{Name: "ENABLE_PCA", Value: "true"}, corev1.EnvVar{Name: "ENABLE_OPENSSL_TRACKING", Value: "false"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_MIN_BYTES", Value: "0"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_PREVIEW_BYTES", Value: "256"}, corev1.EnvVar{Name: "TLS_PLAINTEXT_PROCESS_ALLOWLIST", Value: ""})
		if options.TLSPlaintext != nil {
			envs[len(envs)-4].Value = "true"
			envs[len(envs)-3].Value = strconv.FormatInt(int64(options.TLSPlaintext.MinBytes), 10)
			preview := options.TLSPlaintext.PreviewBytes
			if preview == 0 {
				preview = 256
			}
			envs[len(envs)-2].Value = strconv.FormatInt(int64(preview), 10)
			envs[len(envs)-1].Value = strings.Join(options.TLSPlaintext.ProcessAllowlist, ",")
		}
	}
	return envs
}

func agentCapabilities(features []Feature, plaintext bool) []corev1.Capability {
	caps := []corev1.Capability{"BPF", "PERFMON", "NET_ADMIN"}
	if featureEnabled(features, SecondaryNetworks) || plaintext {
		caps = append(caps, "SYS_ADMIN")
	}
	if plaintext {
		caps = append(caps, "SYS_PTRACE")
	}
	return caps
}

func addTLSPlaintextVolumes(volumes *[]corev1.Volume, mounts *[]corev1.VolumeMount) {
	for _, item := range []struct{ name, path string }{{"host-usr", "/usr"}, {"host-lib", "/lib"}, {"host-lib64", "/lib64"}} {
		*volumes = append(*volumes, corev1.Volume{Name: item.name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: item.path, Type: hostPathType(corev1.HostPathDirectory)}}})
		*mounts = append(*mounts, corev1.VolumeMount{Name: item.name, MountPath: "/host" + item.path, ReadOnly: true})
	}
}

func agentVolumes(features []Feature) ([]corev1.Volume, []corev1.VolumeMount) {
	var volumes []corev1.Volume
	var mounts []corev1.VolumeMount
	if featureEnabled(features, PacketDrop) {
		appendHostPath(&volumes, &mounts, "bpf-kernel-debug", "/sys/kernel/debug", corev1.HostPathDirectory)
	}
	if featureEnabled(features, NetworkEvents) || featureEnabled(features, UDNMapping) {
		for _, volume := range []struct {
			name, path string
			typeValue  corev1.HostPathType
		}{{"var-run-ovn", "/var/run/ovn-ic", corev1.HostPathDirectoryOrCreate}, {"var-run-ovs", "/var/run/openvswitch", corev1.HostPathDirectoryOrCreate}, {"var-run-netns", "/var/run/netns", corev1.HostPathDirectory}} {
			appendHostPath(&volumes, &mounts, volume.name, volume.path, volume.typeValue)
		}
	}
	if featureEnabled(features, SecondaryNetworks) {
		appendHostPath(&volumes, &mounts, "var-run-netns", "/var/run/netns", corev1.HostPathDirectory)
	}
	return volumes, mounts
}

func appendHostPath(volumes *[]corev1.Volume, mounts *[]corev1.VolumeMount, name, path string, pathType corev1.HostPathType) {
	for i := range *volumes {
		if (*volumes)[i].Name == name {
			return
		}
	}
	*volumes = append(*volumes, corev1.Volume{Name: name, VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: path, Type: hostPathType(pathType)}}})
	*mounts = append(*mounts, corev1.VolumeMount{Name: name, MountPath: path, MountPropagation: mountPropagation(corev1.MountPropagationHostToContainer)})
}

func featureEnabled(features []Feature, wanted Feature) bool {
	for _, feature := range features {
		if feature == wanted {
			return true
		}
	}
	return false
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
	case SecondaryNetworks:
		return "ENABLE_SECONDARY_NETWORKS", true
	default:
		return "", false
	}
}

func int64Ptr(value int64) *int64                                                     { return &value }
func hostPathType(value corev1.HostPathType) *corev1.HostPathType                     { return &value }
func mountPropagation(value corev1.MountPropagationMode) *corev1.MountPropagationMode { return &value }
