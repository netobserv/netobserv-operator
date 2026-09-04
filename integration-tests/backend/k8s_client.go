package e2etests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	routeclient "github.com/openshift/client-go/route/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	k8sClient     kubernetes.Interface
	k8sDynClient  dynamic.Interface
	k8sRestConfig *rest.Config
	routeV1Client routeclient.Interface
)

var gvrMap = map[string]schema.GroupVersionResource{
	// Standard K8s
	"pod":         {Version: "v1", Resource: "pods"},
	"cm":          {Version: "v1", Resource: "configmaps"},
	"namespace":   {Version: "v1", Resource: "namespaces"},
	"job":         {Group: "batch", Version: "v1", Resource: "jobs"},
	"statefulset": {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"crd":         {Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"},

	// Monitoring
	"prometheusRule": {Group: "monitoring.coreos.com", Version: "v1", Resource: "prometheusrules"},

	// OpenShift config
	"infrastructures":       {Group: "config.openshift.io", Version: "v1", Resource: "infrastructures"},
	"featuregate":           {Group: "config.openshift.io", Version: "v1", Resource: "featuregates"},
	"clusterversion":        {Group: "config.openshift.io", Version: "v1", Resource: "clusterversions"},
	"network.operator":      {Group: "operator.openshift.io", Version: "v1", Resource: "networks"},
	"authentication.config": {Group: "config.openshift.io", Version: "v1", Resource: "authentications"},
	"imagedigestmirrorset":  {Group: "config.openshift.io", Version: "v1", Resource: "imagedigestmirrorsets"},
	"oauth":                 {Group: "config.openshift.io", Version: "v1", Resource: "oauths"},

	// OpenShift route
	"route": {Group: "route.openshift.io", Version: "v1", Resource: "routes"},

	// OLM
	"subscription":    {Group: "operators.coreos.com", Version: "v1alpha1", Resource: "subscriptions"},
	"csv":             {Group: "operators.coreos.com", Version: "v1alpha1", Resource: "clusterserviceversions"},
	"operatorgroup":   {Group: "operators.coreos.com", Version: "v1", Resource: "operatorgroups"},
	"catalogsource":   {Group: "operators.coreos.com", Version: "v1alpha1", Resource: "catalogsources"},
	"packagemanifest": {Group: "packages.operators.coreos.com", Version: "v1", Resource: "packagemanifests"},

	// NetObserv
	"flowcollector":      {Group: "flows.netobserv.io", Version: "v1beta2", Resource: "flowcollectors"},
	"flowcollectorslice": {Group: "flows.netobserv.io", Version: "v1alpha1", Resource: "flowcollectorslices"},
	"flowmetric":         {Group: "flows.netobserv.io", Version: "v1alpha1", Resource: "flowmetrics"},

	// Kafka (Strimzi)
	"kafka":         {Group: "kafka.strimzi.io", Version: "v1beta2", Resource: "kafkas"},
	"kafkatopic":    {Group: "kafka.strimzi.io", Version: "v1beta2", Resource: "kafkatopics"},
	"kafkauser":     {Group: "kafka.strimzi.io", Version: "v1beta2", Resource: "kafkausers"},
	"kafkanodepool": {Group: "kafka.strimzi.io", Version: "v1beta2", Resource: "kafkanodepools"},

	// Loki
	"lokistack": {Group: "loki.grafana.com", Version: "v1", Resource: "lokistacks"},

	// Cloud credentials
	"CredentialsRequest": {Group: "cloudcredential.openshift.io", Version: "v1", Resource: "credentialsrequests"},

	// ODF
	"objectbucketclaims": {Group: "objectbucket.io", Version: "v1alpha1", Resource: "objectbucketclaims"},

	// Network attachment
	"net-attach-def": {Group: "k8s.cni.cncf.io", Version: "v1", Resource: "network-attachment-definitions"},

	// Storage
	"storageclass": {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},

	// Monitoring
	"servicemonitor": {Group: "monitoring.coreos.com", Version: "v1", Resource: "servicemonitors"},

	// Virtualization
	"hyperconverged": {Group: "hco.kubevirt.io", Version: "v1beta1", Resource: "hyperconvergeds"},
	"virtualmachine": {Group: "kubevirt.io", Version: "v1", Resource: "virtualmachines"},
	"vmi":            {Group: "kubevirt.io", Version: "v1", Resource: "virtualmachineinstances"},

	// NMState
	"nncp":    {Group: "nmstate.io", Version: "v1", Resource: "nodenetworkconfigurationpolicies"},
	"nmstate": {Group: "nmstate.io", Version: "v1", Resource: "nmstates"},

	// OpenShift monitoring
	"alertingrule": {Group: "monitoring.openshift.io", Version: "v1", Resource: "alertingrules"},

	// OpenTelemetry
	"opentelemetrycollector": {Group: "opentelemetry.io", Version: "v1beta1", Resource: "opentelemetrycollectors"},

	// OpenShift cluster operator
	"clusteroperator": {Group: "config.openshift.io", Version: "v1", Resource: "clusteroperators"},

	// Network Policy (AdminNetworkPolicy)
	"adminnetworkpolicy":         {Group: "policy.networking.k8s.io", Version: "v1alpha1", Resource: "adminnetworkpolicies"},
	"baselineadminnetworkpolicy": {Group: "policy.networking.k8s.io", Version: "v1alpha1", Resource: "baselineadminnetworkpolicies"},

	// User-defined networks (OVN)
	"userdefinednetwork":        {Group: "k8s.ovn.org", Version: "v1", Resource: "userdefinednetworks"},
	"clusteruserdefinednetwork": {Group: "k8s.ovn.org", Version: "v1", Resource: "clusteruserdefinednetworks"},
}

func initK8sClient() error {
	kc := kubeconfigPath
	if kc == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cannot determine home directory: %w", err)
		}
		kc = filepath.Join(home, ".kube", "config")
	}
	config, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		return fmt.Errorf("cannot build kubeconfig: %w", err)
	}
	config.QPS = 50
	config.Burst = 100
	k8sRestConfig = config
	k8sClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cannot create kubernetes clientset: %w", err)
	}
	k8sDynClient, err = dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cannot create dynamic client: %w", err)
	}
	routeV1Client, err = routeclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cannot create route client: %w", err)
	}
	return nil
}

func resolveGVR(resourceType string) (schema.GroupVersionResource, error) {
	gvr, ok := gvrMap[resourceType]
	if !ok {
		return schema.GroupVersionResource{}, fmt.Errorf("unknown resource type %q — add it to gvrMap in k8s_client.go", resourceType)
	}
	return gvr, nil
}

func getDynamicResource(resourceType, name, namespace string) (*unstructured.Unstructured, error) {
	gvr, err := resolveGVR(resourceType)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	if namespace == "" {
		return k8sDynClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	}
	return k8sDynClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
}

func getConditionStatus(obj *unstructured.Unstructured, condType string) (status, reason, message string) {
	conditions, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if err != nil || !found {
		return "", "", ""
	}
	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if t, _ := cond["type"].(string); t == condType {
			status, _ = cond["status"].(string)
			reason, _ = cond["reason"].(string)
			message, _ = cond["message"].(string)
			return
		}
	}
	return "", "", ""
}

// getNestedField extracts a string value using a dot-delimited path (e.g. ".status.phase")
func getNestedField(obj map[string]interface{}, dotPath string) (string, bool) {
	dotPath = strings.TrimPrefix(dotPath, ".")
	fields := strings.Split(dotPath, ".")
	val, found, err := unstructured.NestedString(obj, fields...)
	if err != nil || !found {
		return "", false
	}
	return val, true
}

func deleteDynamicResource(resourceType, name, namespace string) error {
	gvr, err := resolveGVR(resourceType)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if namespace == "" {
		return k8sDynClient.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
	}
	return k8sDynClient.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func patchDynamicResource(resourceType, name, namespace string, pt types.PatchType, patchData []byte) error {
	gvr, err := resolveGVR(resourceType)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if namespace == "" {
		_, err = k8sDynClient.Resource(gvr).Patch(ctx, name, pt, patchData, metav1.PatchOptions{})
	} else {
		_, err = k8sDynClient.Resource(gvr).Namespace(namespace).Patch(ctx, name, pt, patchData, metav1.PatchOptions{})
	}
	return err
}
