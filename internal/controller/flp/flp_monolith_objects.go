package flp

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/volumes"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	monoName           = constants.FLPName
	monoShortName      = constants.FLPShortName
	monoConfigMap      = monoName + "-config"
	monoDynConfigMap   = monoName + "-config-dynamic"
	monoServiceMonitor = monoName + "-monitor"
	monoPromRule       = monoName + "-alert"
	monoCertSecretName = monoName + "-cert"
)

type monolithBuilder struct {
	info            *reconcilers.Instance
	desired         *flowslatest.FlowCollectorSpec
	flowMetrics     *metricslatest.FlowMetricList
	fcSlices        []sliceslatest.FlowCollectorSlice
	detectedSubnets []flowslatest.SubnetLabel
	version         string
	promTLS         *flowslatest.CertificateReference
	volumes         volumes.Builder
}

func newMonolithBuilder(info *reconcilers.Instance, desired *flowslatest.FlowCollectorSpec, flowMetrics *metricslatest.FlowMetricList, fcSlices []sliceslatest.FlowCollectorSlice, detectedSubnets []flowslatest.SubnetLabel) (monolithBuilder, error) {
	version := helper.ExtractVersion(info.Images[reconcilers.MainImage])
	promTLS, err := getPromTLS(desired, constants.FLPMetricsSvcName)
	if err != nil {
		return monolithBuilder{}, err
	}

	return monolithBuilder{
		info:            info,
		desired:         desired,
		flowMetrics:     flowMetrics,
		fcSlices:        fcSlices,
		detectedSubnets: detectedSubnets,
		version:         helper.MaxLabelLength(version),
		promTLS:         promTLS,
	}, nil
}

func (b *monolithBuilder) daemonSet(annotations map[string]string) *appsv1.DaemonSet {
	netType := hostNetwork
	if b.info.ClusterInfo.IsOpenShift() {
		netType = hostPort
	}
	pod := podTemplate(
		monoName,
		b.version,
		b.info.Images[reconcilers.MainImage],
		monoConfigMap,
		b.desired,
		&b.volumes,
		netType,
		annotations,
	)
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     monoName,
				"version": b.version,
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": monoName},
			},
			Template: pod,
		},
	}
}

func (b *monolithBuilder) deployment(annotations map[string]string) *appsv1.Deployment {
	pod := podTemplate(
		monoName,
		b.version,
		b.info.Images[reconcilers.MainImage],
		monoConfigMap,
		b.desired,
		&b.volumes,
		svc,
		annotations,
	)
	replicas := b.desired.Processor.GetFLPReplicas()
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     monoName,
				"version": b.version,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": monoName},
			},
			Template: pod,
		},
	}
}

func (b *monolithBuilder) configMaps() (*corev1.ConfigMap, string, *corev1.ConfigMap, error) {
	pipeline, err := createPipeline(
		b.desired,
		b.flowMetrics,
		b.fcSlices,
		b.detectedSubnets,
		b.info.Loki,
		b.info.ClusterInfo.GetID(),
		&b.volumes,
		newGRPCPipeline(b.desired, &b.volumes),
	)
	if err != nil {
		return nil, "", nil, err
	}

	// Get static and dynamic CM
	static, dynamic, err := getJSONConfigs(b.desired, &b.volumes, b.promTLS, pipeline, monoDynConfigMap)
	if err != nil {
		return nil, "", nil, err
	}
	staticCM, digest, err := configMap(monoConfigMap, b.info.Namespace, static, monoName)
	if err != nil {
		return nil, "", nil, err
	}
	dynamicCM, _, err := configMap(monoDynConfigMap, b.info.Namespace, dynamic, monoName)
	if err != nil {
		return nil, "", nil, err
	}

	return staticCM, digest, dynamicCM, err
}

func (b *monolithBuilder) service() *corev1.Service {
	advancedConfig := helper.GetAdvancedProcessorConfig(b.desired)
	port := *advancedConfig.Port
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        monoName,
			Namespace:   b.info.Namespace,
			Annotations: map[string]string{},
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     monoName,
				"version": b.version,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": monoName},
			Ports: []corev1.ServicePort{{
				Name:       constants.FLPPortName,
				Port:       port,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt32(port),
			}},
		},
	}
	if b.info.ClusterInfo.IsOpenShift() {
		svc.Annotations[constants.OpenShiftCertificateAnnotation] = monoCertSecretName
	}
	return &svc
}

func (b *monolithBuilder) promService() *corev1.Service {
	return promService(
		b.desired,
		constants.FLPMetricsSvcName,
		b.info.Namespace,
		monoName,
	)
}

func (b *monolithBuilder) serviceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      monoName,
			Namespace: b.info.Namespace,
			Labels: map[string]string{
				"part-of": constants.OperatorName,
				"app":     monoName,
			},
		},
	}
}

func (b *monolithBuilder) serviceMonitor() *monitoringv1.ServiceMonitor {
	return serviceMonitor(
		b.desired,
		monoServiceMonitor,
		constants.FLPMetricsSvcName,
		b.info.Namespace,
		monoName,
		b.version,
		b.info.IsDownstream,
		b.info.ClusterInfo.HasPromServiceDiscoveryRole(),
	)
}

func (b *monolithBuilder) prometheusRule(rules []monitoringv1.Rule) *monitoringv1.PrometheusRule {
	return prometheusRule(
		rules,
		monoPromRule,
		b.info.Namespace,
		monoName,
		b.version,
	)
}
