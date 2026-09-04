package static

import (
	"context"
	"fmt"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/netobserv/netobserv-operator/internal/controller/consoleplugin"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
)

var (
	clog = log.Log.WithName("static-controller")
)

type Controller struct {
	client.Client
	mgr    *manager.Manager
	status status.Instance
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting Static controller")
	r := Controller{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.StaticController),
	}

	// This controller runs unconditionally (not bound to FlowCollector), and uses the operator Deployment as a trigger.
	b := ctrl.NewControllerManagedBy(mgr).
		Named("StaticController").
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
				if o.GetNamespace() == mgr.Config.Namespace && o.GetName() == constants.ControllerName {
					return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
				}
				return nil
			}),
			reconcilers.IgnoreStatusChange,
		).
		Watches(
			&appsv1.Deployment{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
				if o.GetNamespace() == mgr.Config.Namespace && o.GetName() == constants.StaticPluginName {
					return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
				}
				return nil
			}),
		).
		Watches(
			&networkingv1.NetworkPolicy{},
			&handler.EnqueueRequestForObject{},
			reconcilers.OperatorOwned(mgr.Config.Namespace),
			reconcilers.IgnoreStatusChange,
		)
	if mgr.Config.StaticPluginConfig.InheritTolerationFromSubscription != "" {
		b = b.Watches(
			&olm.Subscription{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
				if o.GetNamespace() == mgr.Config.Namespace && o.GetName() == mgr.Config.StaticPluginConfig.InheritTolerationFromSubscription {
					return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
				}
				return nil
			}),
			reconcilers.IgnoreStatusChange,
		)
	}
	return nil, b.Complete(&r)
}

// Reconcile is the controller entry point for reconciling current state with desired state.
// It manages the controller status at a high level. Business logic is delegated into `reconcile`.
func (r *Controller) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	ctx = log.IntoContext(ctx, clog)

	commit := r.status.Reset()
	defer commit(ctx, r.Client)

	// Create operator-owning client wrapper
	scp, err := helper.NewControllerClientHelper(ctx, r.mgr.Config.Namespace, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get controller client: %w", err)
	}

	if r.mgr.ClusterInfo.HasConsolePlugin() {
		// Only deploy static plugin on OpenShift 4.15+
		if !r.mgr.ClusterInfo.IsOpenShift() {
			clog.Info("Skipping static plugin reconciler (not OpenShift)")
		} else if supported, _, err := r.mgr.ClusterInfo.IsOpenShiftVersionAtLeast("4.15.0"); err != nil {
			return ctrl.Result{}, err
		} else if !supported {
			clog.Info("Skipping static plugin reconciler (OpenShift version < 4.15)")
		} else {
			ri := r.newDefaultReconcilerInstance(scp, r.mgr.Config.ResolveWebConsoleImage(r.mgr.ClusterInfo))
			staticPluginReconciler := consoleplugin.NewStaticReconciler(ri, r.mgr.Config)
			if err := staticPluginReconciler.ReconcileStaticPlugin(ctx, true); err != nil {
				clog.Error(err, "Static plugin reconcile failure")
				// Set status failure unless it was already set
				if !r.status.HasFailure() {
					r.status.SetFailure("StaticPluginError", err.Error())
				}
				return ctrl.Result{}, err
			}
		}
	}

	opReconciler := newOperatorReconciler(
		r.newDefaultReconcilerInstance(scp, ""),
		r.mgr.Config,
	)
	if err := opReconciler.reconcile(ctx); err != nil {
		clog.Error(err, "Operator network policy reconcile failure")
		r.status.SetFailure("OperatorNetworkPolicyError", err.Error())
		return ctrl.Result{}, err
	}

	r.status.SetReady()
	return ctrl.Result{}, nil
}

func (r *Controller) newDefaultReconcilerInstance(clh *helper.Client, image string) *reconcilers.Instance {
	// force default namespace
	reconcilersInfo := reconcilers.Common{
		Client:      *clh,
		Namespace:   r.mgr.Config.Namespace,
		ClusterInfo: r.mgr.ClusterInfo,
		Watcher:     nil,
		Loki:        &helper.LokiConfig{},
		Vendor:      r.mgr.Config.Vendor,
		TLSConfig:   r.mgr.ClusterInfo.GetComponentTLSConfig(),
	}
	var images map[reconcilers.ImageRef]string
	if image != "" {
		images = map[reconcilers.ImageRef]string{
			reconcilers.MainImage: image,
		}
	}
	return reconcilersInfo.NewInstance(images, r.status)
}
