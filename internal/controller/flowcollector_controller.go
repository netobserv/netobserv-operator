package controllers

import (
	"context"
	"fmt"

	osv1 "github.com/openshift/api/console/v1"
	securityv1 "github.com/openshift/api/security/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/consoleplugin"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/demoloki"
	"github.com/netobserv/netobserv-operator/internal/controller/ebpf"
	"github.com/netobserv/netobserv-operator/internal/controller/lokistack"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/cleanup"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/watchers"
)

const (
	flowsFinalizer = "flows.netobserv.io/finalizer"
)

// FlowCollectorReconciler reconciles a FlowCollector object
type FlowCollectorReconciler struct {
	client.Client
	mgr              *manager.Manager
	status           status.Instance
	watcher          *watchers.Watcher
	ctrl             controller.Controller
	lokistackWatcher *lokistack.Watcher
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting FlowCollector controller")
	r := FlowCollectorReconciler{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.FlowCollectorController),
	}

	builder := ctrl.NewControllerManagedBy(mgr.Manager).
		Named("legacy").
		For(&flowslatest.FlowCollector{}, reconcilers.IgnoreStatusChange).
		Owns(&appsv1.Deployment{}, reconcilers.UpdateOrDeleteOnlyPred).
		Owns(&appsv1.DaemonSet{}, reconcilers.UpdateOrDeleteOnlyPred).
		Owns(&ascv2.HorizontalPodAutoscaler{}, reconcilers.UpdateOrDeleteOnlyPred).
		Owns(&corev1.Namespace{}, reconcilers.UpdateOrDeleteOnlyPred).
		Owns(&corev1.Service{}, reconcilers.UpdateOrDeleteOnlyPred).
		Owns(&corev1.ServiceAccount{}, reconcilers.UpdateOrDeleteOnlyPred)

	if mgr.ClusterInfo.IsOpenShift() {
		builder.Owns(&securityv1.SecurityContextConstraints{}, reconcilers.UpdateOrDeleteOnlyPred)
	}
	if mgr.ClusterInfo.HasConsolePlugin() {
		builder.Owns(&osv1.ConsolePlugin{}, reconcilers.UpdateOrDeleteOnlyPred)
	}

	r.lokistackWatcher = lokistack.Start(ctx, mgr, builder, func() controller.Controller { return r.ctrl })

	// When a PrometheusRule changes, trigger reconcile so console-plugin config is updated (recording-rule annotations)
	if mgr.ClusterInfo.HasPromRule() {
		builder.Watches(
			&monitoringv1.PrometheusRule{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
				// Only trigger reconcile for PrometheusRules with netobserv=true label
				labels := o.GetLabels()
				if labels != nil && labels["netobserv"] == "true" {
					return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
				}
				return []reconcile.Request{}
			}),
		)
		log.Info("PrometheusRule CRD detected, watching for netobserv=true rules")
	}

	ctrl, err := builder.Build(&r)
	if err != nil {
		return nil, err
	}
	r.ctrl = ctrl
	r.watcher = watchers.NewWatcher(ctrl)

	return nil, nil
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the FlowCollector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *FlowCollectorReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	l := log.Log.WithName("legacy") // clear context (too noisy)
	ctx = log.IntoContext(ctx, l)

	// Get flowcollector & create dedicated client
	clh, desired, err := helper.NewFlowCollectorClientHelper(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get FlowCollector: %w", err)
	} else if desired == nil {
		// Delete case
		return ctrl.Result{}, nil
	}

	defer r.status.Commit(ctx, r.Client)

	err = r.reconcile(ctx, clh, desired)
	if err != nil {
		l.Error(err, "FlowCollector reconcile failure")
		// Set status failure unless it was already set
		if !r.status.HasFailure() {
			r.status.SetFailure("ChildReconcilerError", err.Error())
		}
		return ctrl.Result{}, err
	}

	r.status.SetReady()
	return ctrl.Result{}, nil
}

func (r *FlowCollectorReconciler) reconcile(ctx context.Context, clh *helper.Client, desired *flowslatest.FlowCollector) error {
	ns := desired.Spec.GetNamespace()
	previousNamespace := r.status.GetDeployedNamespace(desired)
	lokiConfig := helper.NewLokiConfig(&desired.Spec.Loki, ns)
	reconcilersInfo := r.newCommonInfo(clh, ns, &lokiConfig)

	if err := r.checkFinalizer(ctx, desired); err != nil {
		return err
	}

	if err := cleanup.CleanPastReferences(ctx, r.Client, ns); err != nil {
		return err
	}
	r.watcher.Reset(ns)

	lokiStatus := r.lokistackWatcher.Reconcile(ctx, desired)

	// Create reconcilers
	cpReconciler := consoleplugin.NewReconciler(reconcilersInfo.NewInstance(
		map[reconcilers.ImageRef]string{
			reconcilers.MainImage:                r.mgr.Config.ConsolePluginImage,
			reconcilers.ConsolePluginCompatImage: r.mgr.Config.ConsolePluginCompatImage,
		},
		r.mgr.Status.ForComponent(status.WebConsole),
	))

	// Check namespace changed
	if ns != previousNamespace {
		// Update namespace in status
		if err := r.status.SetDeployedNamespace(ctx, r.Client, ns); err != nil {
			return r.status.Error("ChangeNamespaceError", err)
		}
	}

	// eBPF agent
	ebpfAgentController := ebpf.NewAgentController(reconcilersInfo.NewInstance(
		map[reconcilers.ImageRef]string{
			reconcilers.MainImage:        r.mgr.Config.EBPFAgentImage,
			reconcilers.BpfByteCodeImage: r.mgr.Config.EBPFByteCodeImage,
		},
		r.mgr.Status.ForComponent(status.EBPFAgents),
	))
	if err := ebpfAgentController.Reconcile(ctx, desired); err != nil {
		return err
	}

	// Console plugin
	if err := cpReconciler.Reconcile(ctx, desired, lokiStatus); err != nil {
		return err
	}

	lokiReconciler := demoloki.NewReconciler(reconcilersInfo.NewInstance(
		map[reconcilers.ImageRef]string{
			reconcilers.MainImage: r.mgr.Config.DemoLokiImage,
		},
		r.mgr.Status.ForComponent(status.DemoLoki),
	))
	if err := lokiReconciler.Reconcile(ctx, desired); err != nil {
		return err
	}

	return nil
}

func (r *FlowCollectorReconciler) checkFinalizer(ctx context.Context, desired *flowslatest.FlowCollector) error {
	// Previous version of the operator (1.5) had a finalizer, this isn't the case anymore.
	// Remove any finalizer that could remain after an upgrade.
	if controllerutil.ContainsFinalizer(desired, flowsFinalizer) {
		controllerutil.RemoveFinalizer(desired, flowsFinalizer)
		return r.Update(ctx, desired)
	}

	return nil
}

func (r *FlowCollectorReconciler) newCommonInfo(clh *helper.Client, ns string, loki *helper.LokiConfig) reconcilers.Common {
	return reconcilers.Common{
		Client:       *clh,
		Namespace:    ns,
		ClusterInfo:  r.mgr.ClusterInfo,
		Watcher:      r.watcher,
		Loki:         loki,
		IsDownstream: r.mgr.Config.DownstreamDeployment,
	}
}
