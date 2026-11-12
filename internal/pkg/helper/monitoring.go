package helper

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

func GetSecretOrConfigMap(file *flowslatest.FileReference) monitoringv1.SecretOrConfigMap {
	if file.Type == flowslatest.RefTypeConfigMap {
		return monitoringv1.SecretOrConfigMap{
			ConfigMap: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: file.Name,
				},
				Key: file.File,
			},
		}
	}
	return monitoringv1.SecretOrConfigMap{
		Secret: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: file.Name,
			},
			Key: file.File,
		},
	}
}

func GetServiceMonitorTLSConfig(tls *flowslatest.ServerTLS, serverName string) (monitoringv1.Scheme, *monitoringv1.TLSConfig) {
	if tls.Type == flowslatest.TLSAuto {
		return "https", &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName: ptr.To(serverName),
			},
			TLSFilesConfig: monitoringv1.TLSFilesConfig{
				CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
			},
		}
	}

	if tls.Type == flowslatest.TLSProvided {
		tlsOut := monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName:         ptr.To(serverName),
				InsecureSkipVerify: &tls.InsecureSkipVerify,
			},
		}
		if !tls.InsecureSkipVerify && tls.ProvidedCAFile != nil && tls.ProvidedCAFile.File != "" {
			tlsOut.SafeTLSConfig.CA = GetSecretOrConfigMap(tls.ProvidedCAFile)
		}
		return "https", &tlsOut
	}

	return "http", nil
}
