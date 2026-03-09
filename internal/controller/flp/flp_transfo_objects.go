package flp

import (
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

const (
	transfoName           = constants.FLPTransfoName
	transfoShortName      = constants.FLPShortName + "transfo"
	transfoConfigMap      = transfoName + "-config"
	transfoDynConfigMap   = transfoName + "-config-dynamic"
	transfoServiceMonitor = transfoName + "-monitor"
	transfoPromRule       = transfoName + "-alert"
)

type transfoBuilder struct {
	info            *reconcilers.Instance
	desired         *flowslatest.FlowCollectorSpec
	flowMetrics     *metricslatest.FlowMetricList
	fcSlices        []sliceslatest.FlowCollectorSlice
	detectedSubnets []flowslatest.SubnetLabel
	version         string
	promTLS         *flowslatest.CertificateReference
	volumes         volumes.Builder
}

func newTransfoBuilder(info *reconcilers.Instance, desired *flowslatest.FlowCollectorSpec, flowMetrics *metricslatest.FlowMetricList, fcSlices []sliceslatest.FlowCollectorSlice, detectedSubnets []flowslatest.SubnetLabel) (transfoBuilder, error) {
	version := helper.ExtractVersion(info.Images[reconcilers.MainImage])
	promTLS, err := getPromTLS(desired, constants.FLPTransfoMetricsSvcName)
	if err != nil {
		return transfoBuilder{}, err
	}
	return transfoBuilder{
		info:            info,
		desired:         desired,
		flowMetrics:     flowMetrics,
		fcSlices:        fcSlices,
		detectedSubnets: detectedSubnets,
		version:         helper.MaxLabelLength(version),
		promTLS:         promTLS,
	}, nil
}

func (b *transfoBuilder) deployment(annotations map[string]string) *appsv1.Deployment {
	pod := podTemplate(
		transfoName,
		b.version,
		b.info.Images[reconcilers.MainImage],
		transfoConfigMap,
		b.desired,
		&b.volumes,
		pull,
		annotations,
	)
	replicas := b.desired.Processor.GetFLPReplicas()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      transfoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     transfoName,
				"version": b.version,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": transfoName},
			},
			Template: pod,
		},
	}
}

func (b *transfoBuilder) configMaps() (*corev1.ConfigMap, string, *corev1.ConfigMap, error) {
	pipeline, err := createPipeline(
		b.desired,
		b.flowMetrics,
		b.fcSlices,
		b.detectedSubnets,
		b.info.Loki,
		b.info.ClusterInfo.GetID(),
		&b.volumes,
		newKafkaPipeline(b.desired, &b.volumes),
	)
	if err != nil {
		return nil, "", nil, err
	}

	// Get static and dynamic CM
	static, dynamic, err := getJSONConfigs(b.desired, &b.volumes, b.promTLS, pipeline, transfoDynConfigMap)
	if err != nil {
		return nil, "", nil, err
	}
	staticCM, digest, err := configMap(transfoConfigMap, b.info.Namespace, static, transfoName)
	if err != nil {
		return nil, "", nil, err
	}
	dynamicCM, _, err := configMap(transfoDynConfigMap, b.info.Namespace, dynamic, transfoName)
	if err != nil {
		return nil, "", nil, err
	}

	return staticCM, digest, dynamicCM, err
}

func (b *transfoBuilder) promService() *corev1.Service {
	return promService(
		b.desired,
		constants.FLPTransfoMetricsSvcName,
		b.info.Namespace,
		transfoName,
	)
}

func (b *transfoBuilder) autoScaler() *ascv2.HorizontalPodAutoscaler {
	return &ascv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      transfoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     transfoName,
			},
		},
		Spec: ascv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: ascv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       transfoName,
			},
			MinReplicas: b.desired.Processor.KafkaConsumerAutoscaler.MinReplicas,
			MaxReplicas: b.desired.Processor.KafkaConsumerAutoscaler.MaxReplicas,
			Metrics:     b.desired.Processor.KafkaConsumerAutoscaler.Metrics,
		},
	}
}

func (b *transfoBuilder) serviceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      transfoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     transfoName,
			},
		},
	}
}

func (b *transfoBuilder) serviceMonitor() *monitoringv1.ServiceMonitor {
	return serviceMonitor(
		b.desired,
		transfoServiceMonitor,
		constants.FLPTransfoMetricsSvcName,
		b.info.Namespace,
		transfoName,
		b.version,
		b.info.IsDownstream,
		b.info.ClusterInfo.HasPromServiceDiscoveryRole(),
	)
}

func (b *transfoBuilder) prometheusRule(rules []monitoringv1.Rule) *monitoringv1.PrometheusRule {
	return prometheusRule(
		rules,
		transfoPromRule,
		b.info.Namespace,
		transfoName,
		b.version,
	)
}
