package manager

import (
	"testing"

	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pf4Image = "quay.io/netobserv/console-plugin:test-pf4"
	pf5Image = "quay.io/netobserv/console-plugin:test-pf5"
	pf6Image = "quay.io/netobserv/console-plugin:test"
)

const testPluginImagesRaw = "4.0.0=" + pf4Image + ";4.15.0=" + pf5Image + ";4.22.0=" + pf6Image

func testConfig(t *testing.T) *Config {
	t.Helper()
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages(testPluginImagesRaw))
	return cfg
}

func TestParseConsolePluginImages(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages(testPluginImagesRaw)
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 3)
	assert.Equal(t, "4.0.0", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, pf4Image, cfg.ConsolePluginImageVariants[0].Image)
	assert.Equal(t, "4.15.0", cfg.ConsolePluginImageVariants[1].MinVersion)
	assert.Equal(t, pf5Image, cfg.ConsolePluginImageVariants[1].Image)
	assert.Equal(t, "4.22.0", cfg.ConsolePluginImageVariants[2].MinVersion)
	assert.Equal(t, pf6Image, cfg.ConsolePluginImageVariants[2].Image)
}

func TestParseConsolePluginImages_SingleEntry(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("4.0.0=registry/img:latest")
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 1)
	assert.Equal(t, "4.0.0", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, "registry/img:latest", cfg.ConsolePluginImageVariants[0].Image)
}

func TestParseConsolePluginImages_Empty(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("")
	require.NoError(t, err)
	assert.Empty(t, cfg.ConsolePluginImageVariants)
}

func TestParseConsolePluginImages_InvalidNoEquals(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("missing-equals")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected format")
}

func TestParseConsolePluginImages_InvalidNoImage(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("4.0.0=")
	assert.Error(t, err)
}

func TestParseConsolePluginImages_InvalidNoVersion(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("=registry/img:v1")
	assert.Error(t, err)
}

func TestResolveConsolePluginImage_OCP413(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.13.0", "")
	assert.Equal(t, pf4Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP414(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.14.9", "")
	assert.Equal(t, pf4Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP415(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.15.0", "")
	assert.Equal(t, pf5Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP418(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.18.3", "")
	assert.Equal(t, pf5Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP421(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.21.0", "")
	assert.Equal(t, pf5Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP422(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.22.0", "")
	assert.Equal(t, pf6Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_OCP425(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.25.0", "")
	assert.Equal(t, pf6Image, cfg.ResolveConsolePluginImage(info))
}

func TestResolveConsolePluginImage_NonOpenShift(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("", "")
	assert.Equal(t, pf6Image, cfg.ResolveConsolePluginImage(info), "should default to last entry (most current)")
}

func TestResolveConsolePluginImage_NoVariants(t *testing.T) {
	cfg := &Config{}
	info := &cluster.Info{}
	info.Mock("4.18.0", "")
	assert.Equal(t, "", cfg.ResolveConsolePluginImage(info))
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
	}
	require.NoError(t, cfg.ParseConsolePluginImages("4.0.0=plugin:test"))
	assert.NoError(t, cfg.Validate())
}

func TestValidate_NoPluginImages(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
	}
	assert.Error(t, cfg.Validate())
	assert.Contains(t, cfg.Validate().Error(), "console plugin images can't be empty")
}
