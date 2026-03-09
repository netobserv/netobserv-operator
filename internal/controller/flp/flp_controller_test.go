//nolint:revive
package flp

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	. "github.com/netobserv/netobserv-operator/internal/controller/controllerstest"
	"github.com/netobserv/netobserv-operator/internal/pkg/resources"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

const (
	timeout                     = test.Timeout
	interval                    = test.Interval
	conntrackEndTimeout         = 10 * time.Second
	conntrackTerminatingTimeout = 5 * time.Second
	conntrackHeartbeatInterval  = 30 * time.Second
)

var (
	outputRecordTypes = flowslatest.LogTypeAll
	updateCR          = func(key types.NamespacedName, updater func(*flowslatest.FlowCollector)) {
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
	flpKey1 := types.NamespacedName{
		Name:      constants.FLPName,
		Namespace: operatorNamespace,
	}
	flpKeyKafkaTransformer := types.NamespacedName{
		Name:      transfoName,
		Namespace: operatorNamespace,
	}
	rbKeyConfigWatcherMono := types.NamespacedName{Name: resources.GetRoleBindingName(monoShortName, constants.ConfigWatcherRole), Namespace: operatorNamespace}
	rbKeyHostNetworkMono := types.NamespacedName{Name: resources.GetClusterRoleBindingName(monoShortName, constants.HostNetworkRole)}
	rbKeyLokiWriterMono := types.NamespacedName{Name: resources.GetClusterRoleBindingName(monoShortName, constants.LokiWriterRole)}
	rbKeyInformerMono := types.NamespacedName{Name: resources.GetClusterRoleBindingName(monoShortName, constants.FLPInformersRole)}
	rbKeyConfigWatcherTransfo := types.NamespacedName{Name: resources.GetRoleBindingName(transfoShortName, constants.ConfigWatcherRole), Namespace: operatorNamespace}
	rbKeyHostNetworkTransfo := types.NamespacedName{Name: resources.GetClusterRoleBindingName(transfoShortName, constants.HostNetworkRole)}
	rbKeyLokiWriterTransfo := types.NamespacedName{Name: resources.GetClusterRoleBindingName(transfoShortName, constants.LokiWriterRole)}
	rbKeyInformerTransfo := types.NamespacedName{Name: resources.GetClusterRoleBindingName(transfoShortName, constants.FLPInformersRole)}

	// Created objects to cleanup
	cleanupList := []client.Object{}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Deploying as DaemonSet", func() {
		var digest string
		ds := appsv1.DaemonSet{}
		It("Should create successfully", func() {
			created := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
					Processor: flowslatest.FlowCollectorFLP{
						ImagePullPolicy: "Never",
						LogLevel:        "error",
						Advanced: &flowslatest.AdvancedProcessorConfig{
							Env: map[string]string{
								"GOGC": "200",
							},
							ConversationHeartbeatInterval: &metav1.Duration{
								Duration: conntrackHeartbeatInterval,
							},
							ConversationEndTimeout: &metav1.Duration{
								Duration: conntrackEndTimeout,
							},
							ConversationTerminatingTimeout: &metav1.Duration{
								Duration: conntrackTerminatingTimeout,
							},
						},
						LogTypes: &outputRecordTypes,
						Metrics: flowslatest.FLPMetrics{
							IncludeList: &[]flowslatest.FLPMetric{"node_ingress_bytes_total", "namespace_ingress_bytes_total", "workload_ingress_bytes_total"},
						},
					},
				},
			}

			// Create
			Expect(k8sClient.Create(ctx, created)).Should(Succeed())

			By("Expecting to create the flowlogs-pipeline DaemonSet")
			Eventually(func() error {
				if err := k8sClient.Get(ctx, flpKey1, &ds); err != nil {
					return err
				}
				digest = ds.Spec.Template.Annotations[constants.PodConfigurationDigest]
				if digest == "" {
					return fmt.Errorf("%q annotation can't be empty", constants.PodConfigurationDigest)
				}
				return nil
			}, timeout, interval).Should(Succeed())

			By("Expecting to create the flowlogs-pipeline ServiceAccount")
			Eventually(func() interface{} {
				svcAcc := v1.ServiceAccount{}
				if err := k8sClient.Get(ctx, flpKey1, &svcAcc); err != nil {
					return err
				}
				return svcAcc
			}, timeout, interval).Should(Satisfy(func(svcAcc v1.ServiceAccount) bool {
				return svcAcc.Labels != nil && svcAcc.Labels["app"] == constants.FLPName
			}))

			By("Expecting to create two flowlogs-pipeline role bindings")
			rb1 := rbacv1.RoleBinding{}
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyConfigWatcherMono, &rb1)
			}, timeout, interval).Should(Succeed())
			Expect(rb1.Subjects).Should(HaveLen(1))
			Expect(rb1.Subjects[0].Name).Should(Equal("flowlogs-pipeline"))
			Expect(rb1.RoleRef.Name).Should(Equal("netobserv-config-watcher"))

			rb2 := rbacv1.ClusterRoleBinding{}
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyHostNetworkMono, &rb2)
			}, timeout, interval).Should(Succeed())
			Expect(rb2.Subjects).Should(HaveLen(1))
			Expect(rb2.Subjects[0].Name).Should(Equal("flowlogs-pipeline"))
			Expect(rb2.RoleRef.Name).Should(Equal("netobserv-hostnetwork"))

			rb3 := rbacv1.ClusterRoleBinding{}
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyInformerMono, &rb3)
			}, timeout, interval).Should(Succeed())
			Expect(rb3.Subjects).Should(HaveLen(1))
			Expect(rb3.Subjects[0].Name).Should(Equal("flowlogs-pipeline"))
			Expect(rb3.RoleRef.Name).Should(Equal("netobserv-informers"))

			By("Not expecting Loki role (requires LokiStack)")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyLokiWriterMono, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-loki-writer-flp" not found`))

			By("Not expecting transformer role bindings")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyConfigWatcherTransfo, &rbacv1.RoleBinding{})
			}, timeout, interval).Should(MatchError(`rolebindings.rbac.authorization.k8s.io "netobserv-config-watcher-flptransfo" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyHostNetworkTransfo, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-hostnetwork-flptransfo" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyInformerTransfo, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-informers-flptransfo" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyLokiWriterTransfo, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-loki-writer-flptransfo" not found`))

			By("Expecting flowlogs-pipeline-config configmap to be created")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-config",
					Namespace: operatorNamespace,
				}, &cm)
			}, timeout, interval).Should(Succeed())
		})

		It("Should update successfully", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor = flowslatest.FlowCollectorFLP{
					ImagePullPolicy: "Never",
					LogLevel:        "error",
					Advanced: &flowslatest.AdvancedProcessorConfig{
						Env: map[string]string{
							// we'll test that env vars are sorted, to keep idempotency
							"GOMAXPROCS": "33",
							"GOGC":       "400",
						},
						Port: ptr.To(int32(7891)),
						ConversationHeartbeatInterval: &metav1.Duration{
							Duration: conntrackHeartbeatInterval,
						},
						ConversationEndTimeout: &metav1.Duration{
							Duration: conntrackEndTimeout,
						},
						ConversationTerminatingTimeout: &metav1.Duration{
							Duration: conntrackTerminatingTimeout,
						},
					},
					LogTypes: &outputRecordTypes,
					Metrics: flowslatest.FLPMetrics{
						IncludeList:   &[]flowslatest.FLPMetric{"node_ingress_bytes_total"},
						DisableAlerts: []flowslatest.HealthRuleTemplate{flowslatest.AlertLokiError},
					},
				}
				fc.Spec.Loki = flowslatest.FlowCollectorLoki{}
			})

			Eventually(func(g Gomega) {
				By("CR updated")
				err := k8sClient.Get(ctx, flpKey1, &ds)
				g.Expect(err).NotTo(HaveOccurred())
				err = checkDigestUpdate(&digest, ds.Spec.Template.Annotations)
				g.Expect(err).NotTo(HaveOccurred())

				By("Creating the required HostPort to access flowlogs-pipeline through the NodeIP")
				var cnt *v1.Container
				for i := range ds.Spec.Template.Spec.Containers {
					if ds.Spec.Template.Spec.Containers[i].Name == constants.FLPName {
						cnt = &ds.Spec.Template.Spec.Containers[i]
						break
					}
				}
				g.Expect(cnt).ToNot(BeNil(), "can't find a container named", constants.FLPName)
				var cp *v1.ContainerPort
				for i := range cnt.Ports {
					if cnt.Ports[i].Name == constants.FLPPortName {
						cp = &cnt.Ports[i]
						break
					}
				}
				g.Expect(cp).ToNot(BeNil(), "can't find a container port named", constants.FLPPortName)
				g.Expect(*cp).To(Equal(v1.ContainerPort{
					Name:          constants.FLPPortName,
					HostPort:      7891,
					ContainerPort: 7891,
					Protocol:      "TCP",
				}))
				g.Expect(cnt.Env).To(Equal([]v1.EnvVar{
					{Name: "GOGC", Value: "400"},
					{Name: "GOMAXPROCS", Value: "33"},
					{Name: "GODEBUG", Value: "http2server=0"},
				}))

				By("Allocating the proper toleration to allow its placement in the master nodes")
				g.Expect(ds.Spec.Template.Spec.Tolerations).
					To(ContainElement(v1.Toleration{Operator: v1.TolerationOpExists}))
			}, timeout, interval).Should(Succeed())
		})

		It("Should redeploy if the spec doesn't change but the external flowlogs-pipeline-config does", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Advanced = &flowslatest.AdvancedLokiConfig{
					WriteMaxRetries: ptr.To(int32(7)),
				}
			})

			By("Expecting that the flowlogsPipeline.PodConfigurationDigest attribute has changed")
			Eventually(func() error {
				if err := k8sClient.Get(ctx, flpKey1, &ds); err != nil {
					return err
				}
				return checkDigestUpdate(&digest, ds.Spec.Template.Annotations)
			}).Should(Succeed())
		})
	})

	Context("With Kafka", func() {
		It("Should update kafka config successfully", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.DeploymentModel = flowslatest.DeploymentModelKafka
				fc.Spec.Kafka = flowslatest.FlowCollectorKafka{
					Address: "localhost:9092",
					Topic:   "FLP",
					TLS: flowslatest.ClientTLS{
						CACert: flowslatest.CertificateReference{
							Type:     "secret",
							Name:     "some-secret",
							CertFile: "ca.crt",
						},
					},
				}
			})
		})

		It("Should deploy kafka transformer", func() {
			By("Expecting transformer deployment to be created")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKeyKafkaTransformer, &appsv1.Deployment{})
			}, timeout, interval).Should(Succeed())

			By("Not Expecting transformer service to be created")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKeyKafkaTransformer, &v1.Service{})
			}, timeout, interval).Should(MatchError(`services "flowlogs-pipeline-transformer" not found`))

			By("Expecting to create transformer flowlogs-pipeline role bindings")
			rb1 := rbacv1.RoleBinding{}
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyConfigWatcherTransfo, &rb1)
			}, timeout, interval).Should(Succeed())
			Expect(rb1.Subjects).Should(HaveLen(1))
			Expect(rb1.Subjects[0].Name).Should(Equal("flowlogs-pipeline-transformer"))
			Expect(rb1.RoleRef.Name).Should(Equal("netobserv-config-watcher"))

			rb2 := rbacv1.ClusterRoleBinding{}
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyInformerTransfo, &rb2)
			}, timeout, interval).Should(Succeed())
			Expect(rb2.Subjects).Should(HaveLen(1))
			Expect(rb2.Subjects[0].Name).Should(Equal("flowlogs-pipeline-transformer"))
			Expect(rb2.RoleRef.Name).Should(Equal("netobserv-informers"))

			By("Not expecting hostnetwork role (not needed with Kafka)")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyHostNetworkTransfo, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-hostnetwork-flptransfo" not found`))

			By("Not expecting Loki role (requires LokiStack)")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyLokiWriterTransfo, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-loki-writer-flptransfo" not found`))

			By("Not expecting mono role bindings")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyConfigWatcherMono, &rbacv1.RoleBinding{})
			}, timeout, interval).Should(MatchError(`rolebindings.rbac.authorization.k8s.io "netobserv-config-watcher-flp" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyHostNetworkMono, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-hostnetwork-flp" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyInformerMono, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-informers-flp" not found`))
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, rbKeyLokiWriterMono, &rbacv1.ClusterRoleBinding{})
			}, timeout, interval).Should(MatchError(`clusterrolebindings.rbac.authorization.k8s.io "netobserv-loki-writer-flp" not found`))
		})

		It("Should delete previous flp deployment", func() {
			By("Expecting monolith to be deleted")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKey1, &appsv1.DaemonSet{})
			}, timeout, interval).Should(MatchError(`daemonsets.apps "flowlogs-pipeline" not found`))
		})
	})

	Context("Adding auto-scaling", func() {
		hpa := ascv2.HorizontalPodAutoscaler{}
		It("Should update with HPA", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor.KafkaConsumerAutoscaler = flowslatest.FlowCollectorHPA{
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
				}
			})
		})

		It("Should have HPA installed", func() {
			By("Expecting HPA to be created")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKeyKafkaTransformer, &hpa)
			}, timeout, interval).Should(Succeed())
			Expect(*hpa.Spec.MinReplicas).To(Equal(int32(1)))
			Expect(hpa.Spec.MaxReplicas).To(Equal(int32(1)))
			Expect(*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization).To(Equal(int32(90)))
		})

		It("Should autoscale when the HPA options change", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor.KafkaConsumerAutoscaler.MinReplicas = ptr.To(int32(2))
				fc.Spec.Processor.KafkaConsumerAutoscaler.MaxReplicas = 2
			})

			By("Changing the Horizontal Pod Autoscaler instance")
			Eventually(func() error {
				if err := k8sClient.Get(ctx, flpKeyKafkaTransformer, &hpa); err != nil {
					return err
				}
				if *hpa.Spec.MinReplicas != int32(2) || hpa.Spec.MaxReplicas != int32(2) ||
					*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != int32(90) {
					return fmt.Errorf("expected {2, 2, 90}: Got %v, %v, %v",
						*hpa.Spec.MinReplicas, hpa.Spec.MaxReplicas,
						*hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Back without Kafka", func() {
		It("Should remove kafka config successfully", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.DeploymentModel = flowslatest.DeploymentModelDirect
			})
		})

		It("Should deploy single flp again", func() {
			By("Expecting daemonset to be created")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKey1, &appsv1.DaemonSet{})
			}, timeout, interval).Should(Succeed())
		})

		It("Should delete kafka transformer", func() {
			By("Expecting transformer deployment to be deleted")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpKeyKafkaTransformer, &appsv1.Deployment{})
			}, timeout, interval).Should(MatchError(`deployments.apps "flowlogs-pipeline-transformer" not found`))
		})
	})

	Context("Checking monitoring resources", func() {
		It("Should create desired objects when they're not found (e.g. case of an operator upgrade)", func() {
			psvc := v1.Service{}
			sm := monitoringv1.ServiceMonitor{}
			pr := monitoringv1.PrometheusRule{}
			By("Expecting prometheus service to exist")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-prom",
					Namespace: operatorNamespace,
				}, &psvc)
			}, timeout, interval).Should(Succeed())

			By("Expecting ServiceMonitor to exist")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-monitor",
					Namespace: operatorNamespace,
				}, &sm)
			}, timeout, interval).Should(Succeed())

			By("Expecting PrometheusRule to exist and be updated")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-alert",
					Namespace: operatorNamespace,
				}, &pr)
			}, timeout, interval).Should(Succeed())
			Expect(pr.Spec.Groups).Should(HaveLen(1))
			Expect(pr.Spec.Groups[0].Rules).Should(HaveLen(9))

			// Manually delete ServiceMonitor
			By("Deleting ServiceMonitor")
			Eventually(func() error {
				return k8sClient.Delete(ctx, &sm)
			}, timeout, interval).Should(Succeed())

			// Do a dummy change that will trigger reconcile, and make sure SM is created again
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor.LogLevel = "trace"
			})
			By("Expecting ServiceMonitor to exist")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-monitor",
					Namespace: operatorNamespace,
				}, &sm)
			}, timeout, interval).Should(Succeed())

			// Manually delete Rule
			By("Deleting prom rule")
			Eventually(func() error {
				return k8sClient.Delete(ctx, &pr)
			}, timeout, interval).Should(Succeed())

			// Do a dummy change that will trigger reconcile, and make sure Rule is created again
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Processor.LogLevel = "debug"
			})
			By("Expecting PrometheusRule to exist")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-alert",
					Namespace: operatorNamespace,
				}, &pr)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Using certificates with loki manual mode", func() {
		flpDS := appsv1.DaemonSet{}
		It("Should update Loki to use TLS", func() {
			// Create CM certificate
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "loki-ca",
					Namespace: operatorNamespace,
				},
				Data: map[string]string{"ca.crt": "certificate data"},
			}
			cleanupList = append(cleanupList, cm)
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeManual
				fc.Spec.Loki.Manual.TLS = flowslatest.ClientTLS{
					Enable: true,
					CACert: flowslatest.CertificateReference{
						Type:     flowslatest.RefTypeConfigMap,
						Name:     "loki-ca",
						CertFile: "ca.crt",
					},
				}
			})
		})

		It("Should have certificate mounted", func() {
			By("Expecting certificate mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(2))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
			Expect(flpDS.Spec.Template.Spec.Volumes[1].Name).To(Equal("loki-certs-ca"))
		})

		It("Should restore no TLS config", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Manual.TLS = flowslatest.ClientTLS{
					Enable: false,
				}
			})
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(1))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
		})
	})

	Context("Using certificates with loki distributed mode", func() {
		flpDS := appsv1.DaemonSet{}
		It("Should update Loki to use TLS", func() {
			// Create CM certificate
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "loki-distri-ca",
					Namespace: operatorNamespace,
				},
				Data: map[string]string{"ca.crt": "certificate data"},
			}
			cleanupList = append(cleanupList, cm)
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeMicroservices
				fc.Spec.Loki.Microservices = flowslatest.LokiMicroservicesParams{
					IngesterURL: "http://loki-ingested:3100/",
					QuerierURL:  "http://loki-queries:3100/",
					TLS: flowslatest.ClientTLS{
						Enable: true,
						CACert: flowslatest.CertificateReference{
							Type:     flowslatest.RefTypeConfigMap,
							Name:     "loki-distri-ca",
							CertFile: "ca.crt",
						},
					},
				}
			})
		})

		It("Should have certificate mounted", func() {
			By("Expecting certificate mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(2))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
			Expect(flpDS.Spec.Template.Spec.Volumes[1].Name).To(Equal("loki-certs-ca"))
		})

		It("Should restore no TLS config", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Microservices.TLS = flowslatest.ClientTLS{
					Enable: false,
				}
			})
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(1))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
		})
	})

	Context("Using certificates with loki monolithic mode", func() {
		flpDS := appsv1.DaemonSet{}
		It("Should update Loki to use TLS", func() {
			// Create CM certificate
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "loki-mono-ca",
					Namespace: operatorNamespace,
				},
				Data: map[string]string{"ca.crt": "certificate data"},
			}
			cleanupList = append(cleanupList, cm)
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeMonolithic
				fc.Spec.Loki.Monolithic = flowslatest.LokiMonolithParams{
					InstallDemoLoki: ptr.To(false),
					URL:             "http://loki-mono:3100/",
					TLS: flowslatest.ClientTLS{
						Enable: true,
						CACert: flowslatest.CertificateReference{
							Type:     flowslatest.RefTypeConfigMap,
							Name:     "loki-mono-ca",
							CertFile: "ca.crt",
						},
					},
				}
			})
		})

		It("Should have certificate mounted", func() {
			By("Expecting certificate mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(2))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
			Expect(flpDS.Spec.Template.Spec.Volumes[1].Name).To(Equal("loki-certs-ca"))
		})

		It("Should restore no TLS config", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Monolithic.TLS = flowslatest.ClientTLS{
					Enable: false,
				}
			})
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(1))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
		})
	})

	Context("Using Certificates With Loki in LokiStack Mode", func() {
		flpDS := appsv1.DaemonSet{}
		It("Should update Loki config successfully", func() {
			// Create CM certificate
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "lokistack-gateway-ca-bundle",
					Namespace: operatorNamespace,
				},
				Data: map[string]string{"service-ca.crt": "certificate data"},
			}
			cleanupList = append(cleanupList, cm)
			Expect(k8sClient.Create(ctx, cm)).Should(Succeed())
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeLokiStack
				fc.Spec.Loki.LokiStack = flowslatest.LokiStackRef{
					Name:      "lokistack",
					Namespace: operatorNamespace,
				}
			})
		})

		It("Should have certificate mounted", func() {
			By("Expecting certificate mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(3))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
			Expect(flpDS.Spec.Template.Spec.Volumes[1].Name).To(Equal("flowlogs-pipeline"))
			Expect(flpDS.Spec.Template.Spec.Volumes[2].Name).To(Equal("loki-certs-ca"))
		})

		It("Should deploy Loki roles", func() {
			By("Expecting FLP Writer ClusterRoleBinding")
			Eventually(func() interface{} {
				var crb rbacv1.ClusterRoleBinding
				return k8sClient.Get(ctx, rbKeyLokiWriterMono, &crb)
			}, timeout, interval).Should(Succeed())
		})

		It("Should restore no TLS config in manual mode", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				fc.Spec.Loki.Mode = flowslatest.LokiModeManual
				fc.Spec.Loki.Manual.TLS = flowslatest.ClientTLS{
					Enable: false,
				}
			})
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey1, &flpDS); err != nil {
					return err
				}
				return flpDS.Spec.Template.Spec.Volumes
			}, timeout, interval).Should(HaveLen(1))
			Expect(flpDS.Spec.Template.Spec.Volumes[0].Name).To(Equal("config-volume"))
		})
	})

	Context("Checking CR ownership", func() {
		It("Should be garbage collected", func() {
			// Retrieve CR to get its UID
			By("Getting the CR")
			flowCR := getCR(crKey)

			By("Expecting flowlogs-pipeline daemonset to be garbage collected")
			Eventually(func() interface{} {
				d := appsv1.DaemonSet{}
				_ = k8sClient.Get(ctx, flpKey1, &d)
				return &d
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting flowlogs-pipeline service account to be garbage collected")
			Eventually(func() interface{} {
				svcAcc := v1.ServiceAccount{}
				_ = k8sClient.Get(ctx, flpKey1, &svcAcc)
				return &svcAcc
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting flowlogs-pipeline configmap to be garbage collected")
			Eventually(func() interface{} {
				cm := v1.ConfigMap{}
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      "flowlogs-pipeline-config",
					Namespace: operatorNamespace,
				}, &cm)
				return &cm
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

func checkDigestUpdate(oldDigest *string, annots map[string]string) error {
	newDigest := annots[constants.PodConfigurationDigest]
	if newDigest == "" {
		return fmt.Errorf("%q annotation can't be empty", constants.PodConfigurationDigest)
	} else if newDigest == *oldDigest {
		return fmt.Errorf("expect digest to change, but is still %s", *oldDigest)
	}
	*oldDigest = newDigest
	return nil
}
