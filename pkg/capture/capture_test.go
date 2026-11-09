package capture

import (
	"encoding/json"
	"strings"
	"testing"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestBuildCollectorResources(t *testing.T) {
	service := BuildCollectorService("collector", "capture", map[string]string{"run": "collector"})
	if service.Spec.Ports[0].Port != 9999 || service.Spec.Ports[1].Port != 9401 || service.Spec.Selector["run"] != "collector" {
		t.Fatalf("unexpected collector service: %#v", service)
	}

	pod := BuildCollectorPod(&CollectorPodOptions{Name: "collector", Namespace: "capture", ServiceAccount: "capture", Image: "cli", Command: []string{"flows"}})
	if pod.Spec.Containers[0].Command[0] != "flows" || pod.Spec.ServiceAccountName != "capture" {
		t.Fatalf("unexpected collector pod: %#v", pod)
	}
	deadline := int64(120)
	pod = BuildCollectorPod(&CollectorPodOptions{Name: "collector", Namespace: "capture", ServiceAccount: "capture", Image: "cli", Command: []string{"flows"}, ActiveDeadlineSeconds: &deadline})
	if pod.Spec.ActiveDeadlineSeconds == nil || *pod.Spec.ActiveDeadlineSeconds != deadline {
		t.Fatalf("unexpected collector deadline: %#v", pod.Spec.ActiveDeadlineSeconds)
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
	if envValue(container.Env, "METRICS_ENABLE") != "true" || envValue(container.Env, "METRICS_SERVER_PORT") != "9400" {
		t.Fatalf("expected common health metrics to be enabled: %#v", container.Env)
	}
	if len(container.Ports) == 0 || container.Ports[0].ContainerPort != 9400 {
		t.Fatalf("expected health metrics port 9400: %#v", container.Ports)
	}
	for _, name := range []string{"ENABLE_DNS_TRACKING", "ENABLE_RTT", "ENABLE_TLS_TRACKING"} {
		if envValue(container.Env, name) != "true" {
			t.Fatalf("expected %s to be enabled: %#v", name, container.Env)
		}
	}
	if container.SecurityContext.Privileged == nil || *container.SecurityContext.Privileged {
		t.Fatal("non-privileged features should not require privileged mode")
	}
}

func TestBuildAgentDaemonSetUsesConfiguredMetricsPort(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector", MetricsPort: 23456,
	})

	container := ds.Spec.Template.Spec.Containers[0]
	if envValue(container.Env, "METRICS_SERVER_PORT") != "23456" || container.Ports[0].ContainerPort != 23456 {
		t.Fatalf("expected configured health metrics port: env=%s ports=%v", envValue(container.Env, "METRICS_SERVER_PORT"), container.Ports)
	}
}

func TestBuildAgentDaemonSetMetricsUsesOnePort(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector", Type: Metrics, MetricsPort: 23456,
	})

	ports := ds.Spec.Template.Spec.Containers[0].Ports
	if len(ports) != 1 || ports[0].Name != "prometheus" || ports[0].ContainerPort != 23456 {
		t.Fatalf("metrics agents must expose one prometheus port: %#v", ports)
	}
}

func TestBuildAgentDaemonSetTLSPlaintext(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector", Type: Packets,
		TLSPlaintext: &TLSPlaintextOptions{MinBytes: 32, PreviewBytes: 128, ProcessAllowlist: []string{"nginx"}},
	})
	container := ds.Spec.Template.Spec.Containers[0]
	if !ds.Spec.Template.Spec.HostPID || container.SecurityContext == nil || *container.SecurityContext.Privileged {
		t.Fatal("TLS plaintext capture must request host PID without privileged mode")
	}
	for _, name := range []string{"ENABLE_OPENSSL_TRACKING", "TLS_PLAINTEXT_MIN_BYTES", "TLS_PLAINTEXT_PREVIEW_BYTES", "TLS_PLAINTEXT_PROCESS_ALLOWLIST"} {
		if envValue(container.Env, name) == "" {
			t.Fatalf("expected %s to be configured: %#v", name, container.Env)
		}
	}
	if envValue(container.Env, "ENABLE_OPENSSL_TRACKING") != "true" || envValue(container.Env, "TLS_PLAINTEXT_MIN_BYTES") != "32" || envValue(container.Env, "TLS_PLAINTEXT_PREVIEW_BYTES") != "128" || envValue(container.Env, "TLS_PLAINTEXT_PROCESS_ALLOWLIST") != "nginx" {
		t.Fatalf("unexpected TLS plaintext environment: %#v", container.Env)
	}
	if len(container.VolumeMounts) != 3 || len(ds.Spec.Template.Spec.Volumes) != 3 {
		t.Fatalf("expected host library mounts: mounts=%v volumes=%v", container.VolumeMounts, ds.Spec.Template.Spec.Volumes)
	}
}

func TestBuildAgentDaemonSetFeaturePrivileges(t *testing.T) {
	tests := []struct {
		name       string
		feature    Feature
		volume     string
		capability corev1.Capability
	}{
		{"packet drops", PacketDrop, "bpf-kernel-debug", ""},
		{"network events", NetworkEvents, "var-run-ovn", ""},
		{"secondary networks", SecondaryNetworks, "var-run-netns", "SYS_ADMIN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := BuildAgentDaemonSet(&AgentOptions{Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector", Features: []Feature{tt.feature}})
			container := ds.Spec.Template.Spec.Containers[0]
			if *container.SecurityContext.Privileged || *container.SecurityContext.AllowPrivilegeEscalation {
				t.Fatal("feature must not enable privileged mode or privilege escalation")
			}
			if envValue(container.Env, FeatureEnvNameValue(tt.feature)) != "true" {
				t.Fatalf("feature environment is not enabled: %#v", container.Env)
			}
			if !hasVolume(ds.Spec.Template.Spec.Volumes, tt.volume) {
				t.Fatalf("expected volume %q: %#v", tt.volume, ds.Spec.Template.Spec.Volumes)
			}
			if tt.capability != "" && !hasCapability(container.SecurityContext.Capabilities.Add, tt.capability) {
				t.Fatalf("expected capability %q: %#v", tt.capability, container.SecurityContext.Capabilities.Add)
			}
		})
	}
}

func TestBuildAgentDaemonSetDeduplicatesFeatureVolumes(t *testing.T) {
	ds := BuildAgentDaemonSet(&AgentOptions{
		Name: "capture", Namespace: "capture", ServiceAccount: "capture", Image: "agent", TargetHost: "collector",
		Features: []Feature{NetworkEvents, UDNMapping, SecondaryNetworks},
	})

	if got := len(ds.Spec.Template.Spec.Volumes); got != 3 {
		t.Fatalf("expected three host volumes, got %d: %#v", got, ds.Spec.Template.Spec.Volumes)
	}
	if got := len(ds.Spec.Template.Spec.Containers[0].VolumeMounts); got != 3 {
		t.Fatalf("expected three host mounts, got %d: %#v", got, ds.Spec.Template.Spec.Containers[0].VolumeMounts)
	}
	if !hasVolume(ds.Spec.Template.Spec.Volumes, "var-run-netns") {
		t.Fatal("expected the shared netns volume")
	}
}

func FeatureEnvNameValue(feature Feature) string {
	name, _ := FeatureEnvName(feature)
	return name
}

func hasVolume(volumes []corev1.Volume, name string) bool {
	for i := range volumes {
		if volumes[i].Name == name {
			return true
		}
	}
	return false
}

func hasCapability(capabilities []corev1.Capability, wanted corev1.Capability) bool {
	for _, capability := range capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
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
	securityContext := ds.Spec.Template.Spec.Containers[0].SecurityContext
	if *securityContext.Privileged {
		t.Fatal("PacketDrop should use the granted capabilities without privileged mode")
	}
	if securityContext.AllowPrivilegeEscalation == nil || *securityContext.AllowPrivilegeEscalation {
		t.Fatal("FlowCapture agents must not allow privilege escalation")
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

func TestAddSubnetLabels(t *testing.T) {
	config := CollectorPipelineConfig("collector.capture.svc")
	AddSubnetLabels(config, map[string][]string{
		"Services": {"172.30.0.0/16"},
		"Machines": {"10.0.0.0/16", "fd00::/48"},
	})

	enrich := config.Parameters[0].Transform.Network
	if len(enrich.SubnetLabels) != 2 || enrich.SubnetLabels[0].Name != "Machines" || enrich.SubnetLabels[1].Name != "Services" {
		t.Fatalf("subnet labels are not sorted: %#v", enrich.SubnetLabels)
	}
	if len(enrich.Rules) < 5 || enrich.Rules[len(enrich.Rules)-2].Type != "add_subnet_label" || enrich.Rules[len(enrich.Rules)-1].Type != "add_subnet_label" {
		t.Fatalf("subnet label rules were not appended: %#v", enrich.Rules)
	}
}

func TestAddFilterQueriesRoutesCollectorPipeline(t *testing.T) {
	config := CollectorPipelineConfig("collector.capture.svc")
	AddFilterQueries(config, []string{`SrcPort>1024`, `DstPort<80`})

	if len(config.Pipeline) != 3 || config.Pipeline[1].Name != "filter" || config.Pipeline[2].Follows != "filter" {
		t.Fatalf("unexpected collector pipeline: %#v", config.Pipeline)
	}
	for i := range config.Parameters {
		if config.Parameters[i].Name == "filter" {
			rules := config.Parameters[i].Transform.Filter.Rules
			if len(rules) != 2 || rules[0].KeepEntryQuery != `SrcPort>1024` || rules[1].KeepEntryQuery != `DstPort<80` {
				t.Fatalf("unexpected query rules: %#v", rules)
			}
			return
		}
	}
	t.Fatal("expected filter stage parameter")
}

func TestAddFilterQueriesQuotesBareStringValues(t *testing.T) {
	config := CollectorPipelineConfig("collector")
	AddFilterQueries(config, []string{`K8S_FlowLayer=app AND DstK8S_Type=Pod AND SrcPort>1024`})

	for i := range config.Parameters {
		if config.Parameters[i].Name != "filter" {
			continue
		}
		got := config.Parameters[i].Transform.Filter.Rules[0].KeepEntryQuery
		want := `K8S_FlowLayer="app" AND DstK8S_Type="Pod" AND SrcPort>1024`
		if got != want {
			t.Fatalf("unexpected normalized query: got %q, want %q", got, want)
		}
		return
	}
	t.Fatal("expected filter stage parameter")
}

func TestAddFilterQueriesRoutesMetricsPipeline(t *testing.T) {
	config := MetricsPipelineConfig()
	AddFilterQueries(config, []string{"SrcPort>1024"})
	if len(config.Pipeline) != 3 || config.Pipeline[1].Name != "filter" || config.Pipeline[2].Name != "prometheus" || config.Pipeline[2].Follows != "filter" {
		t.Fatalf("unexpected metrics pipeline: %#v", config.Pipeline)
	}
}

func TestMetricsPipelineConfig(t *testing.T) {
	config := MetricsPipelineConfig()
	if config.MetricsSettings.Port != 9401 || config.MetricsSettings.Prefix != "on_demand_netobserv_" {
		t.Fatalf("unexpected metrics settings: %#v", config.MetricsSettings)
	}
	metrics := config.Parameters[1].Encode.Prom.Metrics
	if len(metrics) != 36 || metrics[0].Name != "node_egress_bytes_total" || metrics[len(metrics)-1].Name != "workload_ipsec_flows_total" {
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
	if serviceMonitor.Labels["monitoring.openshift.io/collection-profile"] != "full" {
		t.Fatalf("expected ServiceMonitor to opt into the full OpenShift collection profile: %#v", serviceMonitor.Labels)
	}

	service := BuildMetricsServiceFor("capture-agent", "capture", map[string]string{"flowcapture": "capture", "component": "agent"})
	if service.Spec.Selector["flowcapture"] != "capture" || service.Labels["app"] != "capture-agent" {
		t.Fatalf("unexpected FlowCapture metrics service: %#v", service)
	}
	flowCaptureMonitor := BuildMetricsServiceMonitorFor("capture-agent", "capture", map[string]string{"app": "capture-agent"})
	if flowCaptureMonitor.Spec.Selector.MatchLabels["app"] != "capture-agent" {
		t.Fatalf("unexpected FlowCapture service monitor selector: %#v", flowCaptureMonitor.Spec.Selector)
	}
	if got := flowCaptureMonitor.Spec.Endpoints[0].RelabelConfigs[0].Replacement; got == nil || *got != "capture" {
		t.Fatalf("unexpected FlowCapture target label: %#v", flowCaptureMonitor.Spec.Endpoints[0].RelabelConfigs)
	}
	healthService := BuildAgentHealthServiceFor("capture-agent", "capture", map[string]string{"flowcapture": "capture-agent"})
	if healthService.Name != "capture-agent-health" || healthService.Spec.Ports[0].Port != 9400 {
		t.Fatalf("unexpected FlowCapture health service: %#v", healthService)
	}
	healthService = BuildAgentHealthServiceForPort("capture-agent", "capture", map[string]string{"flowcapture": "capture-agent"}, 23456)
	if healthService.Spec.Ports[0].Port != 23456 || healthService.Spec.Ports[0].TargetPort.IntValue() != 23456 {
		t.Fatalf("unexpected custom FlowCapture health service: %#v", healthService)
	}
	healthMonitor := BuildAgentHealthServiceMonitorFor("capture-agent", "capture", map[string]string{"app": "capture-agent-health"})
	if got := healthMonitor.Spec.Endpoints[0].RelabelConfigs[0].Replacement; got == nil || *got != "capture" {
		t.Fatalf("unexpected FlowCapture health target label: %#v", healthMonitor.Spec.Endpoints[0].RelabelConfigs)
	}
}

func TestBuildMetricsDashboard(t *testing.T) {
	dashboard := BuildMetricsDashboard()
	data := dashboard.Data["on-demand-netobserv-flow-metrics.json"]
	if dashboard.Namespace != "openshift-config-managed" || data == "" {
		t.Fatalf("unexpected dashboard ConfigMap: %#v", dashboard)
	}
	var generated struct {
		Title string            `json:"title"`
		Rows  []json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal([]byte(data), &generated); err != nil {
		t.Fatalf("generated dashboard is not valid JSON: %v", err)
	}
	if generated.Title != "NetObserv / On Demand" || len(generated.Rows) == 0 {
		t.Fatalf("unexpected generated dashboard: title=%q rows=%d", generated.Title, len(generated.Rows))
	}
}

func TestBuildFlowCaptureDashboard(t *testing.T) {
	dashboard := BuildFlowCaptureDashboard()
	data := dashboard.Data["flowcaptures.json"]
	if dashboard.Name != "netobserv-flowcaptures" || dashboard.Namespace != "openshift-config-managed" || data == "" {
		t.Fatalf("unexpected FlowCapture dashboard ConfigMap: %#v", dashboard)
	}
	var generated struct {
		Title      string            `json:"title"`
		Rows       []json.RawMessage `json:"rows"`
		Templating struct {
			List []struct {
				Name  string `json:"name"`
				Query string `json:"query"`
			} `json:"list"`
		} `json:"templating"`
	}
	if err := json.Unmarshal([]byte(data), &generated); err != nil {
		t.Fatalf("generated FlowCapture dashboard is not valid JSON: %v", err)
	}
	if generated.Title != "NetObserv / FlowCaptures" || len(generated.Rows) == 0 || len(generated.Templating.List) != 1 {
		t.Fatalf("unexpected generated FlowCapture dashboard: %#v", generated)
	}
	if generated.Templating.List[0].Name != "flowcapture" || generated.Templating.List[0].Query == "" {
		t.Fatalf("missing FlowCapture dashboard variable: %#v", generated.Templating.List)
	}
	if !strings.Contains(data, "Total egress traffic") || !strings.Contains(data, "TCP latencies") {
		t.Fatalf("FlowCapture dashboard is missing OnDemand metric panels")
	}
	if strings.Count(data, "\"title\": \"Capture output\"") != 1 || strings.Contains(data, "Metrics processing") {
		t.Fatalf("FlowCapture dashboard output sections were not merged")
	}
	for _, metric := range []string{
		"flowcapture_agent_exported_batch_total",
		"flowcapture_agent_filtered_flows_total",
		"flowcapture_agent_dropped_flows_total",
		"flowcapture_agent_evicted_packets_total",
		"flowcapture_agent_openssl_data_events_total",
	} {
		if !strings.Contains(data, metric) {
			t.Fatalf("FlowCapture dashboard is missing capture activity metric %q", metric)
		}
	}
	if !strings.Contains(data, "flowcapture_collector_records_total") || !strings.Contains(data, "flowcapture_collector_output_bytes_total") {
		t.Fatalf("FlowCapture dashboard is missing collector output metrics")
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
