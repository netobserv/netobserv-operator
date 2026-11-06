package utils

import (
	"e2eframework"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// WaitForDeploymentAvailable waits for the named Deployment, in the NetObserv
// namespace, to report the Available condition.
func WaitForDeploymentAvailable(t *testing.T, cfg *envconf.Config, name string) {
	t.Helper()
	client, err := cfg.NewClient()
	if err != nil {
		t.Fatalf("building client: %v", err)
	}
	dep := appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: e2eframework.Namespace}}
	err = wait.For(
		conditions.New(client.Resources()).DeploymentConditionMatch(&dep, appsv1.DeploymentAvailable, corev1.ConditionTrue),
		wait.WithTimeout(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("deployment %s/%s did not become available: %v", e2eframework.Namespace, name, err)
	}
}

// AssertUnset fails the test if the given dotted path is set to a non-empty
// value in the FlowCollector. Such a value would mean a controller-internal
// default leaked back into the CR spec, which only happens when the
// reconciler mistakenly uses `Update` instead of `Patch`.
func AssertUnset(t *testing.T, fc map[string]interface{}, label string, path ...string) {
	t.Helper()
	v, found, err := unstructured.NestedString(fc, path...)
	if err != nil {
		t.Fatalf("reading %s: %v", label, err)
	}
	if found && v != "" {
		t.Errorf("%s should not have been set by the operator, got %q (was the FlowCollector `Update`d instead of `Patch`ed?)", label, v)
	}
}

// PodSpec waits for the named Deployment or DaemonSet to exist and returns its
// pod template. It deliberately does not wait for readiness: the eBPF agent is
// not expected to run successfully on every Kind host, but the operator still
// renders its DaemonSet.
func PodSpec(t *testing.T, cfg *envconf.Config, namespace, name string, daemonSet bool) corev1.PodSpec {
	t.Helper()
	client, err := cfg.NewClient()
	if err != nil {
		t.Fatalf("building client: %v", err)
	}
	meta := metav1.ObjectMeta{Name: name, Namespace: namespace}

	var obj k8s.Object
	var template func() corev1.PodSpec
	if daemonSet {
		ds := &appsv1.DaemonSet{ObjectMeta: meta}
		obj, template = ds, func() corev1.PodSpec { return ds.Spec.Template.Spec }
	} else {
		dp := &appsv1.Deployment{ObjectMeta: meta}
		obj, template = dp, func() corev1.PodSpec { return dp.Spec.Template.Spec }
	}

	err = wait.For(
		conditions.New(client.Resources()).ResourceMatch(obj, func(k8s.Object) bool { return true }),
		wait.WithTimeout(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("%s/%s never appeared: %v", namespace, name, err)
	}
	return template()
}

func FindContainer(t *testing.T, spec *corev1.PodSpec, name string) corev1.Container {
	t.Helper()
	var found []string
	for i := range spec.Containers {
		if spec.Containers[i].Name == name {
			return spec.Containers[i]
		}
		found = append(found, spec.Containers[i].Name)
	}
	t.Fatalf("no container named %q, found %v", name, found)
	return corev1.Container{}
}

// Quantities renders a ResourceList as plain strings, so that a mismatch prints
// readably instead of dumping resource.Quantity internals.
func Quantities(list corev1.ResourceList) map[string]string {
	out := make(map[string]string, len(list))
	for name, q := range list {
		out[string(name)] = q.String()
	}
	return out
}

func EnvValue(ctr *corev1.Container, name string) (string, bool) {
	for _, e := range ctr.Env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}
