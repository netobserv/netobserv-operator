package networkpolicy

import (
	"context"
	"fmt"

	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/network-observability-operator/internal/controller/reconcilers"
	"github.com/netobserv/network-observability-operator/internal/pkg/cluster"
	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager/status"
)

type Reconciler struct {
	client.Client
	mgr    *manager.Manager
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
	return nil, ctrl.NewControllerManagedBy(mgr).
		For(&flowslatest.FlowCollector{}, reconcilers.IgnoreStatusChange).
		Named("networkPolicy").
		Owns(&networkingv1.NetworkPolicy{}, reconcilers.UpdateOrDeleteOnlyPred).
		Complete(&r)
}

// Reconcile is the controller entry point for reconciling current state with desired state.
// It manages the controller status at a high level. Business logic is delegated into `reconcile`.
func (r *Reconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	l := log.Log.WithName("networkpolicy") // clear context (too noisy)
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
		l.Error(err, "Network policy reconcile failure")
		// Set status failure unless it was already set
		if !r.status.HasFailure() {
			r.status.SetFailure("NetworkPolicyError", err.Error())
		}
		return ctrl.Result{}, err
	}

	r.status.SetReady()
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcile(ctx context.Context, clh *helper.Client, desired *flowslatest.FlowCollector) error {
	l := log.FromContext(ctx)

	cni, err := r.mgr.ClusterInfo.GetCNI()
	if err != nil {
		return err
	}

	// Get API server endpoint IPs for network policy
	var apiServerIPs []string
	if r.mgr.ClusterInfo.IsOpenShift() {
		apiServerIPs, err = cluster.GetAPIServerEndpointIPs(ctx, r.Client, r.mgr.ClusterInfo)
		if err != nil {
			l.Error(err, "Failed to get API server endpoint IPs")
			return fmt.Errorf("cannot determine API server endpoint IPs: %w", err)
		}
	}

	npName, desiredNp := buildMainNetworkPolicy(desired, r.mgr, cni, apiServerIPs)
	if err := reconcilers.ReconcileNetworkPolicy(ctx, clh, npName, desiredNp); err != nil {
		return err
	}

	privilegedNpName, desiredPrivilegedNp := buildPrivilegedNetworkPolicy(desired, r.mgr, cni)
	return reconcilers.ReconcileNetworkPolicy(ctx, clh, privilegedNpName, desiredPrivilegedNp)
}
