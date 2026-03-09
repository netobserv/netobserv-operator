package loki

import (
	"fmt"
	"hash/fnv"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	cfg "github.com/netobserv/netobserv-operator/internal/controller/loki/config"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
)

const (
	configMapName = "loki-config"
	configFile    = "local-config.yaml"
	configVolume  = "loki-config"
	configPath    = "/etc/loki"
	port          = 3100
	storeVolume   = "loki-store"
	storePath     = "/loki-store"
)

type builder struct {
	info     *reconcilers.Instance
	labels   map[string]string
	selector map[string]string
	desired  *flowslatest.FlowCollectorSpec
	volumes  volumes.Builder
}

func newBuilder(info *reconcilers.Instance, desired *flowslatest.FlowCollectorSpec, name string) builder {
	return builder{
		info: info,
		labels: map[string]string{
			"app": name,
		},
		selector: map[string]string{
			"app": name,
		},
		desired: desired,
	}
}

func (b *builder) deployment(name, cm string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(int32(1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: b.selector,
			},
			Template: *b.podTemplate(name, cm),
		},
	}
}

func (b *builder) podTemplate(name, cmDigest string) *corev1.PodTemplateSpec {
	annotations := map[string]string{}
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	volumes = append(volumes, corev1.Volume{
		Name: storeVolume,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: storeVolume,
			},
		},
	})

	volumeMounts = append(volumeMounts, corev1.VolumeMount{
		Name:      storeVolume,
		MountPath: storePath,
	})

	if cmDigest != "" {
		annotations[constants.PodConfigurationDigest] = cmDigest

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

	return &corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      b.labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            name,
				Image:           b.info.Images[reconcilers.MainImage],
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args: []string{
					fmt.Sprintf("-config.file=%s/%s", configPath, configFile),
				},
				VolumeMounts:    b.volumes.AppendMounts(volumeMounts),
				SecurityContext: helper.ContainerDefaultSecurityContext(),
			}},
			Volumes: b.volumes.AppendVolumes(volumes),
		},
	}
}

func (b *builder) service(name string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: b.selector,
			Ports: []corev1.ServicePort{{
				Port:     port,
				Protocol: corev1.ProtocolTCP,
				// Some Kubernetes versions might automatically set TargetPort to Port. We need to
				// explicitly set it here so the reconcile loop verifies that the owned service
				// is equal as the desired service
				TargetPort: intstr.FromInt32(port),
			}},
		},
	}
}

// returns a configmap with a digest of its configuration contents, which will be used to
// detect any configuration change
func (b *builder) configMap() (*corev1.ConfigMap, string, error) {
	configStr := cfg.GetLokiConfigStr()
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
	_, err := hasher.Write([]byte(configStr))
	if err != nil {
		return nil, "", err
	}
	digest := strconv.FormatUint(hasher.Sum64(), 36)
	return &configMap, digest, nil
}

func (b *builder) persistentVolumeClaim() *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      storeVolume,
			Namespace: b.info.Namespace,
			Labels:    b.labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("10Gi"),
				},
			},
			VolumeMode: ptr.To(corev1.PersistentVolumeFilesystem),
		},
	}
}
