package manager

import (
	"errors"
)

// Config of the operator.
type Config struct {
	// DemoLokiImage is the image of the zero click loki deployment that is managed by the operator
	DemoLokiImage string
	// EBPFAgentImage is the image of the eBPF agent that is managed by the operator
	EBPFAgentImage string
	// FlowlogsPipelineImage is the image of the Flowlogs-Pipeline that is managed by the operator
	FlowlogsPipelineImage string
	// ConsolePluginImage is the image of the Console Plugin that is managed by the operator
	ConsolePluginImage string
	// ConsolePluginCompatImage is a backward compatible image of the Console Plugin that is managed by the operator (e.g. a Patterfly 4 variant)
	ConsolePluginCompatImage string
	// EBPFByteCodeImage is the ebpf byte code image used by EBPF Manager
	EBPFByteCodeImage string
	// Default namespace
	Namespace string
	// Release kind is either upstream or downstream
	DownstreamDeployment bool
	// Hold mode: when enabled, all operator-controlled resources are deleted while keeping CRDs (FlowCollector, FlowCollectorSlice, FlowMetric) and namespaces
	Hold bool
}

func (cfg *Config) Validate() error {
	if cfg.EBPFAgentImage == "" {
		return errors.New("eBPF agent image argument can't be empty")
	}
	if cfg.FlowlogsPipelineImage == "" {
		return errors.New("flowlogs-pipeline image argument can't be empty")
	}
	if cfg.ConsolePluginImage == "" {
		return errors.New("console plugin image argument can't be empty")
	}
	if cfg.Namespace == "" {
		return errors.New("namespace argument can't be empty")
	}
	return nil
}
