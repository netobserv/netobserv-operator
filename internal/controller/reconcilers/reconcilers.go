package reconcilers

import (
	"context"
	"fmt"
	"reflect"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/roles"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	IgnoreStatusChange = builder.WithPredicates(predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Update only if spec / annotations / labels change, ie. ignore status changes.
			// Also react to deletion: setting a deletionTimestamp doesn't bump the generation,
			// so we must catch it here for finalizers to be processed.
			return (e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()) ||
				IsMarkedForDeletion(e.ObjectNew) != IsMarkedForDeletion(e.ObjectOld) ||
				!equality.Semantic.DeepEqual(e.ObjectNew.GetAnnotations(), e.ObjectOld.GetAnnotations()) ||
				!equality.Semantic.DeepEqual(e.ObjectNew.GetLabels(), e.ObjectOld.GetLabels())
		},
		CreateFunc:  func(_ event.CreateEvent) bool { return true },
		DeleteFunc:  func(_ event.DeleteEvent) bool { return true },
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	})
	UpdateOrDeleteOnlyPred = builder.WithPredicates(predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Update only if new object is owned - we want to watch for status changes as well (e.g. to know when a deployment is ready)
			return helper.IsOwned(e.ObjectNew)
		},
		CreateFunc: func(_ event.CreateEvent) bool { return false },
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Update only if it was owned and confirmed as deleted by the api server
			return helper.IsOwned(e.Object) && !e.DeleteStateUnknown
		},
		GenericFunc: func(_ event.GenericEvent) bool { return false },
	})
	OperatorOwned = func(ns string) builder.Predicates {
		return builder.WithPredicates(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return helper.IsOperatorOwned(ns, e.ObjectNew)
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return helper.IsOperatorOwned(ns, e.Object)
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return helper.IsOperatorOwned(ns, e.Object)
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return helper.IsOperatorOwned(ns, e.Object)
			},
		})
	}
)

// IsMarkedForDeletion returns true when the object has a non-zero deletionTimestamp.
func IsMarkedForDeletion(o client.Object) bool {
	ts := o.GetDeletionTimestamp()
	return ts != nil && !ts.IsZero()
}

// ReconcileClusterRoleBinding updates the current role binding with the provided service account as a subject.
// It does NOT try to create or delete it: operand CRBs are expected to be preinstalled. The operator does not have create permission.
func ReconcileClusterRoleBinding(ctx context.Context, cl *helper.Client, namespace, sa string, ref roles.ClusterRoleName, isDelete bool) error {
	log := log.FromContext(ctx)
	crb := rbacv1.ClusterRoleBinding{}
	if err := cl.Get(ctx, types.NamespacedName{Name: string(ref)}, &crb); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("can't reconcile ClusterRoleBinding %s, it should be preinstalled; was it removed? - %w", ref, err)
		}
		return fmt.Errorf("can't reconcile ClusterRoleBinding %s: %w", ref, err)
	}
	subject := rbacv1.Subject{
		Kind:      "ServiceAccount",
		Name:      sa,
		Namespace: namespace,
	}
	index := findSubject(crb.Subjects, subject)
	if index >= 0 && isDelete {
		log.Info("DELETING subject from ClusterRoleBinding "+string(ref), "Namespace", namespace, "SA", sa)
		crb.Subjects = append(crb.Subjects[:index], crb.Subjects[index+1:]...)
		err := cl.Update(ctx, &crb)
		if err != nil {
			log.Error(err, "Failed to delete subject from ClusterRoleBinding "+string(ref), "Namespace", namespace, "SA", sa)
			return err
		}
		return nil
	}
	if index < 0 && !isDelete {
		log.Info("ADDING subject to ClusterRoleBinding "+string(ref), "Namespace", namespace, "SA", sa)
		crb.Subjects = append(crb.Subjects, subject)
		err := cl.Update(ctx, &crb)
		if err != nil {
			log.Error(err, "Failed to add subject to ClusterRoleBinding "+string(ref), "Namespace", namespace, "SA", sa)
			return err
		}
	}
	return nil
}

// EmptyClusterRoleBinding removes all subjects from the given preinstalled ClusterRoleBinding.
// The binding itself is kept (it is expected to be preinstalled as an empty shell, and the operator
// does not have delete permission on it). Used to clean up upon FlowCollector deletion.
func EmptyClusterRoleBinding(ctx context.Context, cl *helper.Client, ref roles.ClusterRoleName) error {
	log := log.FromContext(ctx)
	crb := rbacv1.ClusterRoleBinding{}
	if err := cl.Get(ctx, types.NamespacedName{Name: string(ref)}, &crb); err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("can't empty ClusterRoleBinding %s, it should be preinstalled; was it removed? - %w", ref, err)
		}
		return fmt.Errorf("can't empty ClusterRoleBinding %s: %w", ref, err)
	}
	if len(crb.Subjects) == 0 {
		return nil
	}
	log.Info("EMPTYING subjects from ClusterRoleBinding " + string(ref))
	crb.Subjects = nil
	if err := cl.Update(ctx, &crb); err != nil {
		log.Error(err, "Failed to empty subjects from ClusterRoleBinding "+string(ref))
		return err
	}
	return nil
}

func findSubject(current []rbacv1.Subject, subject rbacv1.Subject) int {
	for i := range current {
		if current[i].Kind == subject.Kind && current[i].Name == subject.Name && current[i].Namespace == subject.Namespace {
			return i
		}
	}
	return -1
}

func ReconcileRoleBinding(ctx context.Context, cl *helper.Client, desired *rbacv1.RoleBinding) error {
	actual := rbacv1.RoleBinding{}
	if err := cl.Get(ctx, types.NamespacedName{Name: desired.ObjectMeta.Name, Namespace: desired.ObjectMeta.Namespace}, &actual); err != nil {
		if errors.IsNotFound(err) {
			return cl.CreateOwned(ctx, desired)
		}
		return fmt.Errorf("can't reconcile RoleBinding %s: %w", desired.Name, err)
	}
	if helper.IsSubSet(actual.Labels, desired.Labels) &&
		actual.RoleRef == desired.RoleRef &&
		reflect.DeepEqual(actual.Subjects, desired.Subjects) {
		if actual.RoleRef != desired.RoleRef {
			// Roleref cannot be updated deleting and creating a new rolebinding
			log := log.FromContext(ctx)
			log.Info("Deleting old RoleBinding", "Namespace", actual.GetNamespace(), "Name", actual.GetName())
			err := cl.Delete(ctx, &actual)
			if err != nil {
				log.Error(err, "error deleting old RoleBinding", "Namespace", actual.GetNamespace(), "Name", actual.GetName())
			}
			return cl.CreateOwned(ctx, desired)
		}
		// role binding already reconciled. Exiting
		return nil
	}
	return cl.UpdateIfOwned(ctx, &actual, desired)
}

func ReconcileConfigMap(ctx context.Context, cl *helper.Client, current, desired *corev1.ConfigMap) error {
	if current == nil {
		if desired == nil {
			return nil
		}
		return cl.CreateOwned(ctx, desired)
	}
	if desired == nil {
		if helper.IsOwned(current) {
			return cl.Delete(ctx, current)
		}
		return nil
	}
	if helper.IsSubSet(current.Labels, desired.Labels) && reflect.DeepEqual(current.Data, desired.Data) {
		// configmap already reconciled. Exiting
		return nil
	}
	return cl.UpdateIfOwned(ctx, current, desired)
}

// ReconcileDaemonSet reconciles a DaemonSet and checks pod health when not ready.
func ReconcileDaemonSet(ctx context.Context, ci *Instance, old, n *appsv1.DaemonSet, containerName string, report *helper.ChangeReport) error {
	if !ci.Managed.Exists(old) {
		ci.Status.SetCreatingDaemonSet(n)
		return ci.CreateOwned(ctx, n)
	}
	ci.Status.CheckDaemonSetHealth(ctx, ci.Client, old)
	if helper.PodChanged(&old.Spec.Template, &n.Spec.Template, containerName, report) {
		return ci.UpdateIfOwned(ctx, old, n)
	}
	return nil
}

// ReconcileDeployment reconciles a Deployment and checks pod health when not ready.
func ReconcileDeployment(ctx context.Context, ci *Instance, old, n *appsv1.Deployment, containerName string, ignoreReplicas bool, report *helper.ChangeReport) error {
	if !ci.Managed.Exists(old) {
		ci.Status.SetCreatingDeployment(n)
		return ci.CreateOwned(ctx, n)
	}
	ci.Status.CheckDeploymentHealth(ctx, ci.Client, old)
	if ignoreReplicas {
		n.Spec.Replicas = old.Spec.Replicas
	}
	if helper.DeploymentChanged(old, n, containerName, report) {
		return ci.UpdateIfOwned(ctx, old, n)
	}
	return nil
}

func ReconcileHPA(ctx context.Context, ci *Instance, old, n *ascv2.HorizontalPodAutoscaler, desired *flowslatest.FlowCollectorHPA, report *helper.ChangeReport) error {
	// Delete or Create / Update Autoscaler according to HPA option
	if desired.IsHPAEnabled() {
		if !ci.Managed.Exists(old) {
			return ci.CreateOwned(ctx, n)
		} else if helper.AutoScalerChanged(old, *desired, report) {
			return ci.UpdateIfOwned(ctx, old, n)
		}
	} else {
		if err := ci.Managed.TryDelete(ctx, old); err != nil {
			return err
		}
	}
	return nil
}

func ReconcileService(ctx context.Context, ci *Instance, old, n *corev1.Service, report *helper.ChangeReport) error {
	if !ci.Managed.Exists(old) {
		if err := ci.CreateOwned(ctx, n); err != nil {
			return err
		}
	} else if helper.ServiceChanged(old, n, report) {
		// In case we're updating an existing service, we need to build from the old one to keep immutable fields such as clusterIP
		newSVC := old.DeepCopy()
		newSVC.Spec.Ports = n.Spec.Ports
		newSVC.ObjectMeta.Annotations = n.ObjectMeta.Annotations
		if err := ci.UpdateIfOwned(ctx, old, newSVC); err != nil {
			return err
		}
	}
	return nil
}

func ReconcileNetworkPolicy(ctx context.Context, cl *helper.Client, name types.NamespacedName, desired *networkingv1.NetworkPolicy) error {
	current := networkingv1.NetworkPolicy{}
	if err := cl.Get(ctx, name, &current); err != nil {
		if errors.IsNotFound(err) {
			if desired != nil {
				return cl.CreateOwned(ctx, desired)
			}
			return nil
		}
		return fmt.Errorf("can't reconcile Network Policy %s: %w", name.Name, err)
	}
	if desired == nil {
		if helper.IsOwned(&current) {
			return cl.Delete(ctx, &current)
		}
		return nil
	}
	if helper.IsSubSet(current.Labels, desired.Labels) && reflect.DeepEqual(current.Spec, desired.Spec) {
		// network policy already reconciled. Exiting
		return nil
	}
	return cl.UpdateIfOwned(ctx, &current, desired)
}

func GenericReconcile[K client.Object](ctx context.Context, m *NamespacedObjectManager, cl *helper.Client, old, n K, report *helper.ChangeReport, changeFunc func(old, n K, report *helper.ChangeReport) bool) error {
	if !m.Exists(old) {
		return cl.CreateOwned(ctx, n)
	}
	if changeFunc(old, n, report) {
		return cl.UpdateIfOwned(ctx, old, n)
	}
	return nil
}
