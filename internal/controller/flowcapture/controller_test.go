package flowcapture

import (
	"strings"
	"testing"
	"time"

	flowcapturev1alpha1 "github.com/netobserv/netobserv-operator/api/flowcapture/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	sharedcapture "github.com/netobserv/netobserv-operator/pkg/capture"
	securityv1 "github.com/openshift/api/security/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestStatusForPod(t *testing.T) {
	tests := []struct {
		name           string
		podPhase       corev1.PodPhase
		expectedPhase  string
		expectedReason string
	}{
		{name: "missing", expectedPhase: phasePending, expectedReason: "CollectorPending"},
		{name: "pending", podPhase: corev1.PodPending, expectedPhase: phasePending, expectedReason: "CollectorPending"},
		{name: "running", podPhase: corev1.PodRunning, expectedPhase: phaseRunning, expectedReason: "CaptureRunning"},
		{name: "completed", podPhase: corev1.PodSucceeded, expectedPhase: phaseCompleted, expectedReason: "CaptureCompleted"},
		{name: "failed", podPhase: corev1.PodFailed, expectedPhase: phaseFailed, expectedReason: "CollectorFailed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pod *corev1.Pod
			if tt.name != "missing" {
				pod = &corev1.Pod{Status: corev1.PodStatus{Phase: tt.podPhase}}
			}
			phase, reason, _ := statusForPod(pod)
			if phase != tt.expectedPhase || reason != tt.expectedReason {
				t.Fatalf("statusForPod() = (%q, %q), want (%q, %q)", phase, reason, tt.expectedPhase, tt.expectedReason)
			}
		})
	}

	t.Run("collector completed while retained for output copy", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{sharedcapture.CaptureCompletedAnnotation: "true"}},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		phase, reason, _ := statusForPod(pod)
		if phase != phaseCompleted || reason != "CaptureCompleted" {
			t.Fatalf("statusForPod() = (%q, %q), want completed", phase, reason)
		}
	})

	t.Run("copy command", func(t *testing.T) {
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "sample-collector", Namespace: "capture"}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}}
		_, _, message := statusForPod(pod)
		want := "oc cp -n capture sample-collector:output ./output"
		if !strings.Contains(message, want) {
			t.Fatalf("statusForPod() message = %q, want it to contain %q", message, want)
		}
	})
}

func TestAppendPhaseConditionKeepsHistory(t *testing.T) {
	var conditions []metav1.Condition
	if !appendPhaseCondition(&conditions, phasePending, "CollectorPending", "waiting") {
		t.Fatal("expected starting condition to be added")
	}
	if !appendPhaseCondition(&conditions, phaseRunning, "CaptureRunning", "running") {
		t.Fatal("expected running condition to be added")
	}
	if !appendPhaseCondition(&conditions, phaseCompleted, "CaptureCompleted", "completed") {
		t.Fatal("expected completed condition to be added")
	}
	if appendPhaseCondition(&conditions, phaseCompleted, "CaptureCompleted", "completed") {
		t.Fatal("did not expect duplicate completed condition")
	}
	if len(conditions) != 3 {
		t.Fatalf("got %d conditions, want 3: %#v", len(conditions), conditions)
	}
	for i, want := range []string{conditionStarting, conditionRunning, conditionCompleted} {
		if conditions[i].Type != want || conditions[i].Status != metav1.ConditionTrue {
			t.Fatalf("condition %d = %#v, want %s=True", i, conditions[i], want)
		}
	}
}

func TestCapturePrecedes(t *testing.T) {
	created := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	later := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC))
	first := &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "first", CreationTimestamp: created}}
	second := &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "second", CreationTimestamp: later}}
	if !capturePrecedes(first, second) || capturePrecedes(second, first) {
		t.Fatal("expected the earlier capture to precede the later capture")
	}
	second.CreationTimestamp = created
	if !capturePrecedes(first, second) {
		t.Fatal("expected names to provide a deterministic tie-breaker")
	}
}

func TestCaptureNamesAreUniqueAcrossNamespaces(t *testing.T) {
	r := &Reconciler{config: &manager.Config{Namespace: "netobserv"}}
	first := r.captureNames(&flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-a"}})
	second := r.captureNames(&flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-b"}})
	if first.AgentDaemonSet == second.AgentDaemonSet {
		t.Fatalf("agent names collided: %q", first.AgentDaemonSet)
	}
	if first.AgentNamespace != "netobserv-privileged" || second.AgentNamespace != first.AgentNamespace {
		t.Fatalf("unexpected agent namespaces: %q, %q", first.AgentNamespace, second.AgentNamespace)
	}
	if first.AgentMetricsPort == 9400 || first.AgentMetricsPort == second.AgentMetricsPort {
		t.Fatalf("expected capture-specific health ports, got %d and %d", first.AgentMetricsPort, second.AgentMetricsPort)
	}

	r.config.DefaultOperandsNamespace = "observability"
	if got := r.captureNames(&flowcapturev1alpha1.FlowCapture{}).AgentNamespace; got != "observability-privileged" {
		t.Fatalf("configured agent namespace = %q, want observability-privileged", got)
	}
}

func TestDesiredAgentUsesPrivilegedNamespaceAndNamespacedCollectorService(t *testing.T) {
	r := &Reconciler{config: &manager.Config{Namespace: "netobserv"}}
	fc := &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-a"}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Flows}}
	names := r.captureNames(fc)
	agent := desiredAgent(fc, "agent:latest", &names)
	if agent.Namespace != "netobserv-privileged" {
		t.Fatalf("agent namespace = %q, want netobserv-privileged", agent.Namespace)
	}
	if agent.Spec.Template.Spec.ServiceAccountName != names.AgentDaemonSet {
		t.Fatalf("agent service account = %q, want %q", agent.Spec.Template.Spec.ServiceAccountName, names.AgentDaemonSet)
	}
	if got := agent.Labels[flowCaptureNameLabel]; got != fc.Name {
		t.Fatalf("agent FlowCapture name label = %q, want %q", got, fc.Name)
	}
	if got := agent.Labels[flowCaptureNamespaceLabel]; got != fc.Namespace {
		t.Fatalf("agent FlowCapture namespace label = %q, want %q", got, fc.Namespace)
	}
	if got := agent.Spec.Template.Labels[flowCaptureNameLabel]; got != fc.Name {
		t.Fatalf("agent pod FlowCapture name label = %q, want %q", got, fc.Name)
	}
	if got := agent.Spec.Template.Spec.Containers[0].Env; len(got) == 0 {
		t.Fatal("expected agent environment to be configured")
	}
	for _, env := range agent.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "TARGET_HOST" && env.Value != "capture-collector.team-a.svc" {
			t.Fatalf("collector host = %q, want capture-collector.team-a.svc", env.Value)
		}
	}
}

func TestDesiredCollectorRemainsCopyableAfterCompletion(t *testing.T) {
	r := &Reconciler{config: &manager.Config{Namespace: "netobserv"}}
	fc := &flowcapturev1alpha1.FlowCapture{
		ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-a"},
		Spec: flowcapturev1alpha1.FlowCaptureSpec{
			OutputType: flowcapturev1alpha1.Flows,
		},
	}
	maxTime := metav1.Duration{Duration: time.Minute}
	fc.Spec.MaxTime = &maxTime
	names := r.captureNames(fc)
	pod := desiredCollector(fc, "cli:latest", &names)
	if pod.Spec.ActiveDeadlineSeconds != nil {
		t.Fatalf("managed collector must not have an active deadline: %d", *pod.Spec.ActiveDeadlineSeconds)
	}
}

func TestDesiredCollectorEnablesTLSPlaintextOutput(t *testing.T) {
	r := &Reconciler{config: &manager.Config{Namespace: "netobserv"}}
	fc := &flowcapturev1alpha1.FlowCapture{
		ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-a"},
		Spec: flowcapturev1alpha1.FlowCaptureSpec{
			OutputType:   flowcapturev1alpha1.Packets,
			TLSPlaintext: &flowcapturev1alpha1.TLSPlaintextSpec{PreviewBytes: 256},
		},
	}
	names := r.captureNames(fc)
	pod := desiredCollector(fc, "cli:latest", &names)

	for i, arg := range pod.Spec.Containers[0].Command {
		if arg == "--options" {
			if i+1 >= len(pod.Spec.Containers[0].Command) || pod.Spec.Containers[0].Command[i+1] != "background|managed|enable_openssl" {
				t.Fatalf("collector options = %v, want enable_openssl", pod.Spec.Containers[0].Command)
			}
			return
		}
	}
	t.Fatalf("collector command has no --options: %v", pod.Spec.Containers[0].Command)
}

func TestDesiredFlowCaptureSCC(t *testing.T) {
	scc := desiredFlowCaptureSCC()
	if scc.Name != "flowcapture" || !scc.AllowHostNetwork || !scc.AllowHostPorts {
		t.Fatalf("unexpected SCC identity or host access: %#v", scc)
	}
	if scc.AllowPrivilegedContainer || scc.AllowPrivilegeEscalation == nil || *scc.AllowPrivilegeEscalation {
		t.Fatal("FlowCapture SCC must not allow privileged containers or privilege escalation")
	}
	if scc.FSGroup.Type != securityv1.FSGroupStrategyRunAsAny || scc.RunAsUser.Type != securityv1.RunAsUserStrategyRunAsAny {
		t.Fatalf("unexpected SCC strategies: fsGroup=%q runAsUser=%q", scc.FSGroup.Type, scc.RunAsUser.Type)
	}
}

func TestDesiredFlowCapturePlaintextSCC(t *testing.T) {
	scc := desiredFlowCapturePlaintextSCC()
	if scc.Name != "flowcapture-plaintext" || !scc.AllowHostPID || !scc.AllowPrivilegedContainer || scc.AllowPrivilegeEscalation == nil || !*scc.AllowPrivilegeEscalation {
		t.Fatalf("unexpected plaintext SCC: %#v", scc)
	}
	if len(scc.AllowedCapabilities) != 4 || scc.AllowedCapabilities[3] != "SYS_PTRACE" {
		t.Fatalf("plaintext SCC must allow SYS_PTRACE: %#v", scc.AllowedCapabilities)
	}
}

func TestIsManagedCaptureResource(t *testing.T) {
	r := &Reconciler{config: &manager.Config{Namespace: "netobserv"}}
	fc := &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team-a"}}
	names := r.captureNames(fc)
	tests := []struct {
		name string
		obj  client.Object
		want bool
	}{
		{"agent daemonset", &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"flowcapture": names.AgentDaemonSet, "component": "agent"}}}, true},
		{"foreign daemonset", &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"flowcapture": "other", "component": "agent"}}}, false},
		{"agent service account", &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"flowcapture": names.AgentDaemonSet, "component": "agent"}}}, true},
		{"unlabelled service account", &corev1.ServiceAccount{}, false},
		{"health service", &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: names.AgentDaemonSet + "-health", Labels: map[string]string{"app": names.AgentDaemonSet + "-health"}}}, true},
		{"health monitor", &monitoringv1.ServiceMonitor{Spec: monitoringv1.ServiceMonitorSpec{Selector: metav1.LabelSelector{MatchLabels: map[string]string{"app": names.AgentDaemonSet + "-health"}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isManagedCaptureResource(tt.obj, &names); got != tt.want {
				t.Fatalf("isManagedCaptureResource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestActiveMetricsCapture(t *testing.T) {
	current := &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "current", Namespace: "one", CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC))}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Metrics}}
	tests := []struct {
		name      string
		candidate *flowcapturev1alpha1.FlowCapture
		blocked   bool
	}{
		{name: "active metrics in another namespace", candidate: &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "two", CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Metrics}}, blocked: true},
		{name: "newer metrics capture", candidate: &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "two", CreationTimestamp: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC))}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Metrics}}, blocked: false},
		{name: "same capture", candidate: current, blocked: false},
		{name: "completed metrics", candidate: &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "two"}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Metrics}, Status: flowcapturev1alpha1.FlowCaptureStatus{Phase: phaseCompleted}}, blocked: false},
		{name: "flow capture", candidate: &flowcapturev1alpha1.FlowCapture{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "two"}, Spec: flowcapturev1alpha1.FlowCaptureSpec{OutputType: flowcapturev1alpha1.Flows}}, blocked: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := activeMetricsCapture(tt.candidate, current) != nil
			if got != tt.blocked {
				t.Fatalf("activeMetricsCapture() = %t, want %t", got, tt.blocked)
			}
		})
	}
}

func TestResourcesEqualIgnoresServerManagedFields(t *testing.T) {
	desired := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "capture-agent", Namespace: "capture"}}
	current := desired.DeepCopy()
	desired.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"})
	current.SetGroupVersionKind(schema.GroupVersionKind{})
	current.Annotations = map[string]string{"openshift.io/internal-registry-pull-secret-ref": "capture-agent-dockercfg"}
	current.ImagePullSecrets = []corev1.LocalObjectReference{{Name: "capture-agent-dockercfg"}}
	current.Secrets = []corev1.ObjectReference{{Name: "capture-agent-token"}}
	current.ResourceVersion = "42"
	current.UID = "uid"
	current.CreationTimestamp = metav1.Now()

	if !resourcesEqual(desired, current) {
		t.Fatal("expected API-managed ServiceAccount fields to be ignored")
	}

	current.Labels = map[string]string{"changed": "true"}
	if resourcesEqual(desired, current) {
		t.Fatal("expected controller-managed changes to be detected")
	}
}

func TestResourcesEqualIgnoresDaemonSetStatus(t *testing.T) {
	desired := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "capture-agent", Namespace: "capture"}}
	current := desired.DeepCopy()
	current.Status.NumberReady = 2

	if !resourcesEqual(desired, current) {
		t.Fatal("expected DaemonSet status to be ignored")
	}
}

func TestSetClusterMonitoringLabelIsIdempotent(t *testing.T) {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"existing": "label"}}}
	if !setClusterMonitoringLabel(ns) {
		t.Fatal("expected the label to be added")
	}
	if setClusterMonitoringLabel(ns) {
		t.Fatal("did not expect an update when the label is already present")
	}
	if ns.Labels[clusterMonitoringLabel] != "true" || ns.Labels["existing"] != "label" {
		t.Fatalf("unexpected namespace labels: %#v", ns.Labels)
	}
}
