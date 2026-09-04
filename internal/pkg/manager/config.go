package manager

import (
	"errors"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
)

// Config of the operator.
type Config struct {
	// DemoLokiImage is the image of the zero click loki deployment that is managed by the operator
	DemoLokiImage string
	// EBPFAgentImage is the image of the eBPF agent that is managed by the operator
	EBPFAgentImage string
	// FlowlogsPipelineImage is the image of the Flowlogs-Pipeline that is managed by the operator
	FlowlogsPipelineImage string
	// WebConsoleImage is the image of the web console that is managed by the operator
	WebConsoleImage string
	// WebConsolePF4Image is the image of the web console, patternfly-4 variant, that is managed by the operator
	WebConsolePF4Image string
	// WebConsolePF5Image is the image of the web console, patternfly-5 variant, that is managed by the operator
	WebConsolePF5Image string
	// EBPFByteCodeImage is the ebpf byte code image used by EBPF Manager
	EBPFByteCodeImage string
	// Operator namespace
	Namespace string
	// Vendor / variant
	Vendor constants.Vendor
	// Static plugin configuration
	StaticPluginConfig          StaticPluginConfig
	DeployOperatorNetworkPolicy *bool
}

// Config of the static plugin.
type StaticPluginConfig struct {
	// Inherit toleration from Subscriptions, for static controller (static plugin); this must refer to the subscription name
	InheritTolerationFromSubscription string
	// CPU request for the static plugin Deployment
	CPURequest string
	// Memory request for the static plugin Deployment
	MemoryRequest string
	// CPU limit for the static plugin Deployment (empty = no limit)
	CPULimit string
	// Memory limit for the static plugin Deployment (empty = no limit)
	MemoryLimit string
}

// ResolveWebConsoleImage selects the web console image appropriate for the cluster version.
// On OpenShift, it returns the image from the last variant whose semver MinVersion is satisfied.
// An empty MinVersion stands for the default match.
func (c *Config) ResolveWebConsoleImage(clusterInfo *cluster.Info) string {
	if clusterInfo != nil && (c.Vendor == constants.VendorOpenShift || c.Vendor == constants.VendorOpenShiftDownstream) {
		for _, v := range []struct {
			image      string
			minVersion string
		}{
			{
				minVersion: "4.22.0",
				image:      c.WebConsoleImage,
			},
			{
				minVersion: "4.15.0",
				image:      c.WebConsolePF5Image,
			},
			{
				image: c.WebConsolePF4Image,
			},
		} {
			if len(v.minVersion) == 0 {
				return v.image
			}
			atLeast, _, err := clusterInfo.IsOpenShiftVersionAtLeast(v.minVersion)
			if err == nil && atLeast {
				return v.image
			}
		}
	}

	return c.WebConsoleImage
}

func (c *Config) Validate() error {
	if c.EBPFAgentImage == "" {
		return errors.New("eBPF agent image env can't be empty")
	}
	if c.FlowlogsPipelineImage == "" {
		return errors.New("flowlogs-pipeline image env can't be empty")
	}
	if c.WebConsoleImage == "" {
		return errors.New("web console image env can't be empty")
	}
	if c.Namespace == "" {
		return errors.New("namespace env can't be empty")
	}
	return nil
}
