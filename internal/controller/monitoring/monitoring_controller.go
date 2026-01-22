package monitoring

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/network-observability-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/network-observability-operator/internal/controller/constants"
	"github.com/netobserv/network-observability-operator/internal/controller/reconcilers"
	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager/status"
	"github.com/netobserv/network-observability-operator/internal/pkg/metrics"
	"github.com/netobserv/network-observability-operator/internal/pkg/resources"
)

type Reconciler struct {
	client.Client
	mgr              *manager.Manager
	status           status.Instance
	currentNamespace string
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting Monitoring controller")
	r := Reconciler{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.Monitoring),
	}
	return nil, ctrl.NewControllerManagedBy(mgr).
		For(&flowslatest.FlowCollector{}, reconcilers.IgnoreStatusChange).
		Named("monitoring").
		Owns(&corev1.Namespace{}, reconcilers.UpdateOrDeleteOnlyPred).
		Watches(
			&metricslatest.FlowMetric{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
				if o.GetNamespace() == r.currentNamespace {
					return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
				}
				return []reconcile.Request{}
			}),
			reconcilers.IgnoreStatusChange,
		).
		Complete(&r)
}

// Reconcile is the controller entry point for reconciling current state with desired state.
// It manages the controller status at a high level. Business logic is delegated into `reconcile`.
func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	l := log.Log.WithName("monitoring") // clear context (too noisy)
	ctx = log.IntoContext(ctx, l)

	// In hold mode, skip reconciliation (cleanup is handled by FlowCollector controller)
	if r.mgr.Config.Hold {
		return ctrl.Result{}, nil
	}

	// Get flowcollector & create dedicated client
	clh, desired, err := helper.NewFlowCollectorClientHelper(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get FlowCollector: %w", err)
	} else if desired == nil {
		// Delete case
		return ctrl.Result{}, nil
	}

	r.status.SetUnknown()
	defer r.status.Commit(ctx, r.Client)

	err = r.reconcile(ctx, clh, desired)
	if err != nil {
		l.Error(err, "Monitoring reconcile failure")
		// Set status failure unless it was already set
		if !r.status.HasFailure() {
			r.status.SetFailure("MonitoringError", err.Error())
		}
		return ctrl.Result{}, err
	}

	r.status.SetReady()
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, clh *helper.Client, desired *flowslatest.FlowCollector) error {
	log := log.FromContext(ctx)
	ns := desired.Spec.GetNamespace()
	r.currentNamespace = ns

	// If namespace does not exist, we create it
	nsExist, err := r.namespaceExist(ctx, ns)
	if err != nil {
		return err
	}
	desiredNs := buildNamespace(ns, r.mgr.Config.DownstreamDeployment)
	// always add owned label to desired namespace as we expect it to be created
	helper.AddManagedLabel(desiredNs)
	if nsExist == nil {
		err = r.Create(ctx, desiredNs)
		if err != nil {
			return err
		}
	} else if !helper.SkipOwnership(nsExist) && !helper.IsSubSet(nsExist.ObjectMeta.Labels, desiredNs.ObjectMeta.Labels) {
		err = r.Update(ctx, desiredNs)
		if err != nil {
			return err
		}
	}

	binding := resources.GetExposeMetricsRoleBinding(ns)
	if err := reconcilers.ReconcileRoleBinding(ctx, clh, binding); err != nil {
		return err
	}

	// Dashboards
	if r.mgr.ClusterInfo.IsOpenShift() && r.mgr.ClusterInfo.HasSvcMonitor() {
		// List custom metrics
		fm := metricslatest.FlowMetricList{}
		if err := r.Client.List(ctx, &fm, &client.ListOptions{Namespace: ns}); err != nil {
			return r.status.Error("CantListFlowMetrics", err)
		}
		log.WithValues("items count", len(fm.Items)).Info("FlowMetrics loaded")

		allMetrics := metrics.MergePredefined(fm.Items, &desired.Spec)
		log.WithValues("metrics count", len(allMetrics)).Info("Merged metrics")

		req, err := labels.NewRequirement("netobserv-managed", selection.Exists, []string{})
		if err != nil {
			return r.status.Error("CantQueryRequirement", err)
		}
		// List existing dashboards
		currentDashboards := corev1.ConfigMapList{}
		if err := r.Client.List(ctx, &currentDashboards, &client.ListOptions{
			Namespace:     dashboardCMNamespace,
			LabelSelector: labels.NewSelector().Add(*req),
		}); err != nil {
			return r.status.Error("CantListDashboards", err)
		}

		// Build desired dashboards
		cms := buildFlowMetricsDashboards(allMetrics)
		nsFlowsMetric := getNamespacedFlowsMetric(allMetrics)
		if desiredHealthDashboardCM, del, err := buildHealthDashboard(ns, nsFlowsMetric); err != nil {
			return err
		} else if !del {
			cms = append(cms, desiredHealthDashboardCM)
		}

		for _, cm := range cms {
			current := findAndRemoveConfigMapFromList(&currentDashboards, cm.Name)
			if err := reconcilers.ReconcileConfigMap(ctx, clh, current, cm); err != nil {
				return err
			}
		}

		// Delete any CM that remained in currentDashboards list
		for i := range currentDashboards.Items {
			if err := reconcilers.ReconcileConfigMap(ctx, clh, &currentDashboards.Items[i], nil); err != nil {
				return err
			}
		}
	}

	return nil
}

func getNamespacedFlowsMetric(metrics []metricslatest.FlowMetric) string {
	for i := range metrics {
		if metrics[i].Spec.MetricName == "namespace_flows_total" {
			return "netobserv_namespace_flows_total"
		}
	}
	return "netobserv_workload_flows_total"
}

func findAndRemoveConfigMapFromList(list *corev1.ConfigMapList, name string) *corev1.ConfigMap {
	for i := len(list.Items) - 1; i >= 0; i-- {
		if list.Items[i].Name == name {
			cm := list.Items[i]
			// Remove that element from the list, so the list ends up with elements to delete
			removeFromList(list, i)
			return &cm
		}
	}
	return nil
}

func removeFromList(list *corev1.ConfigMapList, i int) {
	// (quickest removal as order doesn't matter)
	list.Items[i] = list.Items[len(list.Items)-1]
	list.Items = list.Items[:len(list.Items)-1]
}

func (r *Reconciler) namespaceExist(ctx context.Context, nsName string) (*corev1.Namespace, error) {
	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: nsName}, ns)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil
		}
		log.FromContext(ctx).Error(err, "Failed to get namespace")
		return nil, err
	}
	return ns, nil
}
