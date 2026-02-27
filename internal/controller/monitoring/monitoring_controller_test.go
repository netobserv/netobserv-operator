//nolint:revive
package monitoring

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	metricslatest "github.com/netobserv/network-observability-operator/api/flowmetrics/v1alpha1"
	. "github.com/netobserv/network-observability-operator/internal/controller/controllerstest"
	"github.com/netobserv/network-observability-operator/internal/pkg/dashboards"
	"github.com/netobserv/network-observability-operator/internal/pkg/test"
)

const (
	timeout  = test.Timeout
	interval = test.Interval
)

var (
	updateCR = func(key types.NamespacedName, updater func(*flowslatest.FlowCollector)) {
		test.UpdateCR(ctx, k8sClient, key, updater)
	}
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

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Installing CR", func() {
		It("Create control-cm", func() {
			// control-cm is a control object installed in openshift-config-managed, aiming to make sure we never delete configmaps that we don't own
			Expect(k8sClient.Create(ctx, &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "control-cm", Namespace: "openshift-config-managed"},
				Data:       map[string]string{},
			})).Should(Succeed())
		})

		It("Should create successfully", func() {
			created := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: crKey.Name,
				},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
				},
			}

			// Create
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			By("Expecting the monitoring dashboards configmap to be created")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-main",
					Namespace: "openshift-config-managed",
				}, &cm)
			}, timeout, interval).Should(Succeed())

			By("Expecting the infra health dashboards configmap to be created")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "grafana-dashboard-netobserv-health",
					Namespace: "openshift-config-managed",
				}, &cm); err != nil {
					return err
				}
				d, err := dashboards.FromBytes([]byte(cm.Data["netobserv-health-metrics.json"]))
				if err != nil {
					return err
				}
				return d.Titles()
			}, timeout, interval).Should(Equal([]string{"", "Flowlogs-pipeline statistics", "eBPF agent statistics", "Operator statistics", "Resource usage"}))

			By("Expecting control-cm to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "control-cm", Namespace: "openshift-config-managed"}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())
		})

		It("Should update successfully", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor = flowslatest.FlowCollectorFLP{
					Metrics: flowslatest.FLPMetrics{
						IncludeList:   &[]flowslatest.FLPMetric{},
						DisableAlerts: []flowslatest.HealthRuleTemplate{flowslatest.AlertLokiError},
					},
				}
			})

			By("Expecting the flow dashboards configmap to be deleted")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-main",
					Namespace: "openshift-config-managed",
				}, &v1.ConfigMap{})
			}, timeout, interval).Should(MatchError(`configmaps "netobserv-main" not found`))

			By("Expecting the health dashboard to remain")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "grafana-dashboard-netobserv-health",
					Namespace: "openshift-config-managed",
				}, &cm); err != nil {
					return err
				}
				d, err := dashboards.FromBytes([]byte(cm.Data["netobserv-health-metrics.json"]))
				if err != nil {
					return err
				}
				return d.Titles()
			}, timeout, interval).Should(Equal([]string{"", "Flowlogs-pipeline statistics", "eBPF agent statistics", "Operator statistics", "Resource usage"}))

			By("Expecting control-cm to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "control-cm", Namespace: "openshift-config-managed"}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Installing custom dashboards", func() {
		It("Should fail to create invalid metric name", func() {
			Expect(k8sClient.Create(ctx, &metricslatest.FlowMetric{
				ObjectMeta: metav1.ObjectMeta{Name: "metric1", Namespace: operatorNamespace},
				Spec: metricslatest.FlowMetricSpec{
					MetricName: "my-metric",
					Type:       metricslatest.CounterMetric,
				},
			})).Should(MatchError(`FlowMetric.flows.netobserv.io "metric1" is invalid: spec.metricName: Invalid value: "my-metric": spec.metricName in body should match '^[a-zA-Z_][a-zA-Z0-9:_]*$|^$'`))
		})

		It("Should create FlowMetric 1 successfully", func() {
			Expect(k8sClient.Create(ctx, &metricslatest.FlowMetric{
				ObjectMeta: metav1.ObjectMeta{Name: "metric1", Namespace: operatorNamespace},
				Spec: metricslatest.FlowMetricSpec{
					MetricName: "my_metric",
					Type:       metricslatest.CounterMetric,
					Charts: []metricslatest.Chart{
						{
							DashboardName: "My dashboard 01",
							Title:         "title",
							Type:          metricslatest.ChartTypeSingleStat,
							Queries:       []metricslatest.Query{{PromQL: "(query)", Legend: "-", Top: 7}},
						},
					},
				},
			})).Should(Succeed())
		})

		It("Should create FlowMetric 2 successfully", func() {
			Expect(k8sClient.Create(ctx, &metricslatest.FlowMetric{
				ObjectMeta: metav1.ObjectMeta{Name: "metric2", Namespace: operatorNamespace},
				Spec: metricslatest.FlowMetricSpec{
					Type: metricslatest.CounterMetric,
					Charts: []metricslatest.Chart{
						{
							DashboardName: "My dashboard 02",
							Title:         "title",
							Type:          metricslatest.ChartTypeSingleStat,
							Queries:       []metricslatest.Query{{PromQL: "(query)", Legend: "-", Top: 7}},
						},
					},
				},
			})).Should(Succeed())
		})

		It("Should create corresponding dashboards", func() {
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-my-dashboard-01",
					Namespace: "openshift-config-managed",
				}, &cm)
			}, timeout, interval).Should(Succeed())

			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-my-dashboard-02",
					Namespace: "openshift-config-managed",
				}, &cm)
			}, timeout, interval).Should(Succeed())
		})

		It("Should delete dashboard 2", func() {
			By("Getting FlowMetric 2")
			fm := metricslatest.FlowMetric{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "metric2", Namespace: operatorNamespace}, &fm)
			}).Should(Succeed())

			By("Deleting FlowMetric 2")
			Eventually(func() error { return k8sClient.Delete(ctx, &fm) }, timeout, interval).Should(Succeed())

			By("Expecting dashboard 2 to be deleted")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-my-dashboard-02",
					Namespace: "openshift-config-managed",
				}, &v1.ConfigMap{})
			}, timeout, interval).Should(MatchError(`configmaps "netobserv-my-dashboard-02" not found`))

			By("Expecting dashboard 1 to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-my-dashboard-01",
					Namespace: "openshift-config-managed",
				}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())

			By("Expecting the health dashboard to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "grafana-dashboard-netobserv-health",
					Namespace: "openshift-config-managed",
				}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())

			By("Expecting control-cm to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "control-cm", Namespace: "openshift-config-managed"}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Checking CR ownership", func() {
		It("Should be garbage collected", func() {
			// Retrieve CR to get its UID
			By("Getting the CR")
			flowCR := getCR(crKey)

			By("Expecting the health dashboards configmap to be garbage collected")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      "grafana-dashboard-netobserv-health",
					Namespace: "openshift-config-managed",
				}, &cm)
				return &cm
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)

			By("Expecting control-cm to remain")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "control-cm", Namespace: "openshift-config-managed"}, &v1.ConfigMap{})
			}, timeout, interval).Should(Succeed())
		})
	})
}
