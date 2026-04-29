package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	filePath "path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
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
		baseDir, _      = filePath.Abs("testdata")
		networkingDir   = filePath.Join(baseDir, "networking")
		subscriptionDir = filePath.Join(baseDir, "subscription")
		flowFixturePath = filePath.Join(baseDir, "flowcollector_v1beta2_template.yaml")

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

		kubeadminToken string
		namespace      string
	)

	g.BeforeEach(func() {
		if strings.Contains(os.Getenv("E2E_RUN_TAGS"), "disconnected") {
			g.Skip("Skipping tests for disconnected profiles")
		}
		namespace = oc.Namespace()

		g.By("Get kubeadmin token")
		kubeAdminPasswd := os.Getenv("QE_KUBEADMIN_PASSWORD")
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

		OperatorNS.DeployOperatorNamespace(oc)
		deployedUpstreamCatalogSource, catSrcErr := setupCatalogSource(oc, NOcatSrc, catSrcTemplate, imageDigest, catalogSource, isHypershift, &NOSource, &NO)
		o.Expect(catSrcErr).NotTo(o.HaveOccurred())
		ensureNetObservOperatorDeployed(oc, NO, NOSource, deployedUpstreamCatalogSource)
	})

	g.It("Author:memodi-NonPreRelease-Longduration-Medium-60664-Medium-61482-Alerts-with-NetObserv [Serial][Slow]", func() {
		SkipIfOCPBelow(4, 10)
		flpAlertRuleName := "flowlogs-pipeline-alert"
		ebpfAlertRuleName := "ebpf-agent-prom-alert"

		flow := Flowcollector{
			Namespace:  namespace,
			Template:   flowFixturePath,
			LokiEnable: "false",
		}
		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)

		// verify configured alerts for flp
		g.By("Get FLP Alert name and Alert Rules")
		rules, err := getConfiguredAlertRules(oc, flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).To(o.ContainSubstring("NetObservLokiError"))

		// verify configured alerts for ebpf-agent
		g.By("Get EBPF Alert name and Alert Rules")
		ebpfRules, err := getConfiguredAlertRules(oc, ebpfAlertRuleName, namespace+"-privileged")
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(ebpfRules).To(o.ContainSubstring("NetObservDroppedFlows"))

		// verify disable alerts feature
		g.By("Verify alerts can be disabled")
		gen, err := getResourceGeneration(oc, "prometheusRule", flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		disableAlertPatchTemp := `[{"op": "$op", "path": "/spec/processor/metrics/disableAlerts", "value": ["NetObservLokiError"]}]`
		disableAlertPatch := strings.Replace(disableAlertPatchTemp, "$op", "add", 1)
		out, err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=json", "-p", disableAlertPatch).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(out).To(o.ContainSubstring("patched"))

		waitForResourceGenerationUpdate(oc, "prometheusRule", flpAlertRuleName, "generation", gen, namespace)
		rules, err = getConfiguredAlertRules(oc, flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).ToNot(o.ContainSubstring("NetObservLokiError"))

		gen, err = getResourceGeneration(oc, "prometheusRule", flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		disableAlertPatch = strings.Replace(disableAlertPatchTemp, "$op", "remove", 1)
		out, err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=json", "-p", disableAlertPatch).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(out).To(o.ContainSubstring("patched"))
		waitForResourceGenerationUpdate(oc, "prometheusRule", flpAlertRuleName, "generation", gen, namespace)
		rules, err = getConfiguredAlertRules(oc, flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).To(o.ContainSubstring("NetObservLokiError"))

		g.By("delete flowcollector")
		_ = flow.DeleteFlowcollector(oc)

		// verify alert firing.
		// configure flowcollector with incorrect loki URL
		// configure very low CacheMaxFlows to have ebpf alert fired.
		flow = Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			CacheMaxFlows:     "100",
			LokiMode:          "Monolithic",
			MonolithicLokiURL: "http://loki.no-ns.svc:3100",
		}
		g.By("Deploy flowcollector with incorrect loki URL and lower cacheMaxFlows value")
		flow.CreateFlowcollector(oc)

		g.By("Wait for alerts to be active")
		waitForAlertToBeActive(oc, "NetObservLokiError")
	})

	g.It("Author:memodi-Medium-63185-Verify NetOberv must-gather plugin [Serial]", func() {
		SkipIfOCPBelow(4, 10)
		mustgatherDir := "/tmp/must-gather-63185"
		mustgatherImage := "quay.io/netobserv/must-gather"

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:  namespace,
			Template:   flowFixturePath,
			LokiEnable: "false",
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)

		g.By("Run must-gather command")
		defer func() { _, _ = exec.Command("bash", "-c", "rm -rf "+mustgatherDir).Output() }()
		output, err := oc.AsAdmin().WithoutNamespace().Run("adm").Args("must-gather", "--image", mustgatherImage, "--dest-dir="+mustgatherDir).Output()
		o.Expect(err).NotTo(o.HaveOccurred(), "must-gather command failed")
		o.Expect(output).NotTo(o.ContainSubstring("error"))

		g.By("Wait for must-gather directory to be populated")
		var mustgatherLogsDir string
		err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 2*time.Minute, false, func(context.Context) (bool, error) {
			matches, globErr := filePath.Glob(mustgatherDir + "/quay-io-netobserv-must-gather-*")
			if globErr != nil || len(matches) == 0 {
				e2e.Logf("Waiting for must-gather directory to be created...")
				return false, nil
			}
			mustgatherLogsDir = matches[0]
			// Check if at least one expected file exists to confirm completion
			checkPattern := fmt.Sprintf("%s/namespaces/*/pods/*", mustgatherLogsDir)
			checkMatches, _ := filePath.Glob(checkPattern)
			if len(checkMatches) == 0 {
				e2e.Logf("Must-gather directory exists but waiting for pod data to be collected...")
				return false, nil
			}
			e2e.Logf("Must-gather data collection completed")
			return true, nil
		})
		compat_otp.AssertWaitPollNoErr(err, "must-gather data not populated within timeout")

		g.By("Verify operator namespace logs are scraped")
		operatorLogsPattern := fmt.Sprintf("%s/namespaces/openshift-netobserv-operator/pods/netobserv-controller-manager-*/manager/manager/logs/current.log", mustgatherLogsDir)
		operatorlogs, err := filePath.Glob(operatorLogsPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(operatorlogs)).Should(o.BeNumerically(">", 0), "No logs were saved to: "+operatorLogsPattern)
		_, err = os.Stat(operatorlogs[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify flowlogs-pipeline pod logs are scraped")
		pods, err := compat_otp.GetAllPods(oc, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		flpLogsPattern := fmt.Sprintf("%s/namespaces/%s/pods/%s/flowlogs-pipeline/flowlogs-pipeline/logs/current.log", mustgatherLogsDir, namespace, pods[0])
		podlogs, err := filePath.Glob(flpLogsPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(podlogs)).Should(o.BeNumerically(">", 0), "No logs were saved to: "+flpLogsPattern)
		_, err = os.Stat(podlogs[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify eBPF agent pod logs are scraped")
		ebpfPods, err := compat_otp.GetAllPods(oc, namespace+"-privileged")
		o.Expect(err).NotTo(o.HaveOccurred())
		ebpfLogsPattern := fmt.Sprintf("%s/namespaces/%s/pods/%s/netobserv-ebpf-agent/netobserv-ebpf-agent/logs/current.log", mustgatherLogsDir, namespace+"-privileged", ebpfPods[0])
		ebpfLogs, err := filePath.Glob(ebpfLogsPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(ebpfLogs)).Should(o.BeNumerically(">", 0), "No logs were saved to: "+ebpfLogsPattern)
		_, err = os.Stat(ebpfLogs[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify FlowCollector CR is dumped")
		fcPattern := fmt.Sprintf("%s/cluster-scoped-resources/flows.netobserv.io/flowcollectors/cluster.yaml", mustgatherLogsDir)
		fcDump, err := filePath.Glob(fcPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(fcDump)).Should(o.BeNumerically(">", 0), "FlowCollector CR not dumped to: "+fcPattern)
		_, err = os.Stat(fcDump[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify FlowCollector CRD definition is dumped")
		crdPattern := fmt.Sprintf("%s/cluster-scoped-resources/apiextensions.k8s.io/customresourcedefinitions/flowcollectors.flows.netobserv.io.yaml", mustgatherLogsDir)
		crdDump, err := filePath.Glob(crdPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(crdDump)).Should(o.BeNumerically(">", 0), "FlowCollector CRD not dumped to: "+crdPattern)
		_, err = os.Stat(crdDump[0])
		o.Expect(err).NotTo(o.HaveOccurred())
	})

	g.It("Author:aramesha-NonPreRelease-Medium-72875-Verify nodeSelector and tolerations with netobserv components [Serial]", func() {
		SkipIfOCPBelow(4, 12)

		// verify tolerations
		g.By("Get worker node of the cluster")
		workerNode, err := compat_otp.GetFirstWorkerNode(oc)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Taint worker node")
		defer func() {
			err := oc.AsAdmin().WithoutNamespace().Run("adm").Args("taint", "node", workerNode, "netobserv-agent-", "--overwrite").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
		}()
		err = oc.AsAdmin().WithoutNamespace().Run("adm").Args("taint", "node", workerNode, "netobserv-agent=true:NoSchedule", "--overwrite").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:  namespace,
			Template:   flowFixturePath,
			LokiEnable: "false",
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)

		g.By("Add wrong toleration for eBPF spec for the taint netobserv-agent=false:NoSchedule")
		patchValue := `{"scheduling":{"tolerations":[{"effect": "NoSchedule", "key": "netobserv-agent", "value": "false", "operator": "Equal"}]}}`
		_, _ = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "-p", `[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": `+patchValue+`}]`, "--type=json").Output()

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady(oc)

		g.By(fmt.Sprintf("Verify eBPF pod is not scheduled on the %s", workerNode))
		eBPFPod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", flow.Namespace+"-privileged", "pods", "--field-selector", "spec.nodeName="+workerNode+"", "-o", "name").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(eBPFPod).Should(o.BeEmpty())

		g.By("Add correct toleration for eBPF spec for the taint netobserv-agent=true:NoSchedule")
		_ = flow.DeleteFlowcollector(oc)
		flow.CreateFlowcollector(oc)
		patchValue = `{"scheduling":{"tolerations":[{"effect": "NoSchedule", "key": "netobserv-agent", "value": "true", "operator": "Equal"}]}}`
		_, _ = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "-p", `[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": `+patchValue+`}]`, "--type=json").Output()

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady(oc)

		g.By(fmt.Sprintf("Verify eBPF pod is scheduled on the node %s after applying toleration for taint netobserv-agent=true:NoSchedule", workerNode))
		eBPFPod, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", flow.Namespace+"-privileged", "pods", "--field-selector", "spec.nodeName="+workerNode+"", "-o", "name").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(eBPFPod).NotTo(o.BeEmpty())

		// verify nodeSelector
		g.By("Add netobserv label to above worker node")
		defer func() { _, _ = compat_otp.DeleteLabelFromNode(oc, workerNode, "test") }()
		_, _ = compat_otp.AddLabelToNode(oc, workerNode, "netobserv-agent", "true")

		g.By("Patch flowcollector with nodeSelector for eBPF pods")
		_ = flow.DeleteFlowcollector(oc)
		flow.CreateFlowcollector(oc)
		patchValue = `{"scheduling":{"nodeSelector":{"netobserv-agent": "true"}}}`
		_, _ = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "-p", `[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": `+patchValue+`}]`, "--type=json").Output()

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady(oc)

		g.By("Verify all eBPF pods are deployed on the above worker node")
		eBPFpods, err := compat_otp.GetAllPodsWithLabel(oc, flow.Namespace+"-privileged", "app=netobserv-ebpf-agent")
		o.Expect(err).NotTo(o.HaveOccurred())
		for _, pod := range eBPFpods {
			nodeName, err := compat_otp.GetPodNodeName(oc, flow.Namespace+"-privileged", pod)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(nodeName).To(o.Equal(workerNode))
		}
	})

	g.Context("with Loki", func() {
		var (
			lokiDir, _ = filePath.Abs("testdata/loki")
			// Loki Operator variables
			lokiPackageName = "loki-operator"
			lokiSource      CatalogSourceObjects
			lokiCatalog     = "redhat-operators"
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
			ipStackType = checkIPStackType(oc)

			g.By("Deploy loki operator")
			if !validateInfraAndResourcesForLoki(oc, "10Gi", "6") {
				g.Skip("Current platform does not have enough resources available for this test!")
			}

			// check if Loki Operator exists
			var err error
			Lokiexisting, err = CheckOperatorStatus(oc, LO.Namespace, LO.PackageName)
			o.Expect(err).NotTo(o.HaveOccurred())

			lokiChannel, err := getOperatorChannel(oc, lokiCatalog, lokiPackageName)
			if err != nil || lokiChannel == "" {
				g.Skip("Loki channel not found, skip this case")
			}
			lokiSource = CatalogSourceObjects{lokiChannel, lokiCatalog, "openshift-marketplace"}

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

		g.Context("FLP, eBPF and Console metrics:", func() {
			g.When("processor.metrics.TLS == Disabled and agent.ebpf.metrics.TLS == Disabled", func() {
				g.It("Author:aramesha-Critical-50504-Critical-72959-Verify flowlogs-pipeline and eBPF metrics and health [Serial]", func() {
					SkipIfOCPBelow(4, 12)
					var (
						flpPromSM  = "flowlogs-pipeline-monitor"
						namespace  = oc.Namespace()
						eBPFPromSM = "ebpf-agent-svc-monitor"
						curlLive   = "http://localhost:8080/live"
					)

					g.By("Deploy flowcollector")
					flow := Flowcollector{
						Namespace:              namespace,
						Template:               flowFixturePath,
						LokiNamespace:          lokiStackNS,
						FLPMetricServerTLSType: "Disabled",
					}

					defer func() { _ = flow.DeleteFlowcollector(oc) }()
					flow.CreateFlowcollector(oc)

					g.By("Verify flowlogs-pipeline metrics")
					FLPpods, err := compat_otp.GetAllPodsWithLabel(oc, namespace, "app=flowlogs-pipeline")
					o.Expect(err).NotTo(o.HaveOccurred())

					for _, pod := range FLPpods {
						command := []string{"exec", "-n", namespace, pod, "--", "curl", "-s", curlLive}
						output, err := oc.AsAdmin().WithoutNamespace().Run(command...).Args().Output()
						o.Expect(err).NotTo(o.HaveOccurred())
						o.Expect(output).To(o.Equal("{}"))
					}

					FLPtlsScheme, err := getMetricsScheme(oc, flpPromSM, flow.Namespace)
					o.Expect(err).NotTo(o.HaveOccurred())
					FLPtlsScheme = strings.Trim(FLPtlsScheme, "'")
					o.Expect(FLPtlsScheme).To(o.Equal("http"))

					g.By("Wait for a min before scraping metrics")
					time.Sleep(60 * time.Second)

					g.By("Verify prometheus is able to scrape FLP metrics")
					verifyFLPMetrics(oc)

					g.By("Verify eBPF agent metrics")
					eBPFpods, err := compat_otp.GetAllPodsWithLabel(oc, namespace, "app=netobserv-ebpf-agent")
					o.Expect(err).NotTo(o.HaveOccurred())

					for _, pod := range eBPFpods {
						command := []string{"exec", "-n", namespace, pod, "--", "curl", "-s", curlLive}
						output, err := oc.AsAdmin().WithoutNamespace().Run(command...).Args().Output()
						o.Expect(err).NotTo(o.HaveOccurred())
						o.Expect(output).To(o.Equal("{}"))
					}

					eBPFtlsScheme, err := getMetricsScheme(oc, eBPFPromSM, flow.Namespace+"-privileged")
					o.Expect(err).NotTo(o.HaveOccurred())
					eBPFtlsScheme = strings.Trim(eBPFtlsScheme, "'")
					o.Expect(eBPFtlsScheme).To(o.Equal("http"))

					g.By("Wait for a min before scraping metrics")
					time.Sleep(60 * time.Second)

					g.By("Verify prometheus is able to scrape eBPF metrics")
					verifyEBPFMetrics(oc)
				})
			})

			g.When("processor.metrics.TLS == Auto and ebpf.agent.metrics.TLS == Auto", func() {
				g.It("Author:aramesha-Critical-54043-Critical-66031-Critical-72959-Verify flowlogs-pipeline, eBPF and Console metrics [Serial]", func() {
					SkipIfOCPBelow(4, 12)
					var (
						flpPromSM  = "flowlogs-pipeline-monitor"
						flpPromSA  = "flowlogs-pipeline-prom"
						eBPFPromSM = "ebpf-agent-svc-monitor"
						eBPFPromSA = "ebpf-agent-svc-prom"
						namespace  = oc.Namespace()
					)

					flow := Flowcollector{
						Namespace:               namespace,
						Template:                flowFixturePath,
						LokiNamespace:           lokiStackNS,
						EBPFMetricServerTLSType: "Auto",
					}

					defer func() { _ = flow.DeleteFlowcollector(oc) }()
					flow.CreateFlowcollector(oc)

					g.By("Verify flowlogs-pipeline metrics")
					FLPtlsScheme, err := getMetricsScheme(oc, flpPromSM, flow.Namespace)
					o.Expect(err).NotTo(o.HaveOccurred())
					FLPtlsScheme = strings.Trim(FLPtlsScheme, "'")
					o.Expect(FLPtlsScheme).To(o.Equal("https"))

					FLPserverName, err := getMetricsServerName(oc, flpPromSM, flow.Namespace)
					FLPserverName = strings.Trim(FLPserverName, "'")
					o.Expect(err).NotTo(o.HaveOccurred())
					FLPexpectedServerName := fmt.Sprintf("%s.%s.svc", flpPromSA, namespace)
					o.Expect(FLPserverName).To(o.Equal(FLPexpectedServerName))

					g.By("Wait for a min before scraping metrics")
					time.Sleep(60 * time.Second)

					g.By("Verify prometheus is able to scrape FLP and Console metrics")
					verifyFLPMetrics(oc)
					query := fmt.Sprintf("process_start_time_seconds{namespace=\"%s\", job=\"netobserv-plugin-metrics\"}", namespace)
					metrics, err := getMetric(oc, query)
					o.Expect(err).NotTo(o.HaveOccurred())
					o.Expect(popMetricValue(metrics)).Should(o.BeNumerically(">", 0))

					g.By("Verify eBPF metrics")
					eBPFtlsScheme, err := getMetricsScheme(oc, eBPFPromSM, flow.Namespace+"-privileged")
					o.Expect(err).NotTo(o.HaveOccurred())
					eBPFtlsScheme = strings.Trim(eBPFtlsScheme, "'")
					o.Expect(eBPFtlsScheme).To(o.Equal("https"))

					eBPFserverName, err := getMetricsServerName(oc, eBPFPromSM, flow.Namespace+"-privileged")
					eBPFserverName = strings.Trim(eBPFserverName, "'")
					o.Expect(err).NotTo(o.HaveOccurred())
					eBPFexpectedServerName := fmt.Sprintf("%s.%s.svc", eBPFPromSA, namespace+"-privileged")
					o.Expect(eBPFserverName).To(o.Equal(eBPFexpectedServerName))

					g.By("Verify prometheus is able to scrape eBPF agent metrics")
					verifyEBPFMetrics(oc)
				})
			})
		})

		g.It("Author:memodi-High-53595-High-49107-High-45304-High-54929-High-54840-High-68310-Verify flow correctness and metrics [Serial]", func() {
			SkipIfOCPBelow(4, 11)
			g.By("Deploying test server and client pods")
			serverTemplatePath := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServer := TestServerTemplate{
				ServerNS: "test-server-54929",
				Template: serverTemplatePath,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServer.ServerNS)
			err := testServer.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServer.ServerNS)

			clientTemplatePath := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClient := TestClientTemplate{
				ServerNS:   testServer.ServerNS,
				ClientNS:   "test-client-54929",
				ObjectSize: "100K",
				Template:   clientTemplatePath,
			}

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClient.ClientNS)
			err = testClient.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClient.ClientNS)

			startTime := time.Now()

			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("get flowlogs from loki")
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testServer.ServerNS,
				DstK8SNamespace: testClient.ClientNS,
				SrcK8SOwnerName: "nginx-service",
				FlowDirection:   "0",
			}

			g.By("Wait for 2 mins before logs gets collected and written to loki")
			time.Sleep(120 * time.Second)

			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords > 0")

			// verify flow correctness
			verifyFlowCorrectness(testClient.ObjectSize, flowRecords)

			// verify inner metrics
			query := fmt.Sprintf(`sum(rate(netobserv_workload_ingress_bytes_total{SrcK8SNamespace="%s"}[1m]))`, testClient.ClientNS)
			metrics := pollMetrics(oc, query)

			// verfy metric is between 270 and 330
			o.Expect(metrics).Should(o.BeNumerically("~", 330, 270))
		})

		g.It("Author:aramesha-NonPreRelease-Longduration-High-60701-Verify connection tracking [Serial]", func() {
			SkipIfOCPBelow(4, 10)
			startTime := time.Now()

			g.By("Deploying test server and client pods")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-60701",
				Template: serverTemplate,
			}

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")

			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-60701",
				Template: clientTemplate,
			}

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			g.By("Deploy FlowCollector with endConversations LogType")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LogType:       "EndedConversations",
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testClientTemplate.ClientNS,
				DstK8SNamespace: testClientTemplate.ServerNS,
				RecordType:      "endConnection",
				DstK8SOwnerName: "nginx-service",
			}

			g.By("Verify endConnection Records from loki")
			endConnectionRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(endConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of endConnectionRecords > 0")
			verifyConversationRecordTime(endConnectionRecords)

			g.By("Deploy FlowCollector with Conversations LogType")
			_ = flow.DeleteFlowcollector(oc)

			flow.LogType = "Conversations"
			flow.CreateFlowcollector(oc)

			g.By("Wait for a min before logs gets collected and written to loki")
			startTime = time.Now()
			time.Sleep(60 * time.Second)

			g.By("Verify NewConnection Records from loki")
			lokilabels.RecordType = "newConnection"

			newConnectionRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(newConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of newConnectionRecords > 0")
			verifyConversationRecordTime(newConnectionRecords)

			g.By("Verify HeartbeatConnection Records from loki")
			lokilabels.RecordType = "heartbeat"
			heartbeatConnectionRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(heartbeatConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of heartbeatConnectionRecords > 0")
			verifyConversationRecordTime(heartbeatConnectionRecords)

			g.By("Verify EndConnection Records from loki")
			lokilabels.RecordType = "endConnection"
			endConnectionRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(endConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of endConnectionRecords > 0")
			verifyConversationRecordTime(endConnectionRecords)
		})

		g.It("Author:memodi-NonPreRelease-Longduration-High-63839-Verify-multi-tenancy [Disruptive][Slow]", func() {
			SkipIfOCPBelow(4, 15)
			users, usersHTpassFile, htPassSecret := getNewUser(oc, 2)
			defer userCleanup(oc, users, usersHTpassFile, htPassSecret)

			g.By("Creating client server template and template CRBs for testusers")
			// create templates for testuser to be used later
			testUserstemplate := filePath.Join(baseDir, "testuser-client-server_template.yaml")
			stdout, stderr, err := oc.AsAdmin().Run("apply").Args("-f", testUserstemplate).Outputs()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(stderr).To(o.BeEmpty())
			templateResource := strings.Split(stdout, " ")[0]
			templateName := strings.Split(templateResource, "/")[1]
			defer removeTemplatePermissions(oc, users[0].Username)
			addTemplatePermissions(oc, users[0].Username)

			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Deploying test server and client pods")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-63839",
				Template: serverTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err = testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")

			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-63839",
				Template: clientTemplate,
			}

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)
			o.Expect(err).NotTo(o.HaveOccurred())

			// save original context
			origContxt, contxtErr := oc.AsAdmin().WithoutNamespace().Run("config").Args("current-context").Output()
			o.Expect(contxtErr).NotTo(o.HaveOccurred())
			e2e.Logf("orginal context is %v", origContxt)
			defer removeUserAsReader(oc, users[0].Username)
			addUserAsReader(oc, users[0].Username)
			origUser := oc.Username()

			e2e.Logf("current user is %s", origUser)
			defer func() { _ = oc.AsAdmin().WithoutNamespace().Run("config").Args("use-context", origContxt).Execute() }()
			defer oc.ChangeUser(origUser)
			oc.ChangeUser(users[0].Username)

			curUser := oc.Username()
			e2e.Logf("current user is %s", curUser)

			o.Expect(err).NotTo(o.HaveOccurred())
			user0Contxt, contxtErr := oc.WithoutNamespace().Run("config").Args("current-context").Output()
			o.Expect(contxtErr).NotTo(o.HaveOccurred())

			e2e.Logf("user0 context is %v", user0Contxt)

			g.By("Deploying test server and client pods as user0")
			var (
				testUserServerNS = fmt.Sprintf("%s-server", users[0].Username)
				testUserClientNS = fmt.Sprintf("%s-client", users[0].Username)
			)

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testUserClientNS)
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testUserServerNS)
			configFile := compat_otp.ProcessTemplate(oc, "--ignore-unknown-parameters=true", templateName, "-p", "SERVER_NS="+testUserServerNS, "-p", "CLIENT_NS="+testUserClientNS)
			err = oc.WithoutNamespace().Run("create").Args("-f", configFile).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			// only required to getFlowLogs
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testUserServerNS,
				DstK8SNamespace: testUserClientNS,
				SrcK8SOwnerName: "nginx-service",
				FlowDirection:   "0",
			}

			user0token, err := oc.WithoutNamespace().Run("whoami").Args("-t").Output()
			e2e.Logf("token is %s", user0token)
			o.Expect(err).NotTo(o.HaveOccurred())

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
			}
			flowRecords, err = lokilabels.getLokiFlowLogs(user0token, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).NotTo(o.BeNumerically(">", 0), "expected number of flowRecords to be equal to 0")
		})

		g.It("Author:aramesha-NonPreRelease-Critical-59746-NetObserv upgrade testing [Serial]", func() {
			SkipIfOCPBelow(4, 10)
			g.By("Uninstall operator deployed by BeforeEach and delete operator NS")
			NO.uninstallOperator(oc)
			oc.DeleteSpecifiedNamespaceAsAdmin(netobservNS)
			err := Resource{"namespace", netobservNS, ""}.WaitUntilResourceIsGone(oc)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Deploy older version of netobserv operator")
			NOcatSrc = Resource{"catsrc", "redhat-operators", "openshift-marketplace"}
			NOSource = CatalogSourceObjects{"stable", NOcatSrc.Name, NOcatSrc.Namespace}

			NO.CatalogSource = &NOSource

			g.By(fmt.Sprintf("Subscribe operators to %s channel", NOSource.Channel))
			OperatorNS.DeployOperatorNamespace(oc)
			NO.SubscribeOperator(oc)
			// check if NO operator is deployed
			WaitForPodsReadyWithLabel(oc, netobservNS, "app="+NO.OperatorName)
			NOStatus, err := CheckOperatorStatus(oc, netobservNS, NOPackageName)
			o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("found err %v", err))
			o.Expect((NOStatus)).To(o.BeTrue())

			// check if flowcollector API exists
			flowcollectorAPIExists, err := isFlowCollectorAPIExists(oc)
			o.Expect((flowcollectorAPIExists)).To(o.BeTrue())
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Get NetObserv and components versions")
			NOCSV, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[?(@.name=='OPERATOR_CONDITION_NAME')].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())

			preUpgradeNOVersion := strings.Split(NOCSV, ".v")[1]
			preUpgradeEBPFVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[0].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			preUpgradeEBPFVersion = strings.Split(preUpgradeEBPFVersion, ":")[1]
			preUpgradeFLPVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[1].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			preUpgradeFLPVersion = strings.Split(preUpgradeFLPVersion, ":")[1]
			preUpgradePluginVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[2].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			preUpgradePluginVersion = strings.Split(preUpgradePluginVersion, ":")[1]

			g.By("Deploy latest catalog and upgrade to latest version")
			NOcatSrc.Name = "netobserv-konflux-fbc"
			NOcatSrc.Namespace = OperatorNS.Name
			var catsrcErr error
			if catalogSource != "" {
				e2e.Logf("Using %s catalog", catalogSource)
				catsrcErr = NOcatSrc.applyFromTemplate(oc, "-n", NOcatSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+NOcatSrc.Namespace, "IMAGE="+catalogSource)
			} else {
				e2e.Logf("Using default ystream catalog")
				catsrcErr = NOcatSrc.applyFromTemplate(oc, "-n", NOcatSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+NOcatSrc.Namespace)
			}
			o.Expect(catsrcErr).NotTo(o.HaveOccurred())
			_, _ = oc.AsAdmin().WithoutNamespace().Run("patch").Args("subscription", "netobserv-operator", "-n", netobservNS, "-p", `[{"op": "replace", "path": "/spec/source", "value": `+NOcatSrc.Name+`}, {"op": "replace", "path": "/spec/sourceNamespace", "value": `+NOcatSrc.Namespace+`}]`, "--type=json").Output()

			g.By("Wait for a min for operator upgrade")
			time.Sleep(60 * time.Second)

			WaitForPodsReadyWithLabel(oc, netobservNS, "app=netobserv-operator")
			NOStatus, err = CheckOperatorStatus(oc, netobservNS, NOPackageName)
			o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("found err %v", err))
			o.Expect((NOStatus)).To(o.BeTrue())

			g.By("Get NetObserv operator and components versions")
			NOCSV, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[?(@.name=='OPERATOR_CONDITION_NAME')].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())

			postUpgradeNOVersion := strings.Split(NOCSV, ".v")[1]
			postUpgradeEBPFVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[0].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			postUpgradeEBPFVersion = strings.Split(postUpgradeEBPFVersion, ":")[1]
			postUpgradeFLPVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[1].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			postUpgradeFLPVersion = strings.Split(postUpgradeFLPVersion, ":")[1]
			postUpgradePluginVersion, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-l", "app=netobserv-operator", "-n", netobservNS, "-o=jsonpath={.items[*].spec.containers[0].env[2].value}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			postUpgradePluginVersion = strings.Split(postUpgradePluginVersion, ":")[1]

			g.By("Verify versions are updated")
			o.Expect(preUpgradeNOVersion).NotTo(o.Equal(postUpgradeNOVersion))
			o.Expect(preUpgradeEBPFVersion).NotTo(o.Equal(postUpgradeEBPFVersion))
			o.Expect(preUpgradeFLPVersion).NotTo(o.Equal(postUpgradeFLPVersion))
			o.Expect(preUpgradePluginVersion).NotTo(o.Equal(postUpgradePluginVersion))

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			g.By("Get flowlogs from loki")
			err = verifyLokilogsTime(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
		})

		g.It("Author:aramesha-NonPreRelease-High-62989-Verify SCTP, ICMP, ICMPv6 traffic is observed [Disruptive]", func() {
			SkipIfOCPBelow(4, 10)
			var (
				sctpClientPodTemplatePath = filePath.Join(networkingDir, "sctpclient.yaml")
				sctpServerPodTemplatePath = filePath.Join(networkingDir, "sctpserver.yaml")
				sctpServerPodname         = "sctpserver"
				sctpClientPodname         = "sctpclient"
			)

			g.By("install load-sctp-module in all workers")
			prepareSCTPModule(oc)

			g.By("Create netobserv-sctp NS")
			SCTPns := "netobserv-sctp-62989"
			defer oc.DeleteSpecifiedNamespaceAsAdmin(SCTPns)
			oc.CreateSpecifiedNamespaceAsAdmin(SCTPns)
			_ = compat_otp.SetNamespacePrivileged(oc, SCTPns)

			g.By("create sctpClientPod")
			createResourceFromFile(oc, SCTPns, sctpClientPodTemplatePath)
			WaitForPodsReadyWithLabel(oc, SCTPns, "name=sctpclient")

			g.By("create sctpServerPod")
			createResourceFromFile(oc, SCTPns, sctpServerPodTemplatePath)
			WaitForPodsReadyWithLabel(oc, SCTPns, "name=sctpserver")

			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("get primary IP address of sctpServerPod")
			sctpServerPodIP, _ := getPodIP(oc, SCTPns, sctpServerPodname, ipStackType)

			g.By("sctpserver pod start to wait for sctp traffic")
			cmd, _, _, _ := oc.AsAdmin().Run("exec").Args("-n", SCTPns, sctpServerPodname, "--", "/usr/bin/ncat", "-l", "30102", "--sctp").Background()
			defer func() { _ = cmd.Process.Kill() }()
			time.Sleep(5 * time.Second)

			g.By("check sctp process enabled in the sctp server pod")
			msg, err := e2eoutput.RunHostCmd(SCTPns, sctpServerPodname, "ps aux | grep sctp")
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(strings.Contains(msg, "/usr/bin/ncat -l 30102 --sctp")).To(o.BeTrue())

			g.By("sctpclient pod start to send sctp traffic")
			startTime := time.Now()
			_, _ = e2eoutput.RunHostCmd(SCTPns, sctpClientPodname, "echo 'Test traffic using sctp port from sctpclient to sctpserver' | { ncat -v "+sctpServerPodIP+" 30102 --sctp; }")

			g.By("server sctp process will end after get sctp traffic from sctp client")
			time.Sleep(5 * time.Second)
			msg1, err1 := e2eoutput.RunHostCmd(SCTPns, sctpServerPodname, "ps aux | grep sctp")
			o.Expect(err1).NotTo(o.HaveOccurred())
			o.Expect(msg1).NotTo(o.ContainSubstring("/usr/bin/ncat -l 30102 --sctp"))

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			// Scenario1: Verify SCTP traffic
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: SCTPns,
				DstK8SNamespace: SCTPns,
			}

			g.By("Verify SCTP flows are seen on loki")
			parameters := []string{"Proto=\"132\"", "DstPort=\"30102\""}

			SCTPflows, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(SCTPflows)).Should(o.BeNumerically(">", 0), "expected number of SCTP flows > 0")

			// Scenario2: Verify ICMP traffic
			g.By("sctpclient ping sctpserver")
			_, _ = e2eoutput.RunHostCmd(SCTPns, sctpClientPodname, "ping -c 10 "+sctpServerPodIP)
			ICMPEchoReq := 8
			ICMPEchoRes := 0
			if ipStackType == "ipv4single" {
				parameters = []string{"Proto=\"1\""}
			}
			g.By("test ipv6 in ipv6 cluster or dualstack cluster")
			if ipStackType == "ipv6single" || ipStackType == "dualstack" {
				parameters = []string{"Proto=\"58\""}
				ICMPEchoReq = 128
				ICMPEchoRes = 129
			}

			g.By("Wait for a min before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			ICMPflows, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(ICMPflows)).Should(o.BeNumerically(">", 0), "expected number of ICMP flows > 0")

			nICMPFlows := 0
			for _, r := range ICMPflows {
				if r.Flowlog.IcmpType == ICMPEchoReq || r.Flowlog.IcmpType == ICMPEchoRes {
					nICMPFlows++
				}
			}
			o.Expect(nICMPFlows).Should(o.BeNumerically(">", 0), "expected number of ICMP flows of type 8/128 or 0/129 (echo request or reply) > 0")
		})

		g.It("Author:aramesha-NonPreRelease-High-68125-Verify DSCP with NetObserv [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploying test server and client pods")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-68125",
				Template: serverTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-68125",
				Template: clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			compat_otp.By("Check cluster network type")
			networkType := compat_otp.CheckNetworkType(oc)
			o.Expect(networkType).NotTo(o.BeEmpty())
			if networkType == "ovnkubernetes" {
				g.By("Deploy egressQoS for OVN CNI")
				clientDSCPPath := filePath.Join(networkingDir, "test-client-DSCP.yaml")
				egressQoSPath := filePath.Join(networkingDir, "egressQoS.yaml")
				g.By("Deploy nginx client pod and egressQoS")
				createResourceFromFile(oc, testClientTemplate.ClientNS, clientDSCPPath)
				createResourceFromFile(oc, testClientTemplate.ClientNS, egressQoSPath)
			}

			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			// Scenario1: Verify default DSCP value=0
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testClientTemplate.ClientNS,
				DstK8SNamespace: testClientTemplate.ServerNS,
			}
			parameters := []string{"SrcK8S_Name=\"client\""}

			g.By("Verify DSCP value=0")
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Dscp).To(o.Equal(0))
			}

			// Scenario2: Verify egress QoS feature for OVN CNI
			if networkType == "ovnkubernetes" {
				parameters = []string{"SrcK8S_Name=\"client-dscp\", Dscp=\"59\""}

				g.By("Wait for a min before logs gets collected and written to loki")
				time.Sleep(60 * time.Second)

				g.By("Verify DSCP value=59 for flows from DSCP client pod")
				flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 59 should be > 0")

				g.By("Verify DSCP value=0 for flows from pods other than DSCP client pod in test-client namespace")
				parameters = []string{"SrcK8S_Name=\"client\", Dscp=\"0\""}

				flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 0 should be > 0")
			}

			// Scenario3: Explicitly passing QoS value in ping command
			var destinationIP string
			switch ipStackType {
			case "ipv4single":
				destinationIP = "1.1.1.1"
			case "ipv6single":
				destinationIP = "::1"
			default:
				destinationIP = "1.1.1.1"
			}

			g.By("Ping loopback address with custom QoS from client pod")
			startTime = time.Now()
			_, _ = e2eoutput.RunHostCmd(testClientTemplate.ClientNS, "client", "ping -c 10 -Q 0x80 "+destinationIP)

			lokilabels = Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testClientTemplate.ClientNS,
			}
			parameters = []string{"Dscp=\"32\", DstAddr=\"" + destinationIP + "\""}

			g.By("Wait for a min before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			g.By("Verify DSCP value=32")
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 32 > 0")
		})

		g.It("Author:aramesha-NonPreRelease-High-69218-High-71291-Verify cluster ID and zone in multiCluster deployment [Serial]", func() {
			SkipIfOCPBelow(4, 11)
			g.By("Get clusterID of the cluster")
			clusterID, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion", "-o=jsonpath={.items[].spec.clusterID}").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.Logf("Cluster ID is %s", clusterID)

			g.By("Deploy FlowCollector with multiCluster and addZone enabled")
			flow := Flowcollector{
				Namespace:              namespace,
				MultiClusterDeployment: "true",
				AddZone:                "true",
				Template:               flowFixturePath,
				LokiNamespace:          lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			g.By("Verify K8SClusterName = Cluster ID")
			clusteridlabels := Lokilabels{
				App:            "netobserv-flowcollector",
				K8SClusterName: clusterID,
			}
			clusterIDFlowRecords, err := clusteridlabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(clusterIDFlowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows > 0")

			g.By("Verify SrcK8S_Zone and DstK8S_Zone are present and have expected values")
			zonelabels := Lokilabels{
				App:        "netobserv-flowcollector",
				SrcK8SType: "Node",
				DstK8SType: "Node",
			}

			zoneFlowRecords, err := zonelabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			for _, r := range zoneFlowRecords {
				expectedSrcK8SZone, err := compat_otp.GetResourceSpecificLabelValue(oc, "node/"+r.Flowlog.SrcK8SHostName, "", `topology.kubernetes.io/zone`)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(r.Flowlog.SrcK8SZone).To(o.Equal(expectedSrcK8SZone))

				expectedDstK8SZone, err := compat_otp.GetResourceSpecificLabelValue(oc, "node/"+r.Flowlog.DstK8SHostName, "", `topology.kubernetes.io/zone`)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(r.Flowlog.DstK8SZone).To(o.Equal(expectedDstK8SZone))
			}
		})

		g.It("Author:aramesha-NonPreRelease-Longduration-High-73175-Verify eBPF agent filtering [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploy test server and client pods")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-73175",
				Template: serverTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-73175",
				Template: clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			clientServiceInfo, err := getClientServerInfo(oc, testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
			o.Expect(err).NotTo(o.HaveOccurred())

			// Scenario 1:
			// Accept TCP flows between client pod and nginx-service
			// Accept ICMP flows between client and nginx pod
			// Default Reject all other flows
			g.By("Deploy FlowCollector with eBPF filter")
			filterRulesConfig := []map[string]interface{}{
				{
					"action":   "Accept",
					"cidr":     clientServiceInfo["service"]["ip"] + "/32",
					"peerIP":   clientServiceInfo["client"]["ip"],
					"protocol": "TCP",
					"ports":    "80",
					"sampling": 2,
				},
				{
					"action":   "Accept",
					"cidr":     clientServiceInfo["client"]["ip"] + "/32",
					"peerCIDR": clientServiceInfo["server"]["ip"] + "/32",
					"protocol": "ICMP",
					"icmpType": 8,
					"sampling": 3,
				},
			}

			config, err := json.Marshal(filterRulesConfig)
			o.Expect(err).ToNot(o.HaveOccurred())
			filter := string(config)

			flow := Flowcollector{
				Namespace:       namespace,
				Template:        flowFixturePath,
				LokiNamespace:   lokiStackNS,
				EBPFFilterRules: filter,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Ping nginx pod from client pod")
			startTime := time.Now()
			_, _ = e2eoutput.RunHostCmd(testClientTemplate.ClientNS, clientServiceInfo["client"]["name"], "ping -c 10 "+clientServiceInfo["server"]["ip"])

			g.By("Wait for a min before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}

			g.By("Verify number of flows with on UDP Protcol with SrcPort 53 = 0")
			lokiParams := []string{"Proto=\"17\"", "SrcPort=\"53\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically("==", 0), "expected number of flows on UDP with SrcPort 53 = 0")

			g.By("Verify flows from client pod to nginx pod > 0")
			lokilabels.SrcK8SNamespace = testClientTemplate.ClientNS
			lokilabels.DstK8SNamespace = testClientTemplate.ServerNS
			lokiParams = []string{"SrcAddr=" + "\"" + clientServiceInfo["client"]["ip"] + "\"", "DstAddr=" + "\"" + clientServiceInfo["server"]["ip"] + "\""}

			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client pod to nginx pod > 0")

			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Proto).Should(o.BeNumerically("==", 1))
				o.Expect(r.Flowlog.IcmpType).Should(o.BeNumerically("==", 8))
				o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 3))
			}

			g.By("Verify flows from client pod to nginx-service > 0")
			lokilabels.DstK8SType = "Service"

			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client pod to nginx-service > 0")

			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Proto).Should(o.BeNumerically("==", 6))
				o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 2))
			}

			g.By("Verify prometheus is able to scrape eBPF metrics")
			verifyEBPFFilterMetrics(oc, "FilterAccept")
			verifyEBPFFilterMetrics(oc, "FilterNoMatch")

			// Scenario2:
			// Accept only flows with drops
			g.By("Deploy flowcollector with eBPF filter for flows with drops")
			filterRulesConfig = []map[string]interface{}{
				{
					"action":   "Accept",
					"cidr":     "172.30.0.0/16",
					"pktDrops": true,
				},
			}

			config, err = json.Marshal(filterRulesConfig)
			o.Expect(err).ToNot(o.HaveOccurred())
			filter = string(config)

			_ = flow.DeleteFlowcollector(oc)
			flow.EBPFPrivileged = "true"
			flow.EBPFeatures = []string{"\"PacketDrop\""}
			flow.EBPFFilterRules = filter
			flow.CreateFlowcollector(oc)

			g.By("Wait for a min before logs gets collected and written to loki")
			startTime = time.Now()
			time.Sleep(60 * time.Second)

			lokilabels = Lokilabels{
				App: "netobserv-flowcollector",
			}
			lokiParams = []string{"Proto=\"6\""}

			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with drops > 0")

			for _, r := range flowRecords {
				o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			}
		})

		g.It("Author:memodi-Critical-53844-Sanity Test NetObserv [Serial]", func() {
			SkipIfOCPBelow(4, 11)
			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}

			g.By("Verify flows are written to loki")
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")
		})

		g.It("Author:aramesha-High-67782-Verify large volume downloads [Serial]", func() {
			SkipIfOCPBelow(4, 11)
			g.By("Deploy FlowCollector")
			flow := Flowcollector{
				Namespace:              namespace,
				Template:               flowFixturePath,
				LokiNamespace:          lokiStackNS,
				EBPFCacheActiveTimeout: "30s",
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Deploy test server and client pods")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS:  "test-server-67782",
				Template:  serverTemplate,
				LargeBlob: "yes",
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS:   testServerTemplate.ServerNS,
				ClientNS:   "test-client-67782",
				ObjectSize: "100M",
				Template:   clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			g.By("Wait for 2 mins before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(120 * time.Second)

			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testClientTemplate.ServerNS,
				DstK8SNamespace: testClientTemplate.ClientNS,
				SrcK8SOwnerName: "nginx-service",
				FlowDirection:   "0",
			}

			g.By("Verify flows are written to loki")
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

			g.By("Verify flow correctness")
			verifyFlowCorrectness(testClientTemplate.ObjectSize, flowRecords)
		})

		g.It("Author:aramesha-High-75656-Verify TCP flags [Disruptive]", func() {
			SkipIfOCPBelow(4, 13)
			SYNFloodMetricsPath := filePath.Join(baseDir, "SYN_flood_metrics_template.yaml")
			SYNFloodAlertsPath := filePath.Join(baseDir, "SYN_flood_alert_template.yaml")

			g.By("Deploy flowcollector with eBPF filter to Reject flows with tcpFlags SYN-ACK and TCP Protocol")
			filterRulesConfig := []map[string]string{
				{
					"action":   "Reject",
					"cidr":     "0.0.0.0/0",
					"protocol": "TCP",
					"tcpFlags": "SYN-ACK",
				},
			}

			config, err := json.Marshal(filterRulesConfig)
			o.Expect(err).ToNot(o.HaveOccurred())
			filter := string(config)

			flow := Flowcollector{
				Namespace:       namespace,
				Template:        flowFixturePath,
				LokiNamespace:   lokiStackNS,
				EBPFFilterRules: filter,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Ensure flowcollector is ready with Reject flowFilter")
			flowPatch, err := oc.AsAdmin().Run("get").Args("flowcollector", "cluster", "-o", "jsonpath='{.spec.agent.ebpf.flowFilter.rules[0].action}'").Output()
			o.Expect(err).ToNot(o.HaveOccurred())
			o.Expect(flowPatch).To(o.Equal(`'Reject'`))

			g.By("Deploy custom metrics to detect SYN flooding")
			customMetrics := CustomMetrics{
				Namespace: namespace,
				Template:  SYNFloodMetricsPath,
			}

			curv, err := getResourceVersion(oc, "cm", "flowlogs-pipeline-config-dynamic", namespace)
			o.Expect(err).NotTo(o.HaveOccurred())
			customMetrics.createCustomMetrics(oc)
			waitForResourceGenerationUpdate(oc, "cm", "flowlogs-pipeline-config-dynamic", "resourceVersion", curv, namespace)

			g.By("Deploy SYN flooding alert rule")
			defer oc.AsAdmin().WithoutNamespace().Run("delete").Args("alertingrule.monitoring.openshift.io", "netobserv-syn-alerts", "-n", "openshift-monitoring")
			configFile := compat_otp.ProcessTemplate(oc, "--ignore-unknown-parameters=true", "-f", SYNFloodAlertsPath, "-p", "Namespace=openshift-monitoring")
			err = oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
			o.Expect(err).ToNot(o.HaveOccurred())

			g.By("Deploy test client pod to induce SYN flooding")
			template := filePath.Join(baseDir, "test-SYN-flood-client_template.yaml")
			testTemplate := TestClientTemplate{
				ClientNS: "test-client-75656",
				Template: template,
			}

			defer oc.DeleteSpecifiedNamespaceAsAdmin(testTemplate.ClientNS)
			configFile = compat_otp.ProcessTemplate(oc, "--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "CLIENT_NS="+testTemplate.ClientNS)
			err = oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", configFile).Execute()
			o.Expect(err).ToNot(o.HaveOccurred())

			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}

			g.By("Verify no flows with SYN_ACK TCP flag")
			parameters := []string{"Flags=\"SYN_ACK\""}

			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			// Loop needed since even flows with flags SYN, ACK are matched
			count := 0
			for _, r := range flowRecords {
				for _, f := range r.Flowlog.Flags {
					o.Expect(f).ToNot(o.Equal("SYN_ACK"))
				}
			}
			o.Expect(count).Should(o.BeNumerically("==", 0), "expected number of flows with SYN_ACK TCPFlag = 0")
			verifyEBPFFilterMetrics(oc, "FilterReject")

			g.By("Verify SYN flooding flows")
			parameters = []string{"Flags=\"SYN\"", "DstAddr=\"192.168.1.159\""}

			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of SYN flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Bytes).Should(o.BeNumerically("==", 54))
			}

			g.By("Wait for alerts to be active")
			waitForAlertToBeActive(oc, "NetObserv-SYNFlood-out")
			waitForAlertToBeActive(oc, "NetObserv-SYNFlood-in")
		})

		g.It("Author:aramesha-NonPreRelease-Longduration-Medium-78480-NetObserv with sampling 50 [Serial][Slow]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploy DNS pods")
			DNSTemplate := filePath.Join(baseDir, "DNS-pods.yaml")
			DNSNamespace := "dns-traffic"
			defer oc.DeleteSpecifiedNamespaceAsAdmin(DNSNamespace)
			ApplyResourceFromFile(oc, DNSNamespace, DNSTemplate)
			compat_otp.AssertAllPodsToBeReady(oc, DNSNamespace)

			g.By("Deploy test server and client pods")
			servertemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-78480",
				Template: servertemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-78480",
				Template: clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			g.By("Deploy FlowCollector with all features enabled with sampling 50")
			flow := Flowcollector{
				Namespace:      namespace,
				EBPFPrivileged: "true",
				EBPFeatures:    []string{"\"DNSTracking\", \"PacketDrop\", \"FlowRTT\", \"PacketTranslation\""},
				Sampling:       "50",
				LokiNamespace:  lokiStackNS,
				Template:       flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for 4 mins before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(240 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}

			g.By("Verify Packet Drop flows")
			lokiParams := []string{"PktDropLatestState=\"TCP_INVALID_STATE\"", "Proto=\"6\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP Invalid State flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.PktDropLatestDropCause).NotTo(o.BeEmpty())
				o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			}

			lokiParams = []string{"PktDropLatestDropCause=\"SKB_DROP_REASON_NO_SOCKET\"", "Proto=\"6\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of No Socket TCP flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.PktDropLatestState).NotTo(o.BeEmpty())
				o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			}

			g.By("Verify flowRTT flows")
			lokiParams = []string{"Proto=\"6\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.TimeFlowRttNs).Should(o.BeNumerically(">=", 0))
			}

			g.By("Verify TCP DNS flows")
			lokilabels.DstK8SNamespace = DNSNamespace
			lokiParams = []string{"DnsFlagsResponseCode=\"NoError\"", "SrcPort=\"53\"", "DstK8S_Name=\"dnsutils1\"", "Proto=\"6\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP DNS flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.DNSLatencyMs).Should(o.BeNumerically(">=", 0))
			}

			g.By("Verify UDP DNS flows")
			lokiParams = []string{"DnsFlagsResponseCode=\"NoError\"", "SrcPort=\"53\"", "Proto=\"17\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of UDP DNS flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.DNSLatencyMs).Should(o.BeNumerically(">=", 0))
			}

			g.By("Verify Packet Translation flows")
			lokilabels = Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SType:      "Service",
				DstK8SNamespace: testClientTemplate.ServerNS,
				SrcK8SNamespace: testClientTemplate.ClientNS,
			}
			lokiParams = []string{"ZoneId>0"}

			g.By("Verify PacketTranslation flows")
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of PacketTranslation flows > 0")

			clientServiceInfo, err := getClientServerInfo(oc, testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
			verifyPacketTranslationFlows(clientServiceInfo["server"]["ip"], clientServiceInfo["server"]["name"], clientServiceInfo["client"]["ip"], flowRecords)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Verify eBPF feature metrics")
			verifyEBPFFeatureMetrics(oc, "pktdropsmap")
			verifyEBPFFeatureMetrics(oc, "additionalmap") // for RTT/IPSec map size
			verifyEBPFFeatureMetrics(oc, "dnsmap")
			verifyEBPFFeatureMetrics(oc, "xlatmap")
		})

		g.It("Author:aramesha-NonPreRelease-High-79015-Verify PacketTranslation feature [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploy test server and client pods")
			servertemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS:    "test-server-79015",
				ServiceType: "ClusterIP",
				Template:    servertemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-79015",
				Template: clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			g.By("Deploy FlowCollector with PacketTranslation feature enabled")
			flow := Flowcollector{
				Namespace:     namespace,
				EBPFeatures:   []string{"\"PacketTranslation\""},
				LokiNamespace: lokiStackNS,
				Template:      flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for 2 mins before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(120 * time.Second)

			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SType:      "Service",
				DstK8SNamespace: testClientTemplate.ServerNS,
				SrcK8SNamespace: testClientTemplate.ClientNS,
			}
			lokiParams := []string{"ZoneId>0"}

			g.By("Verify PacketTranslation flows")
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of PacketTranslation flows > 0")

			clientServiceInfo, err := getClientServerInfo(oc, testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
			o.Expect(err).NotTo(o.HaveOccurred())
			verifyPacketTranslationFlows(clientServiceInfo["server"]["ip"], clientServiceInfo["server"]["name"], clientServiceInfo["client"]["ip"], flowRecords)
		})

		// NetworkEvents ebpf hook only supported for OCP >= 4.19
		g.It("Author:memodi-NonPreRelease-Medium-77894-TechPreview Network Policies Correlation [Serial]", func() {
			SkipIfOCPBelow(4, 19)
			if !compat_otp.IsTechPreviewNoUpgrade(oc) {
				g.Skip("Skipping because the TechPreviewNoUpgrade is not enabled on the cluster.")
			}

			g.By("Deploy client-server pods in 2 client NS and one Server NS")
			serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-77894",
				Template: serverTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			client1Template := filePath.Join(baseDir, "test-nginx-client_template.yaml")
			testClient1Template := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client1-77894",
				Template: client1Template,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClient1Template.ClientNS)
			err = testClient1Template.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClient1Template.ClientNS)

			testClient2Template := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client2-77894",
				Template: client1Template,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClient2Template.ClientNS)
			err = testClient2Template.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClient2Template.ClientNS)

			// create flowcollector with NWEvents.
			flow := Flowcollector{
				Namespace:      namespace,
				Template:       flowFixturePath,
				LokiNamespace:  lokiStackNS,
				EBPFeatures:    []string{"\"NetworkEvents\""},
				EBPFPrivileged: "true",
			}
			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for 60 secs before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			g.By("get flowlogs from loki")
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SNamespace: testClient1Template.ServerNS,
				DstK8SType:      "Pod",
				SrcK8SType:      "Pod",
			}
			lokiParams := []string{"FlowDirection!=1"}
			lokilabels.SrcK8SNamespace = testClient1Template.ClientNS
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-2*time.Minute), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")

			g.By("deploy BANP policy")
			banpTemplate := filePath.Join(baseDir, "networking", "baselineadminnetworkPolicy.yaml")
			banpParameters := []string{"--ignore-unknown-parameters=true", "-p", "SERVER_NS=" + testClient1Template.ServerNS, "CLIENT1_NS=" + testClient1Template.ClientNS, "CLIENT2_NS=" + testClient2Template.ClientNS, "-f", banpTemplate}

			// banp is a cluster scoped resource so passing empty string for NS arg.
			defer deleteResource(oc, "banp", "default", "")
			err = compat_otp.ApplyClusterResourceFromTemplateWithError(oc, banpParameters...)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Wait for 60 secs before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			g.By("check flows have NW Events")
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-45*time.Second), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
			verifyNetworkEvents(flowRecords, Drop, "BaselineAdminNetworkPolicy", "Ingress")

			g.By("deploy NetworkPolicy")
			netpolTemplate := filePath.Join(baseDir, "networking", "networkPolicy.yaml")
			netpolName := "allow-ingress"
			netPolParameters := []string{"--ignore-unknown-parameters=true", "-p", "NAME=" + netpolName, "SERVER_NS=" + testClient1Template.ServerNS, "ALLOW_NS=" + testClient1Template.ClientNS, "-f", netpolTemplate}
			defer deleteResource(oc, "netpol", netpolName, testClient1Template.ServerNS)
			err = compat_otp.ApplyClusterResourceFromTemplateWithError(oc, netPolParameters...)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Wait for 60 secs before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			g.By("check flows from server to client1")
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-1*time.Minute), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
			verifyNetworkEvents(flowRecords, AllowRelated, "NetworkPolicy", "Ingress")

			g.By("check flows from server to client2")
			lokilabels.SrcK8SNamespace = testClient2Template.ClientNS
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-1*time.Minute), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
			verifyNetworkEvents(flowRecords, Drop, "NetpolNamespace", "Ingress")

			g.By("deploy ANP policy")
			anpTemplate := filePath.Join(baseDir, "networking", "adminnetworkPolicy.yaml")
			anpName := "server-ns"
			anpParameters := []string{"--ignore-unknown-parameters=true", "-p", "NAM=" + anpName, "SERVER_NS=" + testClient1Template.ServerNS, "ALLOW_NS=" + testClient2Template.ClientNS, "DENY_NS=" + testClient1Template.ClientNS, "-f", anpTemplate}
			defer deleteResource(oc, "anp", anpName, "")
			err = compat_otp.ApplyClusterResourceFromTemplateWithError(oc, anpParameters...)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Wait for 60 secs before logs gets collected and written to loki")
			time.Sleep(60 * time.Second)

			g.By("check flows from server to client2")
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-1*time.Minute), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
			verifyNetworkEvents(flowRecords, AllowRelated, "AdminNetworkPolicy", "Ingress")

			g.By("check flows from server to client1")
			lokilabels.SrcK8SNamespace = testClient1Template.ClientNS
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, time.Now().Add(-1*time.Minute), lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
			verifyNetworkEvents(flowRecords, Drop, "AdminNetworkPolicy", "Ingress")
		})

		g.It("Author:aramesha-NonPreRelease-High-80090-Verify FLP tail-based filtering [Serial]", func() {
			SkipIfOCPBelow(4, 15)
			// Accept flows with Source Namespace = < namespace > and
			// Source Name containing 'flowlogs-pipeline-' and
			// NOT Source Port 9401 and
			// having field TimeFlowRttNs
			g.By("Deploy FlowCollector with FLP tail-based filter and FlowRTT enabled")
			FLPFiltersConfig := []map[string]any{
				{
					"query":        fmt.Sprintf(`SrcK8SNamespace="%s" and SrcK8S_Name=~"flowlogs-pipeline-*" and SrcPort!=9401 and with(TimeFlowRttNs)`, namespace),
					"outputTarget": "Loki",
					"sampling":     2,
				},
			}

			config, err := json.Marshal(FLPFiltersConfig)
			o.Expect(err).ToNot(o.HaveOccurred())
			FLPFilter := string(config)

			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				Sampling:      "2",
				LokiNamespace: lokiStackNS,
				EBPFeatures:   []string{`"FlowRTT"`},
				FLPFilters:    FLPFilter,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			// verify logs
			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: namespace,
			}

			g.By("Verify number of flows > 0")
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows > 0")

			for _, r := range flowRecords {
				o.Expect(r.Flowlog.SrcK8SName).Should(o.ContainSubstring("flowlogs-pipeline-"))
				o.Expect(r.Flowlog.SrcPort).ShouldNot(o.BeNumerically("==", 9401))
				o.Expect(r.Flowlog.TimeFlowRttNs).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 4))
			}
		})

		g.It("Author:aramesha-High-81677-Validate UDN with NetObserv [Serial]", func() {
			SkipIfOCPBelow(4, 18)
			var (
				namespace           = oc.Namespace()
				networkingUDNDir, _ = filePath.Abs("testdata/networking/udn")
				udnPodTemplate      = filePath.Join(networkingUDNDir, "udn_test_pod_template.yaml")
				matchLabelKey       = "test.io"
				matchValue          = "netobserv-cudn-" + getRandomString()
				cudnName            = "cudn-network-81677"
				udnName             = "udn-network-81677"
				cudnNS              = []string{"netobserv-cudn1-81677", "netobserv-cudn2-81677"}
				udnNS               = "netobserv-udn-81677"
			)

			g.By("Create three namespaces, 2 for CUDN, 1 for UDN")
			defer deleteNamespace(oc, cudnNS[0])
			defer deleteNamespace(oc, cudnNS[1])
			oc.CreateSpecificNamespaceUDN(cudnNS[0])
			oc.CreateSpecificNamespaceUDN(cudnNS[1])
			for _, ns := range cudnNS {
				defer func() {
					_ = oc.AsAdmin().WithoutNamespace().Run("label").Args("ns", ns, fmt.Sprintf("%s-", matchLabelKey)).Execute()
				}()
				err := oc.AsAdmin().WithoutNamespace().Run("label").Args("ns", ns, fmt.Sprintf("%s=%s", matchLabelKey, matchValue)).Execute()
				o.Expect(err).NotTo(o.HaveOccurred())
			}

			defer deleteNamespace(oc, udnNS)
			oc.CreateSpecificNamespaceUDN(udnNS)

			g.By("Deploy CUDN in CUDNns")
			var cidr, ipv4cidr, ipv6cidr string
			if ipStackType == "ipv4single" {
				cidr = "10.150.0.0/16"
			} else {
				if ipStackType == "ipv6single" {
					cidr = "2010:100:200::0/60"
				} else {
					ipv4cidr = "10.150.0.0/16"
					ipv6cidr = "2010:100:200::0/60"
				}
			}
			defer removeResource(oc, true, true, "clusteruserdefinednetwork", cudnName)
			_, err := applyCUDNtoMatchLabelNS(oc, matchLabelKey, matchValue, cudnName, ipv4cidr, ipv6cidr, cidr, "layer3")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Deploy UDN in UDNns")
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
			createGeneralUDNCRD(oc, udnNS, udnName, ipv4cidr, ipv6cidr, cidr, "layer2")

			g.By("Deploy a pod in each CUDN namespace")
			CUDNpods := make([]udnPodResource, 2)
			for i, ns := range cudnNS {
				CUDNpods[i] = udnPodResource{
					name:      "hello-pod-" + ns,
					namespace: ns,
					label:     "hello-pod",
					template:  udnPodTemplate,
				}
				defer removeResource(oc, true, true, "pod", CUDNpods[i].name, "-n", CUDNpods[i].namespace)
				CUDNpods[i].createUdnPod(oc)
				compat_otp.AssertAllPodsToBeReady(oc, CUDNpods[i].namespace)
			}

			g.By("Deploy 2 pods in UDN namespace")
			UDNpods := make([]udnPodResource, 2)
			for j := range UDNpods {
				UDNpods[j] = udnPodResource{
					name:      fmt.Sprintf("hello-pod-%s-%d", udnNS, j),
					namespace: udnNS,
					label:     "hello-pod",
					template:  udnPodTemplate,
				}
				defer removeResource(oc, true, true, "pod", UDNpods[j].name, "-n", UDNpods[j].namespace)
				UDNpods[j].createUdnPod(oc)
			}
			compat_otp.AssertAllPodsToBeReady(oc, udnNS)

			g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
			flow := Flowcollector{
				Namespace:      namespace,
				EBPFPrivileged: "true",
				EBPFeatures:    []string{"\"UDNMapping\""},
				LokiNamespace:  lokiStackNS,
				Template:       flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			startTime := time.Now()

			g.By("Validate isolation from an UDN pod to a CUDN pod")
			CurlPod2PodFailUDN(oc, udnNS, UDNpods[1].name, CUDNpods[0].namespace, CUDNpods[0].name)
			//default network connectivity is isolated
			CurlPod2PodFail(oc, udnNS, UDNpods[1].name, CUDNpods[0].namespace, CUDNpods[0].name, ipStackType)

			g.By("Validate isolation from a CUDN pod to an UDN pod")
			CurlPod2PodFailUDN(oc, CUDNpods[1].namespace, CUDNpods[1].name, udnNS, UDNpods[1].name)
			//default network connectivity is isolated
			CurlPod2PodFail(oc, CUDNpods[1].namespace, CUDNpods[1].name, udnNS, UDNpods[1].name, ipStackType)

			g.By("Validate connection among CUDN pods")
			CurlPod2PodPassUDN(oc, CUDNpods[0].namespace, CUDNpods[0].name, CUDNpods[1].namespace, CUDNpods[1].name)
			//default network connectivity is isolated
			CurlPod2PodFail(oc, CUDNpods[0].namespace, CUDNpods[0].name, CUDNpods[1].namespace, CUDNpods[1].name, ipStackType)

			g.By("Validate connection among UDN pods")
			CurlPod2PodPassUDN(oc, udnNS, UDNpods[0].name, udnNS, UDNpods[1].name)
			//default network connectivity is isolated
			CurlPod2PodFail(oc, udnNS, UDNpods[1].name, udnNS, UDNpods[0].name, ipStackType)

			g.By("Wait for 3 mins before logs gets collected and written to loki")
			time.Sleep(180 * time.Second)

			g.By("Verify CUDN flows")
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SNamespace: cudnNS[1],
				DstK8SOwnerName: CUDNpods[1].name,
				SrcK8SNamespace: cudnNS[0],
				SrcK8SOwnerName: CUDNpods[0].name,
			}

			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of CUDN flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Udns).Should(o.ContainElement(cudnName))
				o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(cudnName))
				o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(cudnName))
			}

			g.By("Verify UDN flows")
			lokilabels = Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SNamespace: udnNS,
				DstK8SOwnerName: UDNpods[1].name,
				SrcK8SNamespace: udnNS,
				SrcK8SOwnerName: UDNpods[0].name,
			}

			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of UDN flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Udns).Should(o.ContainElement(fmt.Sprintf("%s/%s", udnNS, udnName)))
				o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
				o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
			}
		})

		g.It("Author:aramesha-High-83022-Validate CUDN with Localnet [Serial]", func() {
			SkipIfOCPBelow(4, 19)
			var (
				namespace                = oc.Namespace()
				opNamespace              = "openshift-nmstate"
				buildPruningBaseDir, _   = filePath.Abs("testdata/networking/nmstate")
				testDataDirUDN, _        = filePath.Abs("testdata/networking/udn")
				nmstateCRTemplate        = filePath.Join(buildPruningBaseDir, "nmstate-cr-template.yaml")
				ovnMappingPolicyTemplate = filePath.Join(buildPruningBaseDir, "ovn-mapping-policy-template.yaml")
				matchLabelKey            = "test.io"
				matchValue               = "cudn-network-" + getRandomString()
				secondaryCUDNName        = "secondary-localnet-83022"
				nodeSelectLabel          = "node-role.kubernetes.io/worker"
				udnStatefulSetTemplate   = filePath.Join(testDataDirUDN, "udn_statefulset_template.yaml")
				cudnNS                   = []string{"netobserv-cudn1-83022", "netobserv-cudn2-83022"}
			)

			g.By("Check the platform and network plugin type if it is suitable for running the test")
			networkType := checkNetworkType(oc)
			if !(isPlatformSuitableForNMState(oc)) || !strings.Contains(networkType, "ovn") {
				g.Skip("Skipping for unsupported platform or non-OVN network plugin type!")
			}
			installNMstateOperator(oc)

			workerNode, getNodeErr := compat_otp.GetFirstWorkerNode(oc)
			o.Expect(getNodeErr).NotTo(o.HaveOccurred())
			o.Expect(workerNode).NotTo(o.BeEmpty())

			compat_otp.By("Create NMState CR")
			nmstateCR := nmstateCRResource{
				name:     "nmstate",
				template: nmstateCRTemplate,
			}
			defer deleteNMStateCR(oc, nmstateCR)
			result, crErr := createNMStateCR(oc, nmstateCR, opNamespace)
			compat_otp.AssertWaitPollNoErr(crErr, "create nmstate cr failed")
			o.Expect(result).To(o.BeTrue())
			e2e.Logf("SUCCESS - NMState CR Created")

			compat_otp.By("Configure NNCP for creating OvnMapping NMstate Feature")
			ovnMappingPolicy := ovnMappingPolicyResource{
				name:       "bridge-mapping-83022",
				nodelabel:  nodeSelectLabel,
				labelvalue: "",
				localnet1:  "mylocalnet",
				bridge1:    "br-ex",
				template:   ovnMappingPolicyTemplate,
			}
			defer deleteNNCP(oc, ovnMappingPolicy.name)
			defer func() {
				ovnmapping, deferErr := compat_otp.DebugNodeWithChroot(oc, workerNode, "ovs-vsctl", "get", "Open_vSwitch", ".", "external_ids:ovn-bridge-mappings")
				o.Expect(deferErr).NotTo(o.HaveOccurred())
				if strings.Contains(ovnmapping, ovnMappingPolicy.localnet1) {
					// ovs-vsctl can only use "set" to reserve some fields
					_, err := compat_otp.DebugNodeWithChroot(oc, workerNode, "ovs-vsctl", "set", "Open_vSwitch", ".", "external_ids:ovn-bridge-mappings=\"physnet:br-ex\"")
					o.Expect(err).NotTo(o.HaveOccurred())
				}
			}()
			configErr3 := ovnMappingPolicy.configNNCP(oc)
			o.Expect(configErr3).NotTo(o.HaveOccurred())
			nncpErr3 := checkNNCPStatus(oc, ovnMappingPolicy.name, "Available")
			compat_otp.AssertWaitPollNoErr(nncpErr3, fmt.Sprintf("%s policy applied failed", ovnMappingPolicy.name))

			compat_otp.By("Create two namespaces and label them")
			for _, ns := range cudnNS {
				defer oc.DeleteSpecifiedNamespaceAsAdmin(ns)
				oc.CreateSpecifiedNamespaceAsAdmin(ns)
				err := oc.AsAdmin().WithoutNamespace().Run("label").Args("ns", ns, fmt.Sprintf("%s=%s", matchLabelKey, matchValue)).Execute()
				o.Expect(err).NotTo(o.HaveOccurred())
			}

			compat_otp.By("Create secondary localnet CUDN")
			defer removeResource(oc, true, true, "clusteruserdefinednetwork", secondaryCUDNName)
			_, err := applyLocalnetCUDNtoMatchLabelNS(oc, matchLabelKey, matchValue, secondaryCUDNName, "mylocalnet", "192.168.100.0/24", "192.168.100.1/32", false)
			o.Expect(err).NotTo(o.HaveOccurred())

			compat_otp.By("Deploy statefulset in both cudnNS")
			for _, ns := range cudnNS {
				defer removeResource(oc, true, true, "statefulset", "hello", "-n", ns)
				compat_otp.ApplyNsResourceFromTemplate(oc, ns, "-f", udnStatefulSetTemplate, "NETWORK_NAME="+secondaryCUDNName)
				compat_otp.AssertAllPodsToBeReady(oc, ns)
			}

			g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
			flow := Flowcollector{
				Namespace:      namespace,
				EBPFPrivileged: "true",
				EBPFeatures:    []string{"\"UDNMapping\""},
				LokiNamespace:  lokiStackNS,
				Template:       flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Validate connection among CUDN pods")
			cudn1Pods, err := compat_otp.GetAllPods(oc, cudnNS[0])
			o.Expect(err).NotTo(o.HaveOccurred())
			cudn2Pods, err := compat_otp.GetAllPods(oc, cudnNS[1])
			o.Expect(err).NotTo(o.HaveOccurred())
			startTime := time.Now()
			CurlPod2PodPassUDN(oc, cudnNS[0], cudn1Pods[0], cudnNS[1], cudn2Pods[0])

			g.By("Wait for 2 mins before logs gets collected and written to loki")
			time.Sleep(120 * time.Second)

			g.By("Verify CUDN Localnet flows")
			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				DstK8SNamespace: cudnNS[1],
				DstK8SOwnerName: "hello",
				SrcK8SNamespace: cudnNS[0],
				SrcK8SOwnerName: "hello",
			}

			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of CUDN Localnet flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.Udns).Should(o.ContainElement(secondaryCUDNName))
				o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
				o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
			}
		})

		g.It("Author:aramesha-NonPreRelease-Longduration-Medium-81410-NetObserv with eBPF manager [Serial][Slow]", func() {
			SkipIfOCPBelow(4, 18)
			g.By("Deploy eBPF manager operator")
			// eBPF manager operator variables
			bpfDir, _ := filePath.Abs("testdata/bpfman")
			bpfIDMS := filePath.Join(bpfDir, "image-digest-mirror-set.yaml")
			bpfCatSrcTemplate := filePath.Join(bpfDir, "catalog-source.yaml")

			bpfNS := OperatorNamespace{
				Name:              "bpfman",
				NamespaceTemplate: filePath.Join(bpfDir, "namespace.yaml"),
			}
			bpfCatSrc := Resource{"catsrc", "bpfman-konflux-fbc", bpfNS.Name}
			bpfSource := CatalogSourceObjects{"stable", bpfCatSrc.Name, bpfNS.Name}

			g.By("Deploy bpfman konflux FBC and ImageDigestMirrorSet")
			bpfNS.DeployOperatorNamespace(oc)
			catsrcErr := bpfCatSrc.applyFromTemplate(oc, "-n", bpfNS.Name, "-f", bpfCatSrcTemplate, "-p", "NAMESPACE="+bpfNS.Name)
			o.Expect(catsrcErr).NotTo(o.HaveOccurred())
			bpfCatSrc.WaitUntilCatSrcReady(oc)
			ApplyResourceFromFile(oc, bpfNS.Name, bpfIDMS)

			BPF := SubscriptionObjects{
				OperatorName:  "bpfman-operator",
				Namespace:     "bpfman",
				PackageName:   "bpfman-operator",
				Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
				OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
				CatalogSource: &bpfSource,
			}

			bpfExisting, err := CheckOperatorStatus(oc, BPF.Namespace, BPF.PackageName)
			o.Expect(err).NotTo(o.HaveOccurred())
			// Deploy eBPF manager operator if not present
			if !bpfExisting {
				ensureOperatorDeployed(oc, BPF, bpfSource, "name=bpfman-daemon")
			}

			g.By("Deploy FlowCollector with PacketDrop and Ebpfmanager enabled")
			flow := Flowcollector{
				Namespace:     namespace,
				EBPFeatures:   []string{"\"PacketDrop\", \"EbpfManager\""},
				LokiNamespace: lokiStackNS,
				Template:      flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for 4 mins before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(240 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}

			g.By("Verify Packet Drop flows")
			lokiParams := []string{"PktDropLatestState=\"TCP_INVALID_STATE\"", "Proto=\"6\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP Invalid State flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.PktDropLatestDropCause).NotTo(o.BeEmpty())
				o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			}

			lokiParams = []string{"PktDropLatestDropCause=\"SKB_DROP_REASON_NO_SOCKET\"", "Proto=\"6\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of No Socket TCP flows > 0")
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.PktDropLatestState).NotTo(o.BeEmpty())
				o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
				o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
			}
		})

		g.It("Author:memodi-NonPreRelease-High-82637-Verify IPSec feature [Disruptive]", func() {
			SkipIfOCPBelow(4, 16)
			compat_otp.By("Check if IPSec is enabled in the cluster")
			ipsecEnabled, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("networks.operator.openshift.io", "cluster", "-ojsonpath='{.spec.defaultNetwork.ovnKubernetesConfig.ipsecConfig.mode}'").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			ipsecEnabled = strings.Trim(ipsecEnabled, "'")
			if ipsecEnabled != "Full" {
				g.Skip("IPSec is not enabled in Full mode, skipping test")
			}

			g.By("Deploy FlowCollector IPSec enabled")
			flow := Flowcollector{
				Namespace:     namespace,
				EBPFeatures:   []string{"\"IPSec\""},
				LokiNamespace: lokiStackNS,
				Template:      flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for 2 mins before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(120 * time.Second)

			lokilabels := Lokilabels{
				App: "netobserv-flowcollector",
			}
			g.By("Verify IPSec flows")
			lokiParams := []string{"IPSecStatus=\"success\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of IPSecStatus==success flows > 0")
			metrics, err := getMetric(oc, "sum(netobserv_node_ipsec_flows_total{IPSecStatus=\"success\"})")
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(popMetricValue(metrics)).Should(o.BeNumerically(">", 0))
			o.Expect(err).NotTo(o.HaveOccurred())
			verifyEBPFFeatureMetrics(oc, "additionalmap") // additionalMap for RTT/IPSec map size
		})

		g.It("Author:kapjain-NonPreRelease-Longduration-High-85953-Verify FlowCollector Service deployment model [Serial][Slow]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploy FlowCollector with Service deployment model")
			flow := Flowcollector{
				Namespace:       namespace,
				DeploymentModel: "Service",
				Template:        flowFixturePath,
				LokiNamespace:   lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for pods to fully start and emit startup logs")
			time.Sleep(30 * time.Second)

			g.By("Verify FLP logs show 'Starting GRPC server with TLS'")
			FLPpods, err := compat_otp.GetAllPodsWithLabel(oc, flow.Namespace, "app=flowlogs-pipeline")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Get FLP pod logs to check GRPC server startup message")
			flpLogs, err := oc.AsAdmin().WithoutNamespace().Run("logs").Args("-n", flow.Namespace, FLPpods[0], "--tail=100").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(flpLogs).To(o.ContainSubstring("Starting GRPC server with TLS"), "FLP logs should contains 'Starting GRPC server with TLS'")

			g.By("Verify agent logs show 'Starting GRPC client with TLS'")
			agentPods, err := compat_otp.GetAllPodsWithLabel(oc, flow.Namespace+"-privileged", "app=netobserv-ebpf-agent")
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Get agent pod logs to check GRPC client startup message")
			agentLogs, err := oc.AsAdmin().WithoutNamespace().Run("logs").Args("-n", flow.Namespace+"-privileged", agentPods[0], "--tail=100").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(agentLogs).To(o.ContainSubstring("Starting GRPC client with TLS"), "Agent logs should contains 'Starting GRPC client with TLS'")

			g.By("Wait for a min before logs gets collected and written to loki in TLS mode")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			g.By("Get flowlogs from loki")
			err = verifyLokilogsTime(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Verify default FLP Deployment is created with 3 pods")
			result := verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 3, "")
			o.Expect(result).To(o.BeTrue(), "By default the replica count should be 3")

			g.By("Verify Service is created with correct port configuration")
			service, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("service", "flowlogs-pipeline", "-n", flow.Namespace, "-o", "jsonpath='{.spec.ports[0].port}'").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			service = strings.Trim(service, "'")
			o.Expect(service).To(o.Equal("2055"))

			targetPort, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("service", "flowlogs-pipeline", "-n", flow.Namespace, "-o", "jsonpath='{.spec.ports[0].targetPort}'").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			targetPort = strings.Trim(targetPort, "'")
			o.Expect(targetPort).To(o.Equal("2055"))

			// Test replica management with unmanagedReplicas: False by default
			g.By("Verify deployment does not upscale when unmanagedReplicas is false or not set")
			err = oc.AsAdmin().WithoutNamespace().Run("scale").Args("deployment", "flowlogs-pipeline", "-n", flow.Namespace, "--replicas=4").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 3, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should not scale when unmanagedReplicas is false or not set")

			g.By("Verify deployment does not downscale when unmanagedReplicas is false or not set")
			err = oc.AsAdmin().WithoutNamespace().Run("scale").Args("deployment", "flowlogs-pipeline", "-n", flow.Namespace, "--replicas=2").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 3, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should not scale when unmanagedReplicas is false or not set")

			g.By("Verify deployment scales via consumerReplicas when unmanagedReplicas is false or not set - upscale to 4")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=merge", "-p", `{"spec":{"processor":{"consumerReplicas":4}}}`).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 4, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should scale via consumerReplicas when unmanagedReplicas is false or not set")

			g.By("Verify deployment scales via consumerReplicas when unmanagedReplicas is false or not set - downscale to 2")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=merge", "-p", `{"spec":{"processor":{"consumerReplicas":2}}}`).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 2, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should scale via consumerReplicas when unmanagedReplicas is false or not set")

			// Test replica management with unmanagedReplicas: True
			g.By("Enable unmanagedReplicas and set consumerReplicas to 3")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=merge", "-p", `{"spec":{"processor":{"unmanagedReplicas":true,"consumerReplicas":3}}}`).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Verify deployment upscales when unmanagedReplicas is true")
			err = oc.AsAdmin().WithoutNamespace().Run("scale").Args("deployment", "flowlogs-pipeline", "-n", flow.Namespace, "--replicas=4").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 4, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should scale when unmanagedReplicas is true")

			g.By("Verify deployment downscales when unmanagedReplicas is true")
			err = oc.AsAdmin().WithoutNamespace().Run("scale").Args("deployment", "flowlogs-pipeline", "-n", flow.Namespace, "--replicas=1").Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 1, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should scale when unmanagedReplicas is true")

			g.By("Verify consumerReplicas change does not scale deployment when unmanagedReplicas is true")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=merge", "-p", `{"spec":{"processor":{"consumerReplicas":4}}}`).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 1, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should not scale via consumerReplicas when unmanagedReplicas is true")

			g.By("Verify HPA scales the deployment when unmanagedReplicas is true")
			hpaYAML := filePath.Join(baseDir, "flowlogs_pipeline_hpa_template.yaml")
			hpaFile := compat_otp.ProcessTemplate(oc, "--ignore-unknown-parameters=true", "-f", hpaYAML, "-p", "NAMESPACE="+namespace)
			defer func() {
				_ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("hpa", "flowlogs-pipeline-hpa", "-n", flow.Namespace).Execute()
			}()
			err = oc.WithoutNamespace().Run("create").Args("-f", hpaFile).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 4, ">")
			o.Expect(result).To(o.BeTrue(), "HPA should scale deployment above 4 replicas when unmanagedReplicas is true")

			g.By("Verify HPA does not scale deployment when unmanagedReplicas is false")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("flowcollector", "cluster", "--type=merge", "-p", `{"spec":{"processor":{"unmanagedReplicas":false,"consumerReplicas":2}}}`).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			result = verifyDeploymentReplicas(oc, "flowlogs-pipeline", flow.Namespace, 2, "")
			o.Expect(result).To(o.BeTrue(), "Deployment should be reconciled to consumerReplicas=2 when unmanagedReplicas is false")
		})

		g.It("Author:kapjain-Medium-86372-Verify Gateway API three-level owner metadata [Serial]", func() {
			SkipIfOCPBelow(4, 19)
			startTime := time.Now()
			g.By("Deploy flowcollector")
			gatewayAPITemplate := filePath.Join(baseDir, "gateway-api-template.yaml")
			flow := Flowcollector{
				Namespace:     namespace,
				Template:      flowFixturePath,
				LokiNamespace: lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Deploying Gateway API resources from template")
			gatewayNS := "netobserv-gateway-test"
			gatewayName := "test-gateway-owner"
			defer oc.DeleteSpecifiedNamespaceAsAdmin(gatewayNS)
			err := applyResourceFromTemplateByAdmin(oc, "-f", gatewayAPITemplate)
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Verifying Gateway Deployment exists")
			// The Gateway controller creates a Deployment named gateway-name + gatewayclass-name
			deploymentName := gatewayName + "-openshift-default"
			WaitForDeploymentPodsToBeReady(oc, gatewayNS, deploymentName)

			g.By("Verifying Pods are created by Gateway")
			pods, err := compat_otp.GetAllPodsWithLabel(oc, gatewayNS, fmt.Sprintf("gateway.networking.k8s.io/gateway-name=%s", gatewayName))
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(pods)).Should(o.BeNumerically(">", 0), "expected at least one Gateway pod")

			g.By("Waiting for flow data to be collected and written to Loki")
			time.Sleep(120 * time.Second)

			g.By("Querying flow data from Loki for Gateway pods")
			lokilabels := Lokilabels{
				SrcK8SNamespace: "netobserv-gateway-test",
			}
			parameters := []string{"SrcK8S_OwnerType=\"Gateway\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of Gateway Owner flows > 0")
		})
		g.It("Author:kapjain-Medium-88334-Pause Network Observability functions [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Create a FlowCollector")
			flow := Flowcollector{
				Namespace:       namespace,
				Template:        flowFixturePath,
				LokiNamespace:   lokiStackNS,
				DeploymentModel: "Service",
				SlicesEnable:    "true",
			}
			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)
			flow.WaitForFlowcollectorReady(oc)

			g.By("Get all netobserv-managed components before pause (excluding pods with dynamic IDs)")
			componentsBeforePause, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(
				"service,deployment,daemonset,serviceaccount,networkpolicy,configmap,secret",
				"-A", "-l", "netobserv-managed=true", "-o", "name",
			).Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.Logf("Components before pause (stable names): %s", componentsBeforePause)

			g.By("Pause the FlowCollector")
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args(
				"flowcollector", "cluster",
				"--type=merge",
				"-p", `{"spec":{"execution":{"mode":"OnHold"}}}`,
			).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Wait for FlowCollector status to show 'on hold' message")
			err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 150*time.Second, false, func(context.Context) (done bool, err error) {
				onHoldConditions, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(
					"flowcollector", "cluster",
					"-o", `jsonpath={.status.conditions[?(@.message=="FlowCollector is on hold")]}`,
				).Output()
				if err != nil {
					e2e.Logf("Error getting FlowCollector status: %v", err)
					return false, nil
				}
				if onHoldConditions != "" {
					e2e.Logf("FlowCollector status shows 'on hold'")
					return true, nil
				}
				e2e.Logf("Waiting for FlowCollector to show 'on hold' status...")
				return false, nil
			})
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Verify except for netobserv-plugin-static and network policies and persistent configmaps, all components are deleted")
			componentsAfterPause, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(
				"service,deployment,daemonset,serviceaccount,networkpolicy,configmap,secret",
				"-A", "-l", "netobserv-managed=true", "-o", "name",
			).Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			e2e.Logf("Components after pause: %s", componentsAfterPause)

			// Components with stable names that should remain when paused
			componentsShouldRemain := []string{
				"deployment.apps/netobserv-plugin-static",
				"service/netobserv-plugin-static",
				"networkpolicy.networking.k8s.io/netobserv",
				"configmap/lokistack-ca-bundle",
				"configmap/lokistack-gateway-ca-bundle",
				"configmap/grafana-dashboard-netobserv-health",
				"configmap/netobserv-main",
				"secret/lokistack-query-frontend-http",
			}
			verifyComponentsExist(componentsAfterPause, componentsShouldRemain)

			// Verify netobserv-plugin-static pod exists and other pods are deleted (using pattern since pod names have dynamic IDs)
			podsAfterPause, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(
				"pod", "-A", "-l", "netobserv-managed=true", "-o", "name",
			).Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(podsAfterPause).Should(o.ContainSubstring("pod/netobserv-plugin-static-"), "netobserv-plugin-static pod should exist after pause")
			o.Expect(podsAfterPause).ShouldNot(o.ContainSubstring("pod/flowlogs-pipeline-"), "flowlogs-pipeline pods should be deleted")
			o.Expect(podsAfterPause).ShouldNot(o.ContainSubstring("pod/netobserv-ebpf-agent-"), "netobserv-ebpf-agent pods should be deleted")
			// Verify regular netobserv-plugin pod is deleted (not the static one)
			podLines := strings.Split(podsAfterPause, "\n")
			for _, podLine := range podLines {
				if strings.Contains(podLine, "pod/netobserv-plugin-") && !strings.Contains(podLine, "pod/netobserv-plugin-static-") {
					e2e.Failf("Found non-static netobserv-plugin pod that should be deleted: %s", podLine)
				}
			}

			// Build list of components that should be deleted = originalComponentsList - componentsShouldRemain
			originalComponentsList := strings.Split(strings.TrimSpace(componentsBeforePause), "\n")
			var componentsShouldDelete []string
			for _, component := range originalComponentsList {
				component = strings.TrimSpace(component)
				if component == "" {
					continue
				}
				// Check if this component should remain
				shouldRemain := false
				for _, remainComponent := range componentsShouldRemain {
					if component == remainComponent {
						shouldRemain = true
						break
					}
				}
				// If it shouldn't remain, add to delete list
				if !shouldRemain {
					componentsShouldDelete = append(componentsShouldDelete, component)
				}
			}
			// Verify all components in the delete list are actually deleted
			verifyComponentsDeleted(componentsAfterPause, componentsShouldDelete)

			g.By("Resume the FlowCollector")
			resumeTime := time.Now()
			err = oc.AsAdmin().WithoutNamespace().Run("patch").Args(
				"flowcollector", "cluster",
				"--type=merge",
				"-p", `{"spec":{"execution":{"mode":"Running"}}}`,
			).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())

			g.By("Wait for FlowCollector to be ready")
			flow.WaitForFlowcollectorReady(oc)

			g.By("Verify no 'on hold' message in FlowCollector status")
			onHoldConditionsAfterResume, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(
				"flowcollector", "cluster",
				"-o", `jsonpath={.status.conditions[?(@.message=="FlowCollector is on hold")]}`,
			).Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(onHoldConditionsAfterResume).Should(o.BeEmpty())

			g.By("Wait for a min before logs gets collected and written to loki after resume")
			time.Sleep(60 * time.Second)

			g.By("Verify flows are being created in Loki after resume")
			err = verifyLokilogsTime(kubeadminToken, ls.Route, resumeTime)
			o.Expect(err).NotTo(o.HaveOccurred())
		})

		g.It("Author:aramesha-NonPreRelease-High-88455-Verify TLS Tracking feature [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			g.By("Deploy TLS test server and client pods")
			servertemplate := filePath.Join(baseDir, "test-tls-server_template.yaml")
			testServerTemplate := TestServerTemplate{
				ServerNS: "test-server-88455",
				Template: servertemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testServerTemplate.ServerNS)
			err := testServerTemplate.createServer(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testServerTemplate.ServerNS)

			clientTemplate := filePath.Join(baseDir, "test-tls-client_template.yaml")
			testClientTemplate := TestClientTemplate{
				ServerNS: testServerTemplate.ServerNS,
				ClientNS: "test-client-88455",
				Template: clientTemplate,
			}
			defer oc.DeleteSpecifiedNamespaceAsAdmin(testClientTemplate.ClientNS)
			err = testClientTemplate.createClient(oc)
			o.Expect(err).NotTo(o.HaveOccurred())
			compat_otp.AssertAllPodsToBeReady(oc, testClientTemplate.ClientNS)

			g.By("Deploy FlowCollector with TLS Tracking feature enabled")
			flow := Flowcollector{
				Namespace:     namespace,
				EBPFeatures:   []string{"\"TLSTracking\""},
				LokiNamespace: lokiStackNS,
				Template:      flowFixturePath,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Wait for a min before logs gets collected and written to loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			lokilabels := Lokilabels{
				App:             "netobserv-flowcollector",
				SrcK8SNamespace: testClientTemplate.ServerNS,
				DstK8SNamespace: testClientTemplate.ClientNS,
				DstK8SOwnerName: "tls-client",
				SrcK8SOwnerName: "tls-server-service",
			}

			g.By("Verify HTTP flows")
			lokiParams := []string{"Proto=\"6\"", "SrcPort=\"80\""}
			flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTP flows > 0")
			// Verify TLS fields are not populated
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.TLSVersion).Should(o.BeEmpty(), "expected TLS version to be empty for HTTP")
			}

			g.By("Verify HTTPS flows with TLSVersion 1.2")
			lokiParams = []string{"Proto=\"6\"", "SrcPort=\"443\"", "TLSVersion=\"TLS 1.2\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTPS flows with TLSv1.2 > 0")
			// Verify TLS 1.2 fields
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.TLSTypes).Should(o.ContainElement("ServerHello"), "expected TLS Types to contain ServerHello")
				o.Expect(r.Flowlog.TLSCipherSuite).NotTo(o.BeEmpty())
				// Will be fixed in follow-up
				// o.Expect(r.Flowlog.TLSCurve).NotTo(o.BeEmpty())
			}

			g.By("Verify HTTPS flows with TLSVersion 1.3")
			lokiParams = []string{"Proto=\"6\"", "SrcPort=\"443\"", "TLSVersion=\"TLS 1.3\""}
			flowRecords, err = lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, lokiParams...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTPS flows with TLSv1.3 > 0")
			// Verify TLS 1.3 fields
			for _, r := range flowRecords {
				o.Expect(r.Flowlog.TLSTypes).Should(o.ContainElement("ServerHello"), "expected TLS Types to contain ServerHello")
				o.Expect(r.Flowlog.TLSCurve).Should(o.ContainSubstring("X25519"))
				o.Expect(r.Flowlog.TLSCipherSuite).NotTo(o.BeEmpty())
			}
		})

		g.It("Author:kapjain-Medium-88683-Secure communications between Agent and FLP [Serial]", func() {
			SkipIfOCPBelow(4, 14)
			var (
				certManagerPackageName = "openshift-cert-manager-operator"
				certManagerNS          = "cert-manager-operator"
				certManagerSource      CatalogSourceObjects
				certManagerCatalog     = "redhat-operators"
				certTemplatePath       = filePath.Join(baseDir, "cert_manager_certificates_template.yaml")
			)

			certManager := SubscriptionObjects{
				OperatorName:  "cert-manager-operator-controller-manager",
				Namespace:     certManagerNS,
				PackageName:   certManagerPackageName,
				Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
				OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
				CatalogSource: &certManagerSource,
			}

			g.By("Deploy cert-manager Operator")
			// check if cert-manager Operator exists
			certManagerExisting, err := CheckOperatorStatus(oc, certManager.Namespace, certManager.PackageName)
			o.Expect(err).NotTo(o.HaveOccurred())

			certManagerChannel, err := getOperatorChannel(oc, certManagerCatalog, certManagerPackageName)
			if err != nil || certManagerChannel == "" {
				g.Skip("cert-manager channel not found, skipping test")
			}
			certManagerSource = CatalogSourceObjects{certManagerChannel, certManagerCatalog, "openshift-marketplace"}

			if !certManagerExisting {
				// Create namespace for cert-manager operator
				certManagerNSObj := OperatorNamespace{
					Name:              certManagerNS,
					NamespaceTemplate: filePath.Join(subscriptionDir, "namespace.yaml"),
				}
				certManagerNSObj.DeployOperatorNamespace(oc)

				ensureOperatorDeployed(oc, certManager, certManagerSource, "name=cert-manager-operator")
			}

			defer func() {
				if !certManagerExisting {
					certManager.uninstallOperator(oc)
					oc.DeleteSpecifiedNamespaceAsAdmin(certManagerNS)
				}
			}()

			g.By("Wait for cert-manager CRDs to be available")
			err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
				issuerCRD, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("crd", "issuers.cert-manager.io").Output()
				certCRD, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("crd", "certificates.cert-manager.io").Output()
				if strings.Contains(issuerCRD, "issuers.cert-manager.io") && strings.Contains(certCRD, "certificates.cert-manager.io") {
					e2e.Logf("cert-manager CRDs are available")
					return true, nil
				}
				e2e.Logf("Waiting for cert-manager CRDs to be available...")
				return false, nil
			})
			compat_otp.AssertWaitPollNoErr(err, "cert-manager CRDs did not become available")

			g.By("Create certificates using cert-manager")
			certFile := compat_otp.ProcessTemplate(oc, "--ignore-unknown-parameters=true", "-f", certTemplatePath, "-p", "Namespace="+namespace)
			err = oc.AsAdmin().WithoutNamespace().Run("apply").Args("-f", certFile, "-n", namespace).Execute()
			o.Expect(err).NotTo(o.HaveOccurred())
			defer func() {
				_ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", certFile, "-n", namespace).Execute()
			}()

			g.By("Wait for certificate secrets to be created")
			err = wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
				_, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("secret", "prov-netobserv-ca-secret", "-n", namespace).Output()
				if err != nil {
					return false, nil
				}
				_, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("secret", "prov-flowlogs-pipeline-cert", "-n", namespace).Output()
				if err != nil {
					return false, nil
				}
				_, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("secret", "prov-ebpf-agent-cert", "-n", namespace).Output()
				if err != nil {
					return false, nil
				}
				return true, nil
			})
			compat_otp.AssertWaitPollNoErr(err, "certificate secrets did not become available")

			g.By("Deploy FlowCollector in Service mode with Provided TLS certificates")
			flow := Flowcollector{
				Namespace:                   namespace,
				Template:                    flowFixturePath,
				DeploymentModel:             "Service",
				ServiceTLSType:              "Provided",
				ServiceCASecretName:         "prov-netobserv-ca-secret",
				ServiceServerCertSecretName: "prov-flowlogs-pipeline-cert",
				ServiceClientCertSecretName: "prov-ebpf-agent-cert",
				LokiNamespace:               lokiStackNS,
			}

			defer func() { _ = flow.DeleteFlowcollector(oc) }()
			flow.CreateFlowcollector(oc)

			g.By("Verify eBPF agent is using mTLS")
			ebpfPods, err := compat_otp.GetAllPods(oc, namespace+"-privileged")
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(ebpfPods)).Should(o.BeNumerically(">", 0), "No eBPF agent pods found")

			ebpfLogs, err := oc.AsAdmin().WithoutNamespace().Run("logs").Args("-n", namespace+"-privileged", ebpfPods[0]).Output()
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(ebpfLogs).To(o.ContainSubstring("Starting GRPC client with mTLS"), "eBPF agent logs should show mTLS is enabled")

			g.By("Wait for flow logs to be collected and written to Loki")
			startTime := time.Now()
			time.Sleep(60 * time.Second)

			g.By("Verify flow logs are being stored in Loki using verifyLokilogsTime")
			err = verifyLokilogsTime(kubeadminToken, ls.Route, startTime)
			o.Expect(err).NotTo(o.HaveOccurred())
		})
		//Add future NetObserv + Loki test-cases here

		g.Context("with Kafka", func() {
			var (
				kafkaDir, kafkaTopicPath, kafkaNodePoolPath string
				AMQexisting                                 = false
				amq                                         SubscriptionObjects
				kafkaMetrics                                KafkaMetrics
				kafka                                       Kafka
				kafkaTopic                                  KafkaTopic
				kafkaNodePool                               KafkaNodePool
				kafkaUser                                   KafkaUser
				kafkaNs                                     = "netobserv-kafka"
				kafkaClusterName                            = "kafka-cluster"
				kafkaAddress                                = fmt.Sprintf("%s-kafka-bootstrap.%s:9093", kafkaClusterName, kafkaNs)
				additionalNamespaces                        = fmt.Sprintf("\"%s\"", kafkaNs)
			)

			g.BeforeEach(func() {
				kafkaDir, _ = filePath.Abs("testdata/kafka")
				// Kafka NodePool path
				kafkaNodePoolPath = filePath.Join(kafkaDir, "kafka-node-pool.yaml")
				// Kafka Topic path
				kafkaTopicPath = filePath.Join(kafkaDir, "kafka-topic.yaml")
				// Kafka TLS Template path
				kafkaTLSPath := filePath.Join(kafkaDir, "kafka-tls.yaml")
				// Kafka metrics config Template path
				kafkaMetricsPath := filePath.Join(kafkaDir, "kafka-metrics-config.yaml")
				// Kafka User path
				kafkaUserPath := filePath.Join(kafkaDir, "kafka-user.yaml")

				g.By("Subscribe to AMQ operator")
				kafkaSource := CatalogSourceObjects{"stable", "redhat-operators", "openshift-marketplace"}
				amq = SubscriptionObjects{
					OperatorName:  "amq-streams-cluster-operator",
					Namespace:     "openshift-operators",
					PackageName:   "amq-streams",
					Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
					CatalogSource: &kafkaSource,
				}

				kafkaChannel, err := getOperatorChannel(oc, kafkaSource.SourceName, amq.PackageName)
				if err != nil || kafkaChannel == "" {
					g.Skip("Kafka channel not found, skip this case")
				}

				// check if amq Streams Operator is already present
				AMQexisting, err = CheckOperatorStatus(oc, amq.Namespace, amq.PackageName)
				o.Expect(err).NotTo(o.HaveOccurred())
				if !AMQexisting {
					ensureOperatorDeployed(oc, amq, kafkaSource, "name="+amq.OperatorName)
					// before creating kafka, check the existence of crd kafkas.kafka.strimzi.io
					checkResource(oc, true, true, "kafka.strimzi.io", []string{"crd", "kafkas.kafka.strimzi.io", "-ojsonpath={.spec.group}"})
				}

				kafkaMetrics = KafkaMetrics{
					Namespace: kafkaNs,
					Template:  kafkaMetricsPath,
				}

				kafka = Kafka{
					Name:      kafkaClusterName,
					Namespace: kafkaNs,
					Template:  kafkaTLSPath,
				}

				kafkaNodePool = KafkaNodePool{
					NodePoolName: "kafka-pool",
					Namespace:    kafkaNs,
					Name:         kafka.Name,
					Template:     kafkaNodePoolPath,
				}

				kafkaTopic = KafkaTopic{
					TopicName: "network-flows",
					Name:      kafka.Name,
					Namespace: kafkaNs,
					Template:  kafkaTopicPath,
				}

				kafkaUser = KafkaUser{
					UserName:  "flp-kafka",
					Name:      kafka.Name,
					Namespace: kafkaNs,
					Template:  kafkaUserPath,
				}

				g.By("Deploy Kafka with TLS")
				oc.CreateSpecifiedNamespaceAsAdmin(kafkaNs)
				kafkaMetrics.deployKafkaMetrics(oc)
				kafka.deployKafka(oc)
				kafkaNodePool.deployKafkaNodePool(oc)
				kafkaTopic.deployKafkaTopic(oc)
				kafkaUser.deployKafkaUser(oc)

				g.By("Check if Kafka and Kafka topic are ready")
				// wait for KafkaNodePool, Kafka and KafkaTopic to be ready
				WaitForPodsReadyWithLabel(oc, kafka.Namespace, "strimzi.io/pool-name=kafka-pool")
				waitForKafkaReady(oc, kafka.Name, kafka.Namespace)
				waitForKafkaTopicReady(oc, kafkaTopic.TopicName, kafkaTopic.Namespace)
			})

			g.AfterEach(func() {
				kafkaUser.deleteKafkaUser(oc)
				kafkaTopic.deleteKafkaTopic(oc)
				kafkaNodePool.deleteKafkaNodePool(oc)
				kafka.deleteKafka(oc)
				if !AMQexisting {
					amq.uninstallOperator(oc)
				}
				oc.DeleteSpecifiedNamespaceAsAdmin(kafkaNs)
			})

			g.It("Author:aramesha-NonPreRelease-Longduration-Critical-56362-High-53597-High-56326-High-64880-High-75340-Verify network flows are captured with Kafka with TLS [Serial][Slow]", func() {
				SkipIfOCPBelow(4, 14)

				g.By("Deploy FlowCollector with Kafka TLS")
				flow := Flowcollector{
					Namespace:                         namespace,
					DeploymentModel:                   "Kafka",
					Template:                          flowFixturePath,
					LokiNamespace:                     lokiStackNS,
					KafkaAddress:                      kafkaAddress,
					KafkaTLSEnable:                    "true",
					KafkaNamespace:                    kafkaNs,
					NetworkPolicyAdditionalNamespaces: []string{additionalNamespaces},
				}

				defer func() { _ = flow.DeleteFlowcollector(oc) }()
				flow.CreateFlowcollector(oc)

				g.By("Ensure secrets are synced")
				// ensure certs are synced to privileged NS
				secrets, err := getSecrets(oc, namespace+"-privileged")
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(secrets).To(o.And(o.ContainSubstring(kafkaUser.UserName), o.ContainSubstring(kafka.Name+"-cluster-ca-cert")))

				g.By("Verify prometheus is able to scrape metrics for FLP-Kafka")
				flpPrpmSM := "flowlogs-pipeline-transformer-monitor"
				tlsScheme, err := getMetricsScheme(oc, flpPrpmSM, flow.Namespace)
				o.Expect(err).NotTo(o.HaveOccurred())
				tlsScheme = strings.Trim(tlsScheme, "'")
				o.Expect(tlsScheme).To(o.Equal("https"))

				serverName, err := getMetricsServerName(oc, flpPrpmSM, flow.Namespace)
				serverName = strings.Trim(serverName, "'")
				o.Expect(err).NotTo(o.HaveOccurred())
				flpPromSA := "flowlogs-pipeline-transformer-prom"
				expectedServerName := fmt.Sprintf("%s.%s.svc", flpPromSA, namespace)
				o.Expect(serverName).To(o.Equal(expectedServerName))

				// verify FLP metrics are being populated with Kafka
				// Sleep before making any metrics request
				g.By("Verify prometheus is able to scrape FLP metrics")
				time.Sleep(30 * time.Second)
				verifyFLPMetrics(oc)

				// verify logs
				g.By("Wait for a min before logs gets collected and written to loki")
				startTime := time.Now()
				time.Sleep(60 * time.Second)

				g.By("Get flowlogs from loki")
				err = verifyLokilogsTime(kubeadminToken, ls.Route, startTime)
				o.Expect(err).NotTo(o.HaveOccurred())
			})

			g.It("Author:aramesha-NonPreRelease-Longduration-High-57397-High-65116-Verify network-flows export with Kafka and netobserv installation without Loki[Serial]", func() {
				SkipIfOCPBelow(4, 10)
				g.By("Deploy kafka Topic for export")
				// deploy kafka topic for export
				kafkaTopic2 := KafkaTopic{
					TopicName: "network-flows-export",
					Name:      kafka.Name,
					Namespace: kafkaNs,
					Template:  kafkaTopicPath,
				}

				defer kafkaTopic2.deleteKafkaTopic(oc)
				kafkaTopic2.deployKafkaTopic(oc)
				waitForKafkaTopicReady(oc, kafkaTopic2.TopicName, kafkaTopic2.Namespace)

				kafkaExporterConfig := map[string]interface{}{
					"kafka": map[string]interface{}{
						"address": kafkaAddress,
						"tls": map[string]interface{}{
							"caCert": map[string]interface{}{
								"certFile":  "ca.crt",
								"name":      "kafka-cluster-cluster-ca-cert",
								"namespace": kafkaNs,
								"type":      "secret"},
							"enable":             true,
							"insecureSkipVerify": false,
							"userCert": map[string]interface{}{
								"certFile":  "user.crt",
								"certKey":   "user.key",
								"name":      kafkaUser.UserName,
								"namespace": kafkaNs,
								"type":      "secret"},
						},
						"topic": kafkaTopic2.TopicName},
					"type": "Kafka",
				}

				config, err := json.Marshal(kafkaExporterConfig)
				o.Expect(err).ToNot(o.HaveOccurred())
				kafkaConfig := string(config)

				g.By("Deploy FlowCollector with Kafka TLS")
				flow := Flowcollector{
					Namespace:                         namespace,
					DeploymentModel:                   "Kafka",
					Template:                          flowFixturePath,
					LokiNamespace:                     lokiStackNS,
					KafkaAddress:                      kafkaAddress,
					KafkaTLSEnable:                    "true",
					KafkaNamespace:                    kafkaNs,
					Exporters:                         []string{kafkaConfig},
					NetworkPolicyAdditionalNamespaces: []string{additionalNamespaces},
				}

				defer func() { _ = flow.DeleteFlowcollector(oc) }()
				flow.CreateFlowcollector(oc)

				// Scenario1: Verify flows are exported with Kafka DeploymentModel and with Loki enabled
				g.By("Verify flowcollector is deployed with KAFKA exporter")
				exporterType, err := oc.AsAdmin().Run("get").Args("flowcollector", "cluster", "-o", "jsonpath='{.spec.exporters[0].type}'").Output()
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(exporterType).To(o.Equal(`'Kafka'`))

				g.By("Ensure flows are observed, all pods are running and secrets are synced and plugin pod is deployed")
				// ensure certs are synced to privileged NS
				secrets, err := getSecrets(oc, namespace+"-privileged")
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(secrets).To(o.And(o.ContainSubstring(kafkaUser.UserName), o.ContainSubstring(kafka.Name+"-cluster-ca-cert")))

				// verify logs
				g.By("Wait for a min before logs gets collected and written to loki")
				startTime := time.Now()
				time.Sleep(60 * time.Second)

				g.By("Get flowlogs from loki")
				err = verifyLokilogsTime(kubeadminToken, ls.Route, startTime)
				o.Expect(err).NotTo(o.HaveOccurred())

				g.By("Deploy Kafka consumer pod")
				// using amq-streams/kafka-34-rhel8:2.5.2 version. Update if imagePull issues are observed
				consumerTemplate := filePath.Join(kafkaDir, "topic-consumer-tls.yaml")
				consumer := Resource{"job", kafkaTopic2.TopicName + "-consumer", kafkaNs}
				defer func() { _ = consumer.clear(oc) }()
				err = consumer.applyFromTemplate(oc, "-n", consumer.Namespace, "-f", consumerTemplate, "-p", "NAME="+consumer.Name, "NAMESPACE="+consumer.Namespace, "KAFKA_TOPIC="+kafkaTopic2.TopicName, "CLUSTER_NAME="+kafka.Name, "KAFKA_USER="+kafkaUser.UserName)
				o.Expect(err).NotTo(o.HaveOccurred())

				WaitForPodsReadyWithLabel(oc, consumer.Namespace, "job-name="+consumer.Name)

				consumerPodName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", consumer.Namespace, "-l", "job-name="+consumer.Name, "-o=jsonpath={.items[0].metadata.name}").Output()
				o.Expect(err).NotTo(o.HaveOccurred())

				g.By("Verify Kafka consumer pod logs")
				podLogs, err := compat_otp.WaitAndGetSpecificPodLogs(oc, consumer.Namespace, "", consumerPodName, `'{"AgentIP":'`)
				compat_otp.AssertWaitPollNoErr(err, "Did not get log for the pod with job-name=network-flows-export-consumer label")
				verifyFlowRecordFromLogs(podLogs)

				g.By("Verify NetObserv can be installed without Loki")
				_ = flow.DeleteFlowcollector(oc)
				// Ensure FLP and eBPF pods are deleted
				checkPodDeleted(oc, namespace, "app=flowlogs-pipeline", "flowlogs-pipeline")
				checkPodDeleted(oc, namespace+"-privileged", "app=netobserv-ebpf-agent", "netobserv-ebpf-agent")

				flow.DeploymentModel = "Direct"
				flow.LokiEnable = "false"
				flow.CreateFlowcollector(oc)

				g.By("Verify Kafka consumer pod logs")
				podLogs, err = compat_otp.WaitAndGetSpecificPodLogs(oc, consumer.Namespace, "", consumerPodName, `'{"AgentIP":'`)
				compat_otp.AssertWaitPollNoErr(err, "Did not get log for the pod with job-name=network-flows-export-consumer label")
				verifyFlowRecordFromLogs(podLogs)

				g.By("Verify console plugin pod is not deployed when its disabled in flowcollector")
				_ = flow.DeleteFlowcollector(oc)
				// Ensure FLP and eBPF pods are deleted
				checkPodDeleted(oc, namespace, "app=flowlogs-pipeline", "flowlogs-pipeline")
				checkPodDeleted(oc, namespace+"-privileged", "app=netobserv-ebpf-agent", "netobserv-ebpf-agent")

				flow.PluginEnable = "false"
				flow.CreateFlowcollector(oc)

				// Scenario3: Verify all pods except plugin pod are present with only Plugin disabled in flowcollector
				g.By("Ensure all pods except consolePlugin pod are deployed")
				consolePod, err := compat_otp.GetAllPodsWithLabel(oc, namespace, "app=netobserv-plugin")
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(consolePod)).To(o.Equal(0))
			})

			//Add future NetObserv + Loki + Kafka test-cases here
		})

		g.Context("with VMs", func() {
			var (
				// virt operator vars
				VOexisting                 = false
				virtOperatorNS             = "openshift-cnv"
				virtualizationDir, _       = filePath.Abs("testdata/virtualization")
				kubevirtHyperconvergedPath = filePath.Join(virtualizationDir, "kubevirt-hyperconverged.yaml")
				virtCatsrc                 = Resource{"catsrc", "redhat-operators", "openshift-marketplace"}
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

			g.BeforeEach(func() {
				clusterArch, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("nodes", "-o=jsonpath={.items[0].status.nodeInfo.architecture}").Output()
				o.Expect(err).NotTo(o.HaveOccurred())
				if strings.Contains(clusterArch, "ppc64le") {
					g.Skip("Virtualization operator is not supported on ppc64le architecture. Skip this test!")
				}

				isMetal, err := isClusterBareMetal(oc)
				o.Expect(err).ToNot(o.HaveOccurred())
				if !isMetal && !hasMetalWorkerNodes(oc) {
					g.Skip("Cluster does not have baremetal workers. Skip this test!")
				}

				g.By("Deploy Openshift Virtualization operator")
				VOexisting, err = CheckOperatorStatus(oc, VO.Namespace, VO.PackageName)
				o.Expect(err).NotTo(o.HaveOccurred())
				if !VOexisting {
					ensureOperatorDeployed(oc, VO, virtSource, "name=virt-operator")
				}

				g.By("Deploy OpenShift Virtualization Deployment CR")
				_, err = oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", kubevirtHyperconvergedPath).Output()
				o.Expect(err).ToNot(o.HaveOccurred())
				waitUntilHyperConvergedReady(oc, "kubevirt-hyperconverged", virtOperatorNS)
				WaitForPodsReadyWithLabel(oc, virtOperatorNS, "app.kubernetes.io/managed-by=virt-operator")
			})

			g.AfterEach(func() {
				deleteResource(oc, "hyperconverged", "kubevirt-hyperconverged", virtOperatorNS)
				if !VOexisting {
					VO.uninstallOperator(oc)
				}
			})

			g.It("Author:aramesha-NonPreRelease-Longduration-High-76537-Verify flow enrichment for VM's secondary interfaces [Disruptive][Slow]", func() {
				SkipIfOCPBelow(4, 13)
				var (
					testNS = "test-76537"
					// NAD vars
					networkName   = "l2-network"
					layer2NadPath = filePath.Join(virtualizationDir, "layer2-nad.yaml")
					// VM vars
					testVMStaticIPTemplatePath = filePath.Join(virtualizationDir, "test-vm-static-IP_template.yaml")
				)

				g.By("Deploy Network Attachment Definition in test-76537 namespace")
				defer deleteNamespace(oc, testNS)
				defer deleteResource(oc, "net-attach-def", networkName, testNS)
				_, err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", layer2NadPath).Output()
				o.Expect(err).ToNot(o.HaveOccurred())
				// Wait a min for NAD to come up
				time.Sleep(60 * time.Second)
				checkNAD(oc, networkName, testNS)

				g.By("Deploy test VM1")
				testVM1 := TestVMStaticIPTemplate{
					Name:        "test-vm1",
					Namespace:   testNS,
					NetworkName: networkName,
					Mac:         "02:00:00:00:00:01",
					StaticIP:    "10.10.10.15/24",
					Template:    testVMStaticIPTemplatePath,
				}
				defer deleteResource(oc, "vm", testVM1.Name, testNS)
				err = testVM1.createVMStaticIP(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM1.Name, testVM1.Namespace)

				g.By("Wait for VM1 to get IP assigned")
				vm1Ip, err := waitForVMIPAssignment(oc, testVM1.Name, testVM1.Namespace, 1)
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
				defer deleteResource(oc, "vm", testVM2.Name, testNS)
				err = testVM2.createVMStaticIP(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM2.Name, testVM2.Namespace)

				g.By("Wait for VM2 to get IP assigned")
				vm2Ip, err := waitForVMIPAssignment(oc, testVM2.Name, testVM2.Namespace, 1)
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(vm2Ip).To(o.Equal("10.10.10.14"))

				g.By("Deploy FlowCollector")
				flow := Flowcollector{
					Namespace:      namespace,
					Template:       flowFixturePath,
					LokiNamespace:  lokiStackNS,
					EBPFPrivileged: "true",
				}

				defer func() { _ = flow.DeleteFlowcollector(oc) }()
				flow.CreateFlowcollector(oc)

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
				flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime, parameters...)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

				g.By("Verify flow logs are enriched")
				// Get VM1 pod name and node
				vm1PodName, err := compat_otp.GetAllPodsWithLabel(oc, testNS, "vm.kubevirt.io/name="+testVM1.Name)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(vm1PodName).NotTo(o.BeEmpty())
				vm1node, err := compat_otp.GetPodNodeName(oc, testNS, vm1PodName[0])
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(vm1node).NotTo(o.BeEmpty())

				// Get vm2 pod name and node
				vm2PodName, err := compat_otp.GetAllPodsWithLabel(oc, testNS, "vm.kubevirt.io/name="+testVM2.Name)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(vm2PodName).NotTo(o.BeEmpty())
				vm2node, err := compat_otp.GetPodNodeName(oc, testNS, vm2PodName[0])
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
				SkipIfOCPBelow(4, 18)
				var (
					// UDN vars
					udnNS   = "netobserv-udn-85887"
					udnName = "udn-network-85887"
					// VM vars
					testVMUDNTemplatePath = filePath.Join(virtualizationDir, "test-vm-UDN_template.yaml")
				)

				g.By("Deploy UDN in UDN ns")
				var cidr, ipv4cidr, ipv6cidr string
				defer deleteNamespace(oc, udnNS)
				oc.CreateSpecificNamespaceUDN(udnNS)

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
				createGeneralUDNCRD(oc, udnNS, udnName, ipv4cidr, ipv6cidr, cidr, "layer2")

				g.By("Deploy test VM3")
				testVM3 := TestVMUDNTemplate{
					Name:        "test-vm3",
					Namespace:   udnNS,
					NetworkName: udnName,
					Template:    testVMUDNTemplatePath,
				}
				defer deleteResource(oc, "vm", testVM3.Name, testVM3.Namespace)
				err := testVM3.createVMUDN(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM3.Name, testVM3.Namespace)

				g.By("Wait for VM3 to get IP assigned")
				vm3Ip, err := waitForVMIPAssignment(oc, testVM3.Name, testVM3.Namespace, 0)
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
				defer deleteResource(oc, "vm", testVM4.Name, testVM4.Namespace)
				err = testVM4.createVMUDN(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM4.Name, testVM4.Namespace)

				g.By("Wait for VM4 to get IP assigned")
				vm4Ip, err := waitForVMIPAssignment(oc, testVM4.Name, testVM4.Namespace, 0)
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(vm4Ip).NotTo(o.BeEmpty())

				g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
				flow := Flowcollector{
					Namespace:      namespace,
					EBPFPrivileged: "true",
					EBPFeatures:    []string{"\"UDNMapping\""},
					LokiNamespace:  lokiStackNS,
					Template:       flowFixturePath,
				}

				defer func() { _ = flow.DeleteFlowcollector(oc) }()
				flow.CreateFlowcollector(oc)

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
				flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

				g.By("Verify flow logs are enriched")
				// Get VM3 launcher pod name
				vm3podname, err := compat_otp.GetAllPodsWithLabel(oc, udnNS, "vm.kubevirt.io/name="+testVM3.Name)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(vm3podname).NotTo(o.BeEmpty())
				// Get VM4 launcher pod name
				vm4podname, err := compat_otp.GetAllPodsWithLabel(oc, udnNS, "vm.kubevirt.io/name="+testVM4.Name)
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
				SkipIfOCPBelow(4, 19)
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
				networkType := checkNetworkType(oc)
				if !(isPlatformSuitableForNMState(oc)) || !strings.Contains(networkType, "ovn") {
					g.Skip("Skipping for unsupported platform or non-OVN network plugin type!")
				}
				installNMstateOperator(oc)

				workerNode, getNodeErr := compat_otp.GetFirstWorkerNode(oc)
				o.Expect(getNodeErr).NotTo(o.HaveOccurred())
				o.Expect(workerNode).NotTo(o.BeEmpty())

				compat_otp.By("Create NMState CR")
				defer deleteNMStateCR(oc, nmstateCR)
				result, crErr := createNMStateCR(oc, nmstateCR, opNamespace)
				compat_otp.AssertWaitPollNoErr(crErr, "create nmstate cr failed")
				o.Expect(result).To(o.BeTrue())
				e2e.Logf("SUCCESS - NMState CR Created")

				compat_otp.By("Configure NNCP for creating OvnMapping NMstate Feature")
				defer deleteNNCP(oc, ovnMappingPolicy.name)
				defer func() {
					ovnmapping, deferErr := compat_otp.DebugNodeWithChroot(oc, workerNode, "ovs-vsctl", "get", "Open_vSwitch", ".", "external_ids:ovn-bridge-mappings")
					o.Expect(deferErr).NotTo(o.HaveOccurred())
					if strings.Contains(ovnmapping, ovnMappingPolicy.localnet1) {
						// ovs-vsctl can only use "set" to reserve some fields
						_, err := compat_otp.DebugNodeWithChroot(oc, workerNode, "ovs-vsctl", "set", "Open_vSwitch", ".", "external_ids:ovn-bridge-mappings=\"physnet:br-ex\"")
						o.Expect(err).NotTo(o.HaveOccurred())
					}
				}()
				configErr3 := ovnMappingPolicy.configNNCP(oc)
				o.Expect(configErr3).NotTo(o.HaveOccurred())
				nncpErr3 := checkNNCPStatus(oc, ovnMappingPolicy.name, "Available")
				compat_otp.AssertWaitPollNoErr(nncpErr3, fmt.Sprintf("%s policy applied failed", ovnMappingPolicy.name))

				compat_otp.By("Create two namespaces and label them")
				for _, ns := range cudnNS {
					defer oc.DeleteSpecifiedNamespaceAsAdmin(ns)
					oc.CreateSpecifiedNamespaceAsAdmin(ns)
					err := oc.AsAdmin().WithoutNamespace().Run("label").Args("ns", ns, fmt.Sprintf("%s=%s", matchLabelKey, matchValue)).Execute()
					o.Expect(err).NotTo(o.HaveOccurred())
				}

				compat_otp.By("Create secondary localnet CUDN")
				defer removeResource(oc, true, true, "clusteruserdefinednetwork", secondaryCUDNName)
				_, err := applyLocalnetCUDNtoMatchLabelNS(oc, matchLabelKey, matchValue, secondaryCUDNName, "mylocalnet", "192.168.200.0/24", "192.168.200.1/32", false)
				o.Expect(err).NotTo(o.HaveOccurred())

				g.By("Deploy test VM5")
				testVM5 := TestVMUDNTemplate{
					Name:        "test-vm5",
					Namespace:   cudnNS[0],
					NetworkName: secondaryCUDNName,
					Template:    testVMLocalnetTemplatePath,
				}
				defer deleteResource(oc, "vm", testVM5.Name, testVM5.Namespace)
				err = testVM5.createVMUDN(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM5.Name, testVM5.Namespace)

				// Even though VM comes up as Ready, the IP assignment takes some time
				g.By("Wait for VM5 to get IP assigned")
				vm5Ip, err := waitForVMIPAssignment(oc, testVM5.Name, testVM5.Namespace, 1)
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
				defer deleteResource(oc, "vm", testVM6.Name, testVM6.Namespace)
				err = testVM6.createVMUDN(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				waitUntilVMReady(oc, testVM6.Name, testVM6.Namespace)

				g.By("Wait for VM6 to get IP assigned")
				vm6Ip, err := waitForVMIPAssignment(oc, testVM6.Name, testVM6.Namespace, 1)
				o.Expect(err).ToNot(o.HaveOccurred())
				o.Expect(vm6Ip).NotTo(o.BeEmpty())

				g.By("Deploy FlowCollector with UDNMapping feature enabled with eBPF in privileged mode")
				flow := Flowcollector{
					Namespace:      namespace,
					EBPFPrivileged: "true",
					EBPFeatures:    []string{"\"UDNMapping\""},
					LokiNamespace:  lokiStackNS,
					Template:       flowFixturePath,
				}

				defer func() { _ = flow.DeleteFlowcollector(oc) }()
				flow.CreateFlowcollector(oc)

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

				flowRecords, err := lokilabels.getLokiFlowLogs(kubeadminToken, ls.Route, startTime)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of CUDN Localnet flows > 0")
				for _, r := range flowRecords {
					o.Expect(r.Flowlog.Udns).Should(o.ContainElement(secondaryCUDNName))
					o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
					o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
				}
			})
			//Add future NetObserv + VM test-cases here
		})
	})
})
