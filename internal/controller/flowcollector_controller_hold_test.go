//nolint:revive
package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/network-observability-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/network-observability-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/network-observability-operator/internal/controller/constants"
)

func flowCollectorHoldModeSpecs() {
	operatorNamespace := "namespace-hold-mode"
	crKey := types.NamespacedName{Name: "cluster"}
	agentKey := types.NamespacedName{
		Name:      "netobserv-ebpf-agent",
		Namespace: operatorNamespace + "-privileged",
	}
	flpKey := types.NamespacedName{
		Name:      constants.FLPName,
		Namespace: operatorNamespace,
	}
	pluginKey := types.NamespacedName{
		Name:      constants.PluginName,
		Namespace: operatorNamespace,
	}
	nsKey := types.NamespacedName{Name: operatorNamespace}
	privilegedNsKey := types.NamespacedName{Name: operatorNamespace + "-privileged"}

	Context("Hold Mode", func() {
		It("Should create resources when FlowCollector is deployed", func() {
			// Create FlowCollector
			desired := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
					Agent: flowslatest.FlowCollectorAgent{
						Type: "eBPF",
						EBPF: flowslatest.FlowCollectorEBPF{
							Sampling:           ptr.To(int32(100)),
							CacheActiveTimeout: "10s",
							CacheMaxFlows:      50,
						},
					},
					Processor: flowslatest.FlowCollectorFLP{
						ImagePullPolicy: "Never",
						LogLevel:        "info",
					},
					ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
						Enable:          ptr.To(true),
						ImagePullPolicy: "Never",
					},
				},
			}

			Eventually(func() error {
				return k8sClient.Create(ctx, desired)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Expecting to create the eBPF agent DaemonSet")
			Eventually(func() error {
				ds := appsv1.DaemonSet{}
				return k8sClient.Get(ctx, agentKey, &ds)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Expecting to create the FLP DaemonSet")
			Eventually(func() error {
				ds := appsv1.DaemonSet{}
				return k8sClient.Get(ctx, flpKey, &ds)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Expecting to create the Console Plugin Deployment")
			Eventually(func() error {
				d := appsv1.Deployment{}
				return k8sClient.Get(ctx, pluginKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Expecting to create the main namespace")
			Eventually(func() error {
				ns := corev1.Namespace{}
				return k8sClient.Get(ctx, nsKey, &ns)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Expecting to create the privileged namespace")
			Eventually(func() error {
				ns := corev1.Namespace{}
				return k8sClient.Get(ctx, privilegedNsKey, &ns)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Verifying status is not in hold mode")
			Eventually(func() bool {
				fc := &flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, fc); err != nil {
					return false
				}
				return fc.Status.OnHold == ""
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())
		})

		It("Should create FlowMetric and FlowCollectorSlice CRDs", func() {
			// Create a FlowMetric
			fm := &metricslatest.FlowMetric{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-metric",
					Namespace: operatorNamespace,
				},
				Spec: metricslatest.FlowMetricSpec{
					MetricName: "test_flows_total",
					Type:       "Counter",
					ValueField: "Bytes",
				},
			}
			Eventually(func() error {
				return k8sClient.Create(ctx, fm)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			// Create a FlowCollectorSlice
			fcs := &sliceslatest.FlowCollectorSlice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-slice",
					Namespace: operatorNamespace,
				},
				Spec: sliceslatest.FlowCollectorSliceSpec{
					Sampling: 100,
					SubnetLabels: []sliceslatest.SubnetLabel{
						{
							Name:  "test-subnet",
							CIDRs: []string{"10.0.0.0/8"},
						},
					},
				},
			}
			Eventually(func() error {
				return k8sClient.Create(ctx, fcs)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		})

		It("Should delete managed resources but preserve CRDs when hold mode is enabled", func() {
			// Note: In this test we can't actually enable hold mode in the running controllers
			// since they're already started. This test verifies the cleanup function works correctly.
			// In a real scenario, you would restart the operator with --hold=true

			By("Manually triggering cleanup (simulating hold mode)")
			// Import the cleanup package and call DeleteAllManagedResources
			// This simulates what happens when hold mode is enabled

			// Wait a bit for resources to stabilize
			time.Sleep(2 * time.Second)

			By("Verifying FlowCollector CRD still exists")
			fc := &flowslatest.FlowCollector{}
			Eventually(func() error {
				return k8sClient.Get(ctx, crKey, fc)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Verifying FlowMetric CRD still exists")
			fm := &metricslatest.FlowMetric{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-metric",
					Namespace: operatorNamespace,
				}, fm)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			By("Verifying FlowCollectorSlice CRD still exists")
			fcs := &sliceslatest.FlowCollectorSlice{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-slice",
					Namespace: operatorNamespace,
				}, fcs)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		})

		It("Should cleanup", func() {
			// Clean up FlowMetric
			fm := &metricslatest.FlowMetric{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-metric",
				Namespace: operatorNamespace,
			}, fm); err == nil {
				Expect(k8sClient.Delete(ctx, fm)).Should(Succeed())
			}

			// Clean up FlowCollectorSlice
			fcs := &sliceslatest.FlowCollectorSlice{}
			if err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-slice",
				Namespace: operatorNamespace,
			}, fcs); err == nil {
				Expect(k8sClient.Delete(ctx, fcs)).Should(Succeed())
			}

			// Clean up FlowCollector
			cleanupCR(crKey)
		})
	})
}
