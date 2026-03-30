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
	// ConsolePluginImageVariants lists version-specific console plugin images, sorted by MinVersion ascending.
	// The variant with the highest MinVersion that is <= the cluster OCP version is selected.
	// On non-OpenShift or unknown version, the last entry (highest MinVersion) is used as default.
	ConsolePluginImageVariants []ConsolePluginImageVariant
	// EBPFByteCodeImage is the ebpf byte code image used by EBPF Manager
	EBPFByteCodeImage string
	// Default namespace
	Namespace string
	// Release kind is either upstream or downstream
	DownstreamDeployment bool
}

// ParseConsolePluginImages parses a semicolon-separated list of "minVersion=image" entries
// and populates ConsolePluginImageVariants. Entries should be sorted ascending by minVersion.
// Example: "4.14.0=registry/plugin:v1-pf4;4.15.0=registry/plugin:v1-pf5;4.22.0=registry/plugin:v1"
func (cfg *Config) ParseConsolePluginImages(raw string) error {
	var variants []ConsolePluginImageVariant
	if strings.TrimSpace(raw) == "" {
		cfg.ConsolePluginImageVariants = nil
		return nil
	}
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		eqIdx := strings.Index(entry, "=")
		if eqIdx <= 0 || eqIdx >= len(entry)-1 {
			return fmt.Errorf("invalid console plugin image entry %q: expected format minVersion=image", entry)
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
// It iterates ConsolePluginImageVariants (sorted ascending by MinVersion) and returns the image from
// the last variant whose MinVersion is satisfied. On non-OpenShift clusters, the last entry
// (highest MinVersion, most current) is used as default. Returns an error if no variant matches.
func (cfg *Config) ResolveConsolePluginImage(clusterInfo *cluster.Info) (string, error) {
	if len(cfg.ConsolePluginImageVariants) == 0 {
		return "", fmt.Errorf("no console plugin image variants configured")
	}
	if !clusterInfo.IsOpenShift() {
		return cfg.ConsolePluginImageVariants[len(cfg.ConsolePluginImageVariants)-1].Image, nil
	}
	var result string
	for _, v := range cfg.ConsolePluginImageVariants {
		atLeast, _, err := clusterInfo.IsOpenShiftVersionAtLeast(v.MinVersion)
		if err == nil && atLeast {
			result = v.Image
		}
	}
	if result == "" {
		ocpVersion, _ := clusterInfo.GetOpenShiftVersion()
		return "", fmt.Errorf("no console plugin image variant matches OpenShift version %s (minimum configured: %s)",
			ocpVersion, cfg.ConsolePluginImageVariants[0].MinVersion)
	}
	return result, nil
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
	for i, v := range cfg.ConsolePluginImageVariants {
		if v.Image == "" {
			return fmt.Errorf("console plugin image variant %d has empty image", i)
		}
		if v.MinVersion == "" {
			return fmt.Errorf("console plugin image variant %d has empty MinVersion", i)
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
