package v1beta2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func proxySource(name string, port int32) FlowCollectorProxySource {
	return FlowCollectorProxySource{Name: name, Type: ProxySourceHAProxy, HAProxy: &FlowCollectorHAProxySource{Port: port}}
}

func TestProxyCorrelationValidation(t *testing.T) {
	previous := CurrentClusterInfo
	t.Cleanup(func() { CurrentClusterInfo = previous })
	for _, tc := range []struct {
		name                     string
		enable, loki, kubernetes bool
		sources                  []FlowCollectorProxySource
		wantError                string
	}{
		{name: "disabled without Loki"},
		{name: "requires Loki", enable: true, sources: []FlowCollectorProxySource{proxySource("router", 0)}, wantError: "requires Loki"},
		{name: "requires source", enable: true, loki: true, wantError: "at least one source"},
		{name: "default port", enable: true, loki: true, sources: []FlowCollectorProxySource{proxySource("router", 0)}},
		{name: "two receivers", enable: true, loki: true, sources: []FlowCollectorProxySource{proxySource("router", 0), proxySource("other", 11514)}},
		{name: "privileged port", enable: true, loki: true, sources: []FlowCollectorProxySource{proxySource("router", 514)}, wantError: "between 1024 and 65535"},
		{name: "invalid port", enable: true, loki: true, sources: []FlowCollectorProxySource{proxySource("router", 65536)}, wantError: "between 1024 and 65535"},
		{name: "duplicate name", sources: []FlowCollectorProxySource{proxySource("router", 10514), proxySource("router", 11514)}, wantError: "name must be unique"},
		{name: "duplicate default port", sources: []FlowCollectorProxySource{proxySource("router", 0), proxySource("other", 10514)}, wantError: "receiver port 10514 must be unique"},
		{name: "invalid name", sources: []FlowCollectorProxySource{proxySource("Bad_Name", 0)}, wantError: "sources[0].name"},
		{name: "missing name", sources: []FlowCollectorProxySource{proxySource("", 0)}, wantError: "sources[0].name"},
		{name: "missing type", sources: []FlowCollectorProxySource{{Name: "router", HAProxy: &FlowCollectorHAProxySource{}}}, wantError: "unsupported proxy source type"},
		{name: "unsupported type", sources: []FlowCollectorProxySource{{Name: "router", Type: "Envoy"}}, wantError: "unsupported proxy source type"},
		{name: "missing settings", sources: []FlowCollectorProxySource{{Name: "router", Type: ProxySourceHAProxy}}, wantError: "haproxy is required"},
		{name: "requires OpenShift", enable: true, loki: true, kubernetes: true, sources: []FlowCollectorProxySource{proxySource("router", 0)}, wantError: "require OpenShift"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			CurrentClusterInfo = &clusterInfoMock{version: "4.22.0"}
			if tc.kubernetes {
				CurrentClusterInfo = &clusterInfoMock{}
			}
			desired := &FlowCollectorSpec{Processor: FlowCollectorFLP{ProxyCorrelation: &FlowCollectorProxyCorrelation{Enable: tc.enable, Sources: tc.sources}}, Loki: FlowCollectorLoki{Enable: ptr.To(tc.loki)}}
			v := validator{fc: desired}
			v.validateProxyCorrelation()
			err := errors.Join(v.errors...)
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
			}
		})
	}
	require.False(t, (&FlowCollectorFLP{}).IsProxyCorrelationEnabled())
	require.Empty(t, (&FlowCollectorFLP{}).GetProxySources())
	source := proxySource("router", 0)
	require.EqualValues(t, 10514, source.GetPort())
}
