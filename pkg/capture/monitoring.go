package capture

import (
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const metricsResourceName = "netobserv-cli"

// BuildMetricsResources returns the typed Kubernetes resources needed by the
// standalone CLI to expose a metrics capture to OpenShift monitoring.
// Operator-managed FlowCaptures use the preinstalled expose-metrics role
// through a namespaced RoleBinding instead of creating these cluster-wide
// resources.
func BuildMetricsResources(namespace string) []runtime.Object {
	return []runtime.Object{
		BuildMetricsClusterRole(),
		BuildMetricsClusterRoleBinding(),
		BuildMetricsServiceMonitor(namespace),
		BuildMetricsService(namespace),
	}
}

func BuildMetricsClusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRole"},
		ObjectMeta: metav1.ObjectMeta{Name: metricsResourceName + "-metrics"},
		Rules: []rbacv1.PolicyRule{
			{APIGroups: []string{""}, Resources: []string{"pods", "services", "endpoints"}, Verbs: []string{"get", "list", "watch"}},
			{NonResourceURLs: []string{"/metrics"}, Verbs: []string{"get"}},
		},
	}
}

func BuildMetricsClusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "ClusterRoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: metricsResourceName + "-metrics"},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "prometheus-k8s", Namespace: "openshift-monitoring"}},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: metricsResourceName + "-metrics"},
	}
}

func BuildMetricsService(namespace string) *corev1.Service {
	return BuildMetricsServiceFor(metricsResourceName, namespace, map[string]string{"app": metricsResourceName})
}

// BuildMetricsServiceFor returns a metrics Service with a caller-defined name and pod selector.
func BuildMetricsServiceFor(name, namespace string, selector map[string]string) *corev1.Service {
	return buildCaptureService(name, namespace, selector, 9401)
}

// BuildAgentHealthServiceFor exposes the eBPF agent's common health metrics.
// Unlike the metrics-capture service, this endpoint exists for every capture
// type and uses the agent's own metrics port.
func BuildAgentHealthServiceFor(name, namespace string, selector map[string]string) *corev1.Service {
	return BuildAgentHealthServiceForPort(name, namespace, selector, 9400)
}

// BuildAgentHealthServiceForPort exposes agent health metrics on a capture-specific port.
func BuildAgentHealthServiceForPort(name, namespace string, selector map[string]string, port int32) *corev1.Service {
	return buildCaptureService(name+"-health", namespace, selector, port)
}

func buildCaptureService(name, namespace string, selector map[string]string, port int32) *corev1.Service {
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: map[string]string{"app": name}},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Name: "prometheus", Protocol: corev1.ProtocolTCP, Port: port, TargetPort: intstr.FromInt32(port)}},
			Selector: selector,
		},
	}
}

func BuildMetricsServiceMonitor(namespace string) *monitoringv1.ServiceMonitor {
	return BuildMetricsServiceMonitorFor(metricsResourceName, namespace, map[string]string{"app": metricsResourceName})
}

// BuildMetricsServiceMonitorFor returns a ServiceMonitor selecting the named metrics Service.
func BuildMetricsServiceMonitorFor(name, namespace string, serviceSelector map[string]string) *monitoringv1.ServiceMonitor {
	return buildCaptureServiceMonitor(name, namespace, serviceSelector, []monitoringv1.RelabelConfig{
		{Action: "replace", Regex: `on_demand_netobserv_(workload|namespace)_ingress_.*@(.*)`, Replacement: stringPtr("${2}"), Separator: stringPtr("@"), SourceLabels: []monitoringv1.LabelName{"__name__", "DstK8S_Namespace"}, TargetLabel: "namespace"},
		{Action: "replace", Regex: `on_demand_netobserv_(workload|namespace)_(egress|flows|drop|rtt|dns)_.*@(.*)`, Replacement: stringPtr("${3}"), Separator: stringPtr("@"), SourceLabels: []monitoringv1.LabelName{"__name__", "SrcK8S_Namespace"}, TargetLabel: "namespace"},
	})
}

// BuildAgentHealthServiceMonitorFor adds the FlowCapture identity to every
// scraped sample, including the synthetic up metric used by Grafana.
func BuildAgentHealthServiceMonitorFor(name, namespace string, serviceSelector map[string]string) *monitoringv1.ServiceMonitor {
	return buildCaptureServiceMonitor(name+"-health", namespace, serviceSelector, nil)
}

// BuildCollectorMetricsServiceMonitorFor exposes collector output metrics and
// associates them with the same FlowCapture identity as the agent metrics.
func BuildCollectorMetricsServiceMonitorFor(name, namespace string, serviceSelector map[string]string, flowCapture string) *monitoringv1.ServiceMonitor {
	return buildCaptureServiceMonitorWithLabel(name+"-metrics", namespace, serviceSelector, nil, flowCapture)
}

func buildCaptureServiceMonitor(name, namespace string, serviceSelector map[string]string, metricRelabelings []monitoringv1.RelabelConfig) *monitoringv1.ServiceMonitor {
	captureLabel := strings.TrimSuffix(strings.TrimSuffix(name, "-health"), "-agent")
	return buildCaptureServiceMonitorWithLabel(name, namespace, serviceSelector, metricRelabelings, captureLabel)
}

func buildCaptureServiceMonitorWithLabel(name, namespace string, serviceSelector map[string]string, metricRelabelings []monitoringv1.RelabelConfig, captureLabel string) *monitoringv1.ServiceMonitor {
	relabelings := []monitoringv1.RelabelConfig{{Action: "replace", Replacement: stringPtr(captureLabel), TargetLabel: "flowcapture"}}
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{APIVersion: "monitoring.coreos.com/v1", Kind: "ServiceMonitor"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"monitoring.openshift.io/collection-profile": "full"},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				HonorLabels:          true,
				Interval:             monitoringv1.Duration("15s"),
				RelabelConfigs:       relabelings,
				MetricRelabelConfigs: metricRelabelings,
				Port:                 "prometheus",
				Scheme:               schemePtr(monitoringv1.Scheme("http")),
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{MatchNames: []string{namespace}},
			Selector:          metav1.LabelSelector{MatchLabels: serviceSelector},
		},
	}
}

func stringPtr(value string) *string { return &value }

func schemePtr(value monitoringv1.Scheme) *monitoringv1.Scheme { return &value }
