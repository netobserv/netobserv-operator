package networkpolicy

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/netpol"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const netpolName = "netobserv"

func buildMainNetworkPolicy(desired *flowslatest.FlowCollector, mgr *manager.Manager) (types.NamespacedName, *networkingv1.NetworkPolicy) {
	ns := helper.GetOperandsNamespace(&desired.Spec, mgr.Config)
	cni, _ := mgr.ClusterInfo.GetCNI()

	name := types.NamespacedName{Name: netpolName, Namespace: ns}

	var shouldInstall bool
	if ns == mgr.Config.Namespace || desired.Spec.NetworkPolicy.Enable == nil {
		// If operator and operands are in the same namespace or if the operands knob is unset,
		// then the operands policy follows the operator knob (else, inconsistent config could lead to operands traffic being blocked).
		// Note that inconsistent config is raised in the validation webhook; no need to raise it again from here
		shouldInstall = flowslatest.ShouldInstallNetworkPolicy(mgr.Config.DeployOperatorNetworkPolicy, cni)
	} else {
		shouldInstall = flowslatest.ShouldInstallNetworkPolicy(desired.Spec.NetworkPolicy.Enable, cni)
	}

	if !shouldInstall {
		return name, nil
	}

	ingress := []networkingv1.NetworkPolicyIngressRule{
		netpol.AllowFromSameNamespace(),
	}
	egress := []networkingv1.NetworkPolicyEgressRule{
		netpol.AllowToSameNamespace(),
		netpol.AllowDNS(mgr.Config.Vendor),
		netpol.AllowToAPIServer(mgr.ClusterInfo, mgr.Config.Vendor),
	}

	if desired.Spec.UseLoki() &&
		desired.Spec.Loki.Mode == flowslatest.LokiModeLokiStack &&
		desired.Spec.Loki.LokiStack.Namespace != "" &&
		desired.Spec.Loki.LokiStack.Namespace != ns {
		egress = append(egress, netpol.AllowToLokiStack(&desired.Spec))
	}

	if desired.Spec.DeploymentModel == flowslatest.DeploymentModelService {
		// Can be counter-intuitive, but only the DeploymentModelService mode needs an explicit rule for host-network (agents are still hostnetwork pods)
		fromNs := helper.GetOperandsNamespace(&desired.Spec, mgr.Config) + constants.EBPFPrivilegedNSSuffix
		ingress = append(ingress, netpol.AllowHostNetworkFlows(mgr.ClusterInfo, mgr.Config.Vendor, &desired.Spec, fromNs))
	}

	if mgr.ClusterInfo.IsOpenShift() {
		ingress = append(ingress, netpol.AllowVendorPrometheusScrape(mgr.Config.Vendor, desired.Spec.Processor.GetMetricsPort(), constants.CPMetricsPort))
		egress = append(egress, netpol.AllowPrometheusQuery(mgr.Config.Vendor))

		if desired.Spec.UseWebConsole() {
			ingress = append(ingress, netpol.AllowFromOpenShiftConsole(&desired.Spec.ConsolePlugin))
		}
	}

	if len(desired.Spec.NetworkPolicy.AdditionalNamespaces) > 0 {
		ingress = append(ingress, networkingv1.NetworkPolicyIngressRule{
			From: []networkingv1.NetworkPolicyPeer{netpol.PeerInNamespaces(desired.Spec.NetworkPolicy.AdditionalNamespaces...)},
		})
		egress = append(egress, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{netpol.PeerInNamespaces(desired.Spec.NetworkPolicy.AdditionalNamespaces...)},
		})
	}

	return name, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      netpolName,
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: ingress,
			Egress:  egress,
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"part-of": constants.OperatorName,
				},
			},
		},
	}
}

func buildPrivilegedNetworkPolicy(desired *flowslatest.FlowCollector, mgr *manager.Manager) (types.NamespacedName, *networkingv1.NetworkPolicy) {
	mainNs := helper.GetOperandsNamespace(&desired.Spec, mgr.Config)
	privNs := mainNs + constants.EBPFPrivilegedNSSuffix
	cni, _ := mgr.ClusterInfo.GetCNI()

	name := types.NamespacedName{Name: netpolName, Namespace: privNs}

	var shouldInstall bool
	if mainNs == mgr.Config.Namespace || desired.Spec.NetworkPolicy.Enable == nil {
		// If operator and operands are in the same namespace or if the operands knob is unset,
		// then the operands policy follows the operator knob (else, inconsistent config could lead to operands traffic being blocked).
		// Note that inconsistent config is raised in the validation webhook; no need to raise it again from here
		shouldInstall = flowslatest.ShouldInstallNetworkPolicy(mgr.Config.DeployOperatorNetworkPolicy, cni)
	} else {
		shouldInstall = flowslatest.ShouldInstallNetworkPolicy(desired.Spec.NetworkPolicy.Enable, cni)
	}

	if !shouldInstall {
		return name, nil
	}

	ingress := []networkingv1.NetworkPolicyIngressRule{}

	if mgr.ClusterInfo.IsOpenShift() {
		ingress = append(ingress, netpol.AllowVendorPrometheusScrape(mgr.Config.Vendor, desired.Spec.Agent.EBPF.GetMetricsPort()))
	}

	// Note that we don't need explicit authorizations for egress as agent pods are on hostnetwork, which allows us to further lock the namespace
	return name, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      netpolName,
			Namespace: privNs,
		},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: ingress,
			Egress:  []networkingv1.NetworkPolicyEgressRule{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}
}
