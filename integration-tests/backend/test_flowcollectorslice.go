package e2etests

import (
	"encoding/json"
	"os"
	"time"

	filePath "path/filepath"
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2eoutput "k8s.io/kubernetes/test/e2e/framework/pod/output"
)

var _ = g.Describe("[sig-netobserv] Network_Observability", func() {

	defer g.GinkgoRecover()
	var (
		oc = compat_otp.NewCLI("netobserv", compat_otp.KubeConfigPath())
		// NetObserv Operator variables
		NOcatSrc = Resource{"catsrc", "netobserv-konflux-fbc", netobservNS}
		NOSource = CatalogSourceObjects{"stable", NOcatSrc.Name, NOcatSrc.Namespace}

		// Template directories
		baseDir, _           = filePath.Abs("testdata")
		subscriptionDir      = filePath.Join(baseDir, "subscription")
		flowFixturePath      = filePath.Join(baseDir, "flowcollector_v1beta2_template.yaml")
		flowSliceFixturePath = filePath.Join(baseDir, "flowcollectorSlice_v1alpha1_template.yaml")

		// Operator namespace object
		OperatorNS = OperatorNamespace{
			Name:              netobservNS,
			NamespaceTemplate: filePath.Join(subscriptionDir, "namespace.yaml"),
		}
		NO = SubscriptionObjects{
			OperatorName:  "netobserv-operator",
			Namespace:     netobservNS,
			PackageName:   NOPackageName,
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
			CatalogSource: &NOSource,
		}
		imageDigest    = filePath.Join(subscriptionDir, "image-digest-mirror-set.yaml")
		catSrcTemplate = filePath.Join(subscriptionDir, "catalog-source.yaml")
		catalogSource  = os.Getenv("MULTISTAGE_PARAM_OVERRIDE_NETOBSERV_CS_IMAGE")

		kubeadminToken  string
		kubeAdminPasswd = os.Getenv("QE_KUBEADMIN_PASSWORD")
		namespace       string

		// Loki Operator variables
		lokiDir         = filePath.Join(baseDir, "loki")
		lokiPackageName = "loki-operator"
		lokiSource      CatalogSourceObjects
		ls              *lokiStack
		Lokiexisting    = false
		lokiStackNS     = "netobserv-loki"
		LO              = SubscriptionObjects{
			OperatorName:  "loki-operator-controller-manager",
			Namespace:     loNS,
			PackageName:   lokiPackageName,
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
			CatalogSource: &lokiSource,
		}

		// LokiStack variables
		ipStackType       string
		lokiStackTemplate = filePath.Join(lokiDir, "lokistack-simple.yaml")
		lokiTenant        = "openshift-network"
	)

	g.BeforeEach(func() {
		if strings.Contains(os.Getenv("E2E_RUN_TAGS"), "disconnected") {
			g.Skip("Skipping tests for disconnected profiles")
		}
		namespace = oc.Namespace()

		g.By("Get kubeadmin token")
		if kubeAdminPasswd == "" {
			g.Skip("no kubeAdminPasswd is provided in this profile, set QE_KUBEADMIN_PASSWORD env var")
		}
		serverURL, serverURLErr := oc.AsAdmin().WithoutNamespace().Run("whoami").Args("--show-server").Output()
		o.Expect(serverURLErr).NotTo(o.HaveOccurred())
		currentContext, currentContextErr := oc.WithoutNamespace().Run("config").Args("current-context").Output()
		o.Expect(currentContextErr).NotTo(o.HaveOccurred())
		defer func() {
			rollbackCtxErr := oc.WithoutNamespace().Run("config").Args("set", "current-context", currentContext).Execute()
			o.Expect(rollbackCtxErr).NotTo(o.HaveOccurred())
		}()

		kubeadminToken = getKubeAdminToken(oc, kubeAdminPasswd, serverURL, currentContext)
		o.Expect(kubeadminToken).NotTo(o.BeEmpty())

		isHypershift := compat_otp.IsHypershiftHostedCluster(oc)

		// Deploy NetObserv operator
		OperatorNS.DeployOperatorNamespace(oc)
		deployedUpstreamCatalogSource, catSrcErr := setupCatalogSource(oc, NOcatSrc, catSrcTemplate, imageDigest, catalogSource, isHypershift, &NOSource, &NO)
		o.Expect(catSrcErr).NotTo(o.HaveOccurred())
		ensureNetObservOperatorDeployed(oc, NO, NOSource, deployedUpstreamCatalogSource)

		ipStackType = checkIPStackType(oc)

		g.By("Deploy loki operator")
		if !validateInfraAndResourcesForLoki(oc, "10Gi", "6") {
			g.Skip("Current platform does not have enough resources available for this test!")
		}

		// check if Loki Operator exists
		var err error
		Lokiexisting, err = CheckOperatorStatus(oc, LO.Namespace, LO.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())

		lokiChannel, err := getOperatorChannel(oc, "redhat-operators", "loki-operator")
		if err != nil || lokiChannel == "" {
			g.Skip("Loki channel not found, skip this case")
		}
		lokiSource = CatalogSourceObjects{lokiChannel, "redhat-operators", "openshift-marketplace"}

		// Don't delete if Loki Operator existed already before NetObserv
		//  unless it is not using the 'stable' operator
		// If Loki Operator was installed by NetObserv tests,
		//  it will install and uninstall after each spec/test.
		if !Lokiexisting {
			ensureOperatorDeployed(oc, LO, lokiSource, "name="+LO.OperatorName)
		} else {
			channelName, err := checkOperatorChannel(oc, LO.Namespace, LO.PackageName)
			o.Expect(err).NotTo(o.HaveOccurred())
			if channelName != lokiChannel {
				e2e.Logf("found %s channel for loki operator, removing and reinstalling with %s channel instead", channelName, lokiSource.Channel)
				LO.uninstallOperator(oc)
				ensureOperatorDeployed(oc, LO, lokiSource, "name="+LO.OperatorName)
				Lokiexisting = false
			}
		}

		g.By("Deploy lokiStack")
		// get storageClass Name
		sc, err := getStorageClassName(oc)
		if err != nil || len(sc) == 0 {
			g.Skip("StorageClass not found in cluster, skip this case")
		}

		objectStorageType := getStorageType(oc)
		if len(objectStorageType) == 0 && ipStackType != "ipv6single" {
			g.Skip("Current cluster doesn't have a proper object storage for this test!")
		}
		oc.CreateSpecifiedNamespaceAsAdmin(lokiStackNS)

		ls = &lokiStack{
			Name:          "lokistack",
			Namespace:     lokiStackNS,
			TSize:         "1x.demo",
			StorageType:   objectStorageType,
			StorageSecret: "objectstore-secret",
			StorageClass:  sc,
			BucketName:    "netobserv-loki-" + getInfrastructureName(oc),
			Tenant:        lokiTenant,
			Template:      lokiStackTemplate,
		}

		if ipStackType == "ipv6single" {
			e2e.Logf("running IPv6 test")
			ls.EnableIPV6 = "true"
		}

		err = ls.prepareResourcesForLokiStack(oc)
		if err != nil {
			g.Skip("Skipping test since LokiStack resources were not deployed")
		}

		err = ls.deployLokiStack(oc)
		if err != nil {
			g.Skip("Skipping test since LokiStack was not deployed")
		}

		lokiStackResource := Resource{"lokistack", ls.Name, ls.Namespace}
		err = lokiStackResource.WaitForResourceToAppear(oc)
		if err != nil {
			g.Skip("Skipping test since LokiStack did not become ready")
		}

		err = ls.waitForLokiStackToBeReady(oc)
		if err != nil {
			g.Skip("Skipping test since LokiStack is not ready")
		}
		ls.Route = "https://" + getRouteAddress(oc, ls.Namespace, ls.Name)
	})

	g.AfterEach(func() {
		ls.removeLokiStack(oc)
		ls.removeObjectStorage(oc)
		if !Lokiexisting {
			LO.uninstallOperator(oc)
		}
		oc.DeleteSpecifiedNamespaceAsAdmin(lokiStackNS)
	})

	g.It("Author:aramesha-Critical-86388-Verify flowCollectorSlice collectionMode: AlwaysCollect [Serial]", func() {
		SkipIfOCPBelow(4, 14)
		// Test ping pods template variables
		pingPodsTemplate := filePath.Join(baseDir, "test-ping-pods_template.yaml")
		testPingPodsTemplate := TestPingPodsTemplate{
			ServerNS:    "test-ping-server-86388-always",
			ClientNS:    "test-ping-client-86388-always",
			PingTargets: "192.168.1.0 8.8.8.8",
			Template:    pingPodsTemplate,
		}

		subnetLabelsConfig := []map[string]interface{}{
			{
				"name": "external-api",
				"cidrs": []string{
					"8.8.8.8/32",
					"1.1.1.1/32",
				},
			},
			{
				"name": "internal-service",
				"cidrs": []string{
					"192.168.1.0/24",
				},
			},
		}

		config, err := json.Marshal(subnetLabelsConfig)
		o.Expect(err).ToNot(o.HaveOccurred())
		subnetLabels := string(config)

		g.By("Deploy FlowCollectorSlice")
		startTime := time.Now()
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ClientNS)
		oc.CreateSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ClientNS)
		flowSlice := FlowcollectorSlice{
			Name:         "subnet-label-slice",
			Namespace:    testPingPodsTemplate.ClientNS,
			SubnetLabels: subnetLabels,
			Template:     flowSliceFixturePath,
		}

		defer func() { _ = flowSlice.DeleteFlowcollectorSlice(oc) }()
		flowSlice.CreateFlowcollectorSlice(oc)

		g.By("Deploy FlowCollector with SlicesEnabled in AlwaysCollect mode")
		flow := Flowcollector{
			Namespace:      namespace,
			LokiNamespace:  lokiStackNS,
			CollectionMode: "AlwaysCollect",
			SlicesEnable:   "true",
			Template:       flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)
		flowSlice.WaitForFlowcollectorSliceReady(oc)

		g.By("Deploy test ping server and client pods")
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ServerNS)
		err = testPingPodsTemplate.createPingPods(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testPingPodsTemplate.ServerNS)
		compat_otp.AssertAllPodsToBeReady(oc, testPingPodsTemplate.ClientNS)

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		// Scenario1: Internal IP subnetLabel
		g.By("Verify flows with internal-service subnetLabel")
		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ClientNS,
		}
		parameters := []string{"DstAddr=\"192.168.1.0\""}

		flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client NS to internal-service > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DstSubnetLabel).Should(o.ContainSubstring("internal-service"))
		}

		// Scenario2: External IP subnetLabel
		g.By("Verify flows with external-api subnetLabel")
		parameters = []string{"DstAddr=\"8.8.8.8\""}

		flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client NS to external-api > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DstSubnetLabel).Should(o.ContainSubstring("external-api"))
		}

		// Scenario3: Flows are collected from namespaces without Slice deployed too
		g.By("Verify flows having no subnet label")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ServerNS,
		}

		flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from server NS > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DstSubnetLabel).Should(o.ContainSubstring("external-api"))
		}
	})

	g.It("Author:aramesha-Critical-86388-Verify flowCollectorSlice collectionMode: AllowList [Serial]", func() {
		SkipIfOCPBelow(4, 14)
		// Test ping pods template variables
		pingPodsTemplate := filePath.Join(baseDir, "test-ping-pods_template.yaml")
		testPingPodsTemplate := TestPingPodsTemplate{
			ServerNS:    "test-ping-server-86388-allowlist",
			ClientNS:    "test-ping-client-86388-allowlist",
			PingTargets: "8.8.8.8",
			Template:    pingPodsTemplate,
		}

		g.By("Deploy FlowCollectorSlice")
		startTime := time.Now()
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ClientNS)
		oc.CreateSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ClientNS)
		flowSlice := FlowcollectorSlice{
			Name:      "namespace-slice",
			Namespace: testPingPodsTemplate.ClientNS,
			Sampling:  "3",
			Template:  flowSliceFixturePath,
		}

		defer func() { _ = flowSlice.DeleteFlowcollectorSlice(oc) }()
		flowSlice.CreateFlowcollectorSlice(oc)

		g.By("Deploy FlowCollector with Slices enabled in AllowList mode")
		flow := Flowcollector{
			Namespace:       namespace,
			LokiNamespace:   lokiStackNS,
			CollectionMode:  "AllowList",
			SlicesEnable:    "true",
			NamespacesAllow: []string{"\"/openshift-.*/\""},
			Template:        flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)
		flowSlice.WaitForFlowcollectorSliceReady(oc)

		g.By("Deploy test ping server and client pods")
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testPingPodsTemplate.ServerNS)
		err := testPingPodsTemplate.createPingPods(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testPingPodsTemplate.ServerNS)
		compat_otp.AssertAllPodsToBeReady(oc, testPingPodsTemplate.ClientNS)

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		// Scenario1: Ping from namespace where flowCollectorSlice is deployed
		g.By("Verify flows from client NS")
		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ClientNS,
		}
		parameters := []string{"DstAddr=\"8.8.8.8\""}

		flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client NS > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 3))
		}

		// Scenario2: Ping from namespace where flowCollectorSlice is NOT deployed
		g.By("Verify NO flows are seen from server NS")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ServerNS,
		}

		flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically("==", 0), "expected number of flows from server NS = 0")

		// Scenario3: Flows from namespace in allowedNamespaces section of flowcollector
		g.By("Verify flows are seen to openshift-dns")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: "openshift-dns",
		}

		flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from openshift-dns NS > 0")

		// Scenario4: Flows between namespaces with one in allowedNamespaces section should still be collected
		g.By("Verify flows between namespaces")
		startTime = time.Now()
		// Get server pod IP
		serverPodIP, _ := getPodIP(oc, testPingPodsTemplate.ServerNS, "ping-server", ipStackType)

		// Ping server pod from client pod
		_, _ = e2eoutput.RunHostCmd(testPingPodsTemplate.ClientNS, "ping-client", "ping -c 100 "+serverPodIP)
		time.Sleep(120 * time.Second)

		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ClientNS,
			DstK8SNamespace: testPingPodsTemplate.ServerNS,
		}

		flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows between test namespaces > 0")
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-High-87145-Verify FlowCollectorSlices multi-tenancy [Disruptive][Slow]", func() {
		SkipIfOCPBelow(4, 14)
		g.By("Creating test users")
		users, usersHTpassFile, htPassSecret := getNewUser(oc, 1)
		defer userCleanup(oc, users, usersHTpassFile, htPassSecret)

		g.By("Deploy FlowCollector with Slices enabled")
		flow := Flowcollector{
			Namespace:       namespace,
			LokiNamespace:   lokiStackNS,
			CollectionMode:  "AllowList",
			SlicesEnable:    "true",
			NamespacesAllow: []string{"\"/openshift-.*/\""},
			Template:        flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)

		g.By("Verify FlowCollectorSlices ClusterRoles exist")
		clusterRoleOutput, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterrole", "-o=jsonpath={.items[*].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-admin"))
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-edit"))
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-view"))

		g.By("Deploy test server and client pods for test-a namespace")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplateA := TestServerTemplate{
			ServerNS: "test-a-server-87145",
			Template: serverTemplate,
		}
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplateA.ServerNS)
		err = testServerTemplateA.createServer(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testServerTemplateA.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplateA := TestClientTemplate{
			ServerNS: testServerTemplateA.ServerNS,
			ClientNS: "test-a-client-87145",
			Template: clientTemplate,
		}
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplateA.ClientNS)
		err = testClientTemplateA.createClient(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testClientTemplateA.ClientNS)

		// Save original context
		origContxt, contxtErr := oc.AsAdmin().WithoutNamespace().Run("config").Args("current-context").Output()
		o.Expect(contxtErr).NotTo(o.HaveOccurred())
		e2e.Logf("original context is %v", origContxt)
		defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("config").Args("use-context", origContxt).Execute() }()

		origUser := oc.Username()
		e2e.Logf("original user is %s", origUser)
		defer oc.ChangeUser(origUser)

		testUserName := users[0].Username
		oc.ChangeUser(testUserName)
		e2e.Logf("switched to user: %s", testUserName)

		g.By("Create namespace test-a and grant testuser-0 admin permissions")
		testNSA := testClientTemplateA.ClientNS
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("rolebinding", "testuser-0-admin",
			"--clusterrole=flowcollectorslices.flows.netobserv.io-v1alpha1-admin",
			"--user="+testUserName, "-n", testNSA).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Grant testuser admin access to the server namespace as well for flow visibility
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("rolebinding", "testuser-0-admin-server",
			"--clusterrole=admin",
			"--user="+testUserName, "-n", testServerTemplateA.ServerNS).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Grant testuser admin access to client namespace
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("rolebinding", "testuser-0-admin-client",
			"--clusterrole=admin",
			"--user="+testUserName, "-n", testNSA).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Grant loki reader access for multi-tenancy
		defer removeUserAsReader(oc, testUserName)
		addUserAsReader(oc, testUserName)

		g.By("Verify testuser-0 can create flowcollectorslices in test-a")
		canCreate, err := oc.WithoutNamespace().Run("auth").Args("can-i", "create", "flowcollectorslices", "-n", testNSA).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(canCreate).Should(o.ContainSubstring("yes"))

		g.By("Create a FlowCollectorSlice in test-a namespace")
		flowSliceA := FlowcollectorSlice{
			Name:      "test-a-slice",
			Namespace: testNSA,
			Sampling:  "100",
			Template:  flowSliceFixturePath,
		}
		defer func() { _ = flowSliceA.DeleteFlowcollectorSlice(oc) }()
		flowSliceA.CreateFlowcollectorSlice(oc)
		flowSliceA.WaitForFlowcollectorSliceReady(oc)

		g.By("Verify testuser-0 can view the FlowCollectorSlice in test-a")
		sliceOutput, err := oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "-n", testNSA, "-o=jsonpath={.items[*].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(sliceOutput).Should(o.ContainSubstring("test-a-slice"))

		// Verify sampling value
		samplingValue, err := oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "test-a-slice", "-n", testNSA, "-o=jsonpath={.spec.sampling}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(samplingValue).Should(o.Equal("100"))

		g.By("Verify testuser-0 can update the FlowCollectorSlice in test-a")
		err = oc.WithoutNamespace().Run("patch").Args("flowcollectorslice", "test-a-slice", "-n", testNSA,
			"--type=merge", "-p={\"spec\":{\"sampling\":2}}").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Verify sampling was updated
		samplingValue, err = oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "test-a-slice", "-n", testNSA, "-o=jsonpath={.spec.sampling}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(samplingValue).Should(o.Equal("2"))

		g.By("Get testuser token for loki query")
		user0token, err := oc.WithoutNamespace().Run("whoami").Args("-t").Output()
		e2e.Logf("testuser token: %s", user0token)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for flows to be collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		g.By("Verify testuser-0 can access flows from test-a namespace")
		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testServerTemplateA.ServerNS,
			DstK8SNamespace: testNSA,
			SrcK8SOwnerName: "nginx-service",
			FlowDirection:   "0",
		}
		flowRecords, err := lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected testuser to see flows from test-a namespace")

		g.By("Create namespace test-b and create a FlowCollectorSlice as kubeadmin")
		testServerTemplateB := TestServerTemplate{
			ServerNS: "test-b-server-87145",
			Template: serverTemplate,
		}
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplateB.ServerNS)

		testClientTemplateB := TestClientTemplate{
			ServerNS: testServerTemplateB.ServerNS,
			ClientNS: "test-b-client-87145",
			Template: clientTemplate,
		}
		defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplateB.ClientNS)

		// Switch to admin to create test-b resources
		oc.ChangeUser(origUser)
		err = testServerTemplateB.createServer(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testServerTemplateB.ServerNS)

		err = testClientTemplateB.createClient(oc)
		o.Expect(err).NotTo(o.HaveOccurred())
		compat_otp.AssertAllPodsToBeReady(oc, testClientTemplateB.ClientNS)

		testNSB := testClientTemplateB.ClientNS
		flowSliceB := FlowcollectorSlice{
			Name:      "test-b-slice",
			Namespace: testNSB,
			Sampling:  "3",
			Template:  flowSliceFixturePath,
		}
		defer func() { _ = flowSliceB.DeleteFlowcollectorSlice(oc) }()
		flowSliceB.CreateFlowcollectorSlice(oc)
		flowSliceB.WaitForFlowcollectorSliceReady(oc)

		// Switch back to testuser
		oc.ChangeUser(testUserName)

		g.By("Verify testuser-0 cannot see test-b slice")
		sliceOutputB, err := oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "-n", testNSB).Output()
		o.Expect(err).Should(o.HaveOccurred())
		o.Expect(sliceOutputB).Should(o.MatchRegexp(`User ".*" cannot list resource "flowcollectorslices"`))

		g.By("Verify testuser-0 cannot access flows from test-b namespace")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testServerTemplateB.ServerNS,
			DstK8SNamespace: testNSB,
			SrcK8SOwnerName: "nginx-service",
			FlowDirection:   "0",
		}
		flowRecords, err = lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically("==", 0), "expected testuser to NOT see flows from test-b namespace")

		g.By("Add testuser-0 as viewer for test-b namespace")
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("rolebinding", "testuser-0-view",
			"--clusterrole=flowcollectorslices.flows.netobserv.io-v1alpha1-view",
			"--user="+testUserName, "-n", testNSB).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify testuser-0 can view FlowCollectorSlice in test-b")
		sliceOutput, err = oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "-n", testNSB, "-o=jsonpath={.items[*].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(sliceOutput).Should(o.ContainSubstring("test-b-slice"))

		g.By("Verify testuser-0 cannot update FlowCollectorSlice in test-b (view-only)")
		patchOutput, err := oc.WithoutNamespace().Run("patch").Args("flowcollectorslice", "test-b-slice", "-n", testNSB,
			"--type=merge", "-p={\"spec\":{\"sampling\":25}}").Output()
		o.Expect(err).Should(o.HaveOccurred())
		o.Expect(patchOutput).Should(o.MatchRegexp(`User ".*" cannot patch resource "flowcollectorslices"`))

		g.By("Remove testuser-0's view access from test-b")
		err = oc.AsAdmin().WithoutNamespace().Run("delete").Args("rolebinding", "testuser-0-view", "-n", testNSB).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify access to test-b FlowCollectorSlices is revoked")
		sliceOutputRevoked, err := oc.WithoutNamespace().Run("get").Args("flowcollectorslice", "-n", testNSB).Output()
		o.Expect(err).Should(o.HaveOccurred())
		o.Expect(sliceOutputRevoked).Should(o.MatchRegexp(`User ".*" cannot list resource "flowcollectorslices"`))
	})
})
