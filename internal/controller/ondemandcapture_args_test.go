package controllers

import (
	odcapi "github.com/netobserv/netobserv-operator/api/ondemandcapture/v1beta1"
	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
	"strings"
	"testing"
)

func TestAllCaptureFilters(t *testing.T) {
	rule := odcapi.CaptureFilterRule{Direction: "Ingress", CIDR: "10.0.0.0/8", Protocol: "TCP", SPort: ptr.To(int32(80)), DPort: ptr.To(int32(443)), Port: ptr.To(int32(53)), SPortRange: "80-90", DPortRange: "400-500", PortRange: "100-200", Sports: "80,81", Dports: "443,444", Ports: "53,54", ICMPType: ptr.To(int32(0)), ICMPCode: ptr.To(int32(0)), PeerIP: "10.0.0.1", PeerCIDR: "10.1.0.0/16", Action: "Accept", TCPFlags: "SYN", Drops: true}
	want := []string{"--direction=Ingress", "--cidr=10.0.0.0/8", "--protocol=TCP", "--sport=80", "--dport=443", "--port=53", "--sport_range=80-90", "--dport_range=400-500", "--port_range=100-200", "--sports=80,81", "--dports=443,444", "--ports=53,54", "--icmp_type=0", "--icmp_code=0", "--peer_ip=10.0.0.1", "--peer_cidr=10.1.0.0/16", "--action=Accept", "--tcp_flags=SYN", "--drops"}
	require.Equal(t, want, captureRuleArgs(&rule))
	f := &odcapi.CaptureFilters{CaptureFilterRule: rule, Or: []odcapi.CaptureFilterRule{{Protocol: "UDP", DPort: ptr.To(int32(53))}}}
	require.Equal(t, append(want, "or", "--protocol=UDP", "--dport=53"), captureFilterArgs(f))
	require.NoError(t, validateFilterGroups(f))
	f.CaptureFilterRule = odcapi.CaptureFilterRule{}
	require.Equal(t, []string{"--protocol=UDP", "--dport=53"}, captureFilterArgs(f))
	f.Or = append(f.Or, odcapi.CaptureFilterRule{})
	require.Error(t, validateFilterGroups(f))
}
func TestCaptureArgs(t *testing.T) {
	for _, key := range append(append([]string{}, captureFilterNames...), captureValueOptions...) {
		t.Run(key, func(t *testing.T) {
			_, err := validateCaptureArgs([]string{"--" + key + "=value"})
			require.NoError(t, err)
		})
	}
	for _, key := range captureBoolOptions {
		_, err := validateCaptureArgs([]string{"--" + key})
		require.NoError(t, err)
	}
	for _, key := range managedCaptureOptions {
		for _, prefix := range []string{"--", ""} {
			_, err := validateCaptureArgs([]string{prefix + key + "=override"})
			require.Error(t, err)
		}
	}
	for _, args := range [][]string{{"--unknown=true"}, {"--sampling"}, {"--sampling", "--namespace=x"}, {"--query", "help"}} {
		_, err := validateCaptureArgs(args)
		require.Error(t, err)
	}
	for _, args := range [][]string{{"--drops"}, {"--cidr", "10.0.0.0/8"}, {"--protocol=TCP", "or", "--protocol=UDP"}} {
		has, err := validateCaptureArgs(args)
		require.NoError(t, err)
		require.True(t, has)
	}
	has, err := validateCaptureArgs([]string{"--drops=false"})
	require.NoError(t, err)
	require.False(t, has)
}
func TestAdditionalArgsArePreservedAndAppliedLast(t *testing.T) {
	r := &OnDemandCaptureReconciler{netobservCLIImage: "example/cli:test"}
	o := &odcapi.OnDemandCapture{Spec: odcapi.OnDemandCaptureSpec{CaptureType: odcapi.CaptureTypeFlows, Args: []string{"--max-time=45s", "--query", `SELECT * FROM flow WHERE SrcAddr = '10.0.0.1'; $(literal)`, "--enable_network_events"}}}
	require.NoError(t, r.validateSpec(o))
	args := r.buildCLIArgs(o)
	require.Equal(t, o.Spec.Args, args[len(args)-len(o.Spec.Args):])
	require.Contains(t, strings.Join(args, " "), "--headless")
	o.Spec.CaptureType = odcapi.CaptureTypePackets
	o.Spec.Args = []string{"--cidr=10.0.0.0/8"}
	require.NoError(t, r.validateSpec(o))
	o.Spec.Args = nil
	o.Spec.Filters = &odcapi.CaptureFilters{CaptureFilterRule: odcapi.CaptureFilterRule{Drops: true}}
	require.NoError(t, r.validateSpec(o))
}
