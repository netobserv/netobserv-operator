package cleanup

import (
	"context"
	"reflect"

	osv1 "github.com/openshift/api/console/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// Add to this list any object that we used to generate in past versions, and stopped doing so.
	// For instance, with any object that was renamed between two releases of the operator:
	// The old version with a different name could therefore remain on the cluster after an upgrade.
	cleanupList = []cleanupItem{
		// Old role bindings (1.8 and before)
		{
			ref:         client.ObjectKey{Name: "netobserv-plugin"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-ingester-role-mono"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-transformer-role-mono"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-ingester-role"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-transformer-role"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
	}
	// Need to run only once, at operator startup, this is not part of the reconcile loop
	didRun = false
)

type cleanupItem struct {
	ref         client.ObjectKey
	placeholder client.Object
	namespaced  bool
}

func CleanPastReferences(ctx context.Context, cl client.Client, defaultNamespace string) error {
	if didRun {
		return nil
	}
	log := log.FromContext(ctx)
	log.Info("Check and clean old objects")
	// Search for all past references to clean up. If one is found, delete it.
	for _, item := range cleanupList {
		if item.ref.Namespace == "" && item.namespaced {
			item.ref.Namespace = defaultNamespace
		}
		if err := cl.Get(ctx, item.ref, item.placeholder); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		// Make sure we own that object
		if helper.IsOwned(item.placeholder) {
			log.
				WithValues("name", item.ref.Name).
				WithValues("namespace", item.ref.Namespace).
				Info("Deleting old object")
			if err := cl.Delete(ctx, item.placeholder); err != nil {
				return err
			}
		} else {
			log.
				WithValues("name", item.ref.Name).
				WithValues("namespace", item.ref.Namespace).
				Info("An object was found, but we don't own it - skip deletion")
		}
	}
	didRun = true
	return nil
}

// DeleteAllManagedResources deletes all resources managed by the operator (labeled with netobserv-managed=true)
// This is used in hold mode to clean up all operator-controlled resources while preserving:
//   - FlowCollector CRDs (user-created)
//   - FlowCollectorSlice CRDs (user-created)
//   - FlowMetric CRDs (user-created)
//
// These CRDs are user-created and don't have the netobserv-managed label, so they are automatically preserved.
func DeleteAllManagedResources(ctx context.Context, cl client.Client) error {
	log := log.FromContext(ctx)
	log.Info("Hold mode: cleaning up all managed resources (preserving FlowCollector, FlowCollectorSlice, and FlowMetric CRDs)")

	// Label selector for managed resources
	labelSelector := client.MatchingLabels{"netobserv-managed": "true"}

	// List of resource types to clean up (namespaced resources)
	// Note: We don't include Namespaces here because they can contain resources from other operators or users.
	// We only delete the specific resources we created within namespaces.
	namespacedTypes := []client.ObjectList{
		&appsv1.DeploymentList{},
		&appsv1.DaemonSetList{},
		&corev1.ServiceList{},
		&corev1.ServiceAccountList{},
		&corev1.ConfigMapList{},
		&corev1.SecretList{},
		&ascv2.HorizontalPodAutoscalerList{},
		&networkingv1.NetworkPolicyList{},
		&monitoringv1.ServiceMonitorList{},
		&monitoringv1.PrometheusRuleList{},
		&rbacv1.RoleBindingList{},
	}

	// Cluster-scoped resources
	// Note: We don't include SecurityContextConstraints as they are infrastructure/policy resources
	// that require elevated permissions and don't directly impact cluster workload performance.
	clusterTypes := []client.ObjectList{
		&rbacv1.ClusterRoleList{},
		&rbacv1.ClusterRoleBindingList{},
		&osv1.ConsolePluginList{},
	}

	// Delete namespaced resources
	for _, listObj := range namespacedTypes {
		if err := deleteResourcesByType(ctx, cl, listObj, labelSelector); err != nil {
			return err
		}
	}

	// Delete cluster-scoped resources
	for _, listObj := range clusterTypes {
		if err := deleteResourcesByType(ctx, cl, listObj, labelSelector); err != nil {
			return err
		}
	}

	log.Info("Hold mode: cleanup completed")
	return nil
}

func deleteResourcesByType(ctx context.Context, cl client.Client, listObj client.ObjectList, labelSelector client.MatchingLabels) error {
	log := log.FromContext(ctx)
	typeName := reflect.TypeOf(listObj).String()

	// List resources with the label selector
	if err := cl.List(ctx, listObj, labelSelector); err != nil {
		// Ignore errors for resource types that don't exist in this cluster (e.g., OpenShift-specific resources on vanilla k8s)
		if !errors.IsNotFound(err) && !errors.IsForbidden(err) {
			log.Error(err, "Failed to list resources", "type", typeName)
			return err
		}
		return nil
	}

	// Extract items from the list using meta.ExtractList
	items, err := meta.ExtractList(listObj)
	if err != nil {
		log.Error(err, "Failed to extract items from list", "type", typeName)
		return err
	}

	// Delete each resource
	for _, item := range items {
		obj, ok := item.(client.Object)
		if !ok {
			continue
		}

		// Double-check that it's owned before deleting
		if !helper.IsOwned(obj) {
			log.Info("SKIP deletion since not owned", "type", typeName, "name", obj.GetName(), "namespace", obj.GetNamespace())
			continue
		}

		log.Info("DELETING managed resource", "type", typeName, "name", obj.GetName(), "namespace", obj.GetNamespace())
		if err := cl.Delete(ctx, obj); err != nil {
			if !errors.IsNotFound(err) {
				log.Error(err, "Failed to delete resource", "type", typeName, "name", obj.GetName(), "namespace", obj.GetNamespace())
				return err
			}
		}
	}

	return nil
}
