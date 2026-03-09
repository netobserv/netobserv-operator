package manager

import (
	"context"
	"fmt"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/migrator"
	"github.com/netobserv/netobserv-operator/internal/pkg/narrowcache"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

//+kubebuilder:rbac:groups=core,resources=namespaces;services;serviceaccounts;configmaps;persistentvolumeclaims;secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods;nodes;endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=apps,resources=deployments;daemonsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;rolebindings,verbs=get;list;create;delete;update;watch
//+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;create;delete;update;patch;list;watch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;list;update;watch
//+kubebuilder:rbac:groups=operator.openshift.io,resources=networks,verbs=get;list;watch
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors;flowmetrics;flowcollectorslices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors/status;flowmetrics/status;flowcollectorslices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors/finalizers,verbs=update
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=hostnetwork,verbs=use
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=list;create;update;watch
//+kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=list;get;watch
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=get;create;delete;update;patch;list;watch
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions;networks,verbs=get;list;watch
//+kubebuilder:rbac:groups=loki.grafana.com,resources=network,resourceNames=logs,verbs=create
//+kubebuilder:rbac:groups=loki.grafana.com,resources=lokistacks,verbs=get;list;watch
//+kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=create
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bpfman.io,resources=clusterbpfapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bpfman.io,resources=clusterbpfapplications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions/status,verbs=update;patch
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=create;delete;patch;update;get;watch;list
//+kubebuilder:rbac:groups=k8s.ovn.org,resources=userdefinednetworks;clusteruserdefinednetworks,verbs=get;list;watch
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch

type Registerer func(context.Context, *Manager) (PostCreateHook, error)
type PostCreateHook = func(ctx context.Context) error

type Manager struct {
	manager.Manager
	ClusterInfo *cluster.Info
	Client      client.Client
	Status      *status.Manager
	Config      *Config
}

func NewManager(
	ctx context.Context,
	kcfg *rest.Config,
	opcfg *Config,
	opts *ctrl.Options,
	ctrls []Registerer,
) (*Manager, error) {

	log := log.FromContext(ctx)
	log.Info("Creating manager")

	narrowCache := narrowcache.NewConfig(kcfg,
		narrowcache.ConfigMaps,
		narrowcache.ClusterRoles,
		narrowcache.ClusterRoleBindings,
		narrowcache.Daemonsets,
		narrowcache.Deployments,
		narrowcache.HorizontalPodAutoscalers,
		narrowcache.Namespaces,
		narrowcache.NetworkPolicies,
		narrowcache.Roles,
		narrowcache.RoleBindings,
		narrowcache.Secrets,
		narrowcache.Services,
		narrowcache.ServiceAccounts,
		narrowcache.Endpoints,
		narrowcache.EndpointSlices,
	)
	opts.Client = client.Options{Cache: narrowCache.ControllerRuntimeClientCacheOptions()}

	internalManager, err := ctrl.NewManager(kcfg, *opts)
	if err != nil {
		return nil, err
	}
	client, err := narrowCache.CreateClient(internalManager.GetClient())
	if err != nil {
		return nil, fmt.Errorf("unable to create narrow cache client: %w", err)
	}

	statusMgr := status.NewManager()

	log.Info("Discovering APIs")
	dc, err := discovery.NewDiscoveryClientForConfig(kcfg)
	if err != nil {
		return nil, fmt.Errorf("can't instantiate discovery client: %w", err)
	}
	info, postCreate, err := cluster.NewInfo(ctx, kcfg, dc, func() { statusMgr.Sync(ctx, client) })
	if err != nil {
		return nil, fmt.Errorf("can't collect cluster info: %w", err)
	}
	flowslatest.CurrentClusterInfo = info

	this := &Manager{
		Manager:     internalManager,
		ClusterInfo: info,
		Status:      statusMgr,
		Client:      client,
		Config:      opcfg,
	}

	log.Info("Building controllers")
	for _, f := range ctrls {
		if hook, err := f(ctx, this); err != nil {
			return nil, fmt.Errorf("unable to create controller: %w", err)
		} else if hook != nil {
			if err := internalManager.Add(manager.RunnableFunc(hook)); err != nil {
				return nil, fmt.Errorf("unable to register controller post-create hook: %w", err)
			}
		}
	}

	// On every startup, make sure stored CRs are up to date with the defined storage version.
	// This is simply going to run dummy patches to make the API server keep etcd consistent.
	mig := migrator.New(kcfg, []string{
		"flowcollectors.flows.netobserv.io",
		"flowmetrics.flows.netobserv.io",
	})
	if err := internalManager.Add(mig); err != nil {
		return nil, fmt.Errorf("unable to register migrator: %w", err)
	}

	if err := internalManager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return postCreate(ctx)
	})); err != nil {
		return nil, fmt.Errorf("can't collect more cluster info: %w", err)
	}

	return this, nil
}

func (m *Manager) GetClient() client.Client {
	return m.Client
}
