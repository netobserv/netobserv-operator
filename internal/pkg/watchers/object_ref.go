package watchers

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
)

type objectRef struct {
	kind      flowslatest.MountableType
	name      string
	namespace string
	keys      []string
}

func (w *Watcher) refFromFile(fr *flowslatest.FileReference) objectRef {
	ns := fr.Namespace
	if ns == "" {
		ns = w.defaultNamespace
	}
	return objectRef{
		kind:      fr.Type,
		name:      fr.Name,
		namespace: ns,
		keys:      []string{fr.File},
	}
}

func (w *Watcher) refFromCert(cert *flowslatest.CertificateReference) objectRef {
	ns := cert.Namespace
	if ns == "" {
		ns = w.defaultNamespace
	}
	keys := []string{cert.CertFile}
	if cert.CertKey != "" {
		keys = append(keys, cert.CertKey)
	}
	return objectRef{
		kind:      cert.Type,
		name:      cert.Name,
		namespace: ns,
		keys:      keys,
	}
}
