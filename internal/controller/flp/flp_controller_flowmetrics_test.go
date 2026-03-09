//nolint:revive
package flp

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/fmstatus"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper/cardinality"
)

// nolint:cyclop
func ControllerFlowMetricsSpecs() {
	const operatorNamespace = "main-namespace"
	const otherNamespace = "other-namespace"
	crKey := types.NamespacedName{
		Name: "cluster",
	}
	flpKey := types.NamespacedName{
		Name:      constants.FLPName,
		Namespace: operatorNamespace,
	}
	cmKey := types.NamespacedName{
		Name:      "flowlogs-pipeline-config",
		Namespace: operatorNamespace,
	}
	dcmKey := types.NamespacedName{
		Name:      "flowlogs-pipeline-config-dynamic",
		Namespace: operatorNamespace,
	}
	metric1 := metricslatest.FlowMetric{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metric-1",
			Namespace: operatorNamespace,
		},
		Spec: metricslatest.FlowMetricSpec{
			MetricName: "m_1",
			Type:       metricslatest.CounterMetric,
		},
	}
	metric2 := metricslatest.FlowMetric{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metric-2",
			Namespace: operatorNamespace,
		},
		Spec: metricslatest.FlowMetricSpec{
			MetricName: "m_2",
			Type:       metricslatest.CounterMetric,
			Labels:     []string{"DstAddr"},
		},
	}
	metric3 := metricslatest.FlowMetric{ // With nested labels
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metric-3",
			Namespace: operatorNamespace,
		},
		Spec: metricslatest.FlowMetricSpec{
			MetricName: "m_3",
			Type:       metricslatest.CounterMetric,
			Labels:     []string{"NetworkEvents>Type", "NetworkEvents>Name"},
		},
	}
	metricUnwatched := metricslatest.FlowMetric{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metric-unwatched",
			Namespace: otherNamespace,
		},
		Spec: metricslatest.FlowMetricSpec{
			MetricName: "m_unwatched",
			Type:       metricslatest.CounterMetric,
		},
	}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Deploying default FLP", func() {
		ds := appsv1.DaemonSet{}
		cm := v1.ConfigMap{}
		dcm := v1.ConfigMap{}
		It("Should create successfully", func() {
			created := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
				},
			}

			// Create
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			By("Expecting to create the flowlogs-pipeline DaemonSet")
			Eventually(func() error {
				return k8sClient.Get(ctx, flpKey, &ds)
			}, timeout, interval).Should(Succeed())

			By("Expecting flowlogs-pipeline-config configmap to be created")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, cmKey, &cm)
			}, timeout, interval).Should(Succeed())

			By("Expecting flowlogs-pipeline-config-dynamic configmap to be created")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, dcmKey, &dcm); err != nil {
					return err
				}
				metrics, err := getConfiguredMetrics(&dcm)
				if err != nil {
					return err
				}
				return metrics
			}, timeout, interval).Should(HaveLen(8)) // only default metrics
		})
	})

	Context("Creating FlowMetrics", func() {
		It("Should create successfully", func() {
			Expect(k8sClient.Create(ctx, &metric1)).Should(Succeed())
			Expect(k8sClient.Create(ctx, &metricUnwatched)).Should(Succeed())
			Expect(k8sClient.Create(ctx, &metric2)).Should(Succeed())
			Expect(k8sClient.Create(ctx, &metric3)).Should(Succeed())
		})

		It("Should update configmap with custom metrics", func() {
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				err := k8sClient.Get(ctx, dcmKey, &cm)
				if err != nil {
					return err
				}
				metrics, err := getConfiguredMetrics(&cm)
				if err != nil {
					return err
				}
				return metrics
			}, timeout, interval).Should(Satisfy(func(metrics api.MetricsItems) bool {
				names := getSortedMetricsNames(metrics)
				return slices.Contains(names, metric1.Spec.MetricName) &&
					slices.Contains(names, metric2.Spec.MetricName) &&
					slices.Contains(names, metric3.Spec.MetricName) &&
					!slices.Contains(names, metricUnwatched.Spec.MetricName)
			}))
		})

		It("Should be updated with status", func() {
			Eventually(func() interface{} {
				err := k8sClient.Get(ctx, helper.NamespacedName(&metric1), &metric1)
				if err != nil {
					return err
				}
				return metric1.Status.Conditions
			}, timeout, interval).Should(Satisfy(func(conds []metav1.Condition) bool {
				ready := meta.FindStatusCondition(conds, fmstatus.ConditionReady)
				card := meta.FindStatusCondition(conds, fmstatus.ConditionCardinalityWarning)
				// Metrics 1 has cardinality FINE (no label)
				return ready != nil && card != nil && ready.Status == metav1.ConditionTrue && card.Status == metav1.ConditionFalse &&
					ready.Reason == "Ready" && card.Reason == string(cardinality.WarnFine)
			}))

			Eventually(func() interface{} {
				err := k8sClient.Get(ctx, helper.NamespacedName(&metric2), &metric2)
				if err != nil {
					return err
				}
				return metric2.Status.Conditions
			}, timeout, interval).Should(Satisfy(func(conds []metav1.Condition) bool {
				// Metrics 2 has cardinality AVOID (Addr label)
				ready := meta.FindStatusCondition(conds, fmstatus.ConditionReady)
				card := meta.FindStatusCondition(conds, fmstatus.ConditionCardinalityWarning)
				return ready != nil && card != nil && ready.Status == metav1.ConditionTrue && card.Status == metav1.ConditionTrue &&
					ready.Reason == "Ready" && card.Reason == string(cardinality.WarnAvoid)
			}))

			Eventually(func() interface{} {
				err := k8sClient.Get(ctx, helper.NamespacedName(&metric3), &metric3)
				if err != nil {
					return err
				}
				return metric3.Status.Conditions
			}, timeout, interval).Should(Satisfy(func(conds []metav1.Condition) bool {
				// Metrics 3 has cardinality FINE (NetworkEvents nested labels)
				ready := meta.FindStatusCondition(conds, fmstatus.ConditionReady)
				card := meta.FindStatusCondition(conds, fmstatus.ConditionCardinalityWarning)
				return ready != nil && card != nil && ready.Status == metav1.ConditionTrue && card.Status == metav1.ConditionFalse &&
					ready.Reason == "Ready" && card.Reason == string(cardinality.WarnFine)
			}))
		})
	})

	Context("Updating a FlowMetric", func() {
		It("Should update successfully", func() {
			metric1.Spec.MetricName = "m_1_bis"
			Expect(k8sClient.Update(ctx, &metric1)).To(Succeed())
		})

		It("Should update configmap with custom metrics", func() {
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				err := k8sClient.Get(ctx, dcmKey, &cm)
				if err != nil {
					return err
				}
				metrics, err := getConfiguredMetrics(&cm)
				if err != nil {
					return err
				}
				return metrics
			}, timeout, interval).Should(Satisfy(func(metrics api.MetricsItems) bool {
				names := getSortedMetricsNames(metrics)
				return slices.Contains(names, "m_1_bis") &&
					slices.Contains(names, metric2.Spec.MetricName) &&
					!slices.Contains(names, metricUnwatched.Spec.MetricName)
			}))
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)
		})
	})
}
