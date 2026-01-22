//nolint:revive
package cleanup

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/netobserv/network-observability-operator/internal/pkg/test"
)

var (
	ctx          context.Context
	k8sClient    client.Client
	suiteContext *test.SuiteContext
)

func TestCleanup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cleanup Suite")
}

var _ = BeforeSuite(func() {
	// Base path is ".." because we're in internal/pkg/cleanup (3 levels deep)
	// The test framework adds "../.." to basePath, so this resolves to ../../../ from our location
	ctx, k8sClient, suiteContext = test.PrepareEnvTest(nil, []string{}, "..")
})

var _ = AfterSuite(func() {
	test.TeardownEnvTest(suiteContext)
})

var _ = Describe("DeleteAllManagedResources", func() {
	const timeout = 10 * time.Second
	const interval = 250 * time.Millisecond

	var testNamespace string

	BeforeEach(func() {
		// Create unique test namespace for each test to avoid conflicts
		testNamespace = fmt.Sprintf("test-cleanup-ns-%d", time.Now().UnixNano())
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
	})

	AfterEach(func() {
		// Clean up test namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("When managed resources exist", func() {
		It("Should delete resources with netobserv-managed=true label", func() {
			// Create a managed Deployment
			managedDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managed-deployment",
					Namespace: testNamespace,
					Labels: map[string]string{
						"netobserv-managed": "true",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, managedDeployment)).To(Succeed())

			// Create a managed DaemonSet
			managedDaemonSet := &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managed-daemonset",
					Namespace: testNamespace,
					Labels: map[string]string{
						"netobserv-managed": "true",
					},
				},
				Spec: appsv1.DaemonSetSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, managedDaemonSet)).To(Succeed())

			// Create a managed Service
			managedService := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "managed-service",
					Namespace: testNamespace,
					Labels: map[string]string{
						"netobserv-managed": "true",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{Port: 80},
					},
				},
			}
			Expect(k8sClient.Create(ctx, managedService)).To(Succeed())

			// Create a managed ClusterRole
			managedClusterRole := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{
					Name: "managed-clusterrole-cleanup-test",
					Labels: map[string]string{
						"netobserv-managed": "true",
					},
				},
			}
			Expect(k8sClient.Create(ctx, managedClusterRole)).To(Succeed())

			// Verify resources exist before cleanup
			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "managed-deployment",
				Namespace: testNamespace,
			}, d)).To(Succeed())

			// Run cleanup
			err := DeleteAllManagedResources(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			// Verify resources are deleted
			Eventually(func() bool {
				d := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "managed-deployment",
					Namespace: testNamespace,
				}, d)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				ds := &appsv1.DaemonSet{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "managed-daemonset",
					Namespace: testNamespace,
				}, ds)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				s := &corev1.Service{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "managed-service",
					Namespace: testNamespace,
				}, s)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				cr := &rbacv1.ClusterRole{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-clusterrole-cleanup-test",
				}, cr)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())
		})

		It("Should NOT delete resources without netobserv-managed label", func() {
			// Create an unmanaged Deployment
			unmanagedDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "unmanaged-deployment",
					Namespace: testNamespace,
					Labels: map[string]string{
						"app": "other-app",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, unmanagedDeployment)).To(Succeed())

			// Run cleanup
			err := DeleteAllManagedResources(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			// Give some time for any potential deletion
			time.Sleep(1 * time.Second)

			// Verify unmanaged resource still exists
			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "unmanaged-deployment",
				Namespace: testNamespace,
			}, d)).To(Succeed())

			// Cleanup
			Expect(k8sClient.Delete(ctx, unmanagedDeployment)).To(Succeed())
		})

		It("Should NOT delete resources with netobserv-managed=false", func() {
			// Create a resource explicitly marked as not managed
			notManagedDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "not-managed-deployment",
					Namespace: testNamespace,
					Labels: map[string]string{
						"netobserv-managed": "false",
					},
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "test"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "test"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "test", Image: "test:latest"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, notManagedDeployment)).To(Succeed())

			// Run cleanup
			err := DeleteAllManagedResources(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			// Give some time for any potential deletion
			time.Sleep(1 * time.Second)

			// Verify resource still exists
			d := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "not-managed-deployment",
				Namespace: testNamespace,
			}, d)).To(Succeed())

			// Cleanup
			Expect(k8sClient.Delete(ctx, notManagedDeployment)).To(Succeed())
		})

		It("Should handle various resource types", func() {
			// Create multiple types of managed resources
			resources := []client.Object{
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-cm",
						Namespace: testNamespace,
						Labels:    map[string]string{"netobserv-managed": "true"},
					},
				},
				&corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-secret",
						Namespace: testNamespace,
						Labels:    map[string]string{"netobserv-managed": "true"},
					},
				},
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-sa",
						Namespace: testNamespace,
						Labels:    map[string]string{"netobserv-managed": "true"},
					},
				},
				&ascv2.HorizontalPodAutoscaler{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-hpa",
						Namespace: testNamespace,
						Labels:    map[string]string{"netobserv-managed": "true"},
					},
					Spec: ascv2.HorizontalPodAutoscalerSpec{
						ScaleTargetRef: ascv2.CrossVersionObjectReference{
							Kind: "Deployment",
							Name: "test",
						},
						MinReplicas: func() *int32 { v := int32(1); return &v }(),
						MaxReplicas: 10,
					},
				},
				&networkingv1.NetworkPolicy{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "managed-np",
						Namespace: testNamespace,
						Labels:    map[string]string{"netobserv-managed": "true"},
					},
					Spec: networkingv1.NetworkPolicySpec{
						PodSelector: metav1.LabelSelector{},
					},
				},
				&rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "managed-crb-cleanup-test",
						Labels: map[string]string{"netobserv-managed": "true"},
					},
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "test",
					},
				},
			}

			for _, res := range resources {
				Expect(k8sClient.Create(ctx, res)).To(Succeed())
			}

			// Run cleanup
			err := DeleteAllManagedResources(ctx, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			// Verify all resources are deleted
			Eventually(func() bool {
				cm := &corev1.ConfigMap{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-cm", Namespace: testNamespace,
				}, cm)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				s := &corev1.Secret{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-secret", Namespace: testNamespace,
				}, s)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				sa := &corev1.ServiceAccount{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-sa", Namespace: testNamespace,
				}, sa)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				hpa := &ascv2.HorizontalPodAutoscaler{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-hpa", Namespace: testNamespace,
				}, hpa)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				np := &networkingv1.NetworkPolicy{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-np", Namespace: testNamespace,
				}, np)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())

			Eventually(func() bool {
				crb := &rbacv1.ClusterRoleBinding{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: "managed-crb-cleanup-test",
				}, crb)
				return errors.IsNotFound(err)
			}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())
		})
	})
})

var _ = Describe("CleanPastReferences", Ordered, func() {
	const timeout = 10 * time.Second
	const interval = 250 * time.Millisecond
	const testNamespace = "test-past-refs-ns"

	BeforeAll(func() {
		// Create test namespace once for all tests
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Create(ctx, ns)
	})

	AfterAll(func() {
		// Clean up test namespace after all tests
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_ = k8sClient.Delete(ctx, ns)
	})

	Context("When old resources from previous versions exist", func() {
		It("Should delete old ClusterRoleBindings that are no longer used", func() {
			// Create old ClusterRoleBindings that are in the cleanup list
			// These need the netobserv-managed label to be considered "owned" by the operator
			oldBindings := []string{
				"netobserv-plugin",
				"flowlogs-pipeline-ingester-role-mono",
				"flowlogs-pipeline-transformer-role-mono",
				"flowlogs-pipeline-ingester-role",
				"flowlogs-pipeline-transformer-role",
			}

			for _, name := range oldBindings {
				crb := &rbacv1.ClusterRoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name: name,
						Labels: map[string]string{
							"netobserv-managed": "true",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind: "ClusterRole",
						Name: "test-role",
					},
				}
				Expect(k8sClient.Create(ctx, crb)).To(Succeed())
			}

			// Verify they exist
			for _, name := range oldBindings {
				crb := &rbacv1.ClusterRoleBinding{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name: name,
				}, crb)).To(Succeed())
			}

			// Run cleanup - this will set didRun = true
			err := CleanPastReferences(ctx, k8sClient, testNamespace)
			Expect(err).NotTo(HaveOccurred())

			// Verify all old bindings are deleted
			for _, name := range oldBindings {
				Eventually(func() bool {
					crb := &rbacv1.ClusterRoleBinding{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: name,
					}, crb)
					return errors.IsNotFound(err)
				}).WithTimeout(timeout).WithPolling(interval).Should(BeTrue())
			}
		})

		It("Should only run once (idempotent)", func() {
			// Create another old binding after the first cleanup
			// This also needs the netobserv-managed label
			crb := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "flowlogs-pipeline-ingester-role-test",
					Labels: map[string]string{
						"netobserv-managed": "true",
					},
				},
				RoleRef: rbacv1.RoleRef{
					Kind: "ClusterRole",
					Name: "test-role",
				},
			}
			Expect(k8sClient.Create(ctx, crb)).To(Succeed())

			// Run cleanup again - should not run because didRun=true from previous test
			err := CleanPastReferences(ctx, k8sClient, testNamespace)
			Expect(err).NotTo(HaveOccurred())

			// Give some time
			time.Sleep(1 * time.Second)

			// The new binding should still exist because cleanup only runs once
			crb2 := &rbacv1.ClusterRoleBinding{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name: "flowlogs-pipeline-ingester-role-test",
			}, crb2)
			Expect(err).NotTo(HaveOccurred(), "Resource should still exist as cleanup runs only once")

			// Cleanup manually
			_ = k8sClient.Delete(ctx, crb2)
		})
	})
})
