package e2etests

import (
	"context"
	"fmt"
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
)

// enableSCTPModuleOnNode Manual way to enable sctp in a cluster
func enableSCTPModuleOnNode(oc *exutil.CLI, nodeName, role string) {
	e2e.Logf("This is %s worker node: %s", role, nodeName)
	checkSCTPCmd := "cat /sys/module/sctp/initstate"
	output, err := compat_otp.DebugNodeWithChroot(oc, nodeName, "bash", "-c", checkSCTPCmd)
	var installCmd string
	if err != nil || !strings.Contains(output, "live") {
		e2e.Logf("No sctp module installed, will enable sctp module!!!")
		installCmd = "modprobe sctp"

		// Try 3 times to enable sctp
		o.Eventually(func() error {
			_, installErr := compat_otp.DebugNodeWithChroot(oc, nodeName, "bash", "-c", installCmd)
			if installErr != nil && strings.EqualFold(role, "rhel") {
				e2e.Logf("%v", installErr)
				g.Skip("Yum insall to enable sctp cannot work in a disconnected cluster, skip the test!!!")
			}
			return installErr
		}, "15s", "5s").ShouldNot(o.HaveOccurred(), fmt.Sprintf("Failed to install sctp module on node %s", nodeName))

		// Wait for sctp applied
		o.Eventually(func() string {
			output, err := compat_otp.DebugNodeWithChroot(oc, nodeName, "bash", "-c", checkSCTPCmd)
			if err != nil {
				e2e.Logf("Wait for sctp applied, %v", err)
			}
			return output
		}, "60s", "10s").Should(o.ContainSubstring("live"), fmt.Sprintf("Failed to load sctp module on node %s", nodeName))
	} else {
		e2e.Logf("sctp module is loaded on node %s\n%s", nodeName, output)
	}
}

func prepareSCTPModule(oc *exutil.CLI) {
	nodesOutput, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("node").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	if strings.Contains(nodesOutput, "SchedulingDisabled") || strings.Contains(nodesOutput, "NotReady") {
		g.Skip("There are already some nodes in NotReady or SchedulingDisabled status in cluster, skip the test!!! ")
	}

	workerNodeList, err := e2enode.GetReadySchedulableNodes(context.TODO(), oc.KubeFramework().ClientSet)
	if err != nil || len(workerNodeList.Items) == 0 {
		g.Skip("Can not find any woker nodes in the cluster")
	}

	// Will enable sctp by command
	rhelWorkers, err := compat_otp.GetAllWorkerNodesByOSID(oc, "rhel")
	o.Expect(err).NotTo(o.HaveOccurred())
	if len(rhelWorkers) > 0 {
		e2e.Logf("There are %v number rhel workers in this cluster, will use manual way to load sctp module.", len(rhelWorkers))
		for _, worker := range rhelWorkers {
			enableSCTPModuleOnNode(oc, worker, "rhel")
		}

	}

	rhcosWorkers, err := compat_otp.GetAllWorkerNodesByOSID(oc, "rhcos")
	o.Expect(err).NotTo(o.HaveOccurred())
	e2e.Logf("%v", rhcosWorkers)
	if len(rhcosWorkers) > 0 {
		for _, worker := range rhcosWorkers {
			enableSCTPModuleOnNode(oc, worker, "rhcos")
		}
	}
}
