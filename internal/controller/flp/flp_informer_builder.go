package flp

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	var replicas int32 = 1
	version := helper.MaxLabelLength(helper.ExtractVersion(b.Images[reconcilers.MainImage]))

	// Determine the correct processor selector based on deployment model
	processorSelector := "app=flowlogs-pipeline"
	if b.desired.UseKafka() {
		processorSelector = "app=flowlogs-pipeline-transformer"
	}

	container := corev1.Container{
		Name:            informerName,
		Image:           b.Images[reconcilers.MainImage],
		ImagePullPolicy: corev1.PullPolicy(b.desired.Processor.ImagePullPolicy),
		Command:         []string{"/app/flp-informers"},
		Args: []string{
			fmt.Sprintf("--processor-selector=%s", processorSelector),
			"--processor-port=9090",
			"--resync-interval=60",
			fmt.Sprintf("--log-level=%s", b.desired.Processor.LogLevel),
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
		},
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("256Mi"),
				corev1.ResourceCPU:    resource.MustParse("200m"),
			},
		},
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
