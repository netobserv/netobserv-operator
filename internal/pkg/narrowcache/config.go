package narrowcache

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Config struct {
	cfg  *rest.Config
	info []GVKInfo
}

type GVKInfo struct {
	Obj     client.Object
	Getter  func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error)
	Watcher func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error)
	Cleanup func(obj runtime.Object)
}

var (
	ConfigMaps = GVKInfo{
		Obj: &corev1.ConfigMap{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().ConfigMaps(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().ConfigMaps(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			cm := obj.(*corev1.ConfigMap)
			cm.SetManagedFields([]metav1.ManagedFieldsEntry{})
			cm.BinaryData = nil
		},
	}
	ClusterRoles = GVKInfo{
		Obj: &rbacv1.ClusterRole{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.RbacV1().ClusterRoles().Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.RbacV1().ClusterRoles().Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			cr := obj.(*rbacv1.ClusterRole)
			cr.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	ClusterRoleBindings = GVKInfo{
		Obj: &rbacv1.ClusterRoleBinding{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.RbacV1().ClusterRoleBindings().Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.RbacV1().ClusterRoleBindings().Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			crb := obj.(*rbacv1.ClusterRoleBinding)
			crb.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	Daemonsets = GVKInfo{
		Obj: &appsv1.DaemonSet{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.AppsV1().DaemonSets(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.AppsV1().DaemonSets(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			ds := obj.(*appsv1.DaemonSet)
			ds.SetManagedFields([]metav1.ManagedFieldsEntry{})
			ds.Status.Conditions = []appsv1.DaemonSetCondition{}
		},
	}
	Deployments = GVKInfo{
		Obj: &appsv1.Deployment{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.AppsV1().Deployments(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.AppsV1().Deployments(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			dpl := obj.(*appsv1.Deployment)
			dpl.SetManagedFields([]metav1.ManagedFieldsEntry{})
			dpl.Status.Conditions = []appsv1.DeploymentCondition{}
		},
	}
	HorizontalPodAutoscalers = GVKInfo{
		Obj: &ascv2.HorizontalPodAutoscaler{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.AutoscalingV2().HorizontalPodAutoscalers(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.AutoscalingV2().HorizontalPodAutoscalers(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			hpa := obj.(*ascv2.HorizontalPodAutoscaler)
			hpa.SetManagedFields([]metav1.ManagedFieldsEntry{})
			hpa.Status.CurrentMetrics = []ascv2.MetricStatus{}
			hpa.Status.Conditions = []ascv2.HorizontalPodAutoscalerCondition{}
		},
	}
	Namespaces = GVKInfo{
		Obj: &corev1.Namespace{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().Namespaces().Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().Namespaces().Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			ns := obj.(*corev1.Namespace)
			ns.SetManagedFields([]metav1.ManagedFieldsEntry{})
			ns.Status.Conditions = []corev1.NamespaceCondition{}
		},
	}
	NetworkPolicies = GVKInfo{
		Obj: &networkingv1.NetworkPolicy{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.NetworkingV1().NetworkPolicies(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.NetworkingV1().NetworkPolicies(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			np := obj.(*networkingv1.NetworkPolicy)
			np.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	Roles = GVKInfo{
		Obj: &rbacv1.Role{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.RbacV1().Roles(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.RbacV1().Roles(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			ro := obj.(*rbacv1.Role)
			ro.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	RoleBindings = GVKInfo{
		Obj: &rbacv1.RoleBinding{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.RbacV1().RoleBindings(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.RbacV1().RoleBindings(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			rb := obj.(*rbacv1.RoleBinding)
			rb.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	Secrets = GVKInfo{
		Obj: &corev1.Secret{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().Secrets(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().Secrets(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			sc := obj.(*corev1.Secret)
			sc.SetManagedFields([]metav1.ManagedFieldsEntry{})
			sc.StringData = nil
		},
	}
	Services = GVKInfo{
		Obj: &corev1.Service{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().Services(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().Services(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			sv := obj.(*corev1.Service)
			sv.SetManagedFields([]metav1.ManagedFieldsEntry{})
			sv.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{}
			sv.Status.Conditions = []metav1.Condition{}
		},
	}
	ServiceAccounts = GVKInfo{
		Obj: &corev1.ServiceAccount{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().ServiceAccounts(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().ServiceAccounts(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			sa := obj.(*corev1.ServiceAccount)
			sa.SetManagedFields([]metav1.ManagedFieldsEntry{})
		},
	}
	Endpoints = GVKInfo{
		//nolint:staticcheck // SA1019: Endpoints is deprecated but used as fallback for k8s < 1.21
		Obj: &corev1.Endpoints{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.CoreV1().Endpoints(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.CoreV1().Endpoints(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			//nolint:staticcheck // SA1019: Endpoints is deprecated but used as fallback for k8s < 1.21
			e := obj.(*corev1.Endpoints)
			e.SetManagedFields([]metav1.ManagedFieldsEntry{})
			//nolint:staticcheck // SA1019: Endpoints is deprecated but used as fallback for k8s < 1.21
			e.Subsets = []corev1.EndpointSubset{}
		},
	}
	EndpointSlices = GVKInfo{
		Obj: &discoveryv1.EndpointSlice{},
		Getter: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (runtime.Object, error) {
			return cl.DiscoveryV1().EndpointSlices(key.Namespace).Get(ctx, key.Name, metav1.GetOptions{})
		},
		Watcher: func(ctx context.Context, cl kubernetes.Interface, key client.ObjectKey) (watch.Interface, error) {
			opts := metav1.ListOptions{FieldSelector: fields.OneTermEqualSelector(metav1.ObjectNameField, key.Name).String()}
			return cl.DiscoveryV1().EndpointSlices(key.Namespace).Watch(ctx, opts)
		},
		Cleanup: func(obj runtime.Object) {
			e := obj.(*discoveryv1.EndpointSlice)
			e.SetManagedFields([]metav1.ManagedFieldsEntry{})
			e.Endpoints = []discoveryv1.Endpoint{}
			e.Ports = []discoveryv1.EndpointPort{}
		},
	}
	NewLiveClient func(c *rest.Config) (kubernetes.Interface, error) = func(c *rest.Config) (kubernetes.Interface, error) {
		return kubernetes.NewForConfig(c)
	}
)

func NewConfig(cfg *rest.Config, info ...GVKInfo) *Config {
	return &Config{
		cfg:  cfg,
		info: info,
	}
}

func (c *Config) ControllerRuntimeClientCacheOptions() *client.CacheOptions {
	disabled := []client.Object{}
	for _, info := range c.info {
		disabled = append(disabled, info.Obj)
	}
	return &client.CacheOptions{DisableFor: disabled}
}

// CreateClient creates a new client layer that sits on top of the provided `underlying` client.
// This client implements Get for the provided GVKs, using a specific cache.
//
// Other kinds of requests (ie. non-get and non-managed GVKs) are forwarded to the `underlying`
// client.
//
// Furthermore, cached objects are also available as `source.Source`.
func (c *Config) CreateClient(underlying client.Client) (*Client, error) {
	liveClient, err := NewLiveClient(c.cfg)
	if err != nil {
		return nil, err
	}
	watchedGVKs := make(map[string]GVKInfo, len(c.info))
	for _, inf := range c.info {
		gvk, err := underlying.GroupVersionKindFor(inf.Obj)
		if err != nil {
			return nil, err
		}
		watchedGVKs[gvk.String()] = inf
	}
	return &Client{
		Client:         underlying,
		liveClient:     liveClient,
		watchedGVKs:    watchedGVKs,
		watchedObjects: make(map[string]*watchedObject),
	}, nil
}
