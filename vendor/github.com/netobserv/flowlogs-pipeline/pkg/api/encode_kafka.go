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

type EncodeKafka struct {
	Address      string                  `yaml:"address" json:"address" doc:"address of kafka server"`
	Topic        string                  `yaml:"topic" json:"topic" doc:"kafka topic to write to"`
	Balancer     KafkaEncodeBalancerEnum `yaml:"balancer,omitempty" json:"balancer,omitempty" doc:"(enum) one of the following:"`
	WriteTimeout int64                   `yaml:"writeTimeout,omitempty" json:"writeTimeout,omitempty" doc:"timeout (in seconds) for write operation performed by the Writer"`
	ReadTimeout  int64                   `yaml:"readTimeout,omitempty" json:"readTimeout,omitempty" doc:"timeout (in seconds) for read operation performed by the Writer"`
	BatchBytes   int64                   `yaml:"batchBytes,omitempty" json:"batchBytes,omitempty" doc:"limit the maximum size of a request in bytes before being sent to a partition"`
	BatchSize    int                     `yaml:"batchSize,omitempty" json:"batchSize,omitempty" doc:"limit on how many messages will be buffered before being sent to a partition"`
	Compression  string                  `yaml:"compression,omitempty" json:"compression,omitempty" doc:"compression codec: none (default), gzip, snappy, lz4, zstd"`
	TLS          *ClientTLS              `yaml:"tls" json:"tls" doc:"TLS client configuration (optional)"`
	SASL         *SASLConfig             `yaml:"sasl" json:"sasl" doc:"SASL configuration (optional)"`
}

type KafkaEncodeBalancerEnum string

const (
	// For doc generation, enum definitions must match format `Constant Type = "value" // doc`
	KafkaRoundRobin KafkaEncodeBalancerEnum = "roundRobin" // RoundRobin balancer
	KafkaLeastBytes KafkaEncodeBalancerEnum = "leastBytes" // LeastBytes balancer
	KafkaHash       KafkaEncodeBalancerEnum = "hash"       // Hash balancer
	KafkaCrc32      KafkaEncodeBalancerEnum = "crc32"      // Crc32 balancer
	KafkaMurmur2    KafkaEncodeBalancerEnum = "murmur2"    // Murmur2 balancer
)
