//nolint:revive
package vanillafs

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netobserv/netobserv-operator/internal/controller/flp"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/envtest"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

const (
	env = test.EnvVanillaFullStack
)

var (
	ctx          context.Context
	k8sClient    client.Client
	suiteContext *test.SuiteContext
)

func TestAPIsVanillaFullStack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FLP Controller Suite - Vanilla Full Stack")
}

// go test ./... runs always Ginkgo test suites in parallel and they would interfere
// this way we make sure that both test sub-suites are executed serially
var _ = Describe("FLP FlowCollector Controller - Vanilla Full Stack", Ordered, Serial, func() {
	ctxGetter := func() (context.Context, client.Client) { return ctx, k8sClient }
	envtest.ControllerSpecs(env, ctxGetter)
	envtest.ControllerFlowMetricsSpecs(ctxGetter)
})

var _ = BeforeSuite(func() {
	ctx, k8sClient, suiteContext = test.PrepareEnvTest(
		env,
		[]manager.Registerer{flp.Start},
		"main-namespace",
		[]string{"other-namespace"},
	)
})

var _ = AfterSuite(func() {
	test.TeardownEnvTest(suiteContext)
})
