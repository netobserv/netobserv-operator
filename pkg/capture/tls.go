package capture

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const (
	CollectorCAConfigMapName = "collector-ca"
	CollectorTLSConfigName   = "collector-tls-config"
)

// ConfigureCollectorTLS adds the common client-side TLS wiring used by an
// agent sending flows to the ephemeral collector. Certificate resolution and
// resource creation remain the responsibility of the caller.
func ConfigureCollectorTLS(spec *corev1.PodSpec, container *corev1.Container) {
	spec.Volumes = append(spec.Volumes, corev1.Volume{
		Name: CollectorCAConfigMapName,
		VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: CollectorCAConfigMapName},
		}},
	})
	container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
		Name: CollectorCAConfigMapName, MountPath: "/etc/collector-ca", ReadOnly: true,
	})
	container.EnvFrom = append(container.EnvFrom, corev1.EnvFromSource{
		ConfigMapRef: &corev1.ConfigMapEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{Name: CollectorTLSConfigName},
			Optional:             ptr.To(false),
		},
	})
}
