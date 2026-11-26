package e2etests

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	filePath "path/filepath"
	"strconv"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func (cudncrd *cudnCRDResource) createCUDNCRDSingleStack() {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"CIDR="+cudncrd.cidr, "PREFIX="+strconv.Itoa(int(cudncrd.prefix)), "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createCUDNCRDDualStack() {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"IPv4CIDR="+cudncrd.IPv4cidr, "IPv4PREFIX="+strconv.Itoa(int(cudncrd.IPv4prefix)), "IPv6CIDR="+cudncrd.IPv6cidr, "IPv6PREFIX="+strconv.Itoa(int(cudncrd.IPv6prefix)), "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer2SingleStackCUDNCRD() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"CIDR="+cudncrd.cidr, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer2DualStackCUDNCRD() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue,
			"IPv4CIDR="+cudncrd.IPv4cidr, "IPv6CIDR="+cudncrd.IPv6cidr, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func (cudncrd *cudnCRDResource) createLayer3LocalnetCUDNCRD() {
	err := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", cudncrd.template, "-p", "CRDNAME="+cudncrd.crdname, "LABELKEY="+cudncrd.labelkey, "LABELVALUE="+cudncrd.labelvalue, "PHYSICALNETWORK="+cudncrd.physicalnetworkname, "SUBNET="+cudncrd.subnet, "EXCLUDESUBNET="+cudncrd.excludesubnet, "ROLE="+cudncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create cudn CRD %s due to %v", cudncrd.crdname, err))
}

func applyCUDNtoMatchLabelNS(matchLabelKey, matchValue, crdName, ipv4cidr, ipv6cidr, cidr, topology string) (cudnCRDResource, error) {
	var (
		networkingUDNDir, _  = filePath.Abs("testdata/networking/udn")
		cudnCRDSingleStack   = filePath.Join(networkingUDNDir, "cudn_crd_singlestack_template.yaml")
		cudnCRDdualStack     = filePath.Join(networkingUDNDir, "cudn_crd_dualstack_template.yaml")
		cudnCRDL2dualStack   = filePath.Join(networkingUDNDir, "cudn_crd_layer2_dualstack_template.yaml")
		cudnCRDL2SingleStack = filePath.Join(networkingUDNDir, "cudn_crd_layer2_singlestack_template.yaml")
	)

	ipStackType := checkIPStackType()
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
			cudncrd.createCUDNCRDDualStack()
		case "ipv6single":
			cudncrd.prefix = 64
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDSingleStack
			cudncrd.createCUDNCRDSingleStack()
		case "ipv4single":
			cudncrd.prefix = 24
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDSingleStack
			cudncrd.createCUDNCRDSingleStack()
		}
	case "layer2":
		switch ipStackType {
		case "dualstack":
			cudncrd.IPv4cidr = ipv4cidr
			cudncrd.IPv6cidr = ipv6cidr
			cudncrd.template = cudnCRDL2dualStack
			cudncrd.createLayer2DualStackCUDNCRD()
		default:
			cudncrd.cidr = cidr
			cudncrd.template = cudnCRDL2SingleStack
			cudncrd.createLayer2SingleStackCUDNCRD()
		}
	}
	err := waitCUDNCRDApplied(cudncrd.crdname)
	if err != nil {
		return cudncrd, err
	}
	return cudncrd, nil
}

func applyLocalnetCUDNtoMatchLabelNS(matchLabelKey, matchValue, crdName, physicalNetworkName, subnet, excludeSubnet string, vlan bool) (cudnCRDResource, error) {
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

	cudncrd.createLayer3LocalnetCUDNCRD()
	err := waitCUDNCRDApplied(cudncrd.crdname)
	if err != nil {
		return cudncrd, err
	}
	return cudncrd, nil
}

func (udncrd *udnCRDResource) createUdnCRDSingleStack() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "CIDR="+udncrd.cidr, "PREFIX="+strconv.Itoa(int(udncrd.prefix)), "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createUdnCRDDualStack() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "IPv4CIDR="+udncrd.IPv4cidr, "IPv4PREFIX="+strconv.Itoa(int(udncrd.IPv4prefix)), "IPv6CIDR="+udncrd.IPv6cidr, "IPv6PREFIX="+strconv.Itoa(int(udncrd.IPv6prefix)), "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createLayer2DualStackUDNCRD() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "IPv4CIDR="+udncrd.IPv4cidr, "IPv6CIDR="+udncrd.IPv6cidr, "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func (udncrd *udnCRDResource) createLayer2SingleStackUDNCRD() {
	err := wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", udncrd.template, "-p", "CRDNAME="+udncrd.crdname, "NAMESPACE="+udncrd.namespace, "CIDR="+udncrd.cidr, "MTU="+strconv.Itoa(int(udncrd.mtu)), "ROLE="+udncrd.role)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("fail to create udn CRD %s due to %v", udncrd.crdname, err))
}

func createGeneralUDNCRD(namespace, crdName, ipv4cidr, ipv6cidr, cidr, layer string) {
	// This is a function for common CRD creation without special requirement for parameters which is can be used for common cases and to reduce code lines in case level.
	var (
		networkingUDNDir, _     = filePath.Abs("testdata/networking/udn")
		udnCRDdualStack         = filePath.Join(networkingUDNDir, "udn_crd_dualstack2_template.yaml")
		udnCRDSingleStack       = filePath.Join(networkingUDNDir, "udn_crd_singlestack_template.yaml")
		udnCRDLayer2dualStack   = filePath.Join(networkingUDNDir, "udn_crd_layer2_dualstack_template.yaml")
		udnCRDLayer2SingleStack = filePath.Join(networkingUDNDir, "udn_crd_layer2_singlestack_template.yaml")
	)

	ipStackType := checkIPStackType()
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
			udncrd.createUdnCRDDualStack()
		case "ipv6single":
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				prefix:    64,
				template:  udnCRDSingleStack,
			}
			udncrd.createUdnCRDSingleStack()
		default:
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				prefix:    24,
				template:  udnCRDSingleStack,
			}
			udncrd.createUdnCRDSingleStack()
		}
		err := waitUDNCRDApplied(namespace, udncrd.crdname)
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
			udncrd.createLayer2DualStackUDNCRD()

		default:
			udncrd = udnCRDResource{
				crdname:   crdName,
				namespace: namespace,
				role:      "Primary",
				cidr:      cidr,
				template:  udnCRDLayer2SingleStack,
			}
			udncrd.createLayer2SingleStackUDNCRD()
			err := waitUDNCRDApplied(namespace, udncrd.crdname)
			o.Expect(err).NotTo(o.HaveOccurred())
		}
	default:
		e2e.Logf("Not surpport UDN type for now.")
	}
}

func waitUDNCRDApplied(ns, crdName string) error {
	checkErr := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 60*time.Second, false, func(_ context.Context) (bool, error) {
		obj, err := getDynamicResource("userdefinednetwork", crdName, ns)
		if err != nil {
			e2e.Logf("Failed to get UDN %v, error: %s. Trying again", crdName, err)
			return false, nil
		}
		status, _, _ := getConditionStatus(obj, "NetworkAllocationSucceeded")
		if status != "True" {
			e2e.Logf("UDN CRD was not applied yet, trying again.")
			return false, nil
		}
		return true, nil
	})
	return checkErr
}

func waitCUDNCRDApplied(crdName string) error {
	checkErr := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 30*time.Second, false, func(_ context.Context) (bool, error) {
		obj, err := getDynamicResource("clusteruserdefinednetwork", crdName, "")
		if err != nil {
			e2e.Logf("Failed to get CUDN %v, error: %s. Trying again", crdName, err)
			return false, nil
		}
		status, _, _ := getConditionStatus(obj, "NetworkCreated")
		if status != "True" {
			e2e.Logf("CUDN CRD was not applied yet, trying again.")
			return false, nil
		}
		return true, nil
	})
	return checkErr
}

func (pod *udnPodResource) createUdnPod() {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 20*time.Second, false, func(_ context.Context) (bool, error) {
		err1 := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", pod.template, "-p", "NAME="+pod.name, "NAMESPACE="+pod.namespace, "LABEL="+pod.label)
		if err1 != nil {
			e2e.Logf("the err:%v, and try next round", err1)
			return false, nil
		}
		return true, nil
	})

	assertWaitPollNoErr(err, fmt.Sprintf("fail to create pod %v", pod.name))
}

func createUDNNamespace(ns string) {
	nsJSON := fmt.Sprintf(`{"apiVersion":"v1","kind":"Namespace","metadata":{"name":"%s","labels":{"k8s.ovn.org/primary-user-defined-network":""}}}`, ns)
	cmd := exec.Command("oc", "create", "-f", "-")
	cmd.Stdin = strings.NewReader(nsJSON)
	output, err := cmd.CombinedOutput()
	o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("failed to create UDN namespace %s: %s", ns, string(output)))
	e2e.Logf("UDN namespace %s created with primary-user-defined-network label", ns)
}

// getPodIPUDN returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
func getPodIPUDN(namespace, podName, netName string) (string, string) {
	ipStack := checkIPStackType()
	cmdIPv4 := "ip a sho " + netName + " | awk 'NR==3{print $2}' |grep -Eo '((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])'"
	cmdIPv6 := "ip -o -6 addr show dev " + netName + " | awk '$3 == \"inet6\" && $6 == \"global\" {print $4}' | cut -d'/' -f1"
	switch ipStack {
	case "ipv4single":
		podIPv4, err := execCommandInSpecificPod(namespace, podName, cmdIPv4)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod %s IPv4 in namespace %s is %q", podName, namespace, podIPv4)
		return podIPv4, ""
	case "ipv6single":
		podIPv6, err := execCommandInSpecificPod(namespace, podName, cmdIPv6)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod %s IPv6 in namespace %s is %q", podName, namespace, podIPv6)
		return podIPv6, ""
	default:
		podIPv4, err := execCommandInSpecificPod(namespace, podName, cmdIPv4)
		o.Expect(err).NotTo(o.HaveOccurred())
		podIPv6, err := execCommandInSpecificPod(namespace, podName, cmdIPv6)
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("The UDN pod's %s IPv6 and IPv4 IP in namespace %s is %q %q", podName, namespace, podIPv6, podIPv4)
		return podIPv6, podIPv4
	}
}

// CurlPod2PodFailUDN ensures no connectivity from a udn pod to pod regardless of network addressing type on cluster
func CurlPod2PodFailUDN(namespaceSrc, podNameSrc, namespaceDst, podNameDst string) {
	// getPodIPUDN will returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
	podIP1, podIP2 := getPodIPUDN(namespaceDst, podNameDst, "ovn-udn1")
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
func CurlPod2PodPassUDN(namespaceSrc, podNameSrc, namespaceDst, podNameDst string) {
	// getPodIPUDN will returns IPv6 and IPv4 in vars in order on dual stack respectively and main IP in case of single stack (v4 or v6) in 1st var, and nil in 2nd var
	podIP1, podIP2 := getPodIPUDN(namespaceDst, podNameDst, "ovn-udn1")
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

func deleteNMStateCR(rs nmstateCRResource) {
	e2e.Logf("delete %s CR %s", "nmstate", rs.name)
	err := deleteDynamicResource("nmstate", rs.name, "")
	if err != nil && !apierrors.IsNotFound(err) {
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

func checkNmstateCR(namespace string) (bool, error) {
	WaitForPodsReadyWithLabel(namespace, "component=kubernetes-nmstate-handler")
	WaitForPodsReadyWithLabel(namespace, "component=kubernetes-nmstate-webhook")
	/*
		Due to bug OCPBUGS-54295 nmstate-console-plugin pod cannot be successfully created, comment it for now
			err = waitForPodWithLabelReady(oc, namespace, "app=nmstate-console-plugin")
			if err != nil {
				e2e.Logf("nmstate-console-plugin pod did not transition to ready state %v", err)
				return false, err
			}*/
	WaitForPodsReadyWithLabel(namespace, "component=kubernetes-nmstate-metrics")
	e2e.Logf("nmstate-handler, nmstate-webhook, nmstate-console-plugin and nmstate-metrics pods created successfully")
	return true, nil
}

func createNMStateCR(nmstatecr nmstateCRResource, namespace string) (bool, error) {
	g.By("Creating NMState CR from template")

	err := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", nmstatecr.template, "-p", "NAME="+nmstatecr.name)
	if err != nil {
		e2e.Logf("Error creating NMState CR %v", err)
		return false, err
	}

	result, err := checkNmstateCR(namespace)
	return result, err
}

func deleteNNCP(name string) {
	e2e.Logf("delete nncp %s", name)
	err := deleteDynamicResource("nncp", name, "")
	if err != nil && !apierrors.IsNotFound(err) {
		e2e.Logf("Failed to delete nncp %s, error:%s", name, err)
	}
}

func (bvpr *ovnMappingPolicyResource) configNNCP() error {
	err := applyResourceFromTemplateByAdmin("--ignore-unknown-parameters=true", "-f", bvpr.template, "-p", "NAME="+bvpr.name, "NODELABEL="+bvpr.nodelabel, "LABELVALUE="+bvpr.labelvalue,
		"LOCALNET1="+bvpr.localnet1, "BRIDGE1="+bvpr.bridge1)
	if err != nil {
		e2e.Logf("Error configure ovnmapping %v", err)
		return err
	}
	return nil
}

func checkNNCPStatus(policyName string, expectedStatus string) error {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 3*time.Minute, false, func(_ context.Context) (bool, error) {
		e2e.Logf("Checking status of nncp %s", policyName)
		obj, getErr := getDynamicResource("nncp", policyName, "")
		if getErr != nil {
			e2e.Logf("Failed to get nncp status, error:%s. Trying again", getErr)
			return false, nil
		}
		condStatus, _, _ := getConditionStatus(obj, expectedStatus)
		if condStatus != "True" {
			e2e.Logf("nncp %s condition %s is not True yet. Trying again", policyName, expectedStatus)
			return false, nil
		}
		return true, nil
	})
	return err
}
