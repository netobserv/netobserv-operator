package util //nolint:revive

import (
	"context"
	"errors"
	"io"
	"os"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func SpecForMetrics(metrics ...string) *flowslatest.FlowCollectorSpec {
	fc := flowslatest.FlowCollectorSpec{
		Agent: flowslatest.FlowCollectorAgent{
			EBPF: flowslatest.FlowCollectorEBPF{
				Privileged: true,
				Features:   []flowslatest.AgentFeature{flowslatest.FlowRTT, flowslatest.DNSTracking, flowslatest.PacketDrop, flowslatest.UDNMapping, flowslatest.IPSec, flowslatest.TLSTracking},
			},
		},
		Processor: flowslatest.FlowCollectorFLP{
			Metrics:                flowslatest.FLPMetrics{},
			AddZone:                ptr.To(true),
			MultiClusterDeployment: ptr.To(true),
		},
	}
	if len(metrics) > 0 {
		var conv []flowslatest.FLPMetric
		for _, m := range metrics {
			conv = append(conv, flowslatest.FLPMetric(m))
		}
		fc.Processor.Metrics.IncludeList = &conv
	}
	return &fc
}

// InstallYAMLAsset reads a YAML file (possibly multi-doc) and creates each object.
func InstallYAMLAsset(ctx context.Context, k8sClient client.Client, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := yaml.NewYAMLOrJSONDecoder(f, 4096)
	for {
		obj := &unstructured.Unstructured{}
		if err := dec.Decode(obj); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if len(obj.Object) == 0 {
			continue
		}
		if err := k8sClient.Create(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

// InstallYAMLAssets reads provided YAML files (possibly multi-doc) and creates each object.
func InstallYAMLAssets(ctx context.Context, k8sClient client.Client, path ...string) error {
	for _, p := range path {
		if err := InstallYAMLAsset(ctx, k8sClient, p); err != nil {
			return err
		}
	}
	return nil
}
