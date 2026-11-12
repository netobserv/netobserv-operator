//nolint:revive,staticcheck
package envtest

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

// nolint:cyclop
func FlowCollectorMinimalSpecs(ctxGetter test.ContextGetter) {
	var ctx context.Context
	var k8sClient client.Client
	BeforeEach(func() {
		ctx, k8sClient = ctxGetter()
	})

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

		It("Should install components successfully with default resources", func() {
			By("Expecting to create the agent DaemonSet")
			agent := appsv1.DaemonSet{}
			Eventually(func() error { return k8sClient.Get(ctx, agentKey, &agent) }, timeout, interval).
				Should(Succeed())
			Expect(agent.Spec.Template.Spec.Containers[0].Resources).
				To(Equal(flowslatest.DefaultFlowCollector.Agent.EBPF.Resources))

			By("Expecting to create the flowlogs-pipeline Deployment")
			flp := appsv1.Deployment{}
			Eventually(func() error { return k8sClient.Get(ctx, flpKey, &flp) }, timeout, interval).
				Should(Succeed())
			Expect(flp.Spec.Template.Spec.Containers[0].Resources).
				To(Equal(flowslatest.DefaultFlowCollector.Processor.Resources))

			By("Expecting to create the console plugin Deployment")
			console := appsv1.Deployment{}
			Eventually(func() error { return k8sClient.Get(ctx, cpKey, &console) }, timeout, interval).
				Should(Succeed())
			Expect(console.Spec.Template.Spec.Containers[0].Resources).
				To(Equal(flowslatest.DefaultFlowCollector.ConsolePlugin.Resources))

			By("Expecting to create the console plugin Service")
			Eventually(func() error { return k8sClient.Get(ctx, cpKey, &corev1.Service{}) }, timeout, interval).
				Should(Succeed())
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			test.CleanupCR(ctx, k8sClient, crKey)
		})
	})
}
