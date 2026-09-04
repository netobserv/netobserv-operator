package consoleplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

// TestBuiltObjectsConvergeAgainstAPIServer guards against the whole "reconcile never converges" bug
// family, at the level of the objects the operator builds (Deployment, Service, ...). It is the
// object-level counterpart of the CR-level isomorphism test in flowcollector_controller_iso_envtest.go.
//
// The pitfall: the builder leaves a field empty (e.g. the static plugin leaves ImagePullPolicy unset,
// because its faked FlowCollector never goes through OpenAPI defaulting). The API server then defaults
// the *stored* object (ImagePullPolicy -> IfNotPresent). If the comparator exact-compares such a field,
// it reports a phantom change on every reconcile and issues a redundant Update forever - which, once the
// object is also watched, turns into an update/conflict storm.
//
// This test builds objects exactly as the reconciler does, round-trips them through a real API server so
// they get the same defaulting, then asserts the operator's own comparator sees no change. It is a
// catch-all: any comparator-inspected field that the builder leaves for the API server to default (now or
// in the future) will fail here, not only ImagePullPolicy.
func TestBuiltObjectsConvergeAgainstAPIServer(t *testing.T) {
	assert := assert.New(t)

	err := test.SetupKubeBuilderAssets()
	require.NoError(t, err)

	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	defer func() { _ = testEnv.Stop() }()

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	require.NoError(t, err)

	// Load the CRD so the builder resolves the same OpenAPI defaults (e.g. the plugin port) it would in
	// production, instead of zero values.
	require.NoError(t, helper.SetCRDForTests(test.RepoRoot()))

	ctx := context.Background()
	require.NoError(t, k8sClient.Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}))

	// Build objects with minimal values to trigger API server defaults as much as possible.
	spec := flowslatest.FlowCollectorSpec{
		Namespace: testNamespace,
		ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
			Enable: ptr.To(true),
		},
	}
	loki := helper.LokiConfig{
		LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://loki:3100/", TenantID: "netobserv"},
	}
	builder := getBuilder(&spec, &loki)

	// Deployment
	{
		require.NoError(t, k8sClient.Create(ctx, builder.deployment(constants.PluginName, "digest")))
		stored := appsv1.Deployment{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Name: constants.PluginName, Namespace: testNamespace}, &stored))

		desired := builder.deployment(constants.PluginName, "digest") // fresh, un-mutated by the API server
		report := helper.NewChangeReport("")
		assert.False(
			helper.DeploymentChanged(&stored, desired, constants.PluginName, &report),
			"Deployment does not converge against API-server defaults; reconcile would loop. Report: %s", report.String(),
		)
	}

	// Service
	{
		require.NoError(t, k8sClient.Create(ctx, builder.mainService(constants.PluginName)))
		stored := corev1.Service{}
		require.NoError(t, k8sClient.Get(ctx, client.ObjectKey{Name: constants.PluginName, Namespace: testNamespace}, &stored))

		desired := builder.mainService(constants.PluginName)
		report := helper.NewChangeReport("")
		assert.False(
			helper.ServiceChanged(&stored, desired, &report),
			"Service does not converge against API-server defaults; reconcile would loop. Report: %s", report.String(),
		)
	}
}
