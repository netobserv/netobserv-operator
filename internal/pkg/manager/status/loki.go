package status

import (
	"context"
	"fmt"
	"strings"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func checkLoki(ctx context.Context, c client.Client, fc *flowslatest.FlowCollector) metav1.Condition {
	if !fc.Spec.UseLoki() {
		return metav1.Condition{
			Type:    LokiIssue,
			Reason:  "Unused",
			Status:  metav1.ConditionUnknown,
			Message: "Loki is disabled",
		}
	}
	if fc.Spec.Loki.Mode != flowslatest.LokiModeLokiStack {
		return metav1.Condition{
			Type:    LokiIssue,
			Reason:  "Unused",
			Status:  metav1.ConditionUnknown,
			Message: "Loki is not configured in LokiStack mode",
		}
	}
	lokiStack := &lokiv1.LokiStack{}
	nsname := types.NamespacedName{Name: fc.Spec.Loki.LokiStack.Name, Namespace: fc.Spec.Namespace}
	if len(fc.Spec.Loki.LokiStack.Namespace) > 0 {
		nsname.Namespace = fc.Spec.Loki.LokiStack.Namespace
	}
	err := c.Get(ctx, nsname, lokiStack)
	if err != nil {
		if kerr.IsNotFound(err) {
			return metav1.Condition{
				Type:    LokiIssue,
				Reason:  "LokiStackNotFound",
				Status:  metav1.ConditionTrue,
				Message: fmt.Sprintf("The configured LokiStack reference could not be found [name: %s, namespace: %s]", nsname.Name, nsname.Namespace),
			}
		}
		return metav1.Condition{
			Type:    LokiIssue,
			Reason:  "Error",
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("Error while fetching configured LokiStack: %s", err.Error()),
		}
	}

	// Check LokiStack status conditions
	if len(lokiStack.Status.Conditions) > 0 {
		// Check for specific problem conditions first (Degraded, Error, Failed)
		// These provide more actionable information than just "NotReady"
		// Note: Warnings are handled separately in checkLokiWarnings()
		var issues []string
		for _, cond := range lokiStack.Status.Conditions {
			// Skip the Ready, Pending, and Warning conditions
			if cond.Type == "Ready" || cond.Type == "Pending" || cond.Type == "Warning" {
				continue
			}
			// If any condition has Status=True for a problem condition, report it
			condTypeLower := strings.ToLower(cond.Type)
			if cond.Status == metav1.ConditionTrue && (strings.Contains(condTypeLower, "error") ||
				strings.Contains(condTypeLower, "degraded") ||
				strings.Contains(condTypeLower, "failed")) {
				issues = append(issues, fmt.Sprintf("%s: %s", cond.Type, cond.Message))
			}
		}
		if len(issues) > 0 {
			return metav1.Condition{
				Type:    LokiIssue,
				Reason:  "LokiStackIssues",
				Status:  metav1.ConditionTrue,
				Message: fmt.Sprintf("LokiStack has issues [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(issues, "; ")),
			}
		}

		// If no specific issues found, check the Ready condition
		readyCond := meta.FindStatusCondition(lokiStack.Status.Conditions, "Ready")
		if readyCond != nil && readyCond.Status != metav1.ConditionTrue {
			return metav1.Condition{
				Type:    LokiIssue,
				Reason:  "LokiStackNotReady",
				Status:  metav1.ConditionTrue,
				Message: fmt.Sprintf("LokiStack is not ready [name: %s, namespace: %s]: %s - %s", nsname.Name, nsname.Namespace, readyCond.Reason, readyCond.Message),
			}
		}
	}

	// Check LokiStack component status for failed or pending pods
	componentIssues := checkLokiStackComponents(&lokiStack.Status.Components)
	if len(componentIssues) > 0 {
		return metav1.Condition{
			Type:    LokiIssue,
			Reason:  "LokiStackComponentIssues",
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("LokiStack components have issues [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(componentIssues, "; ")),
		}
	}

	return metav1.Condition{
		Type:   LokiIssue,
		Reason: "NoIssue",
		Status: metav1.ConditionFalse,
	}
}

func checkLokiStackComponents(components *lokiv1.LokiStackComponentStatus) []string {
	if components == nil {
		return nil
	}

	var issues []string

	// Helper function to check a component's pod status map
	checkComponent := func(name string, podStatusMap lokiv1.PodStatusMap) {
		if len(podStatusMap) == 0 {
			return
		}

		// Check for failed pods
		if failedPods, ok := podStatusMap[lokiv1.PodFailed]; ok && len(failedPods) > 0 {
			issues = append(issues, fmt.Sprintf("%s has %d failed pod(s): %s", name, len(failedPods), strings.Join(failedPods, ", ")))
		}

		// Check for pending pods
		if pendingPods, ok := podStatusMap[lokiv1.PodPending]; ok && len(pendingPods) > 0 {
			issues = append(issues, fmt.Sprintf("%s has %d pending pod(s): %s", name, len(pendingPods), strings.Join(pendingPods, ", ")))
		}

		// Check for unknown status pods
		if unknownPods, ok := podStatusMap[lokiv1.PodStatusUnknown]; ok && len(unknownPods) > 0 {
			issues = append(issues, fmt.Sprintf("%s has %d pod(s) with unknown status: %s", name, len(unknownPods), strings.Join(unknownPods, ", ")))
		}
	}

	// Check all LokiStack components
	checkComponent("Compactor", components.Compactor)
	checkComponent("Distributor", components.Distributor)
	checkComponent("IndexGateway", components.IndexGateway)
	checkComponent("Ingester", components.Ingester)
	checkComponent("Querier", components.Querier)
	checkComponent("QueryFrontend", components.QueryFrontend)
	checkComponent("Gateway", components.Gateway)
	checkComponent("Ruler", components.Ruler)

	return issues
}

func checkLokiWarnings(ctx context.Context, c client.Client, fc *flowslatest.FlowCollector) metav1.Condition {
	if !fc.Spec.UseLoki() {
		return metav1.Condition{
			Type:   LokiWarning,
			Reason: "Unused",
			Status: metav1.ConditionUnknown,
		}
	}
	if fc.Spec.Loki.Mode != flowslatest.LokiModeLokiStack {
		return metav1.Condition{
			Type:   LokiWarning,
			Reason: "Unused",
			Status: metav1.ConditionUnknown,
		}
	}
	lokiStack := &lokiv1.LokiStack{}
	nsname := types.NamespacedName{Name: fc.Spec.Loki.LokiStack.Name, Namespace: fc.Spec.Namespace}
	if len(fc.Spec.Loki.LokiStack.Namespace) > 0 {
		nsname.Namespace = fc.Spec.Loki.LokiStack.Namespace
	}
	err := c.Get(ctx, nsname, lokiStack)
	if err != nil {
		// If we can't get the LokiStack, don't report warnings
		// (the main checkLoki will report the error)
		return metav1.Condition{
			Type:   LokiWarning,
			Reason: "NoWarning",
			Status: metav1.ConditionFalse,
		}
	}

	// Check for Warning conditions
	var warnings []string
	for _, cond := range lokiStack.Status.Conditions {
		condTypeLower := strings.ToLower(cond.Type)
		if cond.Status == metav1.ConditionTrue && strings.Contains(condTypeLower, "warning") {
			warnings = append(warnings, fmt.Sprintf("%s: %s", cond.Type, cond.Message))
		}
	}

	if len(warnings) > 0 {
		return metav1.Condition{
			Type:    LokiWarning,
			Reason:  "LokiStackWarnings",
			Status:  metav1.ConditionTrue,
			Message: fmt.Sprintf("LokiStack has warnings [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(warnings, "; ")),
		}
	}

	return metav1.Condition{
		Type:   LokiWarning,
		Reason: "NoWarning",
		Status: metav1.ConditionFalse,
	}
}
