package reconcilers

import (
	"context"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager/status"
	"github.com/netobserv/netobserv-operator/internal/pkg/watchers"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

type Common struct {
	helper.Client
	Watcher      *watchers.Watcher
	Namespace    string
	ClusterInfo  *cluster.Info
	Loki         *helper.LokiConfig
	IsDownstream bool
}

func (c *Common) PrivilegedNamespace() string {
	return c.Namespace + constants.EBPFPrivilegedNSSuffix
}

type ImageRef string

const (
	MainImage                ImageRef = "main"
	BpfByteCodeImage         ImageRef = "bpf-bytecode"
	ConsolePluginCompatImage ImageRef = "console-plugin-compat"
)

type Instance struct {
	*Common
	Managed *NamespacedObjectManager
	Images  map[ImageRef]string
	Status  status.Instance
}

func (c *Common) NewInstance(images map[ImageRef]string, st status.Instance) *Instance {
	managed := NewNamespacedObjectManager(c)
	return &Instance{
		Common:  c,
		Managed: managed,
		Images:  images,
		Status:  st,
	}
}

func (c *Common) ReconcileClusterRoleBinding(ctx context.Context, desired *rbacv1.ClusterRoleBinding) error {
	return ReconcileClusterRoleBinding(ctx, &c.Client, desired)
}

func (c *Common) ReconcileRoleBinding(ctx context.Context, desired *rbacv1.RoleBinding) error {
	return ReconcileRoleBinding(ctx, &c.Client, desired)
}

func (c *Common) ReconcileConfigMap(ctx context.Context, old, n *corev1.ConfigMap) error {
	return ReconcileConfigMap(ctx, &c.Client, old, n)
}

func (i *Instance) ReconcileService(ctx context.Context, old, n *corev1.Service, report *helper.ChangeReport) error {
	return ReconcileService(ctx, i, old, n, report)
}
