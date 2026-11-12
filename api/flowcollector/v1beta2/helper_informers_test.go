package v1beta2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/ptr"
)

func TestInformerCacheProxyEnabledByDefault(t *testing.T) {
	for _, tc := range []struct {
		name    string
		proxy   *FlowCollectorInformerCacheProxy
		enabled bool
	}{
		{name: "omitted", enabled: true},
		{name: "empty config", proxy: &FlowCollectorInformerCacheProxy{}, enabled: true},
		{name: "custom config without enabled", proxy: &FlowCollectorInformerCacheProxy{Replicas: ptr.To(int32(3))}, enabled: true},
		{name: "explicitly enabled", proxy: &FlowCollectorInformerCacheProxy{Enabled: ptr.To(true)}, enabled: true},
		{name: "explicitly disabled", proxy: &FlowCollectorInformerCacheProxy{Enabled: ptr.To(false)}, enabled: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := FlowCollectorFLP{InformerCacheProxy: tc.proxy}
			assert.Equal(t, tc.enabled, spec.IsInformerCacheProxyEnabled())
		})
	}
}
