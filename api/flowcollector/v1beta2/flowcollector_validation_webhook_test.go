package v1beta2

import (
	"context"
	"errors"
	"testing"

	"github.com/coreos/go-semver/semver"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type clusterInfoMock struct {
	cni     NetworkType
	version string
}

func (m *clusterInfoMock) IsOpenShift() bool {
	return m.version != ""
}

func (m *clusterInfoMock) IsOpenShiftVersionAtLeast(v string) (bool, string, error) {
	if m.version == "" {
		return false, "", errors.New("unknown OpenShift version, cannot compare versions")
	}
	openshiftVersion, err := semver.NewVersion(m.version)
	if err != nil {
		return false, "", err
	}
	version, err := semver.NewVersion(v)
	if err != nil {
		return false, "", err
	}
	// Ignore pre-release block for comparison
	openshiftVersion.PreRelease = ""
	return !openshiftVersion.LessThan(*version), m.version, nil
}

func (m *clusterInfoMock) GetNbNodes() (uint16, error) {
	return 1, nil
}

func (m *clusterInfoMock) GetCNI() (NetworkType, error) {
	return m.cni, nil
}

func TestValidateAgent(t *testing.T) {
	tests := []struct {
		name             string
		fc               *FlowCollector
		ocpVersion       string
		expectedError    string
		expectedWarnings admission.Warnings
	}{
		{
			name: "Empty config is valid",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{},
			},
		},
		{
			name: "Valid configuration",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{DNSTracking, FlowRTT},
							Privileged: false,
							Sampling:   ptr.To(int32(100)),
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Action:    "Accept",
										CIDR:      "0.0.0.0/0",
										Direction: "Egress",
										Protocol:  "TCP",
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Invalid filter with duplicate CIDR",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features: []AgentFeature{DNSTracking, FlowRTT},
							Sampling: ptr.To(int32(100)),
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Action:    "Accept",
										CIDR:      "0.0.0.0/0",
										PeerCIDR:  "1.1.1.1/24",
										Direction: "Egress",
										Protocol:  "TCP",
									},
									{
										Action:    "Accept",
										CIDR:      "0.0.0.0/0",
										PeerCIDR:  "1.1.1.1/24",
										Direction: "Egress",
										Protocol:  "UDP",
									},
								},
							},
						},
					},
				},
			},
			expectedError: "flow filter rule CIDR and PeerCIDR 0.0.0.0/0-1.1.1.1/24 already exists",
		},
		{
			name: "PacketDrop Can't detect environment",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{DNSTracking, FlowRTT, PacketDrop},
							Privileged: true,
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"Unknown environment, cannot detect if the feature PacketDrop is supported"},
		},
		{
			name:       "PacketDrop on ocp 4.12 triggers warning",
			ocpVersion: "4.12.5",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{PacketDrop},
							Privileged: true,
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"The PacketDrop feature requires OpenShift 4.14.0 or above (version detected: 4.12.5)"},
		},
		{
			name:       "PacketDrop is valid",
			ocpVersion: "4.16.0",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{PacketDrop},
							Privileged: true,
						},
					},
				},
			},
		},
		{
			name:       "PacketDrop without privilege triggers warning",
			ocpVersion: "4.16.0",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features: []AgentFeature{PacketDrop},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"The PacketDrop feature requires eBPF Agent to run in privileged mode, which is currently disabled in spec.agent.ebpf.privileged, or to use with eBPF Manager"},
		},
		{
			name:       "NetworkEvents on ocp 4.16 triggers warning",
			ocpVersion: "4.16.5",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{NetworkEvents},
							Privileged: true,
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"The NetworkEvents feature requires OpenShift 4.19.0 or above (version detected: 4.16.5)"},
		},
		{
			name:       "NetworkEvents on ocp 4.19.0-0 doesn't trigger warnings",
			ocpVersion: "4.19.0-0.nightly-2025-03-20-063534",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{NetworkEvents},
							Privileged: true,
						},
					},
				},
			},
		},
		{
			name:       "NetworkEvents without privilege triggers warning",
			ocpVersion: "4.19.0",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features: []AgentFeature{NetworkEvents},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"The NetworkEvents feature requires eBPF Agent to run in privileged mode, which is currently disabled in spec.agent.ebpf.privileged"},
		},
		{
			name:       "UDNMapping on ocp 4.18.0-0 doesn't trigger warnings",
			ocpVersion: "4.18.0-0.nightly-2025-06-30-082606",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							Features:   []AgentFeature{UDNMapping},
							Privileged: true,
						},
					},
				},
			},
		},
		{
			name: "FlowFilter different ports configs are mutually exclusive",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Action:      "Accept",
										CIDR:        "0.0.0.0/0",
										Ports:       intstr.FromInt(80),
										SourcePorts: intstr.FromInt(443),
									},
								},
							},
						},
					},
				},
			},
			expectedError: "cannot configure agent filter with ports and sourcePorts, they are mutually exclusive",
		},
		{
			name: "FlowFilter expect invalid ports",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("abcd"),
									},
								},
							},
						},
					},
				},
			},
			expectedError: "invalid port number",
		},
		{
			name: "FlowFilter expect valid ports range",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("80-255"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "FlowFilter expect invalid ports range (order)",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("255-80"),
									},
								},
							},
						},
					},
				},
			},
			expectedError: "start is greater or equal",
		},
		{
			name: "FlowFilter expect invalid ports range",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("80-?"),
									},
								},
							},
						},
					},
				},
			},
			expectedError: "invalid port number",
		},
		{
			name: "FlowFilter expect valid ports couple",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("255,80"),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "FlowFilter expect invalid ports couple",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										Ports: intstr.FromString("80,100,250"),
									},
								},
							},
						},
					},
				},
			},
			expectedError: "expected two integers",
		},
		{
			name: "FlowFilter expect invalid CIDR",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{
						Type: AgentEBPF,
						EBPF: FlowCollectorEBPF{
							FlowFilter: &EBPFFlowFilter{
								Enable: ptr.To(true),
								Rules: []EBPFFlowFilterRule{
									{
										CIDR: "1.1.1.1",
									},
								},
							},
						},
					},
				},
			},
			expectedError: "invalid CIDR",
		},
	}

	for _, test := range tests {
		CurrentClusterInfo = &clusterInfoMock{version: test.ocpVersion}
		v := validator{fc: &test.fc.Spec}
		v.validateAgent()
		if test.expectedError == "" {
			assert.Empty(t, v.errors, test.name)
		} else {
			assert.Len(t, v.errors, 1, test.name)
			assert.ErrorContains(t, v.errors[0], test.expectedError, test.name)
		}
		assert.Equal(t, test.expectedWarnings, v.warnings, test.name)
	}
}

func TestValidateConntrack(t *testing.T) {
	tests := []struct {
		name             string
		fc               *FlowCollector
		expectedError    string
		expectedWarnings admission.Warnings
	}{
		{
			name: "Conntrack with Loki is valid",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						LogTypes: ptr.To(LogTypeConversations),
					},
					Loki: FlowCollectorLoki{
						Enable: ptr.To(true),
					},
				},
			},
		},
		{
			name: "Conntrack ALL is not recommended",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						LogTypes: ptr.To(LogTypeAll),
					},
					Loki: FlowCollectorLoki{
						Enable: ptr.To(true),
					},
				},
			},
			expectedWarnings: admission.Warnings{"Enabling all log types (in spec.processor.logTypes) has a high impact on resources footprint"},
		},
		{
			name: "Conntrack without Loki is not recommended",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						LogTypes: ptr.To(LogTypeConversations),
					},
					Loki: FlowCollectorLoki{
						Enable: ptr.To(false),
					},
				},
			},
			expectedError: "enabling conversation tracking without Loki is not allowed, as it generates extra processing for no benefit",
		},
		{
			name: "Conntrack not allowed with deploymentModel Service",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelService,
					Processor: FlowCollectorFLP{
						LogTypes: ptr.To(LogTypeConversations),
					},
				},
			},
			expectedError: "cannot enable conversation tracking when spec.deploymentModel is Service: you must disable it, or change the deployment model",
		},
	}

	r := FlowCollector{}
	CurrentClusterInfo = &clusterInfoMock{}
	for _, test := range tests {
		warnings, err := r.Validate(context.TODO(), test.fc)
		if test.expectedError == "" {
			assert.NoError(t, err, test.name)
		} else {
			assert.ErrorContains(t, err, test.expectedError, test.name)
		}
		assert.Equal(t, test.expectedWarnings, warnings, test.name)
	}
}

func TestValidateFLP(t *testing.T) {
	tests := []struct {
		name             string
		fc               *FlowCollector
		expectedError    string
		expectedWarnings admission.Warnings
		ocpVersion       string
	}{
		{
			name: "Valid FLP query",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						Filters: []FLPFilterSet{
							{
								Query: `SrcK8S_Namespace="foo" and without(DstK8S_Namespace)`,
							},
						},
					},
				},
			},
		},
		{
			name: "Invalid FLP query",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						Filters: []FLPFilterSet{
							{
								Query: `SrcK8S_Namespace="foo" and without(DstK8S_Namespace)`,
							},
							{
								Query: `invalid query`,
							},
						},
					},
				},
			},
			expectedError: "cannot parse spec.processor.filters[1].query: syntax error",
		},
		{
			name: "Missing feature for alerts",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Variants: []HealthRuleVariant{},
								},
							},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{"HealthRule PacketDropsByKernel requires the PacketDrop agent feature to be enabled"},
		},
		{
			name:       "No missing metrics for alerts by default",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{PacketDrop},
						Privileged: true,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Variants: []HealthRuleVariant{
										{
											GroupBy: GroupByNode,
											Thresholds: HealthRuleThresholds{
												Info: "5.5",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "Missing metrics for alerts",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{PacketDrop},
						Privileged: true,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Variants: []HealthRuleVariant{
										{
											GroupBy: GroupByNode,
											Thresholds: HealthRuleThresholds{
												Info: "5",
											},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"node_ingress_bytes_total"},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{
				"HealthRule PacketDropsByKernel/Node requires enabling at least one metric from this list: node_drop_packets_total",
				"HealthRule PacketDropsByKernel/Node requires enabling at least one metric from this list: node_ingress_packets_total, node_egress_packets_total",
			},
		},
		{
			name:       "Invalid alert threshold",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{PacketDrop},
						Privileged: true,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Variants: []HealthRuleVariant{
										{
											Thresholds: HealthRuleThresholds{
												Info: "nope",
											},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"node_drop_packets_total", "node_ingress_packets_total"},
						},
					},
				},
			},
			expectedError: `cannot parse info threshold as float in spec.processor.metrics.healthRules[0].variants[0]: "nope"`,
		},
		{
			name:       "Invalid alert threshold severities",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{PacketDrop},
						Privileged: true,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Variants: []HealthRuleVariant{
										{
											Thresholds: HealthRuleThresholds{
												Info:     "5",
												Warning:  "50",
												Critical: "10",
											},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"node_drop_packets_total", "node_ingress_packets_total"},
						},
					},
				},
			},
			expectedError: `warning threshold must be lower than 10, which is defined for a higher severity`,
		},
		{
			name:       "Missing thresholds in recording mode",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{DNSTracking},
						Privileged: false,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRuleDNSErrors,
									Mode:     ModeRecording,
									Variants: []HealthRuleVariant{
										{
											Thresholds: HealthRuleThresholds{},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"namespace_dns_latency_seconds"},
						},
					},
				},
			},
			expectedError: `at least one threshold (critical, warning, or info) must be provided in spec.processor.metrics.healthRules[0].variants[0]`,
		},
		{
			name:       "Valid recording mode with thresholds",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{DNSTracking},
						Privileged: false,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRuleDNSErrors,
									Mode:     ModeRecording,
									Variants: []HealthRuleVariant{
										{
											Thresholds: HealthRuleThresholds{
												Critical: "15",
												Warning:  "10",
												Info:     "5",
											},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"namespace_dns_latency_seconds"},
						},
					},
				},
			},
		},
		{
			name:       "Valid variant mode override - Alert health rule with Recording variant",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{PacketDrop},
						Privileged: true,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRulePacketDropsByKernel,
									Mode:     ModeAlert, // Parent mode is Alert
									Variants: []HealthRuleVariant{
										{
											GroupBy: GroupByNode,
											Mode:    ptr.To(ModeRecording), // Variant overrides to Recording
											Thresholds: HealthRuleThresholds{
												Critical: "10",
												Warning:  "5",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name:       "Valid variant mode override - Recording health rule with Alert variant",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					Agent: FlowCollectorAgent{EBPF: FlowCollectorEBPF{
						Features:   []AgentFeature{DNSTracking},
						Privileged: false,
					}},
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							HealthRules: &[]FLPHealthRule{
								{
									Template: HealthRuleDNSErrors,
									Mode:     ModeRecording, // Parent mode is Recording
									Variants: []HealthRuleVariant{
										{
											Mode: ptr.To(ModeAlert), // Variant overrides to Alert
											Thresholds: HealthRuleThresholds{
												Warning: "10",
												Info:    "5",
											},
										},
									},
								},
							},
							DisableAlerts: []AlertTemplate{HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
							IncludeList:   &[]FLPMetric{"namespace_dns_latency_seconds"},
						},
					},
				},
			},
		},
		{
			name: "Both includeList and additionalIncludeList set",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						Metrics: FLPMetrics{
							IncludeList:           &[]FLPMetric{"node_ingress_bytes_total"},
							AdditionalIncludeList: &[]FLPMetric{"namespace_egress_bytes_total"},
							DisableAlerts:         []AlertTemplate{AlertNoFlows, AlertLokiError, HealthRuleExternalEgressHighTrend, HealthRuleExternalIngressHighTrend},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{
				MetricsIncludeListWarning,
			},
		},
		{
			name:       "Missing provided TLS config",
			ocpVersion: "4.18.0",
			fc: &FlowCollector{
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelService,
					Processor: FlowCollectorFLP{
						Service: &ProcessorServiceConfig{
							TLSType: TLSProvided,
						},
					},
				},
			},
			expectedError: "missing configuration in spec.processor.providedCertificates despite spec.processor.tlsType being set to Provided",
		},
	}

	r := FlowCollector{}
	for _, test := range tests {
		CurrentClusterInfo = &clusterInfoMock{version: test.ocpVersion}
		warnings, err := r.Validate(context.TODO(), test.fc)
		if test.expectedError == "" {
			assert.NoError(t, err, test.name)
		} else {
			assert.ErrorContains(t, err, test.expectedError, test.name)
		}
		assert.Equal(t, test.expectedWarnings, warnings, test.name)
	}
}

func TestValidateScheduling(t *testing.T) {
	mismatchWarning := "Mismatch detected between spec.agent.ebpf.advanced.scheduling and spec.processor.advanced.scheduling. In Direct mode, it can lead to inconsistent pod scheduling that would result in errors in the flow collection process."
	tests := []struct {
		name             string
		fc               *FlowCollector
		expectedError    string
		expectedWarnings admission.Warnings
		ocpVersion       string
	}{
		{
			name: "Valid default (Direct)",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelDirect,
				},
			},
		},
		{
			name: "Invalid Agent scheduling",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelDirect,
					Agent: FlowCollectorAgent{
						EBPF: FlowCollectorEBPF{
							Advanced: &AdvancedAgentConfig{
								Scheduling: &SchedulingConfig{
									Tolerations: []corev1.Toleration{{Key: "key"}},
								},
							},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{mismatchWarning},
		},
		{
			name: "Invalid FLP scheduling",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelDirect,
					Processor: FlowCollectorFLP{
						Advanced: &AdvancedProcessorConfig{
							Scheduling: &SchedulingConfig{
								Tolerations: []corev1.Toleration{{Key: "key"}},
							},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{mismatchWarning},
		},
		{
			name: "Invalid FLP and Agent scheduling",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelDirect,
					Agent: FlowCollectorAgent{
						EBPF: FlowCollectorEBPF{
							Advanced: &AdvancedAgentConfig{
								Scheduling: &SchedulingConfig{
									Tolerations: []corev1.Toleration{{Key: "key1"}},
								},
							},
						},
					},
					Processor: FlowCollectorFLP{
						Advanced: &AdvancedProcessorConfig{
							Scheduling: &SchedulingConfig{
								Tolerations: []corev1.Toleration{{Key: "key2"}},
							},
						},
					},
				},
			},
			expectedWarnings: admission.Warnings{mismatchWarning},
		},
		{
			name: "Valid FLP and Agent scheduling",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelDirect,
					Agent: FlowCollectorAgent{
						EBPF: FlowCollectorEBPF{
							Advanced: &AdvancedAgentConfig{
								Scheduling: &SchedulingConfig{
									Tolerations: []corev1.Toleration{{Key: "same_key"}},
								},
							},
						},
					},
					Processor: FlowCollectorFLP{
						Advanced: &AdvancedProcessorConfig{
							Scheduling: &SchedulingConfig{
								Tolerations: []corev1.Toleration{{Key: "same_key"}},
							},
						},
					},
				},
			},
		},
		{
			name: "Valid default (Kafka)",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					DeploymentModel: DeploymentModelKafka,
				},
			},
		},
		{
			name: "No inconsistent scheduling with Kafka",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					Processor: FlowCollectorFLP{
						Advanced: &AdvancedProcessorConfig{
							Scheduling: &SchedulingConfig{
								Tolerations: []corev1.Toleration{{Key: "key"}},
							},
						},
					},
				},
			},
		},
	}

	r := FlowCollector{}
	for _, test := range tests {
		CurrentClusterInfo = &clusterInfoMock{version: test.ocpVersion}
		warnings, err := r.Validate(context.TODO(), test.fc)
		if test.expectedError == "" {
			assert.NoError(t, err, test.name)
		} else {
			assert.ErrorContains(t, err, test.expectedError, test.name)
		}
		assert.Equal(t, test.expectedWarnings, warnings, test.name)
	}
}

func TestElligibleMetrics(t *testing.T) {
	met, tot := GetElligibleMetricsForHealthRule(HealthRulePacketDropsByKernel, &HealthRuleVariant{
		GroupBy: GroupByNamespace,
	})
	assert.Equal(t, []string{"namespace_drop_packets_total", "workload_drop_packets_total"}, met)
	assert.Equal(t, []string{"namespace_ingress_packets_total", "workload_ingress_packets_total", "namespace_egress_packets_total", "workload_egress_packets_total"}, tot)

	met, tot = GetElligibleMetricsForHealthRule(HealthRulePacketDropsByKernel, &HealthRuleVariant{
		GroupBy: GroupByWorkload,
	})
	assert.Equal(t, []string{"workload_drop_packets_total"}, met)
	assert.Equal(t, []string{"workload_ingress_packets_total", "workload_egress_packets_total"}, tot)

	met, tot = GetElligibleMetricsForHealthRule(HealthRulePacketDropsByKernel, &HealthRuleVariant{
		GroupBy: GroupByNode,
	})
	assert.Equal(t, []string{"node_drop_packets_total"}, met)
	assert.Equal(t, []string{"node_ingress_packets_total", "node_egress_packets_total"}, tot)

	met, tot = GetElligibleMetricsForHealthRule(HealthRulePacketDropsByKernel, &HealthRuleVariant{})
	assert.Equal(t, []string{"namespace_drop_packets_total", "workload_drop_packets_total", "node_drop_packets_total"}, met)
	assert.Equal(t, []string{"namespace_ingress_packets_total", "workload_ingress_packets_total", "node_ingress_packets_total", "namespace_egress_packets_total", "workload_egress_packets_total", "node_egress_packets_total"}, tot)
}

func TestValidateNetPol(t *testing.T) {
	tests := []struct {
		name             string
		fc               *FlowCollector
		cni              NetworkType
		expectedError    string
		expectedWarnings admission.Warnings
	}{
		{
			name: "Empty config is valid for ovn-k",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{},
			},
			cni: OVNKubernetes,
		},
		{
			name: "Empty config is valid for sdn",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{},
			},
			cni: OpenShiftSDN,
		},
		{
			name: "Empty config is valid for unknown",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{},
			},
			cni: "unknown",
		},
		{
			name: "Enabled netpol is valid for ovn-k",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					NetworkPolicy: NetworkPolicy{Enable: ptr.To(true)},
				},
			},
			cni: OVNKubernetes,
		},
		{
			name: "Enabled netpol triggers warning for sdn",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					NetworkPolicy: NetworkPolicy{Enable: ptr.To(true)},
				},
			},
			cni:              OpenShiftSDN,
			expectedWarnings: admission.Warnings{"OpenShiftSDN detected with unsupported setting: spec.networkPolicy.enable; this setting will be ignored; to remove this warning set spec.networkPolicy.enable to false."},
		},
		{
			name: "Enabled netpol triggers warning for unknown",
			fc: &FlowCollector{
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
				Spec: FlowCollectorSpec{
					NetworkPolicy: NetworkPolicy{Enable: ptr.To(true)},
				},
			},
			cni:              "",
			expectedWarnings: admission.Warnings{"Network policy is enabled via spec.networkPolicy.enable, despite running on an unknown CNI: this configuration has not been tested; to remove this warning set spec.networkPolicy.enable to false."},
		},
	}

	for _, test := range tests {
		CurrentClusterInfo = &clusterInfoMock{
			version: "4.20.0",
			cni:     test.cni,
		}
		v := validator{fc: &test.fc.Spec}
		v.validateNetPol()
		if test.expectedError == "" {
			assert.Empty(t, v.errors, test.name)
		} else {
			assert.Len(t, v.errors, 1, test.name)
			assert.ErrorContains(t, v.errors[0], test.expectedError, test.name)
		}
		assert.Equal(t, test.expectedWarnings, v.warnings, test.name)
	}
}

func TestHealthRuleVariant_GetMode(t *testing.T) {
	tests := []struct {
		name         string
		variant      HealthRuleVariant
		parentMode   HealthRuleMode
		expectedMode HealthRuleMode
	}{
		{
			name: "Variant mode is nil - inherits parent Alert",
			variant: HealthRuleVariant{
				Mode: nil,
			},
			parentMode:   ModeAlert,
			expectedMode: ModeAlert,
		},
		{
			name: "Variant mode is nil - inherits parent Recording",
			variant: HealthRuleVariant{
				Mode: nil,
			},
			parentMode:   ModeRecording,
			expectedMode: ModeRecording,
		},
		{
			name: "Variant overrides to Recording",
			variant: HealthRuleVariant{
				Mode: ptr.To(ModeRecording),
			},
			parentMode:   ModeAlert,
			expectedMode: ModeRecording,
		},
		{
			name: "Variant overrides to Alert",
			variant: HealthRuleVariant{
				Mode: ptr.To(ModeAlert),
			},
			parentMode:   ModeRecording,
			expectedMode: ModeAlert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.variant.GetMode(tt.parentMode)
			assert.Equal(t, tt.expectedMode, result)
		})
	}
}
