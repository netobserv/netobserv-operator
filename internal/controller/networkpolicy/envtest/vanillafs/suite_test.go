//nolint:revive
package vanillafs

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netobserv/netobserv-operator/internal/controller/networkpolicy"
	"github.com/netobserv/netobserv-operator/internal/controller/networkpolicy/envtest"
	"github.com/netobserv/netobserv-operator/internal/controller/static"
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
	RunSpecs(t, "Networkpolicy Controller Suite - Vanilla Full Stack")
}

// go test ./... runs always Ginkgo test suites in parallel and they would interfere
// this way we make sure that both test sub-suites are executed serially
var _ = Describe("Networkpolicy Controller - Vanilla Full Stack", Ordered, Serial, func() {
	ctxGetter := func() (context.Context, client.Client) { return ctx, k8sClient }
	envtest.ControllerSpecs(env, ctxGetter)
})

var _ = BeforeSuite(func() {
	ctx, k8sClient, suiteContext = test.PrepareEnvTest(
		env,
		[]manager.Registerer{static.Start, networkpolicy.Start},
		"main-namespace",
		[]string{"other-namespace", "main-namespace-privileged"},
	)
})

var _ = AfterSuite(func() {
	test.TeardownEnvTest(suiteContext)
})
