package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	odcapi "github.com/netobserv/netobserv-operator/api/ondemandcapture/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// This opt-in test exercises the controller-generated pod and its RBAC without
// installing or replacing an operator on the target cluster.
func TestOnDemandCaptureLivePod(t *testing.T) {
	kubeconfig := os.Getenv("NETOBSERV_ODC_LIVE_KUBECONFIG")
	if kubeconfig == "" {
		t.Skip("set NETOBSERV_ODC_LIVE_KUBECONFIG for live pod validation")
	}
	image := os.Getenv("NETOBSERV_ODC_LIVE_IMAGE")
	if image == "" {
		t.Fatal("set NETOBSERV_ODC_LIVE_IMAGE")
	}
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	check(err)
	kube, err := kubernetes.NewForConfig(config)
	check(err)
	ns, err := kube.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "netobserv-odc-test-"}}, metav1.CreateOptions{})
	check(err)
	t.Cleanup(func() {
		cleanupCtx, c := context.WithTimeout(context.Background(), 2*time.Minute)
		defer c()
		check(kube.CoreV1().Namespaces().Delete(cleanupCtx, ns.Name, metav1.DeleteOptions{}))
	})
	owner := *metav1.NewControllerRef(ns, corev1.SchemeGroupVersion.WithKind("Namespace"))
	odc := &odcapi.OnDemandCapture{ObjectMeta: metav1.ObjectMeta{Name: "live", Namespace: ns.Name, UID: ns.UID}, Spec: odcapi.OnDemandCaptureSpec{CaptureType: odcapi.CaptureTypeFlows, Duration: &metav1.Duration{Duration: 20 * time.Second}, MaxBytes: ptr.To(int64(500000)), FlowConfig: &odcapi.FlowCaptureConfig{Sampling: 1}}}
	nodes, err := kube.CoreV1().Nodes().List(ctx, metav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	check(err)
	if len(nodes.Items) == 0 {
		t.Fatal("no worker nodes")
	}
	odc.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodes.Items[0].Labels["kubernetes.io/hostname"]}
	scheme := runtime.NewScheme()
	check(corev1.AddToScheme(scheme))
	check(rbacv1.AddToScheme(scheme))
	check(odcapi.AddToScheme(scheme))
	r := &OnDemandCaptureReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).Build(), operatorNamespace: ns.Name, netobservCLIImage: image, ebpfAgentImage: os.Getenv("NETOBSERV_ODC_LIVE_AGENT_IMAGE")}
	check(r.ensureCaptureAccess(ctx, odc))
	pod, err := r.createCLIPod(ctx, odc)
	check(err)
	// Replace fake CR ownership with the real test namespace's ownership.
	pod.ResourceVersion = ""
	pod.OwnerReferences = []metav1.OwnerReference{owner}
	sa := &corev1.ServiceAccount{}
	check(r.Get(ctx, types.NamespacedName{Name: captureName(odc), Namespace: ns.Name}, sa))
	sa.ResourceVersion = ""
	sa.OwnerReferences = []metav1.OwnerReference{owner}
	_, err = kube.CoreV1().ServiceAccounts(ns.Name).Create(ctx, sa, metav1.CreateOptions{})
	check(err)
	data, err := os.ReadFile("../../config/rbac/ondemandcapture_role.yaml")
	check(err)
	role := &rbacv1.ClusterRole{}
	check(yaml.Unmarshal(data, role))
	role.Name = captureName(odc)
	role.OwnerReferences = []metav1.OwnerReference{owner}
	_, err = kube.RbacV1().ClusterRoles().Create(ctx, role, metav1.CreateOptions{})
	check(err)
	_, err = kube.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: role.Name, OwnerReferences: []metav1.OwnerReference{owner}}, RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: role.Name}, Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: sa.Name, Namespace: sa.Namespace}}}, metav1.CreateOptions{})
	check(err)
	t.Cleanup(func() {
		_ = kube.CoreV1().Namespaces().Delete(context.Background(), captureNamespace(odc), metav1.DeleteOptions{})
	})
	_, err = kube.CoreV1().Pods(ns.Name).Create(ctx, pod, metav1.CreateOptions{})
	check(err)
	t.Logf("capture pod: %s/%s", ns.Name, pod.Name)
	err = wait.PollUntilContextTimeout(ctx, 3*time.Second, 8*time.Minute, true, func(ctx context.Context) (bool, error) {
		current, err := kube.CoreV1().Pods(ns.Name).Get(ctx, pod.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for i := range current.Status.ContainerStatuses {
			status := &current.Status.ContainerStatuses[i]
			if status.Name == "cli" && status.State.Terminated != nil {
				return status.State.Terminated.ExitCode == 0, fmtExitError(status.State.Terminated)
			}
		}
		return false, nil
	})
	if err != nil {
		logs, logErr := kube.CoreV1().Pods(ns.Name).GetLogs(pod.Name, &corev1.PodLogOptions{Container: "cli"}).DoRaw(ctx)
		if logErr == nil {
			t.Log(string(logs))
		}
		t.Fatal(err)
	}
	transport, upgrader, err := spdy.RoundTripperFor(config)
	check(err)
	endpoint := kube.CoreV1().RESTClient().Post().Resource("pods").Namespace(ns.Name).Name(pod.Name).SubResource("portforward").URL()
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, endpoint)
	stop, ready := make(chan struct{}), make(chan struct{})
	defer close(stop)
	forwarder, err := portforward.NewOnAddresses(dialer, []string{"127.0.0.1"}, []string{"0:8080"}, stop, ready, io.Discard, os.Stderr)
	check(err)
	forwardErr := make(chan error, 1)
	go func() { forwardErr <- forwarder.ForwardPorts() }()
	select {
	case <-ready:
	case err := <-forwardErr:
		t.Fatalf("port forwarding ended before readiness: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	ports, err := forwarder.GetPorts()
	check(err)
	base := fmt.Sprintf("http://127.0.0.1:%d/flow/", ports[0].Local)
	fetch := func(address string) ([]byte, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
		if err != nil {
			return nil, err
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download HTTP status %d", response.StatusCode)
		}
		return io.ReadAll(response.Body)
	}
	files, err := fetch(base)
	check(err)
	if len(files) == 0 {
		t.Fatal("empty artifact directory response")
	}
	fileLink := regexp.MustCompile(`href="([^"]+\.json)"`).FindSubmatch(files)
	if len(fileLink) != 2 {
		t.Fatal("no captured JSON file in output directory")
	}
	artifact, err := fetch(base + string(fileLink[1]))
	check(err)
	var flows []map[string]any
	check(json.Unmarshal(artifact, &flows))
	if len(flows) == 0 {
		t.Fatal("capture contains no flows")
	}
	t.Logf("downloaded %d flow records over HTTP", len(flows))
	// The capture namespace must be gone before the CLI reports success.
	_, err = kube.CoreV1().Namespaces().Get(ctx, captureNamespace(odc), metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Fatalf("capture namespace not confirmed deleted: %v", err)
	}
}

func fmtExitError(ended *corev1.ContainerStateTerminated) error {
	if ended.ExitCode == 0 {
		return nil
	}
	return fmt.Errorf("CLI exit %d: %s", ended.ExitCode, ended.Message)
}
