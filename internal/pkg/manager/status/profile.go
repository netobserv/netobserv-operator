package status

import (
	"strconv"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
)

type resolvedSummaryBuilder struct {
	strings.Builder
	actualSpec  *flowslatest.FlowCollectorSpec
	defaultSpec *flowslatest.FlowCollectorSpec
	sep         string
}

func getResolvedProfileSummary(spec *flowslatest.FlowCollectorSpec) string {
	b := resolvedSummaryBuilder{actualSpec: spec, defaultSpec: &flowslatest.FlowCollectorSpec{Profile: spec.Profile}}
	b.WriteString("Current profile: ")
	if spec.Profile != "" {
		b.WriteString(string(spec.Profile))
	} else {
		b.WriteString("(none)")
	}
	b.WriteString(" {")
	b.writeItem(
		"spec.agent.ebpf.sampling",
		spec.Agent.EBPF.Sampling != nil,
		func(fcs *flowslatest.FlowCollectorSpec) string {
			return strconv.Itoa(fcs.GetSampling())
		},
	)
	b.writeItem(
		"spec.agent.ebpf.cacheMaxFlows",
		spec.Agent.EBPF.CacheMaxFlows != nil && *spec.Agent.EBPF.CacheMaxFlows != 0,
		func(fcs *flowslatest.FlowCollectorSpec) string {
			return strconv.Itoa(fcs.GetCacheMaxFlows())
		},
	)
	b.writeItem(
		"spec.agent.ebpf.features",
		spec.Agent.EBPF.Features != nil,
		func(fcs *flowslatest.FlowCollectorSpec) string {
			sb := strings.Builder{}
			feats := fcs.GetAgentFeatures()
			var sep string
			sb.WriteRune('[')
			for _, f := range feats {
				sb.WriteString(sep)
				sb.WriteString(string(f))
				sep = ","
			}
			sb.WriteRune(']')
			return sb.String()
		},
	)
	b.writeItem(
		"spec.loki.enable",
		spec.Loki.Enable != nil,
		func(fcs *flowslatest.FlowCollectorSpec) string {
			return strconv.FormatBool(fcs.UseLoki())
		},
	)
	b.WriteString("}")
	return b.String()
}

func (b *resolvedSummaryBuilder) writeItem(specPath string, isOverride bool, extractor func(*flowslatest.FlowCollectorSpec) string) {
	b.WriteString(b.sep)
	if b.sep == "" {
		b.sep = ", "
	}
	b.WriteString(specPath)
	b.WriteRune('=')
	b.WriteString(extractor(b.actualSpec))
	if isOverride {
		b.WriteString(" (override; default: ")
		b.WriteString(extractor(b.defaultSpec))
		b.WriteString(")")
	}
}
