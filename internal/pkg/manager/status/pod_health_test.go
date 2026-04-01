package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestClassifyPodIssue_Healthy(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Empty(t, reason)
	assert.Empty(t, msg)
}

func TestClassifyPodIssue_CrashLoopBackOff(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "CrashLoopBackOff",
						Message: "back-off 5m0s restarting failed container",
					},
				},
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason:  "Error",
						Message: "can't write messages into Kafka",
					},
				},
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "CrashLoopBackOff", reason)
	assert.Equal(t, "can't write messages into Kafka", msg)
}

func TestClassifyPodIssue_OOMKilled(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason: "OOMKilled",
					},
				},
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "OOMKilled", reason)
	assert.Empty(t, msg)
}

func TestClassifyPodIssue_ImagePull(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{
						Reason:  "ImagePullBackOff",
						Message: "Back-off pulling image",
					},
				},
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "ImagePullError", reason)
	assert.Equal(t, "Back-off pulling image", msg)
}

func TestClassifyPodIssue_FrequentRestarts(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{{
				State:        corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
				RestartCount: 15,
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						Reason:  "Error",
						Message: "exit code 1",
					},
				},
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "FrequentRestarts", reason)
	assert.Equal(t, "exit code 1", msg)
}

func TestClassifyPodIssue_PodFailed(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase:   corev1.PodFailed,
			Message: "Pod exceeded memory",
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "PodFailed", reason)
	assert.Equal(t, "Pod exceeded memory", msg)
}

func TestClassifyPodIssue_PendingScheduling(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type:    corev1.PodScheduled,
				Status:  corev1.ConditionFalse,
				Reason:  "Unschedulable",
				Message: "0/3 nodes are available: insufficient memory",
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Equal(t, "PendingScheduling", reason)
	assert.Equal(t, "0/3 nodes are available: insufficient memory", msg)
}

func TestClassifyPodIssue_PendingButScheduled(t *testing.T) {
	pod := &corev1.Pod{
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			}},
		},
	}
	reason, msg := classifyPodIssue(pod)
	assert.Empty(t, reason)
	assert.Empty(t, msg)
}

func TestTruncateMessage(t *testing.T) {
	short := "short message"
	assert.Equal(t, short, truncateMessage(short, 100))

	long := "a very long message that exceeds the limit"
	assert.Equal(t, "a very ...", truncateMessage(long, 7))
}
