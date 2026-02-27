package controllers

import (
	"context"
	"fmt"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	osv1 "github.com/openshift/api/console/v1"
	securityv1 "github.com/openshift/api/security/v1"
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
	"sigs.k8s.io/controller-runtime/pkg/source"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/network-observability-operator/internal/controller/consoleplugin"
	"github.com/netobserv/network-observability-operator/internal/controller/constants"
	"github.com/netobserv/network-observability-operator/internal/controller/ebpf"
	"github.com/netobserv/network-observability-operator/internal/controller/loki"
	"github.com/netobserv/network-observability-operator/internal/controller/reconcilers"
	"github.com/netobserv/network-observability-operator/internal/pkg/cleanup"
	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager/status"
	"github.com/netobserv/network-observability-operator/internal/pkg/watchers"
)

const (
	flowsFinalizer = "flows.netobserv.io/finalizer"
)

// FlowCollectorReconciler reconciles a FlowCollector object
type FlowCollectorReconciler struct {
	client.Client
	mgr                *manager.Manager
	status             status.Instance
	watcher            *watchers.Watcher
	ctrl               controller.Controller
	lokiWatcherStarted bool
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting FlowCollector controller")
	r := FlowCollectorReconciler{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.FlowCollectorLegacy),
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

	if mgr.ClusterInfo.HasLokiStack(ctx) {
		builder.Watches(
			&lokiv1.LokiStack{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []ctrl.Request {
				// When a LokiStack changes, trigger reconcile of the FlowCollector
				return []ctrl.Request{{NamespacedName: constants.FlowCollectorName}}
			}),
		)
		r.lokiWatcherStarted = true
		log.Info("LokiStack CRD detected")
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

	// Dynamically start LokiStack watcher if the API became available
	if desired.Spec.Loki.Mode == flowslatest.LokiModeLokiStack {
		if err := r.ensureLokiStackWatcher(ctx); err != nil {
			l.Error(err, "Failed to start LokiStack watcher")
			// Don't fail reconciliation, just log the error
		}
	}

	// At the moment, status workflow is to start as ready then degrade if necessary
	// Later (when legacy controller is broken down into individual controllers), status should start as unknown and only on success finishes as ready
	r.status.SetReady()
	defer r.status.Commit(ctx, r.Client)

	err = r.reconcile(ctx, clh, desired)
	if err != nil {
		l.Error(err, "FlowCollector reconcile failure")
		// Set status failure unless it was already set
		if !r.status.HasFailure() {
			r.status.SetFailure("FlowCollectorGenericError", err.Error())
		}
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *FlowCollectorReconciler) ensureLokiStackWatcher(ctx context.Context) error {
	// Check if LokiStack API is available but watcher not started
	if !r.mgr.ClusterInfo.HasLokiStack(ctx) {
		return nil
	}

	if r.lokiWatcherStarted {
		return nil
	}

	// LokiStack API is now available, start the watcher
	log := log.FromContext(ctx)
	log.Info("LokiStack CRD detected after startup, starting watcher")

	h := handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, _ *lokiv1.LokiStack) []reconcile.Request {
		// When a LokiStack changes, trigger reconcile of the FlowCollector
		return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
	})

	src := source.Kind(r.mgr.GetCache(), &lokiv1.LokiStack{}, h)
	err := r.ctrl.Watch(src)
	if err != nil {
		return fmt.Errorf("failed to start LokiStack watcher: %w", err)
	}

	r.lokiWatcherStarted = true
	log.Info("LokiStack watcher started successfully")
	return nil
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

	// Create reconcilers
	cpReconciler := consoleplugin.NewReconciler(reconcilersInfo.NewInstance(
		map[reconcilers.ImageRef]string{
			reconcilers.MainImage:                r.mgr.Config.ConsolePluginImage,
			reconcilers.ConsolePluginCompatImage: r.mgr.Config.ConsolePluginCompatImage,
		},
		r.status,
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
		r.status,
	))
	if err := ebpfAgentController.Reconcile(ctx, desired); err != nil {
		return r.status.Error("ReconcileAgentFailed", err)
	}

	// Console plugin
	if err := cpReconciler.Reconcile(ctx, desired); err != nil {
		return r.status.Error("ReconcileConsolePluginFailed", err)
	}

	lokiReconciler := loki.NewReconciler(reconcilersInfo.NewInstance(
		map[reconcilers.ImageRef]string{
			reconcilers.MainImage: r.mgr.Config.DemoLokiImage,
		},
		r.status,
	))
	if err := lokiReconciler.Reconcile(ctx, desired); err != nil {
		return r.status.Error("ReconcileLokiFailed", err)
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
