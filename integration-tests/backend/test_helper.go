package e2etests

import (
	"context"
	"fmt"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// CLI provides automatic namespace lifecycle management similar to
// openshift-tests-private's compat_otp.NewCLI() pattern.
//
// This helper automatically:
// - Generates a unique test namespace (e2e-netobserv-xxxx) in BeforeEach
// - Cleans up the namespace in AfterEach
//
// Usage:
//
//	var oc *CLI
//
//	g.BeforeEach(func() {
//	    oc = NewCLI()
//	    // oc.Namespace() is now available
//	})
//
// Note: This is a simplified version of the openshift-tests-private CLI helper,
// providing just the namespace lifecycle management without the full CLI wrapper.
type CLI struct {
	namespace string
}

// NewCLI creates a new CLI helper with automatic namespace cleanup.
// Must be called from within a BeforeEach block.
//
// The namespace (e2e-netobserv-xxxx) will be automatically deleted in AfterEach via DeferCleanup.
func NewCLI() *CLI {
	cli := &CLI{
		namespace: generateTestNamespace(),
	}

	g.DeferCleanup(func() {
		deleteNamespace(cli.namespace)
	})

	return cli
}

// SetNamespace updates the CLI's namespace context.
func (c *CLI) SetNamespace(ns string) *CLI {
	c.namespace = ns
	return c
}

// Namespace returns the test namespace.
func (c *CLI) Namespace() string {
	return c.namespace
}

// generateTestNamespace creates a unique test namespace with random suffix.
func generateTestNamespace() string {
	randomSuffix := getRandomString()
	namespace := fmt.Sprintf("e2e-netobserv-%s", randomSuffix)
	err := createNamespace(namespace)
	o.Expect(err).NotTo(o.HaveOccurred())
	return namespace
}

// createNamespace creates a namespace.
func createNamespace(name string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := k8sClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// deleteNamespace deletes a namespace and waits for it to be fully removed.
func deleteNamespace(ns string) {
	if deleteNamespaceEnv == "false" {
		e2e.Logf("Skipping namespace deletion for %s (DELETE_NAMESPACE=false)", ns)
		return
	}

	err := k8sClient.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			o.Expect(err).NotTo(o.HaveOccurred())
		}
		return
	}

	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (bool, error) {
		_, getErr := k8sClient.CoreV1().Namespaces().Get(context.Background(), ns, metav1.GetOptions{})
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return true, nil
			}
			return false, getErr
		}
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Namespace %s is not deleted in 3 minutes", ns))
}

// waitForNetworkPolicy waits for a network policy to exist
func waitForNetworkPolicy(namespace, name string, timeoutSeconds int) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return wait.PollUntilContextTimeout(context.Background(), 2*time.Second, timeout, false, func(context.Context) (bool, error) {
		_, err := k8sClient.NetworkingV1().NetworkPolicies(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// hasEgressPort checks if a network policy has an egress rule with a specific port (supports both string and int ports)
func hasEgressPort(policy *networkingv1.NetworkPolicy, portCheck func(*intstr.IntOrString) bool) bool {
	if policy == nil || policy.Spec.Egress == nil {
		return false
	}
	for _, rule := range policy.Spec.Egress {
		if rule.Ports != nil {
			for _, port := range rule.Ports {
				if port.Port != nil && portCheck(port.Port) {
					return true
				}
			}
		}
	}
	return false
}

// hasIngressPort checks if a network policy has an ingress rule with a specific port
func hasIngressPort(policy *networkingv1.NetworkPolicy, port int32) bool {
	if policy == nil || policy.Spec.Ingress == nil {
		return false
	}
	for _, rule := range policy.Spec.Ingress {
		if rule.Ports != nil {
			for _, p := range rule.Ports {
				if p.Port != nil && p.Port.Type == intstr.Int && p.Port.IntVal == port {
					return true
				}
			}
		}
	}
	return false
}
