package e2etests

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	authv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type TestServerTemplate struct {
	ServerNS    string
	LargeBlob   string
	ServiceType string
	Template    string
}

type TestClientTemplate struct {
	ServerNS   string
	ClientNS   string
	ObjectSize string
	Template   string
}

type TestPingPodsTemplate struct {
	ServerNS      string
	ClientNS      string
	ServerPodName string
	ClientPodName string
	PingTargets   string
	Template      string
}

func getRandomString() string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	buffer := make([]byte, 4)
	for index := range buffer {
		buffer[index] = chars[seed.Intn(len(chars))]
	}
	return string(buffer)
}

// contain checks if b is an elememt of a
func contain(a []string, b string) bool {
	for _, c := range a {
		if c == b {
			return true
		}
	}
	return false
}

func getProxyFromEnv() string {
	var proxy string
	if os.Getenv("http_proxy") != "" {
		proxy = os.Getenv("http_proxy")
	} else if os.Getenv("http_proxy") != "" {
		proxy = os.Getenv("https_proxy")
	}
	return proxy
}

func getRouteAddress(ns, routeName string) string {
	obj, err := getDynamicResource("route", routeName, ns)
	o.Expect(err).NotTo(o.HaveOccurred())
	host, found := getNestedField(obj.Object, ".spec.host")
	o.Expect(found).To(o.BeTrue())
	o.Expect(host).NotTo(o.BeEmpty())
	return host
}

// return the infrastructureName. For example:  anli922-jglp4
func getInfrastructureName() string {
	obj, err := getDynamicResource("infrastructures", "cluster", "")
	o.Expect(err).NotTo(o.HaveOccurred())
	name, found := getNestedField(obj.Object, ".status.infrastructureName")
	o.Expect(found).To(o.BeTrue())
	return name
}

func (r Resource) applyFromTemplate(parameters ...string) error {
	file, err := processTemplate(r.Namespace, parameters...)
	defer os.Remove(file)
	if err != nil {
		return fmt.Errorf("can not process %v: %w", parameters, err)
	}

	cmd := exec.Command("oc", "apply", "-f", file, "-n", r.Namespace)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	err = r.WaitForResourceToAppear()
	if err != nil {
		return fmt.Errorf("resource did not appear: %w", err)
	}
	return nil
}

// expect: true means we want the resource contain/compare with the expectedContent, false means the resource is expected not to compare with/contain the expectedContent;
// compare: true means compare the expectedContent with the resource content, false means check if the resource contains the expectedContent;
// args are the arguments used to execute command `oc.AsAdmin.WithoutNamespace().Run("get").Args(args...).Output()`;
func checkResource(expect, compare bool, expectedContent string, resourceType, resourceName, jsonPath string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		obj, getErr := getDynamicResource(resourceType, resourceName, "")
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return false, nil
			}
			e2e.Logf("checkResource error: resource=%s/%s, err=%v", resourceType, resourceName, getErr)
			return false, getErr
		}

		value, found := getNestedField(obj.Object, jsonPath)
		if !found || value == "" {
			return false, nil
		}

		if compare {
			res := strings.Compare(value, expectedContent)
			if (res == 0 && expect) || (res != 0 && !expect) {
				return true, nil
			}
			return false, nil
		}
		res := strings.Contains(value, expectedContent)
		if (res && expect) || (!res && !expect) {
			return true, nil
		}
		return false, nil
	})
	if expect {
		assertWaitPollNoErr(err, fmt.Sprintf("The content doesn't match/contain %s", expectedContent))
	} else {
		assertWaitPollNoErr(err, fmt.Sprintf("The %s still exists in the resource", expectedContent))
	}
}

func getResourceGeneration(resource, name, ns string) (int, error) {
	obj, err := getDynamicResource(resource, name, ns)
	if err != nil {
		return -1, err
	}
	return int(obj.GetGeneration()), nil
}

func getResourceVersion(resource, name, ns string) (int, error) {
	obj, err := getDynamicResource(resource, name, ns)
	if err != nil {
		return -1, err
	}
	var rv int
	_, err = fmt.Sscanf(obj.GetResourceVersion(), "%d", &rv)
	if err != nil {
		return -1, err
	}
	return rv, nil
}

func checkResourceExists(resource, name, ns string) (bool, error) {
	_, err := getDynamicResource(resource, name, ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Assert the status of a resource
func assertResourceStatus(kind, name, namespace, jsonpath, exptdStatus string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		obj, getErr := getDynamicResource(kind, name, namespace)
		if getErr != nil {
			return false, getErr
		}

		status, found := getNestedField(obj.Object, jsonpath)
		if !found || status == "" {
			return false, nil
		}

		if status != exptdStatus {
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("%s/%s value for %s is not %s", kind, name, jsonpath, exptdStatus))
}

// For admin user to create resources in the specified namespace from the file (not template)
func ApplyResourceFromFile(ns, file string) {
	args := []string{"apply", "-f", file}
	if ns != "" {
		args = append(args, "-n", ns)
	}
	cmd := exec.Command("oc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		o.Expect(err).NotTo(o.HaveOccurred(), string(output))
	}
}

// For normal user to create resources in the specified namespace from the file (not template)
func createResourceFromFile(ns, file string) {
	args := []string{"create", "-f", file}
	if ns != "" {
		args = append(args, "-n", ns)
	}
	cmd := exec.Command("oc", args...)
	err := cmd.Run()
	o.Expect(err).NotTo(o.HaveOccurred())
}

func getSecrets(namespace string) (string, error) {
	ctx := context.Background()
	var secrets string
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 360*time.Second, false, func(ctx context.Context) (done bool, err error) {
		secretList, listErr := k8sClient.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
		if listErr != nil {
			return false, listErr
		}

		var names []string
		for _, s := range secretList.Items {
			names = append(names, s.Name)
		}
		secrets = strings.Join(names, " ")
		return true, nil
	})
	o.Expect(err).NotTo(o.HaveOccurred(), "Secrets not available")
	return secrets, err
}

// check if pods with label are fully deleted
func checkPodDeleted(ns, label, checkValue string) {
	ctx := context.Background()
	podCheck := wait.PollUntilContextTimeout(ctx, 5*time.Second, 240*time.Second, false, func(ctx context.Context) (bool, error) {
		pods, err := k8sClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: label})
		if err != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			return true, nil
		}

		for _, pod := range pods.Items {
			if strings.Contains(pod.Name, checkValue) {
				return false, nil
			}
		}
		return true, nil
	})
	assertWaitPollNoErr(podCheck, fmt.Sprintf("Pod \"%s\" exists or not fully deleted", checkValue))
}

func getSAToken(name, ns string) string {
	tokenReq, err := k8sClient.CoreV1().ServiceAccounts(ns).CreateToken(
		context.Background(), name, &authv1.TokenRequest{}, metav1.CreateOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())
	return tokenReq.Status.Token
}

func doHTTPRequest(header http.Header, address, path, query, method string, quiet bool, attempts int, requestBody io.Reader, expectedStatusCode int) ([]byte, error) {
	us, err := buildURL(address, path, query)
	if err != nil {
		return nil, err
	}
	if !quiet {
		e2e.Logf("%s", us)
	}

	req, err := http.NewRequest(strings.ToUpper(method), us, requestBody)
	if err != nil {
		return nil, err
	}

	req.Header = header

	var tr *http.Transport
	proxy := getProxyFromEnv()
	if len(proxy) > 0 {
		proxyURL, err := url.Parse(proxy)
		o.Expect(err).NotTo(o.HaveOccurred())
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			Proxy:           http.ProxyURL(proxyURL),
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client := &http.Client{Transport: tr}

	var resp *http.Response
	success := false

	for attempts > 0 {
		attempts--

		resp, err = client.Do(req)
		if err != nil {
			e2e.Logf("error sending request %v", err)
			continue
		}
		if resp.StatusCode != expectedStatusCode {
			buf, _ := io.ReadAll(resp.Body) // nolint
			e2e.Logf("Error response from server: %s %s (%v), attempts remaining: %d", resp.Status, string(buf), err, attempts)
			if err := resp.Body.Close(); err != nil {
				e2e.Logf("error closing body %v", err)
			}
			continue
		}
		success = true
		break
	}
	if !success {
		return nil, fmt.Errorf("run out of attempts while querying the server")
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			e2e.Logf("error closing body %v", err)
		}
	}()
	return io.ReadAll(resp.Body)
}

func (testTemplate *TestServerTemplate) createServer() error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "SERVER_NS=" + testTemplate.ServerNS}

	if testTemplate.LargeBlob != "" {
		templateParams = append(templateParams, "-p", "LARGE_BLOB="+testTemplate.LargeBlob)
	}
	if testTemplate.ServiceType != "" {
		templateParams = append(templateParams, "-p", "SERVICE_TYPE="+testTemplate.ServiceType)
	}

	return applyResourceFromTemplateByAdmin(templateParams...)
}

func (testTemplate *TestClientTemplate) createClient() error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "SERVER_NS=" + testTemplate.ServerNS, "-p", "CLIENT_NS=" + testTemplate.ClientNS}

	if testTemplate.ObjectSize != "" {
		templateParams = append(templateParams, "-p", "OBJECT_SIZE="+testTemplate.ObjectSize)
	}

	return applyResourceFromTemplateByAdmin(templateParams...)
}

func (testTemplate *TestPingPodsTemplate) createPingPods() error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "SERVER_NS=" + testTemplate.ServerNS, "-p", "CLIENT_NS=" + testTemplate.ClientNS}

	if testTemplate.ServerPodName != "" {
		templateParams = append(templateParams, "-p", "SERVER_POD_NAME="+testTemplate.ServerPodName)
	}
	if testTemplate.ClientPodName != "" {
		templateParams = append(templateParams, "-p", "CLIENT_POD_NAME="+testTemplate.ClientPodName)
	}
	if testTemplate.PingTargets != "" {
		templateParams = append(templateParams, "-p", "PING_TARGETS="+testTemplate.PingTargets)
	}

	return applyResourceFromTemplateByAdmin(templateParams...)
}

func waitForResourceGenerationUpdate(resource, name, field string, prev int, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		var cur int
		switch field {
		case "generation":
			cur, err = getResourceGeneration(resource, name, ns)
		case "resourceVersion":
			cur, err = getResourceVersion(resource, name, ns)
		}
		if err != nil {
			return false, err
		}
		if cur != prev {
			return true, nil
		}
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("%s/%s generation did not update", resource, name))
}

func (r Resource) WaitForResourceToAppear() error {
	return wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		_, getErr := getDynamicResource(r.Kind, r.Name, r.Namespace)
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return false, nil
			}
			return false, getErr
		}
		e2e.Logf("Find %s %s", r.Kind, r.Name)
		return true, nil
	})
}

func (r Resource) WaitUntilResourceIsGone() error {
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		_, getErr := getDynamicResource(r.Kind, r.Name, r.Namespace)
		if getErr != nil {
			// Resource is gone if NotFound
			if apierrors.IsNotFound(getErr) {
				return true, nil
			}
			// Other errors should be retried
			return false, nil
		}
		// Resource still exists
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("can't remove %s/%s in %s project", r.Kind, r.Name, r.Namespace)
	}
	return nil
}

func WaitForPodsReadyWithLabel(ns, label string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(ctx context.Context) (done bool, err error) {
		pods, listErr := k8sClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: label})
		if listErr != nil {
			return false, nil
		}

		if len(pods.Items) == 0 {
			e2e.Logf("Waiting for pod with label %s to appear\n", label)
			return false, nil
		}

		for i := range pods.Items {
			ready := false
			for _, cond := range pods.Items[i].Status.Conditions {
				if cond.Type == corev1.PodReady {
					ready = cond.Status == corev1.ConditionTrue
					break
				}
			}
			if !ready {
				e2e.Logf("Waiting for pod with label %s to be ready...\n", label)
				return false, nil
			}
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("The pod with label %s is not availabile", label))
}

// WaitForAllPodsReady waits for all pods in a namespace to be ready, excluding completed pods
func WaitForAllPodsReady(namespace string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 4*time.Minute, false, func(ctx context.Context) (done bool, err error) {
		pods, listErr := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if listErr != nil {
			e2e.Logf("error listing pods: %v, will retry", listErr)
			return false, nil
		}
		if len(pods.Items) == 0 {
			e2e.Logf("No pods found in namespace %s yet, will retry", namespace)
			return false, nil
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					ready = cond.Status == corev1.ConditionTrue
					break
				}
			}
			if !ready {
				return false, nil
			}
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Some Pods are not ready in NS %s!", namespace))
}

// waitForConfigMapDataInjection waits for a configmap to have its data field populated
// This is useful for waiting on service-ca configmap injection or other dynamic configmap updates
func waitForConfigMapDataInjection(namespace, configMapName, dataKey string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 180*time.Second, false, func(ctx context.Context) (bool, error) {
		cm, getErr := k8sClient.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
		if getErr != nil {
			e2e.Logf("ConfigMap %s/%s not found yet, will retry: %v", namespace, configMapName, getErr)
			return false, nil
		}
		if len(cm.Data) > 0 {
			e2e.Logf("ConfigMap %s/%s has been populated with data", namespace, configMapName)
			return true, nil
		}
		e2e.Logf("ConfigMap %s/%s exists but data not populated yet, will retry", namespace, configMapName)
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("ConfigMap %s/%s data was not populated within timeout", namespace, configMapName))
}

// WaitForDeploymentPodsToBeReady waits for the specific deployment to be ready
func waitForDeploymentPodsToBeReady(namespace, name string) error {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 180*time.Second, false, func(ctx context.Context) (done bool, err error) {
		deploy, getErr := k8sClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				e2e.Logf("Waiting for availability of deployment/%s\n", name)
				return false, nil
			}
			return false, getErr
		}

		specReplicas := int32(1)
		if deploy.Spec.Replicas != nil {
			specReplicas = *deploy.Spec.Replicas
		}
		available := deploy.Status.AvailableReplicas
		updated := deploy.Status.UpdatedReplicas

		if available == specReplicas && updated == specReplicas {
			e2e.Logf("Deployment %s available (%d/%d)\n", name, available, specReplicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, available, specReplicas)
		return false, nil
	})
	return err
}

func waitForStatefulsetReady(namespace, name string) error {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 180*time.Second, false, func(ctx context.Context) (done bool, err error) {
		sts, getErr := k8sClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				e2e.Logf("Waiting for availability of %s statefulset\n", name)
				return false, nil
			}
			return false, getErr
		}

		specReplicas := int32(1)
		if sts.Spec.Replicas != nil {
			specReplicas = *sts.Spec.Replicas
		}
		ready := sts.Status.ReadyReplicas
		updated := sts.Status.UpdatedReplicas

		if ready == specReplicas && updated == specReplicas {
			e2e.Logf("statefulset %s available (%d/%d)\n", name, ready, specReplicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s statefulset (%d/%d)\n", name, ready, specReplicas)
		return false, nil
	})
	return err
}

// waitForServiceEndpoints waits for a service to have available endpoints
func waitForServiceEndpoints(namespace, serviceName string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 360*time.Second, false, func(ctx context.Context) (done bool, err error) {
		ep, getErr := k8sClient.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				e2e.Logf("Waiting for endpoints for service %s to appear\n", serviceName)
				return false, nil
			}
			return false, getErr
		}
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				e2e.Logf("Service %s has available endpoints\n", serviceName)
				return true, nil
			}
		}
		e2e.Logf("Waiting for service %s to have available endpoints...\n", serviceName)
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Service %s does not have available endpoints", serviceName))
}

// wait until DaemonSet is Ready
func waitUntilDaemonSetReady(daemonset, namespace string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 600*time.Second, false, func(ctx context.Context) (done bool, err error) {
		ds, getErr := k8sClient.AppsV1().DaemonSets(namespace).Get(ctx, daemonset, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				return false, nil
			}
			return false, getErr
		}

		desired := ds.Status.DesiredNumberScheduled
		ready := ds.Status.NumberReady
		updated := ds.Status.UpdatedNumberScheduled

		if ready != desired || updated != desired {
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Daemonset %s did not become Ready", daemonset))
}

// WaitForDeploymentPodsToBeReady waits for the specific deployment to be ready
func WaitForDeploymentPodsToBeReady(namespace string, name string) {
	ctx := context.Background()
	var labelSelector string
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 180*time.Second, true, func(ctx context.Context) (done bool, err error) {
		deploy, getErr := k8sClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				e2e.Logf("Waiting for deployment/%s to appear\n", name)
				return false, nil
			}
			return false, getErr
		}

		specReplicas := int32(1)
		if deploy.Spec.Replicas != nil {
			specReplicas = *deploy.Spec.Replicas
		}
		available := deploy.Status.AvailableReplicas
		updated := deploy.Status.UpdatedReplicas

		if deploy.Spec.Selector != nil {
			var parts []string
			for k, v := range deploy.Spec.Selector.MatchLabels {
				parts = append(parts, k+"="+v)
			}
			labelSelector = strings.Join(parts, ",")
		}

		if available == specReplicas && updated == specReplicas {
			e2e.Logf("Deployment %s available (%d/%d)\n", name, available, specReplicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, available, specReplicas)
		return false, nil
	})
	if err != nil && labelSelector != "" {
		pods, listErr := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
		if listErr == nil {
			var condStr string
			for _, pod := range pods.Items {
				condStr += pod.Name + ":"
				for _, c := range pod.Status.Conditions {
					condStr += fmt.Sprintf("%s=%s ", c.Type, c.Status)
				}
				condStr += "|"
			}
			e2e.Failf("deployment %s is not ready:\npod conditions: %s", name, condStr)
		}
	}
	assertWaitPollNoErr(err, fmt.Sprintf("deployment %s is not available", name))
}

// wait until Deployment is Ready
func waitUntilDeploymentReady(deployment, ns string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 600*time.Second, false, func(ctx context.Context) (done bool, err error) {
		deploy, getErr := k8sClient.AppsV1().Deployments(ns).Get(ctx, deployment, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				return false, nil
			}
			return false, getErr
		}

		for _, cond := range deploy.Status.Conditions {
			if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Deployment %s did not become Available", deployment))
}

// waitForNetworkPolicy waits for a network policy to exist
func waitForNetworkPolicy(namespace, name string, timeoutSeconds int) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return wait.PollUntilContextTimeout(context.Background(), 2*time.Second, timeout, false, func(ctx context.Context) (bool, error) {
		_, err := k8sClient.NetworkingV1().NetworkPolicies(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

// verifyDeploymentReplicas waits for and verifies the deployment replica count
// Returns true if verification passes within timeout, false otherwise
// For exact match: verifyDeploymentReplicas(oc, "deploy", "ns", 3, "")
// For numeric comparison: verifyDeploymentReplicas(oc, "deploy", "ns", 5, ">")
// Supported operators: ">", "<", ">=", "<=", "==", "~"
// Calling code should check the result and provide custom message to o.Expect()
func verifyDeploymentReplicas(deployment, namespace string, expectedValue int, operator string) bool {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 180*time.Second, false, func(ctx context.Context) (done bool, err error) {
		deploy, getErr := k8sClient.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
		if getErr != nil {
			if strings.Contains(getErr.Error(), "not found") {
				e2e.Logf("Waiting for deployment/%s to be created\n", deployment)
				return false, nil
			}
			e2e.Logf("Error getting deployment %s in namespace %s: %v", deployment, namespace, getErr)
			return false, nil
		}

		currentReplicas := int(deploy.Status.Replicas)
		availableReplicas := int(deploy.Status.AvailableReplicas)
		updatedReplicas := int(deploy.Status.UpdatedReplicas)

		var conditionMet bool
		if operator == "" || operator == "==" {
			conditionMet = (currentReplicas == expectedValue && availableReplicas == expectedValue && updatedReplicas == expectedValue)
		} else {
			switch operator {
			case ">":
				conditionMet = availableReplicas > expectedValue && availableReplicas == currentReplicas
			case "<":
				conditionMet = availableReplicas < expectedValue && availableReplicas == currentReplicas
			case ">=":
				conditionMet = availableReplicas >= expectedValue && availableReplicas == currentReplicas
			case "<=":
				conditionMet = availableReplicas <= expectedValue && availableReplicas == currentReplicas
			case "~":
				conditionMet = currentReplicas == expectedValue && availableReplicas == currentReplicas
			default:
				e2e.Logf("Unknown operator: %s", operator)
				return false, nil
			}
		}

		if conditionMet {
			e2e.Logf("Deployment %s replica condition met - spec=%d, available=%d, updated=%d (expected: %s %d)\n",
				deployment, currentReplicas, availableReplicas, updatedReplicas, operator, expectedValue)
			return true, nil
		}

		e2e.Logf("Waiting for deployment %s replica condition (expected: %s %d) - current: spec=%d, available=%d, updated=%d\n",
			deployment, operator, expectedValue, currentReplicas, availableReplicas, updatedReplicas)
		return false, nil
	})

	return err == nil
}

// getAllPods returns all pod names in a namespace
func getAllPods(namespace string) ([]string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found in namespace %s", namespace)
	}
	podNames := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		podNames[i] = pod.Name
	}
	return podNames, nil
}

// getAllPodsWithLabel returns all pod names in a namespace matching a label selector
func getAllPodsWithLabel(namespace, label string) ([]string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}

	podNames := make([]string, len(pods.Items))
	for i, pod := range pods.Items {
		podNames[i] = pod.Name
	}
	return podNames, nil
}

// getPodNodeName returns the node name where a pod is running
func getPodNodeName(namespace, podName string) (string, error) {
	pod, err := k8sClient.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

// getFirstWorkerNode returns the first worker node name
func getFirstWorkerNode() (string, error) {
	nodes, err := k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("no worker nodes found")
	}
	return nodes.Items[0].Name, nil
}

// addLabelToNode adds a label to a node
func addLabelToNode(nodeName, label, value string) error {
	patch := fmt.Sprintf(`{"metadata":{"labels":{%q:%q}}}`, label, value)
	_, err := k8sClient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return err
}

// deleteLabelFromNode removes a label from a node
func deleteLabelFromNode(nodeName, label string) error {
	patch := fmt.Sprintf(`{"metadata":{"labels":{%q:null}}}`, label)
	_, err := k8sClient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return err
}

// setNamespacePrivileged sets pod security admission labels to privileged for a namespace
func setNamespacePrivileged(namespace string) error {
	patch := `{"metadata":{"labels":{"pod-security.kubernetes.io/enforce":"privileged","pod-security.kubernetes.io/audit":"privileged","pod-security.kubernetes.io/warn":"privileged","security.openshift.io/scc.podSecurityLabelSync":"false"}}}`
	_, err := k8sClient.CoreV1().Namespaces().Patch(context.Background(), namespace, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return err
}

// getNodeLabelValue gets a specific label value from a node
func getNodeLabelValue(nodeName, labelName string) (string, error) {
	node, err := k8sClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	value, ok := node.Labels[labelName]
	if !ok || value == "" {
		return "", fmt.Errorf("label %s not found on node %s", labelName, nodeName)
	}
	return value, nil
}

// isHypershiftHostedCluster checks if the cluster is a hypershift hosted cluster
func isHypershiftHostedCluster() bool {
	obj, err := getDynamicResource("infrastructures", "cluster", "")
	if err != nil {
		e2e.Logf("error getting infrastructure cluster: %v", err)
		return false
	}

	topology, found := getNestedField(obj.Object, ".status.controlPlaneTopology")
	if !found || topology == "" {
		e2e.Logf("controlPlaneTopology not found")
		return false
	}

	e2e.Logf("controlPlaneTopology is %s", topology)
	return topology == "External"
}

// isTechPreviewNoUpgrade checks if the cluster has TechPreviewNoUpgrade feature set
func isTechPreviewNoUpgrade() bool {
	obj, err := getDynamicResource("featuregate", "cluster", "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}
		e2e.Logf("error getting featuregate cluster: %v", err)
		return false
	}

	featureSet, _ := getNestedField(obj.Object, ".spec.featureSet")
	return featureSet == "TechPreviewNoUpgrade"
}

// getSubscriptionByPackageName finds subscription name by package name (spec.name field)
func getSubscriptionByPackageName(namespace, packageName string) (string, error) {
	gvr, _ := resolveGVR("subscription")
	list, err := k8sDynClient.Resource(gvr).Namespace(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, item := range list.Items {
		specName, _, _ := unstructured.NestedString(item.Object, "spec", "name")
		if specName == packageName {
			return item.GetName(), nil
		}
	}

	return "", fmt.Errorf("subscription with package name %s not found in namespace %s", packageName, namespace)
}

// patchSubscriptionRemoveConfig removes spec.config from a subscription using JSON patch
func patchSubscriptionRemoveConfig(namespace, subscriptionName string) error {
	patch := `[{"op": "remove", "path": "/spec/config"}]`
	return patchDynamicResource("subscription", subscriptionName, namespace, types.JSONPatchType, []byte(patch))
}

// getOAuthHTPasswdSecretName gets the htpasswd secret name from OAuth cluster config
func getOAuthHTPasswdSecretName() (string, error) {
	obj, err := getDynamicResource("oauth", "cluster", "")
	if err != nil {
		return "", err
	}

	providers, found, _ := unstructured.NestedSlice(obj.Object, "spec", "identityProviders")
	if !found || len(providers) == 0 {
		return "", nil
	}
	provider, ok := providers[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid identityProvider format")
	}
	secretName, _, _ := unstructured.NestedString(provider, "htpasswd", "fileData", "name")
	return secretName, nil
}

// createSecretFromFile creates a secret from a file
func createSecretFromFile(namespace, secretName, key, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
		Data:       map[string][]byte{key: data},
	}
	_, err = k8sClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

// extractSecretToFile extracts a secret's data to a file
func extractSecretToFile(namespace, secretName, key, outputPath string) error {
	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, secret.Data[key], 0o600)
}

// updateSecretFromFile updates a secret's data from a file
func updateSecretFromFile(namespace, secretName, key, filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	ctx := context.Background()
	secret, err := k8sClient.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	secret.Data[key] = data
	_, err = k8sClient.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

// patchOAuthAddHTPasswdIdentityProvider adds HTPasswd identity provider to OAuth cluster
func patchOAuthAddHTPasswdIdentityProvider(secretName string) error {
	patch := fmt.Sprintf(`[{"op": "add", "path": "/spec/identityProviders", "value": [{"htpasswd": {"fileData": {"name": "%s"}}, "mappingMethod": "claim", "name": "htpasswd", "type": "HTPasswd"}]}]`, secretName)
	return patchDynamicResource("oauth", "cluster", "", types.JSONPatchType, []byte(patch))
}

// getServerURL gets the Kubernetes API server URL
func getServerURL() (string, error) {
	return k8sRestConfig.Host, nil
}

// getCurrentContext gets the current kubeconfig context name
func getCurrentContext() (string, error) {
	cmd := exec.Command("oc", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// getNodeArchitecture gets the architecture of the first node
func getNodeArchitecture() (string, error) {
	nodes, err := k8sClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(nodes.Items) == 0 {
		return "", fmt.Errorf("no nodes found")
	}
	return nodes.Items[0].Status.NodeInfo.Architecture, nil
}

// getPodNameWithLabel gets the first pod name matching a label selector
func getPodNameWithLabel(namespace, labelSelector string) (string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found with label %s in namespace %s", labelSelector, namespace)
	}
	return pods.Items[0].Name, nil
}

// getServiceClusterIP gets the cluster IP of a service matching a label selector
func getServiceClusterIP(namespace, labelSelector string) (string, error) {
	svcs, err := k8sClient.CoreV1().Services(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", err
	}
	if len(svcs.Items) == 0 {
		return "", fmt.Errorf("no services found with label %s in namespace %s", labelSelector, namespace)
	}
	clusterIP := svcs.Items[0].Spec.ClusterIP
	if clusterIP == "" {
		return "", fmt.Errorf("no services found with label %s in namespace %s", labelSelector, namespace)
	}
	return clusterIP, nil
}

// execInPod executes a command in a pod
func execInPod(namespace, podName string, command []string) (string, error) {
	args := []string{"exec", "-n", namespace, podName, "--"}
	args = append(args, command...)

	cmd := exec.Command("oc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		e2e.Logf("stderr: %s", string(output))
	}

	return strings.TrimSpace(string(output)), err
}

// assertAllPodsToBeReady polls until all pods in namespace are ready
func assertAllPodsToBeReady(namespace string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 4*time.Minute, false, func(ctx context.Context) (bool, error) {
		pods, listErr := k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if listErr != nil {
			e2e.Logf("error listing pods: %v, will retry", listErr)
			return false, nil
		}

		for i := range pods.Items {
			pod := &pods.Items[i]
			if pod.Status.Phase == corev1.PodSucceeded {
				continue
			}
			ready := false
			for _, cond := range pod.Status.Conditions {
				if cond.Type == corev1.PodReady {
					ready = cond.Status == corev1.ConditionTrue
					break
				}
			}
			if !ready {
				return false, nil
			}
		}
		return true, nil
	})
	o.Expect(err).NotTo(o.HaveOccurred())
}

// waitAndGetSpecificPodLogs polls until pod logs contain the filter string, returns logs file path
func waitAndGetSpecificPodLogs(namespace, podName, filter string) (string, error) {
	ctx := context.Background()
	var podLogsPath string

	err := wait.PollUntilContextTimeout(ctx, 20*time.Second, 10*time.Minute, false, func(context.Context) (bool, error) {
		// Get pod logs
		stream, err := k8sClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{}).Stream(ctx)
		if err != nil {
			e2e.Logf("unable to get pod (%s) logs: %v, will retry", podName, err)
			return false, nil
		}
		defer stream.Close()
		logsBytes, err := io.ReadAll(stream)
		if err != nil {
			e2e.Logf("unable to read pod (%s) logs: %v, will retry", podName, err)
			return false, nil
		}

		logsContent := string(logsBytes)

		// Check if filter matches
		if !strings.Contains(logsContent, filter) {
			e2e.Logf("filter %s not found in logs yet, will retry", filter)
			return false, nil
		}

		// Save to temp file
		tmpFile, err := os.CreateTemp("", "podLogs-*.txt")
		if err != nil {
			return false, err
		}
		defer tmpFile.Close()

		_, err = tmpFile.WriteString(logsContent)
		if err != nil {
			return false, err
		}

		podLogsPath = tmpFile.Name()
		return true, nil
	})

	if err != nil {
		return "", err
	}

	absPath, err := filepath.Abs(podLogsPath)
	e2e.Logf("pod logs with filter '%s' saved to %s", filter, absPath)
	return absPath, err
}

// waitForNewPodLogs polls until log lines matching filter appear after the current time.
// Uses SinceTime and TailLines to bound log fetches. Returns the full logs file path.
func waitForNewPodLogs(namespace, podName, filter string) (string, error) {
	sinceTime := metav1.Now()
	e2e.Logf("Waiting for new pod logs since %s with filter %q", sinceTime.Format(time.RFC3339), filter)
	var tailLines int64 = 50

	err := wait.PollUntilContextTimeout(context.Background(), 20*time.Second, 10*time.Minute, false, func(ctx context.Context) (bool, error) {
		stream, err := k8sClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
			SinceTime: &sinceTime,
			TailLines: &tailLines,
		}).Stream(ctx)
		if err != nil {
			e2e.Logf("unable to get pod (%s) logs: %v, will retry", podName, err)
			return false, nil
		}
		defer stream.Close()
		logsBytes, err := io.ReadAll(stream)
		if err != nil {
			e2e.Logf("unable to read pod (%s) logs: %v, will retry", podName, err)
			return false, nil
		}
		if strings.Contains(string(logsBytes), filter) {
			e2e.Logf("Found new log entries matching filter %q", filter)
			return true, nil
		}
		e2e.Logf("Waiting for new log entries matching %q", filter)
		return false, nil
	})
	if err != nil {
		return "", err
	}

	return waitAndGetSpecificPodLogs(namespace, podName, filter)
}

// get pod logs absolute path
func getPodLogs(namespace, podname string) (string, error) {
	ctx := context.Background()
	var logsContent string

	// add polling as logs could be rotated
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 600*time.Second, false, func(_ context.Context) (bool, error) {
		stream, err := k8sClient.CoreV1().Pods(namespace).GetLogs(podname, &corev1.PodLogOptions{}).Stream(ctx)
		if err != nil {
			e2e.Logf("unable to get the pod (%s) logs, retrying: %v", podname, err)
			return false, nil
		}
		defer stream.Close()
		logsBytes, err := io.ReadAll(stream)
		if err != nil {
			e2e.Logf("unable to read the pod (%s) logs: %v", podname, err)
			return false, err
		}
		if len(logsBytes) == 0 {
			return false, nil
		}
		logsContent = string(logsBytes)
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("%s pod logs were not collected", podname))

	return logsContent, nil
}

// wait until NetworkAttachDefinition is Ready
func checkNAD(nad, ns string) {
	ctx := context.Background()
	err := wait.PollUntilContextTimeout(ctx, 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		_, getErr := getDynamicResource("net-attach-def", nad, ns)
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				return false, nil
			}
			return false, getErr
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Network Attach Definition %s did not become Available", nad))
}

// wait until catalogSource is Ready
func (r Resource) WaitUntilCatSrcReady() {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		obj, getErr := getDynamicResource("catalogsource", r.Name, r.Namespace)
		if getErr != nil {
			return false, nil
		}
		state, _ := getNestedField(obj.Object, ".status.connectionState.lastObservedState")
		if state != "READY" {
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Catalog Source %s did not become Ready", r.Name))
}

// check resource is fully deleted
func checkResourceDeleted(resourceType, resourceName, namespace string) {
	ctx := context.Background()
	resourceCheck := wait.PollUntilContextTimeout(ctx, 30*time.Second, 600*time.Second, false, func(context.Context) (bool, error) {
		exists, err := checkResourceExists(resourceType, resourceName, namespace)
		if err != nil {
			return false, err
		}
		if exists {
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(resourceCheck, fmt.Sprintf("found %s \"%s\" exist or not fully deleted", resourceType, resourceName))
}

// delete a resource
func deleteResource(resourceType, resourceName, namespace string, optionalParameters ...string) {
	err := deleteDynamicResource(resourceType, resourceName, namespace)
	o.Expect(err).NotTo(o.HaveOccurred())
	checkResourceDeleted(resourceType, resourceName, namespace)
}

// get kubeadmin token of the cluster
func getKubeAdminToken(kubeAdminPasswd, serverURL, currentContext string) string {

	loginCmd := exec.Command("oc", "login", "-u", "kubeadmin", "-p", kubeAdminPasswd, serverURL, "--insecure-skip-tls-verify=true")
	loginOutput, loginErr := loginCmd.CombinedOutput()
	if loginErr != nil {
		e2e.Logf("oc login failed: %v, output: %s", loginErr, string(loginOutput))
	}
	o.Expect(loginErr).NotTo(o.HaveOccurred(), "oc login failed: %s", string(loginOutput))

	whoamiCmd := exec.Command("oc", "whoami", "-t")
	tokenBytes, tokenErr := whoamiCmd.Output()
	o.Expect(tokenErr).NotTo(o.HaveOccurred(), "oc whoami -t failed")
	kubeadminToken := strings.TrimSpace(string(tokenBytes))

	return kubeadminToken
}

// get nginx pod name, IP and client IP
func getClientServerInfo(serverNS, clientNS, ipStackType string) (map[string]map[string]string, error) {
	nginxPodNames, err := getAllPodsWithLabel(serverNS, "app=nginx")
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(len(nginxPodNames)).To(o.BeNumerically(">", 0))

	nginxPodName := nginxPodNames[0]
	nginxPodIP, _ := getPodIP(serverNS, nginxPodName, ipStackType)

	clientPodIP, _ := getPodIP(clientNS, "client", ipStackType)

	serviceIP := getServiceIPv4(serverNS, "nginx-service")

	clientServerMap := map[string]map[string]string{
		"client": {
			"ip":   clientPodIP,
			"name": "client",
		},
		"server": {
			"ip":   nginxPodIP,
			"name": nginxPodName,
		},
		"service": {
			"ip":   serviceIP,
			"name": "nginx-service",
		},
	}
	return clientServerMap, err
}

func removeResource(asAdmin bool, withoutNamespace bool, parameters ...string) {
	ctx := context.Background()

	// Parse parameters: first is resource type, second is resource name, rest are optional flags like -n namespace
	if len(parameters) < 2 {
		e2e.Logf("invalid parameters for removeResource: %v", parameters)
		return
	}

	resourceType := parameters[0]
	resourceName := parameters[1]
	namespace := ""

	// Check for -n namespace flag
	for i := 2; i < len(parameters)-1; i++ {
		if parameters[i] == "-n" {
			namespace = parameters[i+1]
			break
		}
	}

	deleteErr := deleteDynamicResource(resourceType, resourceName, namespace)
	if deleteErr != nil && apierrors.IsNotFound(deleteErr) {
		e2e.Logf("the resource is deleted already")
		return
	}
	o.Expect(deleteErr).NotTo(o.HaveOccurred())

	// Wait for resource to be deleted
	err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 120*time.Second, false, func(_ context.Context) (bool, error) {
		_, getErr := getDynamicResource(resourceType, resourceName, namespace)
		if getErr != nil {
			if apierrors.IsNotFound(getErr) {
				e2e.Logf("the resource is delete successfully")
				return true, nil
			}
			return false, getErr
		}
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to delete resource %v", parameters))
}

func execCommandInSpecificPod(namespace string, podName string, command string, container ...string) (string, error) {
	e2e.Logf("The command is: %v", command)

	args := []string{"exec", "-n", namespace, podName, "--", "bash", "-c", command}

	// Add container name if specified
	if len(container) > 0 && container[0] != "" {
		args = []string{"exec", "-n", namespace, podName, "-c", container[0], "--", "bash", "-c", command}
	}

	cmd := exec.Command("oc", args...)
	output, err := cmd.Output()
	if err != nil {
		e2e.Logf("Execute command failed with err:%v and output is %v.", err, string(output))
		return string(output), err
	}
	return strings.TrimSpace(string(output)), nil
}

// debugNodeWithCommand executes a command on a node using oc debug node
// The command is executed with chroot /host prefix automatically
func debugNodeWithCommand(nodeName, command string) (string, error) {
	// Use oc debug node command via shell
	// The command format matches oc debug node/<nodename> -- chroot /host <command>
	ctx := context.Background()

	args := []string{
		"debug",
		fmt.Sprintf("node/%s", nodeName),
		"-n", "default",
		"--",
		"chroot",
		"/host",
		"bash",
		"-c",
		command,
	}

	cmd := exec.CommandContext(ctx, "oc", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(stdout.String()), fmt.Errorf("%w: stderr: %s", err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

func checkNetworkType() string {
	obj, err := getDynamicResource("network.operator", "cluster", "")
	if err != nil {
		return ""
	}
	networkType, _ := getNestedField(obj.Object, ".spec.defaultNetwork.type")
	return strings.ToLower(networkType)
}

func checkPlatform() string {
	obj, err := getDynamicResource("infrastructures", "cluster", "")
	if err != nil {
		return ""
	}
	platformType, _ := getNestedField(obj.Object, ".status.platformStatus.type")
	return strings.ToLower(platformType)
}

func isPlatformSuitableForNMState() bool {
	platform := checkPlatform()
	if !strings.Contains(platform, "baremetal") && !strings.Contains(platform, "none") && !strings.Contains(platform, "vsphere") && !strings.Contains(platform, "openstack") {
		e2e.Logf("Skipping for unsupported platform, not baremetal/vsphere/openstack!")
		return false
	}
	return true
}

// listManagedResources lists all netobserv-managed resources (services, deployments,
// daemonsets, serviceaccounts, networkpolicies, configmaps, secrets) across all namespaces
// and returns them in "type/name" format matching `oc get -o name` output.
func listManagedResources() ([]string, error) {
	ctx := context.Background()
	selector := metav1.ListOptions{LabelSelector: "netobserv-managed=true"}
	var names []string

	svcs, err := k8sClient.CoreV1().Services("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, s := range svcs.Items {
		names = append(names, "service/"+s.Name)
	}

	deps, err := k8sClient.AppsV1().Deployments("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, d := range deps.Items {
		names = append(names, "deployment.apps/"+d.Name)
	}

	dss, err := k8sClient.AppsV1().DaemonSets("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, d := range dss.Items {
		names = append(names, "daemonset.apps/"+d.Name)
	}

	sas, err := k8sClient.CoreV1().ServiceAccounts("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, s := range sas.Items {
		names = append(names, "serviceaccount/"+s.Name)
	}

	nps, err := k8sClient.NetworkingV1().NetworkPolicies("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, n := range nps.Items {
		names = append(names, "networkpolicy.networking.k8s.io/"+n.Name)
	}

	cms, err := k8sClient.CoreV1().ConfigMaps("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, c := range cms.Items {
		names = append(names, "configmap/"+c.Name)
	}

	secs, err := k8sClient.CoreV1().Secrets("").List(ctx, selector)
	if err != nil {
		return nil, err
	}
	for _, s := range secs.Items {
		names = append(names, "secret/"+s.Name)
	}

	return names, nil
}

// listManagedPods lists all netobserv-managed pods across all namespaces
// and returns them in "pod/name" format.
func listManagedPods() ([]string, error) {
	pods, err := k8sClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{LabelSelector: "netobserv-managed=true"})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, p := range pods.Items {
		names = append(names, "pod/"+p.Name)
	}
	return names, nil
}

// pollVerifyComponentsDeleted polls until all components in deleteList are gone and all in remainList still exist
func pollVerifyComponentsDeleted(deleteList, remainList []string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 180*time.Second, false, func(context.Context) (bool, error) {
		outputLines, getErr := listManagedResources()
		if getErr != nil {
			e2e.Logf("Error getting components: %v", getErr)
			return false, nil
		}
		for _, component := range deleteList {
			for _, line := range outputLines {
				if strings.TrimSpace(line) == component {
					e2e.Logf("Component %s still present, waiting for deletion...", component)
					return false, nil
				}
			}
		}
		for _, component := range remainList {
			found := false
			for _, line := range outputLines {
				if strings.TrimSpace(line) == component {
					found = true
					break
				}
			}
			if !found {
				e2e.Logf("Expected component %s not found yet, retrying...", component)
				return false, nil
			}
		}
		e2e.Logf("All components verified: deleted=%d, remaining=%d", len(deleteList), len(remainList))
		return true, nil
	})
	o.Expect(err).NotTo(o.HaveOccurred(), "timed out waiting for components to be deleted after pause")
}

// patchFlowCollector applies a JSON patch to the cluster flowcollector
func patchFlowCollector(jsonPatch string) error {
	return patchDynamicResource("flowcollector", "cluster", "", types.JSONPatchType, []byte(jsonPatch))
}

// patchFlowCollectorMerge applies a merge patch to the cluster flowcollector
func patchFlowCollectorMerge(mergePatch string) error {
	return patchDynamicResource("flowcollector", "cluster", "", types.MergePatchType, []byte(mergePatch))
}

// scaleDeployment scales a deployment to the specified replica count
func scaleDeployment(namespace, name string, replicas int32) error {
	scale, err := k8sClient.AppsV1().Deployments(namespace).GetScale(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = k8sClient.AppsV1().Deployments(namespace).UpdateScale(context.Background(), name, scale, metav1.UpdateOptions{})
	return err
}

// taintNode adds or updates a taint on a node
func taintNode(nodeName, key, value, effect string) error {
	ctx := context.Background()
	node, err := k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newTaint := corev1.Taint{Key: key, Value: value, Effect: corev1.TaintEffect(effect)}
	updated := false
	for i, t := range node.Spec.Taints {
		if t.Key == key {
			node.Spec.Taints[i] = newTaint
			updated = true
			break
		}
	}
	if !updated {
		node.Spec.Taints = append(node.Spec.Taints, newTaint)
	}
	_, err = k8sClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// removeTaintFromNode removes a taint from a node by key
func removeTaintFromNode(nodeName, key string) error {
	ctx := context.Background()
	node, err := k8sClient.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var filtered []corev1.Taint
	for _, t := range node.Spec.Taints {
		if t.Key != key {
			filtered = append(filtered, t)
		}
	}
	node.Spec.Taints = filtered
	_, err = k8sClient.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// getPodEnvValue gets an environment variable value from the first pod matching a label selector
func getPodEnvValue(namespace, labelSelector, envName string) (string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 || len(pods.Items[0].Spec.Containers) == 0 {
		return "", fmt.Errorf("no pods found with label %s in namespace %s", labelSelector, namespace)
	}
	for _, env := range pods.Items[0].Spec.Containers[0].Env {
		if env.Name == envName {
			return env.Value, nil
		}
	}
	return "", fmt.Errorf("env variable %s not found", envName)
}

// getPodEnvByIndex gets environment variable value by index from the first pod matching a label selector
func getPodEnvByIndex(namespace, labelSelector string, index int) (string, error) {
	pods, err := k8sClient.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 || len(pods.Items[0].Spec.Containers) == 0 {
		return "", fmt.Errorf("no pods found with label %s in namespace %s", labelSelector, namespace)
	}
	envVars := pods.Items[0].Spec.Containers[0].Env
	if index >= len(envVars) {
		return "", fmt.Errorf("env index %d out of range or empty", index)
	}
	return envVars[index].Value, nil
}

// patchSubscription applies a JSON patch to a subscription
func patchSubscription(namespace, name, jsonPatch string) error {
	return patchDynamicResource("subscription", name, namespace, types.JSONPatchType, []byte(jsonPatch))
}

// labelNamespace adds or updates a label on a namespace
func labelNamespace(namespace, key, value string) error {
	patch := fmt.Sprintf(`{"metadata":{"labels":{%q:%q}}}`, key, value)
	_, err := k8sClient.CoreV1().Namespaces().Patch(context.Background(), namespace, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return err
}

// removeLabelFromNamespace removes a label from a namespace
func removeLabelFromNamespace(namespace, key string) error {
	patch := fmt.Sprintf(`{"metadata":{"labels":{%q:null}}}`, key)
	_, err := k8sClient.CoreV1().Namespaces().Patch(context.Background(), namespace, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	return err
}

// assertWaitPollNoErr validates the result of Wait.Poll operations, expecting no error
// Ported from openshift/origin test/extended/util/compat_otp/assert.go
func assertWaitPollNoErr(e error, msg string) {
	if e == nil {
		return
	}
	var err error
	if errors.Is(e, context.DeadlineExceeded) || e.Error() == "timed out waiting for the condition" {
		err = fmt.Errorf("case: %v\nerror: %s", g.CurrentSpecReport().FullText(), msg)
	} else {
		err = fmt.Errorf("case: %v\nerror: %s", g.CurrentSpecReport().FullText(), e.Error())
	}
	o.Expect(err).NotTo(o.HaveOccurred())
}

// isWebhookTimeoutError checks if the error is a webhook timeout error
func isWebhookTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	// Match specific webhook call timeout patterns, not generic webhook rejections
	return (strings.Contains(errMsg, "calling webhook") || strings.Contains(errMsg, "failed calling webhook")) &&
		(strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "deadline exceeded"))
}

// ensureFRREnabled enables FRR via Network operator if not already enabled.
// This deploys frr-k8s DaemonSet and enables route advertisements.
func ensureFRREnabled() error {
	networkOperator, err := getDynamicResource("network.operator", "cluster", "")
	if err != nil {
		return fmt.Errorf("failed to get network operator: %w", err)
	}

	// Check if FRR is already enabled
	providers, found, err := unstructured.NestedStringSlice(networkOperator.Object, "spec", "additionalRoutingCapabilities", "providers")
	if err != nil {
		return fmt.Errorf("failed to get routing capabilities: %w", err)
	}

	frrEnabled := found && contain(providers, "FRR")
	if !frrEnabled {
		e2e.Logf("Enabling FRR via Network operator")
		patch := `{"spec":{"additionalRoutingCapabilities":{"providers":["FRR"]},"defaultNetwork":{"ovnKubernetesConfig":{"routeAdvertisements":"Enabled"}}}}`
		err = patchDynamicResource("network.operator", "cluster", "", types.MergePatchType, []byte(patch))
		if err != nil {
			return fmt.Errorf("failed to patch network operator: %w", err)
		}
	} else {
		e2e.Logf("FRR already enabled, ensuring routeAdvertisements is enabled")
		patch := `{"spec":{"defaultNetwork":{"ovnKubernetesConfig":{"routeAdvertisements":"Enabled"}}}}`
		err = patchDynamicResource("network.operator", "cluster", "", types.MergePatchType, []byte(patch))
		if err != nil {
			return fmt.Errorf("failed to enable routeAdvertisements: %w", err)
		}
	}

	// Wait for frr-k8s namespace to be created
	err = Resource{"namespace", "openshift-frr-k8s", ""}.WaitForResourceToAppear()
	if err != nil {
		return fmt.Errorf("openshift-frr-k8s namespace not created: %w", err)
	}

	// Wait for frr-k8s DaemonSet to be ready
	waitUntilDaemonSetReady("frr-k8s", "openshift-frr-k8s")

	e2e.Logf("FRR enabled successfully")
	return nil
}
