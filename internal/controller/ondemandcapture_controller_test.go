//nolint:revive
package controllers

import (
	"context"
	"k8s.io/apimachinery/pkg/util/validation"
	"strings"
	"testing"
	"time"

	odcapi "github.com/netobserv/netobserv-operator/api/ondemandcapture/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestOnDemandCaptureReconciler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OnDemandCapture Go CLI Suite")
}

var _ = Describe("Go CLI capture integration", func() {
	var r *OnDemandCaptureReconciler
	var odc *odcapi.OnDemandCapture
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(rbacv1.AddToScheme(scheme)).To(Succeed())
		Expect(odcapi.AddToScheme(scheme)).To(Succeed())
		odc = &odcapi.OnDemandCapture{ObjectMeta: metav1.ObjectMeta{Name: "capture", Namespace: "team", UID: types.UID("12345678-1234-1234-1234-123456789abc")}, Spec: odcapi.OnDemandCaptureSpec{CaptureType: odcapi.CaptureTypeFlows}}
		r = &OnDemandCaptureReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&odcapi.OnDemandCapture{}, &corev1.Pod{}).WithObjects(odc).Build(), operatorNamespace: "team", netobservCLIImage: "example/cli:test", ebpfAgentImage: "example/agent:test"}
	})
	It("names capture namespaces after the CR with a stable unique suffix", func() {
		expected := "netobserv-capture-12345678123412341234123456789abc"
		Expect(captureNamespace(odc)).To(Equal(expected))
		Expect(captureNamespace(odc.DeepCopy())).To(Equal(expected))
		other := odc.DeepCopy()
		other.Namespace = "other-team"
		other.UID = "87654321-4321-4321-4321-cba987654321"
		Expect(captureNamespace(other)).NotTo(Equal(expected))
		odc.Name = strings.Repeat("long-name.", 20) + "capture"
		name := captureNamespace(odc)
		Expect(len(name)).To(BeNumerically("<=", 63))
		Expect(validation.IsDNS1123Label(name)).To(BeEmpty())
		Expect(name).To(HaveSuffix("-12345678123412341234123456789abc"))
		odc.Name = strings.Repeat("a", 253)
		Expect(len(captureNamespace(odc))).To(Equal(63))
		Expect(validation.IsDNS1123Label(captureNamespace(odc))).To(BeEmpty())
		odc.Name = "a" + strings.Repeat("-", 30) + "capture"
		name = captureNamespace(odc)
		Expect(len(name)).To(BeNumerically("<=", 63))
		Expect(validation.IsDNS1123Label(name)).To(BeEmpty())
		Expect(name).To(Equal("netobserv-a-12345678123412341234123456789abc"))
	})

	It("forwards flow settings as separate Go CLI flags", func() {
		odc.Spec.Duration = &metav1.Duration{Duration: time.Minute}
		odc.Spec.MaxBytes = ptr.To(int64(500000))
		odc.Spec.Interfaces = []string{"eth0", "eth1"}
		odc.Spec.NodeSelector = map[string]string{"zone": "west", "role": "worker"}
		odc.Spec.FlowConfig = &odcapi.FlowCaptureConfig{Sampling: 1, EnableDNS: true, EnableRTT: true, EnablePacketDrops: true}
		Expect(r.validateSpec(odc)).To(Succeed())
		Expect(r.buildCLIArgs(odc)).To(Equal([]string{"flows", "--headless", "--copy=true", "--namespace=" + captureNamespace(odc), "--output-dir=/output", "--max-time=1m0s", "--max-bytes=500000", "--interfaces=eth0,eth1", "--node-selector=role:worker", "--node-selector=zone:west", "--sampling=1", "--enable_dns", "--enable_rtt", "--enable_pkt_drop"}))
	})
	It("uses packet destination port and protocol filters", func() {
		odc.Spec.CaptureType = odcapi.CaptureTypePackets
		odc.Spec.PacketConfig = &odcapi.PacketCaptureConfig{Port: 443, Protocol: "tcp"}
		Expect(r.validateSpec(odc)).To(Succeed())
		Expect(r.buildCLIArgs(odc)).To(ContainElements("packets", "--dport=443", "--protocol=TCP"))
	})
	It("forwards metrics features", func() {
		odc.Spec.CaptureType = odcapi.CaptureTypeMetrics
		odc.Spec.MetricsConfig = &odcapi.MetricsCaptureConfig{EnableDNS: true, EnableRTT: true, EnablePacketDrops: true}
		Expect(r.validateSpec(odc)).To(Succeed())
		Expect(r.buildCLIArgs(odc)).To(ContainElements("metrics", "--enable_dns", "--enable_rtt", "--enable_pkt_drop"))
	})
	It("rejects unsupported filters instead of silently ignoring them", func() {
		odc.Spec.Filters = &odcapi.CaptureFilters{Namespaces: []string{"payments"}}
		Expect(r.validateSpec(odc)).To(MatchError(ContainSubstring("not supported")))
	})
	It("rejects invalid limits and packet captures without a filter", func() {
		odc.Spec.Duration = &metav1.Duration{Duration: -time.Second}
		Expect(r.validateSpec(odc)).To(HaveOccurred())
		odc.Spec.Duration = nil
		odc.Spec.MaxBytes = ptr.To(int64(0))
		Expect(r.validateSpec(odc)).To(HaveOccurred())
		odc.Spec.MaxBytes = nil
		odc.Spec.CaptureType = odcapi.CaptureTypePackets
		Expect(r.validateSpec(odc)).To(HaveOccurred())
	})
	It("rejects tenant captures without granting credentials and keeps their retention deadline", func() {
		r.operatorNamespace = "operator"
		request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)}
		_, err := r.Reconcile(ctx, request)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Get(ctx, request.NamespacedName, odc)).To(Succeed())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureFailed))
		Expect(odc.Status.Error).To(ContainSubstring("administrator-only"))
		completed := odc.Status.CompletionTime.DeepCopy()
		_, err = r.Reconcile(ctx, request)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Get(ctx, request.NamespacedName, odc)).To(Succeed())
		Expect(odc.Status.CompletionTime).To(Equal(completed))
		Expect(apierrors.IsNotFound(r.Get(ctx, types.NamespacedName{Name: captureName(odc), Namespace: odc.Namespace}, &corev1.ServiceAccount{}))).To(BeTrue())
		Expect(apierrors.IsNotFound(r.Get(ctx, types.NamespacedName{Name: captureName(odc)}, &rbacv1.ClusterRoleBinding{}))).To(BeTrue())
		_, err = r.createCLIPod(ctx, odc)
		Expect(err).To(HaveOccurred())
		r.operatorNamespace = ""
		Expect(r.ensureCaptureAccess(ctx, odc)).To(HaveOccurred())
	})
	It("revokes legacy tenant access before stopping its launcher", func() {
		odc.Finalizers = []string{ondemandCaptureFinalizer}
		Expect(r.Update(ctx, odc)).To(Succeed())
		Expect(r.ensureCaptureAccess(ctx, odc)).To(Succeed())
		_, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		r.operatorNamespace = "operator"
		request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)}
		_, err = r.Reconcile(ctx, request)
		Expect(err).NotTo(HaveOccurred())
		Expect(apierrors.IsNotFound(r.Get(ctx, types.NamespacedName{Name: captureName(odc)}, &rbacv1.ClusterRoleBinding{}))).To(BeTrue())
		_, err = r.Reconcile(ctx, request)
		Expect(err).NotTo(HaveOccurred())
		Expect(r.Get(ctx, request.NamespacedName, odc)).To(Succeed())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureFailed))
	})
	It("uses in-cluster authentication and retains output in a separate server", func() {
		Expect(r.ensureCaptureAccess(ctx, odc)).To(Succeed())
		Expect(r.ensureCaptureAccess(ctx, odc)).To(Succeed())
		pod, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		again, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		Expect(again.Name).To(Equal(pod.Name))
		Expect(pod.Labels).To(HaveKeyWithValue("part-of", "netobserv-operator"))
		Expect(pod.Spec.Containers).To(HaveLen(2))
		Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/oc-netobserv"}))
		Expect(pod.Spec.Containers[0].Image).To(Equal(r.netobservCLIImage))
		Expect(pod.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "NETOBSERV_COLLECTOR_IMAGE", Value: r.netobservCLIImage}))
		Expect(pod.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "NETOBSERV_AGENT_IMAGE", Value: r.ebpfAgentImage}))
		Expect(pod.Spec.Containers[1].Args).To(Equal([]string{"serve", "--directory=/output", "--listen=127.0.0.1:8080"}))
		Expect(pod.Spec.Volumes).To(HaveLen(2))
		Expect(pod.Spec.Volumes[0].EmptyDir).NotTo(BeNil())
		Expect(pod.Spec.AutomountServiceAccountToken).To(HaveValue(BeFalse()))
		Expect(pod.Spec.Volumes[1].Projected.Sources).To(HaveLen(3))
		Expect(pod.Spec.Volumes[1].Projected.Sources[0].ServiceAccountToken.Path).To(Equal("token"))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(corev1.VolumeMount{Name: "capture-api-access", MountPath: "/var/run/secrets/kubernetes.io/serviceaccount", ReadOnly: true}))
		Expect(pod.Spec.Containers[1].VolumeMounts).To(HaveLen(1))
		Expect(pod.Spec.Containers[1].VolumeMounts[0].Name).To(Equal("output"))
		sa := &corev1.ServiceAccount{}
		Expect(r.Get(ctx, client.ObjectKeyFromObject(pod), sa)).To(Succeed())
		Expect(sa.AutomountServiceAccountToken).To(HaveValue(BeFalse()))
		Expect(pod.Spec.ServiceAccountName).To(Equal(captureName(odc)))
		Expect(metav1.IsControlledBy(pod, odc)).To(BeTrue())
	})
	It("passes the drops-only filter without requiring a separate tracking switch", func() {
		odc.Spec.Filters = &odcapi.CaptureFilters{CaptureFilterRule: odcapi.CaptureFilterRule{Drops: true}}
		Expect(r.buildCLIArgs(odc)).To(ContainElement("--drops"))
		odc.Spec.Filters.Drops = false
		Expect(r.buildCLIArgs(odc)).NotTo(ContainElement("--drops"))
	})
	It("uses the CLI build agent unless an on-demand override is configured", func() {
		r.ebpfAgentImage = ""
		pod, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		for _, env := range pod.Spec.Containers[0].Env {
			Expect(env.Name).NotTo(Equal("NETOBSERV_AGENT_IMAGE"))
		}
	})
	It("keeps running beyond wall-clock duration until the CLI finishes", func() {
		odc.Status.Phase = odcapi.OnDemandCaptureRunning
		odc.Status.PodName = captureName(odc)
		odc.Status.StartTime = &metav1.Time{Time: time.Now().Add(-time.Hour)}
		odc.Spec.Duration = &metav1.Duration{Duration: time.Second}
		_, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		_, err = r.handleRunning(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureRunning))
	})
	It("completes when the CLI exits while the download server stays running", func() {
		pod, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		pod.Status.Phase = corev1.PodRunning
		pod.Status.PodIP = "10.0.0.3"
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "cli", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 0, StartedAt: metav1.Now(), FinishedAt: metav1.Now()}}}, {Name: "files", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}
		Expect(r.Status().Update(ctx, pod)).To(Succeed())
		odc.Status.PodName = pod.Name
		_, err = r.handleRunning(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureCompleted))
		Expect(odc.Status.PodIP).To(Equal("10.0.0.3"))
		Expect(r.Get(ctx, client.ObjectKeyFromObject(pod), &corev1.Pod{})).To(Succeed())
	})
	It("reports the CLI exit error even when the server remains running", func() {
		pod, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		pod.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "cli", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{ExitCode: 1, Message: "permission denied"}}}}
		Expect(r.Status().Update(ctx, pod)).To(Succeed())
		odc.Status.PodName = pod.Name
		_, err = r.handleRunning(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureFailed))
		Expect(odc.Status.Error).To(ContainSubstring("permission denied"))
	})
	It("removes the launcher before its capture namespace and cluster binding", func() {
		odc.Finalizers = []string{ondemandCaptureFinalizer}
		Expect(r.Update(ctx, odc)).To(Succeed())
		Expect(r.ensureCaptureAccess(ctx, odc)).To(Succeed())
		_, err := r.createCLIPod(ctx, odc)
		Expect(err).NotTo(HaveOccurred())
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: captureNamespace(odc), Labels: map[string]string{"app": "netobserv-cli"}}}
		Expect(r.Create(ctx, ns)).To(Succeed())
		for range 3 {
			_, err = r.handleDeletion(ctx, odc)
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(odc.Finalizers).To(BeEmpty())
		Expect(apierrors.IsNotFound(r.Get(ctx, client.ObjectKeyFromObject(ns), &corev1.Namespace{}))).To(BeTrue())
		Expect(apierrors.IsNotFound(r.Get(ctx, types.NamespacedName{Name: captureName(odc)}, &rbacv1.ClusterRoleBinding{}))).To(BeTrue())
	})
	DescribeTable("expires terminal captures using their persisted completion time",
		func(phase odcapi.OnDemandCapturePhase, ttl *int32, age time.Duration, expired bool) {
			odc.Spec.TTLSecondsAfterFinished = ttl
			odc.Finalizers = []string{ondemandCaptureFinalizer}
			Expect(r.Update(ctx, odc)).To(Succeed())
			odc.Status.Phase = phase
			ended := metav1.NewTime(time.Now().Add(-age))
			odc.Status.CompletionTime = &ended
			Expect(r.Status().Update(ctx, odc)).To(Succeed())
			// A fresh reconciler simulates restart: there is no in-memory timer state.
			restarted := &OnDemandCaptureReconciler{Client: r.Client, operatorNamespace: "team"}
			result, err := restarted.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)})
			Expect(err).NotTo(HaveOccurred())
			Expect(r.Get(ctx, client.ObjectKeyFromObject(odc), odc)).To(Succeed())
			Expect(!odc.DeletionTimestamp.IsZero()).To(Equal(expired))
			if !expired {
				Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			}
		},
		Entry("default keeps a recent successful capture", odcapi.OnDemandCaptureCompleted, (*int32)(nil), time.Hour, false),
		Entry("default expires an old successful capture", odcapi.OnDemandCaptureCompleted, (*int32)(nil), 25*time.Hour, true),
		Entry("default expires a failed capture", odcapi.OnDemandCaptureFailed, (*int32)(nil), 25*time.Hour, true),
		Entry("custom retention is honored", odcapi.OnDemandCaptureCompleted, ptr.To(int32(60)), 2*time.Minute, true),
		Entry("extended retention preserves old output", odcapi.OnDemandCaptureCompleted, ptr.To(int32(172800)), 25*time.Hour, false),
		Entry("zero retention cleans up immediately", odcapi.OnDemandCaptureCompleted, ptr.To(int32(0)), time.Second, true),
	)
	It("gives legacy terminal resources a persisted retention start", func() {
		odc.Status.Phase = odcapi.OnDemandCaptureFailed
		Expect(r.Status().Update(ctx, odc)).To(Succeed())
		result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(time.Second))
		Expect(r.Get(ctx, client.ObjectKeyFromObject(odc), odc)).To(Succeed())
		Expect(odc.Status.CompletionTime).NotTo(BeNil())
		Expect(odc.DeletionTimestamp.IsZero()).To(BeTrue())
	})
	It("does not expire an active capture with zero retention", func() {
		odc.Spec.TTLSecondsAfterFinished = ptr.To(int32(0))
		Expect(r.Update(ctx, odc)).To(Succeed())
		request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)}
		for range 4 {
			_, err := r.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(r.Get(ctx, request.NamespacedName, odc)).To(Succeed())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureRunning))
		Expect(odc.DeletionTimestamp.IsZero()).To(BeTrue())
	})

	It("isolates captures and initializes resources idempotently", func() {
		other := odc.DeepCopy()
		other.UID = "different"
		Expect(captureNamespace(other)).NotTo(Equal(captureNamespace(odc)))
		request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(odc)}
		for range 3 {
			_, err := r.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
		}
		Expect(r.Get(ctx, request.NamespacedName, odc)).To(Succeed())
		Expect(odc.Status.Phase).To(Equal(odcapi.OnDemandCaptureRunning))
		pods := &corev1.PodList{}
		Expect(r.List(ctx, pods)).To(Succeed())
		Expect(pods.Items).To(HaveLen(1))
	})
})
