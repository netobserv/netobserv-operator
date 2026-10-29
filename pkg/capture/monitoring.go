package capture

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const metricsResourceName = "netobserv-cli"

// BuildMetricsResources returns the typed Kubernetes resources needed to expose
// a metrics capture to OpenShift monitoring. The CLI and FlowCapture share the
// same labels, port, relabeling and monitoring RBAC through this builder.
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
	return &corev1.Service{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{Name: metricsResourceName, Namespace: namespace, Labels: map[string]string{"app": metricsResourceName}},
		Spec: corev1.ServiceSpec{
			Ports:    []corev1.ServicePort{{Name: "prometheus", Protocol: corev1.ProtocolTCP, Port: 9401, TargetPort: intstrFromInt(9401)}},
			Selector: map[string]string{"app": metricsResourceName},
		},
	}
}

func BuildMetricsServiceMonitor(namespace string) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta:   metav1.TypeMeta{APIVersion: "monitoring.coreos.com/v1", Kind: "ServiceMonitor"},
		ObjectMeta: metav1.ObjectMeta{Name: metricsResourceName, Namespace: namespace},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{{
				HonorLabels: true,
				Interval:    monitoringv1.Duration("15s"),
				MetricRelabelConfigs: []monitoringv1.RelabelConfig{
					{Action: "replace", Regex: `on_demand_netobserv_(workload|namespace)_ingress_.*@(.*)`, Replacement: stringPtr("${2}"), Separator: stringPtr("@"), SourceLabels: []monitoringv1.LabelName{"__name__", "DstK8S_Namespace"}, TargetLabel: "namespace"},
					{Action: "replace", Regex: `on_demand_netobserv_(workload|namespace)_(egress|flows|drop|rtt|dns)_.*@(.*)`, Replacement: stringPtr("${3}"), Separator: stringPtr("@"), SourceLabels: []monitoringv1.LabelName{"__name__", "SrcK8S_Namespace"}, TargetLabel: "namespace"},
				},
				Port:   "prometheus",
				Scheme: schemePtr(monitoringv1.Scheme("http")),
			}},
			NamespaceSelector: monitoringv1.NamespaceSelector{MatchNames: []string{namespace}},
			Selector:          metav1.LabelSelector{MatchLabels: map[string]string{"app": metricsResourceName}},
		},
	}
}

func intstrFromInt(value int32) intstr.IntOrString {
	return intstr.FromInt32(value)
}

func stringPtr(value string) *string { return &value }

func schemePtr(value monitoringv1.Scheme) *monitoringv1.Scheme { return &value }
