//nolint:revive
package controllers

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	. "github.com/netobserv/netobserv-operator/internal/controller/controllerstest"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
)

func flowCollectorEBPFSpecs() {
	// Because the simulated Kube server doesn't manage automatic resource cleanup like an actual Kube would do,
	// we need either to cleanup all created resources manually, or to use different namespaces between tests
	// For simplicity, we'll use a different namespace
	operatorNamespace := "namespace-ebpf-specs"
	agentKey := types.NamespacedName{
		Name:      "netobserv-ebpf-agent",
		Namespace: operatorNamespace + "-privileged",
	}
	crKey := types.NamespacedName{Name: "cluster"}
	saKey := types.NamespacedName{
		Name:      constants.EBPFServiceAccount,
		Namespace: agentKey.Namespace,
	}
	promSvcKey := types.NamespacedName{
		Name:      constants.EBPFAgentMetricsSvcName,
		Namespace: operatorNamespace + "-privileged",
	}
	nsKey := types.NamespacedName{Name: agentKey.Namespace}

	Context("Netobserv eBPF Agent Reconciler", func() {
		It("Should deploy when it does not exist", func() {
			desired := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					DeploymentModel: flowslatest.DeploymentModelDirect,
					Processor: flowslatest.FlowCollectorFLP{
						ImagePullPolicy: "Never",
						LogLevel:        "error",
						Advanced: &flowslatest.AdvancedProcessorConfig{
							Port: ptr.To(int32(9999)),
						},
					},
					Agent: flowslatest.FlowCollectorAgent{
						Type: "eBPF",
						EBPF: flowslatest.FlowCollectorEBPF{
							Sampling:           ptr.To(int32(123)),
							CacheActiveTimeout: "1s",
							CacheMaxFlows:      100,
							Interfaces:         []string{"veth0", "/^br-/"},
							ExcludeInterfaces:  []string{"br-3", "lo"},
							LogLevel:           "trace",
							Advanced: &flowslatest.AdvancedAgentConfig{
								Env: map[string]string{"GOGC": "400", "BUFFERS_LENGTH": "100"},
							},
							Metrics: flowslatest.EBPFMetrics{
								Enable: ptr.To(true),
							},
						},
					},
				},
			}
			Eventually(func() interface{} {
				return k8sClient.Create(ctx, desired)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			ds := appsv1.DaemonSet{}
			By("Expecting to create the netobserv-ebpf-agent DaemonSet")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, agentKey, &ds)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			spec := ds.Spec.Template.Spec
			By("expecting that the netobserv-ebpf-agent daemonset is properly configured")
			Expect(spec.HostNetwork).To(BeTrue())
			Expect(spec.DNSPolicy).To(Equal(v1.DNSClusterFirstWithHostNet))
			Expect(spec.ServiceAccountName).To(Equal(constants.EBPFServiceAccount))
			Expect(len(spec.Containers)).To(Equal(1))
			Expect(spec.Containers[0].SecurityContext.Privileged).To(BeNil())
			Expect(spec.Containers[0].SecurityContext.Capabilities.Add).To(ContainElements(
				[]v1.Capability{"BPF", "PERFMON", "NET_ADMIN"},
			))
			Expect(spec.Containers[0].SecurityContext.RunAsUser).To(Not(BeNil()))
			Expect(*spec.Containers[0].SecurityContext.RunAsUser).To(Equal(int64(0)))
			Expect(spec.Containers[0].Env).To(ContainElements(
				v1.EnvVar{Name: "EXPORT", Value: "grpc"},
				v1.EnvVar{Name: "CACHE_ACTIVE_TIMEOUT", Value: "1s"},
				v1.EnvVar{Name: "CACHE_MAX_FLOWS", Value: "100"},
				v1.EnvVar{Name: "LOG_LEVEL", Value: "trace"},
				v1.EnvVar{Name: "INTERFACES", Value: "veth0,/^br-/"},
				v1.EnvVar{Name: "EXCLUDE_INTERFACES", Value: "br-3,lo"},
				v1.EnvVar{Name: "BUFFERS_LENGTH", Value: "100"},
				v1.EnvVar{Name: "GOGC", Value: "400"},
				v1.EnvVar{Name: "SAMPLING", Value: "123"},
				v1.EnvVar{Name: "TARGET_PORT", Value: "9999"},
			))
			hostFound := false
			for _, env := range spec.Containers[0].Env {
				if env.Name == "TARGET_HOST" {
					if env.ValueFrom == nil ||
						env.ValueFrom.FieldRef == nil ||
						env.ValueFrom.FieldRef.FieldPath != "status.hostIP" {
						Fail(fmt.Sprintf("TARGET_HOST expected to refer to \"status.hostIP\"."+
							" Got: %+v", env.ValueFrom))
					} else {
						hostFound = true
						break
					}
				}
			}
			Expect(hostFound).To(BeTrue(),
				fmt.Sprintf("expected TARGET_HOST env var in %+v", spec.Containers[0].Env))

			ns := v1.Namespace{}
			By("expecting to create the netobserv-privileged namespace")
			Expect(k8sClient.Get(ctx, nsKey, &ns)).To(Succeed())
			Expect(ns.Labels).To(Satisfy(func(labels map[string]string) bool {
				return helper.IsSubSet(ns.Labels, map[string]string{
					"app":                                constants.OperatorName,
					"pod-security.kubernetes.io/enforce": "privileged",
					"pod-security.kubernetes.io/audit":   "privileged",
				})
			}))

			By("expecting to create the netobserv-ebpf-agent service account")
			Expect(k8sClient.Get(ctx, saKey, &v1.ServiceAccount{})).To(Succeed())

			By("Expecting to create the netobserv-ebpf-agent prometheus service")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, promSvcKey, &v1.Service{})
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())
		})

		It("Should update fields that have changed", func() {
			updateCR(crKey, func(fc *flowslatest.FlowCollector) {
				Expect(*fc.Spec.Agent.EBPF.Sampling).To(Equal(int32(123)))
				*fc.Spec.Agent.EBPF.Sampling = 4
				fc.Spec.Agent.EBPF.Privileged = true
				fc.Spec.Agent.EBPF.Metrics.Enable = ptr.To(false)
			})

			ds := appsv1.DaemonSet{}
			By("expecting that the daemonset spec has eventually changed")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, agentKey, &ds); err != nil {
					return err
				}
				expected := v1.EnvVar{Name: "SAMPLING", Value: "4"}
				for _, env := range ds.Spec.Template.Spec.Containers[0].Env {
					if env == expected {
						return nil
					}
				}
				return fmt.Errorf("unexpected env vars: %#v",
					ds.Spec.Template.Spec.Containers[0].Env)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			container := ds.Spec.Template.Spec.Containers[0]
			Expect(container.SecurityContext.Privileged).To(Not(BeNil()))
			Expect(*container.SecurityContext.Privileged).To(BeTrue())
			Expect(container.SecurityContext.Capabilities).To(BeNil())

			By("Expecting to delete the netobserv-ebpf-agent prometheus service")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, promSvcKey, &v1.Service{})
			}).WithTimeout(timeout).WithPolling(interval).Should(MatchError(`services "ebpf-agent-svc-prom" not found`))
		})

		It("Should undeploy everything when deleted", func() {
			// Retrieve CR to get its UID
			flowCR := &flowslatest.FlowCollector{}
			Eventually(func() error {
				return k8sClient.Get(ctx, crKey, flowCR)
			}, timeout, interval).Should(Succeed())

			Expect(k8sClient.Delete(ctx, &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
			})).Should(Succeed())

			By("expecting to delete the flowcollector")
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Name: crKey.Name},
					&flowslatest.FlowCollector{},
				)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Satisfy(errors.IsNotFound))

			By("expecting to delete netobserv-ebpf-agent daemonset")
			Eventually(func() interface{} {
				ds := &appsv1.DaemonSet{}
				if err := k8sClient.Get(ctx, agentKey, ds); err != nil {
					return err
				}
				return ds
			}).WithTimeout(timeout).WithPolling(interval).
				Should(BeGarbageCollectedBy(flowCR))

			By("expecting to delete the netobserv-privileged namespace")
			Eventually(func() interface{} {
				ns := &v1.Namespace{}
				if err := k8sClient.Get(ctx, nsKey, ns); err != nil {
					return err
				}
				return ns
			}).WithTimeout(timeout).WithPolling(interval).
				Should(BeGarbageCollectedBy(flowCR))

			By("expecting to delete the netobserv-ebpf-agent service account")
			Eventually(func() interface{} {
				sa := &v1.ServiceAccount{}
				if err := k8sClient.Get(ctx, saKey, sa); err != nil {
					return err
				}
				return sa
			}).WithTimeout(timeout).WithPolling(interval).
				Should(BeGarbageCollectedBy(flowCR))
		})
	})
}

func flowCollectorEBPFKafkaSpecs() {
	operatorNamespace := "ebpf-kafka-specs"
	agentKey := types.NamespacedName{
		Name:      "netobserv-ebpf-agent",
		Namespace: operatorNamespace + "-privileged",
	}
	crKey := types.NamespacedName{Name: "cluster"}
	flpIngesterKey := types.NamespacedName{
		Name:      constants.FLPName + "-ingester",
		Namespace: operatorNamespace,
	}
	flpTransformerKey := types.NamespacedName{
		Name:      constants.FLPName + "-transformer",
		Namespace: operatorNamespace,
	}
	Context("Netobserv eBPF Agent Reconciler", func() {
		It("Should deploy the agent with the proper configuration", func() {
			descriptor := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace:       operatorNamespace,
					Agent:           flowslatest.FlowCollectorAgent{Type: "eBPF"},
					DeploymentModel: flowslatest.DeploymentModelKafka,
					Kafka: flowslatest.FlowCollectorKafka{
						Address: "kafka-cluster-kafka-bootstrap",
						Topic:   "network-flows",
					},
				},
			}
			Expect(k8sClient.Create(ctx, descriptor)).Should(Succeed())

			ds := appsv1.DaemonSet{}
			By("making sure that the proper environment variables have been passed to the agent")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, agentKey, &ds)
			}).WithTimeout(timeout).WithPolling(interval).Should(Succeed())

			spec := ds.Spec.Template.Spec
			Expect(len(spec.Containers)).To(Equal(1))
			Expect(spec.Containers[0].Env).To(ContainElements(
				v1.EnvVar{Name: "EXPORT", Value: "kafka"},
				v1.EnvVar{Name: "KAFKA_BROKERS", Value: "kafka-cluster-kafka-bootstrap"},
				v1.EnvVar{Name: "KAFKA_TOPIC", Value: "network-flows"},
			))
		})
		It("Should properly deploy flowlogs-pipeline", func() {
			By("deploying flowlogs-pipeline-transformer")
			Eventually(func() interface{} {
				return k8sClient.Get(ctx, flpTransformerKey, &appsv1.Deployment{})
			}, timeout, interval).Should(Succeed())

			By("not deploying flowlogs-pipeline-ingester")
			Expect(k8sClient.Get(ctx, flpIngesterKey, &appsv1.DaemonSet{})).
				Should(Not(Succeed()))
		})
		It("Should correctly undeploy", func() {
			// Retrieve CR to get its UID
			flowCR := &flowslatest.FlowCollector{}
			Eventually(func() error {
				return k8sClient.Get(ctx, crKey, flowCR)
			}, timeout, interval).Should(Succeed())

			Expect(k8sClient.Delete(ctx, &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
			})).Should(Succeed())

			By("expecting to delete the flowcollector")
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Name: crKey.Name},
					&flowslatest.FlowCollector{},
				)
			}).WithTimeout(timeout).WithPolling(interval).
				Should(Satisfy(errors.IsNotFound))

			By("expecting to delete the flowlogs-pipeline-transformer deployment")
			Eventually(func() interface{} {
				dp := &appsv1.Deployment{}
				if err := k8sClient.Get(ctx, flpTransformerKey, dp); err != nil {
					return err
				}
				return dp
			}).WithTimeout(timeout).WithPolling(interval).
				Should(BeGarbageCollectedBy(flowCR))

			By("expecting to delete netobserv-ebpf-agent daemonset")
			Eventually(func() interface{} {
				ds := &appsv1.DaemonSet{}
				if err := k8sClient.Get(ctx, agentKey, ds); err != nil {
					return err
				}
				return ds
			}).WithTimeout(timeout).WithPolling(interval).
				Should(BeGarbageCollectedBy(flowCR))
		})
	})
}
