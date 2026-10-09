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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// SubscriptionObjects objects are used to create operators via OLM
type SubscriptionObjects struct {
	OperatorName     string
	Namespace        string
	OperatorGroup    string // the file used to create operator group
	Subscription     string // the file used to create subscription
	PackageName      string
	CatalogSource    *CatalogSourceObjects `json:",omitempty"`
	OperatorPodLabel string
}

// CatalogSourceObjects defines the source used to subscribe an operator
type CatalogSourceObjects struct {
	Channel         string `json:",omitempty"`
	SourceName      string `json:",omitempty"`
	SourceNamespace string `json:",omitempty"`
}

// OperatorNamespace struct to handle creation of namespace
type OperatorNamespace struct {
	Name              string
	NamespaceTemplate string
}

// waitForPackagemanifestAppear waits for the packagemanifest to appear in the cluster
// chSource: bool value, true means the packagemanifests' source name must match the so.CatalogSource.SourceName
func (so *SubscriptionObjects) waitForPackagemanifestAppear(chSource bool) {
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		if chSource {
			// List with label selector and check if package exists
			gvr, _ := resolveGVR("packagemanifest")
			list, listErr := k8sDynClient.Resource(gvr).List(context.Background(), metav1.ListOptions{
				LabelSelector: "catalog=" + so.CatalogSource.SourceName,
			})
			if listErr != nil {
				return false, nil
			}
			for _, item := range list.Items {
				if item.GetName() == so.PackageName {
					return true, nil
				}
			}
		} else {
			// Get specific package manifest
			_, getErr := getDynamicResource("packagemanifest", so.PackageName, "")
			if getErr != nil {
				e2e.Logf("Waiting for packagemanifest/%s to appear", so.PackageName)
				return false, nil
			}
			return true, nil
		}
		e2e.Logf("Waiting for packagemanifest/%s to appear", so.PackageName)
		return false, nil
	})
	assertWaitPollNoErr(err, fmt.Sprintf("Packagemanifest %s is not available", so.PackageName))
}

// setCatalogSourceObjects set the default values of channel, source namespace and source name if they're not specified
func (so *SubscriptionObjects) setCatalogSourceObjects() {
	// set channel
	if so.CatalogSource.Channel == "" {
		so.CatalogSource.Channel = "stable"
	}

	// set source namespace
	if so.CatalogSource.SourceNamespace == "" {
		so.CatalogSource.SourceNamespace = "openshift-marketplace"
	}

	// set source and check if the packagemanifest exists or not
	if so.CatalogSource.SourceName != "" {
		// Packagemanifests are only created for catalog sources in openshift-marketplace
		// For custom namespaces, skip packagemanifest check and verify catalog source directly
		if so.CatalogSource.SourceNamespace == "openshift-marketplace" {
			so.waitForPackagemanifestAppear(true)
		}
	} else {
		// Check if qe-app-registry catalog source exists
		_, err := getDynamicResource("catalogsource", "qe-app-registry", so.CatalogSource.SourceNamespace)
		if err == nil {
			so.CatalogSource.SourceName = "qe-app-registry"
			so.waitForPackagemanifestAppear(true)
		} else {
			so.waitForPackagemanifestAppear(false)
			// Get catalog source from package manifest
			obj, getErr := getDynamicResource("packagemanifest", so.PackageName, "")
			if getErr != nil {
				e2e.Logf("error getting catalog source name: %v", getErr)
			} else {
				catalogSource, _ := getNestedField(obj.Object, ".status.catalogSource")
				so.CatalogSource.SourceName = catalogSource
			}
		}
	}
}

// SubscribeOperator is used to subcribe the CLO and EO
func (so *SubscriptionObjects) SubscribeOperator() {
	ctx := context.Background()

	// check if the namespace exists, if it doesn't exist, create the namespace
	_, err := k8sClient.CoreV1().Namespaces().Get(context.Background(), so.Namespace, metav1.GetOptions{})
	if err != nil {
		e2e.Logf("The project %s is not found, create it now...", so.Namespace)
		namespaceTemplate, _ := filePath.Abs("testdata/logging/subscription/namespace.yaml")
		namespaceFile, err := processTemplate("", "-f", namespaceTemplate, "-p", "NAMESPACE_NAME="+so.Namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
			cmd := exec.Command("oc", "apply", "-f", namespaceFile)
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(string(output), "AlreadyExists") {
				return false, err
			}
			return true, nil
		})
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("can't create project %s", so.Namespace))
	}

	// check the operator group, if no object found, then create an operator group in the project
	gvr, _ := resolveGVR("operatorgroup")
	ogList, listErr := k8sDynClient.Resource(gvr).Namespace(so.Namespace).List(context.Background(), metav1.ListOptions{})
	o.Expect(listErr).NotTo(o.HaveOccurred())
	if len(ogList.Items) == 0 {
		// create operator group
		ogFile, err := processTemplate(so.Namespace, "-f", so.OperatorGroup, "-p", "OG_NAME="+so.Namespace, "NAMESPACE="+so.Namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
			cmd := exec.Command("oc", "apply", "-f", ogFile)
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(string(output), "AlreadyExists") {
				return false, err
			}
			return true, nil
		})
		assertWaitPollNoErr(err, fmt.Sprintf("can't create operatorgroup %s in %s project", so.Namespace, so.Namespace))
	}

	// check subscription, if there is no subscription objects, then create one
	_, err = getDynamicResource("subscription", so.PackageName, so.Namespace)
	if err != nil {
		so.setCatalogSourceObjects()
		// create subscription object
		subscriptionFile, err := processTemplate(so.Namespace, "-f", so.Subscription, "-p", "PACKAGE_NAME="+so.PackageName, "NAMESPACE="+so.Namespace, "CHANNEL="+so.CatalogSource.Channel, "SOURCE="+so.CatalogSource.SourceName, "SOURCE_NAMESPACE="+so.CatalogSource.SourceNamespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
			cmd := exec.Command("oc", "apply", "-f", subscriptionFile)
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(string(output), "AlreadyExists") {
				return false, err
			}
			return true, nil
		})
		o.Expect(err).NotTo(o.HaveOccurred(), fmt.Sprintf("can't create subscription %s in %s project", so.PackageName, so.Namespace))
	}
}

func (so *SubscriptionObjects) uninstallOperator() {
	// Delete subscription
	_ = Resource{"subscription", so.PackageName, so.Namespace}.clear()

	// Delete CSVs with label
	labelSelector := "operators.coreos.com/" + so.PackageName + "." + so.Namespace + "="
	csvGVR, _ := resolveGVR("csv")
	_ = k8sDynClient.Resource(csvGVR).Namespace(so.Namespace).DeleteCollection(
		context.Background(),
		metav1.DeleteOptions{},
		metav1.ListOptions{LabelSelector: labelSelector},
	)

	// do not remove namespace openshift-logging and openshift-operators-redhat, and preserve the operatorgroup as there may have several operators deployed in one namespace
	// for example: loki-operator and elasticsearch-operator
	if so.Namespace != "openshift-logging" && so.Namespace != "openshift-operators-redhat" && so.Namespace != "openshift-operators" && so.Namespace != "openshift-netobserv-operator" && !strings.HasPrefix(so.Namespace, "e2e-test-") {
		deleteNamespace(so.Namespace)
	}
}

func checkOperatorChannel(operatorNamespace string, operatorName string) (string, error) {
	obj, err := getDynamicResource("subscription", operatorName, operatorNamespace)
	if err != nil {
		return "", err
	}
	channel, _ := getNestedField(obj.Object, ".spec.channel")
	return channel, nil
}

func CheckOperatorStatus(operatorNamespace string, operatorName string) (bool, error) {
	_, err := k8sClient.CoreV1().Namespaces().Get(context.Background(), operatorNamespace, metav1.GetOptions{})
	if err != nil {
		e2e.Logf("%s operator will be created by tests", operatorName)
		return false, nil
	}

	csvName := ""
	obj, err := getDynamicResource("subscription", operatorName, operatorNamespace)
	if err == nil {
		csvName, _ = getNestedField(obj.Object, ".status.installedCSV")
	}

	if csvName == "" {
		// Try listing subscriptions and find by spec.name
		gvr, _ := resolveGVR("subscription")
		list, listErr := k8sDynClient.Resource(gvr).Namespace(operatorNamespace).List(context.Background(), metav1.ListOptions{})
		if listErr == nil {
			for _, item := range list.Items {
				specName, _ := getNestedField(item.Object, ".spec.name")
				if specName == operatorName {
					csvName, _ = getNestedField(item.Object, ".status.installedCSV")
					break
				}
			}
		}
	}

	if csvName == "" {
		e2e.Logf("%s operator will be created by tests", operatorName)
		return false, nil
	}

	e2e.Logf("Found CSV %s for operator %s", csvName, operatorName)
	err = wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (bool, error) {
		csvObj, getErr := getDynamicResource("csv", csvName, operatorNamespace)
		if getErr != nil {
			return false, getErr
		}
		csvState, _ := getNestedField(csvObj.Object, ".status.phase")
		e2e.Logf("CSV %s state: %s", csvName, csvState)
		return csvState == "Succeeded", nil
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (ns *OperatorNamespace) DeployOperatorNamespace() {
	e2e.Logf("Creating %s operator namespace", ns.Name)
	nsParameters := []string{"--ignore-unknown-parameters=true", "-f", ns.NamespaceTemplate, "-p", "NAMESPACE_NAME=" + ns.Name}
	configFile, err := processTemplate("", nsParameters...)
	o.Expect(err).NotTo(o.HaveOccurred())
	cmd := exec.Command("oc", "apply", "-f", configFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		o.Expect(err).NotTo(o.HaveOccurred(), string(output))
	}
}

// setupCatalogSource deploys the catalog source and image digest mirror set
func setupCatalogSource(catSrc Resource, catSrcTemplate, imageDigest, catalogSource string, isHypershift bool, NOSource *CatalogSourceObjects, NO *SubscriptionObjects) error {
	g.By("Deploy konflux FBC and ImageDigestMirrorSet")
	upstreamCatalogSource := "quay.io/netobserv/network-observability-operator-catalog:v0.0.0-sha-main"
	var catsrcErr error

	if catalogSource != "" {
		e2e.Logf("Using %s catalog", catalogSource)
		catsrcErr = catSrc.applyFromTemplate("-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace, "IMAGE="+catalogSource)
	} else if isHypershift {
		e2e.Logf("Using v0.0.0-sha-main catalog for hypershift")
		catsrcErr = catSrc.applyFromTemplate("-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace, "IMAGE="+upstreamCatalogSource)
		NOSource.Channel = "latest"
		NO.CatalogSource = NOSource
	} else {
		e2e.Logf("Using catalog image from template defaults")
		catsrcErr = catSrc.applyFromTemplate("-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace)
	}
	catSrc.WaitUntilCatSrcReady()

	if !isHypershift {
		ApplyResourceFromFile(catSrc.Namespace, imageDigest)
	}
	return catsrcErr
}

// ensureOperatorDeployed checks and deploys an operator if not already present
func ensureOperatorDeployed(operator SubscriptionObjects, operatorSource CatalogSourceObjects, podLabel string) {
	g.By(fmt.Sprintf("Subscribe %s operator to %s channel", operator.OperatorName, operatorSource.Channel))
	operatorExisting, err := CheckOperatorStatus(operator.Namespace, operator.PackageName)
	o.Expect(err).NotTo(o.HaveOccurred())

	if !operatorExisting {
		e2e.Logf("%s operator not found, subscribing to operator", operator.OperatorName)
		operator.SubscribeOperator()

		// Wait for operator pods to be ready
		if podLabel != "" {
			WaitForPodsReadyWithLabel(operator.Namespace, podLabel)
		}

		// Verify operator status
		operatorStatus, err := CheckOperatorStatus(operator.Namespace, operator.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(operatorStatus).To(o.BeTrue())

		e2e.Logf("%s operator deployed successfully", operator.OperatorName)
	} else {
		e2e.Logf("%s operator already exists, skipping deployment", operator.OperatorName)
	}
}

// ensureNetObservOperatorDeployed checks and deploys the NetObserv operator with specific configurations
func ensureNetObservOperatorDeployed(NO SubscriptionObjects, NOSource CatalogSourceObjects) {
	ensureOperatorDeployed(NO, NOSource, "app="+NO.OperatorName)

	// NetObserv-specific checks only if operator was just deployed
	NOexisting, err := CheckOperatorStatus(NO.Namespace, NO.PackageName)
	o.Expect(err).NotTo(o.HaveOccurred())

	if NOexisting {
		// Verify FlowCollector API exists
		flowcollectorAPIExists, err := isFlowCollectorAPIExists()
		o.Expect(flowcollectorAPIExists).To(o.BeTrue())
		o.Expect(err).NotTo(o.HaveOccurred())
	}
}

func getOperatorChannel(catalog string, packageName string) (operatorChannel string, err error) {
	gvr, gvrErr := resolveGVR("packagemanifest")
	if gvrErr != nil {
		return "", gvrErr
	}
	list, listErr := k8sDynClient.Resource(gvr).Namespace("openshift-marketplace").List(context.Background(), metav1.ListOptions{
		LabelSelector: "catalog=" + catalog,
	})
	if listErr != nil {
		return "", listErr
	}
	for _, item := range list.Items {
		if item.GetName() == packageName {
			channels, found, _ := unstructured.NestedSlice(item.Object, "status", "channels")
			if !found || len(channels) == 0 {
				return "", fmt.Errorf("no channels found for package %s in catalog %s", packageName, catalog)
			}
			var channelNames []string
			for _, ch := range channels {
				chMap, ok := ch.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := chMap["name"].(string)
				if name != "" {
					channelNames = append(channelNames, name)
				}
			}
			if len(channelNames) == 0 {
				return "", fmt.Errorf("no channels found for package %s in catalog %s", packageName, catalog)
			}
			return channelNames[len(channelNames)-1], nil
		}
	}
	return "", fmt.Errorf("no channels found for package %s in catalog %s", packageName, catalog)
}

// getCSVVersion returns the version of the installed NetObserv Operator CSV in the given namespace.
// It uses the dynamic client to list CSVs and finds the one whose name starts with the operator name prefix.
func getCSVVersion(operatorNamespace string) (string, error) {
	csvGVR, _ := resolveGVR("csv")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	csvList, err := k8sDynClient.Resource(csvGVR).Namespace(operatorNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list CSVs in namespace %s: %v", operatorNamespace, err)
	}

	var matched []string
	for _, csv := range csvList.Items {
		if !strings.HasPrefix(csv.GetName(), NOPackageName+".") {
			continue
		}
		// Only consider the active CSV (phase "Succeeded"); during upgrades,
		// replaced CSVs have phase "Replacing" or "Deleting".
		phase, _, _ := unstructured.NestedString(csv.Object, "status", "phase")
		if phase != "Succeeded" {
			continue
		}
		version, found, err := unstructured.NestedString(csv.Object, "spec", "version")
		if err != nil || !found {
			return "", fmt.Errorf("version field not found in CSV %s", csv.GetName())
		}
		matched = append(matched, version)
	}
	switch len(matched) {
	case 0:
		return "", fmt.Errorf("no active CSV found for %s in namespace %s", NOPackageName, operatorNamespace)
	case 1:
		return matched[0], nil
	default:
		return "", fmt.Errorf("multiple active CSVs found for %s in namespace %s: %v", NOPackageName, operatorNamespace, matched)
	}
}
