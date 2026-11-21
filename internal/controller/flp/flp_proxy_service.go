package flp

import (
	"context"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/controller/reconcilers"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func proxyService(desired *flowslatest.FlowCollectorSpec, name, namespace, appLabel string) *corev1.Service {
	sources := desired.Processor.GetProxySources()
	if len(sources) == 0 {
		return nil
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			// Keep this Service outside the metrics ServiceMonitor's app selector.
			Labels: map[string]string{"part-of": constants.OperatorName, "app": name},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": appLabel},
		},
	}
	for i, source := range sources {
		port := source.GetPort()
		svc.Spec.Ports = append(svc.Spec.Ports, corev1.ServicePort{
			Name:       flowslatest.ProxySourcePortName(i),
			Port:       port,
			TargetPort: intstr.FromInt32(port),
			Protocol:   corev1.ProtocolUDP,
		})
	}
	return svc
}

func reconcileProxyService(ctx context.Context, info *reconcilers.Instance, current, desired *corev1.Service) error {
	if desired == nil {
		return info.Managed.TryDelete(ctx, current)
	}
	report := helper.NewChangeReport("FLP proxy service")
	defer report.LogIfNeeded(ctx)
	return info.ReconcileService(ctx, current, desired, &report)
}
