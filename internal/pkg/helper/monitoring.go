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

// scraperTokenFile is the projected service account token that Prometheus presents when scraping an
// authenticated metrics endpoint.
const scraperTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token" // nolint:gosec

// GetServiceMonitorBearerTokenFile returns the token file the scraper must send to an authenticated
// metrics endpoint, or an empty string when the endpoint does not require authentication.
//
// Prometheus-operator deprecates bearerTokenFile in favor of authorization, but the latter reads the
// credentials from a Secret, which the operator isn't allowed to create. The operator's own metrics
// endpoint is scraped the same way, see config/openshift/common/monitor.yaml.
func GetServiceMonitorBearerTokenFile(auth *flowslatest.MetricsServerAuth) string {
	if auth.Type == flowslatest.MetricsAuthKubernetes {
		return scraperTokenFile
	}
	return ""
}

func GetServiceMonitorTLSConfig(tls *flowslatest.ServerTLS, serverName string) (monitoringv1.Scheme, *monitoringv1.TLSConfig) {
	if tls.Type == flowslatest.TLSAuto {
		return "https", &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				ServerName: ptr.To(serverName),
			},
			CAFile: "/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt",
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
