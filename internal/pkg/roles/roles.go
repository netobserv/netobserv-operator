package roles

import (
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterRoleName string
type RoleName string

const (
	// [Cluster]Roles, must match names in config/rbac/component_roles.yaml (without netobserv- prefix)
	LokiWriterRole          ClusterRoleName = "netobserv-loki-writer"
	LokiReaderRole          ClusterRoleName = "netobserv-loki-reader"
	PromReaderRole          ClusterRoleName = "netobserv-metrics-reader"
	ExposeMetricsRole       RoleName        = "netobserv-expose-metrics"
	FLPInformersRole        ClusterRoleName = "netobserv-informers"
	HostNetworkRole         ClusterRoleName = "netobserv-hostnetwork"
	ConsoleTokenReviewRole  ClusterRoleName = "netobserv-token-review"
	FlowCollectorViewerRole ClusterRoleName = "netobserv-flowcollector-viewer-role"
	ConfigWatcherRole       RoleName        = "netobserv-config-watcher"
)

// OperandClusterRoleBindings lists the preinstalled ClusterRoleBindings whose subjects are managed
// by the operator: subjects are added when operands need them, and emptied when the FlowCollector
// is deleted. The bindings themselves are preinstalled empty shells and are never created/deleted here.
var OperandClusterRoleBindings = []ClusterRoleName{
	LokiWriterRole,
	FLPInformersRole,
	HostNetworkRole,
	ConsoleTokenReviewRole,
	FlowCollectorViewerRole,
}

func GetRoleBindingName(shortName string, ref RoleName) string {
	return string(ref) + "-" + shortName
}

func GetRoleBinding(namespace, shortName, app, sa string, ref RoleName, fromClusterRole bool) *rbacv1.RoleBinding {
	roleKind := "Role"
	if fromClusterRole {
		roleKind = "ClusterRole"
	}
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(ref) + "-" + shortName,
			Namespace: namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     app,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     roleKind,
			Name:     string(ref),
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      sa,
			Namespace: namespace,
		}},
	}
}

func GetExposeMetricsRoleBinding(ns string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      string(ExposeMetricsRole),
			Namespace: ns,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     string(ExposeMetricsRole),
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      constants.MonitoringServiceAccount,
			Namespace: constants.OpenShiftMonitoringNamespace,
		}},
	}
}
