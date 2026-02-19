package flp

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	configVolume            = "config-volume"
	configPath              = "/etc/flowlogs-pipeline"
	configFile              = "config.json"
	healthPortName          = "health"
	prometheusPortName      = "prometheus"
	profilePortName         = "pprof"
	healthTimeoutSeconds    = 5
	livenessPeriodSeconds   = 10
	startupFailureThreshold = 5
	startupPeriodSeconds    = 10
)

func newGRPCPipeline(desired *flowslatest.FlowCollectorSpec, volumes *volumes.Builder, isOpenShift bool) config.PipelineBuilderStage {
	adv := helper.GetAdvancedProcessorConfig(desired)
	cfg := api.IngestGRPCProto{Port: int(*adv.Port)}
	if desired.DeploymentModel == flowslatest.DeploymentModelService {
		serverCert, clientCA := helper.GetServiceServerTLSConfig(desired.Processor.Service, monoCertSecretName, isOpenShift)
		if serverCert != nil {
			// Communication from agents uses TLS: set up server certificate
			certPath, keyPath := volumes.AddCertificate(serverCert, "svc-certs")
			cfg.CertPath = certPath
			cfg.KeyPath = keyPath
			if clientCA != nil {
				cfg.ClientCAPath = volumes.AddVolume(clientCA, "netobserv-ca")
			}
		}
	}
	return config.NewGRPCPipeline("grpc", cfg)
}

func newKafkaPipeline(desired *flowslatest.FlowCollectorSpec, volumes *volumes.Builder) config.PipelineBuilderStage {
	return config.NewKafkaPipeline("kafka-read", api.IngestKafka{
		Brokers:           []string{desired.Kafka.Address},
		Topic:             desired.Kafka.Topic,
		GroupID:           constants.FLPName, // Without groupid, each message is delivered to each consumers
		Decoder:           api.Decoder{Type: "protobuf"},
		TLS:               getClientTLS(&desired.Kafka.TLS, "kafka-cert", volumes),
		SASL:              getSASL(&desired.Kafka.SASL, "kafka-ingest", volumes),
		PullQueueCapacity: desired.Processor.KafkaConsumerQueueCapacity,
		PullMaxBytes:      desired.Processor.KafkaConsumerBatchSize,
	})
}

func getPromTLS(desired *flowslatest.FlowCollectorSpec, serviceName string) (*flowslatest.CertificateReference, error) {
	var promTLS *flowslatest.CertificateReference
	switch desired.Processor.Metrics.Server.TLS.Type {
	case flowslatest.TLSProvided:
		promTLS = desired.Processor.Metrics.Server.TLS.Provided
		if promTLS == nil {
			return nil, fmt.Errorf("processor TLS configuration set to provided but none is provided")
		}
	case flowslatest.TLSAuto:
		promTLS = &flowslatest.CertificateReference{
			Type:     "secret",
			Name:     serviceName,
			CertFile: "tls.crt",
			CertKey:  "tls.key",
		}
	case flowslatest.TLSDisabled, flowslatest.TLSAutoMTLS:
		// nothing to do there
	}
	return promTLS, nil
}

type flowNetworkType int

const (
	hostNetwork flowNetworkType = iota
	hostPort
	svc
	pull
)

func podTemplate(
	appName, version, imageName, cmName string,
	desired *flowslatest.FlowCollectorSpec,
	vols *volumes.Builder,
	netType flowNetworkType,
	annotations map[string]string,
) corev1.PodTemplateSpec {
	advancedConfig := helper.GetAdvancedProcessorConfig(desired)
	var ports []corev1.ContainerPort
	switch netType {
	case hostNetwork, hostPort:
		ports = []corev1.ContainerPort{{
			Name:          constants.FLPPortName,
			HostPort:      *advancedConfig.Port,
			ContainerPort: *advancedConfig.Port,
			Protocol:      corev1.ProtocolTCP,
		}}
	case svc:
		ports = []corev1.ContainerPort{{
			Name:          constants.FLPPortName,
			ContainerPort: *advancedConfig.Port,
			Protocol:      corev1.ProtocolTCP,
		}}
	case pull:
		// does not listen for flows => no port
	}
	ports = append(ports, corev1.ContainerPort{
		Name:          healthPortName,
		ContainerPort: *advancedConfig.HealthPort,
	})
	ports = append(ports, corev1.ContainerPort{
		Name:          prometheusPortName,
		ContainerPort: desired.Processor.GetMetricsPort(),
	})

	if advancedConfig.ProfilePort != nil {
		ports = append(ports, corev1.ContainerPort{
			Name:          profilePortName,
			ContainerPort: *advancedConfig.ProfilePort,
			Protocol:      corev1.ProtocolTCP,
		})
	}

	volumeMounts := vols.AppendMounts([]corev1.VolumeMount{{
		MountPath: configPath,
		Name:      configVolume,
	}})
	volumes := vols.AppendVolumes([]corev1.Volume{{
		Name: configVolume,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmName,
				},
			},
		},
	}})

	var envs []corev1.EnvVar
	// we need to sort env map to keep idempotency,
	// as equal maps could be iterated in different order
	for _, pair := range helper.KeySorted(advancedConfig.Env) {
		envs = append(envs, corev1.EnvVar{Name: pair[0], Value: pair[1]})
	}
	envs = append(envs, constants.EnvNoHTTP2)

	envs = helper.EnvFromReqsLimits(envs, &desired.Processor.Resources)

	container := corev1.Container{
		Name:            constants.FLPName,
		Image:           imageName,
		ImagePullPolicy: corev1.PullPolicy(desired.Processor.ImagePullPolicy),
		Args:            []string{fmt.Sprintf(`--config=%s/%s`, configPath, configFile)},
		Resources:       *desired.Processor.Resources.DeepCopy(),
		VolumeMounts:    volumeMounts,
		Ports:           ports,
		Env:             envs,
		SecurityContext: helper.ContainerDefaultSecurityContext(),
	}
	if *advancedConfig.EnableKubeProbes {
		container.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/live",
					Port: intstr.FromString(healthPortName),
				},
			},
			TimeoutSeconds: healthTimeoutSeconds,
			PeriodSeconds:  livenessPeriodSeconds,
		}
		container.StartupProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: "/ready",
					Port: intstr.FromString(healthPortName),
				},
			},
			TimeoutSeconds:   healthTimeoutSeconds,
			PeriodSeconds:    startupPeriodSeconds,
			FailureThreshold: startupFailureThreshold,
		}
	}
	dnsPolicy := corev1.DNSClusterFirst
	if netType == hostNetwork {
		dnsPolicy = corev1.DNSClusterFirstWithHostNet
	}
	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/scrape_port"] = fmt.Sprint(desired.Processor.GetMetricsPort())
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     appName,
				"version": version,
			},
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Volumes:            volumes,
			Containers:         []corev1.Container{container},
			ServiceAccountName: appName,
			HostNetwork:        netType == hostNetwork,
			DNSPolicy:          dnsPolicy,
			NodeSelector:       advancedConfig.Scheduling.NodeSelector,
			Tolerations:        advancedConfig.Scheduling.Tolerations,
			Affinity:           advancedConfig.Scheduling.Affinity,
			PriorityClassName:  advancedConfig.Scheduling.PriorityClassName,
		},
	}
}

// returns a configmap with a digest of its configuration contents, which will be used to
// detect any configuration change
func configMap(name, namespace, data, appName string) (*corev1.ConfigMap, string, error) {
	configMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     appName,
			},
		},
		Data: map[string]string{
			configFile: data,
		},
	}
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(data))
	digest := strconv.FormatUint(hasher.Sum64(), 36)
	return &configMap, digest, nil
}

func metricsSettings(desired *flowslatest.FlowCollectorSpec, vol *volumes.Builder, promTLS *flowslatest.CertificateReference) config.MetricsSettings {
	metricsSettings := config.MetricsSettings{
		PromConnectionInfo: api.PromConnectionInfo{
			Port: int(desired.Processor.GetMetricsPort()),
		},
		Prefix:  "netobserv_",
		NoPanic: true,
	}
	if desired.Processor.Metrics.Server.TLS.Type != flowslatest.TLSDisabled {
		cert, key := vol.AddCertificate(promTLS, "prom-certs")
		if cert != "" && key != "" {
			metricsSettings.TLS = &api.PromTLSConf{
				CertPath: cert,
				KeyPath:  key,
			}
		}
	}
	return metricsSettings
}

func getJSONConfigs(desired *flowslatest.FlowCollectorSpec, vol *volumes.Builder, promTLS *flowslatest.CertificateReference, pipeline *PipelineBuilder, dynCMName string) (string, string, error) {
	metricsSettings := metricsSettings(desired, vol, promTLS)
	advancedConfig := helper.GetAdvancedProcessorConfig(desired)
	static, dynamic := pipeline.GetSplitStageParams()
	config := map[string]interface{}{
		"log-level": desired.Processor.LogLevel,
		"health": map[string]interface{}{
			"port": *advancedConfig.HealthPort,
		},
		"pipeline":        pipeline.GetStages(),
		"parameters":      static,
		"metricsSettings": metricsSettings,
		"dynamicParameters": config.DynamicParameters{
			Namespace: desired.Namespace,
			Name:      dynCMName,
			FileName:  configFile,
		},
	}
	if advancedConfig.ProfilePort != nil {
		config["profile"] = map[string]interface{}{
			"port": *advancedConfig.ProfilePort,
		}
	}
	jsonStatic, err := json.Marshal(config)
	if err != nil {
		return "", "", err
	}

	config = map[string]interface{}{
		"parameters": dynamic,
	}
	jsonDynamic, err := json.Marshal(config)
	if err != nil {
		return "", "", err
	}
	return string(jsonStatic), string(jsonDynamic), nil
}

func promService(desired *flowslatest.FlowCollectorSpec, svcName, namespace, appLabel string) *corev1.Service {
	port := desired.Processor.GetMetricsPort()
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     appLabel,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": appLabel},
			Ports: []corev1.ServicePort{{
				Name:     prometheusPortName,
				Port:     port,
				Protocol: corev1.ProtocolTCP,
				// Some Kubernetes versions might automatically set TargetPort to Port. We need to
				// explicitly set it here so the reconcile loop verifies that the owned service
				// is equal as the desired service
				TargetPort: intstr.FromInt32(port),
			}},
		},
	}
	if desired.Processor.Metrics.Server.TLS.Type == flowslatest.TLSAuto {
		svc.ObjectMeta.Annotations = map[string]string{
			constants.OpenShiftCertificateAnnotation: svcName,
		}
	}
	return &svc
}

func serviceMonitor(desired *flowslatest.FlowCollectorSpec, smName, svcName, namespace, appLabel, version string, isDownstream, useEndpointSlices bool) *monitoringv1.ServiceMonitor {
	serverName := fmt.Sprintf("%s.%s.svc", svcName, namespace)
	scheme, smTLS := helper.GetServiceMonitorTLSConfig(&desired.Processor.Metrics.Server.TLS, serverName, isDownstream)
	var sdRole *monitoringv1.ServiceDiscoveryRole
	if useEndpointSlices {
		sdRole = ptr.To(monitoringv1.EndpointSliceRole)
	}
	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      smName,
			Namespace: namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     appLabel,
				"version": version,
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			ServiceDiscoveryRole: sdRole,
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:        prometheusPortName,
					Interval:    "15s",
					Scheme:      &scheme,
					TLSConfig:   smTLS,
					HonorLabels: true,
					// Relabel for Thanos multi-tenant endpoint, which requires having the namespace label
					MetricRelabelConfigs: []monitoringv1.RelabelConfig{
						{
							SourceLabels: []monitoringv1.LabelName{"__name__", "DstK8S_Namespace"},
							Separator:    ptr.To("@"),
							Regex:        "netobserv_(workload|namespace)_ingress_.*@(.*)",
							Replacement:  ptr.To("${2}"),
							TargetLabel:  "namespace",
							Action:       "replace",
						},
						{
							SourceLabels: []monitoringv1.LabelName{"__name__", "SrcK8S_Namespace"},
							Separator:    ptr.To("@"),
							Regex:        "netobserv_(workload|namespace)_(egress|flows|drop|rtt|dns)_.*@(.*)",
							Replacement:  ptr.To("${3}"),
							TargetLabel:  "namespace",
							Action:       "replace",
						},
					},
				},
			},
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
		},
	}
}

func prometheusRule(rules []monitoringv1.Rule, ruleName, namespace, appLabel, version string) *monitoringv1.PrometheusRule {
	flpPrometheusRuleObject := monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name: ruleName,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     appLabel,
				"version": version,
			},
			Namespace: namespace,
		},
		Spec: monitoringv1.PrometheusRuleSpec{
			Groups: []monitoringv1.RuleGroup{
				{
					Name:  "NetObservFlowLogsPipeline",
					Rules: rules,
				},
			},
		},
	}
	return &flpPrometheusRuleObject
}
