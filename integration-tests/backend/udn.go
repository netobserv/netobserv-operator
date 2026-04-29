package e2etests

import (
	"context"
	"fmt"
	"net"
	filePath "path/filepath"
	"strconv"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2eoutput "k8s.io/kubernetes/test/e2e/framework/pod/output"
)

type udnCRDResource struct {
	crdname    string
	namespace  string
	IPv4cidr   string
	IPv4prefix int32
	IPv6cidr   string
	IPv6prefix int32
	cidr       string
	prefix     int32
	mtu        int32
	role       string
	template   string
}

type cudnCRDResource struct {
	crdname             string
	labelvalue          string
	labelkey            string
	IPv4cidr            string
	IPv4prefix          int32
	IPv6cidr            string
	IPv6prefix          int32
	cidr                string
	prefix              int32
	role                string
	physicalnetworkname string
	subnet              string
	excludesubnet       string
	template            string
}

type udnPodResource struct {
	name      string
	namespace string
	label     string
	template  string
}

type nmstateCRResource struct {
	name     string
	template string
}

type ovnMappingPolicyResource struct {
	name       string
	nodelabel  string
	labelvalue string
	localnet1  string
	bridge1    string
	template   string
}

func (cudncrd *cudnCRDResource) createCUDNCRDSingleStack(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"CIDR="+cudncrd.cidr, "PREFIX="+strconv.Itoa(int(cudncrd.prefix)), "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createCUDNCRDDualStack(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"IPv4CIDR="+cudncrd.IPv4cidr, "IPv4PREFIX="+strconv.Itoa(int(cudncrd.IPv4prefix)), "IPv6CIDR="+cudncrd.IPv6cidr, "IPv6PREFIX="+strconv.Itoa(int(cudncrd.IPv6prefix)), "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer2SingleStackCUDNCRD(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"CIDR="+cudncrd.cidr, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer2DualStackCUDNCRD(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"IPv4CIDR="+cudncrd.IPv4cidr, "IPv6CIDR="+cudncrd.IPv6cidr, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer3LocalnetCUDNCRD(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue, "PHYSICALNETWORK="+cudncrd.physicalnetworkname, "SUBNET="+cudncrd.subnet, "EXCLUDESUBNET="+cudncrd.excludesubnet, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func applyCUDNtoMatchLabelNS(oc *exutil.CLI, matchLabelKey, matchValue, crdName, ipv4cidr, ipv6cidr, cidr, topology string) (cudnCRDResource, error) {

	var (
		networkingUDNDir, _  = filePath.Abs("testdata/networking/udn")
		cudnCRDSingleStack   = filePath.Join(networkingUDNDir, "cudn_crd_singlestack_template.yaml")
		cudnCRDdualStack     = filePath.Join(networkingUDNDir, "cudn_crd_dualstack_template.yaml")
		cudnCRDL2dualStack   = filePath.Join(networkingUDNDir, "cudn_crd_layer2_dualstack_template.yaml")
		cudnCRDL2SingleStack = filePath.Join(networkingUDNDir, "cudn_crd_layer2_singlestack_template.yaml")
	)

	ipStackType := checkIPStackType(oc)
	cudncrd := cudnCRDResource{
		crdname:    crdName,
		labelkey:   matchLabelKey,
		labelvalue: matchValue,
		role:       "Primary",
		template:   cudnCRDSingleStack,
	}

	switch topology {
	case "layer3":
		switch ipStackType {
		case "dualstack":
			cudncrd.IPv4cidr = ipv4cidr
			cudncrd.IPv4prefix = 24
			cudncrd.IPv6cidr = ipv6cidr
			cudncrd.IPv6prefix = 64
			cudncrd.template = cudnCRDdualStack
			cudncrd.createCUDNCRDDualStack(oc)
		case "ipv6single":
			cudncrd.prefix = 64
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDSingleStack
			cudncrd.createCUDNCRDSingleStack(oc)
		case "ipv4single":
			cudncrd.prefix = 24
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDSingleStack
			cudncrd.createCUDNCRDSingleStack(oc)
		}
	case "layer2":
		switch ipStackType {
		case "dualstack":
			cudncrd.IPv4cidr = ipv4cidr
			cudncrd.IPv6cidr = ipv6cidr
			cudncrd.template = cudnCRDL2dualStack
			cudncrd.createLayer2DualStackCUDNCRD(oc)
		default:
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDL2SingleStack
			cudncrd.createLayer2SingleStackCUDNCRD(oc)
		}
	}
	err := waitCUDNCRDApplied(oc, cudncrd.crdname)
	if err != nil {
		return cudncrd, err
	}
	return cudncrd, nil
}

func applyLocalnetCUDNtoMatchLabelNS(oc *exutil.CLI, matchLabelKey, matchValue, crdName, physicalNetworkName, subnet, excludeSubnet string, vlan bool) (cudnCRDResource, error) {
	var (
		networkingUDNDir, _                = filePath.Abs("testdata/networking/udn")
		cudnCRDLocalnetSingleStack         = filePath.Join(networkingUDNDir, "cudn_crd_localnet_singlestack_template.yaml")
		cudnCRDLocalnetSingleStackWithVlan = filePath.Join(networkingUDNDir, "cudn_crd_localnet_singlestack_with_vlan_template.yaml")
	)

	cudncrd := cudnCRDResource{
		crdname:             crdName,
		labelkey:            matchLabelKey,
		labelvalue:          matchValue,
		physicalnetworkname: physicalNetworkName,
		subnet:              subnet,
		excludesubnet:       excludeSubnet,
		role:                "Secondary",
	}

	if vlan {
		cudncrd.template = cudnCRDLocalnetSingleStackWithVlan
	} else {
		cudncrd.template = cudnCRDLocalnetSingleStack
	}

	cudncrd.createLayer3LocalnetCUDNCRD(oc)
	err := waitCUDNCRDApplied(oc, cudncrd.crdname)
	if err != nil {
		return cudncrd, err
	}
	return cudncrd, nil
}

func (udncrd *udnCRDResource) createUdnCRDSingleStack(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "CIDR="+udncrd.cidr, "PREFIX="+strconv.Itoa(int(udncrd.prefix)), "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createUdnCRDDualStack(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "IPv4CIDR="+udncrd.IPv4cidr, "IPv4PREFIX="+strconv.Itoa(int(udncrd.IPv4prefix)), "IPv6CIDR="+udncrd.IPv6cidr, "IPv6PREFIX="+strconv.Itoa(int(udncrd.IPv6prefix)), "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createLayer2DualStackUDNCRD(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "IPv4CIDR="+udncrd.IPv4cidr, "IPv6CIDR="+udncrd.IPv6cidr, "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createLayer2SingleStackUDNCRD(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "CIDR="+udncrd.cidr, "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func createGeneralUDNCRD(oc *exutil.CLI, namespace, crdName, ipv4cidr, ipv6cidr, cidr, layer string) {
	// This is a function for common CRD creation without special requirement for parameters which is can be used for common cases and to reduce code lines in case level.
	var (
		networkingUDNDir, _     = filePath.Abs("testdata/networking/udn")
		udnCRDdualStack         = filePath.Join(networkingUDNDir, "udn_crd_dualstack2_template.yaml")
		udnCRDSingleStack       = filePath.Join(networkingUDNDir, "udn_crd_singlestack_template.yaml")
		udnCRDLayer2dualStack   = filePath.Join(networkingUDNDir, "udn_crd_layer2_dualstack_template.yaml")
		udnCRDLayer2SingleStack = filePath.Join(networkingUDNDir, "udn_crd_layer2_singlestack_template.yaml")
	)

	ipStackType := checkIPStackType(oc)
	var udncrd udnCRDResource
	switch layer {
	case "layer3":
		switch ipStackType {
		case "dualstack":
			udncrd = udnCRDResource{
				crdname:    crdName,
				namespace:  namespace,
				role:       "Primary",
				IPv4cidr:   ipv4cidr,
				IPv4prefix: 24,
				IPv6cidr:   ipv6cidr,
				IPv6prefix: 64,
				template:   udnCRDdualStack,
			}
			udncrd.createUdnCRDDualStack(oc)
		case "ipv6single":
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				prefix:    64,
				template:  udnCRDSingleStack,
			}
			udncrd.createUdnCRDSingleStack(oc)
		default:
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				prefix:    24,
				template:  udnCRDSingleStack,
			}
			udncrd.createUdnCRDSingleStack(oc)
		}
		err := waitUDNCRDApplied(oc, namespace, udncrd.crdname)
		o.Expect(err).NotTo(o.HaveOccurred())

	case "layer2":
		switch ipStackType {
		case "dualstack":
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				IPv4cidr:  ipv4cidr,
				IPv6cidr:  ipv6cidr,
				template:  udnCRDLayer2dualStack,
			}
			udncrd.createLayer2DualStackUDNCRD(oc)

		default:
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				template:  udnCRDLayer2SingleStack,
			}
			udncrd.createLayer2SingleStackUDNCRD(oc)
			err := waitUDNCRDApplied(oc, namespace, udncrd.crdname)
			o.Expect(err).NotTo(o.HaveOccurred())
		}
	default:
		e2e.Logf("Not surpport UDN type for now.")
	}
}

func waitUDNCRDApplied(oc *exutil.CLI, ns, crdName string) error {
	checkErr := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(_ context.Context) (bool, error) {
		output, efErr := oc.AsAdmin().WithoutNamespace().Run("wait").Args("UserDefinedNetwork/"+crdName, "-n", ns, "--for", "condition=NetworkAllocationSucceeded=True").Output()
		if efErr != nil {
			e2e.Logf("Failed to get UDN %v, error: %s. Trying again", crdName, efErr)
			return false, nil
		}
		if !strings.Contains(output, fmt.Sprintf("userdefinednetwork.k8s.ovn.org/%s condition met", crdName)) {
			e2e.Logf("UDN CRD was not applied yet, trying again. \n %s", output)
			return false, nil
		}
		return true, nil
	})
	return checkErr
}

func waitCUDNCRDApplied(oc *exutil.CLI, crdName string) error {
	checkErr := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(_ context.Context) (bool, error) {
		output, efErr := oc.AsAdmin().WithoutNamespace().Run("wait").Args("ClusterUserDefinedNetwork/"+crdName, "--for", "condition=NetworkCreated=True").Output()
		if efErr != nil {
			e2e.Logf("Failed to get CUDN %v, error: %s. Trying again", crdName, efErr)
			return false, nil
		}
		if !strings.Contains(output, fmt.Sprintf("clusteruserdefinednetwork.k8s.ovn.org/%s condition met", crdName)) {
			e2e.Logf("CUDN CRD was not applied yet, trying again. \n %s", output)
			return false, nil
		}
		return true, nil
	})
	return checkErr
}

func (pod *udnPodResource) createUdnPod(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", pod.template, "-p", "NAME="+pod.name, "NAMESPACE="+pod.namespace, "LABEL="+pod.label)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})

	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("fail to create pod %v", pod.name))
}

// getPodIPUDN returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
func getPodIPUDN(oc *exutil.CLI, namespace, podName, netName string) (string, string) {
	ipStack := checkIPStackType(oc)
	cmdIPv4 := "ip a sho " + netName + " | awk 'NR==3{print $2}' |grep -Eo '((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])'"
	cmdIPv6 := "ip -o -6 addr show dev " + netName + " | awk '$3 == \"inet6\" && $6 == \"global\" {print $4}' | cut -d'/' -f1"
	switch ipStack {
	case "ipv4single":
		podIPv4, err := execCommandInSpecificPod(oc, namespace, podName, cmdIPv4)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod %s IPv4 in namespace %s is %q", podName, namespace, podIPv4)
		return podIPv4, ""
	case "ipv6single":
		podIPv6, err := execCommandInSpecificPod(oc, namespace, podName, cmdIPv6)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod %s IPv6 in namespace %s is %q", podName, namespace, podIPv6)
		return podIPv6, ""
	default:
		podIPv4, err := execCommandInSpecificPod(oc, namespace, podName, cmdIPv4)
		o.Expect(err).NotTo(o.HaveOccurred())
		podIPv6, err := execCommandInSpecificPod(oc, namespace, podName, cmdIPv6)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod's %s IPv6 and IPv4 IP in namespace %s is %q %q", podName, namespace, podIPv6, podIPv4)
		return podIPv6, podIPv4
	}
}

// CurlPod2PodFailUDN ensures no connectivity from a udn pod to pod regardless of network addressing type on cluster
func CurlPod2PodFailUDN(oc *exutil.CLI, namespaceSrc, podNameSrc, namespaceDst, podNameDst string) {
	// getPodIPUDN will returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
	podIP1, podIP2 := getPodIPUDN(oc, namespaceDst, podNameDst, "ovn-udn1")
	if podIP2 != "" {
		_, err := e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP1, "8080"))
		o.Expect(err).To(o.HaveOccurred())
		_, err = e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP2, "8080"))
		o.Expect(err).To(o.HaveOccurred())
	} else {
		_, err := e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP1, "8080"))
		o.Expect(err).To(o.HaveOccurred())
	}
}

// CurlPod2PodPass checks connectivity across udn pods regardless of network addressing type on cluster
func CurlPod2PodPassUDN(oc *exutil.CLI, namespaceSrc, podNameSrc, namespaceDst, podNameDst string) {
	// getPodIPUDN will returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
	podIP1, podIP2 := getPodIPUDN(oc, namespaceDst, podNameDst, "ovn-udn1")
	if podIP2 != "" {
		_, err := e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP1, "8080"))
		o.Expect(err).NotTo(o.HaveOccurred())
		_, err = e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP2, "8080"))
		o.Expect(err).NotTo(o.HaveOccurred())
	} else {
		_, err := e2eoutput.RunHostCmd(namespaceSrc, podNameSrc, "curl --connect-timeout 5 -s "+net.JoinHostPort(podIP1, "8080"))
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

func deleteNMStateCR(oc *exutil.CLI, rs nmstateCRResource) {
	e2e.Logf("delete %s CR %s", "nmstate", rs.name)
	err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("nmstate", rs.name, "--ignore-not-found=true").Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

func checkNmstateCR(oc *exutil.CLI, namespace string) (bool, error) {
	WaitForPodsReadyWithLabel(oc, namespace, "component=kubernetes-nmstate-handler")
	WaitForPodsReadyWithLabel(oc, namespace, "component=kubernetes-nmstate-webhook")
	/*
		Due to bug OCPBUGS-54295 nmstate-console-plugin pod cannot be successfully created, comment it for now
			err = waitForPodWithLabelReady(oc, namespace, "app=nmstate-console-plugin")
			if err != nil {
				e2e.Logf("nmstate-console-plugin pod did not transition to ready state %v", err)
				return false, err
			}*/
	WaitForPodsReadyWithLabel(oc, namespace, "component=kubernetes-nmstate-metrics")
	e2e.Logf("nmstate-handler, nmstate-webhook, nmstate-console-plugin and nmstate-metrics pods created successfully")
	return true, nil
}

func createNMStateCR(oc *exutil.CLI, nmstatecr nmstateCRResource, namespace string) (bool, error) {
	g.By("Creating NMState CR from template")

	err := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", nmstatecr.template, "-p", "NAME="+nmstatecr.name)
	if err != nil {
		e2e.Logf("Error creating NMState CR %v", err)
		return false, err
	}

	result, err := checkNmstateCR(oc, namespace)
	return result, err
}

func deleteNNCP(oc *exutil.CLI, name string) {
	e2e.Logf("delete nncp %s", name)
	err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("nncp", name, "--ignore-not-found=true").Execute()
	if err != nil {
		e2e.Logf("Failed to delete nncp %s, error:%s", name, err)
	}
}

func (bvpr *ovnMappingPolicyResource) configNNCP(oc *exutil.CLI) error {
	err := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", bvpr.template, "-p", "NAME="+bvpr.name, "NODELABEL="+bvpr.nodelabel, "LABELVALUE="+bvpr.labelvalue,
		"LOCALNET1="+bvpr.localnet1, "BRIDGE1="+bvpr.bridge1)
	if err != nil {
		e2e.Logf("Error configure ovnmapping %v", err)
		return err
	}
	return nil
}

func checkNNCPStatus(oc *exutil.CLI, policyName string, expectedStatus string) error {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 3*time.Minute, false, func(_ context.Context) (bool, error) {
		e2e.Logf("Checking status of nncp %s", policyName)
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("nncp", policyName).Output()
		if err != nil {
			e2e.Logf("Failed to get nncp status, error:%s. Trying again", err)
			return false, nil
		}
		if !strings.Contains(output, expectedStatus) {
			e2e.Logf("nncp status does not meet expectation:%s, error:%s, output:%s. Trying again", expectedStatus, err, output)
			return false, nil
		}
		return true, nil
	})
	return err
}
