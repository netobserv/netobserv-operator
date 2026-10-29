package flowcapture

import (
	"context"
	"fmt"
	"time"

	flowcapturev1alpha1 "github.com/netobserv/netobserv-operator/api/flowcapture/v1alpha1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/roles"
	sharedcapture "github.com/netobserv/netobserv-operator/pkg/capture"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const flowCaptureFinalizer = "flows.netobserv.io/flowcapture-finalizer"

const (
	phasePending   = "Pending"
	phaseRunning   = "Running"
	phaseCompleted = "Completed"
	phaseFailed    = "Failed"
)

const (
	conditionStarting  = "Starting"
	conditionRunning   = "Running"
	conditionCompleted = "Completed"
	conditionFailed    = "Failed"
)

type Reconciler struct {
	client.Client
	apiReader client.Reader
	config    *manager.Config
}

func Start(_ context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	if err := ctrl.NewControllerManagedBy(mgr.Manager).
		For(&flowcapturev1alpha1.FlowCapture{}).
		Owns(&appsv1.DaemonSet{}).Owns(&corev1.Pod{}).Owns(&corev1.Service{}).
		Complete(&Reconciler{Client: mgr.Client, apiReader: mgr.GetAPIReader(), config: mgr.Config}); err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var fc flowcapturev1alpha1.FlowCapture
	if err := r.Get(ctx, req.NamespacedName, &fc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if !fc.Spec.Type.Valid() {
		return ctrl.Result{}, fmt.Errorf("unsupported FlowCapture type %q", fc.Spec.Type)
	}
	if !fc.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&fc, flowCaptureFinalizer) {
			return ctrl.Result{}, nil
		}
		if err := r.removeCapturePermissions(ctx, &fc); err != nil {
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(&fc, flowCaptureFinalizer)
		return ctrl.Result{}, r.Update(ctx, &fc)
	}
	if fc.Status.CompletionTime != nil {
		return ctrl.Result{}, nil
	}
	if !controllerutil.ContainsFinalizer(&fc, flowCaptureFinalizer) {
		controllerutil.AddFinalizer(&fc, flowCaptureFinalizer)
		if err := r.Update(ctx, &fc); err != nil {
			return ctrl.Result{}, err
		}
	}
	for _, feature := range fc.Spec.Features {
		if !sharedcapture.Feature(feature).Valid() {
			return ctrl.Result{}, r.setCaptureFailure(ctx, &fc, "InvalidFeature", fmt.Sprintf("unsupported FlowCapture feature %q", feature))
		}
	}
	features := make([]sharedcapture.Feature, len(fc.Spec.Features))
	for i, feature := range fc.Spec.Features {
		features[i] = sharedcapture.Feature(feature)
	}
	if err := sharedcapture.ValidateFeatures(sharedcapture.Type(fc.Spec.Type), features); err != nil {
		return ctrl.Result{}, r.setCaptureFailure(ctx, &fc, "InvalidFeature", err.Error())
	}
	if err := r.ensureNamespaceAvailable(ctx, &fc); err != nil {
		if statusErr := r.setConflictStatus(ctx, &fc, err.Error()); statusErr != nil {
			return ctrl.Result{}, fmt.Errorf("%w (also failed to update status: %w)", err, statusErr)
		}
		return ctrl.Result{}, err
	}
	if err := r.reconcileResources(ctx, &fc); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.updateStatus(ctx, &fc); err != nil {
		return ctrl.Result{}, err
	}
	if fc.Status.Phase == phasePending || fc.Status.Phase == phaseRunning {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *Reconciler) ensureNamespaceAvailable(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	var captures flowcapturev1alpha1.FlowCaptureList
	if err := r.List(ctx, &captures, client.InNamespace(fc.Namespace)); err != nil {
		return fmt.Errorf("cannot check captures in namespace %q: %w", fc.Namespace, err)
	}
	for i := range captures.Items {
		other := &captures.Items[i]
		if other.Name == fc.Name || other.Status.Phase == phaseCompleted || other.Status.Phase == phaseFailed {
			continue
		}
		return fmt.Errorf("namespace %q already has active FlowCapture %q", fc.Namespace, other.Name)
	}

	var cliAgent appsv1.DaemonSet
	if err := r.Get(ctx, client.ObjectKey{Namespace: fc.Namespace, Name: "netobserv-cli"}, &cliAgent); err == nil {
		return fmt.Errorf("namespace %q already has an active CLI capture", fc.Namespace)
	} else if !errors.IsNotFound(err) {
		return fmt.Errorf("cannot check for CLI capture in namespace %q: %w", fc.Namespace, err)
	}

	var collector flowslatest.FlowCollector
	if err := r.Get(ctx, client.ObjectKey{Name: constants.FlowCollectorName.Name}, &collector); err == nil && collector.Spec.Execution.Mode != flowslatest.OnHold {
		operandNamespace := helper.GetOperandsNamespace(&collector.Spec, r.config)
		if fc.Namespace == operandNamespace || fc.Namespace == operandNamespace+constants.EBPFPrivilegedNSSuffix {
			return fmt.Errorf("namespace %q is already used by the active FlowCollector", fc.Namespace)
		}
	} else if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("cannot check for FlowCollector: %w", err)
	}
	return nil
}

func (r *Reconciler) setConflictStatus(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, message string) error {
	return r.setCaptureFailure(ctx, fc, "CaptureConflict", message)
}

func (r *Reconciler) setCaptureFailure(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, reason, message string) error {
	if fc.Status.Phase == phaseFailed && fc.Status.Reason == reason && fc.Status.Message == message {
		return nil
	}
	fc.Status.Phase = phaseFailed
	fc.Status.Reason = reason
	fc.Status.Message = message
	appendPhaseCondition(&fc.Status.Conditions, phaseFailed, reason, message)
	return r.Status().Update(ctx, fc)
}

func (r *Reconciler) updateStatus(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := sharedcapture.NewNames(fc.Name)
	var collector corev1.Pod
	var pod *corev1.Pod
	reader := r.apiReader
	if reader == nil {
		reader = r.Client
	}
	err := reader.Get(ctx, client.ObjectKey{Namespace: fc.Namespace, Name: names.CollectorPod}, &collector)
	if err == nil {
		pod = &collector
	} else if !errors.IsNotFound(err) {
		return err
	}
	phase, reason, message := statusForPod(pod)

	statusChanged := fc.Status.Phase != phase || fc.Status.Reason != reason || fc.Status.Message != message
	conditionChanged := appendPhaseCondition(&fc.Status.Conditions, phase, reason, message)
	if !statusChanged && !conditionChanged {
		return nil
	}
	fc.Status.Phase = phase
	fc.Status.Reason = reason
	fc.Status.Message = message
	if phase == phaseCompleted || phase == phaseFailed {
		if fc.Status.CompletionTime == nil {
			now := metav1.Now()
			fc.Status.CompletionTime = &now
		}
	}
	return r.Status().Update(ctx, fc)
}

func appendPhaseCondition(conditions *[]metav1.Condition, phase, reason, message string) bool {
	typeName := conditionTypeForPhase(phase)
	condition := metav1.Condition{Type: typeName, Status: metav1.ConditionTrue, Reason: reason, Message: message, LastTransitionTime: metav1.Now()}
	for i := range *conditions {
		if (*conditions)[i].Type != typeName {
			continue
		}
		if (*conditions)[i].Status == condition.Status && (*conditions)[i].Reason == reason && (*conditions)[i].Message == message {
			return false
		}
		(*conditions)[i] = condition
		return true
	}
	*conditions = append(*conditions, condition)
	return true
}

func conditionTypeForPhase(phase string) string {
	switch phase {
	case phaseRunning:
		return conditionRunning
	case phaseCompleted:
		return conditionCompleted
	case phaseFailed:
		return conditionFailed
	default:
		return conditionStarting
	}
}

func statusForPod(pod *corev1.Pod) (phase, reason, message string) {
	if pod == nil {
		return phasePending, "CollectorPending", "collector pod is not scheduled yet"
	}
	if pod.Annotations[sharedcapture.CaptureCompletedAnnotation] == "true" {
		return phaseCompleted, "CaptureCompleted", copyOutputMessage(pod)
	}
	switch pod.Status.Phase {
	case corev1.PodPending, corev1.PodUnknown:
		return phasePending, "CollectorPending", "collector pod is not scheduled yet"
	case corev1.PodRunning:
		return phaseRunning, "CaptureRunning", "capture is running"
	case corev1.PodSucceeded:
		return phaseCompleted, "CaptureCompleted", copyOutputMessage(pod)
	case corev1.PodFailed:
		return phaseFailed, "CollectorFailed", "collector pod failed"
	default:
		return phasePending, "CollectorPending", "collector pod is not scheduled yet"
	}
}

func copyOutputMessage(pod *corev1.Pod) string {
	return fmt.Sprintf("collector reached its capture limit; copy output with: oc cp -n %s %s:output ./output", pod.Namespace, pod.Name)
}

func (r *Reconciler) reconcileResources(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := sharedcapture.NewNames(fc.Name)
	name := names.AgentDaemonSet
	agentImage, collectorImage := r.config.EBPFAgentImage, r.config.CLIImage
	if err := r.ensure(ctx, fc, sharedcapture.BuildServiceAccount(name, fc.Namespace)); err != nil {
		return err
	}
	if err := r.reconcileCapturePermissions(ctx, fc.Namespace, name, false); err != nil {
		return err
	}
	if err := r.ensure(ctx, fc, desiredService(fc, names)); err != nil {
		return err
	}
	if err := r.ensure(ctx, fc, desiredCollector(fc, collectorImage, names)); err != nil {
		return err
	}
	return r.ensure(ctx, fc, desiredAgent(fc, agentImage))
}

// reconcileCapturePermissions only updates the preinstalled ClusterRoleBinding
// shells. FlowCapture must never create Roles or RoleBindings at runtime.
func (r *Reconciler) reconcileCapturePermissions(ctx context.Context, namespace, serviceAccount string, remove bool) error {
	for _, roleName := range []string{
		string(roles.FlowCaptureInformersRole),
		string(roles.FlowCaptureHostNetworkRole),
		string(roles.FlowCaptureMonitoringRole),
	} {
		var binding rbacv1.ClusterRoleBinding
		if err := r.Get(ctx, client.ObjectKey{Name: roleName}, &binding); err != nil {
			if remove && errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("cannot get preinstalled ClusterRoleBinding %q: %w", roleName, err)
		}
		subject := rbacv1.Subject{Kind: "ServiceAccount", Name: serviceAccount, Namespace: namespace}
		index := -1
		for i := range binding.Subjects {
			if binding.Subjects[i] == subject {
				index = i
				break
			}
		}
		if remove && index >= 0 {
			binding.Subjects = append(binding.Subjects[:index], binding.Subjects[index+1:]...)
			if err := r.Update(ctx, &binding); err != nil {
				return fmt.Errorf("cannot remove FlowCapture service account from %q: %w", roleName, err)
			}
		} else if !remove && index < 0 {
			binding.Subjects = append(binding.Subjects, subject)
			if err := r.Update(ctx, &binding); err != nil {
				return fmt.Errorf("cannot add FlowCapture service account to %q: %w", roleName, err)
			}
		}
	}
	return nil
}

func (r *Reconciler) removeCapturePermissions(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	return r.reconcileCapturePermissions(ctx, fc.Namespace, sharedcapture.NewNames(fc.Name).AgentDaemonSet, true)
}

func (r *Reconciler) ensure(ctx context.Context, owner *flowcapturev1alpha1.FlowCapture, obj client.Object) error {
	if err := controllerutil.SetControllerReference(owner, obj, r.Scheme()); err != nil {
		return err
	}
	current := obj.DeepCopyObject().(client.Object)
	if err := r.Get(ctx, client.ObjectKeyFromObject(obj), current); err != nil {
		if errors.IsNotFound(err) {
			if err := r.Create(ctx, obj); err != nil && !errors.IsAlreadyExists(err) {
				return err
			}
			return nil
		}
		return err
	}
	if _, isPod := obj.(*corev1.Pod); isPod {
		return nil
	}
	obj.SetResourceVersion(current.GetResourceVersion())
	return r.Update(ctx, obj)
}

func desiredService(fc *flowcapturev1alpha1.FlowCapture, names sharedcapture.Names) *corev1.Service {
	return sharedcapture.BuildCollectorService(names.CollectorService, fc.Namespace, map[string]string{"flowcapture": fc.Name, "component": "collector"})
}

func desiredCollector(fc *flowcapturev1alpha1.FlowCapture, image string, names sharedcapture.Names) *corev1.Pod {
	maxTime := "1h"
	if fc.Spec.MaxTime != nil {
		maxTime = fc.Spec.MaxTime.Duration.String()
	}
	args := sharedcapture.CollectorCommand(sharedcapture.Type(fc.Spec.Type), fc.Namespace, "info", maxTime, fc.Spec.MaxBytes, "background|managed")
	podOptions := &sharedcapture.CollectorPodOptions{Name: names.CollectorPod, Namespace: fc.Namespace, ServiceAccount: names.AgentDaemonSet, Image: image, Command: args}
	pod := sharedcapture.BuildCollectorPod(podOptions)
	pod.Labels["flowcapture"] = fc.Name
	pod.Labels["component"] = "collector"
	pod.Labels["part-of"] = constants.OperatorName
	return pod
}

func desiredAgent(fc *flowcapturev1alpha1.FlowCapture, image string) *appsv1.DaemonSet {
	names := sharedcapture.NewNames(fc.Name)
	sampling := int(fc.Spec.Sampling)
	if sampling == 0 {
		sampling = 1
	}
	features := make([]sharedcapture.Feature, len(fc.Spec.Features))
	for i, feature := range fc.Spec.Features {
		features[i] = sharedcapture.Feature(feature)
	}
	pipeline := sharedcapture.CollectorPipelineConfig(names.CollectorService)
	if sharedcapture.Type(fc.Spec.Type) == sharedcapture.Metrics {
		pipeline = sharedcapture.MetricsPipelineConfig()
	}
	options := &sharedcapture.AgentOptions{Name: names.AgentDaemonSet, Namespace: fc.Namespace, ServiceAccount: names.AgentDaemonSet, Image: image, TargetHost: names.CollectorService, NodeSelector: fc.Spec.NodeSelector, Sampling: sampling, Interfaces: fc.Spec.Interfaces, ExcludeInterfaces: fc.Spec.ExcludeInterfaces, Features: features, Filters: fc.Spec.Filters, Type: sharedcapture.Type(fc.Spec.Type), PipelineConfig: pipeline}
	return sharedcapture.BuildAgentDaemonSet(options)
}
