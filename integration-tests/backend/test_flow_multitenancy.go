package e2etests

import (
	"context"
	"fmt"
	"os/exec"
	filePath "path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

var _ = g.Describe("[sig-netobserv] Network_Observability Multi-Tenancy", g.Ordered, g.ContinueOnFailure, func() {
	defer g.GinkgoRecover()

	var (
		namespace      string
		kubeadminToken string
		lokiStackNS    = "netobserv-loki"
		ls             *lokiStack

		// Template directories
		lokiDir              = filePath.Join(baseDir, "loki")
		flowSliceFixturePath = filePath.Join(baseDir, "flowcollectorSlice_v1alpha1_template.yaml")

		// Loki Operator variables
		lokiPackageName   = "loki-operator"
		lokiSource        CatalogSourceObjects
		Lokiexisting      = false
		ipStackType       string
		lokiStackTemplate = filePath.Join(lokiDir, "lokistack-simple.yaml")
		lokiTenant        = "openshift-network"
		LO                = SubscriptionObjects{
			OperatorName:  "loki-operator-controller-manager",
			Namespace:     loNS,
			PackageName:   lokiPackageName,
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
			CatalogSource: &lokiSource,
		}
	)

	g.BeforeAll(func() {
		g.By("Get kubeadmin token")
		if kubeAdminPasswd == "" {
			g.Skip("no kubeAdminPasswd is provided in this profile, set QE_KUBEADMIN_PASSWORD env var")
		}
		serverURL, serverURLErr := getServerURL()
		o.Expect(serverURLErr).NotTo(o.HaveOccurred())
		currentContext, currentContextErr := getCurrentContext()
		o.Expect(currentContextErr).NotTo(o.HaveOccurred())
		defer func() {
			rollbackCtxErr := exec.Command("oc", "config", "set", "current-context", currentContext).Run()
			o.Expect(rollbackCtxErr).NotTo(o.HaveOccurred())
		}()

		kubeadminToken = getKubeAdminToken(kubeAdminPasswd, serverURL)
		o.Expect(kubeadminToken).NotTo(o.BeEmpty())

		ipStackType = checkIPStackType()

		g.By("Deploy loki operator")
		if !validateInfraAndResourcesForLoki("10Gi", "6") {
			g.Skip("Current platform does not have enough resources available for this test!")
		}

		var err error
		Lokiexisting, err = CheckOperatorStatus(LO.Namespace, LO.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())

		lokiChannel, err := getOperatorChannel("redhat-operators", "loki-operator")
		if err != nil || lokiChannel == "" {
			g.Skip("Loki channel not found, skip this case")
		}
		lokiSource = CatalogSourceObjects{lokiChannel, "redhat-operators", "openshift-marketplace"}

		if !Lokiexisting {
			ensureOperatorDeployed(LO, lokiSource, "name="+LO.OperatorName)
		} else {
			channelName, err := checkOperatorChannel(LO.Namespace, LO.PackageName)
			o.Expect(err).NotTo(o.HaveOccurred())
			if channelName != lokiChannel {
				e2e.Logf("found %s channel for loki operator, removing and reinstalling with %s channel instead", channelName, lokiSource.Channel)
				LO.uninstallOperator()
				ensureOperatorDeployed(LO, lokiSource, "name="+LO.OperatorName)
				Lokiexisting = false
			}
		}

		// DeferCleanup #1: operator (registered first = cleaned up last)
		g.DeferCleanup(func() {
			if !Lokiexisting {
				LO.uninstallOperator()
			}
		})

		g.By("Deploy lokiStack")
		sc, err := getStorageClassName()
		if err != nil || len(sc) == 0 {
			g.Skip("StorageClass not found in cluster, skip this case")
		}

		objectStorageType := getStorageType()
		if len(objectStorageType) == 0 && ipStackType != "ipv6single" {
			g.Skip("Current cluster doesn't have a proper object storage for this test!")
		}

		err = createNamespace(lokiStackNS)
		o.Expect(err).NotTo(o.HaveOccurred())

		// DeferCleanup #2: namespace (cleaned up after objectStorage/lokiStack, before operator)
		g.DeferCleanup(func() {
			deleteNamespace(lokiStackNS)
		})

		ls = &lokiStack{
			Name:          "lokistack",
			Namespace:     lokiStackNS,
			TSize:         "1x.demo",
			StorageType:   objectStorageType,
			StorageSecret: "objectstore-secret",
			StorageClass:  sc,
			BucketName:    "netobserv-loki-" + getInfrastructureName(),
			Tenant:        lokiTenant,
			Template:      lokiStackTemplate,
		}

		if ipStackType == "ipv6single" {
			e2e.Logf("running IPv6 test")
			ls.EnableIPV6 = "true"
		}

		err = ls.prepareResourcesForLokiStack()
		if err != nil {
			g.Skip("Skipping test since LokiStack resources were not deployed")
		}

		// DeferCleanup #3: object storage (cleaned up after lokiStack, before namespace)
		g.DeferCleanup(func() {
			if ls != nil {
				ls.removeObjectStorage()
			}
		})

		err = ls.deployLokiStack()
		if err != nil {
			g.Skip("Skipping test since LokiStack was not deployed")
		}

		// DeferCleanup #4: lokiStack (registered last = cleaned up first)
		g.DeferCleanup(func() {
			if ls != nil {
				ls.removeLokiStack()
			}
		})

		lokiStackResource := Resource{"lokistack", ls.Name, ls.Namespace}
		err = lokiStackResource.WaitForResourceToAppear()
		if err != nil {
			g.Skip("Skipping test since LokiStack did not become ready")
		}

		err = ls.waitForLokiStackToBeReady()
		if err != nil {
			g.Skip("Skipping test since LokiStack is not ready")
		}
		ls.Route = "https://" + getRouteAddress(ls.Namespace, ls.Name)

		g.By("Create secret-watcher RoleBinding in LokiStack namespace")
		createSecretWatcherRB(lokiStackNS)

	})

	g.BeforeEach(func() {
		oc = NewCLI()
		namespace = oc.Namespace()
	})

	g.It("Author:memodi-NonPreRelease-Longduration-High-63839-Verify-multi-tenancy [Disruptive][Slow]", func() {
		users, usersHTpassFile, htPassSecret := getNewUser(2)
		defer userCleanup(users, usersHTpassFile, htPassSecret)

		g.By("Creating client server template and template CRBs for testusers")
		testUserstemplate := filePath.Join(baseDir, "testuser-client-server_template.yaml")
		cmd := exec.Command("oc", "apply", "-f", testUserstemplate, "-n", namespace)
		outputBytes, err := cmd.CombinedOutput()
		o.Expect(err).NotTo(o.HaveOccurred())
		stdout := strings.TrimSpace(string(outputBytes))
		parts := strings.SplitN(stdout, " ", 2)
		o.Expect(len(parts)).Should(o.BeNumerically(">=", 1), "unexpected oc apply output: "+stdout)
		resourceParts := strings.SplitN(parts[0], "/", 2)
		o.Expect(len(resourceParts)).To(o.Equal(2), "expected resource in kind/name format, got: "+parts[0])
		templateName := resourceParts[1]
		defer removeTemplatePermissions(users[0].Username)
		addTemplatePermissions(users[0].Username)

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:       namespace,
			Template:        flowFixturePath,
			LokiNamespace:   lokiStackNS,
			LokiMode:        "LokiStack",
			InstallDemoLoki: "false",
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Deploying test server and client pods")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-63839",
			Template: serverTemplate,
		}
		err = testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")

		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-63839",
			Template: clientTemplate,
		}

		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		origContxt, contxtErr := getCurrentContext()
		o.Expect(contxtErr).NotTo(o.HaveOccurred())
		defer removeUserAsReader(users[0].Username)
		addUserAsReader(users[0].Username)

		changeUser(users[0], namespace)

		g.By("Deploying test server and client pods as user0")
		var (
			testUserServerNS = fmt.Sprintf("%s-server", users[0].Username)
			testUserClientNS = fmt.Sprintf("%s-client", users[0].Username)
		)

		defer func() {
			_ = exec.Command("oc", "config", "use-context", origContxt).Run()
			deleteNamespace(testServerTemplate.ServerNS)
			deleteNamespace(testClientTemplate.ClientNS)
			deleteNamespace(testUserServerNS)
			deleteNamespace(testUserClientNS)
		}()
		configFile, err := processTemplate(namespace, "--ignore-unknown-parameters=true", templateName, "-p", "SERVER_NS="+testUserServerNS, "-p", "CLIENT_NS="+testUserClientNS)
		o.Expect(err).NotTo(o.HaveOccurred())
		createResourceFromFile("", configFile)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testUserServerNS,
			DstK8SNamespace: testUserClientNS,
			SrcK8SOwnerName: "nginx-service",
			FlowDirection:   "0",
		}

		user0tokenBytes, err := exec.Command("oc", "whoami", "-t").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		user0token := strings.TrimSpace(string(user0tokenBytes))

		g.By("Wait for a min before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		g.By("get flowlogs from loki")
		flowRecords, err := lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords > 0")

		g.By("verify no logs are fetched from an NS that user is not admin for")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testClientTemplate.ServerNS,
			DstK8SNamespace: testClientTemplate.ClientNS,
			SrcK8SOwnerName: "nginx-service",
			FlowDirection:   "0",
			AllowEmpty:      true,
		}
		flowRecords, err = lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
		if err != nil {
			o.Expect(err.Error()).To(o.ContainSubstring("permission"), "expected permission error for unauthorized namespace access, got: %v", err)
		} else {
			o.Expect(len(flowRecords)).To(o.Equal(0), "expected zero flowRecords for unauthorized namespace access")
		}
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-High-87145-Verify FlowCollectorSlices multi-tenancy [Disruptive][Slow]", func() {
		g.By("Creating test users")
		users, usersHTpassFile, htPassSecret := getNewUser(1)
		defer userCleanup(users, usersHTpassFile, htPassSecret)

		g.By("Deploy FlowCollector with Slices enabled")
		flow := Flowcollector{
			Namespace:       namespace,
			LokiMode:        "LokiStack",
			LokiNamespace:   lokiStackNS,
			InstallDemoLoki: "false",
			CollectionMode:  "AllowList",
			SlicesEnable:    "true",
			NamespacesAllow: []string{"\"/openshift-.*/\""},
			Template:        flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Verify FlowCollectorSlices ClusterRoles exist")
		clusterRoles, err := k8sClient.RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		var clusterRoleNames []string
		for _, cr := range clusterRoles.Items {
			clusterRoleNames = append(clusterRoleNames, cr.Name)
		}
		clusterRoleOutput := strings.Join(clusterRoleNames, " ")
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-admin"))
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-edit"))
		o.Expect(clusterRoleOutput).Should(o.ContainSubstring("flowcollectorslices.flows.netobserv.io-v1alpha1-view"))

		g.By("Deploy test server and client pods for test-a namespace")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplateA := TestServerTemplate{
			ServerNS: "test-a-server-87145",
			Template: serverTemplate,
		}
		err = testServerTemplateA.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplateA.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplateA := TestClientTemplate{
			ServerNS: testServerTemplateA.ServerNS,
			ClientNS: "test-a-client-87145",
			Template: clientTemplate,
		}
		err = testClientTemplateA.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplateA.ClientNS)

		origContxt, contxtErr := getCurrentContext()
		o.Expect(contxtErr).NotTo(o.HaveOccurred())
		e2e.Logf("original context is %v", origContxt)
		defer func() {
			_ = exec.Command("oc", "config", "use-context", origContxt).Run()
			deleteNamespace(testServerTemplateA.ServerNS)
			deleteNamespace(testClientTemplateA.ClientNS)
		}()

		testUserName := users[0].Username

		g.By("Create namespace test-a and grant testuser-0 admin permissions")
		testNSA := testClientTemplateA.ClientNS
		_, err = k8sClient.RbacV1().RoleBindings(testNSA).Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "testuser-0-admin", Namespace: testNSA},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "flowcollectorslices.flows.netobserv.io-v1alpha1-admin"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: testUserName}},
		}, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		_, err = k8sClient.RbacV1().RoleBindings(testServerTemplateA.ServerNS).Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "testuser-0-admin-server", Namespace: testServerTemplateA.ServerNS},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "admin"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: testUserName}},
		}, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		_, err = k8sClient.RbacV1().RoleBindings(testNSA).Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "testuser-0-admin-client", Namespace: testNSA},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "admin"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: testUserName}},
		}, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		defer removeUserAsReader(testUserName)
		addUserAsReader(testUserName)

		userContext := changeUser(users[0], namespace)

		g.By("Verify testuser-0 can create flowcollectorslices in test-a")
		canCreateBytes, err := exec.Command("oc", "auth", "can-i", "create", "flowcollectorslices", "-n", testNSA).Output()
		canCreate := string(canCreateBytes)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(canCreate).Should(o.ContainSubstring("yes"))

		g.By("Create a FlowCollectorSlice in test-a namespace")
		flowSliceA := FlowcollectorSlice{
			Name:      "test-a-slice",
			Namespace: testNSA,
			Sampling:  "100",
			Template:  flowSliceFixturePath,
		}
		defer func() { _ = flowSliceA.DeleteFlowcollectorSlice() }()
		flowSliceA.CreateFlowcollectorSlice()
		flowSliceA.WaitForFlowcollectorSliceReady()

		g.By("Verify testuser-0 can view the FlowCollectorSlice in test-a")
		sliceCmd := exec.Command("oc", "get", "flowcollectorslice", "-n", testNSA, "-o=jsonpath={.items[*].metadata.name}")
		sliceOutputBytes, err := sliceCmd.Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		sliceOutput := string(sliceOutputBytes)
		o.Expect(sliceOutput).Should(o.ContainSubstring("test-a-slice"))

		cmd := exec.Command("oc", "get", "flowcollectorslice", "test-a-slice", "-n", testNSA, "-o=jsonpath={.spec.sampling}")
		samplingValueBytes, err := cmd.Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		samplingValue := string(samplingValueBytes)
		o.Expect(samplingValue).Should(o.Equal("100"))

		g.By("Verify testuser-0 can update the FlowCollectorSlice in test-a")
		cmd = exec.Command("oc", "patch", "flowcollectorslice", "test-a-slice", "-n", testNSA,
			"--type=merge", "-p={\"spec\":{\"sampling\":2}}")
		err = cmd.Run()

		o.Expect(err).NotTo(o.HaveOccurred())

		cmd = exec.Command("oc", "get", "flowcollectorslice", "test-a-slice", "-n", testNSA, "-o=jsonpath={.spec.sampling}")
		samplingValueBytes2, err := cmd.Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		samplingValue2 := string(samplingValueBytes2)
		o.Expect(samplingValue2).Should(o.Equal("2"))

		g.By("Get testuser token for loki query")
		user0tokenBytes, err := exec.Command("oc", "whoami", "-t").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		user0token := strings.TrimSpace(string(user0tokenBytes))
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
		testClientTemplateB := TestClientTemplate{
			ServerNS: testServerTemplateB.ServerNS,
			ClientNS: "test-b-client-87145",
			Template: clientTemplate,
		}
		defer func() {
			_ = exec.Command("oc", "config", "use-context", origContxt).Run()
			deleteNamespace(testServerTemplateB.ServerNS)
			deleteNamespace(testClientTemplateB.ClientNS)
		}()

		_ = exec.Command("oc", "config", "use-context", origContxt).Run()
		err = testServerTemplateB.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplateB.ServerNS)

		err = testClientTemplateB.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplateB.ClientNS)

		testNSB := testClientTemplateB.ClientNS
		flowSliceB := FlowcollectorSlice{
			Name:      "test-b-slice",
			Namespace: testNSB,
			Sampling:  "3",
			Template:  flowSliceFixturePath,
		}
		defer func() { _ = flowSliceB.DeleteFlowcollectorSlice() }()
		flowSliceB.CreateFlowcollectorSlice()
		flowSliceB.WaitForFlowcollectorSliceReady()

		_ = exec.Command("oc", "config", "use-context", userContext).Run()

		g.By("Verify testuser-0 cannot see test-b slice")
		sliceOutputBBytes, err := exec.Command("oc", "get", "flowcollectorslice", "-n", testNSB).CombinedOutput()
		sliceOutputB := string(sliceOutputBBytes)
		o.Expect(err).Should(o.HaveOccurred())
		o.Expect(sliceOutputB).Should(o.MatchRegexp(`User ".*" cannot list resource "flowcollectorslices"`))

		g.By("Verify testuser-0 cannot access flows from test-b namespace")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testServerTemplateB.ServerNS,
			DstK8SNamespace: testNSB,
			SrcK8SOwnerName: "nginx-service",
			FlowDirection:   "0",
			AllowEmpty:      true,
		}
		flowRecords, err = lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
		if err != nil {
			o.Expect(err.Error()).Should(o.ContainSubstring("permission"), "expected permission error for unauthorized namespace access, got: %v", err)
		} else {
			o.Expect(len(flowRecords)).Should(o.BeNumerically("==", 0), "expected testuser to NOT see flows from test-b namespace")
		}

		g.By("Add testuser-0 as viewer for test-b namespace")
		_, err = k8sClient.RbacV1().RoleBindings(testNSB).Create(context.Background(), &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "testuser-0-view", Namespace: testNSB},
			RoleRef:    rbacv1.RoleRef{APIGroup: "rbac.authorization.k8s.io", Kind: "ClusterRole", Name: "flowcollectorslices.flows.netobserv.io-v1alpha1-view"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: testUserName}},
		}, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify testuser-0 can view FlowCollectorSlice in test-b")
		sliceBCmd := exec.Command("oc", "get", "flowcollectorslice", "-n", testNSB, "-o=jsonpath={.items[*].metadata.name}")
		sliceOutputBBytes, errB := sliceBCmd.Output()
		o.Expect(errB).NotTo(o.HaveOccurred())
		o.Expect(string(sliceOutputBBytes)).Should(o.ContainSubstring("test-b-slice"))

		g.By("Verify testuser-0 cannot update FlowCollectorSlice in test-b (view-only)")
		patchCmd := exec.Command("oc", "patch", "flowcollectorslice", "test-b-slice", "-n", testNSB,
			"--type=merge", "-p={\"spec\":{\"sampling\":25}}")
		patchOutputBytes, patchErr := patchCmd.CombinedOutput()
		o.Expect(patchErr).Should(o.HaveOccurred())
		patchOutput := string(patchOutputBytes)
		o.Expect(patchOutput).Should(o.MatchRegexp(`User ".*" cannot patch resource "flowcollectorslices"`))

		g.By("Remove testuser-0's view access from test-b")
		err = k8sClient.RbacV1().RoleBindings(testNSB).Delete(context.Background(), "testuser-0-view", metav1.DeleteOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify access to test-b FlowCollectorSlices is revoked")
		cmd = exec.Command("oc", "get", "flowcollectorslice", "-n", testNSB)
		sliceOutputRevokedBytes, err := cmd.CombinedOutput()
		o.Expect(err).Should(o.HaveOccurred())
		sliceOutputRevoked := string(sliceOutputRevokedBytes)
		o.Expect(sliceOutputRevoked).Should(o.MatchRegexp(`User ".*" cannot list resource "flowcollectorslices"`))
	})
})
