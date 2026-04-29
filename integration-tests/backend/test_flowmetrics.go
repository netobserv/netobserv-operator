package e2etests

import (
	"os"
	filePath "path/filepath"
	"regexp"
	"strings"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	e2e "k8s.io/kubernetes/test/e2e/framework"
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
		flowmetricsPath = filePath.Join(baseDir, "flowmetrics_v1alpha1_template.yaml")

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

		flow Flowcollector
	)

	g.BeforeEach(func() {
		if strings.Contains(os.Getenv("E2E_RUN_TAGS"), "disconnected") {
			g.Skip("Skipping tests for disconnected profiles")
		}

		OperatorNS.DeployOperatorNamespace(oc)
		deployedUpstreamCatalogSource, catSrcErr := setupCatalogSource(oc, NOcatSrc, catSrcTemplate, imageDigest, catalogSource, false, &NOSource, &NO)
		o.Expect(catSrcErr).NotTo(o.HaveOccurred())
		ensureNetObservOperatorDeployed(oc, NO, NOSource, deployedUpstreamCatalogSource)

		// Create flowcollector in beforeEach
		flow = Flowcollector{
			Namespace:   oc.Namespace(),
			EBPFeatures: []string{"\"FlowRTT\""},
			LokiEnable:  "false",
			Template:    flowFixturePath,
		}
		flow.CreateFlowcollector(oc)
	})
	g.AfterEach(func() {
		_ = flow.DeleteFlowcollector(oc)
	})

	g.It("Author:memodi-High-73539-Create custom metrics and charts [Serial]", func() {
		SkipIfOCPBelow(4, 12)
		namespace := oc.Namespace()
		customMetrics := CustomMetrics{
			Namespace: namespace,
			Template:  flowmetricsPath,
		}

		mainDashversion, err := getResourceVersion(oc, "cm", "netobserv-main", "openshift-config-managed")
		o.Expect(err).NotTo(o.HaveOccurred())
		curv, err := getResourceVersion(oc, "cm", "flowlogs-pipeline-config-dynamic", namespace)
		o.Expect(err).NotTo(o.HaveOccurred())

		customMetrics.createCustomMetrics(oc)
		waitForResourceGenerationUpdate(oc, "cm", "flowlogs-pipeline-config-dynamic", "resourceVersion", curv, namespace)

		customMetricsConfig := customMetrics.getCustomMetricConfigs()
		var allUniqueDash = make(map[string]bool)
		var uniqueDashboards []string
		for _, cmc := range customMetricsConfig {
			for _, dashboard := range cmc.DashboardNames {
				if _, ok := allUniqueDash[dashboard]; !ok {
					allUniqueDash[dashboard] = true
					uniqueDashboards = append(uniqueDashboards, dashboard)
				}
			}
			// verify custom metrics queries
			for _, query := range cmc.Queries {
				metricsQuery := strings.Replace(query, "$METRIC", "netobserv_"+cmc.MetricName, 1)
				metricVal := pollMetrics(oc, metricsQuery)
				e2e.Logf("metricsQuery %f for query %s", metricVal, metricsQuery)
			}
		}
		// verify dashboard exists
		for _, uniqDash := range uniqueDashboards {
			dashName := strings.ToLower(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(uniqDash, "-"))
			if dashName == "main" {
				waitForResourceGenerationUpdate(oc, "cm", "netobserv-"+dashName, "resourceVersion", mainDashversion, "openshift-config-managed")
			}
			_, _ = checkResourceExists(oc, "cm", "netobserv-"+dashName, "openshift-config-managed")
		}
	})
})
