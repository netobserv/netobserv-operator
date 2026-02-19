package helper

import (
	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
)

func DefaultCertificateReference(name, namespace string) *flowslatest.CertificateReference {
	return &flowslatest.CertificateReference{
		Type:      flowslatest.RefTypeSecret,
		Name:      name,
		Namespace: namespace,
		CertFile:  "tls.crt",
		CertKey:   "tls.key",
	}
}

func DefaultCAReference(name, namespace string) *flowslatest.FileReference {
	return &flowslatest.FileReference{
		Type:      flowslatest.RefTypeConfigMap,
		Name:      name,
		Namespace: namespace,
		File:      "service-ca.crt",
	}
}

// getServiceTLSConfig returns configs for [server cert, ca, client cert]
func getServiceTLSConfig(desired *flowslatest.ProcessorServiceConfig, autoServerName, autoCAName, autoClientName string) (*flowslatest.CertificateReference, *flowslatest.FileReference, *flowslatest.CertificateReference) {
	if desired != nil && desired.TLSType != flowslatest.TLSAuto && desired.TLSType != flowslatest.TLSAutoMTLS {
		if desired.TLSType == flowslatest.TLSDisabled {
			return nil, nil, nil
		}
		return desired.ProvidedCertificates.ServerCert, desired.ProvidedCertificates.CAFile, desired.ProvidedCertificates.ClientCert
	}
	// Mode auto
	var mTLSClientCert, mTLSServerCert *flowslatest.CertificateReference
	if desired != nil && desired.TLSType == flowslatest.TLSAutoMTLS {
		mTLSClientCert = DefaultCertificateReference(autoClientName, "")
		mTLSServerCert = DefaultCertificateReference(autoServerName, "")
	}
	return mTLSServerCert, DefaultCAReference(autoCAName, ""), mTLSClientCert
}

// GetServiceClientTLSConfig returns configs for [ca, client cert]
func GetServiceClientTLSConfig(desired *flowslatest.ProcessorServiceConfig, defaultSecretName string, isOpenShift bool) (*flowslatest.FileReference, *flowslatest.CertificateReference) {
	caConfigMapName := "netobserv-ca"
	if isOpenShift {
		caConfigMapName = "openshift-service-ca.crt"
	}
	_, ca, clientCert := getServiceTLSConfig(desired, "", caConfigMapName, defaultSecretName)
	return ca, clientCert
}

// GetServiceServerTLSConfig returns configs for [server cert, ca]
func GetServiceServerTLSConfig(desired *flowslatest.ProcessorServiceConfig, defaultSecretName string, isOpenShift bool) (*flowslatest.CertificateReference, *flowslatest.FileReference) {
	caConfigMapName := "netobserv-ca"
	if isOpenShift {
		caConfigMapName = "openshift-service-ca.crt"
	}
	serverCert, ca, _ := getServiceTLSConfig(desired, defaultSecretName, caConfigMapName, "")
	return serverCert, ca
}
