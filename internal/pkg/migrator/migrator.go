/*
Copyright 2023 The Tekton Authors, NetObserv

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// copied in Tekton from: https://github.com/knative/pkg/blob/2783cd8cfad9ba907e6f31cafeef3eb2943424ee/apiextensions/storageversion/migrator.go
// local changes: continue the execution even though error happens on patching a resource
// then copied again from: https://github.com/tektoncd/operator/blob/v0.72.0/pkg/reconciler/shared/tektonconfig/upgrade/helper/migrator.go
// local changes: adapted logger
// some refactoring

package migrator

import (
	"context"
	"fmt"
	"time"

	"github.com/netobserv/netobserv-operator/internal/pkg/retry"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/pager"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	retryBackoff = wait.Backoff{
		Steps:    10,
		Duration: 10 * time.Second,
		Factor:   2.0,
		Jitter:   0.1,
	}
)

// Migrator will read custom resource definitions and upgrade
// the associated resources to the latest storage version
type Migrator struct {
	dynamicClient dynamic.Interface
	apixClient    clientset.Interface
	crdGroups     []string
}

// New will return a new Migrator
func New(cfg *rest.Config, crdGroups []string) *Migrator {
	return &Migrator{
		dynamicClient: dynamic.NewForConfigOrDie(cfg),
		apixClient:    clientset.NewForConfigOrDie(cfg),
		crdGroups:     crdGroups,
	}
}

func newForClients(d dynamic.Interface, a clientset.Interface) *Migrator {
	return &Migrator{
		dynamicClient: d,
		apixClient:    a,
	}
}

// Implementing manager.Runnable
func (m *Migrator) Start(ctx context.Context) error {
	l := log.FromContext(ctx).WithName("migrator")
	ctx = log.IntoContext(ctx, l)
	l.Info("ensuring stored resources are up to date")

	for _, crdGroupString := range m.crdGroups {
		crdGroup := schema.ParseGroupResource(crdGroupString)
		if crdGroup.Empty() {
			l.WithValues("crdGroup", crdGroupString).Info("skipping group version (unable to parse)")
			continue
		}
		l.WithValues("crdGroup", crdGroup).Info("migrating group version")
		if err := m.migrateWithRetry(ctx, crdGroup); err != nil {
			if errors.IsNotFound(err) {
				l.WithValues("crdGroup", crdGroup, "error", err).Info("ignoring resource migration - unable to fetch a crdGroup")
				continue
			}
			l.WithValues("crdGroup", crdGroup).Error(err, "failed to migrate a crdGroup")
			// continue the execution, even though failures on the crd migration
		} else {
			l.WithValues("crdGroup", crdGroup).Info("migration completed")
		}
	}
	return nil
}

func (m *Migrator) migrateWithRetry(ctx context.Context, gr schema.GroupResource) error {
	// Retrying to allow time for the conversion webhooks to be ready
	// There's no hurry to get this done, start with a duration > second
	attempt := 0
	return retry.OnError(ctx, retryBackoff, func(error) bool { return true }, func() error {
		log := log.FromContext(ctx)
		log.WithValues("attempt", attempt).Info("try migrate")
		attempt++
		err := m.Migrate(ctx, gr)
		if err != nil {
			log.Error(err, "migration failed")
		}
		return err
	})
}

// Migrate takes a group resource (ie. resource.some.group.dev) and
// updates instances of the resource to the latest storage version
//
// This is done by listing all the resources and performing an empty patch
// which triggers a migration on the K8s API server
//
// Finally the migrator will update the CRD's status and drop older storage
// versions
func (m *Migrator) Migrate(ctx context.Context, gr schema.GroupResource) error {
	crdClient := m.apixClient.ApiextensionsV1().CustomResourceDefinitions()

	crd, err := crdClient.Get(ctx, gr.String(), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("unable to fetch crd %s - %w", gr, err)
	}

	version := storageVersion(crd)

	if version == "" {
		return fmt.Errorf("unable to determine storage version for %s", gr)
	}

	if err := m.migrateResources(ctx, gr.WithVersion(version)); err != nil {
		return err
	}

	patch := `{"status":{"storedVersions":["` + version + `"]}}`
	_, err = crdClient.Patch(ctx, crd.Name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("unable to drop storage version definition %s - %w", gr, err)
	}

	return nil
}

func (m *Migrator) migrateResources(ctx context.Context, gvr schema.GroupVersionResource) error {
	client := m.dynamicClient.Resource(gvr)

	listFunc := func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return client.Namespace(metav1.NamespaceAll).List(ctx, opts)
	}

	onEach := func(obj runtime.Object) error {
		item := obj.(metav1.Object)

		_, err := client.Namespace(item.GetNamespace()).
			Patch(ctx, item.GetName(), types.MergePatchType, []byte("{}"), metav1.PatchOptions{})

		if err != nil && !errors.IsNotFound(err) {
			log.FromContext(ctx).
				WithValues(
					"resourceName", item.GetName(),
					"namespace", item.GetNamespace(),
					"groupVersionResource", gvr,
				).
				Error(err, "unable to patch a resource")
		}

		return nil
	}

	pager := pager.New(listFunc)
	return pager.EachListItem(ctx, metav1.ListOptions{}, onEach)
}

func storageVersion(crd *apix.CustomResourceDefinition) string {
	var version string

	for _, v := range crd.Spec.Versions {
		if v.Storage {
			version = v.Name
			break
		}
	}

	return version
}
