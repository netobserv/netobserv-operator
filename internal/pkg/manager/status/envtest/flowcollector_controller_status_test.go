//nolint:revive
package envtest

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

const (
	timeout  = test.Timeout
	interval = test.Interval
)

// nolint:cyclop
func flowCollectorStatusSpecs() {
	namespace := "status-test"
	crKey := types.NamespacedName{
		Name: "cluster",
	}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	Context("Initialize flowcollector and status", func() {
		It("Should create FlowCollector successfully with preset status", func() {
			toCreate := &flowslatest.FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: crKey.Name,
				},
				Spec: flowslatest.FlowCollectorSpec{
					Namespace: namespace,
					Agent: flowslatest.FlowCollectorAgent{
						EBPF: flowslatest.FlowCollectorEBPF{
							// Use EbpfManager in order to trigger a failure (no kind is registered for the type v1alpha1.ClusterBpfApplication)
							Features: []flowslatest.AgentFeature{flowslatest.EbpfManager},
						},
					},
					ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
						Enable: ptr.To(false),
					},
				},
				Status: flowslatest.FlowCollectorStatus{},
			}

			Eventually(func() interface{} {
				return k8sClient.Create(ctx, toCreate)
			}, timeout, interval).Should(Succeed())
		})

		It("Should manually set workloads deployed", func() {
			// envtest won't automatically set workload statuses, hence do it manually
			By("Updating agent")
			Eventually(func() interface{} {
				ds := appsv1.DaemonSet{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "netobserv-ebpf-agent", Namespace: namespace + "-privileged"}, &ds); err != nil {
					return err
				}
				ds.Status.NumberReady = 1
				ds.Status.DesiredNumberScheduled = 1
				return k8sClient.Status().Update(ctx, &ds)
			}, timeout, interval).Should(Succeed())

			for _, name := range []string{"flowlogs-pipeline", "flowlogs-pipeline-informers"} {
				By("Updating " + name)
				Eventually(func() interface{} {
					dep := appsv1.Deployment{}
					if err := k8sClient.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &dep); err != nil {
						return err
					}
					dep.Status.ReadyReplicas = *dep.Spec.Replicas
					dep.Status.Replicas = *dep.Spec.Replicas
					dep.Status.Conditions = []appsv1.DeploymentCondition{
						{
							Type:   appsv1.DeploymentAvailable,
							Status: v1.ConditionTrue,
						},
					}
					return k8sClient.Status().Update(ctx, &dep)
				}, timeout, interval).Should(Succeed())
			}
		})

		It("Should show status errors", func() {
			Eventually(func() interface{} {
				fc := flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, &fc); err != nil {
					return err
				}
				wrongConditions := getWrongConditions(fc.Status.Conditions)
				if len(wrongConditions) != 4 {
					return fmt.Errorf("%d/%d conditions are wrong, expected 4: %v", len(wrongConditions), len(fc.Status.Conditions), wrongConditions)
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})

		It("Should fix FlowCollector", func() {
			Eventually(func() interface{} {
				fc := flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, &fc); err != nil {
					return err
				}
				fc.Spec.Agent.EBPF.Features = []flowslatest.AgentFeature{}
				return k8sClient.Update(ctx, &fc)
			}, timeout, interval).Should(Succeed())

			Eventually(func() interface{} {
				fc := flowslatest.FlowCollector{}
				if err := k8sClient.Get(ctx, crKey, &fc); err != nil {
					return err
				}
				wrongConditions := getWrongConditions(fc.Status.Conditions)
				if len(wrongConditions) > 0 {
					return fmt.Errorf("%d/%d conditions are unexpected: %v", len(wrongConditions), len(fc.Status.Conditions), wrongConditions)
				}
				return nil
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			test.CleanupCR(ctx, k8sClient, crKey)
		})
	})
}

func getWrongConditions(all []metav1.Condition) []metav1.Condition {
	var wrongConditions []metav1.Condition
	for _, cond := range all {
		if cond.Type == "Ready" {
			if cond.Status != metav1.ConditionTrue {
				wrongConditions = append(wrongConditions, cond)
			}
		} else if cond.Status == metav1.ConditionTrue {
			wrongConditions = append(wrongConditions, cond)
		}
	}
	return wrongConditions
}
