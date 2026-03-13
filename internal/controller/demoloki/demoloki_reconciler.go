package demoloki

import (
	"context"
	"fmt"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
)

// LReconciler reconciles the current loki state with the desired configuration
type LReconciler struct {
	*reconcilers.Instance
	configMap  *corev1.ConfigMap
	deployment *appsv1.Deployment
	pvc        *corev1.PersistentVolumeClaim
	service    *corev1.Service
}

func NewReconciler(cmn *reconcilers.Instance) LReconciler {
	rec := LReconciler{
		Instance:   cmn,
		configMap:  cmn.Managed.NewConfigMap(configMapName),
		deployment: cmn.Managed.NewDeployment(constants.LokiDev),
		pvc:        cmn.Managed.NewPersistentVolumeClaim(storeVolume),
		service:    cmn.Managed.NewService(constants.LokiDev),
	}
	return rec
}

// Reconcile is the reconciler entry point to reconcile the current loki state with the desired configuration
func (r *LReconciler) Reconcile(ctx context.Context, desired *flowslatest.FlowCollector) error {
	l := log.FromContext(ctx).WithName("demo-loki")
	ctx = log.IntoContext(ctx, l)

	defer r.Status.Commit(ctx, r.Client)

	err := r.reconcile(ctx, desired)
	if err != nil {
		l.Error(err, "Demo Loki reconcile failure")
		// Set status failure unless it was already set
		if !r.Status.HasFailure() {
			r.Status.SetFailure("DemoLokiError", err.Error())
		}
		return err
	}

	return nil
}

func (r *LReconciler) reconcile(ctx context.Context, desired *flowslatest.FlowCollector) error {
	// Retrieve current owned objects
	err := r.Managed.FetchAll(ctx)
	if err != nil {
		return err
	}

	if desired.Spec.UseLokiDev() {
		// Create object builder
		builder := newBuilder(r.Instance, &desired.Spec, constants.LokiDev)

		cmDigest, err := r.reconcileConfigMap(ctx, &builder)
		if err != nil {
			return err
		}

		if err = r.reconcilePVC(ctx, &builder); err != nil {
			return err
		}

		if err = r.reconcileDeployment(ctx, &builder, constants.LokiDev, cmDigest); err != nil {
			return err
		}

		if err = r.reconcileServices(ctx, &builder, constants.LokiDev); err != nil {
			return err
		}
	} else {
		// delete any existing owned object (only if they were created by netobserv)
		if err := r.DeleteIfOwned(ctx, r.configMap); err != nil {
			return err
		}
		if err := r.DeleteIfOwned(ctx, r.deployment); err != nil {
			return err
		}
		if err := r.DeleteIfOwned(ctx, r.pvc); err != nil {
			return err
		}
		if err := r.DeleteIfOwned(ctx, r.service); err != nil {
			return err
		}
		r.Status.SetUnused("Demo mode not enabled")
	}

	return nil
}

func (r *LReconciler) reconcileConfigMap(ctx context.Context, builder *builder) (string, error) {
	newCM, configDigest, err := builder.configMap()
	if err != nil {
		return "", err
	}
	if !r.Managed.Exists(r.configMap) {
		if err := r.CreateOwned(ctx, newCM); err != nil {
			return "", err
		}
	} else if !reflect.DeepEqual(newCM.Data, r.configMap.Data) {
		if err := r.UpdateIfOwned(ctx, r.configMap, newCM); err != nil {
			return "", err
		}
	}
	return configDigest, nil
}

func (r *LReconciler) reconcilePVC(ctx context.Context, builder *builder) error {
	l := log.FromContext(ctx)
	desiredPVC := builder.persistentVolumeClaim()
	if !r.Managed.Exists(r.pvc) {
		if err := r.CreateOwned(ctx, desiredPVC); err != nil {
			return err
		}
	} else {
		// PVC specs are immutable, so we can only check if the critical fields match
		// and log an error if they don't (we cannot update them)
		report := helper.NewChangeReport("PVC spec")
		if helper.PersistentVolumeClaimSpecChanged(r.pvc, desiredPVC, &report) {
			report.LogIfNeeded(ctx)
			l.Error(fmt.Errorf("PVC spec mismatch detected"),
				"PersistentVolumeClaim has immutable fields that differ from desired spec",
				"pvc", r.pvc.Name,
				"desiredSpec", desiredPVC.Spec,
				"currentSpec", r.pvc.Spec,
			)
			// Note: We cannot update PVCs as their specs are immutable
			// The user would need to delete and recreate the PVC manually if changes are needed
		}
	}
	return nil
}

func (r *LReconciler) reconcileDeployment(ctx context.Context, builder *builder, name string, cm string) error {
	report := helper.NewChangeReport("Loki deployment")
	defer report.LogIfNeeded(ctx)

	return reconcilers.ReconcileDeployment(
		ctx,
		r.Instance,
		r.deployment,
		builder.deployment(name, cm),
		name,
		false,
		&report,
	)
}

func (r *LReconciler) reconcileServices(ctx context.Context, builder *builder, name string) error {
	report := helper.NewChangeReport("Loki services")
	defer report.LogIfNeeded(ctx)

	if err := r.ReconcileService(ctx, r.service, builder.service(name), &report); err != nil {
		return err
	}
	return nil
}
