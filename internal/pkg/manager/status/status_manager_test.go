package status

import (
	"context"
	"fmt"
	"slices"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func TestStatusWorkflow(t *testing.T) {
	s := NewManager()
	sl := s.ForComponent(FlowCollectorController)
	sm := s.ForComponent(Monitoring)

	sl.SetCreatingDaemonSet(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "test"}})
	sm.SetFailure("AnError", "bad one")

	conds := s.getConditions()
	assertHasConditionTypes(t, conds, []string{"Ready", "WaitingFlowCollectorController", "WaitingMonitoring"})
	assertHasCondition(t, conds, "Ready", "Failure", metav1.ConditionFalse)
	assertHasCondition(t, conds, "WaitingFlowCollectorController", "CreatingDaemonSet", metav1.ConditionTrue)
	assertHasCondition(t, conds, "WaitingMonitoring", "AnError", metav1.ConditionTrue)

	sl.CheckDaemonSetProgress(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Status: appsv1.DaemonSetStatus{
		DesiredNumberScheduled: 3,
		NumberReady:            1,
	}})
	sm.SetUnknown()

	conds = s.getConditions()
	assertHasConditionTypes(t, conds, []string{"Ready", "WaitingFlowCollectorController", "WaitingMonitoring"})
	assertHasCondition(t, conds, "Ready", "Pending", metav1.ConditionFalse)
	assertHasCondition(t, conds, "WaitingFlowCollectorController", "DaemonSetNotReady", metav1.ConditionTrue)
	assertHasCondition(t, conds, "WaitingMonitoring", "Unknown", metav1.ConditionUnknown)

	sl.CheckDaemonSetProgress(&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Status: appsv1.DaemonSetStatus{
		DesiredNumberScheduled: 3,
		NumberReady:            3,
	}})
	sm.SetUnused("message")

	conds = s.getConditions()
	assertHasConditionTypes(t, conds, []string{"Ready", "WaitingFlowCollectorController", "WaitingMonitoring"})
	assertHasCondition(t, conds, "Ready", "Ready", metav1.ConditionTrue)
	assertHasCondition(t, conds, "WaitingFlowCollectorController", "Ready", metav1.ConditionFalse)
	assertHasCondition(t, conds, "WaitingMonitoring", "ComponentUnused", metav1.ConditionUnknown)

	sl.CheckDeploymentProgress(&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "test"}, Status: appsv1.DeploymentStatus{
		ReadyReplicas: 2,
		Replicas:      2,
	}})
	sm.SetReady()

	conds = s.getConditions()
	assertHasConditionTypes(t, conds, []string{"Ready", "WaitingFlowCollectorController", "WaitingMonitoring"})
	assertHasCondition(t, conds, "Ready", "Ready", metav1.ConditionTrue)
	assertHasCondition(t, conds, "WaitingFlowCollectorController", "Ready", metav1.ConditionFalse)
	assertHasCondition(t, conds, "WaitingMonitoring", "Ready", metav1.ConditionFalse)
}

func TestDegradedStatus(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)
	plugin := s.ForComponent(WebConsole)

	agent.SetReady()
	plugin.SetDegraded("PluginRegistrationFailed", "console operator unreachable")

	conds := s.getConditions()
	assertHasCondition(t, conds, "Ready", "Ready,Degraded", metav1.ConditionTrue)
	assertHasCondition(t, conds, "WaitingEBPFAgents", "Ready", metav1.ConditionFalse)
	assertHasCondition(t, conds, "WaitingWebConsole", "PluginRegistrationFailed", metav1.ConditionTrue)

	cs := plugin.Get()
	assert.Equal(t, StatusDegraded, cs.Status)
	assert.Equal(t, "PluginRegistrationFailed", cs.Reason)
	assert.Equal(t, "console operator unreachable", cs.Message)
}

func TestUnusedStatusInCRD(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)
	agent.SetUnused("FlowCollector is on hold")

	cs := agent.Get()
	assert.Equal(t, StatusUnused, cs.Status)

	fc := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc, nil)
	require.NotNil(t, fc.Status.Components)
	require.NotNil(t, fc.Status.Components.Agent)
	assert.Equal(t, "Unused", fc.Status.Components.Agent.State)
	assert.Equal(t, "ComponentUnused", fc.Status.Components.Agent.Reason)
	assert.Equal(t, "FlowCollector is on hold", fc.Status.Components.Agent.Message)
}

func TestPopulatePreservesAgentPluginAgainstPlaceholderUnknown(t *testing.T) {
	s := NewManager()
	_ = s.ForComponent(EBPFAgents)
	_ = s.ForComponent(WebConsole)

	prev := &flowslatest.FlowCollectorComponentsStatus{
		Agent: &flowslatest.FlowCollectorComponentStatus{
			State:   "Unused",
			Reason:  "ComponentUnused",
			Message: "FlowCollector is on hold",
		},
		Plugin: &flowslatest.FlowCollectorComponentStatus{
			State:   "Unused",
			Reason:  "ComponentUnused",
			Message: "FlowCollector is on hold",
		},
	}

	fc := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc, prev)

	require.NotNil(t, fc.Status.Components.Agent)
	assert.Equal(t, "Unused", fc.Status.Components.Agent.State)
	require.NotNil(t, fc.Status.Components.Plugin)
	assert.Equal(t, "Unused", fc.Status.Components.Plugin.State)
}

func TestReplicaCounts(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	agent.CheckDaemonSetProgress(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent"},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 5,
			UpdatedNumberScheduled: 5,
			NumberReady:            5,
		},
	})

	cs := agent.Get()
	assert.Equal(t, StatusReady, cs.Status)
	assert.NotNil(t, cs.DesiredReplicas)
	assert.NotNil(t, cs.ReadyReplicas)
	assert.Equal(t, int32(5), *cs.DesiredReplicas)
	assert.Equal(t, int32(5), *cs.ReadyReplicas)
}

func TestReplicaPreservationAcrossTransitions(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	agent.CheckDaemonSetProgress(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent"},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 5,
			UpdatedNumberScheduled: 5,
			NumberReady:            5,
		},
	})
	cs := agent.Get()
	require.Equal(t, int32(5), *cs.DesiredReplicas)

	// Transition to failure: replicas should be preserved
	agent.SetFailure("AgentKafkaError", "cannot connect to Kafka")
	cs = agent.Get()
	assert.Equal(t, StatusFailure, cs.Status)
	require.NotNil(t, cs.DesiredReplicas, "DesiredReplicas should survive failure transition")
	assert.Equal(t, int32(5), *cs.DesiredReplicas)

	// Transition to degraded: replicas should still be preserved
	agent.SetDegraded("SomeWarning", "non-critical issue")
	cs = agent.Get()
	assert.Equal(t, StatusDegraded, cs.Status)
	require.NotNil(t, cs.DesiredReplicas, "DesiredReplicas should survive degraded transition")
	assert.Equal(t, int32(5), *cs.DesiredReplicas)

	// Transition to in-progress: replicas should still be preserved
	agent.SetNotReady("Updating", "rolling update in progress")
	cs = agent.Get()
	assert.Equal(t, StatusInProgress, cs.Status)
	require.NotNil(t, cs.DesiredReplicas, "DesiredReplicas should survive in-progress transition")
	assert.Equal(t, int32(5), *cs.DesiredReplicas)

	// Transition back to ready: replicas should still be preserved
	agent.SetReady()
	cs = agent.Get()
	assert.Equal(t, StatusReady, cs.Status)
	require.NotNil(t, cs.DesiredReplicas)
	assert.Equal(t, int32(5), *cs.DesiredReplicas)
}

func TestDeploymentReplicaCounts(t *testing.T) {
	s := NewManager()
	plugin := s.ForComponent(WebConsole)

	// Nil Spec.Replicas defaults to 1
	plugin.CheckDeploymentProgress(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Spec:       appsv1.DeploymentSpec{},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
			Replicas:      1,
			Conditions: []appsv1.DeploymentCondition{{
				Type:   appsv1.DeploymentAvailable,
				Status: corev1.ConditionTrue,
			}},
		},
	})

	cs := plugin.Get()
	assert.Equal(t, StatusReady, cs.Status)
	require.NotNil(t, cs.DesiredReplicas)
	assert.Equal(t, int32(1), *cs.DesiredReplicas)
	assert.Equal(t, int32(1), *cs.ReadyReplicas)
}

func TestDeploymentNotAvailable(t *testing.T) {
	s := NewManager()
	plugin := s.ForComponent(WebConsole)

	plugin.CheckDeploymentProgress(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(2))},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
			Replicas:      2,
			Conditions: []appsv1.DeploymentCondition{{
				Type:    appsv1.DeploymentAvailable,
				Status:  corev1.ConditionFalse,
				Message: "minimum availability not met",
			}},
		},
	})

	cs := plugin.Get()
	assert.Equal(t, StatusInProgress, cs.Status)
	assert.Contains(t, cs.Message, "not ready: 1/2")
	require.NotNil(t, cs.DesiredReplicas)
	assert.Equal(t, int32(2), *cs.DesiredReplicas)
	assert.Equal(t, int32(1), *cs.ReadyReplicas)
}

func TestDeploymentNilSetsInProgress(t *testing.T) {
	s := NewManager()
	plugin := s.ForComponent(WebConsole)

	plugin.CheckDeploymentProgress(nil)

	cs := plugin.Get()
	assert.Equal(t, StatusInProgress, cs.Status)
	assert.Equal(t, "DeploymentNotCreated", cs.Reason)
}

func TestDaemonSetNilSetsInProgress(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	agent.CheckDaemonSetProgress(nil)

	cs := agent.Get()
	assert.Equal(t, StatusInProgress, cs.Status)
	assert.Equal(t, "DaemonSetNotCreated", cs.Reason)
}

func TestDeploymentMissingConditionFallback(t *testing.T) {
	s := NewManager()
	plugin := s.ForComponent(WebConsole)

	// Deployment with no Available condition but ready replicas match
	plugin.CheckDeploymentProgress(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 3,
			Replicas:      3,
		},
	})
	cs := plugin.Get()
	assert.Equal(t, StatusReady, cs.Status)

	// Now with mismatch
	plugin2 := s.ForComponent(WebConsole)
	plugin2.CheckDeploymentProgress(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "plugin"},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas: 1,
			Replicas:      3,
		},
	})
	cs = plugin2.Get()
	assert.Equal(t, StatusInProgress, cs.Status)
	assert.Contains(t, cs.Message, "missing condition")
}

func TestSetPodHealthDegradation(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	// Start with ready
	agent.SetReady()
	assert.Equal(t, StatusReady, agent.Get().Status)

	// Inject pod health issues — should degrade to StatusDegraded
	agent.setPodHealth(PodHealthSummary{
		UnhealthyCount: 2,
		Issues:         "2 CrashLoopBackOff (pod-a, pod-b): can't write messages into Kafka",
	})

	cs := agent.Get()
	assert.Equal(t, StatusDegraded, cs.Status)
	assert.Equal(t, "UnhealthyPods", cs.Reason)
	assert.Contains(t, cs.Message, "Kafka")
	assert.Equal(t, int32(2), cs.PodHealth.UnhealthyCount)
}

func TestSetPodHealthFromInProgress(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	agent.SetNotReady("DaemonSetNotReady", "DaemonSet not ready: 0/2")
	assert.Equal(t, StatusInProgress, agent.Get().Status)

	agent.setPodHealth(PodHealthSummary{
		UnhealthyCount: 2,
		Issues:         "2 CrashLoopBackOff (pod-a, pod-b): Error",
	})

	cs := agent.Get()
	assert.Equal(t, StatusDegraded, cs.Status, "InProgress + unhealthy pods should become Degraded")
	assert.Equal(t, "UnhealthyPods", cs.Reason)
	assert.Equal(t, int32(2), cs.PodHealth.UnhealthyCount)
}

func TestSetPodHealthNoDowngradeFromFailure(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)

	// Already in failure — pod health should not override to degraded
	agent.SetFailure("CriticalError", "crash")
	agent.setPodHealth(PodHealthSummary{
		UnhealthyCount: 1,
		Issues:         "1 CrashLoopBackOff (pod-x)",
	})

	cs := agent.Get()
	assert.Equal(t, StatusFailure, cs.Status, "setPodHealth should not override Failure with Degraded")
	assert.Equal(t, int32(1), cs.PodHealth.UnhealthyCount, "PodHealth should still be recorded")
}

func TestToCRDStatusWithPodHealth(t *testing.T) {
	cs := ComponentStatus{
		Name:            EBPFAgents,
		Status:          StatusDegraded,
		Reason:          "UnhealthyPods",
		Message:         "2 CrashLoopBackOff (pod-a, pod-b)",
		DesiredReplicas: ptr.To(int32(5)),
		ReadyReplicas:   ptr.To(int32(3)),
		PodHealth: PodHealthSummary{
			UnhealthyCount: 2,
			Issues:         "2 CrashLoopBackOff (pod-a, pod-b)",
		},
	}

	crd := cs.toCRDStatus()
	assert.Equal(t, "Degraded", crd.State)
	assert.Equal(t, "UnhealthyPods", crd.Reason)
	assert.Equal(t, int32(5), *crd.DesiredReplicas)
	assert.Equal(t, int32(3), *crd.ReadyReplicas)
	assert.Equal(t, int32(2), crd.UnhealthyPodCount)
	assert.Equal(t, "2 CrashLoopBackOff (pod-a, pod-b)", crd.PodIssues)
}

func TestPopulateComponentStatuses(t *testing.T) {
	s := NewManager()
	agent := s.ForComponent(EBPFAgents)
	plugin := s.ForComponent(WebConsole)
	monitoring := s.ForComponent(Monitoring)

	agent.CheckDaemonSetProgress(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "agent"},
		Status: appsv1.DaemonSetStatus{
			DesiredNumberScheduled: 3,
			UpdatedNumberScheduled: 3,
			NumberReady:            3,
		},
	})
	plugin.SetReady()
	monitoring.SetFailure("DashboardError", "dashboard CM missing")

	fc := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc, nil)

	require.NotNil(t, fc.Status.Components)
	require.NotNil(t, fc.Status.Components.Agent)
	assert.Equal(t, "Ready", fc.Status.Components.Agent.State)
	assert.Equal(t, ptr.To(int32(3)), fc.Status.Components.Agent.DesiredReplicas)
	assert.Equal(t, ptr.To(int32(3)), fc.Status.Components.Agent.ReadyReplicas)

	require.NotNil(t, fc.Status.Components.Plugin)
	assert.Equal(t, "Ready", fc.Status.Components.Plugin.State)

	require.NotNil(t, fc.Status.Integrations)
	require.NotNil(t, fc.Status.Integrations.Monitoring)
	assert.Equal(t, "Failure", fc.Status.Integrations.Monitoring.State)
	assert.Equal(t, "DashboardError", fc.Status.Integrations.Monitoring.Reason)
}

func TestPopulateProcessorAggregation(t *testing.T) {
	t.Run("monolith only", func(t *testing.T) {
		s := NewManager()
		parent := s.ForComponent(FLPParent)
		mono := s.ForComponent(FLPMonolith)
		_ = s.ForComponent(FLPTransformer) // registered but unused

		parent.SetReady()
		mono.CheckDeploymentProgress(&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "flp"},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr.To(int32(2))},
			Status: appsv1.DeploymentStatus{
				ReadyReplicas: 2,
				Replicas:      2,
				Conditions: []appsv1.DeploymentCondition{{
					Type:   appsv1.DeploymentAvailable,
					Status: corev1.ConditionTrue,
				}},
			},
		})

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Components)
		require.NotNil(t, fc.Status.Components.Processor)
		assert.Equal(t, "Ready", fc.Status.Components.Processor.State)
		require.NotNil(t, fc.Status.Components.Processor.DesiredReplicas)
		assert.Equal(t, int32(2), *fc.Status.Components.Processor.DesiredReplicas)
	})

	t.Run("transformer failure overrides parent ready", func(t *testing.T) {
		s := NewManager()
		parent := s.ForComponent(FLPParent)
		_ = s.ForComponent(FLPMonolith)
		transformer := s.ForComponent(FLPTransformer)

		parent.SetReady()
		transformer.SetFailure("KafkaConnectionError", "cannot connect to broker")

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Components)
		require.NotNil(t, fc.Status.Components.Processor)
		assert.Equal(t, "Failure", fc.Status.Components.Processor.State)
		assert.Equal(t, "KafkaConnectionError", fc.Status.Components.Processor.Reason)
	})

	t.Run("parent ready with unused sub-reconcilers", func(t *testing.T) {
		s := NewManager()
		parent := s.ForComponent(FLPParent)
		mono := s.ForComponent(FLPMonolith)
		trans := s.ForComponent(FLPTransformer)

		parent.SetReady()
		mono.SetUnused("direct mode")
		trans.SetUnused("direct mode")

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Components)
		require.NotNil(t, fc.Status.Components.Processor)
		assert.Equal(t, "Ready", fc.Status.Components.Processor.State)
	})

	t.Run("on hold parent unused with unused subs yields processor unused", func(t *testing.T) {
		s := NewManager()
		parent := s.ForComponent(FLPParent)
		mono := s.ForComponent(FLPMonolith)
		trans := s.ForComponent(FLPTransformer)

		parent.SetUnused("FlowCollector is on hold")
		mono.SetUnused("FlowCollector is on hold")
		trans.SetUnused("FlowCollector is on hold")

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Components)
		require.NotNil(t, fc.Status.Components.Processor)
		assert.Equal(t, "Unused", fc.Status.Components.Processor.State)
	})
}

func TestPopulateLokiStatus(t *testing.T) {
	t.Run("lokistack ready", func(t *testing.T) {
		s := NewManager()
		ls := s.ForComponent(LokiStack)
		ls.SetReady()

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Integrations)
		require.NotNil(t, fc.Status.Integrations.Loki)
		assert.Equal(t, "Ready", fc.Status.Integrations.Loki.State)
	})

	t.Run("demo loki failure overrides ready", func(t *testing.T) {
		s := NewManager()
		demo := s.ForComponent(DemoLoki)
		demo.SetFailure("DeployFailed", "cannot create PVC")

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Integrations)
		require.NotNil(t, fc.Status.Integrations.Loki)
		assert.Equal(t, "Failure", fc.Status.Integrations.Loki.State)
		assert.Equal(t, "DeployFailed", fc.Status.Integrations.Loki.Reason)
	})

	t.Run("loki unused", func(t *testing.T) {
		s := NewManager()
		ls := s.ForComponent(LokiStack)
		ls.SetUnused("Loki is disabled")

		fc := &flowslatest.FlowCollector{}
		s.populateComponentStatuses(fc, nil)

		require.NotNil(t, fc.Status.Integrations)
		require.NotNil(t, fc.Status.Integrations.Loki)
		assert.Equal(t, "Unused", fc.Status.Integrations.Loki.State)
	})
}

func TestControllerComponentsNotInCRDStatus(t *testing.T) {
	s := NewManager()
	ctrl := s.ForComponent(FlowCollectorController)
	static := s.ForComponent(StaticController)
	np := s.ForComponent(NetworkPolicy)

	ctrl.SetReady()
	static.SetReady()
	np.SetReady()

	fc := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc, nil)

	assert.Nil(t, fc.Status.Components.Agent)
	assert.Nil(t, fc.Status.Components.Plugin)
	assert.Nil(t, fc.Status.Components.Processor)
	assert.Nil(t, fc.Status.Integrations.Monitoring)
	assert.Nil(t, fc.Status.Integrations.Loki)
}

func TestExporterStatus(t *testing.T) {
	s := NewManager()
	s.SetExporterStatus("kafka-export-0", "Kafka", "Ready", "Configured", "")
	s.SetExporterStatus("ipfix-export-0", "IPFIX", "Failure", "ConnectionRefused", "cannot connect")

	fc := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc, nil)

	require.NotNil(t, fc.Status.Integrations)
	assert.Len(t, fc.Status.Integrations.Exporters, 2)
	found := map[string]bool{}
	for _, e := range fc.Status.Integrations.Exporters {
		found[e.Name] = true
		if e.Name == "kafka-export-0" {
			assert.Equal(t, "Ready", e.State)
			assert.Equal(t, "Kafka", e.Type)
		} else if e.Name == "ipfix-export-0" {
			assert.Equal(t, "Failure", e.State)
			assert.Equal(t, "cannot connect", e.Message)
		}
	}
	assert.True(t, found["kafka-export-0"])
	assert.True(t, found["ipfix-export-0"])

	s.ClearExporters()
	fc2 := &flowslatest.FlowCollector{}
	s.populateComponentStatuses(fc2, nil)
	assert.Empty(t, fc2.Status.Integrations.Exporters)
}

func TestKafkaCondition(t *testing.T) {
	t.Run("no kafka components returns nil", func(t *testing.T) {
		s := NewManager()
		agent := s.ForComponent(EBPFAgents)
		mono := s.ForComponent(FLPMonolith)
		agent.SetReady()
		mono.SetReady()

		assert.Nil(t, s.GetKafkaCondition())
	})

	t.Run("transformer placeholder unknown implies do not strip kafka ready from API", func(t *testing.T) {
		s := NewManager()
		tr := s.ForComponent(FLPTransformer)
		ts := s.getStatus(FLPTransformer)
		require.NotNil(t, ts)
		assert.Equal(t, StatusUnknown, ts.Status)
		assert.Nil(t, s.GetKafkaCondition())

		tr.SetUnused("direct mode")
		ts = s.getStatus(FLPTransformer)
		require.NotNil(t, ts)
		assert.NotEqual(t, StatusUnknown, ts.Status)
	})

	t.Run("healthy transformer returns KafkaReady=True", func(t *testing.T) {
		s := NewManager()
		transformer := s.ForComponent(FLPTransformer)
		transformer.SetReady()

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, "KafkaReady", cond.Type)
		assert.Equal(t, metav1.ConditionTrue, cond.Status)
		assert.Equal(t, "Ready", cond.Reason)
	})

	t.Run("failed transformer returns KafkaReady=False", func(t *testing.T) {
		s := NewManager()
		transformer := s.ForComponent(FLPTransformer)
		transformer.SetFailure("KafkaError", "broker unreachable")

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Equal(t, "KafkaIssue", cond.Reason)
		assert.Contains(t, cond.Message, "Transformer: broker unreachable")
	})

	t.Run("transformer with unhealthy pods", func(t *testing.T) {
		s := NewManager()
		transformer := s.ForComponent(FLPTransformer)
		transformer.SetReady()
		transformer.setPodHealth(PodHealthSummary{
			UnhealthyCount: 1,
			Issues:         "1 CrashLoopBackOff (flp-kafka-0)",
		})

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Contains(t, cond.Message, "Transformer pods:")
	})

	t.Run("agent with kafka-related pod issues", func(t *testing.T) {
		s := NewManager()
		transformer := s.ForComponent(FLPTransformer)
		transformer.SetReady()
		agent := s.ForComponent(EBPFAgents)
		agent.SetReady()
		agent.setPodHealth(PodHealthSummary{
			UnhealthyCount: 3,
			Issues:         "3 CrashLoopBackOff (agent-a, agent-b, agent-c): can't write Kafka messages",
		})

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Contains(t, cond.Message, "Agent pods:")
	})

	t.Run("agent with non-kafka issues does not trigger KafkaReady", func(t *testing.T) {
		s := NewManager()
		transformer := s.ForComponent(FLPTransformer)
		transformer.SetReady()
		agent := s.ForComponent(EBPFAgents)
		agent.SetReady()
		agent.setPodHealth(PodHealthSummary{
			UnhealthyCount: 1,
			Issues:         "1 OOMKilled (agent-x)",
		})

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionTrue, cond.Status, "Non-kafka agent issues should not affect KafkaReady")
	})

	t.Run("kafka exporter failure", func(t *testing.T) {
		s := NewManager()
		s.SetExporterStatus("kafka-export-0", "Kafka", string(StatusFailure), "BrokerDown", "connection refused")

		cond := s.GetKafkaCondition()
		require.NotNil(t, cond)
		assert.Equal(t, metav1.ConditionFalse, cond.Status)
		assert.Contains(t, cond.Message, "Exporter kafka-export-0: connection refused")
	})

	t.Run("IPFIX exporter failure does not trigger KafkaReady", func(t *testing.T) {
		s := NewManager()
		s.SetExporterStatus("ipfix-export-0", "IPFIX", string(StatusFailure), "Down", "timeout")

		assert.Nil(t, s.GetKafkaCondition())
	})
}

// fakeRecorder implements record.EventRecorder for testing event emission.
type fakeRecorder struct {
	events []fakeEvent
}

type fakeEvent struct {
	object    runtime.Object
	eventType string
	reason    string
	message   string
}

func (r *fakeRecorder) Event(object runtime.Object, eventtype, reason, message string) {
	r.events = append(r.events, fakeEvent{object, eventtype, reason, message})
}

func (r *fakeRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	r.events = append(r.events, fakeEvent{object, eventtype, reason, fmt.Sprintf(messageFmt, args...)})
}

func (r *fakeRecorder) AnnotatedEventf(object runtime.Object, _ map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	r.events = append(r.events, fakeEvent{object, eventtype, reason, fmt.Sprintf(messageFmt, args...)})
}

func TestEmitStateTransitionEvents(t *testing.T) {
	rec := &fakeRecorder{}
	s := NewManager()
	s.SetEventRecorder(rec)
	fc := &flowslatest.FlowCollector{}

	agent := s.ForComponent(EBPFAgents)

	// First call with ready — no previous state, should not emit
	agent.SetReady()
	s.emitStateTransitionEvents(ctx(), fc)
	assert.Empty(t, rec.events, "First call should not emit events (no previous state)")

	// Transition to failure
	agent.SetFailure("KafkaError", "broker down")
	s.emitStateTransitionEvents(ctx(), fc)
	require.Len(t, rec.events, 1)
	assert.Equal(t, "Warning", rec.events[0].eventType)
	assert.Equal(t, "ComponentFailure", rec.events[0].reason)
	assert.Contains(t, rec.events[0].message, "EBPFAgents")
	assert.Contains(t, rec.events[0].message, "broker down")

	// Transition to ready (recovery)
	rec.events = nil
	agent.SetReady()
	s.emitStateTransitionEvents(ctx(), fc)
	require.Len(t, rec.events, 1)
	assert.Equal(t, "Normal", rec.events[0].eventType)
	assert.Equal(t, "ComponentRecovered", rec.events[0].reason)

	// Same state again — no event
	rec.events = nil
	agent.SetReady()
	s.emitStateTransitionEvents(ctx(), fc)
	assert.Empty(t, rec.events, "Same state should not emit events")

	// Transition to degraded
	rec.events = nil
	agent.SetDegraded("HighRestarts", "5 pods restarting")
	s.emitStateTransitionEvents(ctx(), fc)
	require.Len(t, rec.events, 1)
	assert.Equal(t, "Warning", rec.events[0].eventType)
	assert.Equal(t, "ComponentDegraded", rec.events[0].reason)
}

func TestEmitEventsNilRecorder(_ *testing.T) {
	s := NewManager()
	fc := &flowslatest.FlowCollector{}

	agent := s.ForComponent(EBPFAgents)
	agent.SetFailure("Error", "bad")
	s.emitStateTransitionEvents(ctx(), fc)
}

func TestConditionPolarity(t *testing.T) {
	tests := []struct {
		status    Status
		expected  metav1.ConditionStatus
		defReason string
	}{
		{StatusReady, metav1.ConditionFalse, "Ready"},
		{StatusFailure, metav1.ConditionTrue, "NotReady"},
		{StatusInProgress, metav1.ConditionTrue, "NotReady"},
		{StatusDegraded, metav1.ConditionTrue, "NotReady"},
		{StatusUnknown, metav1.ConditionUnknown, "Unknown"},
		{StatusUnused, metav1.ConditionUnknown, "Unused"},
	}
	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			cs := ComponentStatus{Name: EBPFAgents, Status: tc.status}
			cond := cs.toCondition()
			assert.Equal(t, tc.expected, cond.Status)
			assert.Equal(t, tc.defReason, cond.Reason)
		})
	}

	// Custom reason overrides default
	cs := ComponentStatus{Name: EBPFAgents, Status: StatusFailure, Reason: "KafkaError"}
	cond := cs.toCondition()
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, "KafkaError", cond.Reason)
}

func TestGlobalConditionCounts(t *testing.T) {
	s := NewManager()
	a := s.ForComponent(EBPFAgents)
	b := s.ForComponent(WebConsole)
	c := s.ForComponent(Monitoring)

	a.SetReady()
	b.SetReady()
	c.SetReady()

	conds := s.getConditions()
	assertHasCondition(t, conds, "Ready", "Ready", metav1.ConditionTrue)
	readyCond := findCondition(conds, "Ready")
	require.NotNil(t, readyCond)
	assert.Contains(t, readyCond.Message, "3 ready components")
	assert.Contains(t, readyCond.Message, "0 with failure")
}

func TestGlobalConditionUnusedNotCounted(t *testing.T) {
	s := NewManager()
	a := s.ForComponent(EBPFAgents)
	b := s.ForComponent(Monitoring)

	a.SetReady()
	b.SetUnused("disabled")

	conds := s.getConditions()
	readyCond := findCondition(conds, "Ready")
	require.NotNil(t, readyCond)
	assert.Equal(t, metav1.ConditionTrue, readyCond.Status)
	assert.Equal(t, "Ready", readyCond.Reason)
	assert.Contains(t, readyCond.Message, "1 ready components")
	assert.Contains(t, readyCond.Message, "0 with failure")
}

func TestNeedsRequeue(t *testing.T) {
	t.Run("all ready returns false", func(t *testing.T) {
		s := NewManager()
		agent := s.ForComponent(EBPFAgents)
		plugin := s.ForComponent(WebConsole)
		agent.SetReady()
		plugin.SetReady()
		assert.False(t, s.NeedsRequeue())
	})

	t.Run("in-progress component returns true", func(t *testing.T) {
		s := NewManager()
		agent := s.ForComponent(EBPFAgents)
		plugin := s.ForComponent(WebConsole)
		agent.SetReady()
		plugin.SetNotReady("Deploying", "rolling out")
		assert.True(t, s.NeedsRequeue())
	})

	t.Run("unhealthy pods returns true", func(t *testing.T) {
		s := NewManager()
		agent := s.ForComponent(EBPFAgents)
		agent.SetReady()
		agent.setPodHealth(PodHealthSummary{
			UnhealthyCount: 2,
			Issues:         "2 CrashLoopBackOff (pod-a, pod-b)",
		})
		assert.True(t, s.NeedsRequeue())
	})

	t.Run("unused and unknown do not trigger", func(t *testing.T) {
		s := NewManager()
		a := s.ForComponent(EBPFAgents)
		b := s.ForComponent(WebConsole)
		a.SetUnused("disabled")
		b.SetUnknown()
		assert.False(t, s.NeedsRequeue())
	})

	t.Run("failure without pod issues does not trigger", func(t *testing.T) {
		s := NewManager()
		agent := s.ForComponent(EBPFAgents)
		agent.SetFailure("ConfigError", "bad config")
		assert.False(t, s.NeedsRequeue())
	})
}

func ctx() context.Context {
	return context.Background()
}

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}

func assertHasCondition(t *testing.T, conditions []metav1.Condition, searchType, reason string, value metav1.ConditionStatus) {
	for _, c := range conditions {
		if c.Type == searchType {
			assert.Equal(t, reason, c.Reason, conditions)
			assert.Equal(t, value, c.Status, conditions)
			return
		}
	}
	assert.Fail(t, "Condition type not found", searchType, conditions)
}

func assertHasConditionTypes(t *testing.T, conditions []metav1.Condition, expectedTypes []string) {
	var types []string
	for _, c := range conditions {
		types = append(types, c.Type)
	}
	slices.Sort(types)
	assert.Equal(t, expectedTypes, types)
}
