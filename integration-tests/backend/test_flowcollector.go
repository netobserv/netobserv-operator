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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
	e2eoutput "k8s.io/kubernetes/test/e2e/framework/pod/output"
)

var _ = g.Describe("[sig-netobserv] Network_Observability", func() {

	defer g.GinkgoRecover()
	var (
		namespace     string
		networkingDir = filePath.Join(baseDir, "networking")
	)

	g.BeforeEach(func() {
		oc := NewCLI()
		namespace = oc.Namespace()
	})

	g.It("Author:memodi-NonPreRelease-Longduration-Medium-60664-Medium-61482-Alerts-with-NetObserv [Serial][Slow]", func() {
		flpAlertRuleName := "flowlogs-pipeline-alert"
		ebpfAlertRuleName := "ebpf-agent-prom-alert"

		flow := Flowcollector{
			Namespace:       namespace,
			Template:        flowFixturePath,
			LokiEnable:      "false",
			InstallDemoLoki: "false",
		}
		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		// verify configured alerts for flp
		g.By("Get FLP Alert name and Alert Rules")
		rules, err := getConfiguredAlertRules(flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).To(o.ContainSubstring("NetObservLokiError"))

		// verify configured alerts for ebpf-agent
		g.By("Get EBPF Alert name and Alert Rules")
		ebpfRules, err := getConfiguredAlertRules(ebpfAlertRuleName, namespace+"-privileged")
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(ebpfRules).To(o.ContainSubstring("NetObservDroppedFlows"))

		// verify disable alerts feature
		g.By("Verify alerts can be disabled")
		gen, err := getResourceGeneration("prometheusRule", flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		disableAlertPatchTemp := `[{"op": "$op", "path": "/spec/processor/metrics/disableAlerts", "value": ["NetObservLokiError"]}]`
		disableAlertPatch := strings.Replace(disableAlertPatchTemp, "$op", "add", 1)
		err = patchFlowCollector(disableAlertPatch)
		o.Expect(err).NotTo(o.HaveOccurred())

		waitForResourceGenerationUpdate("prometheusRule", flpAlertRuleName, "generation", gen, namespace)
		rules, err = getConfiguredAlertRules(flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).ToNot(o.ContainSubstring("NetObservLokiError"))

		gen, err = getResourceGeneration("prometheusRule", flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		disableAlertPatch = strings.Replace(disableAlertPatchTemp, "$op", "remove", 1)
		err = patchFlowCollector(disableAlertPatch)
		o.Expect(err).NotTo(o.HaveOccurred())
		waitForResourceGenerationUpdate("prometheusRule", flpAlertRuleName, "generation", gen, namespace)
		rules, err = getConfiguredAlertRules(flpAlertRuleName, namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(rules).To(o.ContainSubstring("NetObservNoFlows"))
		o.Expect(rules).To(o.ContainSubstring("NetObservLokiError"))

		g.By("delete flowcollector")
		_ = flow.DeleteFlowcollector()

		// verify alert becomes pending.
		// configure flowcollector with incorrect loki URL
		// configure very low CacheMaxFlows to have ebpf alert fired.
		flow = Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			CacheMaxFlows:     "100",
			InstallDemoLoki:   "false",
			MonolithicLokiURL: "http://loki.no-ns.svc:3100",
		}
		g.By("Deploy flowcollector with incorrect loki URL and lower cacheMaxFlows value")
		flow.CreateFlowcollector()

		g.By("Wait for alerts to be pending")
		waitForAlertToBePending("NetObservLokiError")
	})

	g.It("Author:memodi-Medium-63185-Verify NetOberv must-gather plugin [Serial]", func() {
		SkipIfOCPBelow("v4.10")
		mustgatherDir := "/tmp/must-gather-63185"
		mustgatherImage := "quay.io/netobserv/must-gather"

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:       namespace,
			Template:        flowFixturePath,
			LokiEnable:      "false",
			InstallDemoLoki: "false",
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		// Note: In older OCP versions, oc adm inspect outputs benign discovery errors that don't affect data collection.
		g.By("Run must-gather command")
		defer func() { _, _ = exec.Command("bash", "-c", "rm -rf "+mustgatherDir).Output() }()
		cmd := exec.Command("oc", "adm", "must-gather", "--image", mustgatherImage, "--dest-dir="+mustgatherDir)
		outputBytes, err := cmd.CombinedOutput()
		output := string(outputBytes)
		o.Expect(err).NotTo(o.HaveOccurred(), "must-gather command failed: %s", output)

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
		assertWaitPollNoErr(err, "must-gather data not populated within timeout")

		g.By("Verify operator namespace logs are scraped")
		operatorLogsPattern := fmt.Sprintf("%s/namespaces/openshift-netobserv-operator/pods/netobserv-controller-manager-*/manager/manager/logs/current.log", mustgatherLogsDir)
		operatorlogs, err := filePath.Glob(operatorLogsPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(operatorlogs)).Should(o.BeNumerically(">", 0), "No logs were saved to: "+operatorLogsPattern)
		_, err = os.Stat(operatorlogs[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify flowlogs-pipeline pod logs are scraped")
		pods, err := getAllPods(namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		flpLogsPattern := fmt.Sprintf("%s/namespaces/%s/pods/%s/flowlogs-pipeline/flowlogs-pipeline/logs/current.log", mustgatherLogsDir, namespace, pods[0])
		podlogs, err := filePath.Glob(flpLogsPattern)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(podlogs)).Should(o.BeNumerically(">", 0), "No logs were saved to: "+flpLogsPattern)
		_, err = os.Stat(podlogs[0])
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify eBPF agent pod logs are scraped")
		ebpfPods, err := getAllPods(namespace + "-privileged")
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
		// verify tolerations
		g.By("Get worker node of the cluster")
		workerNode, err := getFirstWorkerNode()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Taint worker node")
		defer func() {
			err := removeTaintFromNode(workerNode, "netobserv-agent")
			o.Expect(err).NotTo(o.HaveOccurred())
		}()
		err = taintNode(workerNode, "netobserv-agent", "true", "NoSchedule")
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:       namespace,
			Template:        flowFixturePath,
			LokiEnable:      "false",
			InstallDemoLoki: "false",
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Add wrong toleration for eBPF spec for the taint netobserv-agent=false:NoSchedule")
		patchValue := `{"scheduling":{"tolerations":[{"effect": "NoSchedule", "key": "netobserv-agent", "value": "false", "operator": "Equal"}]}}`
		err = patchFlowCollector(`[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": ` + patchValue + `}]`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady()

		g.By(fmt.Sprintf("Verify eBPF pod is not scheduled on the %s", workerNode))
		eBPFPods, err := k8sClient.CoreV1().Pods(flow.Namespace+"-privileged").List(context.Background(), metav1.ListOptions{FieldSelector: "spec.nodeName=" + workerNode})
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(eBPFPods.Items).Should(o.BeEmpty())

		g.By("Add correct toleration for eBPF spec for the taint netobserv-agent=true:NoSchedule")
		err = flow.DeleteFlowcollector()
		o.Expect(err).NotTo(o.HaveOccurred())
		flow.CreateFlowcollector()
		patchValue = `{"scheduling":{"tolerations":[{"effect": "NoSchedule", "key": "netobserv-agent", "value": "true", "operator": "Equal"}]}}`
		err = patchFlowCollector(`[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": ` + patchValue + `}]`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady()

		g.By(fmt.Sprintf("Verify eBPF pod is scheduled on the node %s after applying toleration for taint netobserv-agent=true:NoSchedule", workerNode))
		eBPFPods, err = k8sClient.CoreV1().Pods(flow.Namespace+"-privileged").List(context.Background(), metav1.ListOptions{FieldSelector: "spec.nodeName=" + workerNode})
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(eBPFPods.Items).NotTo(o.BeEmpty())

		// verify nodeSelector
		g.By("Add netobserv label to above worker node")
		defer func() { _ = deleteLabelFromNode(workerNode, "netobserv-agent") }()
		err = addLabelToNode(workerNode, "netobserv-agent", "true")
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Patch flowcollector with nodeSelector for eBPF pods")
		err = flow.DeleteFlowcollector()
		o.Expect(err).NotTo(o.HaveOccurred())
		flow.CreateFlowcollector()
		patchValue = `{"scheduling":{"nodeSelector":{"netobserv-agent": "true"}}}`
		err = patchFlowCollector(`[{"op": "replace", "path": "/spec/agent/ebpf/advanced", "value": ` + patchValue + `}]`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Ensure flowcollector is ready")
		flow.WaitForFlowcollectorReady()

		g.By("Verify all eBPF pods are deployed on the above worker node")
		eBPFpods, err := getAllPodsWithLabel(flow.Namespace+"-privileged", "app=netobserv-ebpf-agent")
		o.Expect(err).NotTo(o.HaveOccurred())
		for _, pod := range eBPFpods {
			nodeName, err := getPodNodeName(flow.Namespace+"-privileged", pod)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(nodeName).To(o.Equal(workerNode))
		}
	})

	g.It("Author:osmakal-High-89198-Verify processor metrics configuration with includeList and additionalIncludeList [Serial]", func() {
		g.By("Deploy initial FlowCollector without additionalIncludeList")
		flow := Flowcollector{
			Namespace:       namespace,
			Template:        flowFixturePath,
			LokiEnable:      "false",
			InstallDemoLoki: "false",
		}
		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for FlowCollector to reconcile")
		flow.WaitForFlowcollectorReady()

		g.By("Wait for baseline metrics to be available")
		time.Sleep(90 * time.Second)

		// Capture baseline metrics (defaults only)
		g.By("Query Prometheus for baseline netobserv_* metrics")
		baselineMetrics, err := getAllNetobservMetricNames()
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to query Prometheus for baseline")

		e2e.Logf("Found %d baseline netobserv metrics in Prometheus", len(baselineMetrics))

		baselineSet := make(map[string]bool)
		for _, m := range baselineMetrics {
			baselineSet[m] = true
		}

		// Apply patch to add additional metrics
		g.By("Patch FlowCollector with additionalIncludeList")
		additionalIncludeList := []string{"namespace_egress_bytes_total", "namespace_ingress_bytes_total"}
		patch := `[{"op": "add", "path": "/spec/processor/metrics/additionalIncludeList", "value": ["namespace_egress_bytes_total", "namespace_ingress_bytes_total"]}]`
		err = patchFlowCollector(patch)
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to patch FlowCollector")

		g.By("Wait for FlowCollector to reconcile after patch")
		flow.WaitForFlowcollectorReady()

		g.By("Wait for updated metrics to be available")
		time.Sleep(90 * time.Second)

		// Get all metrics after patch
		g.By("Query Prometheus for all netobserv_* metrics after patch")
		allMetrics, err := getAllNetobservMetricNames()
		o.Expect(err).NotTo(o.HaveOccurred(), "failed to query Prometheus after patch")

		e2e.Logf("Found %d netobserv metrics in Prometheus after patch", len(allMetrics))

		actualSet := make(map[string]bool)
		for _, m := range allMetrics {
			actualSet[m] = true
		}

		// Verify all baseline metrics are still present (superset check)
		for _, metric := range baselineMetrics {
			o.Expect(actualSet[metric]).To(o.BeTrue(),
				fmt.Sprintf("baseline metric %s should still exist in %v", metric, allMetrics))
		}

		// Verify the additional metrics are now present
		for _, metric := range additionalIncludeList {
			fullMetricName := "netobserv_" + metric
			o.Expect(actualSet[fullMetricName]).To(o.BeTrue(),
				fmt.Sprintf("additional metric %s should exist in %v", fullMetricName, allMetrics))
		}

		// Verify the total count increased by exactly len(additionalIncludeList)
		expectedCount := len(baselineMetrics) + len(additionalIncludeList)
		o.Expect(len(allMetrics)).To(o.Equal(expectedCount),
			fmt.Sprintf("expected %d metrics (baseline: %d + additional: %d) but found %d: %v",
				expectedCount, len(baselineMetrics), len(additionalIncludeList), len(allMetrics), allMetrics))
	})

	g.Context("FLP, eBPF and Console metrics:", func() {
		g.When("processor.metrics.TLS == Disabled and agent.ebpf.metrics.TLS == Disabled", func() {
			g.It("Author:aramesha-Critical-50504-Critical-72959-Verify flowlogs-pipeline and eBPF metrics and health [Serial]", func() {
				var (
					flpPromSM  = "flowlogs-pipeline-monitor"
					eBPFPromSM = "ebpf-agent-svc-monitor"
					curlLive   = "http://localhost:8080/live"
				)

				g.By("Deploy flowcollector")
				flow := Flowcollector{
					Namespace:              namespace,
					Template:               flowFixturePath,
					MonolithicLokiURL:      fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
					FLPMetricServerTLSType: "Disabled",
				}

				defer func() { _ = flow.DeleteFlowcollector() }()
				flow.CreateFlowcollector()

				g.By("Verify flowlogs-pipeline metrics")
				FLPpods, err := getAllPodsWithLabel(namespace, "app=flowlogs-pipeline")
				o.Expect(err).NotTo(o.HaveOccurred())

				for _, pod := range FLPpods {
					command := []string{"curl", "-s", curlLive}
					output, err := execInPod(namespace, pod, command)
					o.Expect(err).NotTo(o.HaveOccurred())
					o.Expect(output).To(o.Equal("{}"))
				}

				FLPtlsScheme, err := getMetricsScheme(flpPromSM, flow.Namespace)
				o.Expect(err).NotTo(o.HaveOccurred())
				FLPtlsScheme = strings.Trim(FLPtlsScheme, "'")
				o.Expect(FLPtlsScheme).To(o.Equal("http"))

				g.By("Wait for a min before scraping metrics")
				time.Sleep(60 * time.Second)

				g.By("Verify prometheus is able to scrape FLP metrics")
				verifyFLPMetrics()

				g.By("Verify eBPF agent metrics")
				eBPFtlsScheme, err := getMetricsScheme(eBPFPromSM, flow.Namespace+"-privileged")
				o.Expect(err).NotTo(o.HaveOccurred())
				eBPFtlsScheme = strings.Trim(eBPFtlsScheme, "'")
				o.Expect(eBPFtlsScheme).To(o.Equal("http"))

				g.By("Wait for a min before scraping metrics")
				time.Sleep(60 * time.Second)

				g.By("Verify prometheus is able to scrape eBPF metrics")
				verifyEBPFMetrics()
			})
		})

		g.When("processor.metrics.TLS == Auto and ebpf.agent.metrics.TLS == Auto", func() {
			g.It("Author:aramesha-Critical-54043-Critical-66031-Critical-72959-Verify flowlogs-pipeline, eBPF and Console metrics [Serial]", func() {
				var (
					flpPromSM  = "flowlogs-pipeline-monitor"
					flpPromSA  = "flowlogs-pipeline-prom"
					eBPFPromSM = "ebpf-agent-svc-monitor"
					eBPFPromSA = "ebpf-agent-svc-prom"
				)

				flow := Flowcollector{
					Namespace:               namespace,
					Template:                flowFixturePath,
					MonolithicLokiURL:       fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
					EBPFMetricServerTLSType: "Auto",
				}

				defer func() { _ = flow.DeleteFlowcollector() }()
				flow.CreateFlowcollector()

				g.By("Verify flowlogs-pipeline metrics")
				FLPtlsScheme, err := getMetricsScheme(flpPromSM, flow.Namespace)
				o.Expect(err).NotTo(o.HaveOccurred())
				FLPtlsScheme = strings.Trim(FLPtlsScheme, "'")
				o.Expect(FLPtlsScheme).To(o.Equal("https"))

				FLPserverName, err := getMetricsServerName(flpPromSM, flow.Namespace)
				FLPserverName = strings.Trim(FLPserverName, "'")
				o.Expect(err).NotTo(o.HaveOccurred())
				FLPexpectedServerName := fmt.Sprintf("%s.%s.svc", flpPromSA, namespace)
				o.Expect(FLPserverName).To(o.Equal(FLPexpectedServerName))

				g.By("Wait for a min before scraping metrics")
				time.Sleep(60 * time.Second)

				g.By("Verify prometheus is able to scrape FLP and Console metrics")
				verifyFLPMetrics()
				query := fmt.Sprintf("process_start_time_seconds{namespace=\"%s\", job=\"netobserv-plugin-metrics\"}", namespace)
				metrics, err := getMetric(query)
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(popMetricValue(metrics)).Should(o.BeNumerically(">", 0))

				g.By("Verify eBPF metrics")
				eBPFtlsScheme, err := getMetricsScheme(eBPFPromSM, flow.Namespace+"-privileged")
				o.Expect(err).NotTo(o.HaveOccurred())
				eBPFtlsScheme = strings.Trim(eBPFtlsScheme, "'")
				o.Expect(eBPFtlsScheme).To(o.Equal("https"))

				eBPFserverName, err := getMetricsServerName(eBPFPromSM, flow.Namespace+"-privileged")
				eBPFserverName = strings.Trim(eBPFserverName, "'")
				o.Expect(err).NotTo(o.HaveOccurred())
				eBPFexpectedServerName := fmt.Sprintf("%s.%s.svc", eBPFPromSA, namespace+"-privileged")
				o.Expect(eBPFserverName).To(o.Equal(eBPFexpectedServerName))

				g.By("Verify prometheus is able to scrape eBPF agent metrics")
				verifyEBPFMetrics()
			})
		})
	})

	g.It("Author:memodi-High-53595-High-49107-High-45304-High-54929-High-54840-High-68310-Verify flow correctness and metrics [Serial]", func() {
		g.By("Deploying test server and client pods")
		serverTemplatePath := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServer := TestServerTemplate{
			ServerNS: "test-server-54929",
			Template: serverTemplatePath,
		}
		defer deleteNamespace(testServer.ServerNS)
		err := testServer.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServer.ServerNS)

		clientTemplatePath := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClient := TestClientTemplate{
			ServerNS:   testServer.ServerNS,
			ClientNS:   "test-client-54929",
			ObjectSize: "100K",
			Template:   clientTemplatePath,
		}

		defer deleteNamespace(testClient.ClientNS)
		err = testClient.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClient.ClientNS)

		startTime := time.Now()

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

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

		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords > 0")

		// verify flow correctness
		verifyFlowCorrectness(testClient.ObjectSize, flowRecords)

		// verify inner metrics
		query := fmt.Sprintf(`sum(rate(netobserv_workload_ingress_bytes_total{SrcK8S_Namespace="%s"}[1m]))`, testClient.ClientNS)
		metrics := pollMetrics(query)

		// verify metric is between 200 and 400
		o.Expect(metrics).Should(o.BeNumerically("~", 300, 100))
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-High-60701-Verify connection tracking [Serial]", func() {
		startTime := time.Now()

		g.By("Deploying test server and client pods")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-60701",
			Template: serverTemplate,
		}

		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")

		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-60701",
			Template: clientTemplate,
		}

		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		g.By("Deploy FlowCollector with endConversations LogType")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			DeploymentModel:   "Direct",
			LogType:           "EndedConversations",
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

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
		endConnectionRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(endConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of endConnectionRecords > 0")
		verifyConversationRecordTime(endConnectionRecords)

		g.By("Deploy FlowCollector with Conversations LogType")
		_ = flow.DeleteFlowcollector()

		flow.LogType = "Conversations"
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		startTime = time.Now()
		time.Sleep(60 * time.Second)

		g.By("Verify NewConnection Records from loki")
		lokilabels.RecordType = "newConnection"

		newConnectionRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(newConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of newConnectionRecords > 0")
		verifyConversationRecordTime(newConnectionRecords)

		g.By("Verify HeartbeatConnection Records from loki")
		lokilabels.RecordType = "heartbeat"
		heartbeatConnectionRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(heartbeatConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of heartbeatConnectionRecords > 0")
		verifyConversationRecordTime(heartbeatConnectionRecords)

		g.By("Verify EndConnection Records from loki")
		lokilabels.RecordType = "endConnection"
		endConnectionRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(endConnectionRecords)).Should(o.BeNumerically(">", 0), "expected number of endConnectionRecords > 0")
		verifyConversationRecordTime(endConnectionRecords)
	})

	g.It("Author:aramesha-NonPreRelease-Critical-59746-NetObserv upgrade testing [Serial]", func() {
		// Uninstall operator even if test fails/passes, then redeploy for subsequent tests
		g.DeferCleanup(func() {
			NO.uninstallOperator()
			deleteNamespace(netobservNS)
			OperatorNS.DeployOperatorNamespace()
			isHypershift := isHypershiftHostedCluster()
			catSrcErr := setupCatalogSource(NOcatSrc, catSrcTemplate, imageDigest, catalogSource, isHypershift, &NOSource, &NO)
			o.Expect(catSrcErr).NotTo(o.HaveOccurred())
			ensureNetObservOperatorDeployed(NO, NOSource)
		})

		g.By("Uninstall operator deployed by BeforeSuite and delete operator NS")
		NO.uninstallOperator()
		deleteNamespace(netobservNS)
		err := Resource{"namespace", netobservNS, ""}.WaitUntilResourceIsGone()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy older version of netobserv operator")
		NOReleasedcatSrc := Resource{"catsrc", "redhat-operators", "openshift-marketplace"}
		NOReleasedSource := CatalogSourceObjects{"stable", NOReleasedcatSrc.Name, NOReleasedcatSrc.Namespace}

		// Use local copy instead of modifying global NO
		NOReleased := NO
		NOReleased.CatalogSource = &NOReleasedSource

		g.By(fmt.Sprintf("Subscribe operators to %s channel", NOSource.Channel))
		OperatorNS.DeployOperatorNamespace()
		NOReleased.SubscribeOperator()
		WaitForPodsReadyWithLabel(netobservNS, "app="+NO.OperatorName)
		NOStatus, err := CheckOperatorStatus(netobservNS, NOPackageName)
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("found err %v", err))
		o.Expect(NOStatus).To(o.BeTrue())

		// check if flowcollector API exists
		flowcollectorAPIExists, err := isFlowCollectorAPIExists()
		o.Expect((flowcollectorAPIExists)).To(o.BeTrue())
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Get NetObserv and components versions")
		NOCSV, err := getPodEnvValue(netobservNS, "app=netobserv-operator", "OPERATOR_CONDITION_NAME")
		o.Expect(err).NotTo(o.HaveOccurred())

		preUpgradeNOVersion := strings.Split(NOCSV, ".v")[1]
		preUpgradeEBPFVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 0)
		o.Expect(err).NotTo(o.HaveOccurred())
		preUpgradeEBPFVersion = strings.Split(preUpgradeEBPFVersion, ":")[1]
		preUpgradeFLPVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 1)
		o.Expect(err).NotTo(o.HaveOccurred())
		preUpgradeFLPVersion = strings.Split(preUpgradeFLPVersion, ":")[1]
		preUpgradePluginVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 2)
		o.Expect(err).NotTo(o.HaveOccurred())
		preUpgradePluginVersion = strings.Split(preUpgradePluginVersion, ":")[1]

		g.By("Deploy latest catalog and upgrade to latest version")
		var catsrcErr error
		if catalogSource != "" {
			e2e.Logf("Using %s catalog", catalogSource)
			catsrcErr = NOcatSrc.applyFromTemplate("-n", NOcatSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+NOcatSrc.Namespace, "IMAGE="+catalogSource)
		} else {
			e2e.Logf("Using default ystream catalog")
			catsrcErr = NOcatSrc.applyFromTemplate("-n", NOcatSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+NOcatSrc.Namespace)
		}
		o.Expect(catsrcErr).NotTo(o.HaveOccurred())
		_ = patchSubscription(netobservNS, "netobserv-operator", `[{"op": "replace", "path": "/spec/source", "value": "`+NOcatSrc.Name+`"}, {"op": "replace", "path": "/spec/sourceNamespace", "value": "`+NOcatSrc.Namespace+`"}]`)

		g.By("Wait for a min for operator upgrade")
		time.Sleep(60 * time.Second)

		WaitForPodsReadyWithLabel(netobservNS, "app=netobserv-operator")
		NOStatus, err = CheckOperatorStatus(netobservNS, NOPackageName)
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("found err %v", err))
		o.Expect((NOStatus)).To(o.BeTrue())

		g.By("Get NetObserv operator and components versions")
		NOCSV, err = getPodEnvValue(netobservNS, "app=netobserv-operator", "OPERATOR_CONDITION_NAME")
		o.Expect(err).NotTo(o.HaveOccurred())

		postUpgradeNOVersion := strings.Split(NOCSV, ".v")[1]
		postUpgradeEBPFVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 0)
		o.Expect(err).NotTo(o.HaveOccurred())
		postUpgradeEBPFVersion = strings.Split(postUpgradeEBPFVersion, ":")[1]
		postUpgradeFLPVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 1)
		o.Expect(err).NotTo(o.HaveOccurred())
		postUpgradeFLPVersion = strings.Split(postUpgradeFLPVersion, ":")[1]
		postUpgradePluginVersion, err := getPodEnvByIndex(netobservNS, "app=netobserv-operator", 2)
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
		err = verifyMonolithicLokilogsTime(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
	})

	g.It("Author:aramesha-NonPreRelease-High-62989-Verify SCTP, ICMP, ICMPv6 traffic is observed [Disruptive]", func() {
		var (
			sctpClientPodTemplatePath = filePath.Join(networkingDir, "sctpclient.yaml")
			sctpServerPodTemplatePath = filePath.Join(networkingDir, "sctpserver.yaml")
			sctpServerPodname         = "sctpserver"
			sctpClientPodname         = "sctpclient"
		)

		ipStackType := checkIPStackType()

		g.By("install load-sctp-module in all workers")
		prepareSCTPModule()

		g.By("Create netobserv-sctp NS")
		SCTPns := "netobserv-sctp-62989"
		defer deleteNamespace(SCTPns)
		_ = createNamespace(SCTPns)
		_ = setNamespacePrivileged(SCTPns)

		g.By("create sctpClientPod")
		createResourceFromFile(SCTPns, sctpClientPodTemplatePath)
		WaitForPodsReadyWithLabel(SCTPns, "name=sctpclient")

		g.By("create sctpServerPod")
		createResourceFromFile(SCTPns, sctpServerPodTemplatePath)
		WaitForPodsReadyWithLabel(SCTPns, "name=sctpserver")

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("get primary IP address of sctpServerPod")
		sctpServerPodIP, _ := getPodIP(SCTPns, sctpServerPodname, ipStackType)

		g.By("sctpserver pod start to wait for sctp traffic")
		cmd := exec.Command("oc", "exec", "-n", SCTPns, sctpServerPodname, "--", "/usr/bin/ncat", "-l", "30102", "--sctp")
		_ = cmd.Start()
		defer func() {
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}()
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

		SCTPflows, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
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

		ICMPflows, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
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
		g.By("Deploying test server and client pods")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-68125",
			Template: serverTemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-68125",
			Template: clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		g.By("Check cluster network type")
		networkType := checkNetworkType()
		o.Expect(networkType).NotTo(o.BeEmpty())
		if networkType == "ovnkubernetes" {
			g.By("Deploy egressQoS for OVN CNI")
			clientDSCPPath := filePath.Join(networkingDir, "test-client-DSCP.yaml")
			egressQoSPath := filePath.Join(networkingDir, "egressQoS.yaml")
			g.By("Deploy nginx client pod and egressQoS")
			createResourceFromFile(testClientTemplate.ClientNS, clientDSCPPath)
			createResourceFromFile(testClientTemplate.ClientNS, egressQoSPath)
		}

		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

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
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
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
			flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 59 should be > 0")

			g.By("Verify DSCP value=0 for flows from pods other than DSCP client pod in test-client namespace")
			parameters = []string{"SrcK8S_Name=\"client\", Dscp=\"0\""}

			flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 0 should be > 0")
		}

		// Scenario3: Explicitly passing QoS value in ping command
		ipStackType := checkIPStackType()
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
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with DSCP value 32 > 0")
	})

	g.It("Author:aramesha-NonPreRelease-High-69218-High-71291-Verify cluster ID and zone in multiCluster deployment [Serial]", func() {
		g.By("Get clusterID of the cluster")
		cvObj, err := getDynamicResource("clusterversion", "version", "")
		o.Expect(err).NotTo(o.HaveOccurred())
		clusterID, _ := getNestedField(cvObj.Object, ".spec.clusterID")
		e2e.Logf("Cluster ID is %s", clusterID)

		g.By("Deploy FlowCollector with multiCluster and addZone enabled")
		flow := Flowcollector{
			Namespace:              namespace,
			MultiClusterDeployment: "true",
			AddZone:                "true",
			Template:               flowFixturePath,
			MonolithicLokiURL:      fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		// verify logs
		g.By("Wait for a min before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		g.By("Verify K8SClusterName = Cluster ID")
		clusteridlabels := Lokilabels{
			App:            "netobserv-flowcollector",
			K8SClusterName: clusterID,
		}
		clusterIDFlowRecords, err := clusteridlabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(clusterIDFlowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows > 0")

		g.By("Verify SrcK8S_Zone and DstK8S_Zone are present and have expected values")
		zonelabels := Lokilabels{
			App:        "netobserv-flowcollector",
			SrcK8SType: "Node",
			DstK8SType: "Node",
		}

		zoneFlowRecords, err := zonelabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(zoneFlowRecords)).Should(o.BeNumerically(">", 0), "expected zone flow records > 0")
		for _, r := range zoneFlowRecords {
			expectedSrcK8SZone, err := getNodeLabelValue(r.Flowlog.SrcK8SHostName, `topology.kubernetes.io/zone`)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(r.Flowlog.SrcK8SZone).To(o.Equal(expectedSrcK8SZone))

			expectedDstK8SZone, err := getNodeLabelValue(r.Flowlog.DstK8SHostName, `topology.kubernetes.io/zone`)
			o.Expect(err).NotTo(o.HaveOccurred())
			o.Expect(r.Flowlog.DstK8SZone).To(o.Equal(expectedDstK8SZone))
		}
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-High-73175-Verify eBPF agent filtering [Serial]", func() {
		g.By("Deploy test server and client pods")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-73175",
			Template: serverTemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-73175",
			Template: clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		ipStackType := checkIPStackType()
		clientServiceInfo, err := getClientServerInfo(testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
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
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFFilterRules:   filter,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Ping nginx pod from client pod")
		startTime := time.Now()
		_, _ = e2eoutput.RunHostCmd(testClientTemplate.ClientNS, clientServiceInfo["client"]["name"], "ping -c 10 "+clientServiceInfo["server"]["ip"])

		g.By("Wait for a min before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}

		g.By("Verify number of flows with on UDP Protcol with SrcPort 53 = 0")
		lokilabels.AllowEmpty = true
		lokiParams := []string{"Proto=\"17\"", "SrcPort=\"53\""}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically("==", 0), "expected number of flows on UDP with SrcPort 53 = 0")
		lokilabels.AllowEmpty = false

		g.By("Verify flows from client pod to nginx pod > 0")
		lokilabels.SrcK8SNamespace = testClientTemplate.ClientNS
		lokilabels.DstK8SNamespace = testClientTemplate.ServerNS
		lokiParams = []string{"SrcAddr=" + "\"" + clientServiceInfo["client"]["ip"] + "\"", "DstAddr=" + "\"" + clientServiceInfo["server"]["ip"] + "\""}

		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client pod to nginx pod > 0")

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Proto).Should(o.BeNumerically("==", 1))
			o.Expect(r.Flowlog.IcmpType).Should(o.BeNumerically("==", 8))
			o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 3))
		}

		g.By("Verify flows from client pod to nginx-service > 0")
		lokilabels.DstK8SType = "Service"

		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows from client pod to nginx-service > 0")

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Proto).Should(o.BeNumerically("==", 6))
			o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 2))
		}

		g.By("Verify prometheus is able to scrape eBPF metrics")
		verifyEBPFFilterMetrics("FilterAccept")
		verifyEBPFFilterMetrics("FilterNoMatch")

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

		_ = flow.DeleteFlowcollector()
		flow.EBPFPrivileged = "true"
		flow.EBPFeatures = []string{"\"PacketDrop\""}
		flow.EBPFFilterRules = filter
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		startTime = time.Now()
		time.Sleep(60 * time.Second)

		lokilabels = Lokilabels{
			App: "netobserv-flowcollector",
		}
		lokiParams = []string{"Proto=\"6\""}

		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows with drops > 0")

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
		}
	})

	g.It("Author:memodi-Critical-53844-Sanity Test NetObserv [Serial]", func() {
		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for a min before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}

		g.By("Verify flows are written to loki")
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")
	})

	g.It("Author:aramesha-High-67782-Verify large volume downloads [Serial]", func() {
		g.By("Deploy FlowCollector")
		flow := Flowcollector{
			Namespace:              namespace,
			Template:               flowFixturePath,
			MonolithicLokiURL:      fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFCacheActiveTimeout: "30s",
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Deploy test server and client pods")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS:  "test-server-67782",
			Template:  serverTemplate,
			LargeBlob: "yes",
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS:   testServerTemplate.ServerNS,
			ClientNS:   "test-client-67782",
			ObjectSize: "100M",
			Template:   clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

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
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows written to loki > 0")

		g.By("Verify flow correctness")
		verifyFlowCorrectness(testClientTemplate.ObjectSize, flowRecords)
	})

	g.It("Author:aramesha-High-75656-Verify TCP flags [Disruptive]", func() {
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
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFFilterRules:   filter,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Ensure flowcollector is ready with Reject flowFilter")
		fcObj, err := getDynamicResource("flowcollector", "cluster", "")
		o.Expect(err).ToNot(o.HaveOccurred())
		rules, _, _ := unstructured.NestedSlice(fcObj.Object, "spec", "agent", "ebpf", "flowFilter", "rules")
		o.Expect(rules).NotTo(o.BeEmpty())
		rule0 := rules[0].(map[string]interface{})
		o.Expect(rule0["action"]).To(o.Equal("Reject"))

		g.By("Deploy custom metrics to detect SYN flooding")
		customMetrics := CustomMetrics{
			Namespace: namespace,
			Template:  SYNFloodMetricsPath,
		}

		curv, err := getResourceVersion("cm", "flowlogs-pipeline-config-dynamic", namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		customMetrics.createCustomMetrics()
		waitForResourceGenerationUpdate("cm", "flowlogs-pipeline-config-dynamic", "resourceVersion", curv, namespace)

		g.By("Deploy SYN flooding alert rule")
		defer func() {
			_ = deleteDynamicResource("alertingrule", "netobserv-syn-alerts", "openshift-monitoring")
		}()
		configFile, err := processTemplate("openshift-monitoring", "--ignore-unknown-parameters=true", "-f", SYNFloodAlertsPath, "-p", "Namespace=openshift-monitoring")
		o.Expect(err).NotTo(o.HaveOccurred())
		ApplyResourceFromFile("", configFile)

		g.By("Deploy test client pod to induce SYN flooding")
		template := filePath.Join(baseDir, "test-SYN-flood-client_template.yaml")
		testTemplate := TestClientTemplate{
			ClientNS: "test-client-75656",
			Template: template,
		}

		defer deleteNamespace(testTemplate.ClientNS)
		configFile, err = processTemplate("", "--ignore-unknown-parameters=true", "-f", testTemplate.Template, "-p", "CLIENT_NS="+testTemplate.ClientNS)
		o.Expect(err).NotTo(o.HaveOccurred())
		ApplyResourceFromFile("", configFile)

		g.By("Wait for a min before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}

		g.By("Verify no flows with SYN_ACK TCP flag")
		parameters := []string{"Flags=\"SYN_ACK\""}

		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		// Loop needed since even flows with flags SYN, ACK are matched
		count := 0
		for _, r := range flowRecords {
			for _, f := range r.Flowlog.Flags {
				o.Expect(f).ToNot(o.Equal("SYN_ACK"))
			}
		}
		o.Expect(count).Should(o.BeNumerically("==", 0), "expected number of flows with SYN_ACK TCPFlag = 0")
		verifyEBPFFilterMetrics("FilterReject")

		g.By("Verify SYN flooding flows")
		parameters = []string{"Flags=\"SYN\"", "DstAddr=\"192.168.1.159\""}

		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of SYN flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Bytes).Should(o.BeNumerically("==", 54))
		}

		g.By("Wait for alerts to be pending")
		waitForAlertToBePending("NetObserv-SYNFlood-out")
		waitForAlertToBePending("NetObserv-SYNFlood-in")
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-Medium-78480-NetObserv with sampling 50 [Serial][Slow]", func() {
		g.By("Deploy DNS pods")
		DNSTemplate := filePath.Join(baseDir, "DNS-pods.yaml")
		DNSNamespace := "dns-traffic"
		defer deleteNamespace(DNSNamespace)
		ApplyResourceFromFile(DNSNamespace, DNSTemplate)
		assertAllPodsToBeReady(DNSNamespace)

		g.By("Deploy test server and client pods")
		servertemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-78480",
			Template: servertemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-78480",
			Template: clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		g.By("Deploy FlowCollector with all features enabled with sampling 50")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFPrivileged:    "true",
			EBPFeatures:       []string{"\"DNSTracking\", \"PacketDrop\", \"FlowRTT\", \"PacketTranslation\""},
			Sampling:          "50",
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for 4 mins before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(240 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}

		g.By("Verify Packet Drop flows")
		lokiParams := []string{"PktDropLatestState=\"TCP_INVALID_STATE\"", "Proto=\"6\""}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP Invalid State flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.PktDropLatestDropCause).NotTo(o.BeEmpty())
			o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
			o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
		}

		lokiParams = []string{"PktDropLatestDropCause=\"SKB_DROP_REASON_NO_SOCKET\"", "Proto=\"6\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of No Socket TCP flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.PktDropLatestState).NotTo(o.BeEmpty())
			o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
			o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
		}

		g.By("Verify flowRTT flows")
		lokiParams = []string{"Proto=\"6\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.TimeFlowRttNs).Should(o.BeNumerically(">=", 0))
		}

		g.By("Verify TCP DNS flows")
		lokilabels.DstK8SNamespace = DNSNamespace
		lokiParams = []string{"DnsFlagsResponseCode=\"NoError\"", "SrcPort=\"53\"", "DstK8S_Name=\"dnsutils1\"", "Proto=\"6\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP DNS flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DNSLatencyMs).Should(o.BeNumerically(">=", 0))
		}

		g.By("Verify UDP DNS flows")
		lokiParams = []string{"DnsFlagsResponseCode=\"NoError\"", "SrcPort=\"53\"", "Proto=\"17\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of UDP DNS flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.DNSLatencyMs).Should(o.BeNumerically(">=", 0))
		}

		g.By("Verify Packet Translation flows")
		lokilabels = Lokilabels{
			App:             "netobserv-flowcollector",
			DstK8SOwnerName: "nginx-service",
			DstK8SNamespace: testClientTemplate.ServerNS,
			SrcK8SNamespace: testClientTemplate.ClientNS,
			SrcK8SOwnerName: "client",
		}

		ipStackType := checkIPStackType()
		clientServiceInfo, err := getClientServerInfo(testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify PacketTranslation flows")
		lokiParams = []string{
			fmt.Sprintf(`XlatDstAddr="%s"`, clientServiceInfo["server"]["ip"]),
			fmt.Sprintf(`XlatDstK8S_Name="%s"`, clientServiceInfo["server"]["name"]),
			`XlatDstK8S_Type="Pod"`,
			`DstPort="80"`,
			`XlatDstPort="8080"`,
			fmt.Sprintf(`XlatSrcAddr="%s"`, clientServiceInfo["client"]["ip"]),
			`XlatSrcK8S_Name="client"`,
		}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of PacketTranslation flows > 0")

		g.By("Verify eBPF feature metrics")
		verifyEBPFFeatureMetrics("pktdropsmap")
		verifyEBPFFeatureMetrics("additionalmap") // for RTT/IPSec map size
		verifyEBPFFeatureMetrics("dnsmap")
		verifyEBPFFeatureMetrics("xlatmap")
	})

	g.It("Author:aramesha-NonPreRelease-High-79015-Verify PacketTranslation feature [Serial]", func() {
		g.By("Deploy test server and client pods")
		servertemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS:    "test-server-79015",
			ServiceType: "ClusterIP",
			Template:    servertemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-79015",
			Template: clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		ipStackType := checkIPStackType()
		clientServiceInfo, err := getClientServerInfo(testClientTemplate.ServerNS, testClientTemplate.ClientNS, ipStackType)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy FlowCollector with PacketTranslation feature enabled")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFeatures:       []string{"\"PacketTranslation\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for 2 mins before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(120 * time.Second)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			DstK8SOwnerName: "nginx-service",
			DstK8SNamespace: testClientTemplate.ServerNS,
			SrcK8SNamespace: testClientTemplate.ClientNS,
			SrcK8SOwnerName: "client",
		}

		g.By("Verify PacketTranslation flows")
		lokiParams := []string{
			fmt.Sprintf(`XlatDstAddr="%s"`, clientServiceInfo["server"]["ip"]),
			fmt.Sprintf(`XlatDstK8S_Name="%s"`, clientServiceInfo["server"]["name"]),
			`XlatDstK8S_Type="Pod"`,
			`DstPort="80"`,
			`XlatDstPort="8080"`,
			fmt.Sprintf(`XlatSrcAddr="%s"`, clientServiceInfo["client"]["ip"]),
			`XlatSrcK8S_Name="client"`,
		}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of PacketTranslation flows > 0")
	})

	// NetworkEvents ebpf hook only supported for OCP >= 4.19
	g.It("Author:memodi-NonPreRelease-Medium-77894-TechPreview Network Policies Correlation [Serial]", func() {
		SkipIfOCPBelow("v4.19")
		if !isTechPreviewNoUpgrade() {
			g.Skip("Skipping because the TechPreviewNoUpgrade is not enabled on the cluster.")
		}

		g.By("Deploy client-server pods in 2 client NS and one Server NS")
		serverTemplate := filePath.Join(baseDir, "test-nginx-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-77894",
			Template: serverTemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		client1Template := filePath.Join(baseDir, "test-nginx-client_template.yaml")
		testClient1Template := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client1-77894",
			Template: client1Template,
		}
		defer deleteNamespace(testClient1Template.ClientNS)
		err = testClient1Template.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClient1Template.ClientNS)

		testClient2Template := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client2-77894",
			Template: client1Template,
		}
		defer deleteNamespace(testClient2Template.ClientNS)
		err = testClient2Template.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClient2Template.ClientNS)

		// create flowcollector with NWEvents.
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFeatures:       []string{"\"NetworkEvents\""},
			EBPFPrivileged:    "true",
		}
		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

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
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-2*time.Minute), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")

		g.By("deploy BANP policy")
		banpTemplate := filePath.Join(baseDir, "networking", "baselineadminnetworkPolicy.yaml")
		banpParameters := []string{"--ignore-unknown-parameters=true", "-p", "SERVER_NS=" + testClient1Template.ServerNS, "CLIENT1_NS=" + testClient1Template.ClientNS, "CLIENT2_NS=" + testClient2Template.ClientNS, "-f", banpTemplate}

		// banp is a cluster scoped resource so passing empty string for NS arg.
		defer deleteResource("baselineadminnetworkpolicy", "default", "")
		err = applyResourceFromTemplateByAdmin(banpParameters...)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for 60 secs before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		g.By("check flows have NW Events")
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-45*time.Second), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
		verifyNetworkEvents(flowRecords, Drop, "BaselineAdminNetworkPolicy", "Ingress")

		g.By("deploy NetworkPolicy")
		netpolTemplate := filePath.Join(baseDir, "networking", "networkPolicy.yaml")
		netpolName := "allow-ingress"
		netPolParameters := []string{"--ignore-unknown-parameters=true", "-p", "NAME=" + netpolName, "SERVER_NS=" + testClient1Template.ServerNS, "ALLOW_NS=" + testClient1Template.ClientNS, "-f", netpolTemplate}
		defer deleteResource("netpol", netpolName, testClient1Template.ServerNS)
		err = applyResourceFromTemplateByAdmin(netPolParameters...)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for 60 secs before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		g.By("check flows from server to client1")
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-1*time.Minute), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
		verifyNetworkEvents(flowRecords, AllowRelated, "NetworkPolicy", "Ingress")

		g.By("check flows from server to client2")
		lokilabels.SrcK8SNamespace = testClient2Template.ClientNS
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-1*time.Minute), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
		verifyNetworkEvents(flowRecords, Drop, "NetpolNamespace", "Ingress")

		g.By("deploy ANP policy")
		anpTemplate := filePath.Join(baseDir, "networking", "adminnetworkPolicy.yaml")
		anpName := "server-ns"
		anpParameters := []string{"--ignore-unknown-parameters=true", "-p", "NAM=" + anpName, "SERVER_NS=" + testClient1Template.ServerNS, "ALLOW_NS=" + testClient2Template.ClientNS, "DENY_NS=" + testClient1Template.ClientNS, "-f", anpTemplate}
		defer deleteResource("adminnetworkpolicy", anpName, "")
		err = applyResourceFromTemplateByAdmin(anpParameters...)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for 60 secs before logs gets collected and written to loki")
		time.Sleep(60 * time.Second)

		g.By("check flows from server to client2")
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-1*time.Minute), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
		verifyNetworkEvents(flowRecords, AllowRelated, "AdminNetworkPolicy", "Ingress")

		g.By("check flows from server to client1")
		lokilabels.SrcK8SNamespace = testClient1Template.ClientNS
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, time.Now().Add(-1*time.Minute), lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flowRecords with 'flowDirection != 1' > 0")
		verifyNetworkEvents(flowRecords, Drop, "AdminNetworkPolicy", "Ingress")
	})

	g.It("Author:aramesha-NonPreRelease-High-80090-Verify FLP tail-based filtering [Serial]", func() {
		// Accept flows with Source Namespace = < namespace > and
		// Source Name containing 'flowlogs-pipeline-' and
		// NOT Source Port 9401 and
		// having field TimeFlowRttNs
		g.By("Deploy FlowCollector with FLP tail-based filter and FlowRTT enabled")
		FLPFiltersConfig := []map[string]any{
			{
				"query":        fmt.Sprintf(`SrcK8S_Namespace="%s" and SrcK8S_Name=~"flowlogs-pipeline.*" and SrcPort!=9401 and with(TimeFlowRttNs)`, namespace),
				"outputTarget": "Loki",
				"sampling":     2,
			},
		}

		config, err := json.Marshal(FLPFiltersConfig)
		o.Expect(err).ToNot(o.HaveOccurred())
		FLPFilter := string(config)

		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			Sampling:          "2",
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			EBPFeatures:       []string{`"FlowRTT"`},
			FLPFilters:        FLPFilter,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		// verify logs
		g.By("Wait for a min before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: namespace,
		}

		g.By("Verify number of flows > 0")
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of flows > 0")

		for _, r := range flowRecords {
			o.Expect(r.Flowlog.SrcK8SName).Should(o.ContainSubstring("flowlogs-pipeline"))
			o.Expect(r.Flowlog.SrcPort).ShouldNot(o.BeNumerically("==", 9401))
			o.Expect(r.Flowlog.TimeFlowRttNs).Should(o.BeNumerically(">", 0))
			o.Expect(r.Flowlog.Sampling).Should(o.BeNumerically("==", 4))
		}
	})

	g.It("Author:aramesha-High-81677-Validate UDN with NetObserv [Serial]", func() {
		SkipIfOCPBelow("v4.18")
		var (
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
		defer deleteNamespace(cudnNS[0])
		defer deleteNamespace(cudnNS[1])
		createUDNNamespace(cudnNS[0])
		createUDNNamespace(cudnNS[1])
		for _, ns := range cudnNS {
			defer func(namespace string) {
				_ = removeLabelFromNamespace(namespace, matchLabelKey)
			}(ns)
			err := labelNamespace(ns, matchLabelKey, matchValue)
			o.Expect(err).NotTo(o.HaveOccurred())
		}

		defer deleteNamespace(udnNS)
		createUDNNamespace(udnNS)

		g.By("Deploy CUDN in CUDNns")
		ipStackType := checkIPStackType()
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
		defer removeResource(true, true, "clusteruserdefinednetwork", cudnName)
		_, err := applyCUDNtoMatchLabelNS(matchLabelKey, matchValue, cudnName, ipv4cidr, ipv6cidr, cidr, "layer3")
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
		createGeneralUDNCRD(udnNS, udnName, ipv4cidr, ipv6cidr, cidr, "layer2")

		g.By("Deploy a pod in each CUDN namespace")
		CUDNpods := make([]udnPodResource, 2)
		for i, ns := range cudnNS {
			CUDNpods[i] = udnPodResource{
				name:      "hello-pod-" + ns,
				namespace: ns,
				label:     "hello-pod",
				template:  udnPodTemplate,
			}
			defer removeResource(true, true, "pod", CUDNpods[i].name, "-n", CUDNpods[i].namespace)
			CUDNpods[i].createUdnPod()
			assertAllPodsToBeReady(CUDNpods[i].namespace)
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
			defer removeResource(true, true, "pod", UDNpods[j].name, "-n", UDNpods[j].namespace)
			UDNpods[j].createUdnPod()
		}
		assertAllPodsToBeReady(udnNS)

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

		startTime := time.Now()

		g.By("Validate isolation from an UDN pod to a CUDN pod")
		CurlPod2PodFailUDN(udnNS, UDNpods[1].name, CUDNpods[0].namespace, CUDNpods[0].name)
		//default network connectivity is isolated
		CurlPod2PodFail(udnNS, UDNpods[1].name, CUDNpods[0].namespace, CUDNpods[0].name, ipStackType)

		g.By("Validate isolation from a CUDN pod to an UDN pod")
		CurlPod2PodFailUDN(CUDNpods[1].namespace, CUDNpods[1].name, udnNS, UDNpods[1].name)
		//default network connectivity is isolated
		CurlPod2PodFail(CUDNpods[1].namespace, CUDNpods[1].name, udnNS, UDNpods[1].name, ipStackType)

		g.By("Validate connection among CUDN pods")
		CurlPod2PodPassUDN(CUDNpods[0].namespace, CUDNpods[0].name, CUDNpods[1].namespace, CUDNpods[1].name)
		//default network connectivity is isolated
		CurlPod2PodFail(CUDNpods[0].namespace, CUDNpods[0].name, CUDNpods[1].namespace, CUDNpods[1].name, ipStackType)

		g.By("Validate connection among UDN pods")
		CurlPod2PodPassUDN(udnNS, UDNpods[0].name, udnNS, UDNpods[1].name)
		//default network connectivity is isolated
		CurlPod2PodFail(udnNS, UDNpods[1].name, udnNS, UDNpods[0].name, ipStackType)

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

		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
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

		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of UDN flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Udns).Should(o.ContainElement(fmt.Sprintf("%s/%s", udnNS, udnName)))
			o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
			o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(fmt.Sprintf("%s/%s", udnNS, udnName)))
		}
	})

	g.It("Author:aramesha-High-83022-Validate CUDN with Localnet [Serial]", func() {
		SkipIfOCPBelow("v4.19")
		var (
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
		nmstateCR := nmstateCRResource{
			name:     "nmstate",
			template: nmstateCRTemplate,
		}
		defer deleteNMStateCR(nmstateCR)
		result, crErr := createNMStateCR(nmstateCR, opNamespace)
		assertWaitPollNoErr(crErr, "create nmstate cr failed")
		o.Expect(result).To(o.BeTrue())
		e2e.Logf("SUCCESS - NMState CR Created")

		g.By("Configure NNCP for creating OvnMapping NMstate Feature")
		ovnMappingPolicy := ovnMappingPolicyResource{
			name:       "bridge-mapping-83022",
			nodelabel:  nodeSelectLabel,
			labelvalue: "",
			localnet1:  "mylocalnet",
			bridge1:    "br-ex",
			template:   ovnMappingPolicyTemplate,
		}
		defer deleteNNCP(ovnMappingPolicy.name)
		defer func() {
			ovnmapping, deferErr := debugNodeWithCommand(workerNode, "ovs-vsctl get Open_vSwitch . external_ids:ovn-bridge-mappings")
			o.Expect(deferErr).NotTo(o.HaveOccurred())
			if strings.Contains(ovnmapping, ovnMappingPolicy.localnet1) {
				_, err := debugNodeWithCommand(workerNode, "ovs-vsctl set Open_vSwitch . external_ids:ovn-bridge-mappings=\"physnet:br-ex\"")
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
		defer removeResource(true, true, "clusteruserdefinednetwork", secondaryCUDNName)
		_, err := applyLocalnetCUDNtoMatchLabelNS(matchLabelKey, matchValue, secondaryCUDNName, "mylocalnet", "192.168.100.0/24", "192.168.100.1/32", false)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy statefulset in both cudnNS")
		for _, ns := range cudnNS {
			defer removeResource(true, true, "statefulset", "hello", "-n", ns)
			err := applyNsResourceFromTemplateByAdmin(ns, "-f", udnStatefulSetTemplate, "NETWORK_NAME="+secondaryCUDNName)
			o.Expect(err).NotTo(o.HaveOccurred())
			assertAllPodsToBeReady(ns)
		}

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

		g.By("Validate connection among CUDN pods")
		cudn1Pods, err := getAllPods(cudnNS[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		cudn2Pods, err := getAllPods(cudnNS[1])
		o.Expect(err).NotTo(o.HaveOccurred())
		startTime := time.Now()
		CurlPod2PodPassUDN(cudnNS[0], cudn1Pods[0], cudnNS[1], cudn2Pods[0])

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

		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of CUDN Localnet flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.Udns).Should(o.ContainElement(secondaryCUDNName))
			o.Expect(r.Flowlog.DstK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
			o.Expect(r.Flowlog.SrcK8SNetworkName).Should(o.ContainSubstring(secondaryCUDNName))
		}
	})

	g.It("Author:aramesha-NonPreRelease-Longduration-Medium-81410-NetObserv with eBPF manager [Serial][Slow]", func() {
		SkipIfOCPBelow("v4.18")
		g.By("Deploy eBPF manager operator")
		// eBPF manager operator variables
		bpfDir, _ := filePath.Abs("testdata/bpfman")
		bpfIDMS := filePath.Join(bpfDir, "image-digest-mirror-set.yaml")
		bpfCatSrcTemplate := filePath.Join(bpfDir, "catalog-source.yaml")

		bpfNS := OperatorNamespace{
			Name:              "bpfman",
			NamespaceTemplate: filePath.Join(bpfDir, "namespace.yaml"),
		}
		bpfCatSrc := Resource{"catalogsource", "bpfman-konflux-fbc", bpfNS.Name}

		g.By("Deploy bpfman konflux FBC and ImageDigestMirrorSet")
		bpfNS.DeployOperatorNamespace()
		catsrcErr := bpfCatSrc.applyFromTemplate("-n", bpfNS.Name, "-f", bpfCatSrcTemplate, "-p", "NAMESPACE="+bpfNS.Name)
		o.Expect(catsrcErr).NotTo(o.HaveOccurred())
		bpfCatSrc.WaitUntilCatSrcReady()
		ApplyResourceFromFile(bpfNS.Name, bpfIDMS)

		bpfChannel, err := getOperatorChannel(bpfCatSrc.Name, "bpfman-operator")
		if err != nil || bpfChannel == "" {
			g.Skip("bpfman-operator channel not found, skip this case")
		}
		bpfSource := CatalogSourceObjects{bpfChannel, bpfCatSrc.Name, bpfNS.Name}

		BPF := SubscriptionObjects{
			OperatorName:  "bpfman-operator",
			Namespace:     "bpfman",
			PackageName:   "bpfman-operator",
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
			CatalogSource: &bpfSource,
		}

		bpfExisting, err := CheckOperatorStatus(BPF.Namespace, BPF.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())
		// Deploy eBPF manager operator if not present
		if !bpfExisting {
			ensureOperatorDeployed(BPF, bpfSource, "name=bpfman-daemon")
		}

		g.By("Deploy FlowCollector with PacketDrop and Ebpfmanager enabled")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFeatures:       []string{"\"PacketDrop\", \"EbpfManager\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for 2 mins before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(120 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}

		g.By("Verify Packet Drop flows")
		lokiParams := []string{"PktDropLatestState=\"TCP_INVALID_STATE\"", "Proto=\"6\""}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of TCP Invalid State flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.PktDropLatestDropCause).NotTo(o.BeEmpty())
			o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
			o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
		}

		lokiParams = []string{"PktDropLatestDropCause=\"SKB_DROP_REASON_NO_SOCKET\"", "Proto=\"6\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of No Socket TCP flows > 0")
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.PktDropLatestState).NotTo(o.BeEmpty())
			o.Expect(r.Flowlog.PktDropBytes).Should(o.BeNumerically(">", 0))
			o.Expect(r.Flowlog.PktDropPackets).Should(o.BeNumerically(">", 0))
		}
	})

	g.It("Author:memodi-NonPreRelease-High-82637-Verify IPSec feature [Disruptive]", func() {
		SkipIfOCPBelow("v4.16")
		g.By("Check if IPSec is enabled in the cluster")
		netObj, err := getDynamicResource("network.operator", "cluster", "")
		o.Expect(err).NotTo(o.HaveOccurred())
		ipsecEnabled, _, _ := unstructured.NestedString(netObj.Object, "spec", "defaultNetwork", "ovnKubernetesConfig", "ipsecConfig", "mode")
		if ipsecEnabled != "Full" {
			g.Skip("IPSec is not enabled in Full mode, skipping test")
		}

		g.By("Deploy FlowCollector IPSec enabled")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFeatures:       []string{"\"IPSec\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for 2 mins before logs gets collected and written to loki")
		startTime := time.Now()
		time.Sleep(120 * time.Second)

		lokilabels := Lokilabels{
			App: "netobserv-flowcollector",
		}
		g.By("Verify IPSec flows")
		lokiParams := []string{"IPSecStatus=\"success\""}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of IPSecStatus==success flows > 0")
		metrics, err := getMetric("sum(netobserv_node_ipsec_flows_total{IPSecStatus=\"success\"})")
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(popMetricValue(metrics)).Should(o.BeNumerically(">", 0))
		o.Expect(err).NotTo(o.HaveOccurred())
		verifyEBPFFeatureMetrics("additionalmap") // additionalMap for RTT/IPSec map size
	})

	g.It("Author:kapjain-NonPreRelease-Longduration-High-85953-Verify FlowCollector Service deployment model [Serial][Slow]", func() {
		g.By("Deploy FlowCollector with Service deployment model")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Wait for pods to fully start and emit startup logs")
		time.Sleep(30 * time.Second)

		g.By("Verify FLP logs show 'Starting GRPC server with TLS'")
		FLPpods, err := getAllPodsWithLabel(flow.Namespace, "app=flowlogs-pipeline")
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Get FLP pod logs to check GRPC server startup message")
		flpLogs, err := getPodLogs(flow.Namespace, FLPpods[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(flpLogs).To(o.ContainSubstring("Starting GRPC server with TLS"), "FLP logs should contains 'Starting GRPC server with TLS'")

		g.By("Verify agent logs show 'Starting GRPC client with TLS'")
		agentPods, err := getAllPodsWithLabel(flow.Namespace+"-privileged", "app=netobserv-ebpf-agent")
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Get agent pod logs to check GRPC client startup message")
		agentLogs, err := getPodLogs(flow.Namespace+"-privileged", agentPods[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(agentLogs).To(o.ContainSubstring("Starting GRPC client with TLS"), "Agent logs should contains 'Starting GRPC client with TLS'")

		g.By("Wait for a min before logs gets collected and written to loki in TLS mode")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		g.By("Get flowlogs from loki")
		err = verifyMonolithicLokilogsTime(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify default FLP Deployment is created with 3 pods")
		result := verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 3, "")
		o.Expect(result).To(o.BeTrue(), "By default the replica count should be 3")

		g.By("Verify Service is created with correct port configuration")
		svc, err := k8sClient.CoreV1().Services(flow.Namespace).Get(context.Background(), "flowlogs-pipeline", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(fmt.Sprintf("%d", svc.Spec.Ports[0].Port)).To(o.Equal("2055"))
		o.Expect(svc.Spec.Ports[0].TargetPort.String()).To(o.Equal("2055"))

		// Test replica management with unmanagedReplicas: False by default
		g.By("Verify deployment does not upscale when unmanagedReplicas is false or not set")
		err = scaleDeployment(flow.Namespace, "flowlogs-pipeline", 4)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 3, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should not scale when unmanagedReplicas is false or not set")

		g.By("Verify deployment does not downscale when unmanagedReplicas is false or not set")
		err = scaleDeployment(flow.Namespace, "flowlogs-pipeline", 2)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 3, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should not scale when unmanagedReplicas is false or not set")

		g.By("Verify deployment scales via consumerReplicas when unmanagedReplicas is false or not set - upscale to 4")
		err = patchFlowCollectorMerge(`{"spec":{"processor":{"consumerReplicas":4}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 4, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should scale via consumerReplicas when unmanagedReplicas is false or not set")

		g.By("Verify deployment scales via consumerReplicas when unmanagedReplicas is false or not set - downscale to 2")
		err = patchFlowCollectorMerge(`{"spec":{"processor":{"consumerReplicas":2}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 2, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should scale via consumerReplicas when unmanagedReplicas is false or not set")

		// Test replica management with unmanagedReplicas: True
		g.By("Enable unmanagedReplicas and set consumerReplicas to 3")
		err = patchFlowCollectorMerge(`{"spec":{"processor":{"unmanagedReplicas":true,"consumerReplicas":3}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verify deployment upscales when unmanagedReplicas is true")
		err = scaleDeployment(flow.Namespace, "flowlogs-pipeline", 4)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 4, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should scale when unmanagedReplicas is true")

		g.By("Verify deployment downscales when unmanagedReplicas is true")
		err = scaleDeployment(flow.Namespace, "flowlogs-pipeline", 1)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 1, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should scale when unmanagedReplicas is true")

		g.By("Verify consumerReplicas change does not scale deployment when unmanagedReplicas is true")
		err = patchFlowCollectorMerge(`{"spec":{"processor":{"consumerReplicas":4}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 1, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should not scale via consumerReplicas when unmanagedReplicas is true")

		g.By("Verify HPA scales the deployment when unmanagedReplicas is true")
		hpaYAML := filePath.Join(baseDir, "flowlogs_pipeline_hpa_template.yaml")
		hpaFile, err := processTemplate(namespace, "--ignore-unknown-parameters=true", "-f", hpaYAML, "-p", "NAMESPACE="+namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		defer func() {
			_ = k8sClient.AutoscalingV1().HorizontalPodAutoscalers(flow.Namespace).Delete(context.Background(), "flowlogs-pipeline-hpa", metav1.DeleteOptions{})
		}()
		ApplyResourceFromFile("", hpaFile)
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 4, ">")
		o.Expect(result).To(o.BeTrue(), "HPA should scale deployment above 4 replicas when unmanagedReplicas is true")

		g.By("Verify HPA does not scale deployment when unmanagedReplicas is false")
		err = patchFlowCollectorMerge(`{"spec":{"processor":{"unmanagedReplicas":false,"consumerReplicas":2}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())
		result = verifyDeploymentReplicas("flowlogs-pipeline", flow.Namespace, 2, "")
		o.Expect(result).To(o.BeTrue(), "Deployment should be reconciled to consumerReplicas=2 when unmanagedReplicas is false")
	})

	g.It("Author:kapjain-Medium-86372-Verify Gateway API three-level owner metadata [Serial]", func() {
		SkipIfOCPBelow("v4.19")
		startTime := time.Now()
		g.By("Deploy flowcollector")
		gatewayAPITemplate := filePath.Join(baseDir, "gateway-api-template.yaml")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Deploying Gateway API resources from template")
		gatewayNS := "netobserv-gateway-test"
		gatewayName := "test-gateway-owner"
		defer deleteNamespace(gatewayNS)
		err := applyResourceFromTemplateByAdmin("-f", gatewayAPITemplate)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Verifying Gateway Deployment exists")
		// The Gateway controller creates a Deployment named gateway-name + gatewayclass-name
		deploymentName := gatewayName + "-openshift-default"
		WaitForDeploymentPodsToBeReady(gatewayNS, deploymentName)

		g.By("Verifying Pods are created by Gateway")
		pods, err := getAllPodsWithLabel(gatewayNS, fmt.Sprintf("gateway.networking.k8s.io/gateway-name=%s", gatewayName))
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(pods)).Should(o.BeNumerically(">", 0), "expected at least one Gateway pod")

		g.By("Waiting for flow data to be collected and written to Loki")
		time.Sleep(120 * time.Second)

		g.By("Querying flow data from Loki for Gateway pods")
		lokilabels := Lokilabels{
			SrcK8SNamespace: "netobserv-gateway-test",
		}
		parameters := []string{"SrcK8S_OwnerType=\"Gateway\""}
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, parameters...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of Gateway Owner flows > 0")
	})
	g.It("Author:kapjain-Medium-88334-Pause Network Observability functions [Serial]", func() {
		g.By("Create a FlowCollector")
		flow := Flowcollector{
			Namespace:         namespace,
			Template:          flowFixturePath,
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			DeploymentModel:   "Service",
			SlicesEnable:      "true",
		}
		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Get all netobserv-managed components before pause (excluding pods with dynamic IDs)")
		componentsBeforePause, err := listManagedResources()
		o.Expect(err).NotTo(o.HaveOccurred())
		e2e.Logf("Components before pause (stable names): %v", componentsBeforePause)

		g.By("Pause the FlowCollector")
		err = patchFlowCollectorMerge(`{"spec":{"execution":{"mode":"OnHold"}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for FlowCollector status to show 'on hold' message")
		err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 150*time.Second, false, func(context.Context) (done bool, err error) {
			fcObj, getErr := getDynamicResource("flowcollector", "cluster", "")
			if getErr != nil {
				e2e.Logf("Error getting FlowCollector status: %v", getErr)
				return false, nil
			}
			conditions, found, _ := unstructured.NestedSlice(fcObj.Object, "status", "conditions")
			if !found {
				e2e.Logf("Waiting for FlowCollector to show 'on hold' status...")
				return false, nil
			}
			for _, c := range conditions {
				cond, ok := c.(map[string]interface{})
				if ok {
					if msg, _ := cond["message"].(string); msg == "FlowCollector is on hold" {
						e2e.Logf("FlowCollector status shows 'on hold'")
						return true, nil
					}
				}
			}
			e2e.Logf("Waiting for FlowCollector to show 'on hold' status...")
			return false, nil
		})
		o.Expect(err).NotTo(o.HaveOccurred())

		// Components with stable names that should remain when paused
		// Common components across all OCP versions
		componentsShouldRemain := []string{
			"configmap/grafana-dashboard-netobserv-health",
			"configmap/netobserv-main",
			"service/loki",
			"deployment.apps/loki",
			"configmap/loki-config",
		}
		// Static plugin and network policy are only available on OCP 4.15+
		if IsOCPVersionAtLeast("v4.15") {
			componentsShouldRemain = append(componentsShouldRemain,
				"deployment.apps/netobserv-plugin-static",
				"service/netobserv-plugin-static",
				"networkpolicy.networking.k8s.io/netobserv",
			)
		}

		// Build list of components that should be deleted = originalComponentsList - componentsShouldRemain
		var componentsShouldDelete []string
		for _, component := range componentsBeforePause {
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

		g.By("Verify except for netobserv-plugin-static and network policies and persistent configmaps, all components are deleted")
		pollVerifyComponentsDeleted(componentsShouldDelete, componentsShouldRemain)

		// Verify netobserv-plugin-static pod exists and other pods are deleted (using pattern since pod names have dynamic IDs)
		podsAfterPause, err := listManagedPods()
		o.Expect(err).NotTo(o.HaveOccurred())
		if IsOCPVersionAtLeast("v4.15") {
			o.Expect(podsAfterPause).Should(o.ContainElement(o.ContainSubstring("pod/netobserv-plugin-static-")), "netobserv-plugin-static pod should exist after pause")
		}
		o.Expect(podsAfterPause).ShouldNot(o.ContainElement(o.ContainSubstring("pod/flowlogs-pipeline-")), "flowlogs-pipeline pods should be deleted")
		o.Expect(podsAfterPause).ShouldNot(o.ContainElement(o.ContainSubstring("pod/netobserv-ebpf-agent-")), "netobserv-ebpf-agent pods should be deleted")
		// Verify regular netobserv-plugin pod is deleted (not the static one)
		o.Expect(podsAfterPause).ShouldNot(o.ContainElement(o.And(
			o.ContainSubstring("pod/netobserv-plugin-"),
			o.Not(o.ContainSubstring("pod/netobserv-plugin-static-")),
		)), "non-static netobserv-plugin pods should be deleted")

		g.By("Resume the FlowCollector")
		resumeTime := time.Now()
		err = patchFlowCollectorMerge(`{"spec":{"execution":{"mode":"Running"}}}`)
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for FlowCollector to be ready")
		flow.WaitForFlowcollectorReady()

		g.By("Verify no 'on hold' message in FlowCollector status")
		fcResumeObj, errResume := getDynamicResource("flowcollector", "cluster", "")
		o.Expect(errResume).NotTo(o.HaveOccurred())
		resumeConditions, _, _ := unstructured.NestedSlice(fcResumeObj.Object, "status", "conditions")
		onHoldFound := false
		for _, c := range resumeConditions {
			cond, ok := c.(map[string]interface{})
			if ok {
				if msg, _ := cond["message"].(string); msg == "FlowCollector is on hold" {
					onHoldFound = true
				}
			}
		}
		o.Expect(onHoldFound).Should(o.BeFalse())

		g.By("Wait for a min before logs gets collected and written to loki after resume")
		time.Sleep(60 * time.Second)

		g.By("Verify flows are being created in Loki after resume")
		err = verifyMonolithicLokilogsTime(flow.MonolithicLokiURL, resumeTime)
		o.Expect(err).NotTo(o.HaveOccurred())
	})

	g.It("Author:aramesha-NonPreRelease-High-88455-Verify TLS Tracking feature [Serial]", func() {
		g.By("Deploy TLS test server and client pods")
		servertemplate := filePath.Join(baseDir, "test-tls-server_template.yaml")
		testServerTemplate := TestServerTemplate{
			ServerNS: "test-server-88455",
			Template: servertemplate,
		}
		defer deleteNamespace(testServerTemplate.ServerNS)
		err := testServerTemplate.createServer()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testServerTemplate.ServerNS)

		clientTemplate := filePath.Join(baseDir, "test-tls-client_template.yaml")
		testClientTemplate := TestClientTemplate{
			ServerNS: testServerTemplate.ServerNS,
			ClientNS: "test-client-88455",
			Template: clientTemplate,
		}
		defer deleteNamespace(testClientTemplate.ClientNS)
		err = testClientTemplate.createClient()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testClientTemplate.ClientNS)

		g.By("Deploy FlowCollector with TLS Tracking feature enabled")
		flow := Flowcollector{
			Namespace:         namespace,
			EBPFeatures:       []string{"\"TLSTracking\""},
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

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
		flowRecords, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTP flows > 0")
		// Verify TLS fields are not populated
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.TLSVersion).Should(o.BeEmpty(), "expected TLS version to be empty for HTTP")
		}

		g.By("Verify HTTPS flows with TLSVersion 1.3")
		lokiParams = []string{"Proto=\"6\"", "SrcPort=\"443\"", "TLSVersion=\"TLS 1.3\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTPS flows with TLSv1.3 > 0")
		// Verify TLS 1.3 fields
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.TLSTypes).Should(o.ContainElement("ServerHello"), "expected TLS Types to contain ServerHello")
			o.Expect(r.Flowlog.TLSGroup).NotTo(o.BeEmpty())
			o.Expect(r.Flowlog.TLSCipherSuite).NotTo(o.BeEmpty())
		}

		g.By("Verify HTTPS flows with TLSVersion 1.2")
		lokiParams = []string{"Proto=\"6\"", "SrcPort=\"443\"", "TLSVersion=\"TLS 1.2\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTPS flows with TLSv1.2 > 0")
		// Verify TLS 1.2 fields
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.TLSTypes).Should(o.ContainElement("ServerHello"), "expected TLS Types to contain ServerHello")
			o.Expect(r.Flowlog.TLSGroup).Should(o.BeEmpty(), "expected TLS Group to be empty for TLS 1.2")
			o.Expect(r.Flowlog.TLSCipherSuite).NotTo(o.BeEmpty())
		}

		g.By("Verify HTTPS flows with TLSVersion 1.1")
		lokiParams = []string{"Proto=\"6\"", "SrcPort=\"443\"", "TLSVersion=\"TLS 1.1\""}
		flowRecords, err = lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, lokiParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "expected number of HTTPS flows with TLSv1.1 > 0")
		// Verify TLS 1.1 fields
		for _, r := range flowRecords {
			o.Expect(r.Flowlog.TLSTypes).Should(o.ContainElement("ServerHello"), "expected TLS Types to contain ServerHello")
			o.Expect(r.Flowlog.TLSGroup).Should(o.BeEmpty(), "expected TLS Group to be empty for TLS 1.1")
			o.Expect(r.Flowlog.TLSCipherSuite).Should(o.BeEmpty(), "expected TLS CipherSuite to be empty for TLS 1.1")
		}

		g.By("Verify TLS metrics")
		verifyTLSMetrics("TLSVersion")
		verifyTLSMetrics("TLSCipherSuite")
		verifyTLSMetrics("TLSGroup")

		g.By("Wait for TLSInsecureVersion alert to be pending")
		waitForAlertToBePending("TLSInsecureVersion_PerSrcNamespaceWarning")

		g.By("Verify TLS alert has expected labels")
		alertLabels, err := getAlertLabels("TLSInsecureVersion_PerSrcNamespaceWarning")
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(alertLabels).To(o.HaveKeyWithValue("severity", "warning"))
		o.Expect(alertLabels).To(o.HaveKeyWithValue("netobserv", "true"))
	})

	g.It("Author:kapjain-Medium-88683-Secure communications between Agent and FLP [Serial]", func() {
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
		certManagerExisting, err := CheckOperatorStatus(certManager.Namespace, certManager.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())

		certManagerChannel, err := getOperatorChannel(certManagerCatalog, certManagerPackageName)
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
			certManagerNSObj.DeployOperatorNamespace()

			ensureOperatorDeployed(certManager, certManagerSource, "name=cert-manager-operator")
		}

		defer func() {
			if !certManagerExisting {
				certManager.uninstallOperator()
				deleteNamespace(certManagerNS)
			}
		}()

		g.By("Wait for cert-manager CRDs to be available")
		err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
			_, issuerErr := getDynamicResource("crd", "issuers.cert-manager.io", "")
			_, certErr := getDynamicResource("crd", "certificates.cert-manager.io", "")
			if issuerErr == nil && certErr == nil {
				e2e.Logf("cert-manager CRDs are available")
				return true, nil
			}
			e2e.Logf("Waiting for cert-manager CRDs to be available...")
			return false, nil
		})
		assertWaitPollNoErr(err, "cert-manager CRDs did not become available")

		g.By("Wait for cert-manager webhook to be ready")
		waitUntilDeploymentReady("cert-manager-webhook", "cert-manager")

		g.By("Create certificates using cert-manager")
		certFile, err := processTemplate(namespace, "--ignore-unknown-parameters=true", "-f", certTemplatePath, "-p", "Namespace="+namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		// Retry apply because cert-manager webhook may not be ready immediately after CRDs appear
		err = wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 300*time.Second, true, func(context.Context) (done bool, err error) {
			applyCmd := exec.Command("oc", "apply", "-f", certFile, "-n", namespace)
			output, applyErr := applyCmd.CombinedOutput()
			if applyErr != nil {
				e2e.Logf("Waiting for cert-manager webhook to be ready: %s", string(output))
				return false, nil
			}
			return true, nil
		})
		o.Expect(err).NotTo(o.HaveOccurred())
		defer func() {
			_ = exec.Command("oc", "delete", "-f", certFile, "-n", namespace).Run()
		}()

		g.By("Wait for certificate secrets to be created")
		err = wait.PollUntilContextTimeout(context.Background(), 30*time.Second, 300*time.Second, false, func(context.Context) (done bool, err error) {
			for _, secretName := range []string{"prov-netobserv-ca-secret", "prov-flowlogs-pipeline-cert", "prov-ebpf-agent-cert"} {
				_, getErr := k8sClient.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
				if getErr != nil {
					return false, nil
				}
			}
			return true, nil
		})
		assertWaitPollNoErr(err, "certificate secrets did not become available")

		g.By("Deploy FlowCollector with Provided TLS certificates")
		flow := Flowcollector{
			Namespace:                   namespace,
			Template:                    flowFixturePath,
			ServiceTLSType:              "Provided",
			ServiceCASecretName:         "prov-netobserv-ca-secret",
			ServiceServerCertSecretName: "prov-flowlogs-pipeline-cert",
			ServiceClientCertSecretName: "prov-ebpf-agent-cert",
			MonolithicLokiURL:           fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
		}

		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Verify dedicated k8scache service exists with its own serving certificate")
		k8sCacheSvc, err := k8sClient.CoreV1().Services(namespace).Get(context.Background(), "flowlogs-pipeline-k8scache", metav1.GetOptions{})
		o.Expect(err).NotTo(o.HaveOccurred(), "dedicated k8scache service should be created by the informer reconciler")
		o.Expect(k8sCacheSvc.Annotations).To(o.HaveKeyWithValue("service.beta.openshift.io/serving-cert-secret-name", "flowlogs-pipeline-k8scache-cert"))

		g.By("Verify eBPF agent is using mTLS")
		ebpfPods, err := getAllPods(namespace + "-privileged")
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(ebpfPods)).Should(o.BeNumerically(">", 0), "No eBPF agent pods found")

		ebpfLogs, err := getPodLogs(namespace+"-privileged", ebpfPods[0])
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(ebpfLogs).To(o.ContainSubstring("Starting GRPC client with mTLS"), "eBPF agent logs should show mTLS is enabled")

		g.By("Wait for flow logs to be collected and written to Loki")
		startTime := time.Now()
		time.Sleep(60 * time.Second)

		g.By("Verify flow logs are being stored in Loki using verifyLokilogsTime")
		err = verifyMonolithicLokilogsTime(flow.MonolithicLokiURL, startTime)
		o.Expect(err).NotTo(o.HaveOccurred())
	})
	//Add future NetObserv + Loki test-cases here
})
