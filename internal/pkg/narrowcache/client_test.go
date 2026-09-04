package narrowcache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type gvkClient struct {
	client.Client
}

func (c *gvkClient) GroupVersionKindFor(obj runtime.Object) (schema.GroupVersionKind, error) {
	return apiutil.GVKForObject(obj, scheme.Scheme)
}

func TestWatchReestablishment(t *testing.T) {
	assert := assert.New(t)
	ctx, cancel := context.WithCancel(context.Background())

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
		Data:       map[string]string{"key": "value1"},
	}
	goclient := fake.NewClientset(cm)

	// Intercept watch calls to control when the watch channel closes
	watcher1 := watch.NewFake()
	watcher2 := watch.NewFake()
	var watchCount atomic.Int32
	goclient.PrependWatchReactor("configmaps", func(_ k8stesting.Action) (bool, watch.Interface, error) {
		if watchCount.Add(1) == 1 {
			return true, watcher1, nil
		}
		return true, watcher2, nil
	})

	newLiveClientBackup := NewLiveClient
	NewLiveClient = func(_ *rest.Config) (kubernetes.Interface, error) {
		return goclient, nil
	}
	t.Cleanup(func() {
		cancel()
		watcher1.Stop()
		watcher2.Stop()
		NewLiveClient = newLiveClientBackup
	})
	narrowCache := NewConfig(&rest.Config{}, ConfigMaps)
	underlying := &gvkClient{}
	nc, err := narrowCache.CreateClient(underlying)
	assert.NoError(err)

	// First Get: triggers live query + starts watch goroutine
	out := &corev1.ConfigMap{}
	err = nc.Get(ctx, client.ObjectKey{Name: "test-cm", Namespace: "test-ns"}, out)
	assert.NoError(err)
	assert.Equal("value1", out.Data["key"])
	assert.Equal(int32(1), watchCount.Load())

	// Register a handler via GetSource + Start to verify it survives watch re-establishment
	var handlerCalls atomic.Int32
	src, err := nc.GetSource(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cm", Namespace: "test-ns"},
	}, handler.Funcs{
		UpdateFunc: func(_ context.Context, _ event.UpdateEvent, _ workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			handlerCalls.Add(1)
		},
	})
	assert.NoError(err)
	q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	defer q.ShutDown()
	assert.NoError(src.Start(ctx, q))

	// Simulate API watch timeout: closing the channel triggers re-establishment
	watcher1.Stop()

	// Watch should be re-established, and handler notified
	assert.Eventually(func() bool { return watchCount.Load() == 2 }, 2*time.Second, 50*time.Millisecond,
		"expected watch to be re-established")
	assert.Eventually(func() bool { return handlerCalls.Load() > 0 }, 2*time.Second, 50*time.Millisecond,
		"expected handler to be called on re-establishment")

	// Cache still returns valid data after re-establishment
	out2 := &corev1.ConfigMap{}
	assert.NoError(nc.Get(ctx, client.ObjectKey{Name: "test-cm", Namespace: "test-ns"}, out2))
	assert.Equal("value1", out2.Data["key"])

	// Events on the new watch are processed
	updatedCM := cm.DeepCopy()
	updatedCM.Data["key"] = "value2"
	watcher2.Modify(updatedCM)

	assert.Eventually(func() bool {
		out3 := &corev1.ConfigMap{}
		_ = nc.Get(ctx, client.ObjectKey{Name: "test-cm", Namespace: "test-ns"}, out3)
		return out3.Data["key"] == "value2"
	}, 2*time.Second, 50*time.Millisecond, "expected cache to reflect update via new watch")
}
