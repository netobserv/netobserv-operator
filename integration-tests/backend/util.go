package e2etests

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	chars := "abcdefghijklmnopqrstuvwxyz0123456789"
	seed := rand.New(rand.NewSource(time.Now().UnixNano()))
	buffer := make([]byte, 8)
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

func getRouteAddress(oc *exutil.CLI, ns, routeName string) string {
	route, err := oc.AdminRouteClient().RouteV1().Routes(ns).Get(context.Background(), routeName, metav1.GetOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())
	return route.Spec.Host
}

// return the infrastructureName. For example:  anli922-jglp4
func getInfrastructureName(oc *exutil.CLI) string {
	infrastructureName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("infrastructure/cluster", "-o=jsonpath={.status.infrastructureName}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return infrastructureName
}

func (r Resource) applyFromTemplate(oc *exutil.CLI, parameters ...string) error {
	parameters = append(parameters, "-n", r.Namespace)
	file, err := processTemplate(oc, parameters...)
	defer os.Remove(file)
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Can not process %v", parameters))
	output, err := oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", file, "-n", r.Namespace).Output()
	if err != nil {
		return fmt.Errorf("%v", output)
	}
	_ = r.WaitForResourceToAppear(oc)
	return nil
}

func processTemplate(oc *exutil.CLI, parameters ...string) (string, error) {
	var configFile string
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 15*time.Second, false, func(context.Context) (bool, error) {
		output, err := oc.AsAdmin().Run("process").Args(parameters...).OutputToFile(getRandomString() + ".json")
		if err != nil {
			e2e.Logf("the err:%v, and try next round", err)
			return false, nil
		}
		configFile = output
		return true, nil
	})
	return configFile, err
}

// expect: true means we want the resource contain/compare with the expectedContent, false means the resource is expected not to compare with/contain the expectedContent;
// compare: true means compare the expectedContent with the resource content, false means check if the resource contains the expectedContent;
// args are the arguments used to execute command `oc.AsAdmin.WithoutNamespace().Run("get").Args(args...).Output()`;
func checkResource(oc *exutil.CLI, expect, compare bool, expectedContent string, args []string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(args...).Output()
		if err != nil {
			if strings.Contains(output, "NotFound") {
				return false, nil
			}
			return false, err
		}
		if compare {
			res := strings.Compare(output, expectedContent)
			if (res == 0 && expect) || (res != 0 && !expect) {
				return true, nil
			}
			return false, nil
		}
		res := strings.Contains(output, expectedContent)
		if (res && expect) || (!res && !expect) {
			return true, nil
		}
		return false, nil
	})
	if expect {
		compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("The content doesn't match/contain %s", expectedContent))
	} else {
		compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("The %s still exists in the resource", expectedContent))
	}
}

func getResourceGeneration(oc *exutil.CLI, resource, name, ns string) (int, error) {
	gen, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(resource, name, "-o=jsonpath='{.metadata.generation}'", "-n", ns).Output()
	if err != nil {
		return -1, err
	}
	genI, err := strconv.Atoi(strings.Trim(gen, "'"))
	if err != nil {
		return -1, err
	}
	return genI, nil

}

func getResourceVersion(oc *exutil.CLI, resource, name, ns string) (int, error) {
	resV, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(resource, name, "-o=jsonpath='{.metadata.resourceVersion}'", "-n", ns).Output()
	if err != nil {
		return -1, err
	}
	vers, err := strconv.Atoi(strings.Trim(resV, "'"))
	if err != nil {
		return -1, err
	}
	return vers, nil
}

func checkResourceExists(oc *exutil.CLI, resource, name, ns string) (bool, error) {
	stdout, stderr, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(resource, name, "-n", ns).Outputs()
	if err != nil {
		return false, err
	}
	if strings.Contains(stderr, "NotFound") {
		return false, nil
	}
	if strings.Contains(stdout, name) {
		return true, nil
	}
	return false, nil
}

// Assert the status of a resource
func assertResourceStatus(oc *exutil.CLI, kind, name, namespace, jsonpath, exptdStatus string) {
	parameters := []string{kind, name, "-o", "jsonpath=" + jsonpath}
	if namespace != "" {
		parameters = append(parameters, "-n", namespace)
	}
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(parameters...).Output()
		if err != nil {
			return false, err
		}
		if strings.Compare(status, exptdStatus) != 0 {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("%s/%s value for %s is not %s", kind, name, jsonpath, exptdStatus))
}

// For admin user to create resources in the specified namespace from the file (not template)
func ApplyResourceFromFile(oc *exutil.CLI, ns, file string) {
	err := oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", file, "-n", ns).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

func applyResourceFromTemplateByAdmin(oc *exutil.CLI, parameters ...string) error {
	var configFile string
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 15*time.Second, false, func(_ context.Context) (bool, error) {
		output, err := oc.AsAdmin().Run("process").Args(parameters...).OutputToFile(getRandomString() + "resource.json")
		if err != nil {
			e2e.Logf("the err:%v, and try next round", err)
			return false, nil
		}
		configFile = output
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("as admin fail to process %v", parameters))

	e2e.Logf("the file of resource is %s", configFile)
	return oc.WithoutNamespace().AsAdmin().Run("apply").Args("-f", configFile).Execute()
}

// For normal user to create resources in the specified namespace from the file (not template)
func createResourceFromFile(oc *exutil.CLI, ns, file string) {
	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", file, "-n", ns).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

func getSecrets(oc *exutil.CLI, namespace string) (string, error) {
	var secrets string
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (done bool, err error) {
		secrets, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("secrets", "-n", namespace, "-o", "jsonpath='{range .items[*]}{.metadata.name}{\" \"}'").Output()

		if err != nil {
			return false, err
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, "Secrets not available")
	return secrets, err
}

// check if pods with label are fully deleted
func checkPodDeleted(oc *exutil.CLI, ns, label, checkValue string) {
	podCheck := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 240*time.Second, false, func(context.Context) (bool, error) {
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", ns, "-l", label).Output()
		if err != nil || strings.Contains(output, checkValue) {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(podCheck, fmt.Sprintf("Pod \"%s\" exists or not fully deleted", checkValue))
}

func getSAToken(oc *exutil.CLI, name, ns string) string {
	token, err := oc.AsAdmin().WithoutNamespace().Run("create").Args("token", name, "-n", ns).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	return token
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

func (testTemplate *TestServerTemplate) createServer(oc *exutil.CLI) error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "SERVER_NS=" + testTemplate.ServerNS}

	if testTemplate.LargeBlob != "" {
		templateParams = append(templateParams, "-p", "LARGE_BLOB="+testTemplate.LargeBlob)
	}
	if testTemplate.ServiceType != "" {
		templateParams = append(templateParams, "-p", "SERVICE_TYPE="+testTemplate.ServiceType)
	}
	configFile := compat_otp.ProcessTemplate(oc, templateParams...)

	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (testTemplate *TestClientTemplate) createClient(oc *exutil.CLI) error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "SERVER_NS=" + testTemplate.ServerNS, "-p", "CLIENT_NS=" + testTemplate.ClientNS}

	if testTemplate.ObjectSize != "" {
		templateParams = append(templateParams, "-p", "OBJECT_SIZE="+testTemplate.ObjectSize)
	}
	configFile := compat_otp.ProcessTemplate(oc, templateParams...)

	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (testTemplate *TestPingPodsTemplate) createPingPods(oc *exutil.CLI) error {
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
	configFile := compat_otp.ProcessTemplate(oc, templateParams...)

	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
	if err != nil {
		return err
	}
	return nil
}

func waitForResourceGenerationUpdate(oc *exutil.CLI, resource, name, field string, prev int, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		var cur int
		switch field {
		case "generation":
			cur, err = getResourceGeneration(oc, resource, name, ns)
		case "resourceVersion":
			cur, err = getResourceVersion(oc, resource, name, ns)
		}
		if err != nil {
			return false, err
		}
		if cur != prev {
			return true, nil
		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("%s/%s generation did not update", resource, name))
}

func (r Resource) WaitForResourceToAppear(oc *exutil.CLI) error {
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", r.Namespace, r.Kind, r.Name).Output()
		if err != nil {
			msg := fmt.Sprintf("%v", output)
			if strings.Contains(msg, "NotFound") {
				return false, nil
			}
			return false, err
		}
		e2e.Logf("Find %s %s", r.Kind, r.Name)
		return true, nil
	})
	return err
}

func (r Resource) WaitUntilResourceIsGone(oc *exutil.CLI) error {
	err := wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", r.Namespace, r.Kind, r.Name).Output()
		if err != nil {
			errstring := fmt.Sprintf("%v", output)
			if strings.Contains(errstring, "NotFound") || strings.Contains(errstring, "the server doesn't have a resource type") {
				return true, nil
			}
			return true, err
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("can't remove %s/%s in %s project", r.Kind, r.Name, r.Namespace)
	}
	return nil
}

func WaitForPodsReadyWithLabel(oc *exutil.CLI, ns, label string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (done bool, err error) {
		pods, err := oc.AdminKubeClient().CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{LabelSelector: label})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 0 {
			e2e.Logf("Waiting for pod with label %s to appear\n", label)
			return false, nil
		}
		ready := true
		for _, pod := range pods.Items {
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					ready = false
					break
				}
			}
		}
		if !ready {
			e2e.Logf("Waiting for pod with label %s to be ready...\n", label)
		}
		return ready, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("The pod with label %s is not availabile", label))
}

// waitForConfigMapDataInjection waits for a configmap to have its data field populated
// This is useful for waiting on service-ca configmap injection or other dynamic configmap updates
func waitForConfigMapDataInjection(oc *exutil.CLI, namespace, configMapName, dataKey string) {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (bool, error) {
		// Check if .data field is populated (returns {} when empty, populated JSON when injected)
		dataValue, getErr := oc.AsAdmin().WithoutNamespace().Run("get").Args("configmap", configMapName, "-n", namespace, "-o=jsonpath={.data}").Output()
		if getErr != nil {
			e2e.Logf("ConfigMap %s/%s not found yet, will retry: %v", namespace, configMapName, getErr)
			return false, nil
		}
		// Check if data is populated (more than just empty braces "{}")
		if len(dataValue) > 2 {
			e2e.Logf("ConfigMap %s/%s has been populated with data", namespace, configMapName)
			return true, nil
		}
		e2e.Logf("ConfigMap %s/%s exists but data not populated yet, will retry", namespace, configMapName)
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("ConfigMap %s/%s data was not populated within timeout", namespace, configMapName))
}
// WaitForDeploymentPodsToBeReady waits for the specific deployment to be ready
func waitForDeploymentPodsToBeReady(oc *exutil.CLI, namespace, name string) error {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		deployment, err := oc.AdminKubeClient().AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for availability of deployment/%s\n", name)
				return false, nil
			}
			return false, err
		}
		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas && deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas {
			e2e.Logf("Deployment %s available (%d/%d)\n", name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
		return false, nil
	})
	return err
}

func waitForStatefulsetReady(oc *exutil.CLI, namespace, name string) error {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		ss, err := oc.AdminKubeClient().AppsV1().StatefulSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for availability of %s statefulset\n", name)
				return false, nil
			}
			return false, err
		}
		if ss.Status.ReadyReplicas == *ss.Spec.Replicas && ss.Status.UpdatedReplicas == *ss.Spec.Replicas {
			e2e.Logf("statefulset %s available (%d/%d)\n", name, ss.Status.ReadyReplicas, *ss.Spec.Replicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s statefulset (%d/%d)\n", name, ss.Status.ReadyReplicas, *ss.Spec.Replicas)
		return false, nil
	})
	return err
}

// wait until DaemonSet is Ready
func waitUntilDaemonSetReady(oc *exutil.CLI, daemonset, namespace string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		desiredNumber, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("daemonset", daemonset, "-n", namespace, "-o", "jsonpath='{.status.desiredNumberScheduled}'").Output()

		if err != nil {
			// loop until daemonset is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		numberReady, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("daemonset", daemonset, "-n", namespace, "-o", "jsonpath='{.status.numberReady}'").Output()
		if err != nil {
			return false, err
		}
		numberReadyi, err := strconv.Atoi(strings.Trim(numberReady, "'"))
		if err != nil {
			return false, err
		}

		desiredNumberi, err := strconv.Atoi(strings.Trim(desiredNumber, "'"))
		if err != nil {
			return false, err
		}
		if numberReadyi != desiredNumberi {
			return false, nil
		}
		updatedNumber, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("daemonset", daemonset, "-n", namespace, "-o", "jsonpath='{.status.updatedNumberScheduled}'").Output()
		if err != nil {
			return false, err
		}
		updatedNumberi, err := strconv.Atoi(strings.Trim(updatedNumber, "'"))
		if err != nil {
			return false, err
		}
		if updatedNumberi != desiredNumberi {
			return false, nil
		}

		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Daemonset %s did not become Ready", daemonset))
}

// WaitForDeploymentPodsToBeReady waits for the specific deployment to be ready
func WaitForDeploymentPodsToBeReady(oc *exutil.CLI, namespace string, name string) {
	var selectors map[string]string
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, true, func(context.Context) (done bool, err error) {
		deployment, err := oc.AdminKubeClient().AppsV1().Deployments(namespace).Get(context.Background(), name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for deployment/%s to appear\n", name)
				return false, nil
			}
			return false, err
		}
		selectors = deployment.Spec.Selector.MatchLabels
		if deployment.Status.AvailableReplicas == *deployment.Spec.Replicas && deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas {
			e2e.Logf("Deployment %s available (%d/%d)\n", name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
			return true, nil
		}
		e2e.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
		return false, nil
	})
	if err != nil && len(selectors) > 0 {
		var labels []string
		for k, v := range selectors {
			labels = append(labels, k+"="+v)
		}
		label := strings.Join(labels, ",")
		_ = oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", namespace, "-l", label).Execute()
		podStatus, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", namespace, "-l", label, "-ojsonpath={.items[*].status.conditions}").Output()
		containerStatus, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", "-n", namespace, "-l", label, "-ojsonpath={.items[*].status.containerStatuses}").Output()
		e2e.Failf("deployment %s is not ready:\nconditions: %s\ncontainer status: %s", name, podStatus, containerStatus)
	}
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("deployment %s is not available", name))
}

// wait until Deployment is Ready
func waitUntilDeploymentReady(oc *exutil.CLI, deployment, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("deployment", deployment, "-n", ns, "-o", "jsonpath='{.status.conditions[0].type}'").Output()

		if err != nil {
			// loop until deployment is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}

		if strings.Trim(status, "'") != "Available" {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Deployment %s did not become Available", deployment))
}

// verifyDeploymentReplicas waits for and verifies the deployment replica count
// Returns true if verification passes within timeout, false otherwise
// For exact match: verifyDeploymentReplicas(oc, "deploy", "ns", 3, "")
// For numeric comparison: verifyDeploymentReplicas(oc, "deploy", "ns", 5, ">")
// Supported operators: ">", "<", ">=", "<=", "==", "~"
// Calling code should check the result and provide custom message to o.Expect()
func verifyDeploymentReplicas(oc *exutil.CLI, deployment, namespace string, expectedValue int, operator string) bool {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		dep, err := oc.AdminKubeClient().AppsV1().Deployments(namespace).Get(context.Background(), deployment, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				e2e.Logf("Waiting for deployment/%s to be created\n", deployment)
				return false, nil
			}
			e2e.Logf("Error getting deployment %s in namespace %s: %v", deployment, namespace, err)
			return false, nil
		}

		currentReplicas := int(dep.Status.Replicas)
		availableReplicas := int(dep.Status.AvailableReplicas)
		updatedReplicas := int(dep.Status.UpdatedReplicas)

		// Check if replicas match the condition
		var conditionMet bool
		if operator == "" || operator == "==" {
			// Exact match - check all three replica counts
			conditionMet = (currentReplicas == expectedValue && availableReplicas == expectedValue && updatedReplicas == expectedValue)
		} else {
			// Numeric comparison - use available replicas
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

// get pod logs absolute path
func getPodLogs(oc *exutil.CLI, namespace, podname string) (string, error) {
	cargs := []string{"-n", namespace, podname}
	var podLogs string
	var err error

	// add polling as logs could be rotated
	err = wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(_ context.Context) (bool, error) {
		podLogs, err = oc.AsAdmin().WithoutNamespace().Run("logs").Args(cargs...).OutputToFile("podLogs.txt")

		if err != nil {
			e2e.Logf("unable to get the pod (%s) logs", podname)
			return false, err
		}
		podLogsf, err := os.Stat(podLogs)

		if err != nil {
			return false, err
		}
		return podLogsf.Size() > 0, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("%s pod logs were not collected", podname))

	e2e.Logf("pod logs file is %s", podLogs)
	return filepath.Abs(podLogs)
}

// wait until NetworkAttachDefinition is Ready
func checkNAD(oc *exutil.CLI, nad, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		nadOutput, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("net-attach-def", nad, "-n", ns).Output()
		if err != nil {
			// loop until NAD is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		if !strings.Contains(nadOutput, nad) {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Network Attach Definition %s did not become Available", nad))
}

// wait until catalogSource is Ready
func (r Resource) WaitUntilCatSrcReady(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		state, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("catalogsource", r.Name, "-n", r.Namespace, "-o", "jsonpath='{.status.connectionState.lastObservedState}'").Output()
		if err != nil {
			// loop until catalogSource is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}

		if strings.Trim(state, "'") != "READY" {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Catalog Source %s did not become Ready", r.Name))
}

// check resource is fully deleted
func checkResourceDeleted(oc *exutil.CLI, resourceType, resourceName, namespace string) {
	resourceCheck := wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 600*time.Second, false, func(context.Context) (bool, error) {
		output, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args(resourceType, resourceName, "-n", namespace).Output()
		if !strings.Contains(output, "NotFound") {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(resourceCheck, fmt.Sprintf("found %s \"%s\" exist or not fully deleted", resourceType, resourceName))
}

// delete a resource
func deleteResource(oc *exutil.CLI, resourceType, resourceName, namespace string, optionalParameters ...string) {
	cmdArgs := []string{resourceType, resourceName, "-n", namespace}
	cmdArgs = append(cmdArgs, optionalParameters...)
	err := oc.AsAdmin().WithoutNamespace().Run("delete").Args(cmdArgs...).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	checkResourceDeleted(oc, resourceType, resourceName, namespace)
}

// get kubeadmin token of the cluster
func getKubeAdminToken(oc *exutil.CLI, kubeAdminPasswd, serverURL, currentContext string) string {
	longinErr := oc.WithoutNamespace().Run("login").Args("-u", "kubeadmin", "-p", kubeAdminPasswd, serverURL).NotShowInfo().Execute()
	o.Expect(longinErr).NotTo(o.HaveOccurred())
	kubeadminToken, kubeadminTokenErr := oc.WithoutNamespace().Run("whoami").Args("-t").Output()
	o.Expect(kubeadminTokenErr).NotTo(o.HaveOccurred())

	rollbackCtxErr := oc.WithoutNamespace().Run("config").Args("set", "current-context", currentContext).Execute()
	o.Expect(rollbackCtxErr).NotTo(o.HaveOccurred())
	return kubeadminToken
}

// get nginx pod name, IP and client IP
func getClientServerInfo(oc *exutil.CLI, serverNS, clientNS, ipStackType string) (map[string]map[string]string, error) {
	nginxPodName, err := compat_otp.GetAllPodsWithLabel(oc, serverNS, "app=nginx")
	nginxPodIP, _ := getPodIP(oc, serverNS, nginxPodName[0], ipStackType)

	clientPodIP, _ := getPodIP(oc, clientNS, "client", ipStackType)

	serviceIP := getServiceIPv4(oc, serverNS, "nginx-service")

	clientServerMap := map[string]map[string]string{
		"client": {
			"ip":   clientPodIP,
			"name": "client",
		},
		"server": {
			"ip":   nginxPodIP,
			"name": nginxPodName[0],
		},
		"service": {
			"ip":   serviceIP,
			"name": "nginx-service",
		},
	}
	return clientServerMap, err
}

func doAction(oc *exutil.CLI, action string, asAdmin bool, withoutNamespace bool, parameters ...string) (string, error) {
	if asAdmin && withoutNamespace {
		return oc.AsAdmin().WithoutNamespace().Run(action).Args(parameters...).Output()
	}
	if asAdmin && !withoutNamespace {
		return oc.AsAdmin().Run(action).Args(parameters...).Output()
	}
	if !asAdmin && withoutNamespace {
		return oc.WithoutNamespace().Run(action).Args(parameters...).Output()
	}
	if !asAdmin && !withoutNamespace {
		return oc.Run(action).Args(parameters...).Output()
	}
	return "", nil
}

func removeResource(oc *exutil.CLI, asAdmin bool, withoutNamespace bool, parameters ...string) {
	output, err := doAction(oc, "delete", asAdmin, withoutNamespace, parameters...)
	if err != nil && (strings.Contains(output, "NotFound") || strings.Contains(output, "No resources found")) {
		e2e.Logf("the resource is deleted already")
		return
	}
	o.Expect(err).NotTo(o.HaveOccurred())

	err = wait.PollUntilContextTimeout(context.Background(), 3*time.Second, 120*time.Second, false, func(_ context.Context) (bool, error) {
		output, err := doAction(oc, "get", asAdmin, withoutNamespace, parameters...)
		if err != nil && (strings.Contains(output, "NotFound") || strings.Contains(output, "No resources found")) {
			e2e.Logf("the resource is delete successfully")
			return true, nil
		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to delete resource %v", parameters))
}

func execCommandInSpecificPod(oc *exutil.CLI, namespace string, podName string, command string) (string, error) {
	e2e.Logf("The command is: %v", command)
	command1 := []string{"-n", namespace, podName, "--", "bash", "-c", command}
	msg, err := oc.AsAdmin().WithoutNamespace().Run("exec").Args(command1...).Output()
	if err != nil {
		e2e.Logf("Execute command failed with  err:%v  and output is %v.", err, msg)
		return msg, err
	}
	o.Expect(err).NotTo(o.HaveOccurred())
	return msg, nil
}

func checkNetworkType(oc *exutil.CLI) string {
	output, _ := oc.WithoutNamespace().AsAdmin().Run("get").Args("network.operator", "cluster", "-o=jsonpath={.spec.defaultNetwork.type}").Output()
	return strings.ToLower(output)
}

func checkPlatform(oc *exutil.CLI) string {
	output, _ := oc.WithoutNamespace().AsAdmin().Run("get").Args("infrastructure", "cluster", "-o=jsonpath={.status.platformStatus.type}").Output()
	return strings.ToLower(output)
}

func isPlatformSuitableForNMState(oc *exutil.CLI) bool {
	platform := checkPlatform(oc)
	if !strings.Contains(platform, "baremetal") && !strings.Contains(platform, "none") && !strings.Contains(platform, "vsphere") && !strings.Contains(platform, "openstack") {
		e2e.Logf("Skipping for unsupported platform, not baremetal/vsphere/openstack!")
		return false
	}
	return true
}

// Check if BaselineCapabilities have been set
func isBaselineCapsSet(oc *exutil.CLI) bool {
	baselineCapabilitySet, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion", "version", "-o=jsonpath={.spec.capabilities.baselineCapabilitySet}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("baselineCapabilitySet parameters: %v\n", baselineCapabilitySet)
	return len(baselineCapabilitySet) != 0
}

// Check if component is listed in clusterversion.status.capabilities.enabledCapabilities
func isEnabledCapability(oc *exutil.CLI, component string) bool {
	enabledCapabilities, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion", "-o=jsonpath={.items[*].status.capabilities.enabledCapabilities}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("Cluster enabled capability parameters: %v\n", enabledCapabilities)
	return strings.Contains(enabledCapabilities, component)
}

// verifyComponentsDeleted verifies that specified components are NOT present in the output (exact line match)
func verifyComponentsDeleted(componentsOutput string, componentsList []string) {
	outputLines := strings.Split(strings.TrimSpace(componentsOutput), "\n")
	for _, component := range componentsList {
		componentFound := false
		for _, line := range outputLines {
			if strings.TrimSpace(line) == component {
				componentFound = true
				break
			}
		}
		o.Expect(componentFound).Should(o.BeFalse(), fmt.Sprintf("%s should be deleted but was found", component))
	}
}

// verifyComponentsExist verifies that specified components ARE present in the output (exact line match)
func verifyComponentsExist(componentsOutput string, componentsList []string) {
	outputLines := strings.Split(strings.TrimSpace(componentsOutput), "\n")
	for _, component := range componentsList {
		componentFound := false
		for _, line := range outputLines {
			if strings.TrimSpace(line) == component {
				componentFound = true
				break
			}
		}
		o.Expect(componentFound).Should(o.BeTrue(), fmt.Sprintf("%s should exist but was not found", component))
	}
}
