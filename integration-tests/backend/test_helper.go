package e2etests

import (
	"context"
	"fmt"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// Try delete with retry on webhook timeout
	maxRetries := 3
	var deleteErr error
	for i := 0; i < maxRetries; i++ {
		deleteErr = k8sClient.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
		if deleteErr != nil {
			if apierrors.IsNotFound(deleteErr) {
				return // Already deleted
			}
			if isWebhookTimeoutError(deleteErr) && i < maxRetries-1 {
				e2e.Logf("Webhook timeout deleting namespace %s (attempt %d/%d), retrying in 10s...", ns, i+1, maxRetries)
				time.Sleep(10 * time.Second) // Wait before retry
				continue
			}
			if !isWebhookTimeoutError(deleteErr) {
				o.Expect(deleteErr).NotTo(o.HaveOccurred())
				return
			}
		} else {
			break // Success
		}
	}

	// If all retries failed with webhook timeout, log but continue to verify deletion
	if deleteErr != nil && isWebhookTimeoutError(deleteErr) {
		e2e.Logf("Warning: All %d attempts to delete namespace %s failed with webhook timeout, checking if deletion succeeded anyway", maxRetries, ns)
	}

	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (bool, error) {
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

// isWebhookTimeoutError checks if the error is a webhook timeout error
func isWebhookTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return strings.Contains(errMsg, "webhook") &&
		(strings.Contains(errMsg, "timeout") ||
			strings.Contains(errMsg, "deadline exceeded"))
}
