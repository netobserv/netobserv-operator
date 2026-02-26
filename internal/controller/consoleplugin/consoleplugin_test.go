package consoleplugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/consoleplugin/config"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
)

const testImage = "quay.io/netobserv/network-observability-console-plugin:dev"
const testNamespace = constants.PluginName

var testPullPolicy = corev1.PullIfNotPresent
var testResources = corev1.ResourceRequirements{
	Limits: map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU:    resource.MustParse("1"),
		corev1.ResourceMemory: resource.MustParse("512Mi"),
	},
}
var lokiStatusUnused = status.ComponentStatus{
	Name:   status.LokiStack,
	Status: status.StatusUnknown,
}

func getPluginConfig() flowslatest.FlowCollectorConsolePlugin {
	return flowslatest.FlowCollectorConsolePlugin{
		Enable:          ptr.To(true),
		ImagePullPolicy: string(testPullPolicy),
		Resources:       testResources,
		Autoscaler: flowslatest.FlowCollectorHPA{
			Status:      flowslatest.HPAStatusEnabled,
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []ascv2.MetricSpec{{
				Type: ascv2.ResourceMetricSourceType,
				Resource: &ascv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: ascv2.MetricTarget{
						Type:               ascv2.UtilizationMetricType,
						AverageUtilization: &targetCPU,
					},
				},
			}},
		},
		LogLevel: "info",
	}
}

func getServiceSpecs() corev1.Service {
	var service = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port:     9001,
					Protocol: "TCP",
				},
			},
		},
	}

	return service
}

var minReplicas = int32(1)
var maxReplicas = int32(5)
var targetCPU = int32(75)

func getAutoScalerSpecs() (ascv2.HorizontalPodAutoscaler, flowslatest.FlowCollectorConsolePlugin) {
	var autoScaler = ascv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
		},
		Spec: ascv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: ascv2.CrossVersionObjectReference{
				Kind: "Deployment",
				Name: constants.PluginName,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []ascv2.MetricSpec{{
				Type: ascv2.ResourceMetricSourceType,
				Resource: &ascv2.ResourceMetricSource{
					Name: corev1.ResourceCPU,
					Target: ascv2.MetricTarget{
						Type:               ascv2.UtilizationMetricType,
						AverageUtilization: &targetCPU,
					},
				},
			}},
		},
	}

	return autoScaler, getPluginConfig()
}

func getBuilder(spec *flowslatest.FlowCollectorSpec, lk *helper.LokiConfig) builder {
	info := reconcilers.Common{Namespace: testNamespace, Loki: lk, ClusterInfo: &cluster.Info{}}
	b := newBuilder(info.NewInstance(map[reconcilers.ImageRef]string{reconcilers.MainImage: testImage}, status.Instance{}), spec, constants.PluginName)
	_, _, _ = b.configMap(context.Background(), nil, lokiStatusUnused) // build configmap to update builder's volumes
	return b
}

func TestContainerUpdateCheck(t *testing.T) {
	assert := assert.New(t)

	// equals specs
	plugin := getPluginConfig()
	loki := helper.LokiConfig{
		LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://loki:3100/", TenantID: "netobserv"},
	}
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin}
	builder := getBuilder(&spec, &loki)
	old := builder.deployment(constants.PluginName, "digest")
	nEw := builder.deployment(constants.PluginName, "digest")
	report := helper.NewChangeReport("")
	assert.False(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "no change")

	// wrong resources
	spec.ConsolePlugin.Resources.Limits = map[corev1.ResourceName]resource.Quantity{
		corev1.ResourceCPU:    resource.MustParse("500m"),
		corev1.ResourceMemory: resource.MustParse("500Gi"),
	}
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "req/limit changed")
	old = nEw

	// new image
	builder.info.Images[reconcilers.MainImage] = "quay.io/netobserv/network-observability-console-plugin:latest"
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Image changed")
	old = nEw

	// new pull policy
	spec.ConsolePlugin.ImagePullPolicy = string(corev1.PullAlways)
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Pull policy changed")
	old = nEw

	// new log level
	spec.ConsolePlugin.LogLevel = "debug"
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Args changed")
	old = nEw

	// new loki config
	loki = helper.LokiConfig{
		LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://loki:3100/", TenantID: "netobserv", TLS: flowslatest.ClientTLS{
			Enable: true,
			CACert: flowslatest.CertificateReference{
				Type:     "configmap",
				Name:     "cm-name",
				CertFile: "ca.crt",
			},
		}}}
	builder = getBuilder(&spec, &loki)
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Volumes changed")
	old = nEw

	// new loki cert name
	loki.LokiManualParams.TLS.CACert.Name = "cm-name-2"
	builder = getBuilder(&spec, &loki)
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Volumes changed")
	old = nEw

	// new toleration
	spec.ConsolePlugin.Advanced = &flowslatest.AdvancedPluginConfig{
		Scheduling: &flowslatest.SchedulingConfig{
			Tolerations: []corev1.Toleration{{Key: "dummy-key", Operator: corev1.TolerationOpExists}},
		},
	}
	builder = getBuilder(&spec, &loki)
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.True(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "Toleration changed")
	old = nEw

	// test again no change
	loki.LokiManualParams.TLS.CACert.Name = "cm-name-2"
	builder = getBuilder(&spec, &loki)
	nEw = builder.deployment(constants.PluginName, "digest")
	report = helper.NewChangeReport("")
	assert.False(helper.PodChanged(&old.Spec.Template, &nEw.Spec.Template, constants.PluginName, &report))
	assert.Contains(report.String(), "no change")
}

func TestConfigMapUpdateCheck(t *testing.T) {
	assert := assert.New(t)

	// equals specs
	plugin := getPluginConfig()
	loki := helper.LokiConfig{
		LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://loki:3100/", TenantID: "netobserv"},
	}
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin}
	builder := getBuilder(&spec, &loki)
	old, _, _ := builder.configMap(context.Background(), nil, lokiStatusUnused)
	nEw, _, _ := builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.Equal(old.Data, nEw.Data)

	// update loki
	loki = helper.LokiConfig{
		LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://loki:3100/", TenantID: "netobserv", TLS: flowslatest.ClientTLS{
			Enable: true,
			CACert: flowslatest.CertificateReference{
				Type:     "configmap",
				Name:     "cm-name",
				CertFile: "ca.crt",
			},
		}},
	}
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
	old = nEw

	// set status url and enable default tls
	loki.LokiManualParams.StatusURL = "http://loki.status:3100/"
	loki.LokiManualParams.StatusTLS.Enable = true
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
	old = nEw

	// update status ca cert
	loki.LokiManualParams.StatusTLS.CACert = flowslatest.CertificateReference{
		Type:     "configmap",
		Name:     "status-cm-name",
		CertFile: "status-ca.crt",
	}
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
	old = nEw

	// update status user cert
	loki.LokiManualParams.StatusTLS.UserCert = ptr.Deref(helper.DefaultCertificateReference("sec-name", ""), flowslatest.CertificateReference{})
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
}

func TestConfigMapUpdateWithLokistackMode(t *testing.T) {
	assert := assert.New(t)

	// equals specs
	plugin := getPluginConfig()
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "lokistack", Namespace: "ls-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin, Loki: lokiSpec}
	builder := getBuilder(&spec, &loki)
	old, _, _ := builder.configMap(context.Background(), nil, lokiStatusUnused)
	nEw, _, _ := builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.Equal(old.Data, nEw.Data)

	// update lokistack name
	lokiSpec.LokiStack.Name = "lokistack-updated"
	loki = helper.NewLokiConfig(&lokiSpec, "any")

	spec = flowslatest.FlowCollectorSpec{ConsolePlugin: plugin, Loki: lokiSpec}
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
	old = nEw

	// update lokistack namespace
	lokiSpec.LokiStack.Namespace = "ls-namespace-updated"
	loki = helper.NewLokiConfig(&lokiSpec, "any")

	spec = flowslatest.FlowCollectorSpec{ConsolePlugin: plugin, Loki: lokiSpec}
	builder = getBuilder(&spec, &loki)
	nEw, _, _ = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotEqual(old.Data, nEw.Data)
}

func TestConfigMapContent(t *testing.T) {
	assert := assert.New(t)

	agentSpec := flowslatest.FlowCollectorAgent{
		Type: "eBPF",
		EBPF: flowslatest.FlowCollectorEBPF{
			Sampling: ptr.To(int32(1)),
		},
	}
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "lokistack", Namespace: "ls-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{
		Agent:         agentSpec,
		ConsolePlugin: getPluginConfig(),
		Loki:          lokiSpec,
		Processor:     flowslatest.FlowCollectorFLP{SubnetLabels: flowslatest.SubnetLabels{OpenShiftAutoDetect: ptr.To(false)}},
	}
	builder := getBuilder(&spec, &loki)
	cm, _, err := builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.NotNil(cm)
	assert.Nil(err)

	// parse output config and check expected values
	var config config.PluginConfig
	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &config)
	assert.Nil(err)

	// loki config
	assert.Equal(config.Loki.URL, "https://lokistack-gateway-http.ls-namespace.svc.cluster.local.:8080/api/logs/v1/network/")
	assert.Equal(config.Loki.StatusURL, "https://lokistack-query-frontend-http.ls-namespace.svc.cluster.local.:3100/")

	// frontend params
	assert.Equal(config.Frontend.RecordTypes, []api.ConnTrackOutputRecordTypeEnum{api.ConnTrackFlowLog})
	assert.Empty(config.Frontend.Features)
	assert.NotEmpty(config.Frontend.Columns)
	assert.NotEmpty(config.Frontend.Filters)
	assert.NotEmpty(config.Frontend.Scopes)
	assert.Equal(config.Frontend.Sampling, 1)
}

func TestConfigMapExternalRecordingAnnotations(t *testing.T) {
	assert := assert.New(t)
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "lokistack", Namespace: "ls-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{
		ConsolePlugin: getPluginConfig(),
		Loki:          lokiSpec,
		Processor:     flowslatest.FlowCollectorFLP{SubnetLabels: flowslatest.SubnetLabels{OpenShiftAutoDetect: ptr.To(false)}},
	}
	builder := getBuilder(&spec, &loki)

	external := map[string]map[string]string{
		"my_custom_metric": {
			"summary":                     "Custom metric",
			"netobserv_io_network_health": `{"recordingThresholds":{"info":"10"}}`,
		},
	}
	cm, _, err := builder.configMap(context.Background(), external, lokiStatusUnused)
	assert.NotNil(cm)
	assert.NoError(err)

	var pluginConfig config.PluginConfig
	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &pluginConfig)
	assert.NoError(err)
	assert.Contains(pluginConfig.Frontend.RecordingAnnotations, "my_custom_metric")
	assert.Equal("Custom metric", pluginConfig.Frontend.RecordingAnnotations["my_custom_metric"]["summary"])
	assert.Equal(`{"recordingThresholds":{"info":"10"}}`, pluginConfig.Frontend.RecordingAnnotations["my_custom_metric"]["netobserv_io_network_health"])
}

func TestServiceUpdateCheck(t *testing.T) {
	assert := assert.New(t)
	old := getServiceSpecs()

	// equals specs
	serviceSpec := getServiceSpecs()
	report := helper.NewChangeReport("")
	assert.Equal(helper.ServiceChanged(&old, &serviceSpec, &report), false)
	assert.Contains(report.String(), "no change")

	// wrong port protocol
	serviceSpec = getServiceSpecs()
	serviceSpec.Spec.Ports[0].Protocol = "UDP"
	report = helper.NewChangeReport("")
	assert.Equal(helper.ServiceChanged(&old, &serviceSpec, &report), true)
	assert.Contains(report.String(), "Service spec changed")

	// wrong port number
	serviceSpec = getServiceSpecs()
	serviceSpec.Spec.Ports[0].Port = 8080
	report = helper.NewChangeReport("")
	assert.Equal(helper.ServiceChanged(&old, &serviceSpec, &report), true)
	assert.Contains(report.String(), "Service spec changed")
}

func TestBuiltService(t *testing.T) {
	assert := assert.New(t)

	// newly created service should not need update
	plugin := getPluginConfig()
	loki := helper.LokiConfig{LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://foo:1234"}}
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin}
	builder := getBuilder(&spec, &loki)
	old := builder.mainService(constants.PluginName)
	nEw := builder.mainService(constants.PluginName)
	report := helper.NewChangeReport("")
	assert.Equal(helper.ServiceChanged(old, nEw, &report), false)
	assert.Contains(report.String(), "no change")
}

func TestLabels(t *testing.T) {
	assert := assert.New(t)

	plugin := getPluginConfig()
	loki := helper.LokiConfig{LokiManualParams: flowslatest.LokiManualParams{IngesterURL: "http://foo:1234"}}
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin}
	builder := getBuilder(&spec, &loki)

	// Deployment
	depl := builder.deployment(constants.PluginName, "digest")
	assert.Equal("netobserv-plugin", depl.Labels["app"])
	assert.Equal(constants.OperatorName, depl.Labels["part-of"])
	assert.Equal("netobserv-plugin", depl.Spec.Template.Labels["app"])
	assert.Equal(constants.OperatorName, depl.Spec.Template.Labels["part-of"])
	assert.Equal("dev", depl.Labels["version"])
	assert.Equal("dev", depl.Spec.Template.Labels["version"])

	// Service
	svc := builder.mainService(constants.PluginName)
	assert.Equal("netobserv-plugin", svc.Labels["app"])
	assert.Equal(constants.OperatorName, svc.Labels["part-of"])
	assert.Equal("netobserv-plugin", svc.Spec.Selector["app"])
	assert.Equal("dev", svc.Labels["version"])
	assert.Empty(svc.Spec.Selector["version"])
}

func TestAutoScalerUpdateCheck(t *testing.T) {
	assert := assert.New(t)

	// equals specs
	autoScaler, plugin := getAutoScalerSpecs()
	report := helper.NewChangeReport("")
	assert.Equal(helper.AutoScalerChanged(&autoScaler, plugin.Autoscaler, &report), false)
	assert.Contains(report.String(), "no change")

	// wrong max replicas
	autoScaler, plugin = getAutoScalerSpecs()
	autoScaler.Spec.MaxReplicas = 10
	report = helper.NewChangeReport("")
	assert.Equal(helper.AutoScalerChanged(&autoScaler, plugin.Autoscaler, &report), true)
	assert.Contains(report.String(), "Max replicas changed")

	// missing min replicas
	autoScaler, plugin = getAutoScalerSpecs()
	autoScaler.Spec.MinReplicas = nil
	report = helper.NewChangeReport("")
	assert.Equal(helper.AutoScalerChanged(&autoScaler, plugin.Autoscaler, &report), true)
	assert.Contains(report.String(), "Min replicas changed")

	// missing metrics
	autoScaler, plugin = getAutoScalerSpecs()
	autoScaler.Spec.Metrics = []ascv2.MetricSpec{}
	report = helper.NewChangeReport("")
	assert.Equal(helper.AutoScalerChanged(&autoScaler, plugin.Autoscaler, &report), true)
	assert.Contains(report.String(), "Metrics changed")
}

func TestNoMissingFields(t *testing.T) {
	cfg, err := config.GetStaticFrontendConfig()
	assert.NoError(t, err)

	hasField := func(name string) bool {
		for _, f := range cfg.Fields {
			if f.Name == name {
				return true
			}
		}
		return false
	}

	var missing []string
	for _, col := range cfg.Columns {
		if col.Field != "" {
			if !hasField(col.Field) {
				missing = append(missing, col.Field)
			}
		}
		if len(col.Fields) > 0 {
			for _, f := range col.Fields {
				if !hasField(f) {
					missing = append(missing, f)
				}
			}
		}
	}
	assert.Empty(t, missing, "Missing fields should be added in static config file, under 'fields'")
}

func TestLokiStackStatusEmbedding(t *testing.T) {
	assert := assert.New(t)

	plugin := getPluginConfig()
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "lokistack", Namespace: "ls-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin, Loki: lokiSpec}
	builder := getBuilder(&spec, &loki)

	// Test 1: LokiStack with ready status
	lokiStackReady := status.ComponentStatus{
		Name:   status.LokiStack,
		Status: status.StatusReady,
	}
	cm, _, err := builder.configMap(context.Background(), nil, lokiStackReady)
	assert.Nil(err)
	assert.NotNil(cm)

	var cfg config.PluginConfig
	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfg)
	assert.Nil(err)
	assert.Equal("ready", cfg.Loki.Status)
	assert.Empty(cfg.Loki.StatusURL, "StatusURL should be cleared when LokiStack status is embedded")

	// Test 2: LokiStack with pending status (no ready condition)
	lokiStackPending := status.ComponentStatus{
		Name:   status.LokiStack,
		Status: status.StatusInProgress,
		Reason: "PendingComponents",
	}
	cm, _, err = builder.configMap(context.Background(), nil, lokiStackPending)
	assert.Nil(err)
	assert.NotNil(cm)

	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfg)
	assert.Nil(err)
	assert.Equal("pending", cfg.Loki.Status)
	assert.Empty(cfg.Loki.StatusURL)

	// Test 3: No LokiStack provided (nil)
	cm, _, err = builder.configMap(context.Background(), nil, lokiStatusUnused)
	assert.Nil(err)
	assert.NotNil(cm)

	// Create a new config variable to avoid reusing old values
	var cfgNil config.PluginConfig
	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfgNil)
	assert.Nil(err)
	assert.Empty(cfgNil.Loki.Status, "Status should be empty when no LokiStack is provided")
	// StatusURL should be present when using LokiStack mode without actual LokiStack object
	assert.NotEmpty(cfgNil.Loki.StatusURL)
}

func TestLokiStackNamespaceDefaulting(t *testing.T) {
	assert := assert.New(t)

	// Test 1: LokiStack namespace is explicitly set
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "my-lokistack", Namespace: "custom-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "default-namespace")

	// Verify URLs use the explicitly set namespace
	assert.Contains(loki.QuerierURL, "custom-namespace")
	assert.Contains(loki.StatusURL, "custom-namespace")
	assert.NotContains(loki.QuerierURL, "default-namespace")

	// Test 2: LokiStack namespace is empty (should default to FlowCollector namespace)
	lokiSpecDefault := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "my-lokistack", Namespace: ""},
	}
	lokiDefault := helper.NewLokiConfig(&lokiSpecDefault, "flowcollector-namespace")

	// Verify URLs use the defaulted namespace
	assert.Contains(lokiDefault.QuerierURL, "flowcollector-namespace")
	assert.Contains(lokiDefault.StatusURL, "flowcollector-namespace")

	// Test 3: Verify the exact URL format with namespace
	expectedGatewayURL := "https://my-lokistack-gateway-http.flowcollector-namespace.svc.cluster.local.:8080/api/logs/v1/network/"
	expectedStatusURL := "https://my-lokistack-query-frontend-http.flowcollector-namespace.svc.cluster.local.:3100/"
	assert.Equal(expectedGatewayURL, lokiDefault.QuerierURL)
	assert.Equal(expectedGatewayURL, lokiDefault.IngesterURL)
	assert.Equal(expectedStatusURL, lokiDefault.StatusURL)
}

func TestLokiStackNotFoundBehavior(t *testing.T) {
	assert := assert.New(t)

	plugin := getPluginConfig()
	lokiSpec := flowslatest.FlowCollectorLoki{
		Mode:      flowslatest.LokiModeLokiStack,
		LokiStack: flowslatest.LokiStackRef{Name: "missing-lokistack", Namespace: "test-namespace"},
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{ConsolePlugin: plugin, Loki: lokiSpec}
	builder := getBuilder(&spec, &loki)

	// Test behavior when LokiStack is not found (nil is passed)
	// This simulates the reconciler behavior when Get() returns NotFound
	cm, digest, err := builder.configMap(context.Background(), nil, lokiStatusUnused)

	// ConfigMap should still be created successfully
	assert.Nil(err)
	assert.NotNil(cm)
	assert.NotEmpty(digest)

	// Parse the config
	var cfg config.PluginConfig
	err = yaml.Unmarshal([]byte(cm.Data["config.yaml"]), &cfg)
	assert.Nil(err)

	// Verify that:
	// 1. Loki URL is still configured (from LokiStack ref)
	assert.NotEmpty(cfg.Loki.URL)
	assert.Contains(cfg.Loki.URL, "missing-lokistack")

	// 2. StatusURL is present (from default LokiStack configuration)
	assert.NotEmpty(cfg.Loki.StatusURL)

	// 3. Status field is not set (because LokiStack object is not available)
	assert.Empty(cfg.Loki.Status)

	// This ensures the console plugin can still function with a status URL
	// even if the LokiStack resource is temporarily unavailable
}

func TestScopeFilteringWithLoki(t *testing.T) {
	scopeIDs := func(scopes []config.ScopeConfig) []string {
		var ids []string
		for i := range scopes {
			ids = append(ids, scopes[i].ID)
		}
		return ids
	}
	plugin := getPluginConfig()
	lokiSpec := flowslatest.FlowCollectorLoki{
		Enable: ptr.To(true),
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Features: []flowslatest.AgentFeature{flowslatest.UDNMapping},
			},
		},
		ConsolePlugin: plugin,
		Loki:          lokiSpec,
	}
	builder := getBuilder(&spec, &loki)
	frontend, err := config.GetStaticFrontendConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{"cluster", "network", "zone", "host", "namespace", "owner", "resource"}, scopeIDs(frontend.Scopes))

	prom := builder.getPromConfig(context.Background())
	warning, err := builder.setFrontendConfig(&frontend, prom.Metrics)
	require.NoError(t, err)
	assert.Equal(t, []string{"network", "host", "namespace", "owner", "resource"}, scopeIDs(frontend.Scopes))
	assert.Equal(t, "", warning)

	// Make sure scope groups aren't modified
	assert.Contains(t, frontend.Scopes[3].Groups, "hosts")
	assert.Contains(t, frontend.Scopes[3].Groups, "hosts+namespaces")
	assert.Contains(t, frontend.Scopes[3].Groups, "networks+hosts")
	assert.Contains(t, frontend.Scopes[3].Groups, "networks+namespaces")
}

func TestScopeFilteringNoLoki(t *testing.T) {
	scopeIDs := func(scopes []config.ScopeConfig) []string {
		var ids []string
		for i := range scopes {
			ids = append(ids, scopes[i].ID)
		}
		return ids
	}
	plugin := getPluginConfig()
	lokiSpec := flowslatest.FlowCollectorLoki{
		Enable: ptr.To(false),
	}
	loki := helper.NewLokiConfig(&lokiSpec, "any")
	spec := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Features: []flowslatest.AgentFeature{flowslatest.UDNMapping},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			AddZone: ptr.To(true),
			Metrics: flowslatest.FLPMetrics{
				IncludeList: &[]flowslatest.FLPMetric{
					"node_egress_bytes_total",
					"namespace_egress_bytes_total",
				},
			},
		},
		ConsolePlugin: plugin,
		Loki:          lokiSpec,
	}
	builder := getBuilder(&spec, &loki)
	frontend, err := config.GetStaticFrontendConfig()
	require.NoError(t, err)
	assert.Equal(t, []string{"cluster", "network", "zone", "host", "namespace", "owner", "resource"}, scopeIDs(frontend.Scopes))

	prom := builder.getPromConfig(context.Background())
	warning, err := builder.setFrontendConfig(&frontend, prom.Metrics)
	require.NoError(t, err)
	assert.Equal(t, []string{"zone", "host", "namespace"}, scopeIDs(frontend.Scopes))
	assert.Contains(t, warning, "Scope network invalid for metrics")
	assert.Contains(t, warning, "Scope owner invalid for metrics")
	assert.Contains(t, warning, "candidates: netobserv_workload_egress_bytes_total, ")

	// Make sure scope groups are tailored to metrics
	assert.Empty(t, frontend.Scopes[0].Groups)
	assert.Equal(t, []string{"zones"}, frontend.Scopes[1].Groups)
	assert.Equal(t, []string{"zones"}, frontend.Scopes[2].Groups)
}
