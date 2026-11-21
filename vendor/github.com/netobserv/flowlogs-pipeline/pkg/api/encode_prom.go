/*
 * Copyright (C) 2022 IBM, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package api

type PromTLSConf struct {
	CertPath string `yaml:"certPath,omitempty" json:"certPath,omitempty" doc:"path to the certificate file"`
	KeyPath  string `yaml:"keyPath,omitempty" json:"keyPath,omitempty" doc:"path to the key file"`
}

type PromAuthTypeEnum string

const (
	// For doc generation, enum definitions must match format `Constant Type = "value" // doc`
	PromAuthDisabled   PromAuthTypeEnum = "disabled"   // no authentication (default)
	PromAuthKubernetes PromAuthTypeEnum = "kubernetes" // delegate to the Kubernetes API: callers must provide a bearer token accepted by a TokenReview, and be granted access by a SubjectAccessReview on the requested path
)

// PromAuth configures authentication of the clients scraping the metrics endpoint.
type PromAuth struct {
	Type           PromAuthTypeEnum `yaml:"type,omitempty" json:"type,omitempty" doc:"(enum) authentication mode, one of the following:"`
	KubeConfigPath string           `yaml:"kubeConfigPath,omitempty" json:"kubeConfigPath,omitempty" doc:"path to kubeconfig file, to run outside of a cluster (optional)"`
}

type PromEncode struct {
	*PromConnectionInfo `json:",inline,omitempty" doc:"Prometheus connection info (optional); includes:"`
	Metrics             MetricsItems `yaml:"metrics,omitempty" json:"metrics,omitempty" doc:"list of prometheus metric definitions, each includes:"`
	Prefix              string       `yaml:"prefix,omitempty" json:"prefix,omitempty" doc:"prefix added to each metric name"`
	ExpiryTime          Duration     `yaml:"expiryTime,omitempty" json:"expiryTime,omitempty" doc:"time duration of no-flow to wait before deleting prometheus data item (default: 2m)"`
	MaxMetrics          int          `yaml:"maxMetrics,omitempty" json:"maxMetrics,omitempty" doc:"maximum number of metrics to report (default: unlimited)"`
}

type MetricEncodeOperationEnum string

const (
	// For doc generation, enum definitions must match format `Constant Type = "value" // doc`
	MetricGauge        MetricEncodeOperationEnum = "gauge"         // single numerical value that can arbitrarily go up and down
	MetricCounter      MetricEncodeOperationEnum = "counter"       // monotonically increasing counter whose value can only increase
	MetricHistogram    MetricEncodeOperationEnum = "histogram"     // counts samples in configurable buckets
	MetricAggHistogram MetricEncodeOperationEnum = "agg_histogram" // counts samples in configurable buckets, pre-aggregated via an Aggregate stage
)

type PromConnectionInfo struct {
	Address string       `yaml:"address,omitempty" json:"address,omitempty" doc:"endpoint address to expose"`
	Port    int          `yaml:"port,omitempty" json:"port,omitempty" doc:"endpoint port number to expose"`
	TLS     *PromTLSConf `yaml:"tls,omitempty" json:"tls,omitempty" doc:"TLS configuration for the endpoint"`
	Auth    *PromAuth    `yaml:"auth,omitempty" json:"auth,omitempty" doc:"authentication configuration for the endpoint (optional); includes:"`
}

type MetricsItem struct {
	Name       string                    `yaml:"name" json:"name" doc:"the metric name"`
	Type       MetricEncodeOperationEnum `yaml:"type" json:"type" doc:"(enum) one of the following:"`
	Help       string                    `yaml:"help,omitempty" json:"help,omitempty" doc:"the metric help text"`
	Filters    []MetricsFilter           `yaml:"filters" json:"filters" doc:"a list of criteria to filter entries by"`
	ValueKey   string                    `yaml:"valueKey" json:"valueKey" doc:"entry key from which to resolve metric value; leave empty to count log lines"`
	Labels     []string                  `yaml:"labels" json:"labels" doc:"labels to be associated with the metric"`
	Remap      map[string]string         `yaml:"remap" json:"remap" doc:"optional remapping of labels"`
	Flatten    []string                  `yaml:"flatten" json:"flatten" doc:"list fields to be flattened"`
	Buckets    []float64                 `yaml:"buckets" json:"buckets" doc:"histogram buckets"`
	ValueScale float64                   `yaml:"valueScale,omitempty" json:"valueScale,omitempty" doc:"scale factor of the value (MetricVal := FlowVal / Scale)"`
}

type MetricsItems []MetricsItem
type MetricFilterEnum string

const (
	// For doc generation, enum definitions must match format `Constant Type = "value" // doc`
	MetricFilterEqual    MetricFilterEnum = "equal"           // match exactly the provided filter value
	MetricFilterNotEqual MetricFilterEnum = "not_equal"       // the value must be different from the provided filter
	MetricFilterPresence MetricFilterEnum = "presence"        // filter key must be present (filter value is ignored)
	MetricFilterAbsence  MetricFilterEnum = "absence"         // filter key must be absent (filter value is ignored)
	MetricFilterRegex    MetricFilterEnum = "match_regex"     // match filter value as a regular expression
	MetricFilterNotRegex MetricFilterEnum = "not_match_regex" // the filter value must not match the provided regular expression
)

type MetricsFilter struct {
	Key   string           `yaml:"key" json:"key" doc:"the key to match and filter by"`
	Value string           `yaml:"value" json:"value" doc:"the value to match and filter by"`
	Type  MetricFilterEnum `yaml:"type,omitempty" json:"type,omitempty" doc:"the type of filter match (enum)"`
}
