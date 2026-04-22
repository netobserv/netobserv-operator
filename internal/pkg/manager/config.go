package manager

import (
	"errors"
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
)

// ConsolePluginImageVariant maps an OCP version threshold to a console plugin image.
// When multiple variants match, the one with the highest MinVersion wins.
type ConsolePluginImageVariant struct {
	Image      string
	MinVersion string // minimum OCP version (inclusive), e.g. "4.15.0"
}

// Config of the operator.
type Config struct {
	// DemoLokiImage is the image of the zero click loki deployment that is managed by the operator
	DemoLokiImage string
	// EBPFAgentImage is the image of the eBPF agent that is managed by the operator
	EBPFAgentImage string
	// FlowlogsPipelineImage is the image of the Flowlogs-Pipeline that is managed by the operator
	FlowlogsPipelineImage string
	// ConsolePluginImageVariants lists version-specific console plugin images, sorted by MinVersion ascending
	// (semver for versioned entries). ResolveConsolePluginImage picks the highest matching MinVersion on OpenShift.
	// For non-OpenShift, it uses the "default" sentinel entry if present; otherwise the first versioned (non-default)
	// entry, or the first slice element as last resort. Put the baseline image first and include default= in the
	// env string when multiple variants exist so resolution is unambiguous.
	ConsolePluginImageVariants []ConsolePluginImageVariant
	// EBPFByteCodeImage is the ebpf byte code image used by EBPF Manager
	EBPFByteCodeImage string
	// Default namespace
	Namespace string
	// Release kind is either upstream or downstream
	DownstreamDeployment bool
}

// ParseConsolePluginImages parses console plugin image configuration.
// Accepted formats:
//   - Single image:       "registry/plugin:tag"  (used for all clusters)
//   - Version-keyed list: "default=img:pf4;4.15.0=img:pf5;4.22.0=img:pf6"
//     The "default" key is the fallback for non-OpenShift or unmatched OCP versions.
//     Version-keyed entries should be sorted ascending by minVersion.
func (cfg *Config) ParseConsolePluginImages(raw string) error {
	var variants []ConsolePluginImageVariant
	if strings.TrimSpace(raw) == "" {
		cfg.ConsolePluginImageVariants = nil
		return nil
	}

	// Single image (no "=" at all): treat as default-only
	if !strings.Contains(raw, "=") {
		cfg.ConsolePluginImageVariants = []ConsolePluginImageVariant{
			{MinVersion: "default", Image: strings.TrimSpace(raw)},
		}
		return nil
	}

	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		eqIdx := strings.Index(entry, "=")
		if eqIdx <= 0 || eqIdx >= len(entry)-1 {
			return fmt.Errorf("invalid console plugin image entry %q: expected format minVersion=image or default=image", entry)
		}
		variants = append(variants, ConsolePluginImageVariant{
			MinVersion: entry[:eqIdx],
			Image:      entry[eqIdx+1:],
		})
	}
	cfg.ConsolePluginImageVariants = variants
	return nil
}

// ResolveConsolePluginImage selects the console plugin image appropriate for the cluster's OCP version.
// On OpenShift, it returns the image from the last variant whose semver MinVersion is satisfied.
// A "default" entry is used when no version-specific entry matches, or on non-OpenShift if present;
// otherwise on non-OpenShift the first non-default entry applies (then the first element as last resort).
func (cfg *Config) ResolveConsolePluginImage(clusterInfo *cluster.Info) (string, error) {
	if clusterInfo == nil {
		return "", errors.New("cluster info is nil")
	}
	if len(cfg.ConsolePluginImageVariants) == 0 {
		return "", fmt.Errorf("no console plugin image variants configured")
	}

	var defaultImage string
	for _, v := range cfg.ConsolePluginImageVariants {
		if v.MinVersion == "default" {
			defaultImage = v.Image
			break
		}
	}

	if !clusterInfo.IsOpenShift() {
		if defaultImage != "" {
			return defaultImage, nil
		}
		// No explicit default: use first versioned entry (baseline image)
		for _, v := range cfg.ConsolePluginImageVariants {
			if v.MinVersion != "default" {
				return v.Image, nil
			}
		}
		return cfg.ConsolePluginImageVariants[0].Image, nil
	}

	var result string
	for _, v := range cfg.ConsolePluginImageVariants {
		if v.MinVersion == "default" {
			continue
		}
		atLeast, _, err := clusterInfo.IsOpenShiftVersionAtLeast(v.MinVersion)
		if err == nil && atLeast {
			result = v.Image
		}
	}
	if result != "" {
		return result, nil
	}
	if defaultImage != "" {
		return defaultImage, nil
	}
	ocpVersion, _ := clusterInfo.GetOpenShiftVersion()
	return "", fmt.Errorf("no console plugin image variant matches OpenShift version %s (minimum configured: %s)",
		ocpVersion, cfg.ConsolePluginImageVariants[0].MinVersion)
}

func (cfg *Config) Validate() error {
	if cfg.EBPFAgentImage == "" {
		return errors.New("eBPF agent image argument can't be empty")
	}
	if cfg.FlowlogsPipelineImage == "" {
		return errors.New("flowlogs-pipeline image argument can't be empty")
	}
	if len(cfg.ConsolePluginImageVariants) == 0 {
		return errors.New("console plugin images can't be empty")
	}
	var prev *semver.Version
	seenDefault := false
	for i, v := range cfg.ConsolePluginImageVariants {
		if v.Image == "" {
			return fmt.Errorf("console plugin image variant %d has empty image", i)
		}
		if v.MinVersion == "" {
			return fmt.Errorf("console plugin image variant %d has empty MinVersion", i)
		}
		if v.MinVersion == "default" {
			if seenDefault {
				return fmt.Errorf("console plugin image variant %d: duplicate MinVersion %q (ResolveConsolePluginImage would pick the first default only)", i, v.MinVersion)
			}
			seenDefault = true
			continue
		}
		ver, err := semver.NewVersion(v.MinVersion)
		if err != nil {
			return fmt.Errorf("console plugin image variant %d has invalid MinVersion %q: %w", i, v.MinVersion, err)
		}
		if prev != nil && !prev.LessThan(*ver) {
			return fmt.Errorf("console plugin image variant %d MinVersion %q must be strictly greater than previous %q", i, v.MinVersion, prev.String())
		}
		prev = ver
	}
	if cfg.Namespace == "" {
		return errors.New("namespace argument can't be empty")
	}
	return nil
}
