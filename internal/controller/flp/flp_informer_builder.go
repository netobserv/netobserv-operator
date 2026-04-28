package flp

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
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
	config := b.desired.Processor.Informers
	if config == nil {
		config = &flowslatest.FlowCollectorInformers{}
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
	processorPort := int32(k8scachePort)
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

	// Build container args
	args := []string{
		fmt.Sprintf("--processor-selector=%s", processorSelector),
		fmt.Sprintf("--processor-port=%d", processorPort),
		fmt.Sprintf("--resync-interval=%d", resyncInterval),
		fmt.Sprintf("--batch-size=%d", batchSize),
		fmt.Sprintf("--send-timeout=%d", sendTimeout),
		fmt.Sprintf("--update-buffer-size=%d", updateBufferSize),
		fmt.Sprintf("--log-level=%s", b.desired.Processor.LogLevel),
	}

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
		Ports:           ports,
		Resources:       resources,
		LivenessProbe:   livenessProbe,
		ReadinessProbe:  readinessProbe,
		SecurityContext: helper.ContainerDefaultSecurityContext(),
	}

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
