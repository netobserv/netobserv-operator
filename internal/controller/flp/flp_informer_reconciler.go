package flp

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/resources"
)

const (
	informerName      = "flp-informers"
	informerShortName = "informers"
)

type informerReconciler struct {
	*reconcilers.Instance
	deployment     *appsv1.Deployment
	serviceAccount *corev1.ServiceAccount
	rbInformer     *rbacv1.ClusterRoleBinding
}

func newInformerReconciler(cmn *reconcilers.Instance) *informerReconciler {
	rec := informerReconciler{
		Instance:       cmn,
		deployment:     cmn.Managed.NewDeployment(informerName),
		serviceAccount: cmn.Managed.NewServiceAccount(informerName),
		rbInformer:     cmn.Managed.NewCRB(resources.GetClusterRoleBindingName(informerShortName, constants.FLPInformersRole)),
	}
	return &rec
}

func (r *informerReconciler) context(ctx context.Context) context.Context {
	l := log.FromContext(ctx).WithName("informers")
	return log.IntoContext(ctx, l)
}

func (r *informerReconciler) getStatus() *status.Instance {
	return &r.Status
}

func (r *informerReconciler) reconcile(ctx context.Context, desired *flowslatest.FlowCollector, _ *metricslatest.FlowMetricList, _ []sliceslatest.FlowCollectorSlice, _ []flowslatest.SubnetLabel) error {
	// Retrieve current owned objects
	err := r.Managed.FetchAll(ctx)
	if err != nil {
		return err
	}

	if desired.Spec.OnHold() {
		r.Status.SetUnused("FlowCollector is on hold")
		r.Managed.TryDeleteAll(ctx)
		return nil
	}

	builder := newInformerBuilder(r.Instance, &desired.Spec)

	// Reconcile ServiceAccount
	if err := r.reconcileServiceAccount(ctx, &builder); err != nil {
		return err
	}

	// Reconcile RBAC
	if err := r.reconcilePermissions(ctx, &builder); err != nil {
		return err
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &builder); err != nil {
		return err
	}

	return nil
}

func (r *informerReconciler) reconcileServiceAccount(ctx context.Context, builder *informerBuilder) error {
	if !r.Managed.Exists(r.serviceAccount) {
		return r.CreateOwned(ctx, builder.serviceAccount())
	} // We only configure name, update is not needed for now
	return nil
}

func (r *informerReconciler) reconcilePermissions(ctx context.Context, builder *informerBuilder) error {
	// Informers
	r.rbInformer = resources.GetClusterRoleBinding(r.Namespace, informerShortName, informerName, informerName, constants.FLPInformersRole)
	if err := r.ReconcileClusterRoleBinding(ctx, r.rbInformer); err != nil {
		return err
	}
	return nil
}

func (r *informerReconciler) reconcileDeployment(ctx context.Context, builder *informerBuilder) error {
	report := helper.NewChangeReport("FLP informers Deployment")
	defer report.LogIfNeeded(ctx)

	desiredDep, err := builder.deployment()
	if err != nil {
		return err
	}

	return reconcilers.ReconcileDeployment(
		ctx,
		r.Instance,
		r.deployment,
		desiredDep,
		informerName,
		false,
		&report,
	)
}
