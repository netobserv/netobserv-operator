package config

import (
	_ "embed"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"gopkg.in/yaml.v2"
)

type ServerConfig struct {
	Port        int    `yaml:"port,omitempty" json:"port,omitempty"`
	MetricsPort int    `yaml:"metricsPort,omitempty" json:"metricsPort,omitempty"`
	CertPath    string `yaml:"certPath,omitempty" json:"certPath,omitempty"`
	KeyPath     string `yaml:"keyPath,omitempty" json:"keyPath,omitempty"`
	CORSOrigin  string `yaml:"corsOrigin,omitempty" json:"corsOrigin,omitempty"`
	CORSMethods string `yaml:"corsMethods,omitempty" json:"corsMethods,omitempty"`
	CORSHeaders string `yaml:"corsHeaders,omitempty" json:"corsHeaders,omitempty"`
	CORSMaxAge  string `yaml:"corsMaxAge,omitempty" json:"corsMaxAge,omitempty"`
	AuthCheck   string `yaml:"authCheck,omitempty" json:"authCheck,omitempty"`
}

type LokiConfig struct {
	URL    string   `yaml:"url" json:"url"`
	Labels []string `yaml:"labels" json:"labels"`

	Status             string       `yaml:"status,omitempty" json:"status,omitempty"`
	StatusURL          string       `yaml:"statusUrl,omitempty" json:"statusUrl,omitempty"`
	Timeout            api.Duration `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	TenantID           string       `yaml:"tenantID,omitempty" json:"tenantID,omitempty"`
	TokenPath          string       `yaml:"tokenPath,omitempty" json:"tokenPath,omitempty"`
	SkipTLS            bool         `yaml:"skipTls,omitempty" json:"skipTls,omitempty"`
	CAPath             string       `yaml:"caPath,omitempty" json:"caPath,omitempty"`
	StatusSkipTLS      bool         `yaml:"statusSkipTls,omitempty" json:"statusSkipTls,omitempty"`
	StatusCAPath       string       `yaml:"statusCaPath,omitempty" json:"statusCaPath,omitempty"`
	StatusUserCertPath string       `yaml:"statusUserCertPath,omitempty" json:"statusUserCertPath,omitempty"`
	StatusUserKeyPath  string       `yaml:"statusUserKeyPath,omitempty" json:"statusUserKeyPath,omitempty"`
	UseMocks           bool         `yaml:"useMocks,omitempty" json:"useMocks,omitempty"`
	ForwardUserToken   bool         `yaml:"forwardUserToken,omitempty" json:"forwardUserToken,omitempty"`
}

type PrometheusConfig struct {
	URL              string             `yaml:"url" json:"url"`
	DevURL           string             `yaml:"devUrl,omitempty" json:"devUrl,omitempty"`
	AlertManager     AlertManagerConfig `yaml:"alertManager" json:"alertManager"`
	Timeout          api.Duration       `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	TokenPath        string             `yaml:"tokenPath,omitempty" json:"tokenPath,omitempty"`
	SkipTLS          bool               `yaml:"skipTls,omitempty" json:"skipTls,omitempty"`
	CAPath           string             `yaml:"caPath,omitempty" json:"caPath,omitempty"`
	ForwardUserToken bool               `yaml:"forwardUserToken,omitempty" json:"forwardUserToken,omitempty"`
	Metrics          []MetricInfo       `yaml:"metrics,omitempty" json:"metrics,omitempty"`
}

type AlertManagerConfig struct {
	URL     string `yaml:"url" json:"url"`
	SkipTLS bool   `yaml:"skipTls,omitempty" json:"skipTls,omitempty"`
	CAPath  string `yaml:"caPath,omitempty" json:"caPath,omitempty"`
}

type MetricInfo struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Type       string   `yaml:"type,omitempty" json:"type,omitempty"`
	ValueField string   `yaml:"valueField,omitempty" json:"valueField,omitempty"`
	Direction  string   `yaml:"direction,omitempty" json:"direction,omitempty"`
	Labels     []string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type ColumnConfig struct {
	ID   string `yaml:"id" json:"id"`
	Name string `yaml:"name" json:"name"`

	Group      string   `yaml:"group,omitempty" json:"group,omitempty"`
	Field      string   `yaml:"field,omitempty" json:"field,omitempty"`
	Fields     []string `yaml:"fields,omitempty" json:"fields,omitempty"`
	Calculated string   `yaml:"calculated,omitempty" json:"calculated,omitempty"`
	Tooltip    string   `yaml:"tooltip,omitempty" json:"tooltip,omitempty"`
	DocURL     string   `yaml:"docURL,omitempty" json:"docURL,omitempty"`
	Filter     string   `yaml:"filter,omitempty" json:"filter,omitempty"`
	Default    bool     `yaml:"default,omitempty" json:"default,omitempty"`
	Width      int      `yaml:"width,omitempty" json:"width,omitempty"`
	Feature    string   `yaml:"feature" json:"feature"`
}

type FilterConfig struct {
	ID        string `yaml:"id" json:"id"`
	Name      string `yaml:"name" json:"name"`
	Component string `yaml:"component" json:"component"`

	Category               string `yaml:"category,omitempty" json:"category,omitempty"`
	AutoCompleteAddsQuotes bool   `yaml:"autoCompleteAddsQuotes,omitempty" json:"autoCompleteAddsQuotes,omitempty"`
	Hint                   string `yaml:"hint,omitempty" json:"hint,omitempty"`
	Examples               string `yaml:"examples,omitempty" json:"examples,omitempty"`
	DocURL                 string `yaml:"docUrl,omitempty" json:"docUrl,omitempty"`
	Placeholder            string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Feature                string `yaml:"feature,omitempty" json:"feature,omitempty"`
}

type ScopeConfig struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	ShortName   string   `yaml:"shortName" json:"shortName"`
	Description string   `yaml:"description" json:"description"`
	Labels      []string `yaml:"labels" json:"labels"`
	Feature     string   `yaml:"feature,omitempty" json:"feature,omitempty"`
	Groups      []string `yaml:"groups,omitempty" json:"groups,omitempty"`
	Filter      string   `yaml:"filter,omitempty" json:"filter,omitempty"`
	Filters     []string `yaml:"filters,omitempty" json:"filters,omitempty"`
	StepInto    string   `yaml:"stepInto,omitempty" json:"stepInto,omitempty"`
}

type FieldConfig struct {
	Name        string `yaml:"name" json:"name"`
	Type        string `yaml:"type" json:"type"`
	Format      string `yaml:"format" json:"format"`
	Description string `yaml:"description" json:"description"`
	LokiLabel   bool   `yaml:"lokiLabel,omitempty" json:"lokiLabel,omitempty"`
}

type FrontendConfig struct {
	RecordTypes          []api.ConnTrackOutputRecordTypeEnum `yaml:"recordTypes" json:"recordTypes"`
	PortNaming           flowslatest.ConsolePluginPortConfig `yaml:"portNaming,omitempty" json:"portNaming,omitempty"`
	Columns              []ColumnConfig                      `yaml:"columns" json:"columns"`
	Filters              []FilterConfig                      `yaml:"filters,omitempty" json:"filters,omitempty"`
	Scopes               []ScopeConfig                       `yaml:"scopes" json:"scopes"`
	QuickFilters         []flowslatest.QuickFilter           `yaml:"quickFilters,omitempty" json:"quickFilters,omitempty"`
	AlertNamespaces      []string                            `yaml:"alertNamespaces,omitempty" json:"alertNamespaces,omitempty"`
	Sampling             int                                 `yaml:"sampling" json:"sampling"`
	Features             []string                            `yaml:"features" json:"features"`
	Fields               []FieldConfig                       `yaml:"fields" json:"fields"`
	RecordingAnnotations map[string]map[string]string        `yaml:"recordingAnnotations,omitempty" json:"recordingAnnotations,omitempty"`
}

type PluginConfig struct {
	Server     ServerConfig     `yaml:"server" json:"server"`
	Loki       LokiConfig       `yaml:"loki" json:"loki"`
	Prometheus PrometheusConfig `yaml:"prometheus" json:"prometheus"`
	Frontend   FrontendConfig   `yaml:"frontend" json:"frontend"`
}

//go:embed static-frontend-config.yaml
var rawStaticFrontendConfig []byte
var staticFrontendConfig *FrontendConfig

func GetStaticFrontendConfig() (FrontendConfig, error) {
	if staticFrontendConfig == nil {
		cfg := FrontendConfig{}
		err := yaml.Unmarshal(rawStaticFrontendConfig, &cfg)
		if err != nil {
			return cfg, err
		}
		staticFrontendConfig = &cfg
	}
	return *staticFrontendConfig, nil
}
