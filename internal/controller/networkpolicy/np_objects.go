package networkpolicy

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const netpolName = "netobserv"

func peerInNamespace(ns string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"kubernetes.io/metadata.name": ns},
		},
	}
}

func peerInNamespaces(ns []string) networkingv1.NetworkPolicyPeer {
	return networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpIn,
				Values:   ns,
			}},
		},
		PodSelector: &metav1.LabelSelector{}, // see https://issues.redhat.com/browse/OSDOCS-14395 / needed for apiserver
	}
}

func addAllowedNamespaces(np *networkingv1.NetworkPolicy, in, out []string) {
	if len(in) > 0 {
		np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{peerInNamespaces(in)},
		})
	}
	if len(out) > 0 {
		np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{peerInNamespaces(out)},
		})
	}
}

func buildMainNetworkPolicy(desired *flowslatest.FlowCollector, mgr *manager.Manager, cni flowslatest.NetworkType, apiServerIPs []string) (types.NamespacedName, *networkingv1.NetworkPolicy) {
	ns := desired.Spec.GetNamespace()

	name := types.NamespacedName{Name: netpolName, Namespace: ns}
	if cni == flowslatest.OpenShiftSDN || !desired.Spec.DeployNetworkPolicy(cni != "") {
		return name, nil
	}

	np := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      netpolName,
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{
						// Setting empty namespace selector will authorize every pod from the same namespace
						PodSelector: &metav1.LabelSelector{},
					},
				},
			}},
			Egress: []networkingv1.NetworkPolicyEgressRule{{
				To: []networkingv1.NetworkPolicyPeer{
					{
						// Setting empty namespace selector will authorize every pod from the same namespace
						PodSelector: &metav1.LabelSelector{},
					},
				},
			}},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					// TODO: remove this restiction when LokiStack implements network policy by default
					"part-of": constants.OperatorName,
				},
			},
		},
	}
	allowedNamespacesIn := []string{}
	allowedNamespacesOut := []string{}

	if desired.Spec.UseLoki() &&
		desired.Spec.Loki.Mode == flowslatest.LokiModeLokiStack &&
		desired.Spec.Loki.LokiStack.Namespace != "" &&
		desired.Spec.Loki.LokiStack.Namespace != ns {
		allowedNamespacesIn = append(allowedNamespacesIn, desired.Spec.Loki.LokiStack.Namespace)
		allowedNamespacesOut = append(allowedNamespacesOut, desired.Spec.Loki.LokiStack.Namespace)
	}

	if mgr.ClusterInfo.IsOpenShift() {
		allowedNamespacesOut = append(allowedNamespacesOut, constants.DNSNamespace)
		allowedNamespacesOut = append(allowedNamespacesOut, constants.MonitoringNamespace)
		if mgr.Config.DownstreamDeployment {
			allowedNamespacesIn = append(allowedNamespacesIn, constants.MonitoringNamespace)
		} else {
			allowedNamespacesIn = append(allowedNamespacesIn, constants.UWMonitoringNamespace)
		}

		if desired.Spec.UseWebConsole() && mgr.ClusterInfo.HasConsolePlugin() {
			advanced := helper.GetAdvancedPluginConfig(desired.Spec.ConsolePlugin.Advanced)
			np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				From: []networkingv1.NetworkPolicyPeer{
					peerInNamespace(constants.ConsoleNamespace),
				},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(*advanced.Port)),
				}},
			})
		}
		// Allow apiserver/host
		hostNetworkPorts := []networkingv1.NetworkPolicyPort{{
			Protocol: ptr.To(corev1.ProtocolTCP),
			Port:     ptr.To(intstr.FromInt32(constants.WebhookPort)),
		}}
		if desired.Spec.DeploymentModel == flowslatest.DeploymentModelService {
			// Can be counter-intuitive, but only the DeploymentModelService mode needs an explicit rule for host-network (agents are still hostnetwork pods)
			advanced := helper.GetAdvancedProcessorConfig(&desired.Spec)
			hostNetworkPorts = append(hostNetworkPorts, networkingv1.NetworkPolicyPort{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt32(*advanced.Port)),
			})
		}
		np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{
				{
					NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
						"policy-group.network.openshift.io/host-network": "",
					}},
				},
			},
			Ports: hostNetworkPorts,
		})

		// Allow fetching from in-cluster apiserver namespaces
		np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				peerInNamespaces([]string{constants.OpenShiftAPIServerNamespace, constants.OpenShiftKubeAPIServerNamespace}),
			},
			Ports: []networkingv1.NetworkPolicyPort{{
				Protocol: ptr.To(corev1.ProtocolTCP),
				Port:     ptr.To(intstr.FromInt32(constants.K8sAPIServerPort)),
			}},
		})

		// Allow fetching from external apiserver (HyperShift and other external control planes)
		// The kubernetes service may redirect to external endpoints on port 6443
		if len(apiServerIPs) > 0 {
			// Build a single egress rule with multiple IP peers
			peers := []networkingv1.NetworkPolicyPeer{}
			for _, ip := range apiServerIPs {
				cidr := helper.IPToCIDR(ip)
				if cidr != "" {
					peers = append(peers, networkingv1.NetworkPolicyPeer{
						IPBlock: &networkingv1.IPBlock{
							CIDR: cidr,
						},
					})
				}
			}
			np.Spec.Egress = append(np.Spec.Egress, networkingv1.NetworkPolicyEgressRule{
				To: peers,
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(constants.K8sAPIServerPort)),
				}},
			})
		}
	} else {
		// Not OpenShift
		// Allow fetching from apiserver / kube-system
		allowedNamespacesOut = append(allowedNamespacesOut, constants.KubeSystemNamespace)

		if cni == flowslatest.OVNKubernetes && desired.Spec.DeploymentModel == flowslatest.DeploymentModelService {
			// Upstream OVN-K: allow host-network on processor port (from agents)
			// Can be counter-intuitive, but only the DeploymentModelService mode needs an explicit rule for host-network (agents are still hostnetwork pods)
			advanced := helper.GetAdvancedProcessorConfig(&desired.Spec)
			np.Spec.Ingress = append(np.Spec.Ingress, networkingv1.NetworkPolicyIngressRule{
				From: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{
							"kubernetes.io/metadata.name": "ovn-host-network",
						}},
					},
				},
				Ports: []networkingv1.NetworkPolicyPort{{
					Protocol: ptr.To(corev1.ProtocolTCP),
					Port:     ptr.To(intstr.FromInt32(*advanced.Port)),
				}},
			})
		}
	}

	allowedNamespacesIn = append(allowedNamespacesIn, desired.Spec.NetworkPolicy.AdditionalNamespaces...)
	allowedNamespacesOut = append(allowedNamespacesOut, desired.Spec.NetworkPolicy.AdditionalNamespaces...)

	addAllowedNamespaces(&np, allowedNamespacesIn, allowedNamespacesOut)

	return name, &np
}

func buildPrivilegedNetworkPolicy(desired *flowslatest.FlowCollector, mgr *manager.Manager, cni flowslatest.NetworkType) (types.NamespacedName, *networkingv1.NetworkPolicy) {
	mainNs := desired.Spec.GetNamespace()
	privNs := mainNs + constants.EBPFPrivilegedNSSuffix

	name := types.NamespacedName{Name: netpolName, Namespace: privNs}
	if cni == flowslatest.OpenShiftSDN || !desired.Spec.DeployNetworkPolicy(cni != "") {
		return name, nil
	}

	np := networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      netpolName,
			Namespace: privNs,
		},
		Spec: networkingv1.NetworkPolicySpec{
			// Start with no allowed traffic
			Ingress: []networkingv1.NetworkPolicyIngressRule{},
			Egress:  []networkingv1.NetworkPolicyEgressRule{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// Note that we don't need explicit authorizations for egress as agent pods are on hostnetwork, which allows us to further lock the namespace
	allowedNamespacesIn := []string{}

	if mgr.ClusterInfo.IsOpenShift() {
		if mgr.Config.DownstreamDeployment {
			allowedNamespacesIn = append(allowedNamespacesIn, constants.MonitoringNamespace)
		} else {
			allowedNamespacesIn = append(allowedNamespacesIn, constants.UWMonitoringNamespace)
		}
	}

	addAllowedNamespaces(&np, allowedNamespacesIn, nil)

	return name, &np
}
