package e2etests

import (
	"fmt"
	"os/exec"
	filePath "path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = g.Describe("[sig-netobserv] Network_Observability with VMs", g.Ordered, g.ContinueOnFailure, func() {
	defer g.GinkgoRecover()

	var (
		namespace string

		virtualizationDir string

		// Virtualization operator vars
		VOexisting                 = false
		virtOperatorNS             = "openshift-cnv"
		kubevirtHyperconvergedPath string
		virtCatsrc                 = Resource{"catalogsource", "redhat-operators", "openshift-marketplace"}
		virtPackageName            = "kubevirt-hyperconverged"
		virtSource                 = CatalogSourceObjects{"stable", virtCatsrc.Name, virtCatsrc.Namespace}
		VO                         = SubscriptionObjects{
			OperatorName:  "kubevirt-hyperconverged",
			Namespace:     virtOperatorNS,
			PackageName:   virtPackageName,
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "singlenamespace-og.yaml"),
			CatalogSource: &virtSource,
		}
	)

	g.BeforeAll(func() {
		clusterArch, err := getNodeArchitecture()
		o.Expect(err).NotTo(o.HaveOccurred())
		if strings.Contains(clusterArch, "ppc64le") {
			g.Skip("Virtualization operator is not supported on ppc64le architecture. Skip this test!")
		}

		isMetal, err := isClusterBareMetal()
		o.Expect(err).ToNot(o.HaveOccurred())
		if !isMetal && !hasMetalWorkerNodes() {
			g.Skip("Cluster does not have baremetal workers. Skip this test!")
		}

		g.By("Deploy Openshift Virtualization operator")
		virtualizationDir, _ = filePath.Abs("testdata/virtualization")
		kubevirtHyperconvergedPath = filePath.Join(virtualizationDir, "kubevirt-hyperconverged.yaml")

		VOexisting, err = CheckOperatorStatus(VO.Namespace, VO.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())
		if !VOexisting {
			ensureOperatorDeployed(VO, virtSource, "name=virt-operator")
		}

		g.By("Deploy OpenShift Virtualization Deployment CR")
		_, err = exec.Command("oc", "create", "-f", kubevirtHyperconvergedPath).Output()
		o.Expect(err).ToNot(o.HaveOccurred())
		waitUntilHyperConvergedReady("kubevirt-hyperconverged", virtOperatorNS)
		WaitForPodsReadyWithLabel(virtOperatorNS, "app.kubernetes.io/managed-by=virt-operator")
		waitForServiceEndpoints(virtOperatorNS, "kubemacpool-service")
	})

	g.AfterAll(func() {
		_ = deleteDynamicResource("hyperconverged", "kubevirt-hyperconverged", virtOperatorNS)
		_ = Resource{"hyperconverged", "kubevirt-hyperconverged", virtOperatorNS}.WaitUntilResourceIsGone()
		if !VOexisting {
			VO.uninstallOperator()
		}
		cleanupCNVWebhooks()
	})

	g.BeforeEach(func() {
		oc = NewCLI()
		namespace = oc.Namespace()
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-High-76537-Verify flow enrichment for VM's secondary interfaces [Disruptive][Slow]", func() {
		var (
			testNS = "test-76537"
			// NAD vars
			networkName   = "l2-network"
			layer2NadPath = filePath.Join(virtualizationDir, "layer2-nad.yaml")
			// VM vars
			testVMStaticIPTemplatePath = filePath.Join(virtualizationDir, "test-vm-static-IP_template.yaml")
		)

		g.By("Deploy Network Attachment Definition in test-76537 namespace")
		defer deleteNamespace(testNS)
		defer deleteResource("net-attach-def", networkName, testNS)
		_, err := exec.Command("oc", "create", "-f", layer2NadPath).Output()
		o.Expect(err).ToNot(o.HaveOccurred())
		time.Sleep(60 * time.Second)
		checkNAD(networkName, testNS)

		g.By("Deploy test VM1")
		testVM1 := TestVMStaticIPTemplate{
			Name:        "test-vm1",
			Namespace:   testNS,
			NetworkName: networkName,
			Mac:         "02:00:00:00:00:01",
			StaticIP:    "10.10.10.15/24",
			Template:    testVMStaticIPTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM1.Name, testNS)
		err = testVM1.createVMStaticIP()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM1.Name, testVM1.Namespace)

		g.By("Wait for VM1 to get IP assigned")
		vm1Ip, err := waitForVMIPAssignment(testVM1.Name, testVM1.Namespace, 1)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm1Ip).To(o.Equal("10.10.10.15"))

		startTime := time.Now()

		g.By("Deploy test VM2")
		testVM2 := TestVMStaticIPTemplate{
			Name:        "test-vm2",
			Namespace:   testNS,
			NetworkName: networkName,
			Mac:         "02:00:00:00:00:02",
			StaticIP:    "10.10.10.14/24",
			RunCmd:      fmt.Sprintf("[[ping, %s]]", vm1Ip),
			Template:    testVMStaticIPTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM2.Name, testNS)
		err = testVM2.createVMStaticIP()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM2.Name, testVM2.Namespace)

		g.By("Wait for VM2 to get IP assigned")
		vm2Ip, err := waitForVMIPAssignment(testVM2.Name, testVM2.Namespace, 1)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm2Ip).To(o.Equal("10.10.10.14"))

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFPrivileged:    "true",
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testNS,
			SrcK8SOwnerName: testVM2.Name,
			DstK8SNamespace: testNS,
			DstK8SOwnerName: testVM1.Name,
		}
		parameters := []string{"DstAddr=\"10.10.10.15\"", "SrcAddr=\"10.10.10.14\""}

		g.By("Verify flows are written to loki")
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

		g.By("Verify flow logs are enriched")
		vm1PodName, err := getAllPodsWithLabel(testNS, "vm.kubevirt.io/name="+testVM1.Name)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm1PodName).NotTo(o.BeEmpty())
		vm1node, err := getPodNodeName(testNS, vm1PodName[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm1node).NotTo(o.BeEmpty())

		vm2PodName, err := getAllPodsWithLabel(testNS, "vm.kubevirt.io/name="+testVM2.Name)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm2PodName).NotTo(o.BeEmpty())
		vm2node, err := getPodNodeName(testNS, vm2PodName[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm2node).NotTo(o.BeEmpty())

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DstK8SName).Should(o.ContainSubstring(vm1PodName[0]))
			o.Expect(r.Flowlog.SrcK8SName).Should(o.ContainSubstring(vm2PodName[0]))
			o.Expect(r.Flowlog.DstK8SOwnerType).Should(o.ContainSubstring("VirtualMachineInstance"))
			o.Expect(r.Flowlog.SrcK8SOwnerType).Should(o.ContainSubstring("VirtualMachineInstance"))
			o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring("test-76537/l2-network"))
			o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring("test-76537/l2-network"))
		}
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-Medium-85887-Verify UDN with VMs [Disruptive][Slow]", func() {
		SkipIfOCPBelow("v4.18")
		var (
			udnNS                 = "netobserv-udn-85887"
			udnName               = "udn-network-85887"
			testVMUDNTemplatePath = filePath.Join(virtualizationDir, "test-vm-UDN_template.yaml")
		)

		g.By("Deploy UDN in UDN ns")
		var cidr, ipv4cidr, ipv6cidr string
		defer deleteNamespace(udnNS)
		createUDNNamespace(udnNS)

		ipStackType := checkIPStackType()
		if ipStackType == "ipv4single" {
			cidr = "10.151.0.0/16"
		} else {
			if ipStackType == "ipv6single" {
				cidr = "2011:100:200::0/48"
			} else {
				ipv4cidr = "10.151.0.0/16"
				ipv6cidr = "2011:100:200::0/48"
			}
		}
		createGeneralUDNCRD(udnNS, udnName, ipv4cidr, ipv6cidr, cidr, "layer2")

		g.By("Deploy test VM3")
		testVM3 := TestVMUDNTemplate{
			Name:        "test-vm3",
			Namespace:   udnNS,
			NetworkName: udnName,
			Template:    testVMUDNTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM3.Name, testVM3.Namespace)
		err := testVM3.createVMUDN()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM3.Name, testVM3.Namespace)

		g.By("Wait for VM3 to get IP assigned")
		vm3Ip, err := waitForVMIPAssignment(testVM3.Name, testVM3.Namespace, 0)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm3Ip).NotTo(o.BeEmpty())

		startTime := time.Now()

		g.By("Deploy test VM4")
		testVM4 := TestVMUDNTemplate{
			Name:        "test-vm4",
			Namespace:   udnNS,
			NetworkName: udnName,
			RunCmd:      fmt.Sprintf("[[ping, %s]]", vm3Ip),
			Template:    testVMUDNTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM4.Name, testVM4.Namespace)
		err = testVM4.createVMUDN()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM4.Name, testVM4.Namespace)

		g.By("Wait for VM4 to get IP assigned")
		vm4Ip, err := waitForVMIPAssignment(testVM4.Name, testVM4.Namespace, 0)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm4Ip).NotTo(o.BeEmpty())

		g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFPrivileged:    "true",
			EBPFeatures:       []string{"\"UDNMapping\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: udnNS,
			SrcK8SOwnerName: testVM4.Name,
			DstK8SNamespace: udnNS,
			DstK8SOwnerName: testVM3.Name,
		}

		g.By("Verify flows are written to loki")
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

		g.By("Verify flow logs are enriched")
		vm3podname, err := getAllPodsWithLabel(udnNS, "vm.kubevirt.io/name="+testVM3.Name)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm3podname).NotTo(o.BeEmpty())
		vm4podname, err := getAllPodsWithLabel(udnNS, "vm.kubevirt.io/name="+testVM4.Name)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(vm4podname).NotTo(o.BeEmpty())

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DstK8SName).Should(o.ContainSubstring(vm3podname[0]))
			o.Expect(r.Flowlog.SrcK8SName).Should(o.ContainSubstring(vm4podname[0]))
			o.Expect(r.Flowlog.DstK8SOwnerType).Should(o.ContainSubstring("VirtualMachineInstance"))
			o.Expect(r.Flowlog.SrcK8SOwnerType).Should(o.ContainSubstring("VirtualMachineInstance"))
			o.Expect(r.Flowlog.Udns).Should(o.ContainElement(fmt.Sprintf("%s/%s", udnNS, udnName)))
			o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
			o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
		}
	})

	g.It("Author:aramesha-High-85935-Validate CUDN with Localnet and VM's[Serial]", func() {
		SkipIfOCPBelow("v4.19")
		var (
			// NMstate operator vars
			opNamespace       = "openshift-nmstate"
			nmStateDir, _     = filePath.Abs("testdata/networking/nmstate")
			nmstateCRTemplate = filePath.Join(nmStateDir, "nmstate-cr-template.yaml")
			nmstateCR         = nmstateCRResource{
				name:     "nmstate",
				template: nmstateCRTemplate,
			}
			nodeSelectLabel          = "node-role.kubernetes.io/worker"
			ovnMappingPolicyTemplate = filePath.Join(nmStateDir, "ovn-mapping-policy-template.yaml")
			ovnMappingPolicy         = ovnMappingPolicyResource{
				name:       "bridge-mapping-85935",
				nodelabel:  nodeSelectLabel,
				labelvalue: "",
				localnet1:  "mylocalnet",
				bridge1:    "br-ex",
				template:   ovnMappingPolicyTemplate,
			}
			// CUDN vars
			matchLabelKey              = "test.io"
			matchValue                 = "cudn-network-" + getRandomString()
			secondaryCUDNName          = "secondary-localnet-85935"
			cudnNS                     = []string{"netobserv-cudn1-85935", "netobserv-cudn2-85935"}
			testVMLocalnetTemplatePath = filePath.Join(virtualizationDir, "test-vm-localnet_template.yaml")
		)

		g.By("Check the platform and network plugin type if it is suitable for running the test")
		networkType := checkNetworkType()
		if !(isPlatformSuitableForNMState()) || !strings.Contains(networkType, "ovn") {
			g.Skip("Skipping for unsupported platform or non-OVN network plugin type!")
		}

		nmstateCatsrc := Resource{"catalogsource", "redhat-operators", "openshift-marketplace"}
		nmstateSource := CatalogSourceObjects{"stable", nmstateCatsrc.Name, nmstateCatsrc.Namespace}
		NMS := SubscriptionObjects{
			OperatorName:  "kubernetes-nmstate-operator",
			Namespace:     opNamespace,
			PackageName:   "kubernetes-nmstate-operator",
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "singlenamespace-og.yaml"),
			CatalogSource: &nmstateSource,
		}
		ensureOperatorDeployed(NMS, nmstateSource, "name="+NMS.OperatorName)

		workerNode, getNodeErr := getFirstWorkerNode()
		o.Expect(getNodeErr).NotTo(o.HaveOccurred())
		o.Expect(workerNode).NotTo(o.BeEmpty())

		g.By("Create NMState CR")
		defer deleteNMStateCR(nmstateCR)
		result, crErr := createNMStateCR(nmstateCR, opNamespace)
		assertWaitPollNoErr(crErr, "create nmstate cr failed")
		o.Expect(result).To(o.BeTrue())
		e2e.Logf("SUCCESS - NMState CR Created")

		g.By("Configure NNCP for creating OvnMapping NMstate Feature")
		originalMappings, origErr := debugNodeWithCommand(workerNode, "ovs-vsctl get Open_vSwitch . external_ids:ovn-bridge-mappings")
		o.Expect(origErr).NotTo(o.HaveOccurred())
		defer deleteNNCP(ovnMappingPolicy.name)
		defer func() {
			ovnmapping, deferErr := debugNodeWithCommand(workerNode, "ovs-vsctl get Open_vSwitch . external_ids:ovn-bridge-mappings")
			o.Expect(deferErr).NotTo(o.HaveOccurred())
			if strings.Contains(ovnmapping, ovnMappingPolicy.localnet1) {
				_, err := debugNodeWithCommand(workerNode, fmt.Sprintf("ovs-vsctl set Open_vSwitch . external_ids:ovn-bridge-mappings=%q", originalMappings))
				o.Expect(err).NotTo(o.HaveOccurred())
			}
		}()
		configErr3 := ovnMappingPolicy.configNNCP()
		o.Expect(configErr3).NotTo(o.HaveOccurred())
		nncpErr3 := checkNNCPStatus(ovnMappingPolicy.name, "Available")
		assertWaitPollNoErr(nncpErr3, fmt.Sprintf("%s policy applied failed", ovnMappingPolicy.name))

		g.By("Create two namespaces and label them")
		for _, ns := range cudnNS {
			defer deleteNamespace(ns)
			err := createNamespace(ns)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = labelNamespace(ns, matchLabelKey, matchValue)
			o.Expect(err).NotTo(o.HaveOccurred())
		}

		g.By("Create secondary localnet CUDN")
		defer removeResource("clusteruserdefinednetwork", secondaryCUDNName)
		_, err := applyLocalnetCUDNtoMatchLabelNS(matchLabelKey, matchValue, secondaryCUDNName, "mylocalnet", "192.168.200.0/24", "192.168.200.1/32", false)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy test VM5")
		testVM5 := TestVMUDNTemplate{
			Name:        "test-vm5",
			Namespace:   cudnNS[0],
			NetworkName: secondaryCUDNName,
			Template:    testVMLocalnetTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM5.Name, testVM5.Namespace)
		err = testVM5.createVMUDN()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM5.Name, testVM5.Namespace)

		g.By("Wait for VM5 to get IP assigned")
		vm5Ip, err := waitForVMIPAssignment(testVM5.Name, testVM5.Namespace, 1)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm5Ip).NotTo(o.BeEmpty())

		startTime := time.Now()

		g.By("Deploy test VM6")
		testVM6 := TestVMUDNTemplate{
			Name:        "test-vm6",
			Namespace:   cudnNS[1],
			NetworkName: secondaryCUDNName,
			RunCmd:      fmt.Sprintf("[[ping, %s]]", vm5Ip),
			Template:    testVMLocalnetTemplatePath,
		}
		defer deleteResource("virtualmachine", testVM6.Name, testVM6.Namespace)
		err = testVM6.createVMUDN()
		o.Expect(err).NotTo(o.HaveOccurred())
		waitUntilVMReady(testVM6.Name, testVM6.Namespace)

		g.By("Wait for VM6 to get IP assigned")
		vm6Ip, err := waitForVMIPAssignment(testVM6.Name, testVM6.Namespace, 1)
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(vm6Ip).NotTo(o.BeEmpty())

		g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFPrivileged:    "true",
			EBPFeatures:       []string{"\"UDNMapping\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		g.By("Verify CUDN Localnet flows")
		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: cudnNS[1],
			SrcK8SOwnerName: testVM6.Name,
			DstK8SNamespace: cudnNS[0],
			DstK8SOwnerName: testVM5.Name,
		}

		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of CUDN Localnet flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Udns).Should(o.ContainElement(secondaryCUDNName))
			o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
			o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
		}
	})
})
