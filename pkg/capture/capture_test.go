package capture

import (
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildCollectorResources(t *testing.T) {
	service := BuildCollectorService("collector", "capture", map[string]string{"run": "collector"})
	if service.Spec.Ports[0].Port != 9999 || service.Spec.Selector["run"] != "collector" {
		t.Fatalf("unexpected collector service: %#v", service)
	}

	pod := BuildCollectorPod(&CollectorPodOptions{Name: "collector", Namespace: "capture", ServiceAccount: "capture", Image: "cli", Command: []string{"flows"}})
	if pod.Spec.Containers[0].Command[0] != "flows" || pod.Spec.ServiceAccountName != "capture" {
		t.Fatalf("unexpected collector pod: %#v", pod)
	}

	ds := BuildAgentDaemonSet(&AgentOptions{Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector"})
	if ds.Spec.Template.Spec.Containers[0].Env[1].Value != "collector" {
		t.Fatalf("unexpected agent target: %#v", ds.Spec.Template.Spec.Containers[0].Env)
	}
}

func TestBuildAgentDaemonSetFeatures(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector",
		Features: []Feature{DNSTracking, FlowRTT, TLSTracking},
	})

	container := ds.Spec.Template.Spec.Containers[0]
	for _, name := range []string{"ENABLE_DNS_TRACKING", "ENABLE_RTT", "ENABLE_TLS_TRACKING"} {
		if envValue(container.Env, name) != "true" {
			t.Fatalf("expected %s to be enabled: %#v", name, container.Env)
		}
	}
	if container.SecurityContext.Privileged == nil || *container.SecurityContext.Privileged {
		t.Fatal("non-privileged features should not require privileged mode")
	}
}

func TestBuildAgentDaemonSetNormalizesFilters(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector",
		Filters: []FlowFilter{{Protocol: "TCP"}},
	})

	if got := envValue(ds.Spec.Template.Spec.Containers[0].Env, "FLOW_FILTER_RULES"); got != `[{"ip_cidr":"0.0.0.0/0","protocol":"TCP","action":"Accept"}]` {
		t.Fatalf("unexpected normalized filters: %s", got)
	}
}

func TestNormalizeFiltersDoesNotMutateInput(t *testing.T) {
	filters := []FlowFilter{{Protocol: "TCP"}}
	_ = NormalizeFilters(filters)
	if filters[0].Action != "" || filters[0].IPCIDR != "" {
		t.Fatalf("NormalizeFilters mutated its input: %#v", filters)
	}
}

func TestValidateFeaturesForPacketCapture(t *testing.T) {
	if err := ValidateFeatures(Packets, []Feature{PacketTranslation}); err == nil {
		t.Fatal("expected packet feature validation to fail")
	}
	if err := ValidateFeatures(Flows, []Feature{PacketTranslation}); err != nil {
		t.Fatalf("flow feature should be accepted: %v", err)
	}
}

func TestBuildAgentDaemonSetPrivilegedFeature(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector",
		Features: []Feature{PacketDrop},
	})
	if !*ds.Spec.Template.Spec.Containers[0].SecurityContext.Privileged {
		t.Fatal("PacketDrop should require privileged mode")
	}
}

func TestCollectorCommand(t *testing.T) {
	got := CollectorCommand(Flows, "capture", "debug", "5m", 1024, "dns|namespace")
	want := []string{"/network-observability-cli", "get-flows", "--loglevel", "debug", "--maxtime", "5m", "--namespace", "capture", "--maxbytes", "1024", "--options", "dns|namespace"}
	if len(got) != len(want) {
		t.Fatalf("unexpected command: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected command: %v", got)
		}
	}
	metrics := CollectorCommand(Metrics, "capture", "info", "1m", 1024, "")
	for _, arg := range metrics {
		if arg == "--maxbytes" {
			t.Fatalf("metrics command must not include maxbytes: %v", metrics)
		}
	}
}

func TestCollectorPipelineConfig(t *testing.T) {
	config := CollectorPipelineConfig("collector.capture.svc")
	if config.Parameters[1].Write.GRPC.TargetHost != "collector.capture.svc" {
		t.Fatalf("collector target was not configured: %#v", config.Parameters[1])
	}
	if _, err := PipelineConfigJSON(config); err != nil {
		t.Fatalf("invalid collector pipeline config: %v", err)
	}
}

func TestMetricsPipelineConfig(t *testing.T) {
	config := MetricsPipelineConfig()
	if config.MetricsSettings.Port != 9401 || config.MetricsSettings.Prefix != "on_demand_netobserv_" {
		t.Fatalf("unexpected metrics settings: %#v", config.MetricsSettings)
	}
	metrics := config.Parameters[1].Encode.Prom.Metrics
	if len(metrics) != 30 || metrics[0].Name != "node_egress_bytes_total" || metrics[len(metrics)-1].Name != "workload_network_policy_events_total" {
		t.Fatalf("unexpected metrics pipeline: %d metrics, first=%q last=%q", len(metrics), metrics[0].Name, metrics[len(metrics)-1].Name)
	}
	if _, err := PipelineConfigJSON(config); err != nil {
		t.Fatalf("invalid metrics pipeline config: %v", err)
	}
}

func TestBuildMetricsResources(t *testing.T) {
	resources := BuildMetricsResources("capture")
	if len(resources) != 4 {
		t.Fatalf("expected four metrics resources, got %d", len(resources))
	}
	serviceMonitor := resources[2].(*monitoringv1.ServiceMonitor)
	if serviceMonitor.Spec.NamespaceSelector.MatchNames[0] != "capture" || serviceMonitor.Spec.Endpoints[0].Port != "prometheus" {
		t.Fatalf("unexpected service monitor: %#v", serviceMonitor.Spec)
	}
}

func TestBuildMetricsDashboard(t *testing.T) {
	dashboard := BuildMetricsDashboard()
	if dashboard.Namespace != "openshift-config-managed" || dashboard.Data["on-demand-netobserv-flow-metrics.json"] == "" {
		t.Fatalf("unexpected dashboard ConfigMap: %#v", dashboard)
	}
}

func TestParseMachineNetworkCIDRs(t *testing.T) {
	cidrs, err := ParseMachineNetworkCIDRs("networking:\n  machineNetwork:\n  - cidr: 192.168.0.0/16\n  - cidr: 10.0.0.0/8\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(cidrs) != 2 || cidrs[0] != "192.168.0.0/16" || cidrs[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected machine networks: %v", cidrs)
	}
}

func envValue(env []corev1.EnvVar, name string) string {
	for _, item := range env {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}
