//nolint:revive
package cleanup

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
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
