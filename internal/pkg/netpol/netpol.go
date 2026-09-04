package netpol

import (
	"os"
	"strconv"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func ovnHostNetworkPeer(vendor constants.Vendor) networkingv1.NetworkPolicyPeer {
	var annots map[string]string
	switch vendor {
	case constants.VendorOpenShift, constants.VendorOpenShiftDownstream:
		annots = map[string]string{
			"policy-group.network.openshift.io/host-network": "",
		}
	default:
		annots = map[string]string{
			"kubernetes.io/metadata.name": "ovn-host-network",
		}
	}
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: annots,
		},
	}
}

func PeerInNamespace(ns string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"kubernetes.io/metadata.name": ns},
		},
	}
}

func PeerInNamespaces(ns ...string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpIn,
				Values:   ns,
			}},
		},
	}
}

func AllowFromSameNamespace() networkingv1.NetworkPolicyIngressRule {
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}},
	}
}

func AllowToSameNamespace() networkingv1.NetworkPolicyEgressRule {
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{{PodSelector: &metav1.LabelSelector{}}},
	}
}

func AllowFromOpenShiftConsole(desired *flowslatest.FlowCollectorConsolePlugin) networkingv1.NetworkPolicyIngressRule {
	advanced := helper.GetAdvancedPluginConfig(desired.Advanced)
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			PeerInNamespace(constants.OpenShiftConsoleNamespace),
		},
		Ports: []networkingv1.NetworkPolicyPort{{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt32(*advanced.Port)),
		}},
	}
}

func AllowToLokiStack(spec *flowslatest.FlowCollectorSpec) networkingv1.NetworkPolicyEgressRule {
	peer := PeerInNamespace(spec.Loki.LokiStack.Namespace)
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{peer},
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(8080)),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt(3100)),
			},
		},
	}
}

func AllowVendorPrometheusScrape(vendor constants.Vendor, ports ...int32) networkingv1.NetworkPolicyIngressRule {
	// Assume Prometheus is in netobserv namespace, unless vendor is OpenShift
	peer := networkingv1.NetworkPolicyPeer{PodSelector: &metav1.LabelSelector{}}
	switch vendor {
	case constants.VendorOpenShift:
		peer = PeerInNamespace(constants.OpenShiftUWMonitoringNamespace)
	case constants.VendorOpenShiftDownstream:
		peer = PeerInNamespace(constants.OpenShiftMonitoringNamespace)
	}
	portsSpec := []networkingv1.NetworkPolicyPort{}
	for _, p := range ports {
		portsSpec = append(portsSpec, networkingv1.NetworkPolicyPort{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt32(p)),
		})
	}
	return networkingv1.NetworkPolicyIngressRule{
		From:  []networkingv1.NetworkPolicyPeer{peer},
		Ports: portsSpec,
	}
}

func AllowPrometheusQuery(vendor constants.Vendor) networkingv1.NetworkPolicyEgressRule {
	// Assume Prometheus is in netobserv namespace, unless vendor is OpenShift
	peer := networkingv1.NetworkPolicyPeer{PodSelector: &metav1.LabelSelector{}}
	switch vendor {
	case constants.VendorOpenShift, constants.VendorOpenShiftDownstream:
		peer = PeerInNamespace(constants.OpenShiftMonitoringNamespace)
	}
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{peer},
	}
}

func AllowToAPIServer(clusterInfo *cluster.Info, vendor constants.Vendor) networkingv1.NetworkPolicyEgressRule {
	var peer networkingv1.NetworkPolicyPeer
	var port int
	switch vendor {
	case constants.VendorOpenShift, constants.VendorOpenShiftDownstream:
		peer = PeerInNamespaces(constants.OpenShiftAPIServerNamespace, constants.OpenShiftKubeAPIServerNamespace)
		port = constants.K8sAPIServerPort
	default:
		peer = PeerInNamespace(constants.KubeSystemNamespace) // Just as a fallback in case KUBERNETES_SERVICE_HOST isn't set, which shouldn't happen
		port = 443
		if ip := os.Getenv("KUBERNETES_SERVICE_HOST"); ip != "" {
			cidr := helper.IPToCIDR(ip)
			if cidr != "" {
				peer = networkingv1.NetworkPolicyPeer{
					IPBlock: &networkingv1.IPBlock{CIDR: cidr},
				}
			}
		}
		if portStr := os.Getenv("KUBERNETES_SERVICE_PORT"); portStr != "" {
			if p, err := strconv.ParseInt(portStr, 10, 32); err == nil {
				port = int(p)
			}
		}
	}
	// TO CHECK	peer.PodSelector = &metav1.LabelSelector{} // see https://issues.redhat.com/browse/OSDOCS-14395 / needed for apiserver
	peers := []networkingv1.NetworkPolicyPeer{peer}
	ports := []networkingv1.NetworkPolicyPort{{
		Protocol: ptr.To(corev1.ProtocolTCP),
		Port:     ptr.To(intstr.FromInt(port)),
	}}

	// Allow fetching from external apiserver (e.g. HyperShift and other external control planes)
	// The kubernetes service may redirect to external endpoints on port 6443
	for _, ip := range clusterInfo.GetAPIServerIPs() {
		cidr := helper.IPToCIDR(ip)
		if cidr != "" {
			peers = append(peers, networkingv1.NetworkPolicyPeer{
				IPBlock: &networkingv1.IPBlock{CIDR: cidr},
			})
		}
	}
	for _, p := range clusterInfo.GetAPIServerPorts() {
		if p != int32(port) {
			ports = append(ports, networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt32(p)),
			})
		}
	}
	return networkingv1.NetworkPolicyEgressRule{
		To:    peers,
		Ports: ports,
	}
}

func AllowToWebhooks(clusterInfo *cluster.Info, vendor constants.Vendor) networkingv1.NetworkPolicyIngressRule {
	var peer networkingv1.NetworkPolicyPeer
	cni, _ := clusterInfo.GetCNI()
	if cni == flowslatest.OVNKubernetes {
		peer = ovnHostNetworkPeer(vendor)
	} else {
		peer = PeerInNamespace(constants.KubeSystemNamespace)
	}
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{peer},
		Ports: []networkingv1.NetworkPolicyPort{{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt32(constants.WebhookPort)),
		}},
	}
}

func AllowHostNetworkFlows(clusterInfo *cluster.Info, vendor constants.Vendor, spec *flowslatest.FlowCollectorSpec, fromNamespace string) networkingv1.NetworkPolicyIngressRule {
	var peer networkingv1.NetworkPolicyPeer
	cni, _ := clusterInfo.GetCNI()
	if cni == flowslatest.OVNKubernetes {
		peer = ovnHostNetworkPeer(vendor)
	} else {
		peer = PeerInNamespace(fromNamespace)
	}
	advanced := helper.GetAdvancedProcessorConfig(spec)
	return networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{peer},
		Ports: []networkingv1.NetworkPolicyPort{{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt32(*advanced.Port)),
		}},
	}
}

func AllowDNS(vendor constants.Vendor) networkingv1.NetworkPolicyEgressRule {
	var peer networkingv1.NetworkPolicyPeer
	switch vendor {
	case constants.VendorOpenShift, constants.VendorOpenShiftDownstream:
		peer = PeerInNamespace(constants.OpenShiftDNSNamespace)
	default:
		peer = PeerInNamespace(constants.KubeSystemNamespace)
	}
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{peer},
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: ptr.To(corev1.ProtocolUDP),
				Port:     ptr.To(intstr.FromString("dns")),
			},
			{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromString("dns-tcp")),
			},
		},
	}
}
