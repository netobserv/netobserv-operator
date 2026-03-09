package helper

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/stretchr/testify/assert"
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
