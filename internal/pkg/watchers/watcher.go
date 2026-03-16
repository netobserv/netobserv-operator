package watchers

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/narrowcache"
)

var (
	secrets SecretWatchable
	configs ConfigWatchable
)

type Watcher struct {
	ctrl             controller.Controller
	watches          map[string]bool
	wmut             sync.RWMutex
	defaultNamespace string
}

func NewWatcher(ctrl controller.Controller) *Watcher {
	// Note that Watcher doesn't start any informer at this point, in order to keep informers watching strictly
	// the desired object rather than the whole cluster.
	// Since watched objects can be in any namespace, we cannot use namespace-based restriction to limit memory consumption.
	return &Watcher{
		ctrl:    ctrl,
		watches: make(map[string]bool),
	}
}

func kindToWatchable(kind flowslatest.MountableType) Watchable {
	if kind == flowslatest.RefTypeConfigMap {
		return &configs
	}
	return &secrets
}

func (w *Watcher) Reset(namespace string) {
	w.defaultNamespace = namespace
	// Reset all registered watches as inactive
	w.wmut.Lock()
	for k := range w.watches {
		w.watches[k] = false
	}
	w.wmut.Unlock()
}

func key(kind flowslatest.MountableType, name, namespace string) string {
	return string(kind) + "/" + namespace + "/" + name
}

func (w *Watcher) setActiveWatch(key string) bool {
	w.wmut.Lock()
	_, exists := w.watches[key]
	w.watches[key] = true
	w.wmut.Unlock()
	return exists
}

func (w *Watcher) watch(ctx context.Context, cl *narrowcache.Client, kind flowslatest.MountableType, obj client.Object) error {
	k := key(kind, obj.GetName(), obj.GetNamespace())
	// Mark as active
	exists := w.setActiveWatch(k)
	if exists {
		// Don't register again
		return nil
	}
	s, err := cl.GetSource(
		ctx,
		obj,
		handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o client.Object) []reconcile.Request {
			// The watch might be registered, but inactive
			k := key(kind, o.GetName(), o.GetNamespace())
			w.wmut.RLock()
			active := w.watches[k]
			w.wmut.RUnlock()
			if active {
				// Trigger FlowCollector reconcile
				return []reconcile.Request{{NamespacedName: constants.FlowCollectorName}}
			}
			return []reconcile.Request{}
		}),
	)
	if err != nil {
		return err
	}
	// Note that currently, watches are never removed (they can't - cf https://github.com/kubernetes-sigs/controller-runtime/issues/1884)
	// This isn't a big deal here, as the number of watches that we set is very limited and not meant to grow over and over
	// (unless user keeps reconfiguring cert references endlessly)
	err = w.ctrl.Watch(s)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watcher) ProcessMTLSCerts(ctx context.Context, cl helper.Client, tls *flowslatest.ClientTLS, targetNamespace string) (caDigest string, userDigest string, err error) {
	if tls.Enable && tls.CACert.Name != "" {
		caRef := w.refFromCert(&tls.CACert)
		caDigest, err = w.reconcile(ctx, cl, caRef, targetNamespace)
		if err != nil {
			return "", "", err
		}
	}
	if tls.Enable && tls.UserCert.Name != "" {
		userRef := w.refFromCert(&tls.UserCert)
		userDigest, err = w.reconcile(ctx, cl, userRef, targetNamespace)
		if err != nil {
			return "", "", err
		}
	}
	return caDigest, userDigest, nil
}

func (w *Watcher) ProcessMTLSCertsFromRefs(ctx context.Context, cl helper.Client, ca *flowslatest.FileReference, cert *flowslatest.CertificateReference, targetNamespace string) (caDigest string, userDigest string, err error) {
	caRef := w.refFromFile(ca)
	caDigest, err = w.reconcile(ctx, cl, caRef, targetNamespace)
	if err != nil {
		return "", "", err
	}
	userRef := w.refFromCert(cert)
	userDigest, err = w.reconcile(ctx, cl, userRef, targetNamespace)
	if err != nil {
		return "", "", err
	}
	return caDigest, userDigest, nil
}

func (w *Watcher) ProcessCACert(ctx context.Context, cl helper.Client, tls *flowslatest.ClientTLS, targetNamespace string) (caDigest string, err error) {
	if tls.Enable && tls.CACert.Name != "" {
		caRef := w.refFromCert(&tls.CACert)
		caDigest, err = w.reconcile(ctx, cl, caRef, targetNamespace)
		if err != nil {
			return "", err
		}
	}
	return caDigest, nil
}

func (w *Watcher) ProcessCertRef(ctx context.Context, cl helper.Client, cert *flowslatest.CertificateReference, targetNamespace string) (certDigest string, err error) {
	if cert != nil {
		certRef := w.refFromCert(cert)
		certDigest, err = w.reconcile(ctx, cl, certRef, targetNamespace)
		if err != nil {
			return "", err
		}
	}

	return certDigest, nil
}

func (w *Watcher) ProcessFileReference(ctx context.Context, cl helper.Client, file flowslatest.FileReference, targetNamespace string) (fileDigest string, err error) {
	fileDigest, err = w.reconcile(ctx, cl, w.refFromFile(&file), targetNamespace)
	if err != nil {
		return "", err
	}
	return fileDigest, nil
}

func (w *Watcher) ProcessSASL(ctx context.Context, cl helper.Client, sasl *flowslatest.SASLConfig, targetNamespace string) (idDigest string, secretDigest string, err error) {
	idDigest, err = w.reconcile(ctx, cl, w.refFromFile(&sasl.ClientIDReference), targetNamespace)
	if err != nil {
		return "", "", err
	}
	secretDigest, err = w.reconcile(ctx, cl, w.refFromFile(&sasl.ClientSecretReference), targetNamespace)
	if err != nil {
		return "", "", err
	}
	return idDigest, secretDigest, nil
}

func (w *Watcher) reconcile(ctx context.Context, cl helper.Client, ref objectRef, destNamespace string) (string, error) {
	rlog := log.FromContext(ctx, "Name", ref.name, "Source namespace", ref.namespace, "Target namespace", destNamespace)
	ctx = log.IntoContext(ctx, rlog)
	report := helper.NewChangeReport("Watcher for " + string(ref.kind) + " " + ref.name)
	defer report.LogIfNeeded(ctx)

	watchable := kindToWatchable(ref.kind)
	obj := watchable.ProvidePlaceholder()
	err := cl.Get(ctx, types.NamespacedName{Name: ref.name, Namespace: ref.namespace}, obj)
	if err != nil {
		return "", err
	}
	err = w.watch(ctx, cl.Client.(*narrowcache.Client), ref.kind, obj)
	if err != nil {
		return "", err
	}
	digest, err := watchable.GetDigest(obj, ref.keys)
	if err != nil {
		return "", err
	}
	if ref.namespace != destNamespace {
		// copy to namespace
		target := watchable.ProvidePlaceholder()
		err := cl.Get(ctx, types.NamespacedName{Name: ref.name, Namespace: destNamespace}, target)
		if err != nil {
			if !errors.IsNotFound(err) {
				return "", err
			}
			rlog.Info(fmt.Sprintf("creating %s %s in namespace %s", ref.kind, ref.name, destNamespace))
			watchable.PrepareForCreate(obj, &metav1.ObjectMeta{
				Name:      ref.name,
				Namespace: destNamespace,
				Annotations: map[string]string{
					constants.NamespaceCopyAnnotation: ref.namespace + "/" + ref.name,
				},
			})
			if err := cl.CreateOwned(ctx, obj); err != nil {
				return "", err
			}
		} else {
			// Check for update
			targetDigest, err := watchable.GetDigest(target, ref.keys)
			if err != nil {
				return "", err
			}
			if report.Check("Digest changed", targetDigest != digest) {
				// Update existing
				rlog.Info(fmt.Sprintf("updating %s %s in namespace %s", ref.kind, ref.name, destNamespace))
				watchable.PrepareForUpdate(obj, target)
				if err := cl.UpdateOwned(ctx, target, target); err != nil {
					return "", err
				}
			}
		}
	}
	return digest, nil
}

func Annotation(key string) string {
	return constants.PodWatchedSuffix + key
}
