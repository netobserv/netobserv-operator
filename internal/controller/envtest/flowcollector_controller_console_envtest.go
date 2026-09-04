//nolint:revive,staticcheck
package envtest

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1 "github.com/openshift/api/operator/v1"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

// Because the simulated Kube server doesn't manage automatic resource cleanup like an actual Kube would do,
// we need either to cleanup all created resources manually, or to use different namespaces between tests
// For simplicity, we'll use a different namespace
const cpNamespace = "namespace-console-specs"

// nolint:cyclop
func FlowCollectorConsolePluginSpecs(env test.Environment, ctxGetter test.ContextGetter) {
	var ctx context.Context
	var k8sClient client.Client
	BeforeEach(func() {
		ctx, k8sClient = ctxGetter()
	})

	staticCpKey := types.NamespacedName{
		Name:      "netobserv-plugin-static",
		Namespace: "main-namespace",
	}
	cpKey := types.NamespacedName{
		Name:      "netobserv-plugin",
		Namespace: cpNamespace,
	}
	configKey := types.NamespacedName{
		Name:      "console-plugin-config",
		Namespace: cpNamespace,
	}
	crKey := types.NamespacedName{
		Name: "cluster",
	}
	consoleCRKey := types.NamespacedName{
		Name: "cluster",
	}
	subscriptionKey := types.NamespacedName{
		Name:      "netobserv-operator",
		Namespace: "main-namespace",
	}
	if env == test.EnvOpenShift {
		Context("Deploying the static console plugin", func() {
			It("Should create successfully", func() {
				By("Expecting to create the static console plugin Deployment")
				dp := appsv1.Deployment{}
				Eventually(func() interface{} {
					return k8sClient.Get(ctx, staticCpKey, &dp)
				}, timeout, interval).Should(Succeed())

				By("Expecting to create the static console plugin Service")
				svc := v1.Service{}
				Eventually(func() interface{} {
					return k8sClient.Get(ctx, staticCpKey, &svc)
				}, timeout, interval).Should(Succeed())
			})
		})
	}

	Context("Create FlowCollector CR", func() {
		It("Should create CR successfully", func() {
			isStandalone := env != test.EnvOpenShift
			created := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: crKey.Name,
				},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       cpNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
					Agent:           flowslatest.FlowCollectorAgent{Type: flowslatest.AgentEBPF},
					ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
						Enable:          ptr.To(true),
						Standalone:      &isStandalone,
						ImagePullPolicy: "Never",
						Advanced: &flowslatest.AdvancedPluginConfig{
							Register: ptr.To(false),
						},
						Autoscaler: flowslatest.FlowCollectorHPA{
							Status:      flowslatest.HPAStatusEnabled,
							MinReplicas: ptr.To(int32(1)),
							MaxReplicas: 1,
							Metrics: []ascv2.MetricSpec{{
								Type: ascv2.ResourceMetricSourceType,
								Resource: &ascv2.ResourceMetricSource{
									Name: v1.ResourceCPU,
									Target: ascv2.MetricTarget{
										Type:               ascv2.UtilizationMetricType,
										AverageUtilization: ptr.To(int32(90)),
									},
								},
							}},
						},
						PortNaming: flowslatest.ConsolePluginPortConfig{
							Enable: ptr.To(true),
							PortNames: map[string]string{
								"3100": "loki",
							},
						},
					},
				},
			}

			// Create
			Eventually(func() interface{} {
				return k8sClient.Create(ctx, created)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Deploying the console plugin", func() {
		It("Should create successfully", func() {
			By("Expecting to create the console plugin Deployment")
			dp := appsv1.Deployment{}
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, cpKey, &dp); err != nil {
					return err
				}
				return *dp.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(1)))

			if env == test.EnvOpenShift {
				By("In OpenShift, expecting to have the required SCC annotation")
				Expect(dp.Spec.Template.Annotations).To(HaveKeyWithValue("openshift.io/required-scc", "restricted-v2"))
			}

			By("Expecting to create the console plugin Service")
			Eventually(func() interface{} {
				svc := v1.Service{}
				if err := k8sClient.Get(ctx, cpKey, &svc); err != nil {
					return err
				}
				return svc.Spec.Ports[0].Port
			}, timeout, interval).Should(Equal(int32(9001)))

			By("Creating the console plugin configmap")
			Eventually(getConfigMapData(ctx, k8sClient, configKey),
				timeout, interval).Should(ContainSubstring("url: http://loki:3100/"))

		})

		It("Should update successfully", func() {
			Eventually(func() error {
				fc := flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, &fc); err != nil {
					return err
				}
				fc.Spec.ConsolePlugin.Advanced.Port = ptr.To(int32(9099))
				fc.Spec.ConsolePlugin.Replicas = ptr.To(int32(2))
				fc.Spec.ConsolePlugin.Autoscaler.Status = flowslatest.HPAStatusDisabled
				return k8sClient.Update(ctx, &fc)
			}).Should(Succeed())

			By("Expecting the console plugin Deployment to be scaled up")
			Eventually(func() interface{} {
				dp := appsv1.Deployment{}
				if err := k8sClient.Get(ctx, cpKey, &dp); err != nil {
					return err
				}
				return *dp.Spec.Replicas
			}, timeout, interval).Should(Equal(int32(2)))

			By("Expecting the console plugin Service to be updated")
			Eventually(func() interface{} {
				svc := v1.Service{}
				if err := k8sClient.Get(ctx, cpKey, &svc); err != nil {
					return err
				}
				return svc.Spec.Ports[0].Port
			}, timeout, interval).Should(Equal(int32(9099)))
		})

		// No ServiceMonitor expected in VanillaNaked
		if env != test.EnvVanillaNaked {
			It("Should create desired objects when they're not found (e.g. case of an operator upgrade)", func() {
				sm := monitoringv1.ServiceMonitor{}

				By("Expecting ServiceMonitor to exist")
				Eventually(func() interface{} {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      "netobserv-plugin",
						Namespace: cpNamespace,
					}, &sm)
				}, timeout, interval).Should(Succeed())

				// Manually delete ServiceMonitor
				By("Deleting ServiceMonitor")
				Eventually(func() error {
					return k8sClient.Delete(ctx, &sm)
				}, timeout, interval).Should(Succeed())

				// Do a dummy change that will trigger reconcile, and make sure SM is created again
				test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
					fc.Spec.Processor.LogLevel = "trace"
				})
				By("Expecting ServiceMonitor to exist")
				Eventually(func() interface{} {
					return k8sClient.Get(ctx, types.NamespacedName{
						Name:      "netobserv-plugin",
						Namespace: cpNamespace,
					}, &sm)
				}, timeout, interval).Should(Succeed())
			})
		}
	})

	Context("FlowCollector status", func() {
		It("Should have console ready status", func() {
			// Manually set deployment ready
			dp := appsv1.Deployment{}
			Eventually(func() interface{} { return k8sClient.Get(ctx, cpKey, &dp) }, timeout, interval).Should(Succeed())
			dp.Status.AvailableReplicas = 1
			dp.Status.ReadyReplicas = 1
			dp.Status.Replicas = 1
			dp.Status.UpdatedReplicas = 1
			Eventually(func() interface{} { return k8sClient.Status().Update(ctx, &dp) }, timeout, interval).Should(Succeed())

			Eventually(func() interface{} {
				fc := test.GetCR(ctx, k8sClient, crKey)
				return meta.FindStatusCondition(fc.Status.Conditions, "WaitingWebConsole")
			}, timeout, interval).Should(Satisfy(func(c *metav1.Condition) bool {
				return c != nil && c.Status == metav1.ConditionFalse && c.Reason == "Ready"
			}))
		})

		if env == test.EnvOpenShift {
			It("Should have static plugin ready status", func() {
				// Manually set deployment ready
				dp := appsv1.Deployment{}
				Eventually(func() interface{} { return k8sClient.Get(ctx, staticCpKey, &dp) }, timeout, interval).Should(Succeed())
				dp.Status.AvailableReplicas = 1
				dp.Status.ReadyReplicas = 1
				dp.Status.Replicas = 1
				dp.Status.UpdatedReplicas = 1
				Eventually(func() interface{} { return k8sClient.Status().Update(ctx, &dp) }, timeout, interval).Should(Succeed())

				Eventually(func() interface{} {
					fc := test.GetCR(ctx, k8sClient, crKey)
					return meta.FindStatusCondition(fc.Status.Conditions, "WaitingStaticController")
				}, timeout, interval).Should(Satisfy(func(c *metav1.Condition) bool {
					return c != nil && c.Status == metav1.ConditionFalse && c.Reason == "Ready"
				}))
			})
		}
	})

	Context("Configuring the Loki URL", func() {
		It("Should be initially configured with default Loki URL", func() {
			Eventually(getConfigMapData(ctx, k8sClient, configKey),
				timeout, interval).Should(ContainSubstring("url: http://loki:3100/"))
		})
		It("Should update the Loki URL in the Console Plugin if it changes in the Spec", func() {
			test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Monolithic.InstallDemoLoki = ptr.To(false)
				fc.Spec.Loki.Monolithic.URL = "http://loki.namespace:8888"
			})
			Eventually(getConfigMapData(ctx, k8sClient, configKey),
				timeout, interval).Should(ContainSubstring("url: http://loki.namespace:8888"))
		})
		It("Should use the Loki Querier URL instead of the Loki URL, when switching to manual mode", func() {
			test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeManual
				fc.Spec.Loki.Manual.QuerierURL = "http://loki-querier:6789"
			})
			Eventually(getConfigMapData(ctx, k8sClient, configKey),
				timeout, interval).Should(ContainSubstring("url: http://loki-querier:6789"))
		})
	})

	if env == test.EnvOpenShift {
		Context("Registering to the Console CR", func() {
			It("Should start with static plugin registered", func() {
				Eventually(func() interface{} {
					cr := operatorsv1.Console{}
					if err := k8sClient.Get(ctx, consoleCRKey, &cr); err != nil {
						return err
					}
					return cr.Spec.Plugins
				}, timeout, interval).Should(Equal([]string{"netobserv-plugin-static"}))
			})

			It("Should be registered", func() {
				By("Update CR to registered")
				test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
					fc.Spec.ConsolePlugin.Advanced.Register = ptr.To(true)
				})

				By("Expecting the Console CR to have both plugins registered")
				Eventually(func() interface{} {
					cr := operatorsv1.Console{}
					if err := k8sClient.Get(ctx, consoleCRKey, &cr); err != nil {
						return err
					}
					return cr.Spec.Plugins
				}, timeout, interval).Should(Equal([]string{"netobserv-plugin-static", "netobserv-plugin"}))
			})
		})

		Context("Static plugin toleration config", func() {
			It("Should configure toleration from subscription", func() {
				Eventually(func() interface{} {
					subs := olm.Subscription{}
					if err := k8sClient.Get(ctx, subscriptionKey, &subs); err != nil {
						return err
					}
					subs.Spec.Config = &olm.SubscriptionConfig{
						Tolerations: []v1.Toleration{
							{
								Key:      "foo",
								Operator: v1.TolerationOpEqual,
								Value:    "bar",
							},
						},
						NodeSelector: map[string]string{
							"foo": "bar",
						},
					}
					return k8sClient.Update(ctx, &subs)
				}, timeout, interval).Should(Succeed())
			})

			It("Should propagate toleration config", func() {
				Eventually(func() interface{} {
					dp := appsv1.Deployment{}
					if err := k8sClient.Get(ctx, staticCpKey, &dp); err != nil {
						return err
					}
					if len(dp.Spec.Template.Spec.Tolerations) != 1 || dp.Spec.Template.Spec.Tolerations[0].Key != "foo" {
						return fmt.Errorf("expected toleration set, got: %v", dp.Spec.Template.Spec.Tolerations)
					}
					if len(dp.Spec.Template.Spec.NodeSelector) != 1 || dp.Spec.Template.Spec.NodeSelector["foo"] != "bar" {
						return fmt.Errorf("expected node selector set with foo=bar, got: %v", dp.Spec.Template.Spec.NodeSelector)
					}
					return nil
				}, timeout, interval).Should(Succeed())
			})

			It("Should clear toleration config when subscription config is removed", func() {
				Eventually(func() interface{} {
					subs := olm.Subscription{}
					if err := k8sClient.Get(ctx, subscriptionKey, &subs); err != nil {
						return err
					}
					subs.Spec.Config = nil
					return k8sClient.Update(ctx, &subs)
				}, timeout, interval).Should(Succeed())
			})

			It("Should remove toleration and nodeSelector from static plugin deployment", func() {
				Eventually(func() interface{} {
					dp := appsv1.Deployment{}
					if err := k8sClient.Get(ctx, staticCpKey, &dp); err != nil {
						return err
					}
					if len(dp.Spec.Template.Spec.Tolerations) != 0 {
						return fmt.Errorf("expected no tolerations, got: %v", dp.Spec.Template.Spec.Tolerations)
					}
					if len(dp.Spec.Template.Spec.NodeSelector) != 0 {
						return fmt.Errorf("expected no node selector, got: %v", dp.Spec.Template.Spec.NodeSelector)
					}
					return nil
				}, timeout, interval).Should(Succeed())
			})
		})
	}

	Context("Update enable option", func() {
		It("Should be initially enabled", func() {
			Eventually(func() interface{} {
				fc := flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, &fc); err != nil {
					return err
				}
				return *fc.Spec.ConsolePlugin.Enable
			}, timeout, interval).Should(Equal(true))
		})

		It("Should cleanup console plugin if disabled", func() {
			test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.ConsolePlugin.Enable = ptr.To(false)
			})
			Eventually(func() error {
				d := appsv1.Deployment{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Satisfy(errors.IsNotFound))
			Eventually(func() error {
				d := v1.Service{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Satisfy(errors.IsNotFound))
			Eventually(func() error {
				d := v1.ServiceAccount{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Satisfy(errors.IsNotFound))
		})

		It("Should recreate console plugin if enabled back", func() {
			test.UpdateCR(ctx, k8sClient, crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.ConsolePlugin.Enable = ptr.To(true)
			})
			Eventually(func() error {
				d := appsv1.Deployment{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Succeed())
			Eventually(func() error {
				d := v1.Service{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Succeed())
			Eventually(func() error {
				d := v1.ServiceAccount{}
				return k8sClient.Get(ctx, cpKey, &d)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Succeed())
		})
	})

	Context("Checking CR ownership", func() {
		It("Should be garbage collected", func() {
			// Retrieve CR to get its UID
			By("Getting the CR")
			flowCR := test.GetCR(ctx, k8sClient, crKey)

			By("Expecting console plugin deployment to be garbage collected")
			Eventually(func() interface{} {
				d := appsv1.Deployment{}
				_ = k8sClient.Get(ctx, cpKey, &d)
				return &d
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting console plugin service to be garbage collected")
			Eventually(func() interface{} {
				svc := v1.Service{}
				_ = k8sClient.Get(ctx, cpKey, &svc)
				return &svc
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting console plugin service account to be garbage collected")
			Eventually(func() interface{} {
				svcAcc := v1.ServiceAccount{}
				_ = k8sClient.Get(ctx, cpKey, &svcAcc)
				return &svcAcc
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))
		})
	})

	Context("Checking controller ownership", func() {
		It("Should be garbage collected", func() {
			dp := appsv1.Deployment{}
			By("Getting controller deployment")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-controller-manager",
					Namespace: "main-namespace",
				}, &dp)
			}, timeout, interval).Should(Succeed())

			if env == test.EnvOpenShift {
				By("Expecting static console plugin deployment to be garbage collected")
				Eventually(func() interface{} {
					d := appsv1.Deployment{}
					_ = k8sClient.Get(ctx, staticCpKey, &d)
					return &d
				}, timeout, interval).Should(BeGarbageCollectedBy(&dp))

				By("Expecting static console plugin service to be garbage collected")
				Eventually(func() interface{} {
					svc := v1.Service{}
					_ = k8sClient.Get(ctx, staticCpKey, &svc)
					return &svc
				}, timeout, interval).Should(BeGarbageCollectedBy(&dp))
			}
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			test.CleanupCR(ctx, k8sClient, crKey)
		})

		It("Should delete fake controller", func() {
			dp := appsv1.Deployment{}
			By("Retrieve controller deployment")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "netobserv-controller-manager",
					Namespace: "main-namespace",
				}, &dp)
			}, timeout, interval).Should(Succeed())

			By("Delete controller deployment")
			Eventually(func() error {
				return k8sClient.Delete(ctx, &dp)
			}, timeout, interval).Should(Succeed())
		})
	})
}

func getConfigMapData(ctx context.Context, k8sClient client.Client, configKey types.NamespacedName) func() interface{} {
	return func() interface{} {
		ofc := v1.ConfigMap{}
		if err := k8sClient.Get(ctx, configKey, &ofc); err != nil {
			return err
		}
		return ofc.Data["config.yaml"]
	}
}
