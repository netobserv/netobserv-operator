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
	"errors"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
)

type pipeline struct {
	stages []Stage
	config []StageParam
}

// PipelineBuilderStage holds information about a created pipeline stage. This stage can be used to chain a following stage, or several of them (resulting in a fork).
// Example:
//
//	firstStage := NewCollectorPipeline("first stage", ...)
//	secondStage := firstStage.DecodeJSON("second stage")
//	thirdStage := secondStage.WriteLoki("third stage", ...)
//	forkedStage := secondStage.WriteStdout("fork following second stage", ...)
//
// All created stages hold a pointer to the whole pipeline, so that the resulting pipeline can be retrieve from any of the stages:
//
//	forkedStage.GetStages()
//	forkedStage.GetStageParams()
//	// is equivalent to:
//	firstStage.GetStages()
//	firstStage.GetStageParams()
type PipelineBuilderStage struct {
	lastStage string
	pipeline  *pipeline
}

const PresetIngesterStage = "preset-ingester"

// NewPipeline creates a new pipeline from an existing ingest
func NewPipeline(name string, ingest *Ingest) (PipelineBuilderStage, error) {
	if ingest.HAProxy != nil {
		return NewHAProxyPipeline(name, *ingest.HAProxy), nil
	}
	if ingest.Ipfix != nil {
		return NewIPFIXPipeline(name, *ingest.Ipfix), nil
	}
	if ingest.Collector != nil { // for retro-compatibility
		return NewIPFIXPipeline(name, *ingest.Collector), nil
	}
	if ingest.GRPC != nil {
		return NewGRPCPipeline(name, *ingest.GRPC), nil
	}
	if ingest.Kafka != nil {
		return NewKafkaPipeline(name, *ingest.Kafka), nil
	}
	return PipelineBuilderStage{}, errors.New("missing ingest params")
}

// NewCollectorPipeline creates a new pipeline from an `IngestIpfix` initial stage (listening for NetFlows / IPFIX)
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func NewIPFIXPipeline(name string, ingest api.IngestIpfix) PipelineBuilderStage {
	p := pipeline{
		stages: []Stage{{Name: name}},
		config: []StageParam{NewIPFIXParams(name, ingest)},
	}
	return PipelineBuilderStage{pipeline: &p, lastStage: name}
}

// NewHAProxyPipeline creates a pipeline receiving versioned HAProxy access logs.
func NewHAProxyPipeline(name string, ingest api.IngestHAProxy) PipelineBuilderStage {
	p := pipeline{stages: []Stage{{Name: name}}, config: []StageParam{
		{Name: name, Ingest: &Ingest{Type: api.HAProxyType, HAProxy: &ingest}},
	}}
	return PipelineBuilderStage{pipeline: &p, lastStage: name}
}

// NewGRPCPipeline creates a pipeline listening for NetObserv agent protobuf.
func NewGRPCPipeline(name string, ingest api.IngestGRPCProto) PipelineBuilderStage {
	p := pipeline{
		stages: []Stage{{Name: name}},
		config: []StageParam{NewGRPCParams(name, ingest)},
	}
	return PipelineBuilderStage{pipeline: &p, lastStage: name}
}

// NewKafkaPipeline creates a new pipeline from an `IngestKafka` initial stage (listening for flow events on Kafka)
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func NewKafkaPipeline(name string, ingest api.IngestKafka) PipelineBuilderStage {
	p := pipeline{
		stages: []Stage{{Name: name}},
		config: []StageParam{NewKafkaParams(name, ingest)},
	}
	return PipelineBuilderStage{pipeline: &p, lastStage: name}
}

// NewPresetIngesterPipeline creates a new partial pipeline without ingest stage
func NewPresetIngesterPipeline() PipelineBuilderStage {
	p := pipeline{
		stages: []Stage{},
		config: []StageParam{},
	}
	return PipelineBuilderStage{pipeline: &p, lastStage: PresetIngesterStage}
}

func (b *PipelineBuilderStage) next(name string, param StageParam, opts ...StageParamOption) PipelineBuilderStage {
	p := &param
	for _, o := range opts {
		p = o(p)
	}
	b.pipeline.stages = append(b.pipeline.stages, Stage{Name: name, Follows: b.lastStage})
	b.pipeline.config = append(b.pipeline.config, param)
	return PipelineBuilderStage{pipeline: b.pipeline, lastStage: name}
}

// Aggregate chains the current stage with an aggregate stage and returns that new stage
func (b *PipelineBuilderStage) Aggregate(name string, aggs api.Aggregates) PipelineBuilderStage {
	return b.next(name, NewAggregateParams(name, aggs))
}

// ExtractTimebased chains the current stage with a ExtractTimebased stage and returns that new stage
func (b *PipelineBuilderStage) ExtractTimebased(name string, tb api.ExtractTimebased) PipelineBuilderStage {
	return b.next(name, StageParam{Name: name, Extract: &Extract{Type: api.TimebasedType, Timebased: &tb}})
}

// TransformGeneric chains the current stage with a TransformGeneric stage and returns that new stage
func (b *PipelineBuilderStage) TransformGeneric(name string, gen api.TransformGeneric) PipelineBuilderStage {
	return b.next(name, NewTransformGenericParams(name, gen))
}

// TransformFilter chains the current stage with a TransformFilter stage and returns that new stage
func (b *PipelineBuilderStage) TransformFilter(name string, filter api.TransformFilter, opts ...StageParamOption) PipelineBuilderStage {
	return b.next(name, NewTransformFilterParams(name, filter), opts...)
}

// TransformNetwork chains the current stage with a TransformNetwork stage and returns that new stage
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func (b *PipelineBuilderStage) TransformNetwork(name string, nw api.TransformNetwork, opts ...StageParamOption) PipelineBuilderStage {
	return b.next(name, NewTransformNetworkParams(name, nw), opts...)
}

// ConnTrack chains the current stage with a ConnTrack stage and returns that new stage
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func (b *PipelineBuilderStage) ConnTrack(name string, ct api.ConnTrack) PipelineBuilderStage {
	return b.next(name, NewConnTrackParams(name, ct))
}

// EncodePrometheus chains the current stage with a PromEncode stage (to expose metrics in Prometheus format) and returns that new stage
func (b *PipelineBuilderStage) EncodePrometheus(name string, prom api.PromEncode, opts ...StageParamOption) PipelineBuilderStage {
	return b.next(name, NewEncodePrometheusParams(name, prom), opts...)
}

// EncodeKafka chains the current stage with an EncodeKafka stage (writing to a Kafka topic) and returns that new stage
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func (b *PipelineBuilderStage) EncodeKafka(name string, kafka api.EncodeKafka) PipelineBuilderStage {
	return b.next(name, NewEncodeKafkaParams(name, kafka))
}

// EncodeS3 chains the current stage with an EncodeS3 stage (writing to s3 bucket) and returns that new stage
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func (b *PipelineBuilderStage) EncodeS3(name string, s3 api.EncodeS3) PipelineBuilderStage {
	return b.next(name, NewEncodeS3Params(name, s3))
}

// EncodeOtelLogs chains the current stage with an EncodeOtelLogs stage (writing logs to open telemetry) and returns that new stage
func (b *PipelineBuilderStage) EncodeOtelLogs(name string, logs api.EncodeOtlpLogs) PipelineBuilderStage {
	return b.next(name, NewEncodeOtelLogsParams(name, logs))
}

// EncodeOtelMetrics chains the current stage with an EncodeOtelMetrics stage (writing metrics to open telemetry) and returns that new stage
func (b *PipelineBuilderStage) EncodeOtelMetrics(name string, metrics api.EncodeOtlpMetrics) PipelineBuilderStage {
	return b.next(name, NewEncodeOtelMetricsParams(name, metrics))
}

// EncodeOtelTraces chains the current stage with an EncodeOtelTraces stage (writing traces to open telemetry) and returns that new stage
func (b *PipelineBuilderStage) EncodeOtelTraces(name string, traces api.EncodeOtlpTraces) PipelineBuilderStage {
	return b.next(name, NewEncodeOtelTracesParams(name, traces))
}

// WriteStdout chains the current stage with a WriteStdout stage and returns that new stage
func (b *PipelineBuilderStage) WriteStdout(name string, stdout api.WriteStdout) PipelineBuilderStage {
	return b.next(name, NewWriteStdoutParams(name, stdout))
}

// WriteLoki chains the current stage with a WriteLoki stage and returns that new stage
//
//nolint:gocritic // hugeParam can be ignored: func only used at init
func (b *PipelineBuilderStage) WriteLoki(name string, loki api.WriteLoki) PipelineBuilderStage {
	return b.next(name, NewWriteLokiParams(name, loki))
}

// WriteIpfix chains the current stage with a WriteIpfix stage and returns that new stage
func (b *PipelineBuilderStage) WriteIpfix(name string, ipfix api.WriteIpfix) PipelineBuilderStage {
	return b.next(name, NewWriteIpfixParams(name, ipfix))
}

// GetStages returns the current pipeline stages. It can be called from any of the stages, they share the same pipeline reference.
func (b *PipelineBuilderStage) GetStages() []Stage {
	return b.pipeline.stages
}

// GetStageParams returns the current pipeline stage params. It can be called from any of the stages, they share the same pipeline reference.
func (b *PipelineBuilderStage) GetStageParams() []StageParam {
	return b.pipeline.config
}

func (b *PipelineBuilderStage) GetSplitStageParams() ([]*StageParam, []*StageParam) {
	static := []*StageParam{}
	dynamic := []*StageParam{}
	for _, param := range b.pipeline.config {
		if param.isDynamic {
			dynamic = append(dynamic, &param)
		} else {
			static = append(static, &param)
		}
	}
	return static, dynamic
}

// IntoRootConfig injects the current pipeline and params in the provided config.Root object.
func (b *PipelineBuilderStage) IntoRootConfig(cfs *Root) *Root {
	cfs.Pipeline = b.GetStages()
	cfs.Parameters = b.GetStageParams()
	return cfs
}

// ToRootConfig returns the current pipeline and params as a new config.Root object.
func (b *PipelineBuilderStage) ToRootConfig() *Root {
	return b.IntoRootConfig(&Root{})
}

type StageParamOption func(*StageParam) *StageParam

// Dynamic: this stage config will be whatched and hot-reloaded, without requiring a full restart.
// This is not available for all stage types.
func Dynamic(s *StageParam) *StageParam {
	s.isDynamic = true
	return s
}
