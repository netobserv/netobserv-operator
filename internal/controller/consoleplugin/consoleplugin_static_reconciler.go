package consoleplugin

import (
	"context"
	"fmt"

	osv1 "github.com/openshift/api/console/v1"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/netpol"
)

type StaticReconciler struct {
	CPReconciler
	managerConfig *manager.Config
	netpol        *networkingv1.NetworkPolicy
}

func NewStaticReconciler(cmn *reconcilers.Instance, cfg *manager.Config) StaticReconciler {
	return StaticReconciler{
		CPReconciler: CPReconciler{
			Instance:       cmn,
			deployment:     cmn.Managed.NewDeployment(constants.StaticPluginName),
			service:        cmn.Managed.NewService(constants.StaticPluginName),
			serviceAccount: cmn.Managed.NewServiceAccount(constants.StaticPluginName),
		},
		netpol:        cmn.Managed.NewNetworkPolicy(constants.StaticPluginName),
		managerConfig: cfg,
	}
}

func (r *StaticReconciler) ReconcileStaticPlugin(ctx context.Context, enable bool) error {
	// Retrieve toleration from subscription
	var sched *flowslatest.SchedulingConfig
	subName := r.managerConfig.StaticPluginConfig.InheritTolerationFromSubscription
	if subName != "" {
		sub := olm.Subscription{}
		if err := r.Client.Get(ctx, types.NamespacedName{Name: subName, Namespace: r.Namespace}, &sub); err != nil {
			return err
		}
		if sub.Spec != nil && sub.Spec.Config != nil {
			sched = &flowslatest.SchedulingConfig{
				Tolerations:  sub.Spec.Config.Tolerations,
				NodeSelector: sub.Spec.Config.NodeSelector,
				Affinity:     sub.Spec.Config.Affinity,
			}
		}
	}

	resources, err := buildStaticPluginResources(&r.managerConfig.StaticPluginConfig)
	if err != nil {
		return fmt.Errorf("building static plugin resources: %w", err)
	}
	// Fake a FlowCollector to create console plugin and expose forms
	return r.reconcileStatic(ctx, &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			NetworkPolicy: flowslatest.NetworkPolicy{
				Enable: r.managerConfig.DeployOperatorNetworkPolicy,
			},
			ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
				Enable:    ptr.To(enable),
				LogLevel:  "info",
				Resources: resources,
				Advanced: &flowslatest.AdvancedPluginConfig{
					Register:   ptr.To(true),
					Scheduling: sched,
				},
			},
		},
	})
}

func buildStaticPluginResources(cfg *manager.StaticPluginConfig) (corev1.ResourceRequirements, error) {
	cpuRequest := cfg.CPURequest
	if cpuRequest == "" {
		cpuRequest = "10m"
	}
	memoryRequest := cfg.MemoryRequest
	if memoryRequest == "" {
		memoryRequest = "64Mi"
	}

	cpuQty, err := resource.ParseQuantity(cpuRequest)
	if err != nil {
		return corev1.ResourceRequirements{}, fmt.Errorf("invalid CPU request %q: %w", cpuRequest, err)
	}
	memoryQty, err := resource.ParseQuantity(memoryRequest)
	if err != nil {
		return corev1.ResourceRequirements{}, fmt.Errorf("invalid memory request %q: %w", memoryRequest, err)
	}
	requests := corev1.ResourceList{
		corev1.ResourceCPU:    cpuQty,
		corev1.ResourceMemory: memoryQty,
	}
	limits := corev1.ResourceList{}
	if cfg.CPULimit != "" {
		cpuLimit, err := resource.ParseQuantity(cfg.CPULimit)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid CPU limit %q: %w", cfg.CPULimit, err)
		}
		limits[corev1.ResourceCPU] = cpuLimit
	}
	if cfg.MemoryLimit != "" {
		memoryLimit, err := resource.ParseQuantity(cfg.MemoryLimit)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("invalid memory limit %q: %w", cfg.MemoryLimit, err)
		}
		limits[corev1.ResourceMemory] = memoryLimit
	}
	return corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}, nil
}

// Reconcile is the reconciler entry point to reconcile the static plugin state with the desired configuration
func (r *StaticReconciler) reconcileStatic(ctx context.Context, desired *flowslatest.FlowCollector) error {
	l := log.FromContext(ctx).WithName("static-console-plugin")
	ctx = log.IntoContext(ctx, l)

	// Retrieve current owned objects
	err := r.Managed.FetchAll(ctx)
	if err != nil {
		return err
	}

	if r.ClusterInfo.HasConsolePlugin() {
		r.checkAutoPatch(ctx, desired, constants.StaticPluginName)

		// Create object builder
		builder := newBuilder(r.Instance, &desired.Spec, constants.StaticPluginName)

		if err = r.reconcileStaticPlugin(ctx, &builder, constants.StaticPluginName); err != nil {
			return err
		}

		if err = r.reconcileDeployment(ctx, &builder, &desired.Spec, constants.StaticPluginName, ""); err != nil {
			return err
		}

		if err = r.reconcileServices(ctx, &builder, constants.StaticPluginName); err != nil {
			return err
		}

		if err = r.reconcileNetpol(ctx, desired); err != nil {
			return err
		}
	} else {
		// delete any existing owned object
		if err := r.Managed.TryDeleteAll(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *StaticReconciler) reconcileStaticPlugin(ctx context.Context, builder *builder, name string) error {
	report := helper.NewChangeReport("ConsolePlugin")
	defer report.LogIfNeeded(ctx)

	oldPlg := osv1.ConsolePlugin{}
	pluginExists := true
	err := r.Get(ctx, types.NamespacedName{Name: name}, &oldPlg)
	if err != nil {
		if apierrors.IsNotFound(err) {
			pluginExists = false
		} else {
			return err
		}
	}

	// Check if objects need update
	consolePlugin := builder.consolePlugin(name, "NetObserv static plugin")
	if !pluginExists {
		// Using Create instead of CreateOwned, because ConsolePlugin being a cluster-scope resource, it cannot receive the operator deployment as an owner
		if err := r.Create(ctx, consolePlugin); err != nil {
			return err
		}
	} else if helper.ConsolePluginChanged(&oldPlg, consolePlugin, &report) {
		// Using Update instead of UpdateIfOwned, because ConsolePlugin being a cluster-scope resource, it cannot receive the operator deployment as an owner
		consolePlugin.SetResourceVersion(oldPlg.GetResourceVersion())
		helper.AddManagedLabel(consolePlugin)
		if err := r.Update(ctx, consolePlugin); err != nil {
			return err
		}
	}
	return nil
}

func (r *StaticReconciler) reconcileNetpol(ctx context.Context, desired *flowslatest.FlowCollector) error {
	cni, err := r.ClusterInfo.GetCNI()
	if err != nil {
		return err
	}

	if !flowslatest.ShouldInstallNetworkPolicy(desired.Spec.NetworkPolicy.Enable, cni) {
		if err := r.Managed.TryDelete(ctx, r.netpol); err != nil {
			return err
		}
		return nil
	}

	ingress := []networkingv1.NetworkPolicyIngressRule{
		netpol.AllowFromOpenShiftConsole(&desired.Spec.ConsolePlugin),
	}

	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.StaticPluginName,
			Namespace: r.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     constants.StaticPluginName,
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constants.StaticPluginName,
				},
			},
			Ingress: ingress,
			Egress:  nil,
		},
	}
	nsname := helper.NamespacedName(policy)

	return reconcilers.ReconcileNetworkPolicy(ctx, &r.Client, nsname, policy)
}
