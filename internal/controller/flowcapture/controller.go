package flowcapture

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	flowcapturev1alpha1 "github.com/netobserv/netobserv-operator/api/flowcapture/v1alpha1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/enqueuer"
	"github.com/netobserv/netobserv-operator/internal/pkg/roles"
	sharedcapture "github.com/netobserv/netobserv-operator/pkg/capture"
	configv1 "github.com/openshift/api/config/v1"
	securityv1 "github.com/openshift/api/security/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const flowCaptureFinalizer = "flows.netobserv.io/flowcapture-finalizer"

const clusterMonitoringLabel = "openshift.io/cluster-monitoring"

const flowCaptureDashboardManagedAnnotation = "flows.netobserv.io/managed"

const (
	flowCaptureNameLabel      = "flows.netobserv.io/flowcapture-name"
	flowCaptureNamespaceLabel = "flows.netobserv.io/flowcapture-namespace"
)

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
	managedQ  enqueuer.Dynamic
}

func Start(_ context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	r := &Reconciler{Client: mgr.Client, apiReader: mgr.GetAPIReader(), config: mgr.Config}
	controller, err := ctrl.NewControllerManagedBy(mgr.Manager).
		For(&flowcapturev1alpha1.FlowCapture{}).
		Build(r)
	if err != nil {
		return nil, err
	}
	r.managedQ = mgr.NewDynamicControllerEnqueuer("flowcapture-managed", controller)
	return nil, nil
}

//nolint:cyclop // Reconciliation intentionally covers the complete capture lifecycle.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if r.managedQ != nil {
		r.managedQ.ResetActiveWatches()
	}
	var fc flowcapturev1alpha1.FlowCapture
	if err := r.Get(ctx, req.NamespacedName, &fc); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	if !fc.Spec.OutputType.Valid() {
		return ctrl.Result{}, fmt.Errorf("unsupported FlowCapture type %q", fc.Spec.OutputType)
	}
	if r.isOpenShift() && fc.DeletionTimestamp.IsZero() {
		if err := r.reconcileFlowCaptureDashboard(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}
	if !fc.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&fc, flowCaptureFinalizer) {
			return ctrl.Result{}, nil
		}
		if err := r.cleanupCaptureResources(ctx, &fc); err != nil {
			return ctrl.Result{}, err
		}
		if r.isOpenShift() {
			if err := r.deleteFlowCaptureDashboard(ctx, &fc); err != nil {
				return ctrl.Result{}, err
			}
		}
		controllerutil.RemoveFinalizer(&fc, flowCaptureFinalizer)
		return ctrl.Result{}, r.Update(ctx, &fc)
	}
	if fc.Status.CompletionTime != nil {
		if err := r.cleanupCompletedCapture(ctx, &fc); err != nil {
			return ctrl.Result{}, err
		}
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
	if err := sharedcapture.ValidateFeatures(sharedcapture.Type(fc.Spec.OutputType), features); err != nil {
		return ctrl.Result{}, r.setCaptureFailure(ctx, &fc, "InvalidFeature", err.Error())
	}
	if err := r.validateCaptureLimits(&fc); err != nil {
		return ctrl.Result{}, r.setCaptureFailure(ctx, &fc, "CaptureLimit", err.Error())
	}
	if err := r.ensureCaptureConcurrency(ctx, &fc); err != nil {
		if statusErr := r.updateCaptureStatus(ctx, &fc, phasePending, "CaptureConcurrencyLimit", err.Error()); statusErr != nil {
			return ctrl.Result{}, fmt.Errorf("%w (also failed to update status: %w)", err, statusErr)
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	if err := r.ensureCaptureConflicts(ctx, &fc); err != nil {
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

func (r *Reconciler) ensureCaptureConcurrency(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	var captures flowcapturev1alpha1.FlowCaptureList
	reader := r.apiReader
	if reader == nil {
		reader = r.Client
	}
	if err := reader.List(ctx, &captures); err != nil {
		return fmt.Errorf("cannot check active FlowCaptures: %w", err)
	}
	if !captureAdmitted(captures.Items, fc, r.maxConcurrentCaptures()) {
		return fmt.Errorf("maximum of %d concurrent FlowCaptures is already running", r.maxConcurrentCaptures())
	}
	return nil
}

func (r *Reconciler) ensureCaptureConflicts(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	if fc.Spec.OutputType == flowcapturev1alpha1.Metrics {
		var allCaptures flowcapturev1alpha1.FlowCaptureList
		if err := r.List(ctx, &allCaptures); err != nil {
			return fmt.Errorf("cannot check active metrics captures: %w", err)
		}
		for i := range allCaptures.Items {
			if other := activeMetricsCapture(&allCaptures.Items[i], fc); other != nil {
				return fmt.Errorf("active metrics FlowCapture %q already exists in namespace %q", other.Name, other.Namespace)
			}
		}
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

func isActiveCapture(fc *flowcapturev1alpha1.FlowCapture) bool {
	return fc.Status.CompletionTime == nil && (fc.Status.Phase == phaseRunning || (fc.Status.Phase == phasePending && fc.Status.Reason != "CaptureConcurrencyLimit"))
}

func captureAdmitted(captures []flowcapturev1alpha1.FlowCapture, current *flowcapturev1alpha1.FlowCapture, limit int) bool {
	if isActiveCapture(current) {
		return true
	}
	preceding := 0
	for i := range captures {
		capture := &captures[i]
		if capture.Name == current.Name && capture.Namespace == current.Namespace {
			continue
		}
		if isPendingAdmission(capture) && capturePrecedes(capture, current) {
			preceding++
		}
	}
	return preceding < limit
}

func isPendingAdmission(fc *flowcapturev1alpha1.FlowCapture) bool {
	return fc.Status.CompletionTime == nil && fc.Status.Phase != phaseCompleted && fc.Status.Phase != phaseFailed
}

func (r *Reconciler) maxConcurrentCaptures() int {
	if r.config == nil || r.config.MaxConcurrentCaptures == 0 {
		return manager.DefaultMaxConcurrentCaptures
	}
	return r.config.MaxConcurrentCaptures
}

func (r *Reconciler) maxCaptureDuration() time.Duration {
	if r.config == nil || r.config.MaxCaptureDuration == 0 {
		return manager.DefaultMaxCaptureDuration
	}
	return r.config.MaxCaptureDuration
}

func (r *Reconciler) validateCaptureLimits(fc *flowcapturev1alpha1.FlowCapture) error {
	if fc.Spec.MaxTime != nil && fc.Spec.MaxTime.Duration > r.maxCaptureDuration() {
		return fmt.Errorf("maxTime %s exceeds the operator limit of %s", fc.Spec.MaxTime.Duration, r.maxCaptureDuration())
	}
	if r.config != nil && r.config.MaxCaptureBytes > 0 && fc.Spec.MaxBytes > r.config.MaxCaptureBytes {
		return fmt.Errorf("maxBytes %d exceeds the operator limit of %d", fc.Spec.MaxBytes, r.config.MaxCaptureBytes)
	}
	return nil
}

func activeMetricsCapture(candidate, current *flowcapturev1alpha1.FlowCapture) *flowcapturev1alpha1.FlowCapture {
	if candidate.Name == current.Name && candidate.Namespace == current.Namespace {
		return nil
	}
	if candidate.Spec.OutputType != flowcapturev1alpha1.Metrics || candidate.Status.Phase == phaseCompleted || candidate.Status.Phase == phaseFailed {
		return nil
	}
	if !capturePrecedes(candidate, current) {
		return nil
	}
	return candidate
}

func capturePrecedes(first, second *flowcapturev1alpha1.FlowCapture) bool {
	if first.CreationTimestamp.Equal(&second.CreationTimestamp) {
		return first.Name < second.Name
	}
	return first.CreationTimestamp.Before(&second.CreationTimestamp)
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
	if sharedcapture.Type(fc.Spec.OutputType) == sharedcapture.Metrics {
		return r.updateMetricsStatus(ctx, fc)
	}
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

func (r *Reconciler) updateMetricsStatus(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := r.captureNames(fc)
	var agent appsv1.DaemonSet
	reader := r.apiReader
	if reader == nil {
		reader = r.Client
	}
	if err := reader.Get(ctx, client.ObjectKey{Namespace: names.AgentNamespace, Name: names.AgentDaemonSet}, &agent); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		return r.updateCaptureStatus(ctx, fc, phasePending, "AgentPending", "agent DaemonSet is not scheduled yet")
	}
	maxTime := r.maxCaptureDuration()
	if fc.Spec.MaxTime != nil {
		maxTime = fc.Spec.MaxTime.Duration
	}
	if time.Now().After(fc.CreationTimestamp.Add(maxTime)) {
		return r.updateCaptureStatus(ctx, fc, phaseCompleted, "CaptureCompleted", "metrics capture reached its capture limit")
	}
	if agent.Status.DesiredNumberScheduled > 0 && agent.Status.NumberReady == agent.Status.DesiredNumberScheduled {
		return r.updateCaptureStatus(ctx, fc, phaseRunning, "CaptureRunning", "capture is running")
	}
	return r.updateCaptureStatus(ctx, fc, phasePending, "AgentPending", "agent DaemonSet is not ready yet")
}

func (r *Reconciler) updateCaptureStatus(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, phase, reason, message string) error {
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
	names := r.captureNames(fc)
	if err := r.ensureCaptureNamespace(ctx); err != nil {
		return err
	}
	agentImage, collectorImage := r.config.EBPFAgentImage, r.config.CaptureImage
	if err := r.reconcileCaptureSetup(ctx, fc, &names); err != nil {
		return err
	}
	if sharedcapture.Type(fc.Spec.OutputType).HasCollector() {
		if err := r.reconcileCollectorResources(ctx, fc, &names, collectorImage); err != nil {
			return err
		}
	} else if err := r.deleteCollectorResources(ctx, fc, &names); err != nil {
		return err
	}
	selector := map[string]string{"flowcapture": names.AgentDaemonSet, "component": "agent"}
	if err := r.ensureUnowned(ctx, sharedcapture.BuildAgentHealthServiceForPort(names.AgentDaemonSet, names.AgentNamespace, selector, names.AgentMetricsPort)); err != nil {
		return err
	}
	if err := r.ensureUnowned(ctx, sharedcapture.BuildAgentHealthServiceMonitorFor(names.AgentDaemonSet, names.AgentNamespace, map[string]string{"app": names.AgentDaemonSet + "-health"})); err != nil {
		return err
	}
	if sharedcapture.Type(fc.Spec.OutputType) == sharedcapture.Metrics {
		metricsService := sharedcapture.BuildMetricsServiceFor(names.AgentDaemonSet, names.AgentNamespace, selector)
		if r.managedQ != nil {
			if err := r.managedQ.EnqueueOnChange(ctx, metricsService, flowCaptureRequest(fc), reconcilers.IgnoreStatusChangeEventFilter); err != nil {
				return err
			}
		}
		if err := r.ensureUnowned(ctx, metricsService); err != nil {
			return err
		}
		if err := r.ensureUnowned(ctx, sharedcapture.BuildMetricsServiceMonitorFor(names.AgentDaemonSet, names.AgentNamespace, map[string]string{"app": names.AgentDaemonSet})); err != nil {
			return err
		}
	}
	var subnets map[string][]string
	if flowCaptureAutoDetectSubnets(fc) {
		var err error
		subnets, err = r.discoverSubnets(ctx)
		if err != nil {
			return r.setCaptureFailure(ctx, fc, "SubnetDiscoveryFailed", err.Error())
		}
	}
	agent := desiredAgent(fc, agentImage, &names, subnets)
	if r.managedQ != nil {
		if err := r.managedQ.EnqueueOnChange(ctx, agent, flowCaptureRequest(fc), reconcilers.IgnoreStatusChangeEventFilter); err != nil {
			return err
		}
	}
	return r.ensureUnowned(ctx, agent)
}

func flowCaptureAutoDetectSubnets(fc *flowcapturev1alpha1.FlowCapture) bool {
	return fc.Spec.SubnetLabels.AutoDetect != nil && *fc.Spec.SubnetLabels.AutoDetect
}

func (r *Reconciler) reconcileCollectorResources(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, names *captureNames, image string) error {
	if r.isOpenShift() {
		if err := r.ensureClusterMonitoringLabel(ctx, fc.Namespace); err != nil {
			return err
		}
	}
	if err := r.ensure(ctx, fc, desiredService(fc, names.Names)); err != nil {
		return err
	}
	if err := r.ensure(ctx, fc, sharedcapture.BuildCollectorMetricsServiceMonitorFor(names.CollectorService, fc.Namespace, map[string]string{"flowcapture": fc.Name, "component": "collector"}, names.AgentDaemonSet)); err != nil {
		return err
	}
	if r.managedQ != nil {
		if err := r.managedQ.EnqueueOnChange(ctx, desiredService(fc, names.Names), flowCaptureRequest(fc), reconcilers.IgnoreStatusChangeEventFilter); err != nil {
			return err
		}
	}
	for _, object := range []client.Object{
		desiredCollectorServiceAccount(names),
		desiredCollectorRole(names),
		desiredCollectorRoleBinding(names),
		r.desiredCollector(fc, image, names),
	} {
		if err := r.ensure(ctx, fc, object); err != nil {
			return err
		}
	}
	networkPolicy, err := r.desiredCollectorNetworkPolicy(ctx, fc, names.AgentNamespace)
	if err != nil {
		return err
	}
	return r.ensure(ctx, fc, networkPolicy)
}

func (r *Reconciler) reconcileCaptureSetup(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, names *captureNames) error {
	if r.isOpenShift() {
		if err := r.ensureUnowned(ctx, desiredFlowCaptureSCC()); err != nil {
			return fmt.Errorf("cannot ensure FlowCapture SCC: %w", err)
		}
		if fc.Spec.TLSPlaintext != nil {
			if err := r.ensureUnowned(ctx, desiredFlowCapturePlaintextSCC()); err != nil {
				return fmt.Errorf("cannot ensure FlowCapture plaintext SCC: %w", err)
			}
		}
	}
	// The collector uses a capture-specific, minimally-permissioned
	// ServiceAccount. Only the agent ServiceAccount below is granted
	// host-network/capture permissions.
	agentServiceAccount := sharedcapture.BuildServiceAccount(names.AgentDaemonSet, names.AgentNamespace)
	agentServiceAccount.Labels = flowCaptureLabels(names.AgentDaemonSet, "agent", names.CollectorNamespace, fc.Name)
	if err := r.ensureUnowned(ctx, agentServiceAccount); err != nil {
		return err
	}
	if err := r.reconcileCapturePermissions(ctx, names.AgentNamespace, names.AgentDaemonSet, fc.Spec.TLSPlaintext != nil, false); err != nil {
		return err
	}
	if r.isOpenShift() {
		if err := r.ensureClusterMonitoringLabel(ctx, names.AgentNamespace); err != nil {
			return err
		}
		if err := r.ensureUnowned(ctx, roles.GetExposeMetricsRoleBinding(names.AgentNamespace)); err != nil {
			return err
		}
	}
	return nil
}

// desiredFlowCaptureSCC is reconciled by the operator on OpenShift. SCCs are not
// supported as resources in an OLM InstallPlan, so they must not be shipped as
// bundle manifests. The same object remains in the Kustomize manifests for
// deployments that apply those manifests directly.
func desiredFlowCaptureSCC() *securityv1.SecurityContextConstraints {
	return &securityv1.SecurityContextConstraints{
		ObjectMeta:               metav1.ObjectMeta{Name: "netobserv-flowcapture"},
		AllowHostDirVolumePlugin: true,
		AllowHostIPC:             false,
		AllowHostNetwork:         true,
		AllowHostPID:             false,
		AllowHostPorts:           true,
		AllowPrivilegeEscalation: ptr.To(false),
		AllowPrivilegedContainer: false,
		AllowedCapabilities: []corev1.Capability{
			"BPF", "PERFMON", "NET_ADMIN", "SYS_ADMIN",
		},
		FSGroup:            securityv1.FSGroupStrategyOptions{Type: securityv1.FSGroupStrategyRunAsAny},
		RunAsUser:          securityv1.RunAsUserStrategyOptions{Type: securityv1.RunAsUserStrategyRunAsAny},
		SELinuxContext:     securityv1.SELinuxContextStrategyOptions{Type: securityv1.SELinuxStrategyRunAsAny},
		SupplementalGroups: securityv1.SupplementalGroupsStrategyOptions{Type: securityv1.SupplementalGroupsStrategyRunAsAny},
		UserNamespaceLevel: securityv1.NamespaceLevelAllowHost,
		Volumes: []securityv1.FSType{
			securityv1.FSTypeConfigMap, securityv1.FSTypeDownwardAPI, securityv1.FSTypeEmptyDir,
			securityv1.FSTypeHostPath, securityv1.FSProjected, securityv1.FSTypeSecret,
		},
	}
}

func desiredFlowCapturePlaintextSCC() *securityv1.SecurityContextConstraints {
	scc := desiredFlowCaptureSCC()
	scc.Name = "netobserv-flowcapture-plaintext"
	scc.AllowHostPID = true
	scc.AllowedCapabilities = append(scc.AllowedCapabilities, "SYS_PTRACE")
	return scc
}

func flowCaptureRequest(fc *flowcapturev1alpha1.FlowCapture) reconcile.Request {
	return reconcile.Request{NamespacedName: client.ObjectKeyFromObject(fc)}
}

type captureNames struct {
	sharedcapture.Names
	AgentNamespace          string
	CollectorServiceAccount string
	CollectorNamespace      string
	AgentMetricsPort        int32
}

func (r *Reconciler) captureNamespace() string {
	return helper.GetOperandsNamespace(&flowslatest.FlowCollectorSpec{}, r.config) + constants.EBPFPrivilegedNSSuffix
}

func (r *Reconciler) isOpenShift() bool {
	return r.config.Vendor == constants.VendorOpenShift || r.config.Vendor == constants.VendorOpenShiftDownstream
}

func (r *Reconciler) captureNames(fc *flowcapturev1alpha1.FlowCapture) captureNames {
	names := sharedcapture.NewNames(fc.Name)
	// Agent resources are shared by all FlowCaptures, so names based only on
	// the request name would collide when two namespaces use the same name.
	digest := sha256.Sum256([]byte(fc.Namespace + "/" + fc.Name))
	names.AgentDaemonSet = fmt.Sprintf("fc-%x-agent", digest[:5])
	metricsPort := int32(20000 + (uint16(digest[0])<<8|uint16(digest[1]))%10000)
	return captureNames{Names: names, AgentNamespace: r.captureNamespace(), CollectorServiceAccount: fmt.Sprintf("fc-%x-collector", digest[:5]), CollectorNamespace: fc.Namespace, AgentMetricsPort: metricsPort}
}

func (r *Reconciler) ensureCaptureNamespace(ctx context.Context) error {
	namespace := r.captureNamespace()
	current := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: namespace}, current); err != nil {
		if !errors.IsNotFound(err) {
			return fmt.Errorf("cannot get FlowCapture agent namespace %q: %w", namespace, err)
		}
		return r.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				"app":                                constants.OperatorName,
				"pod-security.kubernetes.io/enforce": "privileged",
			},
		}})
	}
	updated := current.DeepCopy()
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}
	changed := false
	for key, value := range map[string]string{
		"app":                                constants.OperatorName,
		"pod-security.kubernetes.io/enforce": "privileged",
	} {
		if updated.Labels[key] != value {
			updated.Labels[key] = value
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return r.Update(ctx, updated)
}

func (r *Reconciler) discoverSubnets(ctx context.Context) (map[string][]string, error) {
	if !r.isOpenShift() {
		return nil, fmt.Errorf("automatic subnet detection is not supported on this cluster")
	}

	var configMap corev1.ConfigMap
	if err := r.Get(ctx, types.NamespacedName{Name: "cluster-config-v1", Namespace: "kube-system"}, &configMap); err != nil {
		return nil, fmt.Errorf("cannot get OpenShift cluster network configuration: %w", err)
	}
	machineCIDRs, err := sharedcapture.ParseMachineNetworkCIDRs(configMap.Data["install-config"])
	if err != nil {
		return nil, fmt.Errorf("cannot parse OpenShift machine network configuration: %w", err)
	}

	var network configv1.Network
	if err := r.Get(ctx, types.NamespacedName{Name: "cluster"}, &network); err != nil {
		return nil, fmt.Errorf("cannot get OpenShift Network configuration: %w", err)
	}
	result := map[string][]string{"Machines": machineCIDRs}
	for _, clusterNetwork := range network.Spec.ClusterNetwork {
		if clusterNetwork.CIDR != "" {
			result["Pods"] = append(result["Pods"], clusterNetwork.CIDR)
		}
	}
	for _, serviceNetwork := range network.Spec.ServiceNetwork {
		if serviceNetwork != "" {
			result["Services"] = append(result["Services"], serviceNetwork)
		}
	}
	return result, nil
}

func (r *Reconciler) ensureClusterMonitoringLabel(ctx context.Context, namespace string) error {
	var ns corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: namespace}, &ns); err != nil {
		return fmt.Errorf("cannot get FlowCapture namespace %q: %w", namespace, err)
	}
	if !setClusterMonitoringLabel(&ns) {
		return nil
	}
	if err := r.Update(ctx, &ns); err != nil {
		return fmt.Errorf("cannot enable cluster monitoring for FlowCapture namespace %q: %w", namespace, err)
	}
	return nil
}

func setClusterMonitoringLabel(ns *corev1.Namespace) bool {
	if ns.Labels[clusterMonitoringLabel] == "true" {
		return false
	}
	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}
	ns.Labels[clusterMonitoringLabel] = "true"
	return true
}

func (r *Reconciler) reconcileFlowCaptureDashboard(ctx context.Context) error {
	desired := sharedcapture.BuildFlowCaptureDashboard()
	for name, dashboardJSON := range desired.Data {
		if !json.Valid([]byte(dashboardJSON)) {
			return fmt.Errorf("generated FlowCapture dashboard %q is not valid JSON", name)
		}
	}

	key := client.ObjectKeyFromObject(desired)
	current := &corev1.ConfigMap{}
	err := r.Get(ctx, key, current)
	if errors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return fmt.Errorf("cannot get FlowCapture dashboard ConfigMap: %w", err)
	}
	if current.Annotations[flowCaptureDashboardManagedAnnotation] != "flowcapture" {
		return fmt.Errorf("FlowCapture dashboard is not managed by the FlowCapture controller")
	}
	if equality.Semantic.DeepEqual(current.Data, desired.Data) && equality.Semantic.DeepEqual(current.Labels, desired.Labels) {
		return nil
	}
	current.Data = desired.Data
	current.Labels = desired.Labels
	return r.Update(ctx, current)
}

func (r *Reconciler) deleteFlowCaptureDashboard(ctx context.Context, _ *flowcapturev1alpha1.FlowCapture) error {
	var captures flowcapturev1alpha1.FlowCaptureList
	if err := r.List(ctx, &captures); err != nil {
		return fmt.Errorf("cannot list FlowCaptures for dashboard cleanup: %w", err)
	}
	if len(captures.Items) > 1 {
		return nil
	}

	current := &corev1.ConfigMap{}
	key := client.ObjectKey{Name: "netobserv-flowcaptures", Namespace: "openshift-config-managed"}
	if err := r.Get(ctx, key, current); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot get FlowCapture dashboard ConfigMap for deletion: %w", err)
	}
	if current.Annotations[flowCaptureDashboardManagedAnnotation] != "flowcapture" {
		return nil
	}
	if err := r.Delete(ctx, current, client.Preconditions{UID: &current.UID}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("cannot delete FlowCapture dashboard ConfigMap: %w", err)
	}
	return nil
}

// reconcileCapturePermissions only updates the preinstalled ClusterRoleBinding
// shells. FlowCapture must never create Roles or RoleBindings at runtime.
func (r *Reconciler) reconcileCapturePermissions(ctx context.Context, namespace, serviceAccount string, plaintext, remove bool) error {
	roleNames := []string{
		string(roles.FlowCaptureInformersRole),
		string(roles.FlowCaptureSCCRole),
		string(roles.FlowCapturePlaintextSCCRole),
	}
	for _, roleName := range roleNames {
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
		wantSubject := roleName != string(roles.FlowCapturePlaintextSCCRole) || plaintext
		if (!wantSubject || remove) && index >= 0 {
			binding.Subjects = append(binding.Subjects[:index], binding.Subjects[index+1:]...)
			if err := r.Update(ctx, &binding); err != nil {
				return fmt.Errorf("cannot remove FlowCapture service account from %q: %w", roleName, err)
			}
		} else if wantSubject && !remove && index < 0 {
			binding.Subjects = append(binding.Subjects, subject)
			if err := r.Update(ctx, &binding); err != nil {
				return fmt.Errorf("cannot add FlowCapture service account to %q: %w", roleName, err)
			}
		}
	}
	return nil
}

func (r *Reconciler) removeCapturePermissions(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := r.captureNames(fc)
	return r.reconcileCapturePermissions(ctx, names.AgentNamespace, names.AgentDaemonSet, fc.Spec.TLSPlaintext != nil, true)
}

func (r *Reconciler) deleteAgentDaemonSet(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := r.captureNames(fc)
	name := names.AgentDaemonSet
	agent := &appsv1.DaemonSet{}
	key := client.ObjectKey{Name: name, Namespace: names.AgentNamespace}
	if err := r.Get(ctx, key, agent); err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot get completed FlowCapture agent DaemonSet %q: %w", name, err)
	}
	if !isManagedCaptureResource(agent, &names) {
		return nil
	}
	if err := r.Delete(ctx, agent, client.Preconditions{UID: &agent.UID}); err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("cannot delete completed FlowCapture agent DaemonSet %q: %w", name, err)
	}
	return nil
}

func (r *Reconciler) cleanupCompletedCapture(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	if err := r.deleteAgentDaemonSet(ctx, fc); err != nil {
		return err
	}
	if err := r.deleteAgentResources(ctx, fc); err != nil {
		return err
	}
	return r.removeCapturePermissions(ctx, fc)
}

func (r *Reconciler) cleanupCaptureResources(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	if err := r.cleanupCompletedCapture(ctx, fc); err != nil {
		return err
	}
	return r.removeCapturePermissions(ctx, fc)
}

func (r *Reconciler) deleteCollectorResources(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture, names *captureNames) error {
	objects := []client.Object{
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorPod, Namespace: names.CollectorNamespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorService, Namespace: names.CollectorNamespace}},
		&monitoringv1.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorService + "-metrics", Namespace: names.CollectorNamespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace}},
		&networkingv1.NetworkPolicy{ObjectMeta: metav1.ObjectMeta{Name: fc.Name + "-collector", Namespace: names.CollectorNamespace}},
	}
	for _, object := range objects {
		current := object.DeepCopyObject().(client.Object)
		if err := r.Get(ctx, client.ObjectKeyFromObject(object), current); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("cannot get metrics collector resource %q: %w", object.GetName(), err)
		}
		if !metav1.IsControlledBy(current, fc) {
			continue
		}
		uid := current.GetUID()
		if err := r.Delete(ctx, current, client.Preconditions{UID: &uid}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("cannot delete metrics collector resource %q: %w", object.GetName(), err)
		}
	}
	return nil
}

func (r *Reconciler) deleteAgentResources(ctx context.Context, fc *flowcapturev1alpha1.FlowCapture) error {
	names := r.captureNames(fc)
	objects := []client.Object{
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet, Namespace: names.AgentNamespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet, Namespace: names.AgentNamespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet + "-health", Namespace: names.AgentNamespace}},
		&monitoringv1.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet, Namespace: names.AgentNamespace}},
		&monitoringv1.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet + "-health", Namespace: names.AgentNamespace}},
	}
	for _, obj := range objects {
		current := obj.DeepCopyObject().(client.Object)
		if err := r.Get(ctx, client.ObjectKeyFromObject(obj), current); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("cannot get completed FlowCapture resource %q: %w", obj.GetName(), err)
		}
		if !isManagedCaptureResource(current, &names) {
			continue
		}
		uid := current.GetUID()
		if err := r.Delete(ctx, current, client.Preconditions{UID: &uid}); err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("cannot delete completed FlowCapture resource %q: %w", obj.GetName(), err)
		}
	}
	return nil
}

func isManagedCaptureResource(obj client.Object, names *captureNames) bool {
	switch resource := obj.(type) {
	case *appsv1.DaemonSet:
		return resource.Labels["flowcapture"] == names.AgentDaemonSet && resource.Labels["component"] == "agent"
	case *corev1.ServiceAccount:
		return resource.Labels["flowcapture"] == names.AgentDaemonSet && resource.Labels["component"] == "agent"
	case *corev1.Service:
		return resource.Labels["app"] == resource.Name && (resource.Name == names.AgentDaemonSet || resource.Name == names.AgentDaemonSet+"-health")
	case *monitoringv1.ServiceMonitor:
		app := resource.Spec.Selector.MatchLabels["app"]
		return app == names.AgentDaemonSet || app == names.AgentDaemonSet+"-health"
	default:
		return false
	}
}

func (r *Reconciler) ensure(ctx context.Context, owner *flowcapturev1alpha1.FlowCapture, obj client.Object) error {
	if err := controllerutil.SetControllerReference(owner, obj, r.Scheme()); err != nil {
		return err
	}
	return r.ensureObject(ctx, obj)
}

func (r *Reconciler) ensureUnowned(ctx context.Context, obj client.Object) error {
	return r.ensureObject(ctx, obj)
}

func (r *Reconciler) ensureObject(ctx context.Context, obj client.Object) error {
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
	if resourcesEqual(obj, current) {
		return nil
	}
	obj.SetResourceVersion(current.GetResourceVersion())
	return r.Update(ctx, obj)
}

// resourcesEqual compares the fields managed by the controller while ignoring
// fields populated by the API server or admission controllers. In particular,
// OpenShift adds image-pull secrets to ServiceAccounts after creation; writing
// the original desired object back on every reconcile would otherwise create a
// reconcile loop.
func resourcesEqual(desired, current client.Object) bool {
	d := desired.DeepCopyObject().(client.Object)
	c := current.DeepCopyObject().(client.Object)

	for _, obj := range []client.Object{d, c} {
		obj.GetObjectKind().SetGroupVersionKind(schema.GroupVersionKind{})
		obj.SetUID("")
		obj.SetResourceVersion("")
		obj.SetGeneration(0)
		obj.SetCreationTimestamp(metav1.Time{})
		obj.SetManagedFields(nil)
		obj.SetDeletionTimestamp(nil)
		obj.SetDeletionGracePeriodSeconds(nil)
	}

	switch d := d.(type) {
	case *corev1.ServiceAccount:
		c := c.(*corev1.ServiceAccount)
		// OpenShift manages pull-secret annotations as well as the references
		// below; they are not controller-owned fields.
		d.SetAnnotations(nil)
		c.SetAnnotations(nil)
		d.Secrets = nil
		d.ImagePullSecrets = nil
		c.Secrets = nil
		c.ImagePullSecrets = nil
	case *corev1.Service:
		c := c.(*corev1.Service)
		d.Status = corev1.ServiceStatus{}
		c.Status = corev1.ServiceStatus{}
		// These fields are assigned by the API server and are not part of
		// the desired Service definition.
		d.Spec.ClusterIP = ""
		d.Spec.ClusterIPs = nil
		d.Spec.IPFamilies = nil
		d.Spec.IPFamilyPolicy = nil
		d.Spec.HealthCheckNodePort = 0
		c.Spec.ClusterIP = ""
		c.Spec.ClusterIPs = nil
		c.Spec.IPFamilies = nil
		c.Spec.IPFamilyPolicy = nil
		c.Spec.HealthCheckNodePort = 0
	case *appsv1.DaemonSet:
		c := c.(*appsv1.DaemonSet)
		d.Status = appsv1.DaemonSetStatus{}
		c.Status = appsv1.DaemonSetStatus{}
	}

	return equality.Semantic.DeepEqual(d, c)
}

func desiredService(fc *flowcapturev1alpha1.FlowCapture, names sharedcapture.Names) *corev1.Service {
	return sharedcapture.BuildCollectorService(names.CollectorService, fc.Namespace, map[string]string{"flowcapture": fc.Name, "component": "collector"})
}

func (r *Reconciler) desiredCollectorNetworkPolicy(_ context.Context, fc *flowcapturev1alpha1.FlowCapture, _ string) (*networkingv1.NetworkPolicy, error) {
	// The agents use hostNetwork. On OVN-Kubernetes, traffic from host-network
	// pods is not represented by their node IP in NetworkPolicy evaluation, so
	// neither a namespace selector nor dynamically discovered node IP blocks
	// allow the agents to reach this pod. Keep the policy port-scoped instead:
	// the collector is only exposed through its internal ClusterIP Service and
	// only its gRPC and Prometheus ports are allowed.
	return &networkingv1.NetworkPolicy{
		TypeMeta:   metav1.TypeMeta{APIVersion: "networking.k8s.io/v1", Kind: "NetworkPolicy"},
		ObjectMeta: metav1.ObjectMeta{Name: fc.Name + "-collector", Namespace: fc.Namespace},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{MatchLabels: map[string]string{"flowcapture": fc.Name, "component": "collector"}},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress},
			Ingress: []networkingv1.NetworkPolicyIngressRule{{Ports: []networkingv1.NetworkPolicyPort{
				{Port: ptr.To(intstr.FromInt32(9999)), Protocol: ptr.To(corev1.ProtocolTCP)},
				{Port: ptr.To(intstr.FromInt32(9401)), Protocol: ptr.To(corev1.ProtocolTCP)},
			}}},
		},
	}, nil
}

func desiredCollectorServiceAccount(names *captureNames) *corev1.ServiceAccount {
	return sharedcapture.BuildServiceAccount(names.CollectorServiceAccount, names.CollectorNamespace)
}

func desiredCollectorRole(names *captureNames) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "Role"},
		ObjectMeta: metav1.ObjectMeta{Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace},
		Rules:      []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"pods"}, Verbs: []string{"get", "patch"}}},
	}
}

func desiredCollectorRoleBinding(names *captureNames) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta:   metav1.TypeMeta{APIVersion: "rbac.authorization.k8s.io/v1", Kind: "RoleBinding"},
		ObjectMeta: metav1.ObjectMeta{Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace},
		RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "Role", Name: names.CollectorServiceAccount},
		Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: names.CollectorServiceAccount, Namespace: names.CollectorNamespace}},
	}
}

func (r *Reconciler) desiredCollector(fc *flowcapturev1alpha1.FlowCapture, image string, names *captureNames) *corev1.Pod {
	return buildDesiredCollector(fc, image, names, r.maxCaptureDuration().String())
}

func desiredCollector(fc *flowcapturev1alpha1.FlowCapture, image string, names *captureNames) *corev1.Pod {
	return buildDesiredCollector(fc, image, names, manager.DefaultMaxCaptureDuration.String())
}

func buildDesiredCollector(fc *flowcapturev1alpha1.FlowCapture, image string, names *captureNames, defaultMaxTime string) *corev1.Pod {
	maxTime := defaultMaxTime
	if fc.Spec.MaxTime != nil {
		maxTime = fc.Spec.MaxTime.Duration.String()
	}
	collectorOptions := "background|managed"
	if fc.Spec.TLSPlaintext != nil {
		// The CLI uses this option to persist plaintext records and correlate them
		// with the packet capture. The agent is configured separately below.
		collectorOptions += "|enable_openssl"
	}
	args := sharedcapture.CollectorCommand(sharedcapture.Type(fc.Spec.OutputType), fc.Namespace, "info", maxTime, fc.Spec.MaxBytes, collectorOptions)
	// Managed collectors must remain Running after the capture limit is reached:
	// their emptyDir output is copied with oc cp before the user cleans them up.
	podOptions := &sharedcapture.CollectorPodOptions{Name: names.CollectorPod, Namespace: fc.Namespace, ServiceAccount: names.CollectorServiceAccount, Image: image, Command: args}
	pod := sharedcapture.BuildCollectorPod(podOptions)
	for key, value := range flowCaptureLabels(fc.Name, "collector", fc.Namespace, fc.Name) {
		pod.Labels[key] = value
	}
	pod.Labels["part-of"] = constants.OperatorName
	return pod
}

func desiredAgent(fc *flowcapturev1alpha1.FlowCapture, image string, names *captureNames, subnets map[string][]string) *appsv1.DaemonSet {
	sampling := int(fc.Spec.Sampling)
	if sampling == 0 {
		sampling = 1
	}
	features := make([]sharedcapture.Feature, len(fc.Spec.Features))
	for i, feature := range fc.Spec.Features {
		features[i] = sharedcapture.Feature(feature)
	}
	collectorHost := names.CollectorService + "." + fc.Namespace + ".svc"
	pipeline := sharedcapture.CollectorPipelineConfig(collectorHost)
	if sharedcapture.Type(fc.Spec.OutputType) == sharedcapture.Metrics {
		pipeline = sharedcapture.MetricsPipelineConfig()
	}
	sharedcapture.AddSubnetLabels(pipeline, subnets)
	sharedcapture.AddFilterQueries(pipeline, fc.Spec.Queries)
	var plaintext *sharedcapture.TLSPlaintextOptions
	if fc.Spec.TLSPlaintext != nil {
		plaintext = &sharedcapture.TLSPlaintextOptions{MinBytes: fc.Spec.TLSPlaintext.MinBytes, PreviewBytes: fc.Spec.TLSPlaintext.PreviewBytes, ProcessAllowlist: fc.Spec.TLSPlaintext.ProcessAllowlist}
	}
	options := &sharedcapture.AgentOptions{Name: names.AgentDaemonSet, Namespace: names.AgentNamespace, ServiceAccount: names.AgentDaemonSet, Image: image, TargetHost: collectorHost, NodeSelector: fc.Spec.NodeSelector, Sampling: sampling, Interfaces: fc.Spec.Interfaces, ExcludeInterfaces: fc.Spec.ExcludeInterfaces, Features: features, Filters: fc.Spec.Filters, Type: sharedcapture.Type(fc.Spec.OutputType), MetricsPort: names.AgentMetricsPort, Labels: flowCaptureLabels(names.AgentDaemonSet, "agent", fc.Namespace, fc.Name), PipelineConfig: pipeline, TLSPlaintext: plaintext}
	return sharedcapture.BuildAgentDaemonSet(options)
}

func flowCaptureLabels(resourceName, component, namespace, captureName string) map[string]string {
	return map[string]string{
		"flowcapture":             resourceName,
		"component":               component,
		flowCaptureNameLabel:      captureName,
		flowCaptureNamespaceLabel: namespace,
	}
}
