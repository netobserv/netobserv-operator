//nolint:revive,staticcheck
package envtest

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/roles"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

func ControllerSARSpecs(ctxGetter test.ContextGetter) {
	const operatorNamespace = "main-namespace"
	consistentlyTimeout := 2 * time.Second

	var ctx context.Context
	var k8sClient client.Client

	crKey := types.NamespacedName{Name: "cluster"}
	flpKey := types.NamespacedName{
		Name:      constants.FLPName,
		Namespace: operatorNamespace,
	}

	Context("SAR permission checks", Ordered, func() {
		var restoreSAR func()

		BeforeAll(func() {
			restoreSAR = roles.EnableSARChecks(true)
			ctx, k8sClient = ctxGetter()
		})
		AfterAll(func() {
			restoreSAR()
		})

		It("Should fail reconciliation when CRBs are missing", func() {
			Eventually(func() any {
				return k8sClient.Create(ctx, &flowslatest.FlowCollector{
					ObjectMeta: metav1.ObjectMeta{Name: crKey.Name},
					Spec: flowslatest.FlowCollectorSpec{
						Namespace:       operatorNamespace,
						DeploymentModel: flowslatest.DeploymentModelService,
					},
				})
			}, timeout, interval).Should(Succeed())

			By("Expecting FLP deployment to NOT be created (SAR denied)")
			Consistently(func() error {
				return k8sClient.Get(ctx, flpKey, &appsv1.Deployment{})
			}, consistentlyTimeout, interval).Should(MatchError(ContainSubstring("not found")))
		})

		It("Should succeed after installing RBAC", func() {
			By("Installing ClusterRole and ClusterRoleBinding for informers")
			cr := &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "informers"},
				Rules: []rbacv1.PolicyRule{{
					APIGroups: []string{"apps"},
					Resources: []string{"replicasets"},
					Verbs:     []string{"get", "list", "watch"},
				}},
			}
			Expect(k8sClient.Create(ctx, cr)).To(Succeed())

			crb := &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "informers-test"},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "informers",
				},
				Subjects: []rbacv1.Subject{{
					Kind:      "ServiceAccount",
					Name:      constants.FLPName,
					Namespace: operatorNamespace,
				}},
			}
			Expect(k8sClient.Create(ctx, crb)).To(Succeed())

			By("Expecting FLP deployment to be created")
			Eventually(func() error {
				return k8sClient.Get(ctx, flpKey, &appsv1.Deployment{})
			}, timeout, interval).Should(Succeed())
		})

		It("Should clean up", func() {
			test.CleanupCR(ctx, k8sClient, crKey)
			Expect(k8sClient.Delete(ctx, &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "informers-test"},
			})).To(Succeed())
			Expect(k8sClient.Delete(ctx, &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "informers"},
			})).To(Succeed())
		})
	})
}
