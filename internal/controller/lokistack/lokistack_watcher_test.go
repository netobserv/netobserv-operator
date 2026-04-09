package lokistack

import (
	"context"
	"strings"
	"testing"

	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
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

func initLSWatcher() (*mockClient, *Watcher) {
	clust := &cluster.Info{}
	clust.Mock("", "", cluster.LokiStack)
	client := &mockClient{}
	lsw := Watcher{
		mgr:                &manager.Manager{ClusterInfo: clust},
		cl:                 client,
		status:             status.NewManager().ForComponent(status.LokiStack),
		lokiWatcherStarted: true,
	}
	return client, &lsw
}

func TestCheckLoki_Disabled(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr.To(false),
			},
		},
	}

	_, lsw := initLSWatcher()
	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusUnknown,
		Reason:  "ComponentUnused",
		Message: "Loki is disabled",
	}, st)
}

func TestCheckLoki_NotLokiStackMode(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeManual,
			},
		},
	}

	_, lsw := initLSWatcher()
	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusUnknown,
		Reason:  "ComponentUnused",
		Message: "Loki is not configured in LokiStack mode",
	}, st)
}

func TestCheckLoki_LokiStackNotFound(t *testing.T) {
	fc := &flowslatest.FlowCollector{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: flowslatest.FlowCollectorSpec{
			Namespace: "netobserv",
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).
		Return(kerr.NewNotFound(schema.GroupResource{}, "loki"))

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusFailure,
		Reason:  "CantFetchLokiStack",
		Message: ` "loki" not found`,
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusInProgress,
		Reason:  "LokiStackNotReady",
		Message: `LokiStack is not ready [name: loki, namespace: netobserv]: PendingComponents - Some components are still starting`,
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusFailure,
		Reason:  "LokiStackIssues",
		Message: `LokiStack has issues [name: loki, namespace: netobserv]: StorageError: Cannot connect to S3 backend`,
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.StatusFailure, st.Status)
	assert.Equal(t, "LokiStackIssues", st.Reason)
	assert.Contains(t, st.Message, "Missing object storage secret")
	assert.Contains(t, st.Message, "The schema configuration does not contain the most recent schema version and needs an update")
}

func TestCheckLoki_LokiStackWithJustWarnings(t *testing.T) {
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusDegraded,
		Reason:  "LokiStackWarnings",
		Message: `LokiStack has warnings [name: loki, namespace: netobserv]: Warning: The schema configuration does not contain the most recent schema version`,
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusInProgress,
		Reason:  "LokiStackComponentIssues",
		Message: `LokiStack components have issues [name: loki, namespace: netobserv]: Ingester has 2 failed pod(s): ingester-0, ingester-1; Querier has 1 pending pod(s): querier-0`,
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name: "loki",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "loki", Namespace: "netobserv"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusReady,
		Reason:  "",
		Message: "",
	}, st)
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
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name:      "custom-loki",
					Namespace: "observability",
				},
			},
		},
	}

	client, lsw := initLSWatcher()
	nsname := types.NamespacedName{Name: "custom-loki", Namespace: "observability"}
	client.On("Get", mock.Anything, nsname, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*lokiv1.LokiStack)
		*arg = *lokiStack
	}).Return(nil)

	st := lsw.Reconcile(context.Background(), fc)

	assert.Equal(t, status.ComponentStatus{
		Name:    status.LokiStack,
		Status:  status.StatusReady,
		Reason:  "",
		Message: "",
	}, st)
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
	issuesStr := strings.Join(issues, "; ")
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
