package reconcilers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestManagedObjectEventFilter(t *testing.T) {
	owned := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"netobserv-managed": "true"}}}
	unowned := &corev1.ConfigMap{}

	assert.False(t, ManagedObjectEventFilter(nil, owned), "managed-object sources should ignore creates")
	assert.True(t, ManagedObjectEventFilter(owned, owned), "owned updates should trigger")
	assert.True(t, ManagedObjectEventFilter(owned, nil), "confirmed deletes of owned objects should trigger")
	assert.False(t, ManagedObjectEventFilter(unowned, unowned), "unowned updates should be ignored")
	assert.False(t, ManagedObjectEventFilter(unowned, nil), "deletes of unowned objects should be ignored")
}

func TestIgnoreStatusChangeEventFilter(t *testing.T) {
	oldObject := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Generation: 1,
		Labels:     map[string]string{"key": "value"},
	}}
	newObject := oldObject.DeepCopy()

	assert.True(t, IgnoreStatusChangeEventFilter(nil, newObject), "creates should trigger")
	assert.True(t, IgnoreStatusChangeEventFilter(oldObject, nil), "deletes should trigger")
	assert.False(t, IgnoreStatusChangeEventFilter(oldObject, newObject), "unchanged metadata should be ignored")

	newObject = oldObject.DeepCopy()
	newObject.Generation++
	assert.True(t, IgnoreStatusChangeEventFilter(oldObject, newObject), "spec generation changes should trigger")
	newObject = oldObject.DeepCopy()
	newObject.Labels["key"] = "updated"
	assert.True(t, IgnoreStatusChangeEventFilter(oldObject, newObject), "label changes should trigger")
}
