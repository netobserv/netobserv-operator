package flp

import (
	"testing"

	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

func TestHAProxyPipelineIsolationAndDisable(t *testing.T) {
	for _, mode := range []flowslatest.FlowCollectorDeploymentModel{flowslatest.DeploymentModelService, flowslatest.DeploymentModelDirect, flowslatest.DeploymentModelKafka} {
		t.Run(string(mode), func(t *testing.T) {
			desired := getConfig()
			desired.DeploymentModel = mode
			for _, enabled := range []bool{true, false} {
				desired.Processor.ProxyCorrelation = &flowslatest.FlowCollectorProxyCorrelation{Enable: enabled, Sources: []flowslatest.FlowCollectorProxySource{
					{Name: "router", Type: flowslatest.ProxySourceHAProxy, HAProxy: &flowslatest.FlowCollectorHAProxySource{Port: 11514}},
				}}
				var cm *corev1.ConfigMap
				var svc *corev1.Service
				var err error
				if mode == flowslatest.DeploymentModelKafka {
					b := transfBuilder("netobserv", &desired)
					cm, _, _, err = b.configMaps()
					svc = b.promService()
				} else {
					b := monoBuilder("netobserv", &desired)
					cm, _, _, err = b.configMaps()
					svc = b.promService()
				}
				require.NoError(t, err)
				var cfg config.Root
				require.NoError(t, yaml.Unmarshal([]byte(cm.Data[configFile]), &cfg))
				foundInput, foundOutput := false, false
				for _, param := range cfg.Parameters {
					if param.Name == "ingest-proxy-router" {
						foundInput = true
						require.Equal(t, ":11514", param.Ingest.HAProxy.Address)
					}
					if param.Name == "loki-proxy-router" {
						foundOutput = true
						require.Equal(t, "netobserv-proxy", string(param.Write.Loki.StaticLabels["app"]))
						require.Equal(t, "TimeReceived", string(param.Write.Loki.TimestampLabel))
					}
				}
				require.Equal(t, enabled, foundInput)
				require.Equal(t, enabled, foundOutput)
				for _, stage := range cfg.Pipeline {
					if stage.Follows == "ingest-proxy-router" {
						require.Equal(t, "enrich-proxy-router", stage.Name)
					}
					if stage.Follows == "enrich-proxy-router" {
						require.Contains(t, []string{"scope-proxy-router", "loki-proxy-router"}, stage.Name)
					}
				}
				udp := false
				for _, port := range svc.Spec.Ports {
					if port.Name == "proxy-0" {
						udp = true
						require.Equal(t, corev1.ProtocolUDP, port.Protocol)
						require.EqualValues(t, 11514, port.Port)
					}
				}
				require.Equal(t, enabled, udp)
			}
		})
	}
}

func TestMultipleProxySources(t *testing.T) {
	for _, mode := range []flowslatest.FlowCollectorDeploymentModel{flowslatest.DeploymentModelService, flowslatest.DeploymentModelDirect, flowslatest.DeploymentModelKafka} {
		t.Run(string(mode), func(t *testing.T) {
			desired := getConfig()
			desired.DeploymentModel = mode
			desired.Processor.ProxyCorrelation = &flowslatest.FlowCollectorProxyCorrelation{Enable: true, Sources: []flowslatest.FlowCollectorProxySource{
				{Name: "router-a", Type: flowslatest.ProxySourceHAProxy, HAProxy: &flowslatest.FlowCollectorHAProxySource{}},
				{Name: "router-b", Type: flowslatest.ProxySourceHAProxy, HAProxy: &flowslatest.FlowCollectorHAProxySource{Port: 11514}},
			}}
			var cm *corev1.ConfigMap
			var svc *corev1.Service
			var err error
			if mode == flowslatest.DeploymentModelKafka {
				b := transfBuilder("netobserv", &desired)
				cm, _, _, err = b.configMaps()
				svc = b.promService()
			} else {
				b := monoBuilder("netobserv", &desired)
				cm, _, _, err = b.configMaps()
				svc = b.promService()
			}
			require.NoError(t, err)
			var cfg config.Root
			require.NoError(t, yaml.Unmarshal([]byte(cm.Data[configFile]), &cfg))
			inputs := map[string]string{}
			outputs := map[string]bool{}
			names := map[string]bool{}
			for _, param := range cfg.Parameters {
				require.False(t, names[param.Name], "duplicate pipeline stage %s", param.Name)
				names[param.Name] = true
				if param.Ingest != nil && param.Ingest.HAProxy != nil {
					inputs[param.Name] = param.Ingest.HAProxy.Address
				}
				if param.Write != nil && param.Write.Loki != nil && param.Write.Loki.StaticLabels["app"] == "netobserv-proxy" {
					outputs[param.Name] = true
				}
			}
			require.Equal(t, map[string]string{"ingest-proxy-router-a": ":10514", "ingest-proxy-router-b": ":11514"}, inputs)
			require.Equal(t, map[string]bool{"loki-proxy-router-a": true, "loki-proxy-router-b": true}, outputs)
			ports := map[string]int32{}
			for _, port := range svc.Spec.Ports {
				if port.Protocol == corev1.ProtocolUDP {
					ports[port.Name] = port.Port
				}
			}
			require.Equal(t, map[string]int32{"proxy-0": 10514, "proxy-1": 11514}, ports)
		})
	}
}
