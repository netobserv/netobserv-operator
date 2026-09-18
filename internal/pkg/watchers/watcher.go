package watchers

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/enqueuer"
)

var (
	secrets SecretWatchable
	configs ConfigWatchable
)

type Watcher struct {
	ctrlQ             enqueuer.Dynamic
	wmut              sync.RWMutex
	defaultNamespace  string
	operatorNamespace string
}

func NewWatcher(ctrlQ enqueuer.Dynamic, opNamespace string) *Watcher {
	// Note that Watcher doesn't start any informer at this point, in order to keep informers watching strictly
	// the desired object rather than the whole cluster.
	// Since watched objects can be in any namespace, we cannot use namespace-based restriction to limit memory consumption.
	return &Watcher{
		ctrlQ:             ctrlQ,
		operatorNamespace: opNamespace,
	}
}

func kindToWatchable(kind flowslatest.MountableType) Watchable {
	if kind == flowslatest.RefTypeConfigMap {
		return &configs
	}
	return &secrets
}

func (w *Watcher) Reset(namespace string) {
	w.ctrlQ.ResetActiveWatches()
	w.wmut.Lock()
	defer w.wmut.Unlock()
	w.defaultNamespace = namespace
}

func (w *Watcher) getDefaultNamespace() string {
	w.wmut.RLock()
	defer w.wmut.RUnlock()
	return w.defaultNamespace
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
		if ref.kind == flowslatest.RefTypeSecret && errors.IsForbidden(err) {
			// Hint for user about manual secret watching setup
			return "", fmt.Errorf("make sure you grant Secret access permissions to the operator: "+
				"kubectl create rolebinding secret-watcher -n %s --clusterrole=netobserv-secret-watcher --serviceaccount=%s:netobserv-controller-manager"+
				" ; kubectl create rolebinding secret-creator -n %s --clusterrole=netobserv-secret-creator --serviceaccount=%s:netobserv-controller-manager"+
				" - error was: %w", ref.namespace, w.operatorNamespace, destNamespace, w.operatorNamespace, err)
		}
		return "", err
	}
	err = w.ctrlQ.EnqueueOnChange(ctx, obj, reconcile.Request{NamespacedName: constants.FlowCollectorName})
	if err != nil {
		return "", fmt.Errorf("narrowcache EnqueueOnChange error: %w", err)
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
				if ref.kind == flowslatest.RefTypeSecret && errors.IsForbidden(err) {
					// Hint for user about manual secret watching setup
					return "", fmt.Errorf("make sure you grant Secret write permissions to the operator: "+
						"kubectl create rolebinding secret-creator -n %s --clusterrole=netobserv-secret-creator --serviceaccount=%s:netobserv-controller-manager"+
						" - error was: %w", destNamespace, w.operatorNamespace, err)
				}
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
					if ref.kind == flowslatest.RefTypeSecret && errors.IsForbidden(err) {
						// Hint for user about manual secret watching setup
						return "", fmt.Errorf("make sure you grant Secret write permissions to the operator: "+
							"kubectl create rolebinding secret-creator -n %s --clusterrole=netobserv-secret-creator --serviceaccount=%s:netobserv-controller-manager"+
							" - error was: %w", destNamespace, w.operatorNamespace, err)
					}
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
