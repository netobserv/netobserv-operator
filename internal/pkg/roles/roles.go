package roles

import (
	"context"
	"fmt"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	authv1 "k8s.io/api/authorization/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterRoleName string

const (
	LokiWriterRole          ClusterRoleName = "netobserv-loki-writer"
	FLPInformersRole        ClusterRoleName = "netobserv-informers"
	HostNetworkRole         ClusterRoleName = "netobserv-hostnetwork"
	ConsoleTokenReviewRole  ClusterRoleName = "netobserv-token-review"
	FlowCollectorViewerRole ClusterRoleName = "netobserv-flowcollector-viewer-role"
)

var (
	sarCheckEnabled = true
	// Resource representative of the ClusterRole for permission checks
	reprResourceForRole = map[ClusterRoleName]authv1.ResourceAttributes{
		LokiWriterRole: {
			Verb:     "create",
			Group:    "loki.grafana.com",
			Resource: "network",
		},
		FLPInformersRole: {
			Verb:     "watch",
			Group:    "apps",
			Resource: "replicasets",
		},
		HostNetworkRole: {
			Verb:     "use",
			Group:    "security.openshift.io",
			Resource: "securitycontextconstraints",
			Name:     "hostnetwork",
		},
		ConsoleTokenReviewRole: {
			Verb:     "create",
			Group:    "authentication.k8s.io",
			Resource: "tokenreviews",
		},
		FlowCollectorViewerRole: {
			Verb:     "get",
			Group:    "flows.netobserv.io",
			Resource: "flowcollectors",
		},
	}
)

func GetRoleBindingName(shortName string, ref constants.RoleName) string {
	return string(ref) + "-" + shortName
}

func GetRoleBinding(namespace, shortName, app, sa string, ref constants.RoleName, fromClusterRole bool) *rbacv1.RoleBinding {
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
			Name:      string(constants.ExposeMetricsRole),
			Namespace: ns,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     string(constants.ExposeMetricsRole),
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      constants.MonitoringServiceAccount,
			Namespace: constants.OpenShiftMonitoringNamespace,
		}},
	}
}

// EnableSARChecks enables or disables SAR checks - used for tests. Restoring function is returned.
func EnableSARChecks(enable bool) func() {
	prev := sarCheckEnabled
	sarCheckEnabled = enable
	return func() {
		sarCheckEnabled = prev
	}
}

// CheckHasPermission performs a SubjectAccessReview (SAR) for the required service account and cluster role.
func CheckHasPermission(ctx context.Context, cl client.Client, namespace, sa string, roleName ClusterRoleName) error {
	if !sarCheckEnabled {
		return nil
	}
	attr, ok := reprResourceForRole[roleName]
	if !ok {
		return fmt.Errorf("no representative resource defined for %s", roleName)
	}
	sar := &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User:               fmt.Sprintf("system:serviceaccount:%s:%s", namespace, sa),
			ResourceAttributes: &attr,
		},
	}
	if err := cl.Create(ctx, sar); err != nil {
		return fmt.Errorf("fail to run SAR for role %s: %w", roleName, err)
	}
	if !sar.Status.Allowed {
		return fmt.Errorf("missing cluster role binding for %s. To grant this permission, run: "+
			"kubectl create clusterrolebinding %s-custom --clusterrole=%s --serviceaccount=%s:%s",
			sa, roleName, roleName, namespace, sa)
	}
	return nil
}
