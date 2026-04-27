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

import (
	"fmt"
)

type IngestIpfix struct {
	HostName   string            `yaml:"hostName,omitempty" json:"hostName,omitempty" doc:"the hostname to listen on; defaults to 0.0.0.0"`
	Port       uint              `yaml:"port,omitempty" json:"port,omitempty" doc:"the port number to listen on, for IPFIX/NetFlow v9. Omit or set to 0 to disable IPFIX/NetFlow v9 ingestion. If both port and portLegacy are omitted, defaults to 2055"`
	PortLegacy uint              `yaml:"portLegacy,omitempty" json:"portLegacy,omitempty" doc:"the port number to listen on, for legacy NetFlow v5. Omit or set to 0 to disable NetFlow v5 ingestion"`
	Workers    uint              `yaml:"workers,omitempty" json:"workers,omitempty" doc:"the number of netflow/ipfix decoding workers"`
	Sockets    uint              `yaml:"sockets,omitempty" json:"sockets,omitempty" doc:"the number of listening sockets"`
	Mapping    []NetFlowMapField `yaml:"mapping,omitempty" json:"mapping,omitempty" doc:"custom field mapping"`
}

// NetFlowMapField copied from github.com/netsampler/goflow2/producer (we don't want this hard dependency in api)
type NetFlowMapField struct {
	PenProvided bool   `json:"penprovided" yaml:"penprovided"`
	Type        uint16 `json:"field" yaml:"field"`
	Pen         uint32 `json:"pen" yaml:"pen"`
	Destination string `json:"destination" yaml:"destination"`
	Endian      string `json:"endianness" yaml:"endianness"`
}

func (i *IngestIpfix) SetDefaults() {
	if i.HostName == "" {
		i.HostName = "0.0.0.0"
	}
	if i.Port == 0 && i.PortLegacy == 0 {
		i.Port = 2055
	}
	if i.Workers == 0 {
		i.Workers = 1
	}
	if i.Sockets == 0 {
		i.Sockets = 1
	}
}

func (i *IngestIpfix) String() string {
	hasMapping := "no"
	if len(i.Mapping) > 0 {
		hasMapping = "yes"
	}
	return fmt.Sprintf(
		"hostname=%s, port=%d, portLegacy=%d, workers=%d, sockets=%d, mapping=%s",
		i.HostName,
		i.Port,
		i.PortLegacy,
		i.Workers,
		i.Sockets,
		hasMapping,
	)
}
