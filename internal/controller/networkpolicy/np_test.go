package networkpolicy

import (
	"testing"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/pkg/cluster"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	"github.com/stretchr/testify/assert"

	ascv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

var outputRecordTypes = flowslatest.LogTypeAll

func getLoki() flowslatest.FlowCollectorLoki {
	return flowslatest.FlowCollectorLoki{
		Mode: flowslatest.LokiModeManual,
		Manual: flowslatest.LokiManualParams{
			IngesterURL: "http://loki:3100/",
		},
		Enable: ptr.To(true),
		WriteBatchWait: &metav1.Duration{
			Duration: 1,
		},
		WriteBatchSize: 102400,
		Advanced: &flowslatest.AdvancedLokiConfig{
			WriteMinBackoff: &metav1.Duration{
				Duration: 1,
			},
			WriteMaxBackoff: &metav1.Duration{
				Duration: 300,
			},
			WriteMaxRetries: ptr.To(int32(10)),
			StaticLabels:    map[string]string{"app": "netobserv-flowcollector"},
		},
	}
}

func getConfig() flowslatest.FlowCollector {
	return flowslatest.FlowCollector{
		Spec: flowslatest.FlowCollectorSpec{
			DeploymentModel: flowslatest.DeploymentModelService,
			Agent:           flowslatest.FlowCollectorAgent{Type: flowslatest.AgentEBPF},
			Processor: flowslatest.FlowCollectorFLP{
				LogLevel:         "trace",
				ConsumerReplicas: ptr.To(int32(1)),
				KafkaConsumerAutoscaler: flowslatest.FlowCollectorHPA{
					Status:  flowslatest.HPAStatusEnabled,
					Metrics: []ascv2.MetricSpec{},
				},
				LogTypes: &outputRecordTypes,
				Advanced: &flowslatest.AdvancedProcessorConfig{
					Port:       ptr.To(int32(2055)),
					HealthPort: ptr.To(int32(8080)),
				},
			},
			Loki: getLoki(),
			Kafka: flowslatest.FlowCollectorKafka{
				Address: "kafka",
				Topic:   "flp",
			},
		},
	}
}

func TestNpBuilder(t *testing.T) {
	assert := assert.New(t)

	desired := getConfig()
	mgr := &manager.Manager{ClusterInfo: &cluster.Info{}}

	desired.Spec.NetworkPolicy.Enable = nil
	name, np := buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, nil)
	assert.Equal(netpolName, name.Name)
	assert.Equal("netobserv", name.Namespace)
	assert.NotNil(np)
	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes)
	assert.Equal(netpolName, name.Name)
	assert.Equal("netobserv-privileged", name.Namespace)
	assert.NotNil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(false)
	_, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, nil)
	assert.Nil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes)
	assert.Nil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(true)
	name, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, nil)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{
		{
			From: []networkingv1.NetworkPolicyPeer{
				{PodSelector: &metav1.LabelSelector{}},
			},
		},
		{
			From: []networkingv1.NetworkPolicyPeer{
				{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ovn-host-network"}}},
			}, Ports: []networkingv1.NetworkPolicyPort{{Protocol: ptr.To(v1.ProtocolTCP), Port: ptr.To(intstr.FromInt(2055))}},
		},
	}, np.Spec.Ingress)

	assert.Equal([]networkingv1.NetworkPolicyEgressRule{
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"kube-system"},
				}},
			}},
		}},
	}, np.Spec.Egress)

	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{}, np.Spec.Ingress)

	desired.Spec.NetworkPolicy.AdditionalNamespaces = []string{"foo", "bar"}
	name, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, nil)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{
		{From: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{
			From: []networkingv1.NetworkPolicyPeer{
				{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "ovn-host-network"}}},
			}, Ports: []networkingv1.NetworkPolicyPort{{Protocol: ptr.To(v1.ProtocolTCP), Port: ptr.To(intstr.FromInt(2055))}},
		},
		{From: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"foo", "bar"},
				}},
			}},
		}},
	}, np.Spec.Ingress)

	assert.Equal([]networkingv1.NetworkPolicyEgressRule{
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"kube-system", "foo", "bar"},
				}},
			}},
		}},
	}, np.Spec.Egress)

	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{}, np.Spec.Ingress)
}

func TestNpBuilderSDN(t *testing.T) {
	assert := assert.New(t)

	desired := getConfig()
	mgr := &manager.Manager{ClusterInfo: &cluster.Info{}}

	desired.Spec.NetworkPolicy.Enable = nil
	_, np := buildMainNetworkPolicy(&desired, mgr, flowslatest.OpenShiftSDN, nil)
	assert.Nil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OpenShiftSDN)
	assert.Nil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(true)
	_, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.OpenShiftSDN, nil)
	assert.Nil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.OpenShiftSDN)
	assert.Nil(np)
}

func TestNpBuilderKindnet(t *testing.T) {
	assert := assert.New(t)

	desired := getConfig()
	mgr := &manager.Manager{ClusterInfo: &cluster.Info{}}

	desired.Spec.NetworkPolicy.Enable = nil
	name, np := buildMainNetworkPolicy(&desired, mgr, flowslatest.Kindnet, nil)
	assert.Equal(netpolName, name.Name)
	assert.Equal("netobserv", name.Namespace)
	assert.NotNil(np)
	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.Kindnet)
	assert.Equal(netpolName, name.Name)
	assert.Equal("netobserv-privileged", name.Namespace)
	assert.NotNil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(false)
	_, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.Kindnet, nil)
	assert.Nil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.Kindnet)
	assert.Nil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(true)
	name, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.Kindnet, nil)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{
		{From: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
	}, np.Spec.Ingress)

	assert.Equal([]networkingv1.NetworkPolicyEgressRule{
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"kube-system"},
				}},
			}},
		}},
	}, np.Spec.Egress)

	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.Kindnet)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{}, np.Spec.Ingress)

	desired.Spec.NetworkPolicy.AdditionalNamespaces = []string{"foo", "bar"}
	name, np = buildMainNetworkPolicy(&desired, mgr, flowslatest.Kindnet, nil)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{
		{From: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{From: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"foo", "bar"},
				}},
			}},
		}},
	}, np.Spec.Ingress)

	assert.Equal([]networkingv1.NetworkPolicyEgressRule{
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}},
		}},
		{To: []networkingv1.NetworkPolicyPeer{
			{PodSelector: &metav1.LabelSelector{}, NamespaceSelector: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      "kubernetes.io/metadata.name",
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"kube-system", "foo", "bar"},
				}},
			}},
		}},
	}, np.Spec.Egress)

	name, np = buildPrivilegedNetworkPolicy(&desired, mgr, flowslatest.Kindnet)
	assert.NotNil(np)
	assert.Equal(np.ObjectMeta.Name, name.Name)
	assert.Equal(np.ObjectMeta.Namespace, name.Namespace)
	assert.Equal([]networkingv1.NetworkPolicyIngressRule{}, np.Spec.Ingress)
}

func TestNpBuilderOtherCNI(t *testing.T) {
	assert := assert.New(t)

	desired := getConfig()
	mgr := &manager.Manager{ClusterInfo: &cluster.Info{}}

	desired.Spec.NetworkPolicy.Enable = nil
	_, np := buildMainNetworkPolicy(&desired, mgr, "", nil)
	assert.Nil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, "")
	assert.Nil(np)

	desired.Spec.NetworkPolicy.Enable = ptr.To(true)
	_, np = buildMainNetworkPolicy(&desired, mgr, "", nil)
	assert.NotNil(np)
	_, np = buildPrivilegedNetworkPolicy(&desired, mgr, "")
	assert.NotNil(np)
}

func TestNpBuilderWithAPIServerIPs(t *testing.T) {
	assert := assert.New(t)

	desired := getConfig()
	clusterInfo := &cluster.Info{}
	clusterInfo.Mock("4.14.0", flowslatest.OVNKubernetes) // Mock as OpenShift 4.14 with OVN
	mgr := &manager.Manager{
		ClusterInfo: clusterInfo,
		Config:      &manager.Config{DownstreamDeployment: false},
	}

	// Test with specific API server IPs (HyperShift scenario)
	apiServerIPs := []string{"172.20.0.1", "10.0.0.5"}
	_, np := buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, apiServerIPs)
	assert.NotNil(np)

	// Verify that we have a single egress rule with multiple IP peers
	found := false
	for _, egressRule := range np.Spec.Egress {
		if len(egressRule.To) == 2 && egressRule.To[0].IPBlock != nil && egressRule.To[1].IPBlock != nil {
			found = true
			// Verify both IPs are present with correct /32 CIDR for IPv4
			cidrs := []string{egressRule.To[0].IPBlock.CIDR, egressRule.To[1].IPBlock.CIDR}
			assert.Contains(cidrs, "172.20.0.1/32")
			assert.Contains(cidrs, "10.0.0.5/32")
		}
	}
	assert.True(found, "Expected to find a single egress rule with multiple API server IPs")

	// Test without API server IPs - should not create the external API server egress rule
	_, npWithoutIPs := buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, nil)
	assert.NotNil(npWithoutIPs)

	// Verify that we do NOT have an egress rule for external API server when IPs are not provided
	// This is more secure - we prefer breaking connectivity over opening all IPs on port 6443
	foundExternalAPIRule := false
	for _, egressRule := range npWithoutIPs.Spec.Egress {
		// Check if this is an external API server rule (would have IPBlock or empty To with port 6443)
		if len(egressRule.To) == 0 && len(egressRule.Ports) > 0 {
			for _, port := range egressRule.Ports {
				if port.Port != nil && port.Port.IntVal == 6443 {
					foundExternalAPIRule = true
				}
			}
		}
	}
	assert.False(foundExternalAPIRule, "Should not create external API server egress rule without specific IPs")

	// Test with IPv6 addresses
	apiServerIPsV6 := []string{"2001:db8::1", "2001:db8::2"}
	_, npV6 := buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, apiServerIPsV6)
	assert.NotNil(npV6)

	// Verify IPv6 addresses get /128 CIDR
	foundV6 := false
	for _, egressRule := range npV6.Spec.Egress {
		if len(egressRule.To) == 2 && egressRule.To[0].IPBlock != nil && egressRule.To[1].IPBlock != nil {
			foundV6 = true
			cidrs := []string{egressRule.To[0].IPBlock.CIDR, egressRule.To[1].IPBlock.CIDR}
			assert.Contains(cidrs, "2001:db8::1/128", "IPv6 addresses should use /128")
			assert.Contains(cidrs, "2001:db8::2/128", "IPv6 addresses should use /128")
		}
	}
	assert.True(foundV6, "Expected to find IPv6 egress rule")

	// Test with mixed IPv4 and IPv6
	apiServerIPsMixed := []string{"192.168.1.1", "2001:db8::1"}
	_, npMixed := buildMainNetworkPolicy(&desired, mgr, flowslatest.OVNKubernetes, apiServerIPsMixed)
	assert.NotNil(npMixed)

	foundMixed := false
	for _, egressRule := range npMixed.Spec.Egress {
		if len(egressRule.To) == 2 && egressRule.To[0].IPBlock != nil && egressRule.To[1].IPBlock != nil {
			foundMixed = true
			cidrs := []string{egressRule.To[0].IPBlock.CIDR, egressRule.To[1].IPBlock.CIDR}
			assert.Contains(cidrs, "192.168.1.1/32", "IPv4 should use /32")
			assert.Contains(cidrs, "2001:db8::1/128", "IPv6 should use /128")
		}
	}
	assert.True(foundMixed, "Expected to find mixed IPv4/IPv6 egress rule")
}
