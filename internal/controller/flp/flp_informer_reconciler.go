package flp

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/roles"
)

const (
	informerName      = "flowlogs-pipeline-informers"
	informerShortName = "informers"
)

type informerReconciler struct {
	*reconcilers.Instance
	deployment     *appsv1.Deployment
	service        *corev1.Service
	serviceAccount *corev1.ServiceAccount
}

func newInformerReconciler(cmn *reconcilers.Instance) *informerReconciler {
	rec := informerReconciler{
		Instance:       cmn,
		deployment:     cmn.Managed.NewDeployment(informerName),
		service:        cmn.Managed.NewService(k8sCacheServiceName),
		serviceAccount: cmn.Managed.NewServiceAccount(informerName),
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
		return fmt.Errorf("failed to fetch all managed resources: %w", err)
	}

	if desired.Spec.OnHold() {
		r.Status.SetUnused("FlowCollector is on hold")
		return r.Managed.TryDeleteAll(ctx)
	}

	// Check if informers are enabled (default: false)
	if !desired.Spec.Processor.IsInformerCacheProxyEnabled() {
		// Informers disabled - cleanup resources and use local informers mode
		r.Status.SetUnused("Centralized informers disabled - using local informers mode")
		return r.Managed.TryDeleteAll(ctx)
	}

	// Informers enabled - proceed with reconciliation
	builder := newInformerBuilder(r.Instance, &desired.Spec)

	// Reconcile ServiceAccount
	if err := r.reconcileServiceAccount(ctx, &builder); err != nil {
		return fmt.Errorf("failed to reconcile service account: %w", err)
	}

	// Reconcile k8scache Service (targets processor pods, managed here so certificates
	// are always driven by informerCacheProxy config regardless of deployment model)
	if err := r.reconcileService(ctx, &builder); err != nil {
		return fmt.Errorf("failed to reconcile k8scache service: %w", err)
	}

	// Reconcile Deployment
	if err := r.reconcileDeployment(ctx, &builder); err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}

	r.Status.SetReady()
	return nil
}

func (r *informerReconciler) reconcileServiceAccount(ctx context.Context, builder *informerBuilder) error {
	if !r.Managed.Exists(r.serviceAccount) {
		if err := r.CreateOwned(ctx, builder.serviceAccount()); err != nil {
			return fmt.Errorf("failed to create service account: %w", err)
		}
	} // We only configure name, update is not needed for now

	// Check installed CRB, and notify any missing one
	if err := roles.CheckHasPermission(ctx, r.Client, r.Namespace, informerName, roles.FLPInformersRole); err != nil {
		return err
	}

	return nil
}

func (r *informerReconciler) reconcileService(ctx context.Context, builder *informerBuilder) error {
	report := helper.NewChangeReport("k8scache Service")
	defer report.LogIfNeeded(ctx)
	return r.ReconcileService(ctx, r.service, builder.service(), &report)
}

func (r *informerReconciler) reconcileDeployment(ctx context.Context, builder *informerBuilder) error {
	report := helper.NewChangeReport("FLP informers Deployment")
	defer report.LogIfNeeded(ctx)

	desiredDep, err := builder.deployment()
	if err != nil {
		return fmt.Errorf("failed to build deployment: %w", err)
	}

	if err := reconcilers.ReconcileDeployment(
		ctx,
		r.Instance,
		r.deployment,
		desiredDep,
		informerName,
		false,
		&report,
	); err != nil {
		return fmt.Errorf("failed to reconcile deployment: %w", err)
	}
	return nil
}
