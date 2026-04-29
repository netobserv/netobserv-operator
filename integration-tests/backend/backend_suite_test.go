package e2etests

import (
	"flag"
	"fmt"
	"testing"

	g "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	e2eframework "k8s.io/kubernetes/test/e2e/framework"
)

func init() {
	// Initialize framework flags - must be done before flag.Parse()
	exutil.InitStandardFlags()
}

var _ = g.BeforeSuite(func() {
	// Parse flags
	flag.Parse()

	// Set up provider config after parsing flags
	e2eframework.AfterReadingAllFlags(exutil.TestContext)

	// Initialize test
	gomega.Expect(exutil.InitTest(false)).NotTo(gomega.HaveOccurred())

	oc := exutil.NewCLIForMonitorTest("netobserv")
	var err error
	clusterVersion, err = GetOCPVersion(oc)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	g.GinkgoWriter.Printf("Running tests against OCP %s\n", clusterVersion.String())
})

func TestBackend(t *testing.T) {
	exutil.WithCleanup(func() {
		gomega.RegisterFailHandler(g.Fail)

		suiteConfig, reporterConfig := g.GinkgoConfiguration()

		// Apply focus filter

		if len(suiteConfig.FocusStrings) > 0 {
			combinedFocus := make([]string, len(suiteConfig.FocusStrings))
			for i, userFocus := range suiteConfig.FocusStrings {
				combinedFocus[i] = "sig-netobserv.*" + userFocus
			}
			suiteConfig.FocusStrings = combinedFocus
		} else {
			suiteConfig.FocusStrings = []string{"sig-netobserv"}
		}

		// Configure reporter - suppress default verbose output
		suiteConfig.EmitSpecProgress = true
		suiteConfig.OutputInterceptorMode = "none"
		reporterConfig.SilenceSkips = true // Hide the "S" characters for skipped tests
		reporterConfig.NoColor = true
		reporterConfig.Succinct = true
		reporterConfig.Verbose = false

		// Standard Ginkgo run with custom reporting via hooks
		g.RunSpecs(t, "Backend Suite", suiteConfig, reporterConfig)
	})
}

// Custom reporting hooks
var _ = g.ReportBeforeSuite(func(report g.Report) {
	fmt.Printf("Running Suite: %s - %s\n", report.SuiteDescription, report.SuitePath)
	fmt.Printf("==========================================================================================================\n")
	fmt.Printf("Random Seed: %d\n\n", report.SuiteConfig.RandomSeed)
	fmt.Printf("Will run %d specs\n", report.PreRunStats.SpecsThatWillRun)
})

var _ = g.ReportAfterEach(func(report g.SpecReport) {
	// Only report on specs that actually ran (not skipped on via focused filter)
	if report.State == types.SpecStateSkipped && report.Failure.Message == "" && report.RunTime <= 0 {
		return
	}

	if report.LeafNodeType != types.NodeTypeIt {
		return
	}

	// Print spec progress
	fmt.Printf("%s\n", report.FullText())
	fmt.Printf("%s\n", report.LeafNodeLocation.String())

	// Print result
	switch report.State {
	case types.SpecStatePassed:
		fmt.Printf("• PASSED [%.3f seconds]\n", report.RunTime.Seconds())
	case types.SpecStateSkipped:
		fmt.Printf("• SKIPPED [%.3f seconds]\n", report.RunTime.Seconds())
		if report.Failure.Message != "" {
			fmt.Printf("\n%s\n", report.Failure.Message)
			fmt.Printf("%s\n", report.Failure.Location.String())
		}
	case types.SpecStateFailed, types.SpecStatePanicked, types.SpecStateInvalid, types.SpecStateAborted, types.SpecStateInterrupted, types.SpecStatePending, types.SpecStateTimedout:
		fmt.Printf("• FAILED [%.3f seconds]\n", report.RunTime.Seconds())
		if report.Failure.Message != "" {
			fmt.Printf("\n%s\n", report.Failure.Message)
			fmt.Printf("%s\n", report.Failure.Location.String())
		}
	}

	//   t,
	// types.SpecStateInvalid, types.SpecStatePending, types.SpecStateFailed, types.SpecStateAborted, types.SpecStatePanicked, types.SpecStateInterrupted, types.SpecStateTimedout
})

var _ = g.ReportAfterSuite("NetObserv Summary", func(report g.Report) {
	passed := 0
	failed := 0
	skipped := 0
	ranTests := 0

	// Get only the test specs (not setup/teardown)
	specs := report.SpecReports.WithLeafNodeType(types.NodeTypeIt)

	for _, specReport := range specs {
		switch specReport.State {
		case types.SpecStatePassed:
			passed++
			ranTests++
		case types.SpecStateFailed, types.SpecStatePanicked, types.SpecStateInvalid, types.SpecStateAborted, types.SpecStateInterrupted, types.SpecStatePending, types.SpecStateTimedout:
			failed++
			ranTests++
		case types.SpecStateSkipped:
			// Skip filtered-out specs: they have State==Skipped but RunTime==0 and no failure info
			// Explicitly skipped specs (via Skip()) have State==Skipped but were actually evaluated
			if specReport.Failure.Message != "" || specReport.RunTime > 0 {
				// Explicitly skipped - test body was evaluated
				skipped++
			}
		}
	}

	// Total specs evaluated (passed + failed + explicitly skipped)
	totalEvaluated := ranTests + skipped

	fmt.Printf("------------------------------\n")
	if report.SuiteSucceeded {
		fmt.Printf("\nBackend Suite - %d/%d specs • SUCCESS! [%.3f seconds]\n",
			totalEvaluated, totalEvaluated, report.RunTime.Seconds())
	} else {
		fmt.Printf("\nBackend Suite - %d/%d specs • FAILURE! [%.3f seconds]\n",
			totalEvaluated, totalEvaluated, report.RunTime.Seconds())
	}

	fmt.Printf("\nRan %d tests\n", ranTests)
	fmt.Printf("Passed: %d, Failed: %d, Skipped: %d\n", passed, failed, skipped)
})
