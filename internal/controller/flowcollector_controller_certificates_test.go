//nolint:revive
package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	. "github.com/netobserv/netobserv-operator/internal/controller/controllerstest"
	"github.com/netobserv/netobserv-operator/internal/pkg/test"
	"github.com/netobserv/netobserv-operator/internal/pkg/watchers"
)

var (
	cmw                  watchers.ConfigWatchable
	sw                   watchers.SecretWatchable
	consistentlyTimeout  = 2 * time.Second
	consistentlyInterval = 500 * time.Millisecond
)

// nolint:cyclop
func flowCollectorCertificatesSpecs() {
	const operatorNamespace = "main-namespace"
	crKey := types.NamespacedName{
		Name: "cluster",
	}
	flpKey := types.NamespacedName{
		Name:      "flowlogs-pipeline-transformer",
		Namespace: operatorNamespace,
	}
	pluginKey := types.NamespacedName{
		Name:      constants.PluginName,
		Namespace: operatorNamespace,
	}
	agentKey := types.NamespacedName{
		Name:      constants.EBPFAgentName,
		Namespace: operatorNamespace + "-privileged",
	}
	lokiCert := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "loki-gateway-ca-bundle",
			Namespace: "loki-namespace",
		},
		Data: map[string]string{
			"service-ca.crt": "--- LOKI CA CERT ---",
			"other":          "any",
		},
	}
	expectedLokiHash, _ := cmw.GetDigest(&lokiCert, []string{"service-ca.crt"}) // C80Sbg==
	kafkaCert := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-ca",
			Namespace: operatorNamespace,
		},
		Data: map[string]string{
			"cert.crt": "--- KAFKA CA CERT ---",
			"other":    "any",
		},
	}
	expectedKafkaHash, _ := cmw.GetDigest(&kafkaCert, []string{"cert.crt"}) // tDuVsw==
	kafkaUserCert := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-user",
			Namespace: operatorNamespace,
		},
		Data: map[string][]byte{
			"user.crt": []byte("--- KAFKA USER CERT ---"),
			"user.key": []byte("--- KAFKA USER KEY ---"),
			"other":    []byte("any"),
		},
	}
	expectedKafkaUserHash, _ := sw.GetDigest(&kafkaUserCert, []string{"user.crt", "user.key"}) // QztU6w==
	kafka2Cert := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-exporter-ca",
			Namespace: "kafka-exporter-namespace",
		},
		Data: map[string]string{
			"cert.crt": "--- KAFKA 2 CA CERT ---",
			"other":    "any",
		},
	}
	expectedKafka2Hash, _ := cmw.GetDigest(&kafka2Cert, []string{"cert.crt"}) // RO7D5Q==
	kafka2Sasl := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kafka-exporter-sasl",
			Namespace: operatorNamespace,
		},
		Data: map[string][]byte{
			"username": []byte("aiapaec"),
			"password": []byte("azerty"),
		},
	}
	expectedKafkaSaslHash1, _ := sw.GetDigest(&kafka2Sasl, []string{"username"}) // hlEvyw==
	expectedKafkaSaslHash2, _ := sw.GetDigest(&kafka2Sasl, []string{"password"}) // FOs6Rg==

	BeforeEach(func() {
		// Add any setup steps that needs to be executed before each test
	})

	AfterEach(func() {
		// Add any teardown steps that needs to be executed after each test
	})

	agent := appsv1.DaemonSet{}
	flp := appsv1.Deployment{}
	plugin := appsv1.Deployment{}

	Context("Verify expectations are sane", func() {
		It("Expected hashes should all be different", func() {
			cmEmpty, _ := cmw.GetDigest(&v1.ConfigMap{}, []string{"any"})
			sEmpty, _ := sw.GetDigest(&v1.Secret{}, []string{"any"})
			allKeys := map[string]interface{}{}
			for _, hash := range []string{"", cmEmpty, sEmpty, expectedLokiHash, expectedKafkaHash, expectedKafka2Hash, expectedKafkaUserHash, expectedKafkaSaslHash1, expectedKafkaSaslHash2} {
				allKeys[hash] = nil
			}
			Expect(allKeys).To(HaveLen(9))
		})
	})

	Context("Deploying with Loki and Kafka certificates", func() {
		It("Should create all certs successfully", func() {
			By("Creating Loki certificate")
			Eventually(func() interface{} { return k8sClient.Create(ctx, &lokiCert) }, timeout, interval).Should(Succeed())
			By("Creating Kafka CA certificate")
			Eventually(func() interface{} { return k8sClient.Create(ctx, &kafkaCert) }, timeout, interval).Should(Succeed())
			By("Creating Kafka User certificate")
			Eventually(func() interface{} { return k8sClient.Create(ctx, &kafkaUserCert) }, timeout, interval).Should(Succeed())
			By("Creating Kafka-export CA certificate")
			Eventually(func() interface{} { return k8sClient.Create(ctx, &kafka2Cert) }, timeout, interval).Should(Succeed())
			By("Creating Kafka-export SASL key")
			Eventually(func() interface{} { return k8sClient.Create(ctx, &kafka2Sasl) }, timeout, interval).Should(Succeed())
		})

		flowSpec := flowslatest.FlowCollectorSpec{
			Namespace:       operatorNamespace,
			DeploymentModel: flowslatest.DeploymentModelKafka,
			Agent: flowslatest.FlowCollectorAgent{
				Type: "eBPF",
				EBPF: flowslatest.FlowCollectorEBPF{},
			},
			Loki: flowslatest.FlowCollectorLoki{
				Enable: ptr.To(true),
				Mode:   flowslatest.LokiModeLokiStack,
				LokiStack: flowslatest.LokiStackRef{
					Name:      "loki",
					Namespace: "loki-namespace",
				},
			},
			Kafka: flowslatest.FlowCollectorKafka{
				TLS: flowslatest.ClientTLS{
					Enable: true,
					CACert: flowslatest.CertificateReference{
						Type:     flowslatest.RefTypeConfigMap,
						Name:     kafkaCert.Name,
						CertFile: "cert.crt",
						// No namespace means operator's namespace
					},
					UserCert: flowslatest.CertificateReference{
						Type:     flowslatest.RefTypeSecret,
						Name:     kafkaUserCert.Name,
						CertFile: "user.crt",
						CertKey:  "user.key",
						// No namespace means operator's namespace
					},
				},
			},
			Exporters: []*flowslatest.FlowCollectorExporter{{
				Type: flowslatest.KafkaExporter,
				Kafka: flowslatest.FlowCollectorKafka{
					TLS: flowslatest.ClientTLS{
						Enable: true,
						CACert: flowslatest.CertificateReference{
							Type:      flowslatest.RefTypeConfigMap,
							Name:      kafka2Cert.Name,
							Namespace: kafka2Cert.Namespace,
							CertFile:  "cert.crt",
						},
					},
					SASL: flowslatest.SASLConfig{
						Type: "Plain",
						ClientIDReference: flowslatest.FileReference{
							Type: flowslatest.RefTypeSecret,
							Name: kafka2Sasl.Name,
							File: "username",
						},
						ClientSecretReference: flowslatest.FileReference{
							Type: flowslatest.RefTypeSecret,
							Name: kafka2Sasl.Name,
							File: "password",
						},
					},
				},
			}},
		}

		It("Should create CR successfully", func() {
			Eventually(func() interface{} {
				return k8sClient.Create(ctx, &flowslatest.FlowCollector{
					ObjectMeta: metav1.ObjectMeta{
						Name: crKey.Name,
					},
					Spec: flowSpec,
				})
			}, timeout, interval).Should(Succeed())
		})

		It("Should copy certificates when necessary", func() {
			By("Expecting Loki CA cert copied to operator namespace")
			Eventually(func() interface{} {
				var cm v1.ConfigMap
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: lokiCert.Name, Namespace: operatorNamespace}, &cm); err != nil {
					return err
				}
				return cm.Data
			}, timeout, interval).Should(Equal(map[string]string{
				"service-ca.crt": "--- LOKI CA CERT ---",
				"other":          "any",
			}))
			By("Expecting Kafka CA cert copied to privileged namespace")
			Eventually(func() interface{} {
				var cm v1.ConfigMap
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: kafkaCert.Name, Namespace: operatorNamespace + "-privileged"}, &cm); err != nil {
					return err
				}
				return cm.Data
			}, timeout, interval).Should(Equal(map[string]string{
				"cert.crt": "--- KAFKA CA CERT ---",
				"other":    "any",
			}))
			By("Expecting Kafka User cert copied to privileged namespace")
			Eventually(func() interface{} {
				var s v1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: kafkaUserCert.Name, Namespace: operatorNamespace + "-privileged"}, &s); err != nil {
					return err
				}
				return s.Data
			}, timeout, interval).Should(Equal(map[string][]byte{
				"user.crt": []byte("--- KAFKA USER CERT ---"),
				"user.key": []byte("--- KAFKA USER KEY ---"),
				"other":    []byte("any"),
			}))
		})

		It("Should have all certificates mounted", func() {
			By("Expecting Kafka certificates for Agent mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, agentKey, &agent); err != nil {
					return err
				}
				return test.VolumeNames(agent.Spec.Template.Spec.Volumes)
			}, timeout, interval).Should(ContainElements(
				"kafka-certs-ca",
				"kafka-certs-user",
			))
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, agentKey, &agent); err != nil {
					return err
				}
				return test.Annotations(agent.Spec.Template.Annotations)
			}, timeout, interval).Should(ContainElements(
				"flows.netobserv.io/watched-kafka-ca="+expectedKafkaHash,
				"flows.netobserv.io/watched-kafka-user="+expectedKafkaUserHash,
			))

			By("Expecting Loki certificate for Plugin mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, pluginKey, &plugin); err != nil {
					return err
				}
				return test.VolumeNames(plugin.Spec.Template.Spec.Volumes)
			}, timeout, interval).Should(ContainElements(
				"netobserv-plugin-cert",
				"config-volume",
				"loki-certs-ca",
				"loki-status-certs-ca",
				"loki-status-certs-user",
			))
			Expect(plugin.Spec.Template.Annotations).To(HaveLen(1))

			By("Expecting Loki and Kafka certificates for FLP mounted")
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey, &flp); err != nil {
					return err
				}
				return test.VolumeNames(flp.Spec.Template.Spec.Volumes)
			}, timeout, interval).Should(ContainElements(
				"config-volume",
				"kafka-cert-ca",
				"kafka-cert-user",
				"flowlogs-pipeline",
				"loki-certs-ca",
				"kafka-export-0-ca",
				"kafka-export-0-sasl-id",
				"kafka-export-0-sasl-secret",
			))
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey, &flp); err != nil {
					return err
				}
				return test.Annotations(flp.Spec.Template.Annotations)
			}, timeout, interval).Should(ContainElements(
				"flows.netobserv.io/watched-kafka-ca="+expectedKafkaHash,
				"flows.netobserv.io/watched-kafka-user="+expectedKafkaUserHash,
				"flows.netobserv.io/watched-kafka-export-0-ca="+expectedKafka2Hash,
				"flows.netobserv.io/watched-kafka-export-0-sd1="+expectedKafkaSaslHash1,
				"flows.netobserv.io/watched-kafka-export-0-sd2="+expectedKafkaSaslHash2,
			))
		})
	})

	Context("Updating Kafka certificates", func() {
		var modifiedKafkaHash, modifiedKafkaUserHash string
		It("Should update Kafka certificate", func() {
			By("Updating Kafka CA certificate")
			kafkaCert.Data["cert.crt"] = "--- KAFKA CA CERT MODIFIED ---"
			Eventually(func() interface{} { return k8sClient.Update(ctx, &kafkaCert) }, timeout, interval).Should(Succeed())
			modifiedKafkaHash, _ = cmw.GetDigest(&kafkaCert, []string{"cert.crt"})
			Expect(modifiedKafkaHash).ToNot(Equal(expectedKafkaHash))
			By("Updating Kafka User certificate")
			kafkaUserCert.Data["user.crt"] = []byte("--- KAFKA USER CERT MODIFIED ---")
			Eventually(func() interface{} { return k8sClient.Update(ctx, &kafkaUserCert) }, timeout, interval).Should(Succeed())
			modifiedKafkaUserHash, _ = sw.GetDigest(&kafkaUserCert, []string{"user.crt", "user.key"})
			Expect(modifiedKafkaUserHash).ToNot(Equal(expectedKafkaUserHash))
		})

		It("Should copy certificates when necessary", func() {
			By("Expecting Kafka CA cert updated to privileged namespace")
			Eventually(func() interface{} {
				var cm v1.ConfigMap
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: kafkaCert.Name, Namespace: operatorNamespace + "-privileged"}, &cm); err != nil {
					return err
				}
				return cm.Data
			}, timeout, interval).Should(Equal(map[string]string{
				"cert.crt": "--- KAFKA CA CERT MODIFIED ---",
				"other":    "any",
			}))
			By("Expecting Kafka User cert updated to privileged namespace")
			Eventually(func() interface{} {
				var s v1.Secret
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: kafkaUserCert.Name, Namespace: operatorNamespace + "-privileged"}, &s); err != nil {
					return err
				}
				return s.Data
			}, timeout, interval).Should(Equal(map[string][]byte{
				"user.crt": []byte("--- KAFKA USER CERT MODIFIED ---"),
				"user.key": []byte("--- KAFKA USER KEY ---"),
				"other":    []byte("any"),
			}))
		})

		It("Should change eBPF Agent annotations", func() {
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, agentKey, &agent); err != nil {
					return err
				}
				return test.Annotations(agent.Spec.Template.Annotations)
			}, timeout, interval).Should(ContainElements(
				"flows.netobserv.io/watched-kafka-ca="+modifiedKafkaHash,
				"flows.netobserv.io/watched-kafka-user="+modifiedKafkaUserHash,
			))
		})

		It("Should change FLP annotations", func() {
			Eventually(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey, &flp); err != nil {
					return err
				}
				return test.Annotations(flp.Spec.Template.Annotations)
			}, timeout, interval).Should(ContainElements(
				"flows.netobserv.io/watched-kafka-ca="+modifiedKafkaHash,
				"flows.netobserv.io/watched-kafka-user="+modifiedKafkaUserHash,
				"flows.netobserv.io/watched-kafka-export-0-ca="+expectedKafka2Hash,
				"flows.netobserv.io/watched-kafka-export-0-sd1="+expectedKafkaSaslHash1,
				"flows.netobserv.io/watched-kafka-export-0-sd2="+expectedKafkaSaslHash2,
			))
		})
	})

	Context("Updating Loki certificate", func() {
		It("Should update Loki certificate", func() {
			By("Updating Loki CA certificate")
			lokiCert.Data["service-ca.crt"] = "--- LOKI CA CERT MODIFIED ---"
			Eventually(func() interface{} { return k8sClient.Update(ctx, &lokiCert) }, timeout, interval).Should(Succeed())
		})

		It("Should copy certificates when necessary", func() {
			By("Expecting Loki CA cert updated to operator namespace")
			Eventually(func() interface{} {
				var cm v1.ConfigMap
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: lokiCert.Name, Namespace: operatorNamespace}, &cm); err != nil {
					return err
				}
				return cm.Data
			}, timeout, interval).Should(Equal(map[string]string{
				"service-ca.crt": "--- LOKI CA CERT MODIFIED ---",
				"other":          "any",
			}))
		})

		// Console plugin is not restarted, as Loki certificate is always read from file
		It("Should not trigger Console plugin redeploy", func() {
			lastPluginAnnots := plugin.Spec.Template.Annotations
			Consistently(func() interface{} {
				if err := k8sClient.Get(ctx, pluginKey, &plugin); err != nil {
					return err
				}
				return plugin.Spec.Template.Annotations
			}, consistentlyTimeout, consistentlyInterval).Should(Equal(lastPluginAnnots))
		})

		// FLP is not restarted, as Loki certificate is always read from file
		It("Should not trigger FLP redeploy", func() {
			lastFLPAnnots := flp.Spec.Template.Annotations
			Consistently(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey, &flp); err != nil {
					return err
				}
				return flp.Spec.Template.Annotations
			}, consistentlyTimeout, consistentlyInterval).Should(Equal(lastFLPAnnots))
		})
	})

	Context("Dummy update of Kafka Secret/CM without cert change", func() {
		It("Should update Kafka Secret/CM", func() {
			By("Updating Kafka CM")
			kafkaCert.Annotations = map[string]string{"hey": "new annotation"}
			kafkaCert.Data["other"] = "any MODIFIED"
			Eventually(func() interface{} { return k8sClient.Update(ctx, &kafkaCert) }, timeout, interval).Should(Succeed())
			By("Updating Kafka Secret")
			kafkaUserCert.Annotations = map[string]string{"ho": "new annotation"}
			kafkaUserCert.Data["other"] = []byte("any MODIFIED")
			Eventually(func() interface{} { return k8sClient.Update(ctx, &kafkaUserCert) }, timeout, interval).Should(Succeed())
		})

		It("Should not redeploy Agent", func() {
			lastAgentAnnots := agent.Spec.Template.Annotations
			Consistently(func() interface{} {
				if err := k8sClient.Get(ctx, agentKey, &agent); err != nil {
					return err
				}
				return agent.Spec.Template.Annotations
			}, consistentlyTimeout, consistentlyInterval).Should(Equal(lastAgentAnnots))
		})

		It("Should not redeploy FLP", func() {
			lastFLPAnnots := flp.Spec.Template.Annotations
			Consistently(func() interface{} {
				if err := k8sClient.Get(ctx, flpKey, &flp); err != nil {
					return err
				}
				return flp.Spec.Template.Annotations
			}, consistentlyTimeout, consistentlyInterval).Should(Equal(lastFLPAnnots))
		})
	})

	Context("Checking CR ownership", func() {
		It("Should be garbage collected", func() {
			// Retrieve CR to get its UID
			By("Getting the CR")
			flowCR := getCR(crKey)

			By("Expecting flowlogs-pipeline deployment to be garbage collected")
			Eventually(func() interface{} {
				d := appsv1.Deployment{}
				_ = k8sClient.Get(ctx, flpKey, &d)
				return &d
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting agent daemonset to be garbage collected")
			Eventually(func() interface{} {
				d := appsv1.DaemonSet{}
				_ = k8sClient.Get(ctx, agentKey, &d)
				return &d
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))

			By("Expecting console plugin deployment to be garbage collected")
			Eventually(func() interface{} {
				d := appsv1.Deployment{}
				_ = k8sClient.Get(ctx, pluginKey, &d)
				return &d
			}, timeout, interval).Should(BeGarbageCollectedBy(flowCR))
		})
	})

	Context("Cleanup", func() {
		It("Should delete CR", func() {
			cleanupCR(crKey)
		})
	})
}
