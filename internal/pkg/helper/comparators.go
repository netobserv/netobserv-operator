package helper

import (
	"fmt"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
)

type ReconcileAction int

const (
	ActionNone = iota
	ActionCreate
	ActionUpdate
)

var (
	deepEqual      = equality.Semantic.DeepEqual
	deepDerivative = equality.Semantic.DeepDerivative
)

func DaemonSetChanged(current, desired *appsv1.DaemonSet) ReconcileAction {
	if desired == nil {
		return ActionNone
	}
	if current == nil {
		return ActionCreate
	}
	cSpec, dSpec := current.Spec, desired.Spec
	if !IsSubSet(current.ObjectMeta.Labels, desired.ObjectMeta.Labels) ||
		!deepDerivative(dSpec.Selector, cSpec.Selector) ||
		!deepDerivative(dSpec.Template, cSpec.Template) ||
		assignationChanged(&cSpec.Template, &dSpec.Template, nil) {
		return ActionUpdate
	}

	// Env vars aren't covered by DeepDerivative when they are removed: deep-compare them
	dConts := dSpec.Template.Spec.Containers
	cConts := cSpec.Template.Spec.Containers
	if len(dConts) > 0 && len(cConts) > 0 && !deepEqual(dConts[0].Env, cConts[0].Env) {
		return ActionUpdate
	}

	return ActionNone
}

func DeploymentChanged(old, n *appsv1.Deployment, contName string, report *ChangeReport) bool {
	return report.Check("Pod changed", PodChanged(&old.Spec.Template, &n.Spec.Template, contName, report)) ||
		report.Check("Replicas changed", *old.Spec.Replicas != *n.Spec.Replicas)
}

func PodChanged(old, n *corev1.PodTemplateSpec, containerName string, report *ChangeReport) bool {
	if annotationsChanged(old, n, report) || volumesChanged(old, n, report) || assignationChanged(old, n, report) {
		return true
	}
	// Find containers
	oldContainer := FindContainer(&old.Spec, containerName)
	if oldContainer == nil {
		report.Add("Old container not found")
		return true
	}
	newContainer := FindContainer(&n.Spec, containerName)
	if newContainer == nil {
		report.Add("New container not found")
		return true
	}
	return report.Check("Container changed", containerChanged(oldContainer, newContainer, report))
}

func annotationsChanged(old, n *corev1.PodTemplateSpec, report *ChangeReport) bool {
	if old.Annotations == nil && n.Annotations == nil {
		return false
	}
	if old.Annotations == nil {
		report.Add("New annotations, previously none")
		return true
	}
	// Check domain annotations (config digest, certificate stamp...)
	for k, v := range old.Annotations {
		if strings.HasPrefix(k, constants.AnnotationDomain) {
			if n.Annotations[k] != v {
				report.Add(fmt.Sprintf("Annotation changed: '%s: %s'", k, v))
				return true
			}
		}
	}
	return false
}

func assignationChanged(old, n *corev1.PodTemplateSpec, report *ChangeReport) bool {
	if !deepEqual(n.Spec.NodeSelector, old.Spec.NodeSelector) {
		if report != nil {
			report.Add("NodeSelector changed")
		}
		return true
	}
	if !deepEqual(n.Spec.Tolerations, old.Spec.Tolerations) {
		if report != nil {
			report.Add("Toleration changed")
		}
		return true
	}
	if !deepDerivative(n.Spec.Affinity, old.Spec.Affinity) {
		if report != nil {
			report.Add("Affinity changed")
		}
		return true
	}
	if n.Spec.PriorityClassName != old.Spec.PriorityClassName {
		if report != nil {
			report.Add("PriorityClassName changed")
		}
		return true
	}
	return false
}

func volumesChanged(old, n *corev1.PodTemplateSpec, report *ChangeReport) bool {
	return report.Check("Volumes changed", !deepDerivative(n.Spec.Volumes, old.Spec.Volumes))
}

func containerChanged(old, n *corev1.Container, report *ChangeReport) bool {
	return report.Check("Image changed", n.Image != old.Image) ||
		report.Check("Pull policy changed", n.ImagePullPolicy != old.ImagePullPolicy) ||
		report.Check("Args changed", !deepEqual(n.Args, old.Args)) ||
		report.Check("Resources req/limit changed", !deepDerivative(n.Resources, old.Resources)) ||
		report.Check("Env changed", !deepEqual(n.Env, old.Env)) ||
		report.Check("Liveness probe changed", probeChanged(n.LivenessProbe, old.LivenessProbe)) ||
		report.Check("Startup probe changed", probeChanged(n.StartupProbe, old.StartupProbe))
}

func probeChanged(old, n *corev1.Probe) bool {
	return (old == nil && n != nil) || (old != nil && n == nil)
}

func ServiceChanged(old, n *corev1.Service, report *ChangeReport) bool {
	return report.Check("Service annotations changed", !deepDerivative(n.Annotations, old.Annotations)) ||
		report.Check("Service labels changed", !deepDerivative(n.Labels, old.Labels)) ||
		report.Check("Service spec changed", !deepDerivative(n.Spec, old.Spec))
}

func ServiceMonitorChanged(old, n *monitoringv1.ServiceMonitor, report *ChangeReport) bool {
	return report.Check("ServiceMonitor spec changed", !deepDerivative(n.Spec, old.Spec)) ||
		report.Check("ServiceMonitor labels changed", !IsSubSet(old.Labels, n.Labels))
}

func PrometheusRuleChanged(old, n *monitoringv1.PrometheusRule, report *ChangeReport) bool {
	// Note: DeepDerivative misses changes in Spec.Groups.Rules (covered by test "Expecting PrometheusRule to exist and be updated")
	return report.Check("PrometheusRule spec changed", !deepEqual(n.Spec, old.Spec)) ||
		report.Check("PrometheusRule labels changed", !IsSubSet(old.Labels, n.Labels))
}

// FindContainer searches in pod containers one that matches the provided name
func FindContainer(podSpec *corev1.PodSpec, name string) *corev1.Container {
	for i := range podSpec.Containers {
		if podSpec.Containers[i].Name == name {
			return &podSpec.Containers[i]
		}
	}
	return nil
}

func AutoScalerChanged(asc *ascv2.HorizontalPodAutoscaler, desired flowslatest.FlowCollectorHPA, report *ChangeReport) bool {
	differentPointerValues := func(a, b *int32) bool {
		return (a == nil && b != nil) || (a != nil && b == nil) || (a != nil && *a != *b)
	}
	if report.Check("Max replicas changed", asc.Spec.MaxReplicas != desired.MaxReplicas) ||
		report.Check("Min replicas changed", differentPointerValues(asc.Spec.MinReplicas, desired.MinReplicas)) {
		return true
	}
	if report.Check("Metrics changed", !deepDerivative(desired.Metrics, asc.Spec.Metrics)) {
		return true
	}
	return false
}

// PersistentVolumeClaimSpecChanged compares only the critical immutable fields of PVC specs:
// AccessModes, Storage size, and VolumeMode. Returns true if they differ.
// Note: PVC specs are immutable, so this function is used to detect mismatches that cannot be updated.
func PersistentVolumeClaimSpecChanged(current, desired *corev1.PersistentVolumeClaim, report *ChangeReport) bool {
	// Compare AccessModes
	if report.Check("AccessModes changed", !deepEqual(desired.Spec.AccessModes, current.Spec.AccessModes)) {
		return true
	}

	// Compare Storage resource requests (required field for valid PVCs)
	desiredStorage := desired.Spec.Resources.Requests[corev1.ResourceStorage]
	currentStorage := current.Spec.Resources.Requests[corev1.ResourceStorage]
	if report.Check("Storage size changed", !desiredStorage.Equal(currentStorage)) {
		return true
	}

	// Compare VolumeMode (nil defaults to PersistentVolumeFilesystem)
	getEffectiveMode := func(mode *corev1.PersistentVolumeMode) corev1.PersistentVolumeMode {
		if mode == nil {
			return corev1.PersistentVolumeFilesystem
		}
		return *mode
	}
	desiredMode := getEffectiveMode(desired.Spec.VolumeMode)
	currentMode := getEffectiveMode(current.Spec.VolumeMode)
	return report.Check("VolumeMode changed", desiredMode != currentMode)
}
