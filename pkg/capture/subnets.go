package capture

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// ParseMachineNetworkCIDRs extracts machine network CIDRs from the
// install-config YAML stored in OpenShift's cluster-config-v1 ConfigMap.
// Kubernetes client and platform-specific network discovery remain outside the
// shared package.
func ParseMachineNetworkCIDRs(installConfig string) ([]string, error) {
	var config struct {
		Networking struct {
			MachineNetwork []struct {
				CIDR string `yaml:"cidr"`
			} `yaml:"machineNetwork"`
		} `yaml:"networking"`
	}
	if err := yaml.Unmarshal([]byte(installConfig), &config); err != nil {
		return nil, fmt.Errorf("cannot decode install-config: %w", err)
	}

	cidrs := make([]string, 0, len(config.Networking.MachineNetwork))
	for _, network := range config.Networking.MachineNetwork {
		if network.CIDR != "" {
			cidrs = append(cidrs, network.CIDR)
		}
	}
	return cidrs, nil
}
