package e2etests

import (
	"context"
	"fmt"
	"strings"
	"time"

	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// Kafka struct to handle default Kafka installation
type Kafka struct {
	Name      string
	Namespace string
	Template  string
}

// KafkaMetrics struct to handle kafka metrics config deployment
type KafkaMetrics struct {
	Namespace string
	Template  string
}

// KafkaNodePool struct handles creation of kafka node pool
type KafkaNodePool struct {
	Namespace    string
	NodePoolName string
	Name         string
	Template     string
}

// KafkaTopic struct handles creation of kafka topic
type KafkaTopic struct {
	Namespace string
	TopicName string
	Name      string
	Template  string
}

type KafkaUser struct {
	Namespace string
	UserName  string
	Name      string
	Template  string
}

// deploys default Kafka
func (kafka *Kafka) deployKafka(oc *exutil.CLI) {
	e2e.Logf("Deploy Default Kafka")
	parameters := []string{"--ignore-unknown-parameters=true", "-f", kafka.Template, "-p", "NAMESPACE=" + kafka.Namespace}

	if kafka.Name != "" {
		parameters = append(parameters, "NAME="+kafka.Name)
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, kafka.Namespace, parameters...)
}

// deploys Kafka Metrics
func (kafkaMetrics *KafkaMetrics) deployKafkaMetrics(oc *exutil.CLI) {
	e2e.Logf("Deploy Kafka metrics")
	parameters := []string{"--ignore-unknown-parameters=true", "-f", kafkaMetrics.Template, "-p", "NAMESPACE=" + kafkaMetrics.Namespace}

	compat_otp.ApplyNsResourceFromTemplate(oc, kafkaMetrics.Namespace, parameters...)
}

// creates a Kafka topic
func (kafkaTopic *KafkaTopic) deployKafkaTopic(oc *exutil.CLI) {
	e2e.Logf("Create Kafka topic")
	parameters := []string{"--ignore-unknown-parameters=true", "-f", kafkaTopic.Template, "-p", "NAMESPACE=" + kafkaTopic.Namespace}

	if kafkaTopic.Name != "" {
		parameters = append(parameters, "NAME="+kafkaTopic.Name)
	}

	if kafkaTopic.TopicName != "" {
		parameters = append(parameters, "TOPIC="+kafkaTopic.TopicName)
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, kafkaTopic.Namespace, parameters...)
}

// creates a Kafka nodePool
func (kafkaNodePool *KafkaNodePool) deployKafkaNodePool(oc *exutil.CLI) {
	e2e.Logf("Create Kafka nodePool")
	parameters := []string{"--ignore-unknown-parameters=true", "-f", kafkaNodePool.Template, "-p", "NAMESPACE=" + kafkaNodePool.Namespace}

	if kafkaNodePool.Name != "" {
		parameters = append(parameters, "NAME="+kafkaNodePool.Name)
	}

	if kafkaNodePool.NodePoolName != "" {
		parameters = append(parameters, "NODEPOOL="+kafkaNodePool.NodePoolName)
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, kafkaNodePool.Namespace, parameters...)
}

// deploys KafkaUser
func (kafkaUser *KafkaUser) deployKafkaUser(oc *exutil.CLI) {
	e2e.Logf("Create Kafka User")
	parameters := []string{"--ignore-unknown-parameters=true", "-f", kafkaUser.Template, "-p", "NAMESPACE=" + kafkaUser.Namespace}

	if kafkaUser.UserName != "" {
		parameters = append(parameters, "USER_NAME="+kafkaUser.UserName)
	}

	if kafkaUser.Name != "" {
		parameters = append(parameters, "NAME="+kafkaUser.Name)
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, kafkaUser.Namespace, parameters...)
}

// deletes kafkaUser
func (k *KafkaUser) deleteKafkaUser(oc *exutil.CLI) {
	e2e.Logf("Deleting Kafka user")
	command := []string{"kafkauser", k.UserName, "-n", k.Namespace}
	_, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args(command...).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// deletes kafkaTopic
func (kafkaTopic *KafkaTopic) deleteKafkaTopic(oc *exutil.CLI) {
	e2e.Logf("Deleting Kafka topic")
	command := []string{"kafkatopic", kafkaTopic.TopicName, "-n", kafkaTopic.Namespace}
	_, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args(command...).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// deletes kafkaNodePool
func (kafkaNodePool *KafkaNodePool) deleteKafkaNodePool(oc *exutil.CLI) {
	e2e.Logf("Deleting KafkaNodePool")
	command := []string{"kafkaNodePool", kafkaNodePool.NodePoolName, "-n", kafkaNodePool.Namespace}
	_, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args(command...).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// deletes kafka
func (kafka *Kafka) deleteKafka(oc *exutil.CLI) {
	e2e.Logf("Deleting Kafka")
	command := []string{"kafka", kafka.Name, "-n", kafka.Namespace}
	_, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args(command...).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// Poll to wait for kafka to be ready
func waitForKafkaReady(oc *exutil.CLI, kafkaName string, kafkaNS string) {
	err := wait.PollUntilContextTimeout(context.Background(), 6*time.Second, 360*time.Second, false, func(context.Context) (done bool, err error) {
		command := []string{"kafka", kafkaName, "-n", kafkaNS, `-o=jsonpath={.status.conditions[*].type}`}
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(command...).Output()
		if err != nil {
			e2e.Logf("kafka status ready error: %v", err)
			return false, err
		}
		if strings.Contains(output, "Ready") {
			return true, nil
		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("resource kafka/%s did not appear", kafkaName))
}

// Poll to wait for kafka Topic to be ready
func waitForKafkaTopicReady(oc *exutil.CLI, kafkaTopicName string, kafkaTopicNS string) {
	err := wait.PollUntilContextTimeout(context.Background(), 6*time.Second, 360*time.Second, false, func(context.Context) (done bool, err error) {
		command := []string{"kafkaTopic", kafkaTopicName, "-n", kafkaTopicNS, `-o=jsonpath='{.status.conditions[*].type}'`}
		output, err := oc.AsAdmin().WithoutNamespace().Run("get").Args(command...).Output()
		if err != nil {
			e2e.Logf("kafka Topic status ready error: %v", err)
			return false, err
		}
		status := strings.Replace(output, "'", "", 2)
		e2e.Logf("Waiting for kafka status %s", status)
		if status == "Ready" {
			return true, nil
		}
		return false, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("resource kafkaTopic/%s did not appear", kafkaTopicName))
}
