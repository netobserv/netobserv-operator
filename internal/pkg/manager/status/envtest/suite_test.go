//nolint:revive
package envtest

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllers "github.com/netobserv/netobserv-operator/internal/controller"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

var (
	ctx          context.Context
	k8sClient    client.Client
	suiteContext *test.SuiteContext
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FlowCollector Status Test Suite")
}

// go test ./... runs always Ginkgo test suites in parallel and they would interfere
// this way we make sure that both test sub-suites are executed serially
var _ = Describe("FlowCollector Status", Ordered, Serial, func() {
	flowCollectorStatusSpecs()
})

var _ = BeforeSuite(func() {
	ctx, k8sClient, suiteContext = test.PrepareEnvTest(
		test.EnvVanillaNaked,
		controllers.Registerers,
		"main-namespace",
		nil,
	)
})

var _ = AfterSuite(func() {
	test.TeardownEnvTest(suiteContext)
})
