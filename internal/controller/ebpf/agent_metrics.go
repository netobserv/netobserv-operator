package ebpf

import (
	"context"
	"fmt"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func (c *AgentController) reconcileMetricsService(ctx context.Context, target *flowslatest.FlowCollectorEBPF) error {
	report := helper.NewChangeReport("EBPF Agent prometheus service")
	defer report.LogIfNeeded(ctx)

	if !target.IsEBPFMetricsEnabled() {
		c.Managed.TryDelete(ctx, c.promSvc)
		if c.ClusterInfo.HasSvcMonitor() {
			c.Managed.TryDelete(ctx, c.serviceMonitor)
		}
		if c.ClusterInfo.HasPromRule() {
			c.Managed.TryDelete(ctx, c.prometheusRule)
		}
		return nil
	}

	if err := c.ReconcileService(ctx, c.promSvc, c.promService(target), &report); err != nil {
		return err
	}
	if c.ClusterInfo.HasSvcMonitor() {
		serviceMonitor := c.promServiceMonitoring(target, c.ClusterInfo.HasPromServiceDiscoveryRole())
		if err := reconcilers.GenericReconcile(ctx, c.Managed, &c.Client, c.serviceMonitor,
			serviceMonitor, &report, helper.ServiceMonitorChanged); err != nil {
			return err
		}
	}

	if c.ClusterInfo.HasPromRule() {
		promRules := c.agentPrometheusRule(target)
		if err := reconcilers.GenericReconcile(ctx, c.Managed, &c.Client, c.prometheusRule, promRules, &report, helper.PrometheusRuleChanged); err != nil {
			return err
		}
	}
	return nil
}

func (c *AgentController) promService(target *flowslatest.FlowCollectorEBPF) *corev1.Service {
	port := target.GetMetricsPort()
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.EBPFAgentMetricsSvcName,
			Namespace: c.PrivilegedNamespace(),
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     constants.EBPFAgentName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": constants.EBPFAgentName,
			},
			Ports: []corev1.ServicePort{{
				Name:       "metrics",
				Port:       port,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(port),
			}},
		},
	}
	if target.Metrics.Server.TLS.Type == flowslatest.TLSAuto {
		svc.ObjectMeta.Annotations = map[string]string{
			constants.OpenShiftCertificateAnnotation: constants.EBPFAgentMetricsSvcName,
		}
	}
	return &svc
}

func (c *AgentController) promServiceMonitoring(target *flowslatest.FlowCollectorEBPF, useEndpointSlices bool) *monitoringv1.ServiceMonitor {
	serverName := fmt.Sprintf("%s.%s.svc", constants.EBPFAgentMetricsSvcName, c.PrivilegedNamespace())
	scheme, smTLS := helper.GetServiceMonitorTLSConfig(&target.Metrics.Server.TLS, serverName, c.IsDownstream)
	var sdRole *monitoringv1.ServiceDiscoveryRole
	if useEndpointSlices {
		sdRole = ptr.To(monitoringv1.EndpointSliceRole)
	}
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.EBPFAgentMetricsSvcMonitoringName,
			Namespace: c.PrivilegedNamespace(),
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     constants.EBPFAgentName,
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			ServiceDiscoveryRole: sdRole,
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:      "metrics",
					Interval:  "30s",
					Scheme:    &scheme,
					TLSConfig: smTLS,
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{
					c.PrivilegedNamespace(),
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constants.EBPFAgentName,
				},
			},
		},
	}
}

func (c *AgentController) agentPrometheusRule(target *flowslatest.FlowCollectorEBPF) *monitoringv1.PrometheusRule {
	rules := []monitoringv1.Rule{}
	d := monitoringv1.Duration("10m")

	// EBPF hashmap table is full Not receiving any new flows
	if shouldAddAlert(flowslatest.AlertDroppedFlows, target.Metrics.DisableAlerts) {

		rules = append(rules, monitoringv1.Rule{
			Alert: string(flowslatest.AlertDroppedFlows),
			Annotations: map[string]string{
				"description": "NetObserv eBPF agent is missing packets or flows. The metric netobserv_agent_dropped_flows_total provides more information on the cause. Possible reasons are the BPF hashmap being busy or full, or the capacity limiter being triggered. This may be worked around by increasing cacheMaxFlows value in Flowcollector resource.",
				"summary":     "NetObserv eBPF agent is missing packets or flows",
			},
			Expr: intstr.FromString(fmt.Sprintf("sum(rate(netobserv_agent_dropped_flows_total[1m])) > %d", droppedFlowsAlertThreshold)),
			For:  &d,
			Labels: map[string]string{
				"severity": "warning",
				"app":      "netobserv",
			},
		})
	}

	prometheusRuleObject := monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.EBPFAgentPromAlertRule,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     constants.EBPFAgentName,
			},
			Namespace: c.PrivilegedNamespace(),
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "NetobservEBPFAgentAlerts",
					Rules: rules,
				},
			},
		},
	}
	return &prometheusRuleObject
}

func shouldAddAlert(name flowslatest.EBPFAgentAlert, disabledList []flowslatest.EBPFAgentAlert) bool {
	for _, disabledAlert := range disabledList {
		if name == disabledAlert {
			return false
		}
	}
	return true
}
