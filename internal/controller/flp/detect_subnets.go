package flp

import (
	"context"
	"errors"
	"fmt"
	"net"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *Reconciler) getOpenShiftSubnets(ctx context.Context) ([]flowslatest.SubnetLabel, error) {
	if !r.mgr.ClusterInfo.HasCNO() {
		return nil, nil
	}
	var errs []error
	var svcMachineCIDRs []*net.IPNet

	pods, services, extIPs, err := readNetworkConfig(ctx, r)
	if err != nil {
		errs = append(errs, err)
	}
	for _, strCIDR := range services {
		if _, parsed, err := net.ParseCIDR(strCIDR); err == nil {
			svcMachineCIDRs = append(svcMachineCIDRs, parsed)
		}
	}

	machines, err := readClusterConfig(ctx, r)
	if err != nil {
		errs = append(errs, err)
	}
	for _, strCIDR := range machines {
		if _, parsed, err := net.ParseCIDR(strCIDR); err == nil {
			svcMachineCIDRs = append(svcMachineCIDRs, parsed)
		}
	}

	// API server
	if apiserverIPs, err := cluster.GetAPIServerEndpointIPs(ctx, r, r.mgr.ClusterInfo); err == nil {
		// Check if this isn't already an IP covered in Services or Machines subnets
		for _, ip := range apiserverIPs {
			if parsed := net.ParseIP(ip); parsed != nil {
				var alreadyCovered bool
				for _, cidr := range svcMachineCIDRs {
					if cidr.Contains(parsed) {
						alreadyCovered = true
						break
					}
				}
				if !alreadyCovered {
					cidr := helper.IPToCIDR(ip)
					services = append(services, cidr)
				}
			}
		}
	} else {
		errs = append(errs, fmt.Errorf("can't get API server endpoint IPs: %w", err))
	}

	// Additional OVN subnets
	moreMachines, err := readNetworkOperatorConfig(ctx, r)
	if err != nil {
		errs = append(errs, err)
	}
	machines = append(machines, moreMachines...)

	var subnets []flowslatest.SubnetLabel
	if len(machines) > 0 {
		subnets = append(subnets, flowslatest.SubnetLabel{
			Name:  "Machines",
			CIDRs: machines,
		})
	}
	if len(pods) > 0 {
		subnets = append(subnets, flowslatest.SubnetLabel{
			Name:  "Pods",
			CIDRs: pods,
		})
	}
	if len(services) > 0 {
		subnets = append(subnets, flowslatest.SubnetLabel{
			Name:  "Services",
			CIDRs: services,
		})
	}
	if len(extIPs) > 0 {
		subnets = append(subnets, flowslatest.SubnetLabel{
			Name:  "EXT:ExternalIP",
			CIDRs: extIPs,
		})
	}
	return subnets, errors.Join(errs...)
}

func readNetworkConfig(ctx context.Context, cl client.Client) ([]string, []string, []string, error) {
	// Pods and Services subnets are found in CNO config
	var pods, services, extIPs []string
	network := &configv1.Network{}
	if err := cl.Get(ctx, types.NamespacedName{Name: "cluster"}, network); err != nil {
		return nil, nil, nil, fmt.Errorf("can't get Network (config) information: %w", err)
	}
	for _, podsNet := range network.Spec.ClusterNetwork {
		pods = append(pods, podsNet.CIDR)
	}
	services = network.Spec.ServiceNetwork
	if network.Spec.ExternalIP != nil && len(network.Spec.ExternalIP.AutoAssignCIDRs) > 0 {
		extIPs = network.Spec.ExternalIP.AutoAssignCIDRs
	}
	return pods, services, extIPs, nil
}

func readClusterConfig(ctx context.Context, cl client.Client) ([]string, error) {
	// Nodes subnet found in CM cluster-config-v1 (kube-system)
	cm := &corev1.ConfigMap{}
	if err := cl.Get(ctx, types.NamespacedName{Name: "cluster-config-v1", Namespace: "kube-system"}, cm); err != nil {
		return nil, fmt.Errorf(`can't read "cluster-config-v1" ConfigMap: %w`, err)
	}
	return readMachineFromConfig(cm)
}

func readMachineFromConfig(cm *corev1.ConfigMap) ([]string, error) {
	type ClusterConfig struct {
		Networking struct {
			MachineNetwork []struct {
				CIDR string `yaml:"cidr"`
			} `yaml:"machineNetwork"`
		} `yaml:"networking"`
	}

	var rawConfig string
	var ok bool
	if rawConfig, ok = cm.Data["install-config"]; !ok {
		return nil, fmt.Errorf(`can't find key "install-config" in "cluster-config-v1" ConfigMap`)
	}
	var config ClusterConfig
	if err := yaml.Unmarshal([]byte(rawConfig), &config); err != nil {
		return nil, fmt.Errorf(`can't deserialize content of "cluster-config-v1" ConfigMap: %w`, err)
	}

	var cidrs []string
	for _, cidr := range config.Networking.MachineNetwork {
		cidrs = append(cidrs, cidr.CIDR)
	}

	return cidrs, nil
}

func readNetworkOperatorConfig(ctx context.Context, cl client.Client) ([]string, error) {
	// Additional OVN subnets: https://github.com/openshift/cluster-network-operator/blob/fda7a9f07ab6f78d032d310cdd77f21d04f1289a/pkg/network/ovn_kubernetes.go#L76-L77
	var machines []string
	networkOp := &operatorv1.Network{}
	if err := cl.Get(ctx, types.NamespacedName{Name: "cluster"}, networkOp); err != nil {
		return nil, fmt.Errorf("can't get Network (operator) information: %w", err)
	}
	internalSubnet := "100.64.0.0/16"
	transitSwitchSubnet := "100.88.0.0/16"
	masqueradeSubnet := "169.254.0.0/17"
	ovnk := networkOp.Spec.DefaultNetwork.OVNKubernetesConfig
	if ovnk != nil {
		if ovnk.V4InternalSubnet != "" {
			internalSubnet = ovnk.V4InternalSubnet
		}
		if ovnk.IPv4 != nil && ovnk.IPv4.InternalTransitSwitchSubnet != "" {
			transitSwitchSubnet = ovnk.IPv4.InternalTransitSwitchSubnet
		}
		if ovnk.GatewayConfig != nil && ovnk.GatewayConfig.IPv4.InternalMasqueradeSubnet != "" {
			masqueradeSubnet = ovnk.GatewayConfig.IPv4.InternalMasqueradeSubnet
		}
	}
	machines = append(machines, internalSubnet)
	machines = append(machines, transitSwitchSubnet)
	machines = append(machines, masqueradeSubnet)
	return machines, nil
}
