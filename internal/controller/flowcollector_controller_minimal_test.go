//nolint:revive
package controllers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
)

// nolint:cyclop
func flowCollectorMinimalSpecs() {
	crKey := types.NamespacedName{
		Name: "cluster",
	}
	agentKey := types.NamespacedName{
		Name:      "netobserv-ebpf-agent",
		Namespace: "netobserv-privileged",
	}
	flpKey := types.NamespacedName{
		Name:      constants.FLPName,
		Namespace: "netobserv",
	}
	cpKey := types.NamespacedName{
		Name:      "netobserv-plugin",
		Namespace: "netobserv",
	}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Minimal FlowCollector (empty spec)", func() {
		It("Should create CR successfully", func() {
			Eventually(func() interface{} {
				return k8sClient.Create(ctx, &flowslatest.FlowCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: crKey.Name,
					},
				})
			}, timeout, interval).Should(Succeed())
		})

		It("Should install components successfully", func() {
			By("Expecting to create the agent DaemonSet")
			Eventually(func() error {
				ds := appsv1.DaemonSet{}
				return k8sClient.Get(ctx, agentKey, &ds)
			}, timeout, interval).Should(Succeed())

			By("Expecting to create the flowlogs-pipeline Deployment")
			Eventually(func() error {
				depl := appsv1.Deployment{}
				return k8sClient.Get(ctx, flpKey, &depl)
			}, timeout, interval).Should(Succeed())

			By("Expecting to create the console plugin Deployment")
			Eventually(func() interface{} {
				dp := appsv1.Deployment{}
				return k8sClient.Get(ctx, cpKey, &dp)
			}, timeout, interval).Should(Succeed())

			By("Expecting to create the console plugin Service")
			Eventually(func() interface{} {
				svc := corev1.Service{}
				return k8sClient.Get(ctx, cpKey, &svc)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)
		})
	})
}
