package helper

import (
	"fmt"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
)

type LokiConfig struct {
	flowslatest.LokiManualParams
}

func NewLokiConfig(spec *flowslatest.FlowCollectorLoki, namespace string) LokiConfig {
	loki := LokiConfig{}
	switch spec.Mode {
	case flowslatest.LokiModeLokiStack:
		ns := namespace
		if len(spec.LokiStack.Namespace) > 0 {
			ns = spec.LokiStack.Namespace
		}
		// NB: trailing dot (...local.:8080) is a DNS optimization for exact name match without extra search
		gatewayURL := fmt.Sprintf("https://%s-gateway-http.%s.svc.cluster.local.:8080/api/logs/v1/network/", spec.LokiStack.Name, ns)
		loki.LokiManualParams = flowslatest.LokiManualParams{
			QuerierURL:  gatewayURL,
			IngesterURL: gatewayURL,
			StatusURL:   fmt.Sprintf("https://%s-query-frontend-http.%s.svc.cluster.local.:3100/", spec.LokiStack.Name, ns),
			TenantID:    "network",
			AuthToken:   flowslatest.LokiAuthForwardUserToken,
			TLS: flowslatest.ClientTLS{
				Enable: true,
				CACert: flowslatest.CertificateReference{
					Type:      flowslatest.RefTypeConfigMap,
					Name:      fmt.Sprintf("%s-gateway-ca-bundle", spec.LokiStack.Name),
					Namespace: spec.LokiStack.Namespace,
					CertFile:  "service-ca.crt",
				},
			},
			StatusTLS: flowslatest.ClientTLS{
				Enable: true,
				CACert: flowslatest.CertificateReference{
					Type:      flowslatest.RefTypeConfigMap,
					Name:      fmt.Sprintf("%s-ca-bundle", spec.LokiStack.Name),
					Namespace: spec.LokiStack.Namespace,
					CertFile:  "service-ca.crt",
				},
				UserCert: flowslatest.CertificateReference{
					Type:      flowslatest.RefTypeSecret,
					Name:      fmt.Sprintf("%s-query-frontend-http", spec.LokiStack.Name),
					Namespace: spec.LokiStack.Namespace,
					CertFile:  "tls.crt",
					CertKey:   "tls.key",
				},
			},
		}
	case flowslatest.LokiModeMonolithic:
		if spec.Monolithic.InstallDemoLoki != nil && *spec.Monolithic.InstallDemoLoki {
			loki.LokiManualParams = flowslatest.LokiManualParams{
				QuerierURL:  "http://loki:3100/",
				IngesterURL: "http://loki:3100/",
				TenantID:    "netobserv",
				AuthToken:   flowslatest.LokiAuthDisabled,
			}
		} else {
			loki.LokiManualParams = flowslatest.LokiManualParams{
				QuerierURL:  spec.Monolithic.URL,
				IngesterURL: spec.Monolithic.URL,
				StatusURL:   spec.Monolithic.URL,
				TenantID:    spec.Monolithic.TenantID,
				TLS:         spec.Monolithic.TLS,
			}
		}
	case flowslatest.LokiModeMicroservices:
		loki.LokiManualParams = flowslatest.LokiManualParams{
			QuerierURL:  spec.Microservices.QuerierURL,
			IngesterURL: spec.Microservices.IngesterURL,
			StatusURL:   spec.Microservices.QuerierURL,
			TenantID:    spec.Microservices.TenantID,
			TLS:         spec.Microservices.TLS,
		}
	case flowslatest.LokiModeManual:
		loki.LokiManualParams = spec.Manual
	default:
		// Default / fallback => manual
		loki.LokiManualParams = spec.Manual
	}
	return loki
}

func (l *LokiConfig) UseForwardToken() bool {
	return l.LokiManualParams.AuthToken == flowslatest.LokiAuthForwardUserToken
}

func (l *LokiConfig) UseHostToken() bool {
	return l.LokiManualParams.AuthToken == flowslatest.LokiAuthUseHostToken
}
