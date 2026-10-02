package narrowcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestDynamicSourceActivationCanBeReset(t *testing.T) {
	registry := idempotentSources{registered: make(map[string]map[string]bool)}
	obj := &corev1.ConfigMap{}

	assert.False(t, registry.setActive("dynamic", "v1/ConfigMap", obj), "first registration should be new")
	assert.True(t, registry.isActive("dynamic", "v1/ConfigMap", obj))

	registry.resetActive("dynamic")
	assert.False(t, registry.isActive("dynamic", "v1/ConfigMap", obj), "reset should deactivate old names")

	assert.True(t, registry.setActive("dynamic", "v1/ConfigMap", obj), "reactivation should reuse the existing source")
	assert.True(t, registry.isActive("dynamic", "v1/ConfigMap", obj))
}
