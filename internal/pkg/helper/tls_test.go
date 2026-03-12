package helper

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
)

func TestGetServiceTLSConfig_Disabled(t *testing.T) {
	ca, cert := GetServiceClientTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSDisabled}, "default-cert-secret", false)
	assert.Nil(t, ca)
	assert.Nil(t, cert)

	cert, ca = GetServiceServerTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSDisabled}, "default-cert-secret", false)
	assert.Nil(t, ca)
	assert.Nil(t, cert)
}

func TestGetServiceTLSConfig_Auto(t *testing.T) {
	ca, cert := GetServiceClientTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAuto}, "default-cert-secret", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "netobserv-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Nil(t, cert)

	cert, ca = GetServiceServerTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAuto}, "default-cert-secret", false)
	assert.Nil(t, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "default-cert-secret",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)

	// OpenShift
	ca, cert = GetServiceClientTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAuto}, "default-cert-secret", true)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "openshift-service-ca.crt",
		File: "service-ca.crt",
	}, ca)
	assert.Nil(t, cert)

	cert, ca = GetServiceServerTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAuto}, "default-cert-secret", true)
	assert.Nil(t, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "default-cert-secret",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)
}

func TestGetServiceTLSConfig_AutoMTLS(t *testing.T) {
	ca, cert := GetServiceClientTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAutoMTLS}, "default-cert-secret-a", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "netobserv-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "default-cert-secret-a",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)

	cert, ca = GetServiceServerTLSConfig(&flowslatest.ProcessorServiceConfig{TLSType: flowslatest.TLSAutoMTLS}, "default-cert-secret-b", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "netobserv-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "default-cert-secret-b",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)
}

func TestGetServiceTLSConfig_Provided_TLS(t *testing.T) {
	cfg := flowslatest.ProcessorServiceConfig{
		TLSType: flowslatest.TLSProvided,
		ProvidedCertificates: &flowslatest.ClientServerTLS{
			ServerCert: DefaultCertificateReference("custom-server-cert", ""),
			CAFile:     DefaultCAReference("custom-ca", ""),
		},
	}
	ca, cert := GetServiceClientTLSConfig(&cfg, "default-cert-secret-a", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "custom-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Nil(t, cert)

	cert, ca = GetServiceServerTLSConfig(&cfg, "default-cert-secret-b", false)
	assert.Nil(t, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "custom-server-cert",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)
}

func TestGetServiceTLSConfig_Provided_MTLS(t *testing.T) {
	cfg := flowslatest.ProcessorServiceConfig{
		TLSType: flowslatest.TLSProvided,
		ProvidedCertificates: &flowslatest.ClientServerTLS{
			ServerCert: DefaultCertificateReference("custom-server-cert", ""),
			CAFile:     DefaultCAReference("custom-ca", ""),
			ClientCert: DefaultCertificateReference("custom-client-cert", ""),
		},
	}
	ca, cert := GetServiceClientTLSConfig(&cfg, "default-cert-secret-a", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "custom-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "custom-client-cert",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)

	cert, ca = GetServiceServerTLSConfig(&cfg, "default-cert-secret-b", false)
	assert.Equal(t, &flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "custom-ca",
		File: "service-ca.crt",
	}, ca)
	assert.Equal(t, &flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     "custom-server-cert",
		CertFile: "tls.crt",
		CertKey:  "tls.key",
	}, cert)
}
