package flp

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/netobserv/flowlogs-pipeline/pkg/api"
	"github.com/netobserv/flowlogs-pipeline/pkg/config"
	promConfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/flp/fmstatus"
	"github.com/netobserv/netobserv-operator/internal/pkg/conversion"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper/loki"
	otelConfig "github.com/netobserv/netobserv-operator/internal/pkg/helper/otel"
	"github.com/netobserv/netobserv-operator/internal/pkg/metrics"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
)

const (
	ovnkSecondary               = "ovn-kubernetes"
	openshiftNamespacesPrefixes = "openshift"
)

type PipelineBuilder struct {
	*config.PipelineBuilderStage
	desired         *flowslatest.FlowCollectorSpec
	flowMetrics     *metricslatest.FlowMetricList
	fcSlices        []sliceslatest.FlowCollectorSlice
	detectedSubnets []flowslatest.SubnetLabel
	volumes         *volumes.Builder
	loki            *helper.LokiConfig
	clusterID       string
}

func createPipeline(
	desired *flowslatest.FlowCollectorSpec,
	flowMetrics *metricslatest.FlowMetricList,
	fcSlices []sliceslatest.FlowCollectorSlice,
	detectedSubnets []flowslatest.SubnetLabel,
	loki *helper.LokiConfig,
	clusterID string,
	volumes *volumes.Builder,
	ingestStage config.PipelineBuilderStage,
) (*PipelineBuilder, error) {
	b := &PipelineBuilder{
		PipelineBuilderStage: &ingestStage,
		desired:              desired,
		flowMetrics:          flowMetrics,
		fcSlices:             fcSlices,
		detectedSubnets:      detectedSubnets,
		loki:                 loki,
		clusterID:            clusterID,
		volumes:              volumes,
	}
	stage := ingestStage
	stage = b.addConnectionTracking(stage)

	stage = b.addEnrichStage(stage)
	var err error
	stage, err = b.addSubnetLabelsStage(stage)
	if err != nil {
		return nil, err
	}
	stage = b.addTruncFiltersDedupStage(stage)

	if b.desired.UseLoki() {
		if err := b.addLokiStage(stage); err != nil {
			return nil, err
		}
	}

	// write on Stdout if logging trace enabled
	if b.desired.Processor.LogLevel == "trace" {
		stage.WriteStdout("stdout", api.WriteStdout{Format: "json"})
	}

	var flpMetrics []api.MetricsItem
	flpMetrics, err = b.addPrometheusStage(stage)
	if err != nil {
		return nil, err
	}

	if err := b.addCustomExportStages(stage, flpMetrics); err != nil {
		return nil, err
	}

	return b, nil
}

func (b *PipelineBuilder) addEnrichStage(previous config.PipelineBuilderStage) config.PipelineBuilderStage {
	addZone := b.desired.Processor.IsZoneEnabled()
	rules := api.NetworkTransformRules{
		{
			Type: api.NetworkAddKubernetes,
			Kubernetes: &api.K8sRule{
				IPField:         "SrcAddr",
				MACField:        "SrcMac",
				InterfacesField: "Interfaces",
				UDNsField:       "Udns",
				Output:          "SrcK8S",
				AddZone:         addZone,
			},
		},
		{
			Type: api.NetworkAddKubernetes,
			Kubernetes: &api.K8sRule{
				IPField:         "DstAddr",
				MACField:        "DstMac",
				InterfacesField: "Interfaces",
				UDNsField:       "Udns",
				Output:          "DstK8S",
				AddZone:         addZone,
			},
		},
	}
	if b.desired.Agent.EBPF.IsPacketTranslationEnabled() {
		rules = append(rules, api.NetworkTransformRules{
			{
				Type: api.NetworkAddKubernetes,
				Kubernetes: &api.K8sRule{
					IPField: "XlatSrcAddr",
					Output:  "XlatSrcK8S",
				},
			},
			{
				Type: api.NetworkAddKubernetes,
				Kubernetes: &api.K8sRule{
					IPField: "XlatDstAddr",
					Output:  "XlatDstK8S",
				},
			},
		}...)
	}

	rules = append(rules, api.NetworkTransformRules{
		{
			Type: api.NetworkReinterpretDirection,
		},
		{
			Type: api.NetworkDecodeTCPFlags,
			DecodeTCPFlags: &api.NetworkGenericRule{
				Input:  "Flags",
				Output: "Flags",
			},
		},
		{
			Type: api.NetworkAddKubernetesInfra,
			KubernetesInfra: &api.K8sInfraRule{
				NamespaceNameFields: []api.K8sReference{
					{Namespace: "SrcK8S_Namespace", Name: "SrcK8S_Name"},
					{Namespace: "DstK8S_Namespace", Name: "DstK8S_Name"},
				},
				Output:        "K8S_FlowLayer",
				InfraPrefixes: []string{b.desired.Namespace, openshiftNamespacesPrefixes},
				InfraRefs: []api.K8sReference{
					{
						Name:      "kubernetes",
						Namespace: "default",
					},
					{
						Name:      "openshift",
						Namespace: "default",
					},
				},
			},
		},
	}...)

	// Propagate 2dary networks config
	var secondaryNetworks []api.SecondaryNetwork
	for _, sn := range b.desired.GetSecondaryIndexes() {
		flpSN := api.SecondaryNetwork{Index: map[string]any{}}
		for _, index := range sn.Index {
			flpSN.Index[strings.ToLower(string(index))] = nil
		}
		secondaryNetworks = append(secondaryNetworks, flpSN)
	}
	if b.desired.Agent.EBPF.IsUDNMappingEnabled() {
		secondaryNetworks = append(secondaryNetworks, api.SecondaryNetwork{
			Name:  ovnkSecondary,
			Index: map[string]any{"udn": nil},
		})
	}

	// enrich stage (transform) configuration
	return previous.TransformNetwork("enrich", api.TransformNetwork{
		Rules: rules,
		DirectionInfo: api.NetworkTransformDirectionInfo{
			ReporterIPField:    "AgentIP",
			SrcHostField:       "SrcK8S_HostIP",
			DstHostField:       "DstK8S_HostIP",
			FlowDirectionField: "FlowDirection",
		},
		KubeConfig: api.NetworkTransformKubeConfig{
			SecondaryNetworks: secondaryNetworks,
			TrackedKinds:      []string{"ReplicaSet", "Deployment", "Gateway"},
		},
	})
}

func (b *PipelineBuilder) addTruncFiltersDedupStage(previous config.PipelineBuilderStage) config.PipelineBuilderStage {
	// Custom filters
	stage := previous
	filters := filtersToFLP(b.desired.Processor.Filters, flowslatest.FLPFilterTargetAll)
	sliceFilters := slicesToFilters(b.desired, b.fcSlices)
	if len(sliceFilters) > 0 {
		filters = append(filters, sliceFilters...)
	}
	filters = b.addOtherTransforms(filters)
	if len(filters) > 0 {
		stage = stage.TransformFilter("filters", api.TransformFilter{Rules: filters, SamplingField: "Sampling"}, config.Dynamic)
	}

	// Dedup stage
	if b.desired.Processor.HasFLPDeduper() {
		dedupRules := []*api.RemoveEntryRule{
			{
				Type: api.RemoveEntryIfEqualD,
				RemoveEntry: &api.TransformFilterGenericRule{
					Input:   "FlowDirection",
					Value:   1,
					CastInt: true,
				},
			},
			{
				Type: api.RemoveEntryIfExistsD,
				RemoveEntry: &api.TransformFilterGenericRule{
					Input: "DstK8S_OwnerName",
				},
			},
		}
		var transformFilter api.TransformFilter
		if b.desired.Processor.Deduper.Mode == flowslatest.FLPDeduperDrop {
			transformFilter = api.TransformFilter{
				Rules: []api.TransformFilterRule{
					{
						Type:                    api.RemoveEntryAllSatisfied,
						RemoveEntryAllSatisfied: dedupRules,
					},
				},
			}
		} else {
			transformFilter = api.TransformFilter{
				Rules: []api.TransformFilterRule{
					{
						Type: api.ConditionalSampling,
						ConditionalSampling: []*api.SamplingCondition{
							{
								Rules: dedupRules,
								Value: uint16(b.desired.Processor.Deduper.Sampling),
							},
						},
					},
				},
			}
		}
		stage = stage.TransformFilter("dedup", transformFilter)
	}
	return stage
}

func filtersToFLP(in []flowslatest.FLPFilterSet, target flowslatest.FLPFilterTarget) []api.TransformFilterRule {
	var rules []api.TransformFilterRule
	for _, f := range in {
		if f.OutputTarget == target {
			rules = append(rules, api.TransformFilterRule{
				Type:              api.KeepEntryQuery,
				KeepEntryQuery:    f.Query,
				KeepEntrySampling: uint16(f.Sampling),
			})
		}
	}
	return rules
}

func flowMetricToFLP(fm *metricslatest.FlowMetric) (*api.MetricsItem, error) {
	metricName := fm.Spec.MetricName
	if metricName == "" {
		metricName = helper.PrometheusMetricName(fm.Name)
	}
	var remap map[string]string
	if len(fm.Spec.Remap) > 0 {
		remap = make(map[string]string, len(fm.Spec.Remap))
		for k, v := range fm.Spec.Remap {
			remap[k] = string(v)
		}
	}
	m := &api.MetricsItem{
		Name:     metricName,
		Type:     api.MetricEncodeOperationEnum(strings.ToLower(string(fm.Spec.Type))),
		Help:     fm.Spec.Help,
		Filters:  []api.MetricsFilter{},
		Labels:   fm.Spec.Labels,
		Remap:    remap,
		Flatten:  fm.Spec.Flatten,
		ValueKey: fm.Spec.ValueField,
	}
	for _, f := range metrics.GetFilters(&fm.Spec) {
		m.Filters = append(m.Filters, api.MetricsFilter{Key: f.Field, Value: f.Value, Type: api.MetricFilterEnum(conversion.PascalToLower(string(f.MatchType), '_'))})
	}
	for _, b := range fm.Spec.Buckets {
		f, err := strconv.ParseFloat(b, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse metric buckets as floats: '%s'; error was: %w", b, err)
		}
		m.Buckets = append(m.Buckets, f)
	}
	if fm.Spec.Divider != "" {
		f, err := strconv.ParseFloat(fm.Spec.Divider, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse metric divider as float: '%s'; error was: %w", fm.Spec.Divider, err)
		}
		m.ValueScale = f
	}
	return m, nil
}

func (b *PipelineBuilder) addConnectionTracking(lastStage config.PipelineBuilderStage) config.PipelineBuilderStage {
	outputFields := []api.OutputField{
		{
			Name:      "Bytes",
			Operation: "sum",
		},
		{
			Name:      "Bytes",
			Operation: "sum",
			SplitAB:   true,
		},
		{
			Name:      "Packets",
			Operation: "sum",
		},
		{
			Name:      "Packets",
			Operation: "sum",
			SplitAB:   true,
		},
		{
			Name:      "numFlowLogs",
			Operation: "count",
		},
		{
			Name:          "TimeFlowStartMs",
			Operation:     "min",
			ReportMissing: true,
		},
		{
			Name:          "TimeFlowEndMs",
			Operation:     "max",
			ReportMissing: true,
		},
		{
			Name:          "FlowDirection",
			Operation:     "first",
			ReportMissing: true,
		},
		{
			Name:          "IfDirection",
			Operation:     "first",
			ReportMissing: true,
		},
		{
			Name:          "AgentIP",
			Operation:     "first",
			ReportMissing: true,
		},
	}

	if b.desired.Agent.EBPF.IsPktDropEnabled() {
		outputPktDropFields := []api.OutputField{
			{
				Name:      "PktDropBytes",
				Operation: "sum",
			},
			{
				Name:      "PktDropBytes",
				Operation: "sum",
				SplitAB:   true,
			},
			{
				Name:      "PktDropPackets",
				Operation: "sum",
			},
			{
				Name:      "PktDropPackets",
				Operation: "sum",
				SplitAB:   true,
			},
			{
				Name:      "PktDropLatestState",
				Operation: "last",
			},
			{
				Name:      "PktDropLatestDropCause",
				Operation: "last",
			},
		}
		outputFields = append(outputFields, outputPktDropFields...)
	}

	if b.desired.Agent.EBPF.IsDNSTrackingEnabled() {
		outDNSTrackingFields := []api.OutputField{
			{
				Name:      "DnsFlagsResponseCode",
				Operation: "last",
			},
			{
				Name:      "DnsLatencyMs",
				Operation: "max",
			},
		}
		outputFields = append(outputFields, outDNSTrackingFields...)
	}

	if b.desired.Agent.EBPF.IsNetworkEventsEnabled() {
		outNetworkEventsFlowFields := []api.OutputField{
			{
				Name:      "NetworkEvents",
				Operation: "last",
			},
		}
		outputFields = append(outputFields, outNetworkEventsFlowFields...)
	}

	if b.desired.Agent.EBPF.IsFlowRTTEnabled() {
		outputFields = append(outputFields, api.OutputField{
			Name:      "MaxTimeFlowRttNs",
			Operation: "max",
			Input:     "TimeFlowRttNs",
		})
	}

	// Connection tracking stage (only if LogTypes is not FLOWS)
	if b.desired.Processor.HasConntrack() {
		outputRecordTypes := helper.GetRecordTypes(&b.desired.Processor)
		advancedConfig := helper.GetAdvancedProcessorConfig(b.desired)
		lastStage = lastStage.ConnTrack("extract_conntrack", api.ConnTrack{
			KeyDefinition: api.KeyDefinition{
				FieldGroups: []api.FieldGroup{
					{Name: "src", Fields: []string{"SrcAddr", "SrcPort"}},
					{Name: "dst", Fields: []string{"DstAddr", "DstPort"}},
					{Name: "common", Fields: []string{"Proto"}},
				},
				Hash: api.ConnTrackHash{
					FieldGroupRefs: []string{
						"common",
					},
					FieldGroupARef: "src",
					FieldGroupBRef: "dst",
				},
			},
			OutputRecordTypes: outputRecordTypes,
			OutputFields:      outputFields,
			Scheduling: []api.ConnTrackSchedulingGroup{
				{
					Selector:             nil, // Default group. Match all flowlogs
					HeartbeatInterval:    api.Duration{Duration: advancedConfig.ConversationHeartbeatInterval.Duration},
					EndConnectionTimeout: api.Duration{Duration: advancedConfig.ConversationEndTimeout.Duration},
					TerminatingTimeout:   api.Duration{Duration: advancedConfig.ConversationTerminatingTimeout.Duration},
				},
			},
			TCPFlags: api.ConnTrackTCPFlags{
				FieldName:           "Flags",
				DetectEndConnection: true,
				SwapAB:              true,
			},
		})
	}
	return lastStage
}

func (b *PipelineBuilder) addOtherTransforms(rules []api.TransformFilterRule) []api.TransformFilterRule {
	if b.desired.Processor.IsMultiClusterEnabled() {
		var clusterName string
		if b.desired.Processor.ClusterName != "" {
			clusterName = b.desired.Processor.ClusterName
		} else {
			// Take clustername from openshift
			clusterName = b.clusterID
		}
		if clusterName != "" {
			rules = append(rules, api.TransformFilterRule{
				Type: api.AddFieldIfDoesntExist,
				AddFieldIfDoesntExist: &api.TransformFilterGenericRule{
					Input: constants.ClusterNameLabelName,
					Value: clusterName,
				},
			})
		}
	}
	return rules
}

// Add transform stage for subnet labels as a dynamic stage
func (b *PipelineBuilder) addSubnetLabelsStage(previous config.PipelineBuilderStage) (config.PipelineBuilderStage, error) {
	// Get all subnet labels
	// Highest priority: admin-defined labels
	allLabels := b.desired.Processor.SubnetLabels.CustomLabels
	var cidrs []*net.IPNet
	for _, label := range allLabels {
		for _, strCIDR := range label.CIDRs {
			_, cidr, err := net.ParseCIDR(strCIDR)
			if err != nil {
				return previous, fmt.Errorf("wrong CIDR for subnet label '%s': %w", label.Name, err)
			}
			cidrs = append(cidrs, cidr)
		}
	}
	// Then: slice labels
	if b.desired.IsSliceEnabled() {
		allLabels = append(allLabels, slicesToFCSubnetLabels(b.fcSlices, cidrs)...)
	}
	// Finally: detected/fallback labels
	allLabels = append(allLabels, b.detectedSubnets...)
	flpLabels := subnetLabelsToFLP(allLabels)

	if len(flpLabels) > 0 {
		rules := api.NetworkTransformRules{
			{
				Type: api.NetworkAddSubnetLabel,
				AddSubnetLabel: &api.NetworkAddSubnetLabelRule{
					Input:  "SrcAddr",
					Output: "SrcSubnetLabel",
				},
			},
			{
				Type: api.NetworkAddSubnetLabel,
				AddSubnetLabel: &api.NetworkAddSubnetLabelRule{
					Input:  "DstAddr",
					Output: "DstSubnetLabel",
				},
			},
		}
		return previous.TransformNetwork("subnets", api.TransformNetwork{
			Rules:        rules,
			SubnetLabels: flpLabels,
		}, config.Dynamic), nil
	}

	return previous, nil
}

func (b *PipelineBuilder) addLokiStage(previous config.PipelineBuilderStage) error {
	advancedConfig := helper.GetAdvancedLokiConfig(b.desired.Loki.Advanced)
	lokiLabels, err := loki.GetLabels(b.desired)
	if err != nil {
		return err
	}
	lokiStage := previous
	// Custom filters: Loki only
	filters := filtersToFLP(b.desired.Processor.Filters, flowslatest.FLPFilterTargetLoki)
	if len(filters) > 0 {
		lokiStage = lokiStage.TransformFilter("filters-loki", api.TransformFilter{Rules: filters, SamplingField: "Sampling"})
	}

	lokiWrite := api.WriteLoki{
		Labels:         lokiLabels,
		BatchSize:      int(b.desired.Loki.WriteBatchSize),
		BatchWait:      helper.UnstructuredDuration(b.desired.Loki.WriteBatchWait),
		MaxBackoff:     helper.UnstructuredDuration(advancedConfig.WriteMaxBackoff),
		MaxRetries:     int(helper.PtrInt32(advancedConfig.WriteMaxRetries)),
		MinBackoff:     helper.UnstructuredDuration(advancedConfig.WriteMinBackoff),
		StaticLabels:   model.LabelSet{},
		Timeout:        helper.UnstructuredDuration(b.desired.Loki.WriteTimeout),
		URL:            b.loki.IngesterURL,
		TimestampLabel: "TimeFlowEndMs",
		TimestampScale: "1ms",
		TenantID:       b.loki.TenantID,
	}

	for k, v := range advancedConfig.StaticLabels {
		lokiWrite.StaticLabels[model.LabelName(k)] = model.LabelValue(v)
	}

	var authorization *promConfig.Authorization
	if b.loki.UseHostToken() || b.loki.UseForwardToken() {
		b.volumes.AddToken(constants.FLPName)
		authorization = &promConfig.Authorization{
			Type:            "Bearer",
			CredentialsFile: constants.TokensPath + constants.FLPName,
		}
	}

	if b.loki.TLS.Enable {
		if b.loki.TLS.InsecureSkipVerify {
			lokiWrite.ClientConfig = &promConfig.HTTPClientConfig{
				Authorization: authorization,
				TLSConfig: promConfig.TLSConfig{
					InsecureSkipVerify: true,
				},
			}
		} else {
			caPath := b.volumes.AddCACertificate(&b.loki.TLS, "loki-certs")
			lokiWrite.ClientConfig = &promConfig.HTTPClientConfig{
				Authorization: authorization,
				TLSConfig: promConfig.TLSConfig{
					CAFile: caPath,
				},
			}
		}
	} else {
		lokiWrite.ClientConfig = &promConfig.HTTPClientConfig{
			Authorization: authorization,
		}
	}
	lokiStage.WriteLoki("loki", lokiWrite)
	return nil
}

func (b *PipelineBuilder) addPrometheusStage(previous config.PipelineBuilderStage) ([]api.MetricsItem, error) {
	// Configure metrics
	var flpMetrics []api.MetricsItem

	// First, add predefined metrics
	predefined := metrics.GetDefinitions(b.desired, false)
	for i := range predefined {
		fm := predefined[i]
		m, err := flowMetricToFLP(&fm)
		if err != nil {
			// Predefined metric failure => bug
			return nil, fmt.Errorf("error reading predefined FlowMetric '%s': %w", fm.Name, err)
		}
		flpMetrics = append(flpMetrics, *m)
	}

	// Then add user-defined FlowMetrics
	for i := range b.flowMetrics.Items {
		fm := &b.flowMetrics.Items[i]
		m, err := flowMetricToFLP(fm)
		if err != nil {
			fmstatus.SetFailure(fm, err.Error())
			continue
		}
		// Update with actual name
		fm.Status.PrometheusName = "netobserv_" + m.Name
		fmstatus.CheckCardinality(fm)
		flpMetrics = append(flpMetrics, *m)
	}

	if len(flpMetrics) > 0 {
		promStage := previous
		// Custom filters: Metrics only
		filters := filtersToFLP(b.desired.Processor.Filters, flowslatest.FLPFilterTargetMetrics)
		if len(filters) > 0 {
			promStage = promStage.TransformFilter("filters-prom", api.TransformFilter{Rules: filters, SamplingField: "Sampling"})
		}
		promStage.EncodePrometheus("prometheus", api.PromEncode{Prefix: "netobserv_", Metrics: flpMetrics}, config.Dynamic)
	}
	return flpMetrics, nil
}

func (b *PipelineBuilder) addCustomExportStages(previous config.PipelineBuilderStage, flpMetrics []api.MetricsItem) error {
	// Custom filters: Exporters only
	stage := previous
	filters := filtersToFLP(b.desired.Processor.Filters, flowslatest.FLPFilterTargetExporters)
	if len(filters) > 0 {
		stage = stage.TransformFilter("filters-exp", api.TransformFilter{Rules: filters, SamplingField: "Sampling"})
	}

	for i, exporter := range b.desired.Exporters {
		if exporter.Type == flowslatest.KafkaExporter {
			b.createKafkaWriteStage(fmt.Sprintf("kafka-export-%d", i), &exporter.Kafka, &stage)
		}
		if exporter.Type == flowslatest.IpfixExporter {
			createIPFIXWriteStage(fmt.Sprintf("ipfix-export-%d", i), &exporter.IPFIX, &stage)
		}
		if exporter.Type == flowslatest.OpenTelemetryExporter {
			err := b.createOpenTelemetryStage(fmt.Sprintf("otel-export-%d", i), &exporter.OpenTelemetry, &stage, flpMetrics)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *PipelineBuilder) createKafkaWriteStage(name string, spec *flowslatest.FlowCollectorKafka, fromStage *config.PipelineBuilderStage) config.PipelineBuilderStage {
	return fromStage.EncodeKafka(name, api.EncodeKafka{
		Address:     spec.Address,
		Topic:       spec.Topic,
		Compression: spec.Compression,
		TLS:         getClientTLS(&spec.TLS, name, b.volumes),
		SASL:        getSASL(&spec.SASL, name, b.volumes),
	})
}

func (b *PipelineBuilder) AddKafkaWriteStage(name string, spec *flowslatest.FlowCollectorKafka) config.PipelineBuilderStage {
	return b.createKafkaWriteStage(name, spec, b.PipelineBuilderStage)
}

func createIPFIXWriteStage(name string, spec *flowslatest.FlowCollectorIPFIXReceiver, fromStage *config.PipelineBuilderStage) config.PipelineBuilderStage {
	return fromStage.WriteIpfix(name, api.WriteIpfix{
		TargetHost:   spec.TargetHost,
		TargetPort:   spec.TargetPort,
		Transport:    getIPFIXTransport(spec.Transport),
		EnterpriseID: spec.EnterpriseID,
	})
}

func getIPFIXTransport(transport string) string {
	switch transport {
	case "UDP":
		return "udp"
	default:
		return "tcp" // Always fallback on tcp
	}
}

func (b *PipelineBuilder) createOpenTelemetryStage(name string, spec *flowslatest.FlowCollectorOpenTelemetry, fromStage *config.PipelineBuilderStage, flpMetrics []api.MetricsItem) error {
	conn := api.OtlpConnectionInfo{
		Address:        spec.TargetHost,
		Port:           spec.TargetPort,
		ConnectionType: getOtelConnType(spec.Protocol),
		TLS:            getClientTLS(&spec.TLS, name, b.volumes),
		Headers:        spec.Headers,
	}

	logsEnabled := spec.Logs.Enable != nil && *spec.Logs.Enable
	metricsEnabled := spec.Metrics.Enable != nil && *spec.Metrics.Enable

	if logsEnabled || metricsEnabled {
		// add transform stage
		transformCfg, err := otelConfig.GetOtelTransformConfig(spec.FieldsMapping)
		if err != nil {
			return err
		}
		transformStage := fromStage.TransformGeneric(fmt.Sprintf("%s-transform", name), *transformCfg)

		// otel logs config
		if logsEnabled {
			// add encode stage(s)
			transformStage.EncodeOtelLogs(fmt.Sprintf("%s-logs", name), api.EncodeOtlpLogs{
				OtlpConnectionInfo: &conn,
			})
		}

		// otel metrics config
		if metricsEnabled {
			metricsCfg, err := otelConfig.GetOtelMetrics(flpMetrics)
			if err != nil {
				return err
			}
			transformStage.EncodeOtelMetrics(fmt.Sprintf("%s-metrics", name), api.EncodeOtlpMetrics{
				OtlpConnectionInfo: &conn,
				Prefix:             "netobserv_",
				Metrics:            metricsCfg,
				PushTimeInterval:   api.Duration{Duration: spec.Metrics.PushTimeInterval.Duration},
				ExpiryTime:         api.Duration{Duration: 2 * time.Minute},
			})
		}
	}
	return nil
}

func getOtelConnType(connType string) string {
	switch connType {
	case "http":
		return "http"
	default:
		return "grpc"
	}
}

func getClientTLS(tls *flowslatest.ClientTLS, volumeName string, volumes *volumes.Builder) *api.ClientTLS {
	if tls.Enable {
		caPath, userCertPath, userKeyPath := volumes.AddMutualTLSCertificates(tls, volumeName)
		return &api.ClientTLS{
			InsecureSkipVerify: tls.InsecureSkipVerify,
			CACertPath:         caPath,
			UserCertPath:       userCertPath,
			UserKeyPath:        userKeyPath,
		}
	}
	return nil
}

func getSASL(sasl *flowslatest.SASLConfig, volumePrefix string, volumes *volumes.Builder) *api.SASLConfig {
	if !sasl.UseSASL() {
		return nil
	}
	t := api.SASLPlain
	if sasl.Type == flowslatest.SASLScramSHA512 {
		t = api.SASLScramSHA512
	}
	idPath := volumes.AddVolume(&sasl.ClientIDReference, volumePrefix+"-sasl-id")
	secretPath := volumes.AddVolume(&sasl.ClientSecretReference, volumePrefix+"-sasl-secret")
	return &api.SASLConfig{
		Type:             t,
		ClientIDPath:     idPath,
		ClientSecretPath: secretPath,
	}
}

func subnetLabelsToFLP(labels []flowslatest.SubnetLabel) []api.NetworkTransformSubnetLabel {
	var cats []api.NetworkTransformSubnetLabel
	for _, subnetLabel := range labels {
		cats = append(cats, api.NetworkTransformSubnetLabel{
			Name:  subnetLabel.Name,
			CIDRs: subnetLabel.CIDRs,
		})
	}
	return cats
}
