package helper

import (
	"encoding/json"
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigmapToSecretOrConfig(t *testing.T) {
	assert := assert.New(t)
	file := flowslatest.FileReference{
		Type: flowslatest.RefTypeConfigMap,
		Name: "Foo",
		File: "test.txt",
	}
	res := GetSecretOrConfigMap(&file)
	assert.Nil(res.Secret)
	assert.NotNil(res.ConfigMap)
	assert.Equal(res.ConfigMap.LocalObjectReference.Name, "Foo")
	assert.Equal(res.ConfigMap.Key, "test.txt")
}

func TestSecretToSecretOrConfig(t *testing.T) {
	assert := assert.New(t)
	file := flowslatest.FileReference{
		Type: flowslatest.RefTypeSecret,
		Name: "Foo",
		File: "test.txt",
	}
	res := GetSecretOrConfigMap(&file)
	assert.Nil(res.ConfigMap)
	assert.NotNil(res.Secret)
	assert.Equal(res.Secret.LocalObjectReference.Name, "Foo")
	assert.Equal(res.Secret.Key, "test.txt")
}

func TestServiceMonitorTLSConfigAuto(t *testing.T) {
	scheme, tls := GetServiceMonitorTLSConfig(&flowslatest.ServerTLS{Type: flowslatest.TLSAuto}, "flowlogs-pipeline-prom.netobserv.svc")
	assert.Equal(t, monitoringv1.Scheme("https"), scheme)
	js, err := json.Marshal(tls)
	require.NoError(t, err)
	assert.JSONEq(t, `{"ca":{},"cert":{},"serverName":"flowlogs-pipeline-prom.netobserv.svc","caFile":"/etc/prometheus/configmaps/serving-certs-ca-bundle/service-ca.crt"}`, string(js))
}
