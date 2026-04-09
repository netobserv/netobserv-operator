package consoleplugin

import (
	"context"
	"fmt"
	"hash/fnv"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	osv1 "github.com/openshift/api/console/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	cfg "github.com/netobserv/netobserv-operator/internal/controller/consoleplugin/config"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper/loki"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/metrics"
	"github.com/netobserv/netobserv-operator/internal/pkg/metrics/alerts"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
)

const proxyAlias = "backend"

const configMapName = "console-plugin-config"

// Annotation on PrometheusRule metadata for recording-rule metadata (key = metric name, value = same as alert annotations).
const recordingAnnotationsAnnotation = "netobserv.io/network-health"
const configFile = "config.yaml"
const configVolume = "config-volume"
const configPath = "/opt/app-root/"
const metricsSvcName = constants.PluginName + "-metrics"
const metricsPort = 9002
const metricsPortName = "metrics"

type builder struct {
	info          *reconcilers.Instance
	imageRef      reconcilers.ImageRef
	labels        map[string]string
	selector      map[string]string
	desired       *flowslatest.FlowCollectorSpec
	advanced      *flowslatest.AdvancedPluginConfig
	volumes       volumes.Builder
	useStandalone bool
}

func newBuilder(info *reconcilers.Instance, desired *flowslatest.FlowCollectorSpec, name string) builder {
	imageToUse := reconcilers.MainImage
	needsPF4, _, err := info.ClusterInfo.IsOpenShiftVersionLessThan("4.15.0")
	if err == nil && needsPF4 {
		imageToUse = reconcilers.ConsolePluginCompatImage
	}

	version := helper.ExtractVersion(info.Images[imageToUse])
	advanced := helper.GetAdvancedPluginConfig(desired.ConsolePlugin.Advanced)
	return builder{
		info:     info,
		imageRef: imageToUse,
		labels: map[string]string{
			"part-of": constants.OperatorName,
			"app":     name,
			"version": helper.MaxLabelLength(version),
		},
		selector: map[string]string{
			"app": name,
		},
		desired:       desired,
		advanced:      &advanced,
		useStandalone: desired.UseStandaloneConsole(info.ClusterInfo.HasConsolePlugin()),
	}
}

func (b *builder) consolePlugin(name, displayName string) *osv1.ConsolePlugin {
	return &osv1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: osv1.ConsolePluginSpec{
			DisplayName: displayName,
			Backend: osv1.ConsolePluginBackend{
				Type: osv1.Service,
				Service: &osv1.ConsolePluginService{
					Name:      name,
					Namespace: b.info.Namespace,
					Port:      *b.advanced.Port,
					BasePath:  "/"},
			},
			Proxy: []osv1.ConsolePluginProxy{
				{
					Endpoint: osv1.ConsolePluginProxyEndpoint{
						Type: osv1.ProxyTypeService,
						Service: &osv1.ConsolePluginProxyServiceConfig{
							Name:      name,
							Namespace: b.info.Namespace,
							Port:      *b.advanced.Port}},
					Alias:         proxyAlias,
					Authorization: osv1.UserToken,
					CACertificate: "",
				},
			},
			I18n: osv1.ConsolePluginI18n{
				LoadType: osv1.Lazy,
			},
		},
	}
}

func (b *builder) serviceMonitor() *monitoringv1.ServiceMonitor {
	serverName := fmt.Sprintf("%s.%s.svc", constants.PluginName, b.info.Namespace)
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.PluginName,
			Namespace: b.info.Namespace,
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:     metricsPortName,
					Interval: "15s",
					Scheme:   ptr.To(monitoringv1.Scheme("https")),
					TLSConfig: &monitoringv1.TLSConfig{
						SafeTLSConfig: monitoringv1.SafeTLSConfig{
							ServerName: ptr.To(serverName),
							CA: monitoringv1.SecretOrConfigMap{
								ConfigMap: &corev1.ConfigMapKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "openshift-service-ca.crt",
									},
									Key: "service-ca.crt",
								},
							},
							Cert: monitoringv1.SecretOrConfigMap{
								Secret: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("%s-cert", constants.PluginName),
									},
									Key: "tls.crt",
								},
							},
							KeySecret: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: fmt.Sprintf("%s-cert", constants.PluginName),
								},
								Key: "tls.key",
							},
						},
					},
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{
					b.info.Namespace,
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": constants.PluginName,
				},
			},
		},
	}
}

func (b *builder) deployment(name, cmDigest string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: b.desired.ConsolePlugin.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: b.selector,
			},
			Template: *b.podTemplate(name, cmDigest),
		},
	}
}

func (b *builder) podTemplate(name, cmDigest string) *corev1.PodTemplateSpec {
	var sa string
	annotations := map[string]string{}
	args := []string{
		"-loglevel", b.desired.ConsolePlugin.LogLevel,
	}
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	if cmDigest != "" {
		sa = name
		annotations[constants.PodConfigurationDigest] = cmDigest

		args = append(args, "-config", filepath.Join(configPath, configFile))

		volumes = append(volumes, corev1.Volume{
			Name: configVolume,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
				},
			},
		})

		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      configVolume,
			MountPath: configPath,
			ReadOnly:  true,
		})
	}

	if !b.useStandalone {
		volumes = append(volumes, corev1.Volume{
			Name: fmt.Sprintf("%s-cert", name),
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-cert", name),
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      fmt.Sprintf("%s-cert", name),
			MountPath: "/var/serving-cert",
			ReadOnly:  true,
		})
	}

	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      b.labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            name,
				Image:           b.info.Images[b.imageRef],
				ImagePullPolicy: corev1.PullPolicy(b.desired.ConsolePlugin.ImagePullPolicy),
				Resources:       *b.desired.ConsolePlugin.Resources.DeepCopy(),
				VolumeMounts:    b.volumes.AppendMounts(volumeMounts),
				Env:             []corev1.EnvVar{constants.EnvNoHTTP2},
				Args:            args,
				SecurityContext: helper.ContainerDefaultSecurityContext(),
			}},
			Volumes:            b.volumes.AppendVolumes(volumes),
			ServiceAccountName: sa,
			NodeSelector:       b.advanced.Scheduling.NodeSelector,
			Tolerations:        b.advanced.Scheduling.Tolerations,
			Affinity:           b.advanced.Scheduling.Affinity,
			PriorityClassName:  b.advanced.Scheduling.PriorityClassName,
		},
	}
}

func (b *builder) autoScaler() *ascv2.HorizontalPodAutoscaler {
	return &ascv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.PluginName,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: ascv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: ascv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       constants.PluginName,
			},
			MinReplicas: b.desired.ConsolePlugin.Autoscaler.MinReplicas,
			MaxReplicas: b.desired.ConsolePlugin.Autoscaler.MaxReplicas,
			Metrics:     b.desired.ConsolePlugin.Autoscaler.Metrics,
		},
	}
}

func (b *builder) mainService(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
			Annotations: map[string]string{
				constants.OpenShiftCertificateAnnotation: fmt.Sprintf("%s-cert", name),
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: b.selector,
			Ports: []corev1.ServicePort{{
				Port:     *b.advanced.Port,
				Protocol: corev1.ProtocolTCP,
				// Some Kubernetes versions might automatically set TargetPort to Port. We need to
				// explicitly set it here so the reconcile loop verifies that the owned service
				// is equal as the desired service
				TargetPort: intstr.FromInt32(*b.advanced.Port),
			}},
		},
	}
}

func (b *builder) metricsService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsSvcName,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: b.selector,
			Ports: []corev1.ServicePort{{
				Port:     metricsPort,
				Protocol: corev1.ProtocolTCP,
				Name:     metricsPortName,
				// Some Kubernetes versions might automatically set TargetPort to Port. We need to
				// explicitly set it here so the reconcile loop verifies that the owned service
				// is equal as the desired service
				TargetPort: intstr.FromInt32(metricsPort),
			}},
		},
	}
}

func (b *builder) getLokiConfig() (cfg.LokiConfig, error) {
	if !b.desired.UseLoki() {
		// Empty config/URL will disable Loki in the console plugin
		return cfg.LokiConfig{}, nil
	}
	lk := b.info.Loki
	lokiLabels, err := loki.GetLabels(b.desired)
	if err != nil {
		return cfg.LokiConfig{}, err
	}
	lconf := cfg.LokiConfig{
		URL:              lk.QuerierURL,
		Labels:           lokiLabels,
		Timeout:          api.Duration{Duration: 30 * time.Second},
		TenantID:         lk.TenantID,
		ForwardUserToken: lk.UseForwardToken(),
	}
	if lk.QuerierURL != lk.StatusURL {
		lconf.StatusURL = lk.StatusURL
	}
	if b.desired.Loki.ReadTimeout != nil {
		lconf.Timeout = api.Duration{Duration: b.desired.Loki.ReadTimeout.Duration}
	}
	if lk.TLS.Enable {
		if lk.TLS.InsecureSkipVerify {
			lconf.SkipTLS = true
		} else {
			caPath := b.volumes.AddCACertificate(&lk.TLS, "loki-certs")
			if caPath != "" {
				lconf.CAPath = caPath
			}
		}
	}
	if lk.StatusTLS.Enable {
		if lk.StatusTLS.InsecureSkipVerify {
			lconf.StatusSkipTLS = true
		} else {
			statusCaPath, userCertPath, userKeyPath := b.volumes.AddMutualTLSCertificates(&lk.StatusTLS, "loki-status-certs")
			if statusCaPath != "" {
				lconf.StatusCAPath = statusCaPath
			}
			if userCertPath != "" && userKeyPath != "" {
				lconf.StatusUserCertPath = userCertPath
				lconf.StatusUserKeyPath = userKeyPath
			}
		}
	}
	if lk.UseHostToken() {
		lconf.TokenPath = b.volumes.AddToken(constants.PluginName)
	}
	return lconf, nil
}

func (b *builder) getPromConfig(ctx context.Context) cfg.PrometheusConfig {
	if !b.desired.UsePrometheus() {
		return cfg.PrometheusConfig{}
	}

	// Default config = manual
	tls := b.desired.Prometheus.Querier.Manual.TLS
	config := cfg.PrometheusConfig{
		URL:              b.desired.Prometheus.Querier.Manual.URL,
		ForwardUserToken: b.desired.Prometheus.Querier.Manual.ForwardUserToken,
		Timeout:          api.Duration{Duration: 30 * time.Second},
	}
	if b.desired.Prometheus.Querier.Timeout != nil {
		config.Timeout = api.Duration{Duration: b.desired.Prometheus.Querier.Timeout.Duration}
	}
	if b.desired.Prometheus.Querier.Mode == "" || b.desired.Prometheus.Querier.Mode == flowslatest.PromModeAuto {
		if b.info.ClusterInfo.IsOpenShift() {
			// NB: trailing dot (...local.:9091) is a DNS optimization for exact name match without extra search
			config.URL = "https://thanos-querier.openshift-monitoring.svc.cluster.local.:9091/"    // requires cluster-monitoringv-view cluster role
			config.DevURL = "https://thanos-querier.openshift-monitoring.svc.cluster.local.:9092/" // restricted to a particular namespace
			config.ForwardUserToken = true
			tls = flowslatest.ClientTLS{
				Enable: true,
				CACert: flowslatest.CertificateReference{
					Type:     flowslatest.RefTypeConfigMap,
					Name:     "openshift-service-ca.crt",
					CertFile: "service-ca.crt",
				},
			}
		} else {
			log.FromContext(ctx).Info("Could not configure Prometheus querier automatically. Using manual configuration.")
		}
	} else if b.desired.Prometheus.Querier.Mode == flowslatest.PromModeManual {
		config.AlertManager = cfg.AlertManagerConfig{
			URL:     b.desired.Prometheus.Querier.Manual.AlertManager.URL,
			SkipTLS: b.desired.Prometheus.Querier.Manual.AlertManager.TLS.InsecureSkipVerify,
		}
		if b.desired.Prometheus.Querier.Manual.AlertManager.TLS.Enable {
			config.AlertManager.CAPath = b.volumes.AddCACertificate(&tls, "prom-am-certs")
		}
	}

	if tls.Enable {
		if tls.InsecureSkipVerify {
			config.SkipTLS = true
		} else {
			caPath := b.volumes.AddCACertificate(&tls, "prom-certs")
			if caPath != "" {
				config.CAPath = caPath
			}
		}
	}

	config.TokenPath = b.volumes.AddToken(constants.PluginName)

	includeList := b.desired.GetIncludeList()
	allMetrics := metrics.GetDefinitions(b.desired, true)
	for i := range allMetrics {
		mSpec := allMetrics[i].Spec
		enabled := slices.Contains(includeList, mSpec.MetricName)
		config.Metrics = append(config.Metrics, cfg.MetricInfo{
			Enabled:    enabled,
			Name:       "netobserv_" + mSpec.MetricName,
			Type:       string(mSpec.Type),
			ValueField: mSpec.ValueField,
			Direction:  string(mSpec.Direction),
			Labels:     mSpec.Labels,
		})
	}

	return config
}

// nolint:cyclop	// no real complexity here, just long boilerplate
func (b *builder) setFrontendConfig(fconf *cfg.FrontendConfig, metrics []cfg.MetricInfo) (string, error) {
	if b.desired.Agent.EBPF.IsPktDropEnabled() || b.desired.Agent.EBPF.IsNetworkEventsEnabled() {
		fconf.Features = append(fconf.Features, "pktDrop")
	}

	if b.desired.Agent.EBPF.IsDNSTrackingEnabled() {
		fconf.Features = append(fconf.Features, "dnsTracking")
	}

	if b.desired.Agent.EBPF.IsFlowRTTEnabled() {
		fconf.Features = append(fconf.Features, "flowRTT")
	}

	if b.desired.Agent.EBPF.IsNetworkEventsEnabled() {
		fconf.Features = append(fconf.Features, "networkEvents")
	}

	if b.desired.Agent.EBPF.IsPacketTranslationEnabled() {
		fconf.Features = append(fconf.Features, "packetTranslation")
	}

	if b.desired.Agent.EBPF.IsUDNMappingEnabled() {
		fconf.Features = append(fconf.Features, "udnMapping")
	}

	if b.desired.Agent.EBPF.IsUDNMappingEnabled() || len(b.desired.GetSecondaryIndexes()) > 0 {
		fconf.Features = append(fconf.Features, "multiNetworks")
	}

	if b.desired.Agent.EBPF.IsIPSecEnabled() {
		fconf.Features = append(fconf.Features, "ipsec")
	}

	if b.desired.Agent.EBPF.IsTLSTrackingEnabled() {
		fconf.Features = append(fconf.Features, "tlsTracking")
	}

	fconf.RecordTypes = helper.GetRecordTypes(&b.desired.Processor)
	fconf.PortNaming = b.desired.ConsolePlugin.PortNaming
	fconf.QuickFilters = b.desired.ConsolePlugin.QuickFilters
	fconf.AlertNamespaces = []string{b.info.Namespace}
	fconf.Sampling = b.desired.GetSampling()
	if b.desired.Processor.IsMultiClusterEnabled() {
		fconf.Features = append(fconf.Features, "multiCluster")
	}
	if b.desired.Processor.IsZoneEnabled() {
		fconf.Features = append(fconf.Features, "zones")
	}
	if b.desired.Processor.IsSubnetLabelsEnabled() {
		fconf.Features = append(fconf.Features, "subnetLabels")
	}

	// Add health rules metadata for frontend
	fconf.RecordingAnnotations = b.getHealthRecordingAnnotations()

	// Filter-out disabled scopes
	var scopes []cfg.ScopeConfig
	var warnings []string
	for i := range fconf.Scopes {
		scope := &fconf.Scopes[i]
		if len(scope.Feature) == 0 || slices.Contains(fconf.Features, scope.Feature) {
			if b.desired.UseLoki() {
				scopes = append(scopes, *scope)
			} else if valid, warning := isScopeValidForMetrics(scope, metrics); valid {
				scopes = append(scopes, *scope)
			} else if scope.ID != "resource" { // don't trigger warning for "resource" scope, it's a known fact it won't be available
				warnings = append(warnings, warning)
			}
		}
	}
	if b.desired.UseLoki() {
		fconf.Scopes = scopes
	} else {
		fconf.Scopes = filterScopeGroupsForMetrics(scopes, metrics)
	}

	return strings.Join(warnings, "; "), nil
}

func isScopeValidForMetrics(scope *cfg.ScopeConfig, metrics []cfg.MetricInfo) (bool, string) {
	var bestMatch *cfg.MetricInfo
	var bestMatchMissingLabels []string
	var candidates []string
	for i := range metrics {
		if metrics[i].ValueField == "Bytes" || metrics[i].ValueField == "Packets" {
			missing := missingLabels(scope.Labels, &metrics[i])
			if len(missing) == 0 {
				if !metrics[i].Enabled {
					candidates = append(candidates, metrics[i].Name)
				} else {
					return true, ""
				}
			} else if bestMatch == nil {
				bestMatch = &metrics[i]
				bestMatchMissingLabels = missing
			}
		}
	}
	if len(candidates) > 0 {
		return false, fmt.Sprintf("Scope %s invalid for metrics (candidates: %s)", scope.ID, strings.Join(candidates, ", "))
	}
	if bestMatch == nil {
		return false, fmt.Sprintf("Scope %s invalid for metrics (no best match)", scope.ID)
	}
	return false, fmt.Sprintf("Scope %s invalid for metrics (best match: %s; missing labels: %s)", scope.ID, bestMatch.Name, strings.Join(bestMatchMissingLabels, ", "))
}

func missingLabels(labels []string, metric *cfg.MetricInfo) []string {
	var missing []string
	for _, label := range labels {
		if !slices.Contains(metric.Labels, label) {
			missing = append(missing, label)
		}
	}
	return missing
}

func filterScopeGroupsForMetrics(scopes []cfg.ScopeConfig, metrics []cfg.MetricInfo) []cfg.ScopeConfig {
	labelsPerScope := make(map[string][]string)
	for i := range scopes {
		labelsPerScope[scopes[i].ID+"s"] = scopes[i].Labels
	}
	for i := range scopes {
		scope := &scopes[i]
		newGroups := []string{}
	NextGroup:
		for _, group := range scope.Groups {
			neededLabels := scope.Labels
			// Group is like "hosts+networks"
			parts := strings.Split(group, "+")
			for _, part := range parts {
				if labels, isValid := labelsPerScope[part]; isValid {
					neededLabels = append(neededLabels, labels...)
				} else {
					// Not a valid scope => reject that group
					continue NextGroup
				}
			}
			// Now, verify that all needed labels (for scope AND groups) are ok with metrics
			for i := range metrics {
				if metrics[i].Enabled && (metrics[i].ValueField == "Bytes" || metrics[i].ValueField == "Packets") {
					missing := missingLabels(neededLabels, &metrics[i])
					if len(missing) == 0 {
						// Valid group
						newGroups = append(newGroups, group)
						continue NextGroup
					}
				}
			}
		}
		scope.Groups = newGroups
	}
	return scopes
}

func (b *builder) getHealthRecordingAnnotations() map[string]map[string]string {
	annotsPerRecording := make(map[string]map[string]string)
	healthRules, _ := alerts.BuildHealthRules(b.desired)
	for _, r := range healthRules {
		rname := r.RecordingName()
		if rname != "" {
			if a, _ := r.GetAnnotations(); len(a) > 0 {
				annotsPerRecording[rname] = a
			}
		}
	}
	return annotsPerRecording
}

// returns a configmap with a digest of its configuration contents, which will be used to
// detect any configuration change. externalRecordingAnnotations is optional (e.g. nil in tests);
// when non-empty, those annotations are merged into the frontend config (from PrometheusRules).
func (b *builder) configMap(ctx context.Context, externalRecordingAnnotations map[string]map[string]string, lokiStatus status.ComponentStatus) (*corev1.ConfigMap, string, error) {
	config := cfg.PluginConfig{
		Server: cfg.ServerConfig{
			Port: int(*b.advanced.Port),
		},
	}
	if b.useStandalone {
		config.Server.AuthCheck = "none"
	} else {
		config.Server.CertPath = "/var/serving-cert/tls.crt"
		config.Server.KeyPath = "/var/serving-cert/tls.key"
	}

	// configure loki
	var err error
	config.Loki, err = b.getLokiConfig()
	if lokiStatus.Status != status.StatusUnknown {
		config.Loki.StatusURL = ""
		if lokiStatus.Status == status.StatusReady {
			config.Loki.Status = "ready"
		} else {
			config.Loki.Status = "pending"
		}
	}
	if err != nil {
		return nil, "", err
	}

	// configure prometheus
	config.Prometheus = b.getPromConfig(ctx)

	// configure frontend from embedded static file
	config.Frontend, err = cfg.GetStaticFrontendConfig()
	if err != nil {
		return nil, "", err
	}
	warning, err := b.setFrontendConfig(&config.Frontend, config.Prometheus.Metrics)
	if err != nil {
		return nil, "", err
	}
	// TODO: with NETOBSERV-2375 => add DEGRADED to console status instead of this log
	if warning != "" {
		log.FromContext(ctx).Info("Frontend config DEGRADED", "message", warning)
	}

	for k, v := range externalRecordingAnnotations {
		if config.Frontend.RecordingAnnotations == nil {
			config.Frontend.RecordingAnnotations = make(map[string]map[string]string)
		}
		config.Frontend.RecordingAnnotations[k] = v
	}

	var configStr string
	bs, err := yaml.Marshal(config)
	if err == nil {
		configStr = string(bs)
	} else {
		return nil, "", err
	}

	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Data: map[string]string{
			configFile: configStr,
		},
	}
	hasher := fnv.New64a()
	_, err = hasher.Write([]byte(configStr))
	if err != nil {
		return nil, "", err
	}
	digest := strconv.FormatUint(hasher.Sum64(), 36)
	return &configMap, digest, nil
}

func (b *builder) serviceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"app": name,
			},
		},
	}
}
