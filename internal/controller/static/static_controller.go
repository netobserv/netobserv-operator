package static

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/network-observability-operator/internal/controller/consoleplugin"
	"github.com/netobserv/network-observability-operator/internal/controller/reconcilers"
	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager/status"
)

var (
	retryBackoff = wait.Backoff{
		Steps:    6,
		Duration: 2 * time.Second,
		Factor:   2,
		Jitter:   0.1,
	}
	clog = log.Log.WithName("static-controller")
)

type Reconciler struct {
	client.Client
	mgr    *manager.Manager
	status status.Instance
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting Static controller")
	r := Reconciler{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.StaticController),
	}

	// Return initReconcile as a post-create hook
	return r.initReconcile, ctrl.NewControllerManagedBy(mgr).
		For(&flowslatest.FlowCollector{}, reconcilers.IgnoreStatusChange).
		Named("staticPlugin").
		Complete(&r)
}

func (r *Reconciler) initReconcile(ctx context.Context) error {
	attempt := 0
	err := retry.OnError(retryBackoff, func(error) bool { return true }, func() error {
		attempt++
		if _, err := r.Reconcile(ctx, ctrl.Request{}); err != nil {
			clog.WithValues("attempt", attempt, "error", err).Info("Initial reconcile: attempt failed")
			return err
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed initial reconcile, all attempts failed: %w", err)
	}
	return nil
}

// Reconcile is the controller entry point for reconciling current state with desired state.
// It manages the controller status at a high level. Business logic is delegated into `reconcile`.
func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	ctx = log.IntoContext(ctx, clog)

	r.status.SetUnknown()
	defer r.status.Commit(ctx, r.Client)

	if r.mgr.ClusterInfo.HasConsolePlugin() {
		// Only deploy static plugin on OpenShift 4.15+
		if !r.mgr.ClusterInfo.IsOpenShift() {
			clog.Info("Skipping static plugin reconciler (not OpenShift)")
		} else if supported, _, err := r.mgr.ClusterInfo.IsOpenShiftVersionAtLeast("4.15.0"); err != nil {
			return ctrl.Result{}, err
		} else if !supported {
			clog.Info("Skipping static plugin reconciler (OpenShift version < 4.15)")
		} else {
			scp, err := helper.NewControllerClientHelper(ctx, r.mgr.Config.Namespace, r.Client)
			if err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to get controller deployment: %w", err)
			}
			staticPluginReconciler := consoleplugin.NewStaticReconciler(r.newDefaultReconcilerInstance(scp))
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

	r.status.SetReady()
	return ctrl.Result{}, nil
}

func (r *Reconciler) newDefaultReconcilerInstance(clh *helper.Client) *reconcilers.Instance {
	// force default namespace
	reconcilersInfo := reconcilers.Common{
		Client:       *clh,
		Namespace:    r.mgr.Config.Namespace,
		ClusterInfo:  r.mgr.ClusterInfo,
		Watcher:      nil,
		Loki:         &helper.LokiConfig{},
		IsDownstream: r.mgr.Config.DownstreamDeployment,
	}
	return reconcilersInfo.NewInstance(map[reconcilers.ImageRef]string{
		reconcilers.MainImage:                r.mgr.Config.ConsolePluginImage,
		reconcilers.ConsolePluginCompatImage: r.mgr.Config.ConsolePluginCompatImage,
	}, r.status)
}
