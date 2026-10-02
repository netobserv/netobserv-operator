package reconcilers

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/enqueuer"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespacedObjectManager provides some helpers to manage (fetch, delete) namespace-scoped objects
type NamespacedObjectManager struct {
	client         client.Client
	enqueuer       enqueuer.FilteredStatic
	Namespace      string
	managedObjects []managedObject
}

type managedObject struct {
	name        string
	kind        string
	placeholder client.Object
	filter      func(client.Object, client.Object) bool
	found       bool
}

func NewNamespacedObjectManager(cmn *Common) *NamespacedObjectManager {
	var filteredEnqueuer enqueuer.FilteredStatic = cmn.ManagedEnqueuer
	if filteredEnqueuer == nil {
		filteredEnqueuer, _ = cmn.Enqueuer.(enqueuer.FilteredStatic)
	}
	return &NamespacedObjectManager{
		client:    cmn.Client,
		enqueuer:  filteredEnqueuer,
		Namespace: cmn.Namespace,
	}
}

// AddManagedObject should be used to register managed objects to be fetched by FetchAll, or deleted when namespace changes
// This is only for namespace-scoped objects that are installed in the desired namespace (in FlowCollector CRD: spec.namespace)
// Cluster-scope objects, or objects installed in a different namespace (e.g. OVS configmap) should not be registered with this function.
func (m *NamespacedObjectManager) AddManagedObject(name string, placeholder client.Object) {
	m.AddManagedObjectWithFilter(name, placeholder, ManagedObjectEventFilter)
}

func (m *NamespacedObjectManager) AddManagedObjectWithFilter(name string, placeholder client.Object, filter func(client.Object, client.Object) bool) {
	m.managedObjects = append(m.managedObjects, managedObject{
		name:        name,
		kind:        reflect.TypeOf(placeholder).String(),
		placeholder: placeholder,
		filter:      filter,
	})
}

func (m *NamespacedObjectManager) NewConfigMap(name string) *corev1.ConfigMap {
	cm := corev1.ConfigMap{}
	m.AddManagedObject(name, &cm)
	return &cm
}

func (m *NamespacedObjectManager) NewPersistentVolumeClaim(name string) *corev1.PersistentVolumeClaim {
	pvc := corev1.PersistentVolumeClaim{}
	m.AddManagedObject(name, &pvc)
	return &pvc
}

func (m *NamespacedObjectManager) NewDeployment(name string) *appsv1.Deployment {
	d := appsv1.Deployment{}
	m.AddManagedObject(name, &d)
	return &d
}

func (m *NamespacedObjectManager) NewDaemonSet(name string) *appsv1.DaemonSet {
	ds := appsv1.DaemonSet{}
	m.AddManagedObject(name, &ds)
	return &ds
}

func (m *NamespacedObjectManager) NewService(name string) *corev1.Service {
	s := corev1.Service{}
	m.AddManagedObject(name, &s)
	return &s
}

func (m *NamespacedObjectManager) NewServiceAccount(name string) *corev1.ServiceAccount {
	sa := corev1.ServiceAccount{}
	m.AddManagedObject(name, &sa)
	return &sa
}

func (m *NamespacedObjectManager) NewHPA(name string) *ascv2.HorizontalPodAutoscaler {
	hpa := ascv2.HorizontalPodAutoscaler{}
	m.AddManagedObject(name, &hpa)
	return &hpa
}

func (m *NamespacedObjectManager) NewServiceMonitor(name string) *monitoringv1.ServiceMonitor {
	sm := monitoringv1.ServiceMonitor{}
	m.AddManagedObject(name, &sm)
	return &sm
}

func (m *NamespacedObjectManager) NewPrometheusRule(name string) *monitoringv1.PrometheusRule {
	sm := monitoringv1.PrometheusRule{}
	m.AddManagedObject(name, &sm)
	return &sm
}

func (m *NamespacedObjectManager) NewRB(name string) *rbacv1.RoleBinding {
	rb := rbacv1.RoleBinding{}
	m.AddManagedObject(name, &rb)
	return &rb
}

func (m *NamespacedObjectManager) NewNetworkPolicy(name string) *networkingv1.NetworkPolicy {
	np := networkingv1.NetworkPolicy{}
	m.AddManagedObjectWithFilter(name, &np, OperatorOwnedEventFilter(m.Namespace))
	return &np
}

// FetchAll fetches all managed objects (registered using AddManagedObject) in the current namespace.
// Placeholders are filled with fetched resources. Resources not found are flagged internally.
func (m *NamespacedObjectManager) FetchAll(ctx context.Context) error {
	log := log.FromContext(ctx)
	fetched := []string{}
	notFound := []string{}
	for i, ref := range m.managedObjects {
		m.managedObjects[i].found = false
		objLog := ref.kind + "/" + ref.name
		ref.placeholder.SetName(ref.name)
		ref.placeholder.SetNamespace(m.Namespace)
		if m.enqueuer != nil {
			request := reconcile.Request{NamespacedName: constants.FlowCollectorName}
			if err := m.enqueuer.EnqueueOnChangeIfManaged(ctx, ref.placeholder, request, ref.filter); err != nil {
				return err
			}
		}
		err := m.client.Get(ctx, types.NamespacedName{Name: ref.name, Namespace: m.Namespace}, ref.placeholder)
		if err != nil {
			if apierrors.IsNotFound(err) {
				notFound = append(notFound, objLog)
			} else {
				log.Error(err, "Failed to get "+objLog)
				return err
			}
		} else {
			fetched = append(fetched, objLog)
			m.managedObjects[i].found = true
			// On success, placeholder is filled with resource. Caller should keep a pointer to it.
		}
	}
	if len(fetched) > 0 {
		log.Info("FETCHED: " + strings.Join(fetched, ","))
	}
	if len(notFound) > 0 {
		log.Info("(Items not deployed: " + strings.Join(notFound, ",") + ")")
	}
	return nil
}

// ManagedObjectEventFilter matches UpdateOrDeleteOnlyPred for name-scoped watch events.
func ManagedObjectEventFilter(oldObject, newObject client.Object) bool {
	if oldObject == nil { // Ignore creates, matching UpdateOrDeleteOnlyPred.
		return false
	}
	obj := newObject
	if obj == nil { // Deletes are confirmed by the API watch.
		obj = oldObject
	}
	return helper.IsOwned(obj)
}

// TryDeleteAll is an helper function that tries to delete all managed objects previously loaded using FetchAll.
func (m *NamespacedObjectManager) TryDeleteAll(ctx context.Context) error {
	var errs []error
	for _, obj := range m.managedObjects {
		if err := m.TryDelete(ctx, obj.placeholder); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// TryDelete is an helper function that tries to delete the provided object previously loaded using FetchAll.
func (m *NamespacedObjectManager) TryDelete(ctx context.Context, obj client.Object) error {
	if m.Exists(obj) {
		log := log.FromContext(ctx)
		kind := reflect.TypeOf(obj).String()
		log.Info("DELETING "+kind, "Namespace", obj.GetNamespace(), "Name", obj.GetName())
		err := m.client.Delete(ctx, obj)
		if err != nil {
			// Ignore NotFound errors - object may have been deleted concurrently
			err = client.IgnoreNotFound(err)
			if err != nil {
				log.Error(err, "Failed to delete old "+kind, "Namespace", obj.GetNamespace(), "Name", obj.GetName())
				return err
			}
		}
	}
	return nil
}

// Exists returns true if the provided object isn't nil and was successfully fetched previously with FetchAll
func (m *NamespacedObjectManager) Exists(obj client.Object) bool {
	if obj == nil {
		return false
	}
	for _, managed := range m.managedObjects {
		if obj == managed.placeholder {
			return managed.found
		}
	}
	return false
}
