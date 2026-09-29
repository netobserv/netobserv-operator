package controllers

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	odcapi "github.com/netobserv/netobserv-operator/api/ondemandcapture/v1beta1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	ondemandCaptureFinalizer = "ondemandcapture.netobserv.io/finalizer"
	captureUIDLabel          = "netobserv.io/capture-uid"
	captureCLIRole           = "netobserv-ondemandcapture-cli"
	fileServerPort           = int32(8080)
)

type OnDemandCaptureReconciler struct {
	client.Client
	ebpfAgentImage    string
	netobservCLIImage string
	operatorNamespace string
}

// StartOnDemandCapture registers the capture controller with the configured operator namespace and images.
func StartOnDemandCapture(_ context.Context, mgr *manager.Manager) (manager.PostCreateHook, error) {
	r := &OnDemandCaptureReconciler{Client: mgr.Client, operatorNamespace: mgr.Config.Namespace, ebpfAgentImage: mgr.Config.NetobservCLIAgentImage, netobservCLIImage: mgr.Config.NetobservCLIImage}
	return nil, r.SetupWithManager(mgr.Manager)
}

// captureName derives stable launcher and access-resource names from the immutable CR UID.
func captureName(odc *odcapi.OnDemandCapture) string { return "ondemandcapture-" + string(odc.UID) }

// captureNamespace keeps the complete UID suffix while fitting a DNS label within 63 characters.
func captureNamespace(odc *odcapi.OnDemandCapture) string {
	// Kubernetes assigns each CR a unique, random UID. Reuse it instead of
	// generating a new suffix on every reconciliation or operator restart.
	suffix := strings.ReplaceAll(string(odc.UID), "-", "")
	name := strings.ReplaceAll(odc.Name, ".", "-")
	const prefix = "netobserv-"
	maxName := 63 - len(prefix) - 1 - len(suffix)
	if len(name) > maxName {
		name = name[:maxName]
	}
	name = strings.TrimRight(name, "-")
	return prefix + name + "-" + suffix
}

// Reconcile advances captures through validation, execution, retention, and finalization.
func (r *OnDemandCaptureReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	odc := &odcapi.OnDemandCapture{}
	if err := r.Get(ctx, req.NamespacedName, odc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !odc.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, odc)
	}
	if err := r.validateCaptureNamespace(odc); err != nil {
		return r.rejectCaptureNamespace(ctx, odc, err.Error())
	}
	if controllerutil.AddFinalizer(odc, ondemandCaptureFinalizer) {
		if err := r.Update(ctx, odc); err != nil {
			return ctrl.Result{}, err
		}
	}
	switch odc.Status.Phase {
	case "":
		if err := r.validateSpec(odc); err != nil {
			return r.failCapture(ctx, odc, err.Error())
		}
		odc.Status.Phase = odcapi.OnDemandCapturePending
		return ctrl.Result{RequeueAfter: time.Second}, r.Status().Update(ctx, odc)
	case odcapi.OnDemandCapturePending:
		if err := r.ensureCaptureAccess(ctx, odc); err != nil {
			return ctrl.Result{}, err
		}
		pod, err := r.createCLIPod(ctx, odc)
		if err != nil {
			return ctrl.Result{}, err
		}
		odc.Status.PodName = pod.Name
		odc.Status.PodNamespace = pod.Namespace
		odc.Status.FileServerPort = fileServerPort
		odc.Status.DaemonSetName = "netobserv-cli"
		odc.Status.Phase = odcapi.OnDemandCaptureRunning
		return ctrl.Result{RequeueAfter: time.Second}, r.Status().Update(ctx, odc)
	case odcapi.OnDemandCaptureRunning:
		return r.handleRunning(ctx, odc)
	case odcapi.OnDemandCaptureCompleted, odcapi.OnDemandCaptureFailed:
		return r.handleFinished(ctx, odc)
	default:
		return ctrl.Result{}, fmt.Errorf("unknown capture phase %q", odc.Status.Phase)
	}
}

// validateCaptureNamespace limits privileged capture execution to the administrator-owned operator namespace.
func (r *OnDemandCaptureReconciler) validateCaptureNamespace(odc *odcapi.OnDemandCapture) error {
	if r.operatorNamespace == "" {
		return fmt.Errorf("operator namespace is not configured; capture execution is disabled")
	}
	if odc.Namespace != r.operatorNamespace {
		return fmt.Errorf("OnDemandCapture is administrator-only; create it in operator namespace %q", r.operatorNamespace)
	}
	return nil
}

// rejectCaptureNamespace revokes legacy access before cleanup and retains the rejection for the configured TTL.
func (r *OnDemandCaptureReconciler) rejectCaptureNamespace(ctx context.Context, odc *odcapi.OnDemandCapture, message string) (ctrl.Result, error) {
	if err := r.revokeCaptureBinding(ctx, odc); err != nil {
		return ctrl.Result{}, err
	}
	if controllerutil.ContainsFinalizer(odc, ondemandCaptureFinalizer) {
		result, err := r.handleDeletion(ctx, odc)
		if err != nil || controllerutil.ContainsFinalizer(odc, ondemandCaptureFinalizer) {
			return result, err
		}
	}
	if odc.Status.Phase == odcapi.OnDemandCaptureFailed && odc.Status.Error == message {
		return r.handleFinished(ctx, odc)
	}
	return r.failCapture(ctx, odc, message)
}

// Use the persisted completion time so retention survives operator restarts.
func (r *OnDemandCaptureReconciler) handleFinished(ctx context.Context, odc *odcapi.OnDemandCapture) (ctrl.Result, error) {
	if odc.Status.CompletionTime == nil || odc.Status.CompletionTime.IsZero() {
		// Older terminal resources may lack a completion timestamp. Give them a
		// full retention window instead of deleting their files immediately.
		now := metav1.Now()
		odc.Status.CompletionTime = &now
		return ctrl.Result{RequeueAfter: time.Second}, r.Status().Update(ctx, odc)
	}
	ttl := int32(86400)
	if odc.Spec.TTLSecondsAfterFinished != nil {
		ttl = *odc.Spec.TTLSecondsAfterFinished
	}
	if ttl < 0 {
		return ctrl.Result{}, fmt.Errorf("ttlSecondsAfterFinished must not be negative")
	}
	remaining := time.Until(odc.Status.CompletionTime.Add(time.Duration(ttl) * time.Second))
	if remaining > 0 {
		return ctrl.Result{RequeueAfter: remaining}, nil
	}
	// A concurrent retention edit must win over a stale expiry decision.
	err := r.Delete(ctx, odc, client.Preconditions{UID: &odc.UID, ResourceVersion: &odc.ResourceVersion})
	return ctrl.Result{RequeueAfter: time.Second}, client.IgnoreNotFound(err)
}

// failCapture persists an error and completion time so failed captures also expire.
func (r *OnDemandCaptureReconciler) failCapture(ctx context.Context, odc *odcapi.OnDemandCapture, message string) (ctrl.Result, error) {
	odc.Status.Phase = odcapi.OnDemandCaptureFailed
	odc.Status.Error = message
	now := metav1.Now()
	odc.Status.CompletionTime = &now
	return ctrl.Result{RequeueAfter: time.Second}, r.Status().Update(ctx, odc)
}

// handleRunning tracks the CLI container independently of the long-lived download server.
func (r *OnDemandCaptureReconciler) handleRunning(ctx context.Context, odc *odcapi.OnDemandCapture) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: odc.Namespace, Name: odc.Status.PodName}, pod); err != nil {
		if apierrors.IsNotFound(err) {
			return r.failCapture(ctx, odc, "capture pod not found")
		}
		return ctrl.Result{}, err
	}
	if !metav1.IsControlledBy(pod, odc) {
		return r.failCapture(ctx, odc, "capture pod belongs to another resource")
	}
	odc.Status.PodIP = pod.Status.PodIP
	if pod.Status.Phase == corev1.PodFailed {
		return r.failCapture(ctx, odc, pod.Status.Reason+": "+pod.Status.Message)
	}
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.Name != "cli" {
			continue
		}
		if status.State.Running != nil {
			t := status.State.Running.StartedAt
			odc.Status.StartTime = &t
		}
		if ended := status.State.Terminated; ended != nil {
			if ended.ExitCode != 0 {
				return r.failCapture(ctx, odc, fmt.Sprintf("capture exited with code %d: %s %s", ended.ExitCode, ended.Reason, ended.Message))
			}
			odc.Status.Phase = odcapi.OnDemandCaptureCompleted
			odc.Status.StartTime = &ended.StartedAt
			odc.Status.CompletionTime = &ended.FinishedAt
			return ctrl.Result{RequeueAfter: time.Second}, r.Status().Update(ctx, odc)
		}
	}
	// The CLI enforces capture limits. Pod startup time is not capture duration,
	// and the output server remains Running after the CLI container completes.
	return ctrl.Result{RequeueAfter: 5 * time.Second}, r.Status().Update(ctx, odc)
}

// handleDeletion stops the launcher before removing its temporary namespace and access binding.
func (r *OnDemandCaptureReconciler) handleDeletion(ctx context.Context, odc *odcapi.OnDemandCapture) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(odc, ondemandCaptureFinalizer) {
		return ctrl.Result{}, nil
	}
	// Stop the launcher first so it cannot create new resources after cleanup.
	pod := &corev1.Pod{}
	err := r.Get(ctx, types.NamespacedName{Namespace: odc.Namespace, Name: captureName(odc)}, pod)
	if err == nil {
		if !metav1.IsControlledBy(pod, odc) {
			return ctrl.Result{}, fmt.Errorf("capture pod belongs to another resource")
		}
		if err = r.Delete(ctx, pod, client.Preconditions{UID: &pod.UID}); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	// Namespaces and cluster bindings cannot be owned by a namespaced CR.
	// Their names include the immutable CR UID; finalization removes them explicitly.
	ns := &corev1.Namespace{}
	err = r.Get(ctx, types.NamespacedName{Name: captureNamespace(odc)}, ns)
	if err == nil {
		if ns.Labels["app"] != "netobserv-cli" {
			return ctrl.Result{}, fmt.Errorf("capture namespace lacks the netobserv-cli label")
		}
		if err = r.Delete(ctx, ns, client.Preconditions{UID: &ns.UID}); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	if err := r.revokeCaptureBinding(ctx, odc); err != nil {
		return ctrl.Result{}, err
	}
	controllerutil.RemoveFinalizer(odc, ondemandCaptureFinalizer)
	return ctrl.Result{}, r.Update(ctx, odc)
}

// revokeCaptureBinding removes only access associated with this immutable capture UID.
func (r *OnDemandCaptureReconciler) revokeCaptureBinding(ctx context.Context, odc *odcapi.OnDemandCapture) error {
	binding := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, types.NamespacedName{Name: captureName(odc)}, binding)
	if err == nil {
		if binding.Labels[captureUIDLabel] != string(odc.UID) {
			return fmt.Errorf("capture binding belongs to another resource")
		}
		if err = r.Delete(ctx, binding, client.Preconditions{UID: &binding.UID}); client.IgnoreNotFound(err) != nil {
			return err
		}
	} else if !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// ensureCaptureAccess creates capture-specific credentials only in the trusted operator namespace.
func (r *OnDemandCaptureReconciler) ensureCaptureAccess(ctx context.Context, odc *odcapi.OnDemandCapture) error {
	if err := r.validateCaptureNamespace(odc); err != nil {
		return err
	}
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: captureName(odc), Namespace: odc.Namespace}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, sa, func() error {
		if len(sa.OwnerReferences) > 0 && !metav1.IsControlledBy(sa, odc) {
			return fmt.Errorf("capture service account belongs to another resource")
		}
		sa.AutomountServiceAccountToken = ptr.To(false)
		return controllerutil.SetControllerReference(odc, sa, r.Scheme())
	})
	if err != nil {
		return err
	}
	binding := &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: captureName(odc)}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, binding, func() error {
		if binding.ResourceVersion != "" && binding.Labels[captureUIDLabel] != string(odc.UID) {
			return fmt.Errorf("capture binding belongs to another resource")
		}
		binding.Labels = map[string]string{captureUIDLabel: string(odc.UID)}
		binding.RoleRef = rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: captureCLIRole}
		binding.Subjects = []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa.Name, Namespace: sa.Namespace}}
		return nil
	})
	return err
}

// createCLIPod creates an owned launcher with CLI-only credentials and a loopback file server.
func (r *OnDemandCaptureReconciler) createCLIPod(ctx context.Context, odc *odcapi.OnDemandCapture) (*corev1.Pod, error) {
	if err := r.validateCaptureNamespace(odc); err != nil {
		return nil, err
	}
	resources := corev1.ResourceRequirements{}
	if odc.Spec.Resources != nil {
		resources = *odc.Spec.Resources.DeepCopy()
	}
	security := &corev1.SecurityContext{AllowPrivilegeEscalation: ptr.To(false), Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: captureName(odc), Namespace: odc.Namespace, Labels: map[string]string{"app": "netobserv-ondemandcapture-collector", "part-of": constants.OperatorName, captureUIDLabel: string(odc.UID)}, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(odc, odcapi.GroupVersion.WithKind("OnDemandCapture"))}}, Spec: corev1.PodSpec{
		ServiceAccountName: captureName(odc), AutomountServiceAccountToken: ptr.To(false), RestartPolicy: corev1.RestartPolicyNever, TerminationGracePeriodSeconds: ptr.To(int64(150)),
		SecurityContext: &corev1.PodSecurityContext{RunAsNonRoot: ptr.To(true), SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}},
		Containers: []corev1.Container{
			{Name: "cli", Image: r.netobservCLIImage, ImagePullPolicy: corev1.PullAlways, Command: []string{"/oc-netobserv"}, Args: r.buildCLIArgs(odc),
				Env:          []corev1.EnvVar{{Name: "NETOBSERV_COLLECTOR_IMAGE", Value: r.netobservCLIImage}},
				VolumeMounts: []corev1.VolumeMount{{Name: "output", MountPath: "/output"}, {Name: "capture-api-access", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount", ReadOnly: true}}, Resources: resources, SecurityContext: security, TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError},
			{Name: "files", Image: r.netobservCLIImage, ImagePullPolicy: corev1.PullAlways, Command: []string{"/oc-netobserv"}, Args: []string{"serve", "--directory=/output", "--listen=127.0.0.1:8080"},
				VolumeMounts: []corev1.VolumeMount{{Name: "output", MountPath: "/output", ReadOnly: true}}, Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: fileServerPort}}, SecurityContext: security.DeepCopy()},
		}, Volumes: []corev1.Volume{{Name: "output", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
	}}
	pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{Name: "capture-api-access", VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{
		DefaultMode: ptr.To(int32(0444)),
		Sources: []corev1.VolumeProjection{
			{ServiceAccountToken: &corev1.ServiceAccountTokenProjection{Path: "token", ExpirationSeconds: ptr.To(int64(3600))}},
			{ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{Name: "kube-root-ca.crt"}, Items: []corev1.KeyToPath{{Key: "ca.crt", Path: "ca.crt"}}}},
			{DownwardAPI: &corev1.DownwardAPIProjection{Items: []corev1.DownwardAPIVolumeFile{{Path: "namespace", FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "metadata.namespace"}}}}},
		},
	}}})
	if r.ebpfAgentImage != "" {
		pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, corev1.EnvVar{Name: "NETOBSERV_AGENT_IMAGE", Value: r.ebpfAgentImage})
	}
	if err := r.Create(ctx, pod); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		if err = r.Get(ctx, client.ObjectKeyFromObject(pod), pod); err != nil {
			return nil, err
		}
		if !metav1.IsControlledBy(pod, odc) {
			return nil, fmt.Errorf("capture pod belongs to another resource")
		}
	}
	return pod, nil
}

// validateSpec rejects unsupported backend settings before granting capture access.
func (r *OnDemandCaptureReconciler) validateSpec(odc *odcapi.OnDemandCapture) error {
	if r.netobservCLIImage == "" {
		return fmt.Errorf("RELATED_IMAGE_NETOBSERV_CLI must be configured")
	}
	switch odc.Spec.CaptureType {
	case odcapi.CaptureTypeFlows, odcapi.CaptureTypePackets, odcapi.CaptureTypeMetrics:
	default:
		return fmt.Errorf("invalid captureType: %s", odc.Spec.CaptureType)
	}
	if odc.Spec.Duration != nil && odc.Spec.Duration.Duration <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if odc.Spec.MaxBytes != nil && *odc.Spec.MaxBytes <= 0 {
		return fmt.Errorf("maxBytes must be positive")
	}
	if odc.Spec.Filters != nil && (len(odc.Spec.Filters.Namespaces) > 0 || len(odc.Spec.Filters.PodLabelSelector) > 0) {
		return fmt.Errorf("namespace and pod-label filters are not supported by the CLI capture backend")
	}
	hasArgsFilter, err := validateCaptureArgs(odc.Spec.Args)
	if err != nil {
		return err
	}
	if err := validateFilterGroups(odc.Spec.Filters); err != nil {
		return err
	}
	if odc.Spec.CaptureType == odcapi.CaptureTypePackets {
		if len(odc.Spec.Interfaces) > 0 {
			return fmt.Errorf("interfaces is not supported for packet capture")
		}
		if !hasArgsFilter && len(captureFilterArgs(odc.Spec.Filters)) == 0 && (odc.Spec.PacketConfig == nil || (odc.Spec.PacketConfig.Port == 0 && odc.Spec.PacketConfig.Protocol == "")) {
			return fmt.Errorf("packet capture requires at least one traffic filter")
		}
	}
	if odc.Spec.CaptureType == odcapi.CaptureTypeMetrics && odc.Spec.MaxBytes != nil {
		return fmt.Errorf("maxBytes is not supported for metrics capture")
	}
	return nil
}

// buildCLIArgs maps structured settings to argv and appends validated native arguments.
func (r *OnDemandCaptureReconciler) buildCLIArgs(odc *odcapi.OnDemandCapture) []string {
	args := []string{string(odc.Spec.CaptureType), "--headless", "--copy=true", "--namespace=" + captureNamespace(odc), "--output-dir=/output"}
	if odc.Spec.Duration != nil {
		args = append(args, "--max-time="+odc.Spec.Duration.Duration.String())
	}
	if odc.Spec.MaxBytes != nil {
		args = append(args, fmt.Sprintf("--max-bytes=%d", *odc.Spec.MaxBytes))
	}
	if len(odc.Spec.Interfaces) > 0 {
		args = append(args, "--interfaces="+strings.Join(odc.Spec.Interfaces, ","))
	}
	keys := make([]string, 0, len(odc.Spec.NodeSelector))
	for key := range odc.Spec.NodeSelector {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "--node-selector="+key+":"+odc.Spec.NodeSelector[key])
	}
	var dns, rtt, drops bool
	if c := odc.Spec.FlowConfig; c != nil && odc.Spec.CaptureType == odcapi.CaptureTypeFlows {
		dns, rtt, drops = c.EnableDNS, c.EnableRTT, c.EnablePacketDrops
		if c.Sampling > 0 {
			args = append(args, fmt.Sprintf("--sampling=%d", c.Sampling))
		}
	}
	if c := odc.Spec.MetricsConfig; c != nil && odc.Spec.CaptureType == odcapi.CaptureTypeMetrics {
		dns, rtt, drops = c.EnableDNS, c.EnableRTT, c.EnablePacketDrops
	}
	if dns {
		args = append(args, "--enable_dns")
	}
	if rtt {
		args = append(args, "--enable_rtt")
	}
	if drops {
		args = append(args, "--enable_pkt_drop")
	}
	if c := odc.Spec.PacketConfig; c != nil && odc.Spec.CaptureType == odcapi.CaptureTypePackets {
		if c.Protocol != "" {
			args = append(args, "--protocol="+strings.ToUpper(c.Protocol))
		}
		if c.Port > 0 {
			args = append(args, fmt.Sprintf("--dport=%d", c.Port))
		}
	}
	args = append(args, captureFilterArgs(odc.Spec.Filters)...)
	return append(args, odc.Spec.Args...)
}

// SetupWithManager watches capture resources and owned pod lifecycle changes.
func (r *OnDemandCaptureReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&odcapi.OnDemandCapture{}, reconcilers.IgnoreStatusChange).Owns(&corev1.Pod{}, reconcilers.UpdateOrDeleteOnlyPred).Complete(r)
}
