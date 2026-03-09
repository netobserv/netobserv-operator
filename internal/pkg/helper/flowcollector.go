package helper

import (
	"strings"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	netobservManagedLabel = "netobserv-managed"
)

func GetRecordTypes(processor *flowslatest.FlowCollectorFLP) []api.ConnTrackOutputRecordTypeEnum {
	if processor.LogTypes != nil {
		switch *processor.LogTypes {
		case flowslatest.LogTypeFlows:
			return []api.ConnTrackOutputRecordTypeEnum{api.ConnTrackFlowLog}
		case flowslatest.LogTypeConversations:
			return []api.ConnTrackOutputRecordTypeEnum{
				api.ConnTrackNewConnection,
				api.ConnTrackHeartbeat,
				api.ConnTrackEndConnection,
			}
		case flowslatest.LogTypeEndedConversations:
			return []api.ConnTrackOutputRecordTypeEnum{api.ConnTrackEndConnection}
		case flowslatest.LogTypeAll:
			return []api.ConnTrackOutputRecordTypeEnum{
				api.ConnTrackFlowLog,
				api.ConnTrackNewConnection,
				api.ConnTrackHeartbeat,
				api.ConnTrackEndConnection,
			}
		}
	}
	return []api.ConnTrackOutputRecordTypeEnum{api.ConnTrackFlowLog}
}

func PtrBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func PtrInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func AddManagedLabel(obj client.Object) {
	// set netobserv-managed label to true so users can easily switch to false if they want to skip ownership
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[netobservManagedLabel] = "true"
	obj.SetLabels(labels)
}

func IsManaged(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels[netobservManagedLabel] == "true"
}

// special case where ownership is ignored if netobserv-managed label is explicitly set to false
// this is used to allow users to create namespaces with custom labels and annotations prior to the reconciliation
func SkipOwnership(obj client.Object) bool {
	labels := obj.GetLabels()
	return labels != nil && labels[netobservManagedLabel] == "false"
}

func IsOwned(obj client.Object) bool {
	// ownership is forced if netobserv-managed label is explicitly set to true
	if IsManaged(obj) {
		return true
	}
	// else we check for owner references
	refs := obj.GetOwnerReferences()
	return len(refs) > 0 && strings.HasPrefix(refs[0].APIVersion, flowslatest.GroupVersion.Group)
}

func GetAdvancedAgentConfig(specConfig *flowslatest.AdvancedAgentConfig) flowslatest.AdvancedAgentConfig {
	cfg := flowslatest.AdvancedAgentConfig{
		Env: map[string]string{},
		Scheduling: &flowslatest.SchedulingConfig{
			NodeSelector:      map[string]string{},
			Tolerations:       []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			Affinity:          nil,
			PriorityClassName: "",
		},
	}

	if specConfig != nil {
		if len(specConfig.Env) > 0 {
			cfg.Env = specConfig.Env
		}
		if specConfig.Scheduling != nil {
			if specConfig.Scheduling.NodeSelector != nil {
				cfg.Scheduling.NodeSelector = specConfig.Scheduling.NodeSelector
			}
			if specConfig.Scheduling.Tolerations != nil {
				cfg.Scheduling.Tolerations = specConfig.Scheduling.Tolerations
			}
			if specConfig.Scheduling.Affinity != nil {
				cfg.Scheduling.Affinity = specConfig.Scheduling.Affinity
			}
			if len(specConfig.Scheduling.PriorityClassName) > 0 {
				cfg.Scheduling.PriorityClassName = specConfig.Scheduling.PriorityClassName
			}
		}
	}

	return cfg
}

func GetAdvancedProcessorConfig(spec *flowslatest.FlowCollectorSpec) flowslatest.AdvancedProcessorConfig {
	var defaultToleration []corev1.Toleration
	if spec.UseHostNetwork() {
		// When using host-network ie. 1-1 pairing with agents, FLP should have same toleration as agents
		defaultToleration = []corev1.Toleration{{Operator: corev1.TolerationOpExists}}
	}
	cfg := flowslatest.AdvancedProcessorConfig{
		Env:                            map[string]string{},
		Port:                           ptr.To(GetFieldDefaultInt32(ProcessorAdvancedPath, "port")),
		HealthPort:                     ptr.To(GetFieldDefaultInt32(ProcessorAdvancedPath, "healthPort")),
		EnableKubeProbes:               ptr.To(GetFieldDefaultBool(ProcessorAdvancedPath, "enableKubeProbes")),
		DropUnusedFields:               ptr.To(GetFieldDefaultBool(ProcessorAdvancedPath, "dropUnusedFields")),
		ConversationHeartbeatInterval:  ptr.To(GetFieldDefaultDuration(ProcessorAdvancedPath, "conversationHeartbeatInterval")),
		ConversationEndTimeout:         ptr.To(GetFieldDefaultDuration(ProcessorAdvancedPath, "conversationEndTimeout")),
		ConversationTerminatingTimeout: ptr.To(GetFieldDefaultDuration(ProcessorAdvancedPath, "conversationTerminatingTimeout")),
		Scheduling: &flowslatest.SchedulingConfig{
			NodeSelector:      map[string]string{},
			Tolerations:       defaultToleration,
			Affinity:          nil,
			PriorityClassName: "",
		},
	}

	specConfig := spec.Processor.Advanced
	if specConfig != nil {
		if len(specConfig.Env) > 0 {
			cfg.Env = specConfig.Env
		}
		if specConfig.Port != nil && *specConfig.Port > 0 {
			cfg.Port = specConfig.Port
		}
		if specConfig.HealthPort != nil && *specConfig.HealthPort > 0 {
			cfg.HealthPort = specConfig.HealthPort
		}
		if specConfig.ProfilePort != nil && *specConfig.ProfilePort > 0 {
			cfg.ProfilePort = specConfig.ProfilePort
		}
		if specConfig.EnableKubeProbes != nil {
			cfg.EnableKubeProbes = specConfig.EnableKubeProbes
		}
		if specConfig.DropUnusedFields != nil {
			cfg.DropUnusedFields = specConfig.DropUnusedFields
		}
		if specConfig.ConversationHeartbeatInterval != nil {
			cfg.ConversationHeartbeatInterval = specConfig.ConversationHeartbeatInterval
		}
		if specConfig.ConversationEndTimeout != nil {
			cfg.ConversationEndTimeout = specConfig.ConversationEndTimeout
		}
		if specConfig.ConversationTerminatingTimeout != nil {
			cfg.ConversationTerminatingTimeout = specConfig.ConversationTerminatingTimeout
		}
		if specConfig.Scheduling != nil {
			if specConfig.Scheduling.NodeSelector != nil {
				cfg.Scheduling.NodeSelector = specConfig.Scheduling.NodeSelector
			}
			if specConfig.Scheduling.Tolerations != nil {
				cfg.Scheduling.Tolerations = specConfig.Scheduling.Tolerations
			}
			if specConfig.Scheduling.Affinity != nil {
				cfg.Scheduling.Affinity = specConfig.Scheduling.Affinity
			}
			if len(specConfig.Scheduling.PriorityClassName) > 0 {
				cfg.Scheduling.PriorityClassName = specConfig.Scheduling.PriorityClassName
			}
		}
	}

	return cfg
}

func GetAdvancedLokiConfig(specConfig *flowslatest.AdvancedLokiConfig) flowslatest.AdvancedLokiConfig {
	cfg := flowslatest.AdvancedLokiConfig{
		WriteMinBackoff: ptr.To(GetFieldDefaultDuration(LokiAdvancedPath, "writeMinBackoff")),
		WriteMaxBackoff: ptr.To(GetFieldDefaultDuration(LokiAdvancedPath, "writeMaxBackoff")),
		WriteMaxRetries: ptr.To(GetFieldDefaultInt32(LokiAdvancedPath, "writeMaxRetries")),
		StaticLabels:    GetFieldDefaultMapString(LokiAdvancedPath, "staticLabels"),
	}

	if specConfig != nil {
		if specConfig.WriteMinBackoff != nil {
			cfg.WriteMinBackoff = specConfig.WriteMinBackoff
		}
		if specConfig.WriteMaxBackoff != nil {
			cfg.WriteMaxBackoff = specConfig.WriteMaxBackoff
		}
		if specConfig.WriteMaxRetries != nil {
			cfg.WriteMaxRetries = specConfig.WriteMaxRetries
		}
		if specConfig.StaticLabels != nil {
			cfg.StaticLabels = specConfig.StaticLabels
		}
	}

	return cfg
}

func GetAdvancedPluginConfig(specConfig *flowslatest.AdvancedPluginConfig) flowslatest.AdvancedPluginConfig {
	cfg := flowslatest.AdvancedPluginConfig{
		Env:      map[string]string{},
		Args:     []string{},
		Register: ptr.To(GetFieldDefaultBool(PluginAdvancedPath, "register")),
		Port:     ptr.To(GetFieldDefaultInt32(PluginAdvancedPath, "port")),
		Scheduling: &flowslatest.SchedulingConfig{
			NodeSelector:      map[string]string{},
			Tolerations:       nil,
			Affinity:          nil,
			PriorityClassName: "",
		},
	}

	if specConfig != nil {
		if len(specConfig.Env) > 0 {
			cfg.Env = specConfig.Env
		}
		if len(specConfig.Args) > 0 {
			cfg.Args = specConfig.Args
		}
		if specConfig.Register != nil {
			cfg.Register = specConfig.Register
		}
		if specConfig.Port != nil && *specConfig.Port > 0 {
			cfg.Port = specConfig.Port
		}
		if specConfig.Scheduling != nil {
			if specConfig.Scheduling.NodeSelector != nil {
				cfg.Scheduling.NodeSelector = specConfig.Scheduling.NodeSelector
			}
			if specConfig.Scheduling.Tolerations != nil {
				cfg.Scheduling.Tolerations = specConfig.Scheduling.Tolerations
			}
			if specConfig.Scheduling.Affinity != nil {
				cfg.Scheduling.Affinity = specConfig.Scheduling.Affinity
			}
			if len(specConfig.Scheduling.PriorityClassName) > 0 {
				cfg.Scheduling.PriorityClassName = specConfig.Scheduling.PriorityClassName
			}
		}
	}

	return cfg
}
