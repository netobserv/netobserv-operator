package status

import (
	"context"
	"fmt"
	"strings"
	"sync"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ComponentName string

const (
	FlowCollectorController     ComponentName = "FlowCollectorController"
	EBPFAgents                  ComponentName = "EBPFAgents"
	WebConsole                  ComponentName = "WebConsole"
	FLPParent                   ComponentName = "FLPParent"
	FLPMonolith                 ComponentName = "FLPMonolith"
	FLPTransformer              ComponentName = "FLPTransformer"
	Monitoring                  ComponentName = "Monitoring"
	StaticController            ComponentName = "StaticController"
	NetworkPolicy               ComponentName = "NetworkPolicy"
	DemoLoki                    ComponentName = "DemoLoki"
	LokiStack                   ComponentName = "LokiStack"
	ConditionConfigurationIssue               = "ConfigurationIssue"
)

type Manager struct {
	statuses      sync.Map
	exporters     sync.Map
	eventRecorder record.EventRecorder
	prevStatuses  sync.Map
}

func NewManager() *Manager {
	return &Manager{}
}

// SetEventRecorder sets the EventRecorder for emitting Kubernetes Events.
func (s *Manager) SetEventRecorder(recorder record.EventRecorder) {
	s.eventRecorder = recorder
}

func (s *Manager) getStatus(cpnt ComponentName) *ComponentStatus {
	v, _ := s.statuses.Load(cpnt)
	if v != nil {
		if s, ok := v.(ComponentStatus); ok {
			return &s
		}
	}
	return nil
}

func (s *Manager) setInProgress(cpnt ComponentName, reason, message string) {
	cs := s.preserveReplicas(cpnt)
	cs.Status = StatusInProgress
	cs.Reason = reason
	cs.Message = message
	s.statuses.Store(cpnt, cs)
}

func (s *Manager) setFailure(cpnt ComponentName, reason, message string) {
	cs := s.preserveReplicas(cpnt)
	cs.Status = StatusFailure
	cs.Reason = reason
	cs.Message = message
	s.statuses.Store(cpnt, cs)
}

func (s *Manager) setDegraded(cpnt ComponentName, reason, message string) {
	cs := s.preserveReplicas(cpnt)
	cs.Status = StatusDegraded
	cs.Reason = reason
	cs.Message = message
	s.statuses.Store(cpnt, cs)
}

func (s *Manager) preserveReplicas(cpnt ComponentName) ComponentStatus {
	cs := ComponentStatus{Name: cpnt}
	if existing := s.getStatus(cpnt); existing != nil {
		cs.DesiredReplicas = existing.DesiredReplicas
		cs.ReadyReplicas = existing.ReadyReplicas
	}
	return cs
}

func (s *Manager) hasFailure(cpnt ComponentName) bool {
	v, _ := s.statuses.Load(cpnt)
	return v != nil && v.(ComponentStatus).Status == StatusFailure
}

func (s *Manager) setReady(cpnt ComponentName) {
	existing := s.getStatus(cpnt)
	cs := ComponentStatus{
		Name:   cpnt,
		Status: StatusReady,
	}
	if existing != nil {
		cs.DesiredReplicas = existing.DesiredReplicas
		cs.ReadyReplicas = existing.ReadyReplicas
	}
	s.statuses.Store(cpnt, cs)
}

func (s *Manager) setUnknown(cpnt ComponentName) {
	s.statuses.Store(cpnt, ComponentStatus{
		Name:   cpnt,
		Status: StatusUnknown,
	})
}

func (s *Manager) setUnused(cpnt ComponentName, message string) {
	s.statuses.Store(cpnt, ComponentStatus{
		Name:    cpnt,
		Status:  StatusUnused,
		Reason:  "ComponentUnused",
		Message: message,
	})
}

func (s *Manager) getConditions() []metav1.Condition {
	global := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Ready",
	}
	conds := []metav1.Condition{}
	counters := make(map[Status]int)
	s.statuses.Range(func(_, v any) bool {
		status := v.(ComponentStatus)
		conds = append(conds, status.toCondition())
		counters[status.Status]++
		return true
	})
	global.Message = fmt.Sprintf("%d ready components, %d with failure, %d pending, %d degraded", counters[StatusReady], counters[StatusFailure], counters[StatusInProgress], counters[StatusDegraded])
	if counters[StatusFailure] > 0 {
		global.Status = metav1.ConditionFalse
		global.Reason = "Failure"
	} else if counters[StatusInProgress] > 0 {
		global.Status = metav1.ConditionFalse
		global.Reason = "Pending"
	} else if counters[StatusDegraded] > 0 {
		global.Reason = "Ready,Degraded"
	}
	return append([]metav1.Condition{global}, conds...)
}

// populateComponentStatuses maps internal ComponentStatus instances to the CRD status fields.
// Integrations are always rebuilt from the in-memory map. Component fields are rebuilt from the map,
// then agent/plugin may be merged from prevComponents (see preserveAgentOrPluginSnapshot).
// prevComponents is the FlowCollector's status.components from the API before this update; when
// another controller commits while the legacy reconciler is between ForComponent (placeholder Unknown)
// and the real status, we keep agent/plugin from prev so users do not see transient Unknown.
func (s *Manager) populateComponentStatuses(fc *flowslatest.FlowCollector, prevComponents *flowslatest.FlowCollectorComponentsStatus) {
	fc.Status.Components = &flowslatest.FlowCollectorComponentsStatus{}
	fc.Status.Integrations = &flowslatest.FlowCollectorIntegrationsStatus{}

	s.statuses.Range(func(_, v any) bool {
		cs := v.(ComponentStatus)
		switch cs.Name {
		case EBPFAgents:
			fc.Status.Components.Agent = cs.toCRDStatus()
		case FLPParent, FLPMonolith, FLPTransformer:
			fc.Status.Components.Processor = mergeProcessorStatus(fc.Status.Components.Processor, &cs)
		case WebConsole:
			fc.Status.Components.Plugin = cs.toCRDStatus()
		case Monitoring:
			fc.Status.Integrations.Monitoring = cs.toCRDStatus()
		case LokiStack, DemoLoki:
			existingIsWeak := fc.Status.Integrations.Loki == nil ||
				fc.Status.Integrations.Loki.State == string(StatusUnknown) ||
				fc.Status.Integrations.Loki.State == string(StatusUnused)
			if existingIsWeak || cs.Status == StatusFailure || cs.Status == StatusDegraded || cs.Status == StatusReady {
				fc.Status.Integrations.Loki = cs.toCRDStatus()
			}
		case FlowCollectorController, StaticController, NetworkPolicy:
			// Reported only through conditions, not dedicated CRD status fields.
		}
		return true
	})

	var exporters []flowslatest.FlowCollectorExporterStatus
	s.exporters.Range(func(_, v any) bool {
		exp := v.(flowslatest.FlowCollectorExporterStatus)
		exporters = append(exporters, exp)
		return true
	})
	fc.Status.Integrations.Exporters = exporters

	if prevComponents != nil {
		fc.Status.Components.Agent = preserveAgentOrPluginSnapshot(fc.Status.Components.Agent, prevComponents.Agent)
		fc.Status.Components.Plugin = preserveAgentOrPluginSnapshot(fc.Status.Components.Plugin, prevComponents.Plugin)
	}
}

// preserveAgentOrPluginSnapshot avoids publishing a transient Unknown written by ForComponent before
// the owning controller finishes, or losing agent/plugin when their keys are absent from the map.
func preserveAgentOrPluginSnapshot(cur, prev *flowslatest.FlowCollectorComponentStatus) *flowslatest.FlowCollectorComponentStatus {
	if prev == nil {
		return cur
	}
	if cur == nil {
		return prev
	}
	if cur.State == string(StatusUnknown) && cur.Reason == "" && cur.Message == "" {
		return prev
	}
	return cur
}

// mergeProcessorStatus handles FLP processor status aggregation from parent, monolith, and transformer.
// Active sub-reconcilers (non-Unknown/Unused) take priority over the parent status.
func mergeProcessorStatus(existing *flowslatest.FlowCollectorComponentStatus, cs *ComponentStatus) *flowslatest.FlowCollectorComponentStatus {
	isInactive := cs.Status == StatusUnknown || cs.Status == StatusUnused
	existingIsWeak := existing == nil ||
		existing.State == string(StatusUnknown) ||
		existing.State == string(StatusUnused)

	if isInactive {
		if existing == nil {
			return cs.toCRDStatus()
		}
		return existing
	}

	if cs.Name == FLPParent {
		if existingIsWeak {
			return cs.toCRDStatus()
		}
		return existing
	}

	crd := cs.toCRDStatus()
	if existingIsWeak || cs.Status == StatusFailure || cs.Status == StatusInProgress || cs.Status == StatusDegraded {
		return crd
	}
	if existing.State == string(StatusReady) && crd.DesiredReplicas != nil {
		existing.DesiredReplicas = crd.DesiredReplicas
		existing.ReadyReplicas = crd.ReadyReplicas
		existing.UnhealthyPodCount = crd.UnhealthyPodCount
		existing.PodIssues = crd.PodIssues
	}
	return existing
}

func (s *Manager) Sync(ctx context.Context, c client.Client) {
	s.updateStatus(ctx, c)
}

func (s *Manager) emitStateTransitionEvents(ctx context.Context, fc *flowslatest.FlowCollector) {
	if s.eventRecorder == nil {
		return
	}
	rlog := log.FromContext(ctx)

	s.statuses.Range(func(key, v any) bool {
		cpnt := key.(ComponentName)
		current := v.(ComponentStatus)

		prev, hasPrev := s.prevStatuses.Load(cpnt)
		s.prevStatuses.Store(cpnt, current)
		if !hasPrev {
			return true
		}
		prevStatus := prev.(ComponentStatus)
		if prevStatus.Status == current.Status {
			return true
		}

		switch {
		case current.Status == StatusFailure:
			msg := fmt.Sprintf("Component %s entered failure state: %s - %s", cpnt, current.Reason, current.Message)
			s.eventRecorder.Event(fc, "Warning", "ComponentFailure", msg)
			rlog.Info("Event emitted", "type", "ComponentFailure", "component", cpnt)
		case current.Status == StatusDegraded:
			msg := fmt.Sprintf("Component %s degraded: %s - %s", cpnt, current.Reason, current.Message)
			s.eventRecorder.Event(fc, "Warning", "ComponentDegraded", msg)
		case prevStatus.Status == StatusFailure && (current.Status == StatusReady || current.Status == StatusInProgress):
			msg := fmt.Sprintf("Component %s recovered from failure", cpnt)
			s.eventRecorder.Event(fc, "Normal", "ComponentRecovered", msg)
		}
		return true
	})
}

func (s *Manager) updateStatus(ctx context.Context, c client.Client) {
	rlog := log.FromContext(ctx)
	rlog.Info("Updating FlowCollector status")

	var updatedFC *flowslatest.FlowCollector
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fc := flowslatest.FlowCollector{}
		if err := c.Get(ctx, constants.FlowCollectorName, &fc); err != nil {
			if kerr.IsNotFound(err) {
				return nil
			}
			return err
		}
		conditions := s.getConditions()
		conditions = append(conditions, checkValidation(ctx, &fc))
		if kafkaCond := s.GetKafkaCondition(); kafkaCond != nil {
			conditions = append(conditions, *kafkaCond)
		} else if ts := s.getStatus(FLPTransformer); ts == nil || ts.Status != StatusUnknown {
			// Skip removal while FLPTransformer is placeholder Unknown (GetKafkaCondition is nil then too).
			meta.RemoveStatusCondition(&fc.Status.Conditions, "KafkaReady")
		}
		for _, cond := range conditions {
			meta.SetStatusCondition(&fc.Status.Conditions, cond)
		}
		var prevComponents *flowslatest.FlowCollectorComponentsStatus
		if fc.Status.Components != nil {
			prevComponents = fc.Status.Components.DeepCopy()
		}
		s.populateComponentStatuses(&fc, prevComponents)
		if err := c.Status().Update(ctx, &fc); err != nil {
			return err
		}
		updatedFC = &fc
		return nil
	})

	if err != nil {
		rlog.Error(err, "failed to update FlowCollector status")
	} else if updatedFC != nil {
		s.emitStateTransitionEvents(ctx, updatedFC)
	}
}

func checkValidation(ctx context.Context, fc *flowslatest.FlowCollector) metav1.Condition {
	warnings, err := fc.Validate(ctx, fc)
	if err != nil {
		return metav1.Condition{
			Type:    ConditionConfigurationIssue,
			Reason:  "Error",
			Status:  metav1.ConditionTrue,
			Message: err.Error(),
		}
	}
	if len(warnings) > 0 {
		return metav1.Condition{
			Type:    ConditionConfigurationIssue,
			Reason:  "Warnings",
			Status:  metav1.ConditionTrue,
			Message: strings.Join(warnings, "; "),
		}
	}
	return metav1.Condition{
		Type:   ConditionConfigurationIssue,
		Reason: "Valid",
		Status: metav1.ConditionFalse,
	}
}

// GetKafkaCondition returns a KafkaReady condition if Kafka is being used.
// It aggregates the health of Kafka-related components: agent (when using Kafka export),
// FLP transformer (Kafka consumer), and any Kafka exporters.
func (s *Manager) GetKafkaCondition() *metav1.Condition {
	hasKafkaIssue := false
	var messages []string

	// Check transformer (only used with Kafka)
	if ts := s.getStatus(FLPTransformer); ts != nil && ts.Status != StatusUnknown && ts.Status != StatusUnused {
		if ts.Status == StatusFailure || ts.Status == StatusDegraded {
			hasKafkaIssue = true
			messages = append(messages, fmt.Sprintf("Transformer: %s", ts.Message))
		}
		if ts.PodHealth.UnhealthyCount > 0 {
			hasKafkaIssue = true
			messages = append(messages, fmt.Sprintf("Transformer pods: %s", ts.PodHealth.Issues))
		}
	}

	// Check agent for Kafka-related pod issues
	if as := s.getStatus(EBPFAgents); as != nil {
		if as.PodHealth.UnhealthyCount > 0 && strings.Contains(strings.ToLower(as.PodHealth.Issues), "kafka") {
			hasKafkaIssue = true
			messages = append(messages, fmt.Sprintf("Agent pods: %s", as.PodHealth.Issues))
		}
	}

	// Check Kafka exporters
	s.exporters.Range(func(_, v any) bool {
		exp := v.(flowslatest.FlowCollectorExporterStatus)
		if exp.Type == "Kafka" && exp.State == string(StatusFailure) {
			hasKafkaIssue = true
			messages = append(messages, fmt.Sprintf("Exporter %s: %s", exp.Name, exp.Message))
		}
		return true
	})

	if hasKafkaIssue {
		return &metav1.Condition{
			Type:    "KafkaReady",
			Status:  metav1.ConditionFalse,
			Reason:  "KafkaIssue",
			Message: strings.Join(messages, "; "),
		}
	}

	// If transformer is active (Kafka mode), report its state
	if ts := s.getStatus(FLPTransformer); ts != nil {
		switch ts.Status {
		case StatusReady:
			return &metav1.Condition{
				Type:   "KafkaReady",
				Status: metav1.ConditionTrue,
				Reason: "Ready",
			}
		case StatusInProgress:
			return &metav1.Condition{
				Type:    "KafkaReady",
				Status:  metav1.ConditionFalse,
				Reason:  "KafkaPending",
				Message: "Kafka transformer is rolling out",
			}
		case StatusUnknown, StatusUnused, StatusFailure, StatusDegraded:
			// Failure/Degraded already handled above via hasKafkaIssue.
			// Unused means transformer not used (e.g. direct mode). Unknown is either not yet
			// reconciled (ForComponent placeholder) or indeterminate — no KafkaReady row in either case.
		}
	}

	return nil
}

// SetExporterStatus sets the status of a specific exporter by name.
func (s *Manager) SetExporterStatus(name, exporterType, state, reason, message string) {
	s.exporters.Store(name, flowslatest.FlowCollectorExporterStatus{
		Name:    name,
		Type:    exporterType,
		State:   state,
		Reason:  reason,
		Message: message,
	})
}

// NeedsRequeue returns true if any component is in a transient state that warrants
// periodic re-checking (e.g., pods crashing into CrashLoopBackOff after a deployment rollout).
// Controllers should use this to return RequeueAfter so the status keeps updating
// even when DaemonSet/Deployment watches don't fire for pod-level changes.
func (s *Manager) NeedsRequeue() bool {
	needsRequeue := false
	s.statuses.Range(func(_, v any) bool {
		cs := v.(ComponentStatus)
		if cs.Status == StatusInProgress || cs.PodHealth.UnhealthyCount > 0 {
			needsRequeue = true
			return false
		}
		return true
	})
	return needsRequeue
}

// ClearExporters removes all exporter statuses (call before re-populating).
func (s *Manager) ClearExporters() {
	s.exporters.Range(func(key, _ any) bool {
		s.exporters.Delete(key)
		return true
	})
}

func (s *Manager) ForComponent(cpnt ComponentName) Instance {
	s.setUnknown(cpnt)
	return Instance{cpnt: cpnt, s: s}
}

type Instance struct {
	cpnt ComponentName
	s    *Manager
}

func (i *Instance) Get() ComponentStatus {
	s := i.s.getStatus(i.cpnt)
	if s != nil {
		return *s
	}
	return ComponentStatus{Name: i.cpnt, Status: StatusUnknown}
}

func (i *Instance) SetReady() {
	i.s.setReady(i.cpnt)
}

func (i *Instance) SetUnknown() {
	i.s.setUnknown(i.cpnt)
}

func (i *Instance) SetUnused(message string) {
	i.s.setUnused(i.cpnt, message)
}

// CheckDeploymentProgress sets the status either as In Progress, or Ready,
// and populates replica counts from the Deployment status.
func (i *Instance) CheckDeploymentProgress(d *appsv1.Deployment) {
	if d == nil {
		i.s.setInProgress(i.cpnt, "DeploymentNotCreated", "Deployment not created")
		return
	}
	defer i.setDeploymentReplicas(d)
	for _, c := range d.Status.Conditions {
		if c.Type == appsv1.DeploymentAvailable {
			if c.Status != v1.ConditionTrue {
				i.s.setInProgress(i.cpnt, "DeploymentNotReady", fmt.Sprintf("Deployment %s not ready: %d/%d (%s)", d.Name, d.Status.ReadyReplicas, d.Status.Replicas, c.Message))
			} else {
				i.s.setReady(i.cpnt)
			}
			return
		}
	}
	if d.Status.ReadyReplicas == d.Status.Replicas && d.Status.Replicas > 0 {
		i.s.setReady(i.cpnt)
	} else {
		i.s.setInProgress(i.cpnt, "DeploymentNotReady", fmt.Sprintf("Deployment %s not ready: %d/%d (missing condition)", d.Name, d.Status.ReadyReplicas, d.Status.Replicas))
	}
}

func (i *Instance) setDeploymentReplicas(d *appsv1.Deployment) {
	cs := i.s.getStatus(i.cpnt)
	if cs != nil {
		var desired int32 = 1
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		cs.DesiredReplicas = ptr.To(desired)
		cs.ReadyReplicas = ptr.To(d.Status.ReadyReplicas)
		i.s.statuses.Store(i.cpnt, *cs)
	}
}

// CheckDaemonSetProgress sets the status either as In Progress, or Ready,
// and populates replica counts from the DaemonSet status.
func (i *Instance) CheckDaemonSetProgress(ds *appsv1.DaemonSet) {
	if ds == nil {
		i.s.setInProgress(i.cpnt, "DaemonSetNotCreated", "DaemonSet not created")
		return
	}
	defer i.setDaemonSetReplicas(ds)
	if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled {
		i.s.setInProgress(i.cpnt, "DaemonSetNotReady", fmt.Sprintf("DaemonSet %s not ready: %d/%d", ds.Name, ds.Status.NumberReady, ds.Status.DesiredNumberScheduled))
	} else {
		i.s.setReady(i.cpnt)
	}
}

func (i *Instance) setDaemonSetReplicas(ds *appsv1.DaemonSet) {
	cs := i.s.getStatus(i.cpnt)
	if cs != nil {
		cs.DesiredReplicas = ptr.To(ds.Status.DesiredNumberScheduled)
		cs.ReadyReplicas = ptr.To(ds.Status.NumberReady)
		i.s.statuses.Store(i.cpnt, *cs)
	}
}

// CheckDeploymentHealth combines CheckDeploymentProgress with pod health checking.
// If the deployment has unhealthy pods, it inspects container statuses for details.
func (i *Instance) CheckDeploymentHealth(ctx context.Context, c client.Client, d *appsv1.Deployment) {
	i.CheckDeploymentProgress(d)
	if d == nil || d.Spec.Selector == nil {
		return
	}
	if d.Status.ReadyReplicas < d.Status.Replicas || d.Status.UnavailableReplicas > 0 {
		health := CheckPodHealth(ctx, c, d.Namespace, d.Spec.Selector.MatchLabels)
		i.setPodHealth(health)
	}
}

// CheckDaemonSetHealth combines CheckDaemonSetProgress with pod health checking.
// If the DaemonSet has unhealthy pods, it inspects container statuses for details.
func (i *Instance) CheckDaemonSetHealth(ctx context.Context, c client.Client, ds *appsv1.DaemonSet) {
	i.CheckDaemonSetProgress(ds)
	if ds == nil || ds.Spec.Selector == nil {
		return
	}
	if ds.Status.NumberReady < ds.Status.DesiredNumberScheduled || ds.Status.NumberUnavailable > 0 {
		health := CheckPodHealth(ctx, c, ds.Namespace, ds.Spec.Selector.MatchLabels)
		i.setPodHealth(health)
	}
}

func (i *Instance) setPodHealth(health PodHealthSummary) {
	cs := i.s.getStatus(i.cpnt)
	if cs != nil {
		cs.PodHealth = health
		if health.UnhealthyCount > 0 && (cs.Status == StatusReady || cs.Status == StatusInProgress) {
			cs.Status = StatusDegraded
			cs.Reason = "UnhealthyPods"
			cs.Message = health.Issues
		}
		i.s.statuses.Store(i.cpnt, *cs)
	}
}

func (i *Instance) SetCreatingDeployment(d *appsv1.Deployment) {
	i.s.setInProgress(i.cpnt, "CreatingDeployment", fmt.Sprintf("Creating deployment %s", d.Name))
}

func (i *Instance) SetCreatingDaemonSet(ds *appsv1.DaemonSet) {
	i.s.setInProgress(i.cpnt, "CreatingDaemonSet", fmt.Sprintf("Creating daemon set %s", ds.Name))
}

func (i *Instance) SetNotReady(reason, message string) {
	i.s.setInProgress(i.cpnt, reason, message)
}

func (i *Instance) SetFailure(reason, message string) {
	i.s.setFailure(i.cpnt, reason, message)
}

func (i *Instance) SetDegraded(reason, message string) {
	i.s.setDegraded(i.cpnt, reason, message)
}

func (i *Instance) Error(reason string, err error) error {
	i.SetFailure(reason, err.Error())
	return err
}

func (i *Instance) HasFailure() bool {
	return i.s.hasFailure(i.cpnt)
}

func (i *Instance) Commit(ctx context.Context, c client.Client) {
	i.s.Sync(ctx, c)
}
