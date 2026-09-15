package e2etests

import (
	"fmt"
	"time"

	filePath "path/filepath"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
)

var _ = g.Describe("[sig-netobserv] Network_Observability", func() {

	defer g.GinkgoRecover()
	var (
		namespace            string
		frrConfigurationPath = filePath.Join(baseDir, "bgp", frrConfigurationFixture)
	)

	g.BeforeEach(func() {
		oc := NewCLI()
		namespace = oc.Namespace()
	})

	g.It("Author:ljira-NonPreRelease-High-Verify BGP ASN enrichment from FRRConfiguration [Serial]", func() {
		g.By("Ensure FRRConfiguration CRD is available")
		err := ensureFRRConfigurationAPI()
		o.Expect(err).NotTo(o.HaveOccurred())

		g.By("Deploy fake FRRConfiguration with advertised prefixes")
		frrConfig := FRRConfiguration{
			Name:           "bgp-asn-enrichment-e2e",
			Namespace:      namespace,
			ASN:            bgpTestASN,
			ExternalPrefix: bgpExternalTarget + "/32",
			InternalPrefix: bgpInternalTarget + "/24",
			Template:       frrConfigurationPath,
		}
		defer func() { _ = frrConfig.DeleteFRRConfiguration() }()
		frrConfig.CreateFRRConfiguration()

		g.By("Deploy FlowCollector with BGP ASN enrichment enabled")
		flow := Flowcollector{
			Namespace:         namespace,
			BgpEnrichment:     "true",
			MonolithicLokiURL: fmt.Sprintf("http://loki.%s.svc:3100/", namespace),
			Template:          flowFixturePath,
		}
		defer func() { _ = flow.DeleteFlowcollector() }()
		flow.CreateFlowcollector()

		g.By("Deploy test ping pods to generate traffic")
		pingPodsTemplate := filePath.Join(baseDir, "test-ping-pods_template.yaml")
		testPingPodsTemplate := TestPingPodsTemplate{
			ServerNS:    "test-ping-server-bgp-asn",
			ClientNS:    "test-ping-client-bgp-asn",
			PingTargets: bgpExternalTarget + " " + bgpInternalTarget,
			Template:    pingPodsTemplate,
		}
		defer deleteNamespace(testPingPodsTemplate.ClientNS)
		defer deleteNamespace(testPingPodsTemplate.ServerNS)
		err = testPingPodsTemplate.createPingPods()
		o.Expect(err).NotTo(o.HaveOccurred())
		assertAllPodsToBeReady(testPingPodsTemplate.ServerNS)
		assertAllPodsToBeReady(testPingPodsTemplate.ClientNS)

		g.By("Wait for flows to be collected and enriched")
		startTime := time.Now()
		time.Sleep(90 * time.Second)

		lokilabels := Lokilabels{
			App:             "netobserv-flowcollector",
			SrcK8SNamespace: testPingPodsTemplate.ClientNS,
		}

		g.By("Verify external destination ASN enrichment")
		externalParams := []string{fmt.Sprintf("DstAddr=\"%s\"", bgpExternalTarget)}
		externalFlows, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, externalParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(externalFlows)).Should(o.BeNumerically(">", 0), "expected flows to external target %s", bgpExternalTarget)
		for _, r := range externalFlows {
			o.Expect(r.Flowlog.DstASN).To(o.Equal(bgpTestASN), "DstASN should match FRRConfiguration router ASN")
		}

		g.By("Verify internal destination ASN enrichment")
		internalParams := []string{fmt.Sprintf("DstAddr=\"%s\"", bgpInternalTarget)}
		internalFlows, err := lokilabels.GetMonolithicLokiFlowLogs(flow.MonolithicLokiURL, startTime, internalParams...)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(len(internalFlows)).Should(o.BeNumerically(">", 0), "expected flows to internal target %s", bgpInternalTarget)
		for _, r := range internalFlows {
			o.Expect(r.Flowlog.DstASN).To(o.Equal(bgpTestASN), "DstASN should match FRRConfiguration router ASN")
		}
	})
})
