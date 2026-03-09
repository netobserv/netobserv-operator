package watchers

import (
	"context"
	"testing"
	"time"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/narrowcache"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const baseNamespace = "base-ns"
const otherNamespace = "other-ns"

var lokiCA = corev1.ConfigMap{
	ObjectMeta: v1.ObjectMeta{
		Name:      "loki-ca",
		Namespace: baseNamespace,
	},
	Data: map[string]string{
		"tls.crt": " -- LOKI CA --",
	},
}
var lokiTLS = flowslatest.ClientTLS{
	Enable: true,
	CACert: flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeConfigMap,
		Name:     lokiCA.Name,
		CertFile: "tls.crt",
	},
}
var otherLokiCA = corev1.ConfigMap{
	ObjectMeta: v1.ObjectMeta{
		Name:      "other-loki-ca",
		Namespace: otherNamespace,
	},
	Data: map[string]string{
		"tls.crt": " -- LOKI OTHER CA --",
	},
}
var otherLokiTLS = flowslatest.ClientTLS{
	Enable: true,
	CACert: flowslatest.CertificateReference{
		Type:      flowslatest.RefTypeConfigMap,
		Name:      otherLokiCA.Name,
		Namespace: otherLokiCA.Namespace,
		CertFile:  "tls.crt",
	},
}
var kafkaCA = corev1.ConfigMap{
	ObjectMeta: v1.ObjectMeta{
		Name:      "kafka-ca",
		Namespace: baseNamespace,
	},
	Data: map[string]string{
		"ca.crt": " -- KAFKA CA --",
	},
}
var kafkaUser = corev1.Secret{
	ObjectMeta: v1.ObjectMeta{
		Name:      "kafka-user",
		Namespace: baseNamespace,
	},
	Data: map[string][]byte{
		"user.crt": []byte(" -- KAFKA USER CERT --"),
		"user.key": []byte(" -- KAFKA USER KEY --"),
	},
}
var kafkaMTLS = flowslatest.ClientTLS{
	Enable: true,
	CACert: flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeConfigMap,
		Name:     kafkaCA.Name,
		CertFile: "ca.crt",
	},
	UserCert: flowslatest.CertificateReference{
		Type:     flowslatest.RefTypeSecret,
		Name:     kafkaUser.Name,
		CertFile: "user.crt",
		CertKey:  "user.key",
	},
}
var kafkaSaslSecret = corev1.Secret{
	ObjectMeta: v1.ObjectMeta{
		Name:      "kafka-sasl",
		Namespace: baseNamespace,
	},
	Data: map[string][]byte{
		"id":    []byte("me"),
		"token": []byte("ssssaaaaassssslllll"),
	},
}
var kafkaSaslConfig = flowslatest.SASLConfig{
	ClientIDReference: flowslatest.FileReference{
		Type: flowslatest.RefTypeSecret,
		Name: kafkaSaslSecret.Name,
		File: "id",
	},
	ClientSecretReference: flowslatest.FileReference{
		Type: flowslatest.RefTypeSecret,
		Name: kafkaSaslSecret.Name,
		File: "token",
	},
}

type fakeReconcile struct{}

func (r *fakeReconcile) Reconcile(context.Context, reconcile.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func initWatcher(t *testing.T) *Watcher {
	m, err := manager.New(&rest.Config{}, manager.Options{
		NewCache: func(_ *rest.Config, _ cache.Options) (cache.Cache, error) {
			return &informertest.FakeInformers{}, nil
		},
	})
	assert.NoError(t, err)
	b := ctrl.NewControllerManagedBy(m).Named("ctrl-" + string(uuid.NewUUID())).For(&corev1.Pod{})
	ctrl, err := b.Build(&fakeReconcile{})
	assert.NoError(t, err)
	return NewWatcher(ctrl)
}

func setupClients(t *testing.T, clientMock client.Client, liveClient kubernetes.Interface) helper.Client {
	// 1. narrow-cache client
	narrowcache.NewLiveClient = func(_ *rest.Config) (kubernetes.Interface, error) {
		return liveClient, nil
	}
	narrowCache := narrowcache.NewConfig(&rest.Config{}, narrowcache.ConfigMaps, narrowcache.Secrets)
	nc, err := narrowCache.CreateClient(clientMock)
	assert.NoError(t, err)
	return helper.UnmanagedClient(nc)
}

func retry(predicate func() bool, attempts int, sleep time.Duration) {
	if !predicate() && attempts > 0 {
		time.Sleep(sleep)
		retry(predicate, attempts-1, sleep)
	}
}

func TestGenDigests(t *testing.T) {
	assert := assert.New(t)

	watcher := initWatcher(t)
	assert.NotNil(watcher)
	watcher.Reset(baseNamespace)
	goclient := fake.NewClientset(&lokiCA, &kafkaCA, &kafkaUser, &kafkaSaslSecret)
	cl := setupClients(t, test.NewClient(), goclient)

	digLoki, err := watcher.ProcessCACert(context.Background(), cl, &lokiTLS, baseNamespace)
	assert.NoError(err)
	assert.Equal("XDamCg==", digLoki)

	// Same output expected from MTLS func
	dig1, dig2, err := watcher.ProcessMTLSCerts(context.Background(), cl, &lokiTLS, baseNamespace)
	assert.NoError(err)
	assert.Equal("XDamCg==", dig1)
	assert.Equal("", dig2)

	// Different output for kafka certs
	dig1, dig2, err = watcher.ProcessMTLSCerts(context.Background(), cl, &kafkaMTLS, baseNamespace)
	assert.NoError(err)
	assert.Equal("EPFv4Q==", dig1)
	assert.Equal("bNKS0Q==", dig2)

	// Different output for sasl via watcher.Process
	dig1, dig2, err = watcher.ProcessSASL(context.Background(), cl, &kafkaSaslConfig, baseNamespace)
	assert.NoError(err)
	assert.Equal("DTk0Pg==", dig1) // for client ID
	assert.Equal("ItNuCg==", dig2) // for client secret

	// Update object, verify the digest has changed
	caCopy := lokiCA
	caCopy.Data["tls.crt"] = " -- LOKI CA MODIFIED --"
	_, err = goclient.CoreV1().ConfigMaps(lokiCA.Namespace).Update(context.TODO(), &caCopy, v1.UpdateOptions{})
	assert.NoError(err)

	// Watches run in separate goroutine; do some retries if we've been too fast
	var digUpdated string
	retry(func() bool {
		digUpdated, err = watcher.ProcessCACert(context.Background(), cl, &lokiTLS, baseNamespace)
		assert.NoError(err)
		return digUpdated != digLoki
	}, 3, 100*time.Millisecond)
	assert.NotEqual(digLoki, digUpdated)
	assert.Equal("Hb65OQ==", digUpdated)

	// Update another key in object, verify the digest hasn't changed
	caCopy.Data["other"] = " -- OTHER --"
	_, err = goclient.CoreV1().ConfigMaps(lokiCA.Namespace).Update(context.TODO(), &caCopy, v1.UpdateOptions{})
	assert.NoError(err)
	time.Sleep(1 * time.Second)

	digUpdated2, err := watcher.ProcessCACert(context.Background(), cl, &lokiTLS, baseNamespace)
	assert.NoError(err)
	assert.Equal(digUpdated2, digUpdated)
}

func TestNoCopy(t *testing.T) {
	assert := assert.New(t)

	watcher := initWatcher(t)
	assert.NotNil(watcher)
	watcher.Reset(baseNamespace)
	goclient := fake.NewClientset(&lokiCA)
	cl := setupClients(t, test.NewClient(), goclient)

	_, _, err := watcher.ProcessMTLSCerts(context.Background(), cl, &lokiTLS, baseNamespace)
	assert.NoError(err)
	actions := goclient.Actions()
	assert.Len(actions, 2)
	assert.Equal("get", actions[0].GetVerb())
	assert.Equal("/v1, Resource=configmaps", actions[0].GetResource().String())
	assert.Equal(lokiCA.Namespace, actions[0].GetNamespace())
	assert.Equal("watch", actions[1].GetVerb())
	assert.Equal("/v1, Resource=configmaps", actions[1].GetResource().String())
	assert.Equal(lokiCA.Namespace, actions[1].GetNamespace())
}

func TestCopyCertificate(t *testing.T) {
	assert := assert.New(t)
	clientMock := test.NewClient()

	watcher := initWatcher(t)
	assert.NotNil(watcher)
	watcher.Reset(baseNamespace)
	goclient := fake.NewClientset(&otherLokiCA)
	cl := setupClients(t, clientMock, goclient)

	_, _, err := watcher.ProcessMTLSCerts(context.Background(), cl, &otherLokiTLS, baseNamespace)
	assert.NoError(err)
	actions := goclient.Actions()
	assert.Len(actions, 3)
	assert.Equal("get", actions[0].GetVerb())
	assert.Equal("/v1, Resource=configmaps", actions[0].GetResource().String())
	assert.Equal(otherLokiCA.Namespace, actions[0].GetNamespace())
	assert.Equal("watch", actions[1].GetVerb())
	assert.Equal("/v1, Resource=configmaps", actions[1].GetResource().String())
	assert.Equal(otherLokiCA.Namespace, actions[1].GetNamespace())
	assert.Equal("get", actions[2].GetVerb())
	assert.Equal("/v1, Resource=configmaps", actions[2].GetResource().String())
	assert.Equal(baseNamespace, actions[2].GetNamespace())
	clientMock.AssertCreateCalled(t)
	clientMock.AssertUpdateNotCalled(t)
}

func TestUpdateCertificate(t *testing.T) {
	assert := assert.New(t)
	clientMock := test.NewClient()

	// Copy cert changing content => should be updated
	copied := otherLokiCA
	copied.Namespace = baseNamespace
	copied.Data = map[string]string{
		"tls.crt": " -- MODIFIED LOKI OTHER CA --",
	}
	clientMock.MockConfigMap(&copied)

	watcher := initWatcher(t)
	assert.NotNil(watcher)
	watcher.Reset(baseNamespace)
	goclient := fake.NewClientset(&otherLokiCA, &copied)
	cl := setupClients(t, clientMock, goclient)

	_, _, err := watcher.ProcessMTLSCerts(context.Background(), cl, &otherLokiTLS, baseNamespace)
	assert.NoError(err)
	clientMock.AssertCreateNotCalled(t)
	clientMock.AssertUpdateCalled(t)
}

func TestNoUpdateCertificate(t *testing.T) {
	assert := assert.New(t)
	clientMock := test.NewClient()
	// Copy cert keeping same content => should not be updated
	copied := otherLokiCA
	copied.Namespace = baseNamespace
	copied.Data = map[string]string{
		"tls.crt": otherLokiCA.Data["tls.crt"],
	}
	clientMock.MockConfigMap(&copied)

	watcher := initWatcher(t)
	assert.NotNil(watcher)
	watcher.Reset(baseNamespace)
	goclient := fake.NewClientset(&otherLokiCA, &copied)
	cl := setupClients(t, clientMock, goclient)

	_, _, err := watcher.ProcessMTLSCerts(context.Background(), cl, &otherLokiTLS, baseNamespace)
	assert.NoError(err)
	clientMock.AssertCreateNotCalled(t)
	clientMock.AssertUpdateNotCalled(t)
}
