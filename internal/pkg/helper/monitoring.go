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

func GetServiceMonitorTLSConfig(tls *flowslatest.ServerTLS, serverName string, isDownstream bool) (monitoringv1.Scheme, *monitoringv1.TLSConfig) {
	if tls.Type == flowslatest.TLSAuto {
		if isDownstream {
			return "https", &monitoringv1.TLSConfig{
				SafeTLSConfig: monitoringv1.SafeTLSConfig{
					ServerName: ptr.To(serverName),
				},
				CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
			}
		}
		// Upstream prometheus disallows CAFile
		return "https", &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName: ptr.To(serverName),
				CA: monitoringv1.SecretOrConfigMap{
					ConfigMap: &corev1.ConfigMapKeySelector{
						Key: "service-ca.crt",
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "openshift-service-ca.crt",
						},
					},
				},
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
