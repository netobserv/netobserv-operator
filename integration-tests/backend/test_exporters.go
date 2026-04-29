package e2etests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	filePath "path/filepath"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
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

		OtelNS = OperatorNamespace{
			Name:              "openshift-opentelemetry-operator",
			NamespaceTemplate: filePath.Join(subscriptionDir, "namespace.yaml"),
		}

		OTELSource = CatalogSourceObjects{"stable", "redhat-operators", "openshift-marketplace"}

		OTEL = SubscriptionObjects{
			OperatorName:  "opentelemetry-operator",
			Namespace:     OtelNS.Name,
			PackageName:   "opentelemetry-product",
			Subscription:  filePath.Join(subscriptionDir, "sub-template.yaml"),
			OperatorGroup: filePath.Join(subscriptionDir, "allnamespace-og.yaml"),
			CatalogSource: &OTELSource,
		}
		namespace string
	)

	g.BeforeEach(func() {
		namespace = oc.Namespace()

		if strings.Contains(os.Getenv("E2E_RUN_TAGS"), "disconnected") {
			g.Skip("Skipping tests for disconnected profiles")
		}

		OperatorNS.DeployOperatorNamespace(oc)
		deployedUpstreamCatalogSource, catSrcErr := setupCatalogSource(oc, NOcatSrc, catSrcTemplate, imageDigest, catalogSource, false, &NOSource, &NO)
		o.Expect(catSrcErr).NotTo(o.HaveOccurred())
		ensureNetObservOperatorDeployed(oc, NO, NOSource, deployedUpstreamCatalogSource)
	})

	g.It("Author:aramesha-High-64156-Verify IPFIX-exporter [Serial]", func() {
		SkipIfOCPBelow(4, 10)
		clusterArch, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("nodes", "-o=jsonpath={.items[0].status.nodeInfo.architecture}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if !strings.Contains(clusterArch, "amd64") {
			g.Skip("IPFIX collector image only supports amd64 architecture. Skip this test!")
		}

		g.By("Create IPFIX namespace")
		ipfixCollectorTemplatePath := filePath.Join(baseDir, "exporters", "ipfix-collector.yaml")
		IPFIXns := "ipfix"
		defer oc.DeleteSpecifiedNamespaceAsAdmin(IPFIXns)
		oc.CreateSpecifiedNamespaceAsAdmin(IPFIXns)
		_ = compat_otp.SetNamespacePrivileged(oc, IPFIXns)

		g.By("Deploy IPFIX collector")
		createResourceFromFile(oc, IPFIXns, ipfixCollectorTemplatePath)
		WaitForPodsReadyWithLabel(oc, IPFIXns, "app=ipfix-collector")

		g.By("Wait for IPFIX collector TCP listener to initialize")
		time.Sleep(10 * time.Second)

		IPFIXconfig := map[string]interface{}{
			"ipfix": map[string]interface{}{
				"targetHost":   "ipfix-collector.ipfix.svc.cluster.local",
				"targetPort":   2055,
				"transport":    "TCP",
				"enterpriseID": 0},
			"type": "IPFIX",
		}

		config, err := json.Marshal(IPFIXconfig)
		o.Expect(err).ToNot(o.HaveOccurred())
		IPFIXexporter := string(config)
		additionalNamespaces := fmt.Sprintf("\"%s\"", IPFIXns)
		samplingValue := 3

		g.By("Deploy FlowCollector with IPFIX exporter and sampling")
		flow := Flowcollector{
			Namespace:                         namespace,
			Template:                          flowFixturePath,
			LokiEnable:                        "false",
			LokiNamespace:                     namespace,
			Exporters:                         []string{IPFIXexporter},
			NetworkPolicyAdditionalNamespaces: []string{additionalNamespaces},
			Sampling:                          strconv.Itoa(samplingValue),
		}

		defer func() { _ = flow.DeleteFlowcollector(oc) }()
		flow.CreateFlowcollector(oc)

		g.By("Verify flowcollector is deployed with IPFIX exporter")
		flowPatch, err := oc.AsAdmin().Run("get").Args("flowcollector", "cluster", "-n", namespace, "-o", "jsonpath='{.spec.exporters[0].type}'").Output()
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(flowPatch).To(o.Equal(`'IPFIX'`))

		g.By("Get IPFIX collector pod")
		collectorPod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", IPFIXns, "-l", "app=ipfix-collector", "-o=jsonpath={.items[0].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Wait for IPFIX flows to be collected")
		time.Sleep(60 * time.Second)

		g.By("Retrieve and parse IPFIX flow records from collector API")
		flowRecords, err := getIPFIXFlowRecordsFromAPI(oc, IPFIXns, collectorPod)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(flowRecords)).Should(o.BeNumerically(">", 0), "No IPFIX flow records found in collector")

		g.By("Verify all IPFIX fields are present and valid")
		for _, record := range flowRecords {
			record.Flowlog.verifyIPFIXFields()
		}

		g.By("Verify sampling value matches FlowCollector configuration (NETOBSERV-2706)")
		for _, record := range flowRecords {
			o.Expect(record.Flowlog.Sampling).Should(o.BeNumerically("==", samplingValue),
				fmt.Sprintf("Expected Sampling=%d, got %d", samplingValue, record.Flowlog.Sampling))
		}
	})

	g.It("Author:memodi-High-74977-Verify OTEL exporter with TLS [Serial]", func() {
		// don't delete the OTEL Operator at the end of the test
		g.By("Subscribe to OTEL Operator")
		OtelNS.DeployOperatorNamespace(oc)
		OTEL.SubscribeOperator(oc)
		WaitForPodsReadyWithLabel(oc, OTEL.Namespace, "app.kubernetes.io/name="+OTEL.OperatorName)
		OTELStatus, err := CheckOperatorStatus(oc, OTEL.Namespace, OTEL.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect((OTELStatus)).To(o.BeTrue())

		g.By("Create OTEL Collector with TLS enabled")
		otelCollectorTemplatePath := filePath.Join(baseDir, "exporters", "otel-collector-tls.yaml")
		otlpEndpoint := 4317
		promEndpoint := "8889"
		collectorname := "otel"
		compat_otp.ApplyNsResourceFromTemplate(oc, namespace, "-f", otelCollectorTemplatePath, "-p", "NAME="+collectorname, "OTLP_GRPC_ENDPOINT="+strconv.Itoa(otlpEndpoint), "OTLP_PROM_PORT="+promEndpoint)
		otelPodLabel := "app.kubernetes.io/component=opentelemetry-collector"
		defer func() {
			oc.AsAdmin().WithoutNamespace().Run("delete").Args("opentelemetrycollector", collectorname, "-n", namespace).Execute()
			oc.AsAdmin().WithoutNamespace().Run("delete").Args("service", collectorname+"-collector", "-n", namespace).Execute()
			oc.AsAdmin().WithoutNamespace().Run("delete").Args("configmap", "service-ca", "-n", namespace).Execute()
		}()
		WaitForPodsReadyWithLabel(oc, namespace, otelPodLabel)

		g.By("Wait for service-ca configmap to be injected with CA bundle")
		waitForConfigMapDataInjection(oc, namespace, "service-ca", "service-ca.crt")

		targetHost := fmt.Sprintf("%s-collector.%s.svc", collectorname, namespace)
		otel_config := map[string]interface{}{
			"openTelemetry": map[string]interface{}{
				"logs": map[string]bool{"enable": true},
				"metrics": map[string]interface{}{"enable": true,
					"pushTimeInterval": "20s"},
				"targetHost": targetHost,
				"targetPort": otlpEndpoint,
				"protocol":   "grpc",
				"tls": map[string]interface{}{
					"enable":             true,
					"insecureSkipVerify": false,
					"caCert": map[string]interface{}{
						"type":     "configmap",
						"name":     "service-ca",
						"certFile": "service-ca.crt",
					},
				},
			},
			"type": "OpenTelemetry",
		}
		config, err := json.Marshal(otel_config)
		o.Expect(err).NotTo(o.HaveOccurred())
		config_str := string(config)

		g.By("Deploy FlowCollector with OTEL TLS exporter and Loki disabled")
		flow := Flowcollector{
			Namespace:     namespace,
			Template:      flowFixturePath,
			LokiEnable:    "false",
			LokiNamespace: namespace,
			Exporters:     []string{config_str},
		}

		defer flow.DeleteFlowcollector(oc)
		flow.CreateFlowcollector(oc)

		g.By("Verify OTEL collector is receiving TLS-encrypted flows")
		otelCollectorPod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", namespace, "-l", otelPodLabel, "-o=jsonpath={.items[0].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		// wait for 60 seconds to ensure we collected enough logs to grep from
		time.Sleep(60 * time.Second)

		g.By("Verify OTEL flowlogs are seen in collector pod logs")
		textToExist := "Attributes:"
		textToNotExist := "INVALID"

		podLogs, err := getPodLogs(oc, namespace, otelCollectorPod)
		o.Expect(err).ToNot(o.HaveOccurred())

		grepCmd := fmt.Sprintf("grep %s %s", textToExist, podLogs)
		textToExistLogs, err := exec.Command("bash", "-c", grepCmd).Output()

		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(len(textToExistLogs)).To(o.BeNumerically(">", 0))

		grepCmd = fmt.Sprintf("grep %s %s || true", textToNotExist, podLogs)
		textToNotExistLogs, err := exec.Command("bash", "-c", grepCmd).Output()
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(len(textToNotExistLogs)).To(o.BeNumerically("==", 0), string(textToNotExistLogs))

		g.By("Verify OTEL prometheus has metrics")
		// Get the service IP for the service with label operator.opentelemetry.io/collector-service-type=base
		svcIP, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("svc", "-n", namespace, "-l", "operator.opentelemetry.io/collector-service-type=base", "-o=jsonpath={.items[0].spec.clusterIP}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Get one of the flowlogs-pipeline pods
		flowlogsPipelinePod, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", namespace, "-l", "app=flowlogs-pipeline", "-o=jsonpath={.items[0].metadata.name}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())

		// Use the flowlogs-pipeline pod to curl the metrics endpoint of the otel collector service
		command := fmt.Sprintf("curl -s http://%s:%s/metrics | grep 'netobserv_workload_flows_total{' | head -1 | awk '{print $2}'", svcIP, promEndpoint)
		cmd := []string{"-n", namespace, flowlogsPipelinePod, "--", "/bin/sh", "-c", command}
		count, err := oc.AsAdmin().WithoutNamespace().Run("exec").Args(cmd...).Output()
		o.Expect(err).ToNot(o.HaveOccurred())
		nCount, err := strconv.Atoi(strings.Trim(count, "\n"))
		o.Expect(err).ToNot(o.HaveOccurred())
		o.Expect(nCount).To(o.BeNumerically(">", 0))
	})
})
