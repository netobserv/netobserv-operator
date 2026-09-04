//nolint:revive
package vanilla

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllers "github.com/netobserv/netobserv-operator/internal/controller"
	"github.com/netobserv/netobserv-operator/internal/controller/envtest"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

const (
	env = test.EnvVanillaNaked
)

var (
	ctx          context.Context
	k8sClient    client.Client
	suiteContext *test.SuiteContext
)

func TestAPIsVanilla(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite - Vanilla")
}

// go test ./... runs always Ginkgo test suites in parallel and they would interfere
// this way we make sure that both test sub-suites are executed serially
var _ = Describe("FlowCollector Controller - Vanilla", Ordered, Serial, func() {
	ctxGetter := func() (context.Context, client.Client) { return ctx, k8sClient }
	envtest.FlowCollectorConsolePluginSpecs(env, ctxGetter)
	envtest.FlowCollectorEBPFSpecs(env, ctxGetter)
	envtest.FlowCollectorEBPFKafkaSpecs(ctxGetter)
	envtest.FlowCollectorMinimalSpecs(ctxGetter)
	envtest.FlowCollectorHoldModeSpecs(ctxGetter)
})

var _ = BeforeSuite(func() {
	ctx, k8sClient, suiteContext = test.PrepareEnvTest(
		env,
		controllers.Registerers,
		"main-namespace",
		[]string{
			"loki-namespace",
			"kafka-exporter-namespace",
			"main-namespace-privileged",
		},
	)
})

var _ = AfterSuite(func() {
	test.TeardownEnvTest(suiteContext)
})
