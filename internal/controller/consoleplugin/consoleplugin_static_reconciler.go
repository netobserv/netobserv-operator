package consoleplugin

import (
	"context"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
)

func NewStaticReconciler(cmn *reconcilers.Instance) CPReconciler {
	rec := CPReconciler{
		Instance:       cmn,
		deployment:     cmn.Managed.NewDeployment(constants.StaticPluginName),
		service:        cmn.Managed.NewService(constants.StaticPluginName),
		serviceAccount: cmn.Managed.NewServiceAccount(constants.StaticPluginName),
	}
	return rec
}

func (r *CPReconciler) ReconcileStaticPlugin(ctx context.Context, enable bool) error {
	// Fake a FlowCollector to create console plugin and expose forms
	return r.reconcileStatic(ctx, &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
				Enable:   ptr.To(enable),
				LogLevel: "info",
				Advanced: &flowslatest.AdvancedPluginConfig{
					Register: ptr.To(true),
				},
			},
		},
	})
}

// Reconcile is the reconciler entry point to reconcile the static plugin state with the desired configuration
func (r *CPReconciler) reconcileStatic(ctx context.Context, desired *flowslatest.FlowCollector) error {
	l := log.FromContext(ctx).WithName("static-console-plugin")
	ctx = log.IntoContext(ctx, l)

	// Retrieve current owned objects
	err := r.Managed.FetchAll(ctx)
	if err != nil {
		return err
	}

	if r.ClusterInfo.HasConsolePlugin() {
		if err = r.checkAutoPatch(ctx, desired, constants.StaticPluginName); err != nil {
			return err
		}
	}

	if r.ClusterInfo.HasConsolePlugin() {
		// Create object builder
		builder := newBuilder(r.Instance, &desired.Spec, constants.StaticPluginName)

		if err = r.reconcilePlugin(ctx, &builder, &desired.Spec, constants.StaticPluginName, "NetObserv static plugin"); err != nil {
			return err
		}

		if err = r.reconcileDeployment(ctx, &builder, &desired.Spec, constants.StaticPluginName, ""); err != nil {
			return err
		}

		if err = r.reconcileServices(ctx, &builder, constants.StaticPluginName); err != nil {
			return err
		}
	} else {
		// delete any existing owned object
		r.Managed.TryDeleteAll(ctx)
	}

	return nil
}
