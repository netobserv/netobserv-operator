package e2etests

import (
	"fmt"
	filePath "path/filepath"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// SR-IOV hardware and cluster access prerequisites are documented in the QE guide.
// The test deploys the SR-IOV Network Operator and, when the cluster has no default
// StorageClass, deploys the QE scripts' NFS provisioner setup.
// https://gitlab.cee.redhat.com/netobserv-qe/netobserv-qe-scripts/-/blob/main/sriov/SRIOV-testing-guide.md?ref_type=heads#sriov-testing-guide

var _ = g.Describe("[sig-netobserv] Network_Observability with SR-IOV", g.Ordered, g.Serial, func() {
	var (
		namespace          string
		hardware           sriovHardware
		nfsProvisionerPath = filePath.Join(baseDir, "networking", "sriov", "nfs-provisioner.yaml")
		sriovCatalogSource = Resource{"catalogsource", "sriov-konflux", "openshift-marketplace"}
		sriovCatalog       = CatalogSourceObjects{"stable", sriovCatalogSource.Name, sriovCatalogSource.Namespace}
		sriovOperator      = SubscriptionObjects{
			OperatorName:  sriovPackage,
			Namespace:     sriovOperatorNS,
			PackageName:   sriovPackage,
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "singlenamespace-og.yaml"),
			CatalogSource: &sriovCatalog,
		}
	)

	g.BeforeAll(func() {
		// Skip unsupported clusters before installing any cluster-wide SR-IOV resources.
		hardware = getNetObservSriovHardware()

		// Keep the cluster-wide operator and config installed for subsequent runs.
		catalogImage := "quay.io/redhat-user-workloads/ocp-art-tenant/art-fbc:ocp__5.0__ose-sriov-network-rhel9-operator"
		catalogErr := setupCatalogSource(
			sriovCatalogSource,
			filePath.Join(baseDir, "networking", "sriov", "catalog-source-template.yaml"),
			filePath.Join(baseDir, "networking", "sriov", "image-digest-mirror-set.yaml"),
			catalogImage,
			false,
			&sriovCatalog,
			&sriovOperator,
		)
		o.Expect(catalogErr).NotTo(o.HaveOccurred(), "failed to set up SR-IOV catalog source and image mirrors")

		operatorExists, err := CheckOperatorStatus(sriovOperator.Namespace, sriovOperator.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to check SR-IOV operator status")
		if !operatorExists {
			ensureOperatorDeployed(sriovOperator, sriovCatalog, "name=sriov-network-operator")
		}

		_, err = getDynamicResource("sriovoperatorconfig", "default", sriovOperatorNS)
		if apierrors.IsNotFound(err) {
			configPath := filePath.Join(baseDir, "networking", "sriov", "sriovoperatorconfig.yaml")
			ApplyResourceFromFile("", configPath)
		} else {
			o.Expect(err).NotTo(o.HaveOccurred(), "failed to check SriovOperatorConfig")
		}
		waitForSriovOperatorReady()
	})

	g.BeforeEach(func() {
		oc := NewCLI()
		namespace = oc.Namespace()
		hardware = discoverNetObservSriovHardware(hardware)
		ensureSriovLokiStorage(nfsProvisionerPath)
	})

	g.It("Author:memodi-NonPreRelease-Medium-67619-Verify NetObserv flows when the FlowCollector starts before SR-IOV [Serial]", func() {
		runNetObservSriovFlowTest(namespace, hardware, true)
	})

	g.It("Author:memodi-NonPreRelease-Medium-67619-Verify NetObserv flows when SR-IOV starts before the FlowCollector [Serial]", func() {
		runNetObservSriovFlowTest(namespace, hardware, false)
	})
})

func runNetObservSriovFlowTest(namespace string, hardware sriovHardware, flowCollectorFirst bool) {
	resourceName := "netobservsriov" + getRandomString()
	sriovDir := filePath.Join(baseDir, "networking", "sriov")
	nodePolicyTemplate := filePath.Join(sriovDir, "sriovnetworknodepolicy-template.yaml")
	networkTemplate := filePath.Join(sriovDir, "sriovnetwork-template.yaml")
	podTemplate := filePath.Join(sriovDir, "sriovpod-template.yaml")
	flow := Flowcollector{
		Namespace:         namespace,
		Template:          flowFixturePath,
		MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		EBPFPrivileged:    "true",
	}
	flowCollectorCreated := false
	defer func() {
		if flowCollectorCreated {
			_, getErr := getDynamicResource("flowcollector", "cluster", "")
			if apierrors.IsNotFound(getErr) {
				return
			}
			o.Expect(getErr).NotTo(o.HaveOccurred())
			o.Expect(flow.DeleteFlowcollector()).To(o.Succeed())
		}
	}()

	if flowCollectorFirst {
		g.By("Deploy FlowCollector before creating SR-IOV interfaces")
		flowCollectorCreated = true
		flow.CreateFlowcollector()
	}

	err := applyNsResourceFromTemplateByAdmin(sriovOperatorNS,
		"--ignore-unknown-parameters=true", "-f", nodePolicyTemplate, "-p",
		"NAMESPACE="+sriovOperatorNS,
		"POLICY_NAME="+resourceName,
		"DEVICE_ID="+hardware.deviceID,
		"PF_NAME="+hardware.pfName,
		"VENDOR_ID="+hardware.vendorID,
		"NODE_NAME="+hardware.nodeName,
		"NUM_VFS="+sriovVFCount,
		"RESOURCE_NAME="+resourceName,
	)
	o.Expect(err).NotTo(o.HaveOccurred(), "failed to create SR-IOV node policy")
	defer waitForSriovNodePolicyResourceRemoved(hardware.nodeName, resourceName)
	defer deleteResource("sriovnetworknodepolicy", resourceName, sriovOperatorNS)

	g.By("Wait for the SR-IOV policy and device resource to become available")
	waitForSriovPolicyAndResource(hardware.nodeName, resourceName)

	ip1, ip2 := "192.168.122.71", "192.168.122.72"
	network1 := resourceName + "-net1"
	network2 := resourceName + "-net2"
	createSriovNetwork(networkTemplate, network1, resourceName, namespace, ip1)
	defer func() {
		deleteResource("sriovnetwork", network1, sriovOperatorNS)
		checkResourceDeleted("net-attach-def", network1, namespace)
	}()
	waitForSriovNetworkAttachment(network1, namespace)
	createSriovNetwork(networkTemplate, network2, resourceName, namespace, ip2)
	defer func() {
		deleteResource("sriovnetwork", network2, sriovOperatorNS)
		checkResourceDeleted("net-attach-def", network2, namespace)
	}()
	waitForSriovNetworkAttachment(network2, namespace)

	pod1 := resourceName + "-pod1"
	pod2 := resourceName + "-pod2"
	createSriovTrafficPod(podTemplate, pod1, namespace, hardware.nodeName, network1, ip2)
	defer deleteResource("pod", pod1, namespace)
	waitForSriovTrafficPodReady(pod1, namespace)
	createSriovTrafficPod(podTemplate, pod2, namespace, hardware.nodeName, network2, ip1)
	defer deleteResource("pod", pod2, namespace)
	waitForSriovTrafficPodReady(pod2, namespace)

	g.By("Verify both pods have their SR-IOV interface and address")
	for podName, ip := range map[string]string{pod1: ip1, pod2: ip2} {
		out, execErr := execInPod(namespace, podName, []string{"ip", "-o", "-4", "addr", "show", "dev", sriovInterface})
		o.Expect(execErr).NotTo(o.HaveOccurred(), "failed to inspect %s in pod %s", sriovInterface, podName)
		o.Expect(out).To(o.ContainSubstring(ip), "pod %s did not receive expected SR-IOV address", podName)
	}

	if !flowCollectorFirst {
		g.By("Deploy FlowCollector after SR-IOV interfaces are ready")
		flowCollectorCreated = true
		flow.CreateFlowcollector()
	}

	g.By("Wait for SR-IOV flows enriched with workload namespaces")
	startTime := time.Now().Add(-time.Minute)
	time.Sleep(30 * time.Second)
	interfaceFilter := fmt.Sprintf("\"Interfaces\":\\[[^]]*\"%s\"", sriovInterface)
	flowRecords := getSriovFlowRecords(
		Lokilabels{App: "netobserv-flowcollector", SrcK8SNamespace: namespace, DstK8SNamespace: namespace},
		flow.MonolithicLokiURL, startTime, interfaceFilter,
	)
	o.Expect(flowRecords).NotTo(o.BeEmpty(), "expected flows on %s enriched with source and destination namespaces", sriovInterface)
}
