package status

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const maxPodNamesInSummary = 5

// PodHealthSummary holds aggregated pod health information for a workload.
type PodHealthSummary struct {
	UnhealthyCount int32
	Issues         string
}

// CheckPodHealth lists pods matching the given label selector in the given namespace,
// inspects container statuses, and returns a summary of unhealthy pods.
// This is intended to be called only when a workload reports not-ready replicas,
// to avoid unnecessary API calls when everything is healthy.
func CheckPodHealth(ctx context.Context, c client.Client, namespace string, matchLabels map[string]string) PodHealthSummary {
	rlog := log.FromContext(ctx)

	podList := corev1.PodList{}
	if err := c.List(ctx, &podList, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(matchLabels),
	}); err != nil {
		rlog.Error(err, "Failed to list pods for health check")
		return PodHealthSummary{}
	}

	type issueGroup struct {
		reason   string
		podNames []string
		sample   string
	}
	groups := make(map[string]*issueGroup)
	var unhealthyCount int32

	for i := range podList.Items {
		pod := &podList.Items[i]
		reason, msg := classifyPodIssue(pod)
		if reason == "" {
			continue
		}
		unhealthyCount++
		g, ok := groups[reason]
		if !ok {
			g = &issueGroup{reason: reason}
			groups[reason] = g
		}
		g.podNames = append(g.podNames, pod.Name)
		if g.sample == "" && msg != "" {
			g.sample = msg
		}
	}

	if unhealthyCount == 0 {
		return PodHealthSummary{}
	}

	var parts []string
	for _, g := range groups {
		count := len(g.podNames)
		names := g.podNames
		if len(names) > maxPodNamesInSummary {
			names = names[:maxPodNamesInSummary]
		}
		part := fmt.Sprintf("%d %s (%s)", count, g.reason, strings.Join(names, ", "))
		if len(g.podNames) > maxPodNamesInSummary {
			part += fmt.Sprintf(" and %d more", count-maxPodNamesInSummary)
		}
		if g.sample != "" {
			part += ": " + truncateMessage(g.sample, 200)
		}
		parts = append(parts, part)
	}

	return PodHealthSummary{
		UnhealthyCount: unhealthyCount,
		Issues:         strings.Join(parts, "; "),
	}
}

// classifyPodIssue returns a reason and message if the pod is unhealthy, or empty strings if healthy.
func classifyPodIssue(pod *corev1.Pod) (string, string) {
	for i := range pod.Status.ContainerStatuses {
		cs := &pod.Status.ContainerStatuses[i]
		if cs.State.Waiting != nil {
			switch cs.State.Waiting.Reason {
			case "CrashLoopBackOff":
				msg := extractTerminationMessage(cs)
				return "CrashLoopBackOff", msg
			case "ImagePullBackOff", "ErrImagePull":
				return "ImagePullError", cs.State.Waiting.Message
			}
		}

		if cs.LastTerminationState.Terminated != nil {
			t := cs.LastTerminationState.Terminated
			if t.Reason == "OOMKilled" {
				return "OOMKilled", t.Message
			}
		}

		if cs.RestartCount > 10 && cs.State.Running != nil {
			msg := extractTerminationMessage(cs)
			return "FrequentRestarts", msg
		}
	}

	if pod.Status.Phase == corev1.PodFailed {
		return "PodFailed", pod.Status.Message
	}

	return "", ""
}

func extractTerminationMessage(cs *corev1.ContainerStatus) string {
	if cs.LastTerminationState.Terminated != nil {
		msg := cs.LastTerminationState.Terminated.Message
		if msg != "" {
			return msg
		}
		return cs.LastTerminationState.Terminated.Reason
	}
	return ""
}

func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen] + "..."
}
