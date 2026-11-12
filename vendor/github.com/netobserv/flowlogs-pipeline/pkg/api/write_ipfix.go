package api

import (
	"errors"
	"fmt"
	"time"
)

type WriteIpfix struct {
	TargetHost      string   `yaml:"targetHost,omitempty" json:"targetHost,omitempty" doc:"IPFIX Collector host target IP"`
	TargetPort      int      `yaml:"targetPort,omitempty" json:"targetPort,omitempty" doc:"IPFIX Collector host target port"`
	Transport       string   `yaml:"transport,omitempty" json:"transport,omitempty" doc:"Transport protocol (tcp/udp) to be used for the IPFIX connection"`
	EnterpriseID    int      `yaml:"enterpriseId,omitempty" json:"enterpriseId,omitempty" doc:"Enterprise ID for exporting non-standard data"`
	TplSendInterval Duration `yaml:"tplSendInterval,omitempty" json:"tplSendInterval,omitempty" doc:"Interval for resending templates to the collector (default: 1m)"`
	PathMTU         int      `yaml:"pathMtu,omitempty" json:"pathMtu,omitempty" doc:"Path MTU, used to bound the size of the exported IPFIX messages when transport is udp; ignored for tcp (default: 1400)"`
}

// Without a path MTU, go-ipfix caps UDP messages at its 512B minimum and drops any larger record,
// which long Kubernetes names easily exceed. 1400 matches the usual pod MTU on encapsulated networks.
const defaultPathMTU = 1400

// go-ipfix derives its max message size from pathMtu minus the IP+UDP headers (48B at most, for
// IPv6), and won't start if that lands under its 512B floor for the cap.
const minPathMTU = 560

func (w *WriteIpfix) SetDefaults() {
	if w.Transport == "" {
		w.Transport = "tcp"
	}
	if w.TplSendInterval.Duration == 0 {
		w.TplSendInterval.Duration = time.Minute
	}
	if w.PathMTU == 0 {
		w.PathMTU = defaultPathMTU
	}
}

func (w *WriteIpfix) Validate() error {
	if w == nil {
		return errors.New("you must provide a configuration")
	}
	if w.TargetHost == "" {
		return errors.New("targetHost can't be empty")
	}
	if w.TargetPort == 0 {
		return errors.New("targetPort can't be empty")
	}
	if w.Transport != "tcp" && w.Transport != "udp" && w.Transport != "" {
		return errors.New("transport should be tcp/udp")
	}
	if w.Transport == "udp" && w.PathMTU != 0 && w.PathMTU < minPathMTU {
		return fmt.Errorf("pathMtu should be at least %d", minPathMTU)
	}
	return nil
}
