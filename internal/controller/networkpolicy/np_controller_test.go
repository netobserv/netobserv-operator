//nolint:revive
package networkpolicy

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	. "github.com/netobserv/netobserv-operator/internal/controller/controllerstest"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

const (
	timeout  = test.Timeout
	interval = test.Interval
)

var (
	getCR = func(key types.NamespacedName) *flowslatest.FlowCollector {
		return test.GetCR(ctx, k8sClient, key)
	}
	cleanupCR = func(key types.NamespacedName) {
		test.CleanupCR(ctx, k8sClient, key)
	}
)

// nolint:cyclop
func ControllerSpecs() {
	const operatorNamespace = "main-namespace"
	crKey := types.NamespacedName{
		Name: "cluster",
	}
	npKey1 := types.NamespacedName{
		Name:      netpolName,
		Namespace: operatorNamespace,
	}

	// Created objects to cleanup
	cleanupList := []client.Object{}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Deploying as DaemonSet", func() {
		np1 := networkingv1.NetworkPolicy{}
		It("Should create successfully", func() {
			created := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
					Processor: flowslatest.FlowCollectorFLP{
						ImagePullPolicy: "Never",
						LogLevel:        "error",
						LogTypes:        &outputRecordTypes,
						Metrics: flowslatest.FLPMetrics{
							IncludeList: &[]flowslatest.FLPMetric{"node_ingress_bytes_total", "namespace_ingress_bytes_total", "workload_ingress_bytes_total"},
						},
					},
					NetworkPolicy: flowslatest.NetworkPolicy{
						Enable: ptr.To(true),
					},
				},
			}

			// Create
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			By("Expecting to create the netobserv NetworkPolicy")
			Eventually(func() error {
				return k8sClient.Get(ctx, npKey1, &np1)
			}, timeout, interval).Should(Succeed())

		})

	})

	Context("Checking CR ownership", func() {
		It("Should be garbage collected", func() {
			// Retrieve CR to get its UID
			By("Getting the CR")
			flowCR := getCR(crKey)

			By("Expecting flowlogs-pipeline daemonset to be garbage collected")
			Eventually(func() interface{} {
				np := networkingv1.NetworkPolicy{}
				_ = k8sClient.Get(ctx, npKey1, &np)
				return &np
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)
		})

		It("Should cleanup other data", func() {
			for _, obj := range cleanupList {
				Eventually(func() error {
					return k8sClient.Delete(ctx, obj)
				}, timeout, interval).Should(Succeed())
			}
		})
	})

}
