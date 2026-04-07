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

const (
	testPluginImagesRaw         = "4.14.0=" + pf4Image + ";4.15.0=" + pf5Image + ";4.22.0=" + pf6Image
	testPluginImagesWithDefault = "default=" + pf4Image + ";4.15.0=" + pf5Image + ";4.22.0=" + pf6Image
)

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
	assert.Equal(t, "4.14.0", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, pf4Image, cfg.ConsolePluginImageVariants[0].Image)
	assert.Equal(t, "4.15.0", cfg.ConsolePluginImageVariants[1].MinVersion)
	assert.Equal(t, pf5Image, cfg.ConsolePluginImageVariants[1].Image)
	assert.Equal(t, "4.22.0", cfg.ConsolePluginImageVariants[2].MinVersion)
	assert.Equal(t, pf6Image, cfg.ConsolePluginImageVariants[2].Image)
}

func TestParseConsolePluginImages_SingleEntry(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("4.14.0=registry/img:latest")
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 1)
	assert.Equal(t, "4.14.0", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, "registry/img:latest", cfg.ConsolePluginImageVariants[0].Image)
}

func TestParseConsolePluginImages_SingleImageNoPrefix(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("quay.io/netobserv/console-plugin:main")
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 1)
	assert.Equal(t, "default", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, "quay.io/netobserv/console-plugin:main", cfg.ConsolePluginImageVariants[0].Image)
}

func TestParseConsolePluginImages_DefaultWithVersions(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages(testPluginImagesWithDefault)
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 3)
	assert.Equal(t, "default", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, pf4Image, cfg.ConsolePluginImageVariants[0].Image)
	assert.Equal(t, "4.15.0", cfg.ConsolePluginImageVariants[1].MinVersion)
	assert.Equal(t, pf5Image, cfg.ConsolePluginImageVariants[1].Image)
}

func TestParseConsolePluginImages_Empty(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("")
	require.NoError(t, err)
	assert.Empty(t, cfg.ConsolePluginImageVariants)
}

func TestParseConsolePluginImages_NoEqualsIsSingleImage(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("registry/image:tag")
	require.NoError(t, err)
	require.Len(t, cfg.ConsolePluginImageVariants, 1)
	assert.Equal(t, "default", cfg.ConsolePluginImageVariants[0].MinVersion)
	assert.Equal(t, "registry/image:tag", cfg.ConsolePluginImageVariants[0].Image)
}

func TestParseConsolePluginImages_InvalidNoImage(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseConsolePluginImages("4.14.0=")
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
	_, err := cfg.ResolveConsolePluginImage(info)
	require.Error(t, err, "below minimum variant; should error")
	assert.Contains(t, err.Error(), "no console plugin image variant matches")
}

func TestResolveConsolePluginImage_OCP414(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.14.9", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf4Image, img)
}

func TestResolveConsolePluginImage_OCP415(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.15.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf5Image, img)
}

func TestResolveConsolePluginImage_OCP418(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.18.3", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf5Image, img)
}

func TestResolveConsolePluginImage_OCP421(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.21.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf5Image, img)
}

func TestResolveConsolePluginImage_OCP422(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.22.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf6Image, img)
}

func TestResolveConsolePluginImage_OCP425(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("4.25.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf6Image, img)
}

func TestResolveConsolePluginImage_NonOpenShift(t *testing.T) {
	cfg := testConfig(t)
	info := &cluster.Info{}
	info.Mock("", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf4Image, img, "should default to first versioned entry (baseline)")
}

func TestResolveConsolePluginImage_NonOpenShiftWithDefault(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages(testPluginImagesWithDefault))
	info := &cluster.Info{}
	info.Mock("", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf4Image, img, "should use default= entry")
}

func TestResolveConsolePluginImage_OCPFallsBackToDefault(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages(testPluginImagesWithDefault))
	info := &cluster.Info{}
	info.Mock("4.13.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf4Image, img, "OCP below minimum versioned entry should fall back to default=")
}

func TestResolveConsolePluginImage_OCPMatchesVersionOverDefault(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages(testPluginImagesWithDefault))
	info := &cluster.Info{}
	info.Mock("4.22.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, pf6Image, img, "OCP matching a versioned entry should pick it over default=")
}

func TestResolveConsolePluginImage_SingleImage(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages("quay.io/netobserv/console-plugin:main"))
	info := &cluster.Info{}
	info.Mock("4.18.0", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, "quay.io/netobserv/console-plugin:main", img)
}

func TestResolveConsolePluginImage_SingleImageNonOpenShift(t *testing.T) {
	cfg := &Config{}
	require.NoError(t, cfg.ParseConsolePluginImages("quay.io/netobserv/console-plugin:main"))
	info := &cluster.Info{}
	info.Mock("", "")
	img, err := cfg.ResolveConsolePluginImage(info)
	require.NoError(t, err)
	assert.Equal(t, "quay.io/netobserv/console-plugin:main", img)
}

func TestResolveConsolePluginImage_NoVariants(t *testing.T) {
	cfg := &Config{}
	info := &cluster.Info{}
	info.Mock("4.18.0", "")
	_, err := cfg.ResolveConsolePluginImage(info)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no console plugin image variants configured")
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
	}
	require.NoError(t, cfg.ParseConsolePluginImages("4.14.0=plugin:test"))
	assert.NoError(t, cfg.Validate())
}

func TestValidate_SingleImage(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
	}
	require.NoError(t, cfg.ParseConsolePluginImages("registry/plugin:tag"))
	assert.NoError(t, cfg.Validate())
}

func TestValidate_DefaultWithVersions(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
	}
	require.NoError(t, cfg.ParseConsolePluginImages("default=plugin:pf4;4.15.0=plugin:pf5"))
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

func TestValidate_InvalidMinVersion(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
		ConsolePluginImageVariants: []ConsolePluginImageVariant{
			{MinVersion: "not-a-version", Image: "plugin:test"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid MinVersion")
}

func TestValidate_MisorderedMinVersions(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
		ConsolePluginImageVariants: []ConsolePluginImageVariant{
			{MinVersion: "4.15.0", Image: "plugin:pf5"},
			{MinVersion: "4.14.0", Image: "plugin:pf4"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be strictly greater than previous")
}

func TestValidate_DuplicateMinVersions(t *testing.T) {
	cfg := &Config{
		EBPFAgentImage:        "agent:test",
		FlowlogsPipelineImage: "flp:test",
		Namespace:             "netobserv",
		ConsolePluginImageVariants: []ConsolePluginImageVariant{
			{MinVersion: "4.15.0", Image: "plugin:a"},
			{MinVersion: "4.15.0", Image: "plugin:b"},
		},
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be strictly greater than previous")
}
