package lokistack

import (
	"context"
	"fmt"
	"strings"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type Watcher struct {
	cl                 client.Client
	mgr                *manager.Manager
	getController      func() controller.Controller
	status             status.Instance
	lokiWatcherStarted bool
}

func Start(ctx context.Context, mgr *manager.Manager, builder *builder.TypedBuilder[reconcile.Request], getController func() controller.Controller) *Watcher {
	log := log.FromContext(ctx)
	log.Info("Starting LokiStack watcher")
	lsw := Watcher{
		cl:            mgr.Client,
		mgr:           mgr,
		getController: getController,
		status:        mgr.Status.ForComponent(status.LokiStack),
	}

	if mgr.ClusterInfo.HasLokiStack(ctx) {
		builder.Watches(
			&lokiv1.LokiStack{},
			handler.EnqueueRequestsFromMapFunc(func(_ context.Context, _ client.Object) []reconcile.Request {
				// When a LokiStack changes, trigger reconcile of the FlowCollector
				return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
			}),
		)
		lsw.lokiWatcherStarted = true
		log.Info("LokiStack CRD detected")
	}

	return &lsw
}

func (lsw *Watcher) Reconcile(ctx context.Context, fc *flowslatest.FlowCollector) (ret status.ComponentStatus) {
	l := log.Log.WithName("lokistack-watcher")
	ctx = log.IntoContext(ctx, l)

	defer func() {
		ret = lsw.status.Get()
	}()
	lsw.status.SetUnknown()

	if !fc.Spec.UseLoki() {
		lsw.status.SetUnused("Loki is disabled")
		return
	}

	if fc.Spec.Loki.Mode != flowslatest.LokiModeLokiStack {
		lsw.status.SetUnused("Loki is not configured in LokiStack mode")
		return
	}

	if !lsw.mgr.ClusterInfo.HasLokiStack(ctx) {
		lsw.status.SetFailure("LokiStackAPIMissing", "Loki is configured in LokiStack mode, but LokiStack API is missing; check that the Loki Operator is correctly installed.")
		return
	}

	if err := lsw.ensureLokiStackWatcher(ctx); err != nil {
		l.Error(err, "Failed to start LokiStack watcher")
		lsw.status.SetFailure("CantWatchLokiStack", err.Error())
		// Don't fail reconciliation, just log the error
	}

	if err := lsw.checkStatus(ctx, fc); err != nil {
		l.Error(err, "Failed to fetch LokiStack status")
	}

	return
}

func (lsw *Watcher) ensureLokiStackWatcher(ctx context.Context) error {
	if lsw.lokiWatcherStarted {
		return nil
	}

	// LokiStack API is now available, start the watcher
	log := log.FromContext(ctx)
	log.Info("LokiStack CRD detected after startup, starting watcher")

	h := handler.TypedEnqueueRequestsFromMapFunc(func(_ context.Context, _ *lokiv1.LokiStack) []reconcile.Request {
		// When a LokiStack changes, trigger reconcile of the FlowCollector
		return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
	})

	src := source.Kind(lsw.mgr.GetCache(), &lokiv1.LokiStack{}, h)
	err := lsw.getController().Watch(src)
	if err != nil {
		return fmt.Errorf("failed to start LokiStack watcher: %w", err)
	}

	lsw.lokiWatcherStarted = true
	log.Info("LokiStack watcher started successfully")
	return nil
}

func (lsw *Watcher) checkStatus(ctx context.Context, fc *flowslatest.FlowCollector) error {
	lokiStack := &lokiv1.LokiStack{}
	nsname := types.NamespacedName{Name: fc.Spec.Loki.LokiStack.Name, Namespace: fc.Spec.Namespace}
	if len(fc.Spec.Loki.LokiStack.Namespace) > 0 {
		nsname.Namespace = fc.Spec.Loki.LokiStack.Namespace
	}
	err := lsw.cl.Get(ctx, nsname, lokiStack)
	if err != nil {
		lsw.status.SetFailure("CantFetchLokiStack", err.Error())
		return err
	}

	// Check LokiStack status conditions
	var issues, warnings []string
	if len(lokiStack.Status.Conditions) > 0 {
		// Check for specific problem conditions first (Degraded, Error, Failed)
		// These provide more actionable information than just "NotReady"
		for _, cond := range lokiStack.Status.Conditions {
			// Skip the Ready and Pending conditions
			if cond.Type == "Ready" || cond.Type == "Pending" {
				continue
			}
			// If any condition has Status=True for a problem condition, report it
			condTypeLower := strings.ToLower(cond.Type)
			if cond.Status == metav1.ConditionTrue && (strings.Contains(condTypeLower, "error") ||
				strings.Contains(condTypeLower, "degraded") ||
				strings.Contains(condTypeLower, "failed")) {
				issues = append(issues, fmt.Sprintf("%s: %s", cond.Type, cond.Message))
			} else if cond.Status == metav1.ConditionTrue && strings.Contains(condTypeLower, "warning") {
				warnings = append(warnings, fmt.Sprintf("%s: %s", cond.Type, cond.Message))
			}
		}
	}

	// Check LokiStack component status for failed or pending pods
	componentIssues := checkLokiStackComponents(&lokiStack.Status.Components)
	allIssues := issues
	// Aggregate warnings and component issues
	allIssues = append(allIssues, componentIssues...)
	allIssues = append(allIssues, warnings...)

	if len(issues) > 0 {
		lsw.status.SetFailure(
			"LokiStackIssues",
			fmt.Sprintf("LokiStack has issues [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(allIssues, "; ")),
		)
		return nil
	}

	// If no specific issues found, check the Ready condition
	readyCond := meta.FindStatusCondition(lokiStack.Status.Conditions, "Ready")
	if readyCond != nil && readyCond.Status != metav1.ConditionTrue {
		msg := readyCond.Message
		if len(allIssues) > 0 {
			msg += "; " + strings.Join(allIssues, "; ")
		}
		lsw.status.SetNotReady(
			"LokiStackNotReady",
			fmt.Sprintf("LokiStack is not ready [name: %s, namespace: %s]: %s - %s", nsname.Name, nsname.Namespace, readyCond.Reason, msg),
		)
		return nil
	}

	if len(componentIssues) > 0 {
		lsw.status.SetNotReady(
			"LokiStackComponentIssues",
			fmt.Sprintf("LokiStack components have issues [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(allIssues, "; ")),
		)
		return nil
	}

	if len(warnings) > 0 {
		lsw.status.SetDegraded(
			"LokiStackWarnings",
			fmt.Sprintf("LokiStack has warnings [name: %s, namespace: %s]: %s", nsname.Name, nsname.Namespace, strings.Join(allIssues, "; ")),
		)
		return nil
	}

	lsw.status.SetReady()
	return nil
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
