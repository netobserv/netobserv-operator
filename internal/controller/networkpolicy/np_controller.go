package networkpolicy

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/enqueuer"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
)

type Reconciler struct {
	client.Client
	mgr    *manager.Manager
	ctrlQ  enqueuer.FilteredDynamic
	status status.Instance
}

func Start(ctx context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	log := log.FromContext(ctx)
	log.Info("Starting Network Policy controller")
	r := Reconciler{
		Client: mgr.Client,
		mgr:    mgr,
		status: mgr.Status.ForComponent(status.NetworkPolicy),
	}
	controller, err := ctrl.NewControllerManagedBy(mgr).
		For(&flowslatest.FlowCollector{}, reconcilers.IgnoreStatusChange).
		Named("networkPolicy").
		Build(&r)
	if err != nil {
		return nil, err
	}
	r.ctrlQ = mgr.NewDynamicControllerEnqueuer("networkPolicy-managed", controller)
	return nil, nil
}

// Reconcile is the controller entry point for reconciling current state with desired state.
// It manages the controller status at a high level. Business logic is delegated into `reconcile`.
func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	l := log.Log.WithName("networkpolicy") // clear context (too noisy)
	ctx = log.IntoContext(ctx, l)

	// Get flowcollector & create dedicated client
	clh, desired, err := helper.NewFlowCollectorClientHelper(ctx, r.Client)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get FlowCollector: %w", err)
	}
	r.ctrlQ.ResetActiveWatches()
	if desired == nil {
		// Delete case
		return ctrl.Result{}, nil
	}

	commit := r.status.Reset()
	defer commit(ctx, r.Client)

	hasNP, err := r.reconcile(ctx, clh, desired)
	if err != nil {
		l.Error(err, "Network policy reconcile failure")
		// Set status failure unless it was already set
		if !r.status.HasFailure() {
			r.status.SetFailure("NetworkPolicyError", err.Error())
		}
		return ctrl.Result{}, err
	} else if !hasNP {
		r.status.SetUnused("Network policy is disabled")
	} else {
		r.status.SetReady()
	}

	return ctrl.Result{}, nil
}

// Returns true if policies are enabled
func (r *Reconciler) reconcile(ctx context.Context, clh *helper.Client, desired *flowslatest.FlowCollector) (bool, error) {
	npName, desiredNp := buildMainNetworkPolicy(desired, r.mgr)
	if err := reconcilers.ReconcileNetworkPolicyWithEnqueuer(ctx, r.ctrlQ, clh, npName, desiredNp); err != nil {
		return false, err
	}

	privilegedNpName, desiredPrivilegedNp := buildPrivilegedNetworkPolicy(desired, r.mgr)
	if err := reconcilers.ReconcileNetworkPolicyWithEnqueuer(ctx, r.ctrlQ, clh, privilegedNpName, desiredPrivilegedNp); err != nil {
		return false, err
	}

	return desiredNp != nil, nil
}
