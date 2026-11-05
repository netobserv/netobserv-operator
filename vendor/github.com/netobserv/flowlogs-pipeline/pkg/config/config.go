/*
 * Copyright (C) 2021 IBM, Inc.
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

package config

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/sirupsen/logrus"
)

type Options struct {
	PipeLine          string
	Parameters        string
	DynamicParameters string
	MetricsSettings   string
	HealthAddr        string
	PprofAddr         string
	K8sCacheServer    K8sCacheServer
}

type Root struct {
	LogLevel          string            `yaml:"log-level,omitempty" json:"log-level,omitempty"`
	MetricsSettings   MetricsSettings   `yaml:"metricsSettings,omitempty" json:"metricsSettings,omitempty"`
	Pipeline          []Stage           `yaml:"pipeline,omitempty" json:"pipeline,omitempty"`
	Parameters        []StageParam      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	PerfSettings      PerfSettings      `yaml:"perfSettings,omitempty" json:"perfSettings,omitempty"`
	DynamicParameters DynamicParameters `yaml:"dynamicParameters,omitempty" json:"dynamicParameters,omitempty"`
}

type DynamicParameters struct {
	Namespace      string `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Name           string `yaml:"name,omitempty" json:"name,omitempty"`
	FileName       string `yaml:"fileName,omitempty" json:"fileName,omitempty"`
	KubeConfigPath string `yaml:"kubeConfigPath,omitempty" json:"kubeConfigPath,omitempty" doc:"path to kubeconfig file (optional)"`
}

type HotReloadStruct struct {
	Parameters []StageParam `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

type K8sCacheServer struct {
	Address string
	Port    int
	// TLS configuration
	TLSEnabled  bool
	TLSCertPath string
	TLSKeyPath  string
	TLSCAPath   string
}

// MetricsSettings is similar to api.PromEncode, but is global to the application, ie. it also works with operational metrics.
// Also, currently FLP doesn't support defining more than one PromEncode stage. If this feature is added later, these global settings
// will help configuring common setting for all PromEncode stages - PromEncode settings would then act as overrides.
type MetricsSettings struct {
	api.PromConnectionInfo `yaml:",inline"`
	DisableGlobalServer    bool   `yaml:"disableGlobalServer,omitempty" json:"disableGlobalServer,omitempty" doc:"disabling the global metrics server makes operational metrics unavailable. If prometheus-encoding stages are defined, they need to contain their own metrics server parameters."`
	Prefix                 string `yaml:"prefix,omitempty" json:"prefix,omitempty" doc:"prefix for names of the operational metrics"`
	NoPanic                bool   `yaml:"noPanic,omitempty" json:"noPanic,omitempty"`
	SuppressDefaultMetrics bool   `yaml:"suppressDefaultMetrics,omitempty" json:"suppressDefaultMetrics,omitempty" doc:"filter out default Prometheus metrics such as Go and process metrics"`
}

// PerfSettings allows setting some internal configuration parameters
type PerfSettings struct {
	BatcherMaxLen  int           `yaml:"batcherMaxLen,omitempty" json:"batcherMaxLen,omitempty"`
	BatcherTimeout time.Duration `yaml:"batcherMaxTimeout,omitempty" json:"batcherMaxTimeout,omitempty"`
	NodeBufferLen  int           `yaml:"nodeBufferLen,omitempty" json:"nodeBufferLen,omitempty"`
}

type Stage struct {
	Name    string `yaml:"name" json:"name"`
	Follows string `yaml:"follows,omitempty" json:"follows,omitempty"`
}

type StageParam struct {
	Name      string     `yaml:"name" json:"name"`
	Ingest    *Ingest    `yaml:"ingest,omitempty" json:"ingest,omitempty"`
	Transform *Transform `yaml:"transform,omitempty" json:"transform,omitempty"`
	Extract   *Extract   `yaml:"extract,omitempty" json:"extract,omitempty"`
	Encode    *Encode    `yaml:"encode,omitempty" json:"encode,omitempty"`
	Write     *Write     `yaml:"write,omitempty" json:"write,omitempty"`
	isDynamic bool       `yaml:"-" json:"-"`
}

type Ingest struct {
	HAProxy   *api.IngestHAProxy   `yaml:"haproxy,omitempty" json:"haproxy,omitempty"`
	Type      string               `yaml:"type" json:"type"`
	File      *File                `yaml:"file,omitempty" json:"file,omitempty"`
	Collector *api.IngestIpfix     `yaml:"collector,omitempty" json:"collector,omitempty"` // deprecated: use 'ipfix' instead
	Ipfix     *api.IngestIpfix     `yaml:"ipfix,omitempty" json:"ipfix,omitempty"`
	Kafka     *api.IngestKafka     `yaml:"kafka,omitempty" json:"kafka,omitempty"`
	GRPC      *api.IngestGRPCProto `yaml:"grpc,omitempty" json:"grpc,omitempty"`
	Synthetic *api.IngestSynthetic `yaml:"synthetic,omitempty" json:"synthetic,omitempty"`
	Stdin     *api.IngestStdin     `yaml:"stdin,omitempty" json:"stdin,omitempty"`
}

type File struct {
	Filename string      `yaml:"filename" json:"filename"`
	Decoder  api.Decoder `yaml:"decoder" json:"decoder"`
	Loop     bool        `yaml:"loop" json:"loop"`
	Chunks   int         `yaml:"chunks" json:"chunks"`
}

type Transform struct {
	Type    string                `yaml:"type" json:"type"`
	Generic *api.TransformGeneric `yaml:"generic,omitempty" json:"generic,omitempty"`
	Filter  *api.TransformFilter  `yaml:"filter,omitempty" json:"filter,omitempty"`
	Network *api.TransformNetwork `yaml:"network,omitempty" json:"network,omitempty"`
}

type Extract struct {
	Type       string                `yaml:"type" json:"type"`
	Aggregates *api.Aggregates       `yaml:"aggregates,omitempty" json:"aggregates,omitempty"`
	ConnTrack  *api.ConnTrack        `yaml:"conntrack,omitempty" json:"conntrack,omitempty"`
	Timebased  *api.ExtractTimebased `yaml:"timebased,omitempty" json:"timebased,omitempty"`
}

type Encode struct {
	Type        string                 `yaml:"type" json:"type"`
	Prom        *api.PromEncode        `yaml:"prom,omitempty" json:"prom,omitempty"`
	Kafka       *api.EncodeKafka       `yaml:"kafka,omitempty" json:"kafka,omitempty"`
	S3          *api.EncodeS3          `yaml:"s3,omitempty" json:"s3,omitempty"`
	OtlpLogs    *api.EncodeOtlpLogs    `yaml:"otlplogs,omitempty" json:"otlplogs,omitempty"`
	OtlpMetrics *api.EncodeOtlpMetrics `yaml:"otlpmetrics,omitempty" json:"otlpmetrics,omitempty"`
	OtlpTraces  *api.EncodeOtlpTraces  `yaml:"otlptraces,omitempty" json:"otlptraces,omitempty"`
}

type Write struct {
	Type   string           `yaml:"type" json:"type"`
	Loki   *api.WriteLoki   `yaml:"loki,omitempty" json:"loki,omitempty"`
	Stdout *api.WriteStdout `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	Ipfix  *api.WriteIpfix  `yaml:"ipfix,omitempty" json:"ipfix,omitempty"`
	GRPC   *api.WriteGRPC   `yaml:"grpc,omitempty" json:"grpc,omitempty"`
}

// ParseConfig creates the internal unmarshalled representation from the Pipeline and Parameters json
func ParseConfig(opts *Options) (Root, error) {
	out := Root{}

	logrus.Debugf("opts.PipeLine = %v ", opts.PipeLine)
	err := JSONUnmarshalStrict([]byte(opts.PipeLine), &out.Pipeline)
	if err != nil {
		logrus.Errorf("error when parsing pipeline: %v", err)
		return out, err
	}
	logrus.Debugf("stages = %v ", out.Pipeline)

	err = JSONUnmarshalStrict([]byte(opts.Parameters), &out.Parameters)
	if err != nil {
		logrus.Errorf("error when parsing pipeline parameters: %v", err)
		return out, err
	}
	logrus.Debugf("params = %v ", out.Parameters)

	if opts.DynamicParameters != "" {
		err = JSONUnmarshalStrict([]byte(opts.DynamicParameters), &out.DynamicParameters)
		if err != nil {
			logrus.Errorf("error when parsing dynamic pipeline parameters: %v", err)
			return out, err
		}
		logrus.Debugf("dynamicParams = %v ", out.DynamicParameters)
	}

	if opts.MetricsSettings != "" {
		err = JSONUnmarshalStrict([]byte(opts.MetricsSettings), &out.MetricsSettings)
		if err != nil {
			logrus.Errorf("error when parsing global metrics settings: %v", err)
			return out, err
		}
		logrus.Debugf("metrics settings = %v ", out.MetricsSettings)
	} else {
		logrus.Infof("using default metrics settings")
	}

	return out, nil
}

// JsonUnmarshalStrict is like Unmarshal except that any fields that are found
// in the data that do not have corresponding struct members, or mapping
// keys that are duplicates, will result in an error.
func JSONUnmarshalStrict(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
