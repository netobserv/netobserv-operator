package cleanup

import (
	"context"

	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	// Add to this list any object that we used to generate in past versions, and stopped doing so.
	// For instance, with any object that was renamed between two releases of the operator:
	// The old version with a different name could therefore remain on the cluster after an upgrade.
	cleanupList = []cleanupItem{
		// Old role bindings (1.8 and before)
		{
			ref:         client.ObjectKey{Name: "netobserv-plugin"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-ingester-role-mono"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-transformer-role-mono"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-ingester-role"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
		{
			ref:         client.ObjectKey{Name: "flowlogs-pipeline-transformer-role"},
			placeholder: &rbacv1.ClusterRoleBinding{},
			namespaced:  false,
		},
	}
	// Need to run only once, at operator startup, this is not part of the reconcile loop
	didRun = false
)

type cleanupItem struct {
	ref         client.ObjectKey
	placeholder client.Object
	namespaced  bool
}

func CleanPastReferences(ctx context.Context, cl client.Client, defaultNamespace string) error {
	if didRun {
		return nil
	}
	log := log.FromContext(ctx)
	log.Info("Check and clean old objects")
	// Search for all past references to clean up. If one is found, delete it.
	for _, item := range cleanupList {
		if item.ref.Namespace == "" && item.namespaced {
			item.ref.Namespace = defaultNamespace
		}
		if err := cl.Get(ctx, item.ref, item.placeholder); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return err
		}
		// Make sure we own that object
		if helper.IsOwned(item.placeholder) {
			log.
				WithValues("name", item.ref.Name).
				WithValues("namespace", item.ref.Namespace).
				Info("Deleting old object")
			if err := cl.Delete(ctx, item.placeholder); err != nil {
				return err
			}
		} else {
			log.
				WithValues("name", item.ref.Name).
				WithValues("namespace", item.ref.Namespace).
				Info("An object was found, but we don't own it - skip deletion")
		}
	}
	didRun = true
	return nil
}
