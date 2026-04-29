package e2etests

import (
	"context"
	"fmt"
	filePath "path/filepath"
	"regexp"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	netobservNS   = "openshift-netobserv-operator"
	NOPackageName = "netobserv-operator"
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

type subscriptionResource struct {
	name             string
	namespace        string
	operatorName     string
	channel          string
	catalog          string
	catalogNamespace string
	template         string
}

type operatorGroupResource struct {
	name             string
	namespace        string
	targetNamespaces string
	template         string
}

// waitForPackagemanifestAppear waits for the packagemanifest to appear in the cluster
// chSource: bool value, true means the packagemanifests' source name must match the so.CatalogSource.SourceName, e.g.: oc get packagemanifests xxxx -l catalog=$source-name
func (so *SubscriptionObjects) waitForPackagemanifestAppear(oc *exutil.CLI, chSource bool) {
	args := []string{"-n", so.CatalogSource.SourceNamespace, "packagemanifests"}
	if chSource {
		args = append(args, "-l", "catalog="+so.CatalogSource.SourceName)
	} else {
		args = append(args, so.PackageName)
	}
	err := wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (done bool, err error) {
		packages, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(args...).Output()
		if err != nil {
			msg := fmt.Sprintf("%v", err)
			if strings.Contains(msg, "No resources found") || strings.Contains(msg, "NotFound") {
				return false, nil
			}
			return false, err
		}
		if strings.Contains(packages, so.PackageName) {
			return true, nil
		}
		e2e.Logf("Waiting for packagemanifest/%s to appear", so.PackageName)
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Packagemanifest %s is not availabile", so.PackageName))
}

// setCatalogSourceObjects set the default values of channel, source namespace and source name if they're not specified
func (so *SubscriptionObjects) setCatalogSourceObjects(oc *exutil.CLI) {
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
		so.waitForPackagemanifestAppear(oc, true)
	} else {
		catsrc, _ := oc.AsAdmin().WithoutNamespace().Run("get").Args("catsrc", "-n", so.CatalogSource.SourceNamespace, "qe-app-registry").Output()
		if catsrc != "" && !(strings.Contains(catsrc, "NotFound")) {
			so.CatalogSource.SourceName = "qe-app-registry"
			so.waitForPackagemanifestAppear(oc, true)
		} else {
			so.waitForPackagemanifestAppear(oc, false)
			source, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("packagemanifests", so.PackageName, "-o", "jsonpath={.status.catalogSource}").Output()
			if err != nil {
				e2e.Logf("error getting catalog source name: %v", err)
			}
			so.CatalogSource.SourceName = source
		}
	}
}

// SubscribeOperator is used to subcribe the CLO and EO
func (so *SubscriptionObjects) SubscribeOperator(oc *exutil.CLI) {
	// check if the namespace exists, if it doesn't exist, create the namespace
	_, err := oc.AdminKubeClient().CoreV1().Namespaces().Get(context.Background(), so.Namespace, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			e2e.Logf("The project %s is not found, create it now...", so.Namespace)
			namespaceTemplate, _ := filePath.Abs("testdata/logging/subscription/namespace.yaml")
			namespaceFile, err := processTemplate(oc, "-f", namespaceTemplate, "-p", "NAMESPACE_NAME="+so.Namespace)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
				output, err := oc.AsAdmin().Run("apply").Args("-f", namespaceFile).Output()
				if err != nil {
					if strings.Contains(output, "AlreadyExists") {
						return true, nil
					}
					return false, err
				}
				return true, nil
			})
			compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("can't create project %s", so.Namespace))
		}
	}

	// check the operator group, if no object found, then create an operator group in the project
	og, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("-n", so.Namespace, "og").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	msg := fmt.Sprintf("%v", og)
	if strings.Contains(msg, "No resources found") {
		// create operator group
		ogFile, err := processTemplate(oc, "-n", so.Namespace, "-f", so.OperatorGroup, "-p", "OG_NAME="+so.Namespace, "NAMESPACE="+so.Namespace)
		o.Expect(err).NotTo(o.HaveOccurred())
		err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
			output, err := oc.AsAdmin().Run("apply").Args("-f", ogFile, "-n", so.Namespace).Output()
			if err != nil {
				if strings.Contains(output, "AlreadyExists") {
					return true, nil
				}
				return false, err
			}
			return true, nil
		})
		compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("can't create operatorgroup %s in %s project", so.Namespace, so.Namespace))
	}

	// check subscription, if there is no subscription objets, then create one
	sub, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", "-n", so.Namespace, so.PackageName).Output()
	if err != nil {
		msg := fmt.Sprint("v%", sub)
		if strings.Contains(msg, "NotFound") {
			so.setCatalogSourceObjects(oc)
			// create subscription object
			subscriptionFile, err := processTemplate(oc, "-n", so.Namespace, "-f", so.Subscription, "-p", "PACKAGE_NAME="+so.PackageName, "NAMESPACE="+so.Namespace, "CHANNEL="+so.CatalogSource.Channel, "SOURCE="+so.CatalogSource.SourceName, "SOURCE_NAMESPACE="+so.CatalogSource.SourceNamespace)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 120*time.Second, false, func(context.Context) (done bool, err error) {
				output, err := oc.AsAdmin().Run("apply").Args("-f", subscriptionFile, "-n", so.Namespace).Output()
				if err != nil {
					if strings.Contains(output, "AlreadyExists") {
						return true, nil
					}
					return false, err
				}
				return true, nil
			})
			compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("can't create subscription %s in %s project", so.PackageName, so.Namespace))
		}
	}
	//WaitForDeploymentPodsToBeReady(oc, so.Namespace, so.OperatorName)
}

func deleteNamespace(oc *exutil.CLI, ns string) {
	err := oc.AdminKubeClient().CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = nil
		}
	}
	o.Expect(err).NotTo(o.HaveOccurred())
	err = wait.PollUntilContextTimeout(context.Background(), 5*time.Second, 180*time.Second, false, func(context.Context) (bool, error) {
		_, err := oc.AdminKubeClient().CoreV1().Namespaces().Get(context.Background(), ns, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("Namespace %s is not deleted in 3 minutes", ns))
}

func (so *SubscriptionObjects) uninstallOperator(oc *exutil.CLI) {
	_ = Resource{"subscription", so.PackageName, so.Namespace}.clear(oc)
	_ = oc.AsAdmin().WithoutNamespace().Run("delete").Args("-n", so.Namespace, "csv", "-l", "operators.coreos.com/"+so.PackageName+"."+so.Namespace+"=").Execute()
	// do not remove namespace openshift-logging and openshift-operators-redhat, and preserve the operatorgroup as there may have several operators deployed in one namespace
	// for example: loki-operator and elasticsearch-operator
	if so.Namespace != "openshift-logging" && so.Namespace != "openshift-operators-redhat" && so.Namespace != "openshift-operators" && so.Namespace != "openshift-netobserv-operator" && !strings.HasPrefix(so.Namespace, "e2e-test-") {
		deleteNamespace(oc, so.Namespace)
	}
}

func checkOperatorChannel(oc *exutil.CLI, operatorNamespace string, operatorName string) (string, error) {
	channelName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", operatorName, "-n", operatorNamespace, "-o=jsonpath={.spec.channel}").Output()
	if err != nil {
		return "", err
	}
	return channelName, nil
}

func CheckOperatorStatus(oc *exutil.CLI, operatorNamespace string, operatorName string) (bool, error) {
	err := oc.AsAdmin().WithoutNamespace().Run("get").Args("namespace", operatorNamespace).Execute()
	if err == nil {
		err1 := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", operatorName, "-n", operatorNamespace).Execute()
		if err1 == nil {
			csvName, err2 := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", operatorName, "-n", operatorNamespace, "-o=jsonpath={.status.installedCSV}").Output()
			o.Expect(err2).NotTo(o.HaveOccurred())
			o.Expect(csvName).NotTo(o.BeEmpty())
			err = wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (bool, error) {
				csvState, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("csv", csvName, "-n", operatorNamespace, "-o=jsonpath={.status.phase}").Output()
				if err != nil {
					return false, err
				}
				e2e.Logf("CSV %s state: %s", csvName, csvState)
				return csvState == "Succeeded", nil
			})
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}
	e2e.Logf("%s operator will be created by tests", operatorName)
	return false, nil
}

func (ns *OperatorNamespace) DeployOperatorNamespace(oc *exutil.CLI) {
	e2e.Logf("Creating %s operator namespace", ns.Name)
	nsParameters := []string{"--ignore-unknown-parameters=true", "-f", ns.NamespaceTemplate, "-p", "NAMESPACE_NAME=" + ns.Name}
	compat_otp.ApplyClusterResourceFromTemplate(oc, nsParameters...)
}

func generateTemplateAbsolutePath(fileName string) string {
	testDataDir, _ := filePath.Abs("testdata/networking/nmstate")
	return filePath.Join(testDataDir, fileName)
}

func operatorInstall(oc *exutil.CLI, sub subscriptionResource, ns OperatorNamespace, og operatorGroupResource) (status bool) {
	//Installing Operator
	g.By("INSTALLING Operator in the namespace")

	//Applying the config of necessary yaml files from templates to create metallb operator
	g.By("Applying namespace template")
	err0 := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", ns.NamespaceTemplate, "-p", "NAME="+ns.Name)
	if err0 != nil {
		e2e.Logf("Error creating namespace %v", err0)
	}

	g.By("Applying operatorgroup yaml")
	err0 = applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", og.template, "-p", "NAME="+og.name, "NAMESPACE="+og.namespace, "TARGETNAMESPACES="+og.targetNamespaces)
	if err0 != nil {
		e2e.Logf("Error creating operator group %v", err0)
	}

	g.By("Creating subscription YAML from template")
	// no need to check for an existing subscription
	err0 = applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", sub.template, "-p", "OPERATORNAME="+sub.operatorName, "SUBSCRIPTIONNAME="+sub.name, "NAMESPACE="+sub.namespace, "CHANNEL="+sub.channel,
		"CATALOGSOURCE="+sub.catalog, "CATALOGSOURCENAMESPACE="+sub.catalogNamespace)
	if err0 != nil {
		e2e.Logf("Error creating subscription %v", err0)
	}

	//confirming operator install
	g.By("Verify the operator finished subscribing")
	errCheck := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (bool, error) {
		subState, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", sub.name, "-n", sub.namespace, "-o=jsonpath={.status.state}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if strings.Compare(subState, "AtLatestKnown") == 0 {
			return true, nil
		}
		// log full status of sub for installation failure debugging
		subState, _ = oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", sub.name, "-n", sub.namespace, "-o=jsonpath={.status}").Output()
		e2e.Logf("Status of subscription: %v", subState)
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(errCheck, fmt.Sprintf("Subscription %s in namespace %v does not have expected status", sub.name, sub.namespace))

	g.By("Get csvName")
	csvName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", sub.name, "-n", sub.namespace, "-o=jsonpath={.status.installedCSV}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(csvName).NotTo(o.BeEmpty())
	errCheck = wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 360*time.Second, false, func(context.Context) (bool, error) {
		csvState, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("csv", csvName, "-n", sub.namespace, "-o=jsonpath={.status.phase}").Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		if strings.Compare(csvState, "Succeeded") == 0 {
			e2e.Logf("CSV check complete!!!")
			return true, nil

		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(errCheck, fmt.Sprintf("CSV %v in %v namespace does not have expected status", csvName, sub.namespace))
	return true
}

func getOpenshiftVersion(oc *exutil.CLI) string {
	version, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion/version", "-ojsonpath={.status.desired.version}").Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`^(\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

func createImageDigestMirrorSet(oc *exutil.CLI, imagedigestmirrorsetname string, imageDigestMirrorSetFile string) error {
	pollInterval := 10 * time.Second
	waitTimeout := 120 * time.Second
	err := oc.AsAdmin().WithoutNamespace().Run("create").Args("-f", imageDigestMirrorSetFile).Execute()
	if err != nil {
		return fmt.Errorf("error applying image digest mirror set: %w", err)
	}
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, waitTimeout, false, func(_ context.Context) (bool, error) {
		err := oc.AsAdmin().WithoutNamespace().
			Run("get").Args("imagedigestmirrorset", imagedigestmirrorsetname).Execute()
		return err == nil, nil
	})
}

func createCatalogSource(oc *exutil.CLI, operatorName string, catalogSourceName string, catalogNamespace string, catalogSourceTemplateFile string) error {
	pollInterval := 10 * time.Second
	waitTimeout := 120 * time.Second
	openshiftVersion := getOpenshiftVersion(oc)
	if openshiftVersion == "" {
		return fmt.Errorf("failed to get OpenShift version")
	}
	image := "quay.io/redhat-user-workloads/ocp-art-tenant/art-fbc:ocp__" + openshiftVersion + "__" + operatorName + "-rhel9-operator"
	e2e.Logf("Creating catalog source with name  '%s' in namespace '%s'", catalogSourceName, catalogNamespace)
	err := applyResourceFromTemplateByAdmin(oc, "--ignore-unknown-parameters=true", "-f", catalogSourceTemplateFile, "-p", "CATALOGSOURCENAME="+catalogSourceName, "CATALOGNAMESPACE="+catalogNamespace, "IMAGE="+image)
	if err != nil {
		return fmt.Errorf("error applying catalog source: %w", err)
	}

	// Wait for CatalogSource to exist and be ready
	return wait.PollUntilContextTimeout(context.Background(), pollInterval, waitTimeout, false, func(_ context.Context) (bool, error) {
		// Check if CatalogSource exists
		err := oc.AsAdmin().WithoutNamespace().Run("get").Args("catalogsource", catalogSourceName, "-n", catalogNamespace).Execute()
		if err != nil {
			e2e.Logf("CatalogSource not found yet: %v", err)
			return false, nil
		}

		// Check if CatalogSource connection state is READY
		connectionState, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("catalogsource", catalogSourceName,
			"-n", catalogNamespace, "-o=jsonpath={.status.connectionState.lastObservedState}").Output()
		if err != nil {
			e2e.Logf("Failed to get connection state: %v", err)
			return false, nil
		}

		if string(connectionState) != "READY" {
			e2e.Logf("CatalogSource connection state is '%s', waiting for 'READY'", string(connectionState))
			return false, nil
		}

		// Check if registry pod is running and ready
		podName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pods", "-n", catalogNamespace,
			"-l", "olm.catalogSource="+catalogSourceName,
			"-o=jsonpath={.items[0].metadata.name}").Output()
		if err != nil || len(podName) == 0 {
			e2e.Logf("Registry pod not found yet: %v", err)
			return false, nil
		}

		// Check pod ready condition
		podReady, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("pod", string(podName), "-n", catalogNamespace,
			"-o=jsonpath={.status.conditions[?(@.type=='Ready')].status}").Output()
		if err != nil {
			e2e.Logf("Failed to get pod ready status: %v", err)
			return false, nil
		}

		if string(podReady) != "True" {
			e2e.Logf("Registry pod '%s' is not ready yet: %s", string(podName), string(podReady))
			return false, nil
		}
		e2e.Logf("CatalogSource '%s' is ready with pod '%s'", catalogSourceName, string(podName))
		return true, nil
	})
}

func getOperatorCatalogSource(oc *exutil.CLI, catalog string, namespace string) string {
	if isBaselineCapsSet(oc) && !(isEnabledCapability(oc, "OperatorLifecycleManager")) {
		g.Skip("Skipping the test as baselinecaps have been set and OperatorLifecycleManager capability is not enabled!")
	}
	catalogSourceNames, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("catalogsource", "-n", namespace, "-o=jsonpath={.items[*].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	if strings.Contains(catalogSourceNames, catalog) {
		return catalog
	}
	return ""
}

func getImageDigestMirrorSet(oc *exutil.CLI, imagedigestmirrorsetname string) string {
	imageDigestMirrorSetNames, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("imagedigestmirrorset", "-o=jsonpath={.items[*].metadata.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	if strings.Contains(imageDigestMirrorSetNames, imagedigestmirrorsetname) {
		return imagedigestmirrorsetname
	}
	return ""
}

func installNMstateOperator(oc *exutil.CLI) {
	var (
		opNamespace              = "openshift-nmstate"
		opName                   = "kubernetes-nmstate-operator"
		catalogNamespace         = "openshift-marketplace"
		catalogSourceName        = "kubernetes-nmstate-operator-fbc-catalog"
		imageDigestMirrorSetName = "kubernetes-nmstate-images-mirror-set"
	)

	e2e.Logf("Check catalogsource and install nmstate operator.")
	namespaceTemplate := generateTemplateAbsolutePath("namespace-template.yaml")
	operatorGroupTemplate := generateTemplateAbsolutePath("operatorgroup-template.yaml")
	subscriptionTemplate := generateTemplateAbsolutePath("subscription-template.yaml")
	catalogSourceTemplate := generateTemplateAbsolutePath("catalogsource-template.yaml")
	imageDigestMirrorSetFile := generateTemplateAbsolutePath("image-digest-mirrorset.yaml")
	sub := subscriptionResource{
		name:             "nmstate-operator-sub",
		namespace:        opNamespace,
		operatorName:     opName,
		channel:          "stable",
		catalog:          catalogSourceName,
		catalogNamespace: catalogNamespace,
		template:         subscriptionTemplate,
	}
	compat_otp.By("Check the image digest mirror set and catalog source")
	imageDigestMirrorSet := getImageDigestMirrorSet(oc, imageDigestMirrorSetName)
	if imageDigestMirrorSet == "" {
		compat_otp.By("Creating image digest mirror set")
		o.Expect(createImageDigestMirrorSet(oc, imageDigestMirrorSetName, imageDigestMirrorSetFile)).NotTo(o.HaveOccurred())
	}
	catalogSource := getOperatorCatalogSource(oc, catalogSourceName, catalogNamespace)
	if catalogSource == "" {
		compat_otp.By("Creating catalog source")
		o.Expect(createCatalogSource(oc, "kubernetes-nmstate", catalogSourceName, catalogNamespace, catalogSourceTemplate)).NotTo(o.HaveOccurred())
	}
	//sub.catalog = catalogSource
	ns := OperatorNamespace{
		Name:              opNamespace,
		NamespaceTemplate: namespaceTemplate,
	}
	og := operatorGroupResource{
		name:             opName,
		namespace:        opNamespace,
		targetNamespaces: opNamespace,
		template:         operatorGroupTemplate,
	}

	operatorInstall(oc, sub, ns, og)
	e2e.Logf("SUCCESS - NMState operator installed")
}

// setupCatalogSource deploys the catalog source and image digest mirror set
func setupCatalogSource(oc *exutil.CLI, catSrc Resource, catSrcTemplate, imageDigest, catalogSource string, isHypershift bool, NOSource *CatalogSourceObjects, NO *SubscriptionObjects) (bool, error) {
	g.By("Deploy konflux FBC and ImageDigestMirrorSet")
	upstreamCatalogSource := "quay.io/netobserv/network-observability-operator-catalog:v0.0.0-sha-main"
	deployedUpstreamCatalogSource := false
	var catsrcErr error

	if catalogSource != "" {
		e2e.Logf("Using %s catalog", catalogSource)
		catsrcErr = catSrc.applyFromTemplate(oc, "-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace, "IMAGE="+catalogSource)
	} else if isHypershift {
		e2e.Logf("Using v0.0.0-sha-main catalog for hypershift")
		catsrcErr = catSrc.applyFromTemplate(oc, "-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace, "IMAGE="+upstreamCatalogSource)
		NOSource.Channel = "latest"
		NO.CatalogSource = NOSource
		deployedUpstreamCatalogSource = true
	} else {
		e2e.Logf("Using default ystream catalog")
		catsrcErr = catSrc.applyFromTemplate(oc, "-n", catSrc.Namespace, "-f", catSrcTemplate, "-p", "NAMESPACE="+catSrc.Namespace)
	}
	catSrc.WaitUntilCatSrcReady(oc)

	if !isHypershift {
		ApplyResourceFromFile(oc, catSrc.Namespace, imageDigest)
	}
	return deployedUpstreamCatalogSource, catsrcErr
}

// ensureOperatorDeployed checks and deploys an operator if not already present
func ensureOperatorDeployed(oc *exutil.CLI, operator SubscriptionObjects, operatorSource CatalogSourceObjects, podLabel string) {
	g.By(fmt.Sprintf("Subscribe %s operator to %s channel", operator.OperatorName, operatorSource.Channel))
	operatorExisting, err := CheckOperatorStatus(oc, operator.Namespace, operator.PackageName)
	o.Expect(err).NotTo(o.HaveOccurred())

	if !operatorExisting {
		e2e.Logf("%s operator not found, subscribing to operator", operator.OperatorName)
		operator.SubscribeOperator(oc)

		// Wait for operator pods to be ready
		if podLabel != "" {
			WaitForPodsReadyWithLabel(oc, operator.Namespace, podLabel)
		}

		// Verify operator status
		operatorStatus, err := CheckOperatorStatus(oc, operator.Namespace, operator.PackageName)
		o.Expect(err).NotTo(o.HaveOccurred())
		o.Expect(operatorStatus).To(o.BeTrue())

		e2e.Logf("%s operator deployed successfully", operator.OperatorName)
	} else {
		e2e.Logf("%s operator already exists, skipping deployment", operator.OperatorName)
	}
}

// ensureNetObservOperatorDeployed checks and deploys the NetObserv operator with specific configurations
func ensureNetObservOperatorDeployed(oc *exutil.CLI, NO SubscriptionObjects, NOSource CatalogSourceObjects, deployedUpstreamCatalogSource bool) {
	ensureOperatorDeployed(oc, NO, NOSource, "app="+NO.OperatorName)

	// NetObserv-specific checks only if operator was just deployed
	NOexisting, err := CheckOperatorStatus(oc, NO.Namespace, NO.PackageName)
	o.Expect(err).NotTo(o.HaveOccurred())

	if NOexisting {
		// Verify FlowCollector API exists
		flowcollectorAPIExists, err := isFlowCollectorAPIExists(oc)
		o.Expect(flowcollectorAPIExists).To(o.BeTrue())
		o.Expect(err).NotTo(o.HaveOccurred())

		// Patch upstream catalog source if needed
		if deployedUpstreamCatalogSource {
			_, err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("csv", "netobserv-operator.v0.0.0-sha-main", "-n", NO.Namespace,
				"--type=json", "--patch", "[{\"op\": \"replace\",\"path\": \"/spec/install/spec/deployments/0/spec/template/spec/containers/0/env/4/value\", \"value\": \"true\"}]").Output()
			o.Expect(err).NotTo(o.HaveOccurred())
		}
	}
}

func getOperatorChannel(oc *exutil.CLI, catalog string, packageName string) (operatorChannel string, err error) {
	channels, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("packagemanifests", "-l", "catalog="+catalog, "-n", "openshift-marketplace", "-o=jsonpath={.items[?(@.metadata.name==\""+packageName+"\")].status.channels[*].name}").Output()
	channelArr := strings.Split(channels, " ")
	return channelArr[len(channelArr)-1], err
}
