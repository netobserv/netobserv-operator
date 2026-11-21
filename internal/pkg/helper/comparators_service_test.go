package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestServiceChangedDetectsPortRemoval(t *testing.T) {
	desired := &corev1.Service{Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "prometheus", Port: 9401, TargetPort: intstr.FromInt32(9401), Protocol: corev1.ProtocolTCP}}}}
	stored := desired.DeepCopy()
	// Server-assigned fields must not cause repeated updates.
	stored.Spec.ClusterIP = "172.30.0.5"
	stored.Spec.ClusterIPs = []string{"172.30.0.5"}
	require.False(t, ServiceChanged(stored, desired, &ChangeReport{}))
	stored.Spec.Ports = append(stored.Spec.Ports, corev1.ServicePort{Name: "haproxy", Port: 10514, TargetPort: intstr.FromInt32(10514), Protocol: corev1.ProtocolUDP})
	require.True(t, ServiceChanged(stored, desired, &ChangeReport{}), "disabling the receiver must remove its UDP Service port")
	require.True(t, ServiceChanged(desired, stored, &ChangeReport{}), "enabling the receiver must add its UDP Service port")
	stored.Spec.Ports = desired.Spec.Ports
	require.False(t, ServiceChanged(stored, desired, &ChangeReport{}))
}
