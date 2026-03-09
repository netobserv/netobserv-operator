package status

import (
	"context"
	"strings"
	"testing"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Simple mock client for testing
type mockClient struct {
	mock.Mock
	client.Client
}

func (m *mockClient) Get(ctx context.Context, key types.NamespacedName, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func TestCheckLoki_Disabled(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(false),
			},
		},
	}

	client := &mockClient{}
	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "Unused", condition.Reason)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Contains(t, condition.Message, "Loki is disabled")
}

func TestCheckLoki_NotLokiStackMode(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeManual,
			},
		},
	}

	client := &mockClient{}
	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "Unused", condition.Reason)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
	assert.Contains(t, condition.Message, "not configured in LokiStack mode")
}

func TestCheckLoki_LokiStackNotFound(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).
		Return(kerr.NewNotFound(schema.GroupResource{}, "loki"))

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "LokiStackNotFound", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "could not be found")
	assert.Contains(t, condition.Message, "loki")
	assert.Contains(t, condition.Message, "netobserv")
}

func TestCheckLoki_LokiStackNotReady(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionFalse,
					Reason:  "PendingComponents",
					Message: "Some components are still starting",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "LokiStackNotReady", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "not ready")
	assert.Contains(t, condition.Message, "PendingComponents")
	assert.Contains(t, condition.Message, "Some components are still starting")
}

func TestCheckLoki_LokiStackWithErrorCondition(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
				{
					Type:    "StorageError",
					Status:  metav1.ConditionTrue,
					Reason:  "S3Unavailable",
					Message: "Cannot connect to S3 backend",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "LokiStackIssues", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "StorageError")
	assert.Contains(t, condition.Message, "Cannot connect to S3 backend")
}

func TestCheckLoki_LokiStackWithWarningAndDegradedConditions(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Warning",
					Status:  metav1.ConditionTrue,
					Reason:  "StorageNeedsSchemaUpdate",
					Message: "The schema configuration does not contain the most recent schema version and needs an update",
				},
				{
					Type:    "Degraded",
					Status:  metav1.ConditionTrue,
					Reason:  "MissingObjectStorageSecret",
					Message: "Missing object storage secret",
				},
				{
					Type:    "Ready",
					Status:  metav1.ConditionFalse,
					Reason:  "ReadyComponents",
					Message: "All components ready",
				},
				{
					Type:    "Pending",
					Status:  metav1.ConditionFalse,
					Reason:  "PendingComponents",
					Message: "One or more LokiStack components pending on dependencies",
				},
				{
					Type:    "Degraded",
					Status:  metav1.ConditionFalse,
					Reason:  "MissingTokenCCOAuthenticationSecret",
					Message: "Missing OpenShift cloud credentials secret",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	// Check that Degraded issue is reported in LokiIssue
	condition := checkLoki(context.Background(), client, fc)
	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "LokiStackIssues", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "Degraded")
	assert.Contains(t, condition.Message, "Missing object storage secret")
	// Warning should NOT be in LokiIssue
	assert.NotContains(t, condition.Message, "Warning")
	assert.NotContains(t, condition.Message, "schema configuration")

	// Check that Warning is reported separately in LokiWarning
	warningCondition := checkLokiWarnings(context.Background(), client, fc)
	assert.Equal(t, LokiWarning, warningCondition.Type)
	assert.Equal(t, "LokiStackWarnings", warningCondition.Reason)
	assert.Equal(t, metav1.ConditionTrue, warningCondition.Status)
	assert.Contains(t, warningCondition.Message, "Warning")
	assert.Contains(t, warningCondition.Message, "The schema configuration does not contain the most recent schema version")
}

func TestCheckLokiWarnings_Disabled(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(false),
			},
		},
	}

	client := &mockClient{}
	condition := checkLokiWarnings(context.Background(), client, fc)

	assert.Equal(t, LokiWarning, condition.Type)
	assert.Equal(t, "Unused", condition.Reason)
	assert.Equal(t, metav1.ConditionUnknown, condition.Status)
}

func TestCheckLokiWarnings_NoWarnings(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLokiWarnings(context.Background(), client, fc)

	assert.Equal(t, LokiWarning, condition.Type)
	assert.Equal(t, "NoWarning", condition.Reason)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
}

func TestCheckLokiWarnings_WithWarning(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
				{
					Type:    "Warning",
					Status:  metav1.ConditionTrue,
					Reason:  "StorageNeedsSchemaUpdate",
					Message: "The schema configuration does not contain the most recent schema version",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLokiWarnings(context.Background(), client, fc)

	assert.Equal(t, LokiWarning, condition.Type)
	assert.Equal(t, "LokiStackWarnings", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "Warning")
	assert.Contains(t, condition.Message, "schema configuration")
}

func TestCheckLoki_LokiStackComponentsWithFailedPods(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
			},
			Components: lokiv1.LokiStackComponentStatus{
				Ingester: lokiv1.PodStatusMap{
					lokiv1.PodFailed: []string{"ingester-0", "ingester-1"},
				},
				Querier: lokiv1.PodStatusMap{
					lokiv1.PodPending: []string{"querier-0"},
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "LokiStackComponentIssues", condition.Reason)
	assert.Equal(t, metav1.ConditionTrue, condition.Status)
	assert.Contains(t, condition.Message, "Ingester")
	assert.Contains(t, condition.Message, "2 failed pod(s)")
	assert.Contains(t, condition.Message, "ingester-0")
	assert.Contains(t, condition.Message, "Querier")
	assert.Contains(t, condition.Message, "1 pending pod(s)")
	assert.Contains(t, condition.Message, "querier-0")
}

func TestCheckLoki_LokiStackHealthy(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki",
			Namespace: "netobserv",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
			},
			Components: lokiv1.LokiStackComponentStatus{
				Ingester: lokiv1.PodStatusMap{
					lokiv1.PodRunning: []string{"ingester-0", "ingester-1"},
				},
				Querier: lokiv1.PodStatusMap{
					lokiv1.PodRunning: []string{"querier-0", "querier-1"},
				},
				Distributor: lokiv1.PodStatusMap{
					lokiv1.PodRunning: []string{"distributor-0"},
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "NoIssue", condition.Reason)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
}

func TestCheckLoki_CustomNamespace(t *testing.T) {
	lokiStack := &lokiv1.LokiStack{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-loki",
			Namespace: "observability",
		},
		Status: lokiv1.LokiStackStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionTrue,
					Reason:  "Ready",
					Message: "All components ready",
				},
			},
		},
	}

	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name:      "custom-loki",
					Namespace: "observability",
				},
			},
		},
	}

	client := &mockClient{}
	nsname := types.NamespacedName{Name: "custom-loki", Namespace: "observability"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	condition := checkLoki(context.Background(), client, fc)

	assert.Equal(t, LokiIssue, condition.Type)
	assert.Equal(t, "NoIssue", condition.Reason)
	assert.Equal(t, metav1.ConditionFalse, condition.Status)
}

func TestCheckLokiStackComponents_AllComponentTypes(t *testing.T) {
	components := &lokiv1.LokiStackComponentStatus{
		Compactor: lokiv1.PodStatusMap{
			lokiv1.PodFailed: []string{"compactor-0"},
		},
		Distributor: lokiv1.PodStatusMap{
			lokiv1.PodPending: []string{"distributor-0"},
		},
		IndexGateway: lokiv1.PodStatusMap{
			lokiv1.PodStatusUnknown: []string{"index-gateway-0"},
		},
		Ingester: lokiv1.PodStatusMap{
			lokiv1.PodFailed: []string{"ingester-0", "ingester-1"},
		},
		Querier: lokiv1.PodStatusMap{
			lokiv1.PodRunning: []string{"querier-0"},
		},
		QueryFrontend: lokiv1.PodStatusMap{
			lokiv1.PodPending: []string{"query-frontend-0"},
		},
		Gateway: lokiv1.PodStatusMap{
			lokiv1.PodRunning: []string{"gateway-0"},
		},
		Ruler: lokiv1.PodStatusMap{
			lokiv1.PodFailed: []string{"ruler-0"},
		},
	}

	issues := checkLokiStackComponents(components)

	assert.Len(t, issues, 6) // Should report 6 issues (failed, pending, and unknown pods)

	// Check that all problematic components are reported
	issuesStr := joinIssues(issues)
	assert.Contains(t, issuesStr, "Compactor has 1 failed pod(s): compactor-0")
	assert.Contains(t, issuesStr, "Distributor has 1 pending pod(s): distributor-0")
	assert.Contains(t, issuesStr, "IndexGateway has 1 pod(s) with unknown status: index-gateway-0")
	assert.Contains(t, issuesStr, "Ingester has 2 failed pod(s): ingester-0, ingester-1")
	assert.Contains(t, issuesStr, "QueryFrontend has 1 pending pod(s): query-frontend-0")
	assert.Contains(t, issuesStr, "Ruler has 1 failed pod(s): ruler-0")

	// Check that healthy components (Querier and Gateway with only running pods) are not reported
	hasQuerier := false
	hasGatewayIssue := false
	for _, issue := range issues {
		if strings.Contains(issue, "Querier") {
			hasQuerier = true
		}
		// Check for "Gateway has" but make sure it's not "IndexGateway has"
		if strings.Contains(issue, "Gateway has") && !strings.Contains(issue, "IndexGateway has") {
			hasGatewayIssue = true
		}
	}
	assert.False(t, hasQuerier, "Querier should not be in issues")
	assert.False(t, hasGatewayIssue, "Gateway should not be in issues")
}

func TestCheckLokiStackComponents_NilComponents(t *testing.T) {
	issues := checkLokiStackComponents(nil)
	assert.Nil(t, issues)
}

func TestCheckLokiStackComponents_EmptyComponents(t *testing.T) {
	components := &lokiv1.LokiStackComponentStatus{}
	issues := checkLokiStackComponents(components)
	assert.Empty(t, issues)
}

func TestCheckLokiStackComponents_OnlyRunningPods(t *testing.T) {
	components := &lokiv1.LokiStackComponentStatus{
		Ingester: lokiv1.PodStatusMap{
			lokiv1.PodRunning: []string{"ingester-0", "ingester-1"},
		},
		Querier: lokiv1.PodStatusMap{
			lokiv1.PodRunning: []string{"querier-0"},
		},
	}

	issues := checkLokiStackComponents(components)
	assert.Empty(t, issues)
}

// Helper functions

func ptr[T any](v T) *T {
	return &v
}

func joinIssues(issues []string) string {
	result := ""
	for _, issue := range issues {
		result += issue + " "
	}
	return result
}
