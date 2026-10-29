package flowcapture

import (
	"strings"
	"testing"

	sharedcapture "github.com/netobserv/netobserv-operator/pkg/capture"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
