// Package tlsconfig exposes the TLS profile conversion shared by the operator
// and the direct CLI deployment path. Cluster discovery and ConfigMap ownership
// intentionally remain with the caller.
package tlsconfig

import (
	"crypto/tls"

	internal "github.com/netobserv/netobserv-operator/internal/pkg/tlsconfig"
	configv1 "github.com/openshift/api/config/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	EnvTLSMinVersion       = internal.EnvTLSMinVersion
	EnvTLSCipherSuites     = internal.EnvTLSCipherSuites
	EnvTLSCurvePreferences = internal.EnvTLSCurvePreferences
)

func ComposeTLSConfig(profile *configv1.TLSSecurityProfile) (*tls.Config, error) {
	return internal.ComposeTLSConfig(profile)
}

func ConfigToEnvVars(config *tls.Config) []corev1.EnvVar {
	return internal.ConfigToEnvVars(config)
}
