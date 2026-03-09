//nolint:revive
package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ascv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
)

// nolint:cyclop
func flowCollectorIsoSpecs() {
	const operatorNamespace = "main-namespace"
	crKey := types.NamespacedName{
		Name: "cluster",
	}

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	// This aims to verify that the CRD is preserved / unchanged upon successive serialization/deserialization
	// A typical pitfall is go's empty values messing up with CRD-defined default values, e.g:
	// If field A (bool) has "omitempty" and defaults to true, and is explicitely set to false,
	// then serialization would remove that field (bc go says false ~= empty), and subsequent kube's deserialization would set it to true.
	// The typical workaround is to use pointers on fields (*bool) so 0-value is nil.
	Context("Isomorphic ser/de", func() {
		zero := int32(0)
		defaultTLS := flowslatest.ClientTLS{
			Enable:             false,
			InsecureSkipVerify: false,
			CACert: flowslatest.CertificateReference{
				Type:     "configmap",
				Name:     "",
				CertFile: "",
				CertKey:  "",
			},
			UserCert: flowslatest.CertificateReference{
				Type:     "configmap",
				Name:     "",
				CertFile: "",
				CertKey:  "",
			},
		}

		specInput := flowslatest.FlowCollectorSpec{
			Namespace:       operatorNamespace,
			DeploymentModel: flowslatest.DeploymentModelDirect,
			Processor: flowslatest.FlowCollectorFLP{
				ImagePullPolicy:            "Always",
				LogLevel:                   "trace",
				Resources:                  v1.ResourceRequirements{Limits: nil, Requests: nil},
				KafkaConsumerReplicas:      ptr.To(int32(3)),
				ConsumerReplicas:           ptr.To(int32(3)),
				KafkaConsumerAutoscaler:    flowslatest.FlowCollectorHPA{Status: "Disabled", MinReplicas: &zero, MaxReplicas: zero, Metrics: []ascv2.MetricSpec{}},
				KafkaConsumerQueueCapacity: int(zero),
				KafkaConsumerBatchSize:     int(zero),
				MultiClusterDeployment:     ptr.To(true),
				ClusterName:                "testCluster",
				AddZone:                    ptr.To(false),
				Advanced: &flowslatest.AdvancedProcessorConfig{
					Port:                           ptr.To(int32(12345)),
					HealthPort:                     ptr.To(int32(12346)),
					ProfilePort:                    ptr.To(int32(12347)),
					ConversationHeartbeatInterval:  &metav1.Duration{Duration: time.Second},
					ConversationEndTimeout:         &metav1.Duration{Duration: time.Second},
					ConversationTerminatingTimeout: &metav1.Duration{Duration: time.Second},
					EnableKubeProbes:               ptr.To(false),
					DropUnusedFields:               ptr.To(false),
				},
				LogTypes: ptr.To(flowslatest.LogTypeAll),
				Metrics: flowslatest.FLPMetrics{
					Server: flowslatest.MetricsServerConfig{
						Port: ptr.To(int32(12347)),
						TLS: flowslatest.ServerTLS{
							Type:     "Disabled",
							Provided: nil,
						},
					},
					DisableAlerts: []flowslatest.HealthRuleTemplate{},
				},
			},
			Agent: flowslatest.FlowCollectorAgent{
				Type: "eBPF",
				IPFIX: flowslatest.FlowCollectorIPFIX{
					Sampling:           2, // 0 is forbidden here
					CacheActiveTimeout: "5s",
					CacheMaxFlows:      100,
					ForceSampleAll:     false,
					ClusterNetworkOperator: flowslatest.ClusterNetworkOperatorConfig{
						Namespace: "test",
					},
					OVNKubernetes: flowslatest.OVNKubernetesConfig{
						Namespace:     "test",
						DaemonSetName: "test",
						ContainerName: "test",
					},
				},
				EBPF: flowslatest.FlowCollectorEBPF{
					Sampling:           &zero,
					CacheActiveTimeout: "15s",
					CacheMaxFlows:      100,
					ImagePullPolicy:    "Always",
					Advanced:           &flowslatest.AdvancedAgentConfig{},
					LogLevel:           "trace",
					Resources:          v1.ResourceRequirements{Limits: nil, Requests: nil},
					Interfaces:         []string{},
					ExcludeInterfaces:  []string{},
					Privileged:         false,
					KafkaBatchSize:     0,
					Features:           nil,
					Metrics: flowslatest.EBPFMetrics{
						Enable: ptr.To(false),
						Server: flowslatest.MetricsServerConfig{
							Port: ptr.To(int32(12347)),
							TLS: flowslatest.ServerTLS{
								Type:     "Disabled",
								Provided: nil,
							},
						},
					},
				},
			},
			ConsolePlugin: flowslatest.FlowCollectorConsolePlugin{
				Enable:          ptr.To(true),
				Replicas:        &zero,
				ImagePullPolicy: "Always",
				Advanced: &flowslatest.AdvancedPluginConfig{
					Register: ptr.To(true),
					Port:     ptr.To(int32(9001)),
				},
				Resources:  v1.ResourceRequirements{Limits: nil, Requests: nil},
				LogLevel:   "trace",
				Autoscaler: flowslatest.FlowCollectorHPA{Status: "Disabled", MinReplicas: &zero, MaxReplicas: zero, Metrics: []ascv2.MetricSpec{}},
				PortNaming: flowslatest.ConsolePluginPortConfig{
					Enable:    ptr.To(false),
					PortNames: map[string]string{},
				},
				QuickFilters: []flowslatest.QuickFilter{},
			},
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeManual,
				Manual: flowslatest.LokiManualParams{
					IngesterURL: "http://loki",
					QuerierURL:  "http://loki",
					StatusURL:   "",
					TenantID:    "test",
					AuthToken:   "Disabled",
					TLS:         defaultTLS,
					StatusTLS:   defaultTLS,
				},
				Microservices: flowslatest.LokiMicroservicesParams{
					IngesterURL: "http://loki-distributor:3100/",
					QuerierURL:  "http://loki-query-frontend:3100/",
					TenantID:    "netobserv",
					TLS:         defaultTLS,
				},
				Monolithic: flowslatest.LokiMonolithParams{
					InstallDemoLoki: ptr.To(false),
					URL:             "http://loki:3100/",
					TenantID:        "netobserv",
					TLS:             defaultTLS,
				},
				LokiStack: flowslatest.LokiStackRef{
					Name:      "loki",
					Namespace: "",
				},
				ReadTimeout:    &metav1.Duration{Duration: time.Second},
				WriteTimeout:   &metav1.Duration{Duration: time.Second},
				WriteBatchWait: &metav1.Duration{Duration: time.Second},
				WriteBatchSize: 100,
				Advanced: &flowslatest.AdvancedLokiConfig{
					WriteMinBackoff: &metav1.Duration{Duration: time.Second},
					WriteMaxBackoff: &metav1.Duration{Duration: 5 * time.Second},
					WriteMaxRetries: ptr.To(int32(2)),
					StaticLabels:    map[string]string{"app": "netobserv-flowcollector"},
				},
			},
			Prometheus: flowslatest.FlowCollectorPrometheus{
				Querier: flowslatest.PrometheusQuerier{
					Enable:  ptr.To(true),
					Mode:    "Auto",
					Timeout: &metav1.Duration{Duration: 30 * time.Second},
					Manual:  flowslatest.PrometheusQuerierManual{URL: "http://prometheus:9090"},
				},
			},
			Kafka: flowslatest.FlowCollectorKafka{
				Address: "http://kafka",
				Topic:   "topic",
				TLS:     defaultTLS,
				SASL: flowslatest.SASLConfig{
					Type: "Disabled",
					ClientIDReference: flowslatest.FileReference{
						Type:      "configmap",
						Name:      "",
						Namespace: "",
						File:      "",
					},
					ClientSecretReference: flowslatest.FileReference{
						Type:      "configmap",
						Name:      "",
						Namespace: "",
						File:      "",
					},
				},
			},
			Exporters: []*flowslatest.FlowCollectorExporter{},
			NetworkPolicy: flowslatest.NetworkPolicy{
				Enable:               ptr.To(true),
				AdditionalNamespaces: []string{},
			},
		}

		It("Should create CR successfully", func() {
			Eventually(func() interface{} {
				return k8sClient.Create(ctx, &flowslatest.FlowCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: crKey.Name,
					},
					Spec: specInput,
				})
			}, timeout, interval).Should(Succeed())
		})

		It("Should not have modified input CR values", func() {
			cr := test.GetCR(ctx, k8sClient, crKey)

			// For easier debugging, we check CR parts one by one
			Expect(cr.Spec.Processor).Should(Equal(specInput.Processor))
			Expect(cr.Spec.Agent).Should(Equal(specInput.Agent))
			Expect(cr.Spec.ConsolePlugin).Should(Equal(specInput.ConsolePlugin))
			Expect(cr.Spec.Loki).Should(Equal(specInput.Loki))
			Expect(cr.Spec.Kafka).Should(Equal(specInput.Kafka))
			Expect(cr.Spec.Exporters).Should(Equal(specInput.Exporters))
			Expect(cr.Spec.NetworkPolicy).Should(Equal(specInput.NetworkPolicy))
			Expect(cr.Spec.Prometheus).Should(Equal(specInput.Prometheus))

			// Catch-all in case we missed something
			Expect(cr.Spec).Should(Equal(specInput))
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)
		})
	})
}
