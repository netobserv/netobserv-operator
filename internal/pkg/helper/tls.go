package helper

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
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

// GetServiceClientTLSConfig returns configs for [ca, client cert]
func GetServiceClientTLSConfig(desired *flowslatest.ProcessorServiceConfig, defaultSecretName string, isOpenShift bool) (*flowslatest.FileReference, *flowslatest.CertificateReference) {
	if desired != nil && desired.TLSType != flowslatest.TLSAuto && desired.TLSType != flowslatest.TLSAutoMTLS {
		if desired.TLSType == flowslatest.TLSDisabled {
			return nil, nil
		}
		if desired.ProvidedCertificates == nil {
			// This should not happen, prevented by the validation webhook
			return nil, nil
		}
		return desired.ProvidedCertificates.CAFile, desired.ProvidedCertificates.ClientCert
	}
	// Mode auto
	caConfigMapName := "netobserv-ca"
	if isOpenShift {
		caConfigMapName = "openshift-service-ca.crt"
	}
	ca := DefaultCAReference(caConfigMapName, "")
	if desired != nil && desired.TLSType == flowslatest.TLSAutoMTLS {
		return ca, DefaultCertificateReference(defaultSecretName, "")
	}
	return ca, nil
}

// GetServiceServerTLSConfig returns configs for [server cert, ca]
func GetServiceServerTLSConfig(desired *flowslatest.ProcessorServiceConfig, defaultSecretName string, isOpenShift bool) (*flowslatest.CertificateReference, *flowslatest.FileReference) {
	if desired != nil && desired.TLSType != flowslatest.TLSAuto && desired.TLSType != flowslatest.TLSAutoMTLS {
		if desired.TLSType == flowslatest.TLSDisabled {
			return nil, nil
		}
		if desired.ProvidedCertificates == nil {
			// This should not happen, prevented by the validation webhook
			return nil, nil
		}
		if desired.ProvidedCertificates.ClientCert != nil {
			// mTLS => provide the CA for server
			return desired.ProvidedCertificates.ServerCert, desired.ProvidedCertificates.CAFile
		}
		// Simple TLS => no CA for server
		return desired.ProvidedCertificates.ServerCert, nil
	}
	// Mode auto
	caConfigMapName := "netobserv-ca"
	if isOpenShift {
		caConfigMapName = "openshift-service-ca.crt"
	}
	serverCert := DefaultCertificateReference(defaultSecretName, "")
	if desired != nil && desired.TLSType == flowslatest.TLSAutoMTLS {
		return serverCert, DefaultCAReference(caConfigMapName, "")
	}
	return serverCert, nil
}
