package flp

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
)

const (
	k8sCacheServiceName    = "flowlogs-pipeline-k8scache"
	k8sCacheCertSecretName = k8sCacheServiceName + "-cert"
)

type informerBuilder struct {
	*reconcilers.Instance
	desired *flowslatest.FlowCollectorSpec
}

func newInformerBuilder(info *reconcilers.Instance, desired *flowslatest.FlowCollectorSpec) informerBuilder {
	return informerBuilder{
		Instance: info,
		desired:  desired,
	}
}

func (b *informerBuilder) deployment() (*appsv1.Deployment, error) {
	// Get configuration from FlowCollector spec
	config := b.desired.Processor.InformerCacheProxy
	if config == nil {
		config = &flowslatest.FlowCollectorInformerCacheProxy{}
	}

	// Replicas: default 2 for HA
	replicas := int32(2)
	if config.Replicas != nil {
		replicas = *config.Replicas
	}

	// Resources: use configured or defaults
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("128Mi"),
			corev1.ResourceCPU:    resource.MustParse("50m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("200m"),
		},
	}
	if config.Resources.Requests != nil || config.Resources.Limits != nil {
		resources = *config.Resources.DeepCopy()
	}

	// Advanced config with defaults
	resyncInterval := 60
	batchSize := 100
	sendTimeout := 10
	updateBufferSize := 100
	processorPort := flowslatest.DefaultK8sCachePort
	if config.Advanced != nil {
		if config.Advanced.ResyncInterval != nil {
			resyncInterval = *config.Advanced.ResyncInterval
		}
		if config.Advanced.BatchSize != nil {
			batchSize = *config.Advanced.BatchSize
		}
		if config.Advanced.SendTimeout != nil {
			sendTimeout = *config.Advanced.SendTimeout
		}
		if config.Advanced.UpdateBufferSize != nil {
			updateBufferSize = *config.Advanced.UpdateBufferSize
		}
		if config.Advanced.ProcessorPort != nil {
			processorPort = *config.Advanced.ProcessorPort
		}
	}

	version := helper.MaxLabelLength(helper.ExtractVersion(b.Images[reconcilers.MainImage]))

	// Determine the correct processor selector based on deployment model
	processorSelector := "app=flowlogs-pipeline"
	if b.desired.UseKafka() {
		processorSelector = "app=flowlogs-pipeline-transformer"
	}

	// Create volume builder for TLS certificates
	var vols volumes.Builder

	// Build container args
	args := []string{
		fmt.Sprintf("--processor-selector=%s", processorSelector),
		fmt.Sprintf("--processor-port=%d", processorPort),
		fmt.Sprintf("--resync-interval=%d", resyncInterval),
		fmt.Sprintf("--batch-size=%d", batchSize),
		fmt.Sprintf("--send-timeout=%d", sendTimeout),
		fmt.Sprintf("--update-buffer-size=%d", updateBufferSize),
		fmt.Sprintf("--log-level=%s", b.desired.Processor.LogLevel),
		// Kubernetes informer configuration - must match processor settings
		// TrackedKinds: hardcoded to match flp_pipeline_builder.go line 205
		"--tracked-kinds=ReplicaSet,Deployment,Gateway",
		// ManagedCNI: hardcoded to match flowlogs-pipeline default (config.go line 32)
		// This ensures informers extract the same node IPs (e.g., OVN mp0) as processors
		"--managed-cni=ovn",
	}

	// Propagate secondary networks config to informers (must match processor settings)
	// Each index set is comma-separated, sets are semicolon-separated to preserve OR semantics
	var snGroups []string
	for _, sn := range b.desired.GetSecondaryIndexes() {
		var indexes []string
		for _, index := range sn.Index {
			indexes = append(indexes, strings.ToLower(string(index)))
		}
		if len(indexes) > 0 {
			snGroups = append(snGroups, strings.Join(indexes, ","))
		}
	}
	if b.desired.Agent.EBPF.IsUDNMappingEnabled() {
		snGroups = append(snGroups, "udn")
	}
	if len(snGroups) > 0 {
		args = append(args, fmt.Sprintf("--secondary-networks=%s", strings.Join(snGroups, ";")))
	}

	// Add TLS configuration if enabled
	b.addTLSArgs(&args, &vols, config)

	// Define container ports
	ports := []corev1.ContainerPort{
		{
			Name:          "grpc",
			ContainerPort: processorPort,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "health",
			ContainerPort: 8080,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "metrics",
			ContainerPort: 9091,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	// Health probes - using HTTP endpoints
	livenessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/healthz",
				Port: intstr.FromInt(8080),
			},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		FailureThreshold:    3,
	}

	readinessProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ready",
				Port: intstr.FromInt(8080),
			},
		},
		InitialDelaySeconds: 5,
		PeriodSeconds:       5,
		TimeoutSeconds:      3,
		FailureThreshold:    2,
	}

	container := corev1.Container{
		Name:            informerName,
		Image:           b.Images[reconcilers.MainImage],
		ImagePullPolicy: corev1.PullPolicy(b.desired.Processor.ImagePullPolicy),
		Command:         []string{"/app/flp-informers"},
		Args:            args,
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				},
			},
			{
				Name: "POD_NAME",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				},
			},
		},
		Ports:                    ports,
		VolumeMounts:             (&vols).GetMounts(),
		Resources:                resources,
		LivenessProbe:            livenessProbe,
		ReadinessProbe:           readinessProbe,
		SecurityContext:          helper.ContainerDefaultSecurityContext(),
		TerminationMessagePolicy: corev1.TerminationMessageFallbackToLogsOnError,
	}

	// Get processor advanced scheduling configuration to apply to informers
	// Informers should be scheduled on the same nodes as processors for optimal communication
	advancedConfig := helper.GetAdvancedProcessorConfig(b.desired)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      informerName,
			Namespace: b.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     informerName,
				"version": version,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": informerName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"part-of": constants.OperatorName,
						"app":     informerName,
						"version": version,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: informerName,
					Containers:         []corev1.Container{container},
					Volumes:            (&vols).GetVolumes(),
					// Apply processor scheduling constraints to informers
					NodeSelector:      advancedConfig.Scheduling.NodeSelector,
					Tolerations:       advancedConfig.Scheduling.Tolerations,
					Affinity:          advancedConfig.Scheduling.Affinity,
					PriorityClassName: advancedConfig.Scheduling.PriorityClassName,
				},
			},
		},
	}, nil
}

func (b *informerBuilder) serviceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      informerName,
			Namespace: b.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     informerName,
			},
		},
	}
}

// service creates a dedicated k8scache Service that targets processor (FLP) pods.
// This service is always separate from the main FLP service so that its certificates
// are driven entirely by the informerCacheProxy config, without special cases per deployment model.
func (b *informerBuilder) service() *corev1.Service {
	processorApp := monoName
	if b.desired.UseKafka() {
		processorApp = transfoName
	}
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sCacheServiceName,
			Namespace: b.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     k8sCacheServiceName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": processorApp},
			Ports: []corev1.ServicePort{
				{
					Name:       "k8scache",
					Port:       b.desired.Processor.GetK8sCachePort(),
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt32(b.desired.Processor.GetK8sCachePort()),
				},
			},
		},
	}
	if b.desired.Processor.InformerCacheProxy.UsesOpenShiftServiceCA(b.ClusterInfo.IsOpenShift()) {
		svc.Annotations = map[string]string{
			constants.OpenShiftCertificateAnnotation: k8sCacheCertSecretName,
		}
	}
	return &svc
}

// addTLSArgs configures TLS arguments for informers client
func (b *informerBuilder) addTLSArgs(args *[]string, vols *volumes.Builder, proxy *flowslatest.FlowCollectorInformerCacheProxy) {
	svcConfig := helper.InformerTLSAsServiceConfig(proxy)
	caFile, clientCert := helper.GetServiceClientTLSConfig(svcConfig, "flowlogs-pipeline-informers-k8scache-client-cert", b.ClusterInfo.IsOpenShift())

	// Enable TLS and add server CA for verification
	if caFile != nil {
		*args = append(*args, "--tls-enabled=true")
		caPath := vols.AddVolume(caFile, "k8scache-server-ca")
		*args = append(*args, fmt.Sprintf("--tls-ca-path=%s", caPath))
	}

	// Add client certificate if provided (mTLS)
	if clientCert != nil {
		certPath, keyPath := vols.AddCertificate(clientCert, "k8scache-client-certs")
		*args = append(*args,
			fmt.Sprintf("--tls-cert-path=%s", certPath),
			fmt.Sprintf("--tls-key-path=%s", keyPath),
		)
	}

	if caFile != nil || clientCert != nil {
		// Set TLS server name for certificate verification
		// Informers connect to processor pods by IP, but TLS certificates contain DNS names.
		// k8scache always has its own dedicated service, regardless of deployment model
		serviceName := fmt.Sprintf("%s.%s.svc", k8sCacheServiceName, b.Namespace)
		*args = append(*args, fmt.Sprintf("--tls-server-name=%s", serviceName))
	}
}
