package e2etests

import (
	"context"
	"fmt"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type TestVMStaticIPTemplate struct {
	Name        string
	Namespace   string
	Mac         string
	StaticIP    string
	NetworkName string
	RunCmd      string
	Template    string
}

type TestVMUDNTemplate struct {
	Name        string
	Namespace   string
	NetworkName string
	RunCmd      string
	Template    string
}

// check if cluster has baremetal workers
func hasMetalWorkerNodes(oc *exutil.CLI) bool {
	workers, err := compat_otp.GetClusterNodesBy(oc, "worker")
	o.Expect(err).NotTo(o.HaveOccurred())
	for _, w := range workers {
		Output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("node", w, "-o", "jsonpath='{.metadata.labels.node\\.kubernetes\\.io/instance-type}'").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if !strings.Contains(Output, "metal") {
			e2e.Logf("Cluster does not have metal worker nodes")
			return false
		}
	}
	return true
}

func isClusterBareMetal(oc *exutil.CLI) (bool, error) {
	output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("infrastructure", "cluster", "-o=jsonpath={.status.platformStatus.type}").Output()
	if err != nil {
		return false, err
	}
	if !strings.Contains(output, "BareMetal") && !strings.Contains(output, "None") {
		return false, nil
	}
	return true, nil
}

// wait until hyperconverged is ready
func waitUntilHyperConvergedReady(oc *exutil.CLI, hc, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("hyperconverged", hc, "-n", ns, "-o", "jsonpath='{.status.conditions[0].status}'").Output()

		if err != nil {
			// loop until hyperconverged is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}

		if strings.Trim(status, "'") != "True" {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("HyperConverged %s did not become Available", hc))
}

func (testTemplate *TestVMStaticIPTemplate) createVMStaticIP(oc *exutil.CLI) error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "NAME=" + testTemplate.Name, "-p", "NAMESPACE=" + testTemplate.Namespace, "-p", "NETWORK_NAME=" + testTemplate.NetworkName, "-p", "MAC=" + testTemplate.Mac, "-p", "STATIC_IP=" + testTemplate.StaticIP}

	if testTemplate.RunCmd != "" {
		templateParams = append(templateParams, "-p", "RUN_CMD="+testTemplate.RunCmd)
	}
	configFile := compat_otp.ProcessTemplate(oc, templateParams...)

	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (testTemplate *TestVMUDNTemplate) createVMUDN(oc *exutil.CLI) error {
	templateParams := []string{"--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "NAME=" + testTemplate.Name, "-p", "NAMESPACE=" + testTemplate.Namespace, "-p", "NETWORK_NAME=" + testTemplate.NetworkName}

	if testTemplate.RunCmd != "" {
		templateParams = append(templateParams, "-p", "RUN_CMD="+testTemplate.RunCmd)
	}
	configFile := compat_otp.ProcessTemplate(oc, templateParams...)

	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
	if err != nil {
		return err
	}
	return nil
}

// wait until virtual machine is Ready
func waitUntilVMReady(oc *exutil.CLI, vm, ns string) {
	err := wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 1200*time.Second, false, func(context.Context) (done bool, err error) {
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("virtualmachine", vm, "-n", ns, "-o", "jsonpath='{.status.conditions[0].status}'").Output()

		if err != nil {
			// loop until virtual machine is found or until timeout
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}

		if strings.Trim(status, "'") != "True" {
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Virtual machine %s did not become Available", vm))
}

// waitForVMIPAssignment waits until the VM has an IP assigned to the specified interface index
func waitForVMIPAssignment(oc *exutil.CLI, vmName, namespace string, interfaceIndex int) (string, error) {
	var vmIP string
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
		ip, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("vmi", vmName, "-n", namespace, fmt.Sprintf("-ojsonpath={.status.interfaces[%d].ipAddress}", interfaceIndex)).Output()
		if err != nil {
			// If VMI not found yet, keep polling
			if strings.Contains(err.Error(), "not found") {
				return false, nil
			}
			return false, err
		}
		// Check if IP is actually assigned (not empty)
		if ip != "" {
			vmIP = ip
			return true, nil
		}
		return false, nil
	})
	return vmIP, err
}
