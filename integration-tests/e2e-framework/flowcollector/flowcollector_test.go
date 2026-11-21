package flowcollector_test

import (
	"context"
	"e2eframework/utils"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	e2eframework "e2eframework"

	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
	"sigs.k8s.io/yaml"
)

const minimalFlowCollector = `
apiVersion: flows.netobserv.io/v1beta2
kind: FlowCollector
metadata:
  name: cluster
`

// workloadDefaults restates the resource defaults from
// flowslatest.DefaultFlowCollector, deliberately written out rather than
// imported from the API package: an end-to-end test should pin the contract a
// user observes, not compare the operator against its own source of truth.
//
// This matters more than it looks. This is the only test in the repo that
// creates a FlowCollector from YAML with these fields genuinely *absent*. Every
// Go-client-based test (envtest, unit) serialises `resources: {}`, because
// `omitempty` does not omit structs - a present field, which therefore never
// received the old object-level OpenAPI default. That is exactly why the switch
// to in-code defaults changed behaviour here and nowhere else.
var workloadDefaults = []struct {
	workload   string
	namespace  string
	container  string
	daemonSet  bool
	requests   map[string]string
	limits     map[string]string
	goMemLimit string // empty when the component does not set it
}{
	{
		workload: "flowlogs-pipeline", namespace: e2eframework.Namespace, container: "flowlogs-pipeline",
		requests:   map[string]string{"cpu": "100m", "memory": "100Mi"},
		limits:     map[string]string{"memory": "800Mi"},
		goMemLimit: "754974720", // 90% of 800Mi
	},
	{
		workload: "netobserv-plugin", namespace: e2eframework.Namespace, container: "netobserv-plugin",
		requests: map[string]string{"cpu": "100m", "memory": "50Mi"},
		limits:   map[string]string{"memory": "100Mi"},
	},
	{
		workload: "netobserv-ebpf-agent", namespace: e2eframework.Namespace + "-privileged",
		container: "netobserv-ebpf-agent", daemonSet: true,
		requests:   map[string]string{"cpu": "100m", "memory": "50Mi"},
		limits:     map[string]string{"memory": "800Mi"},
		goMemLimit: "754974720", // 90% of 800Mi
	},
}

func TestFlowCollector_MinimalSpec(t *testing.T) {
	feature := features.New("FlowCollector/minimal").
		Assess("operator is running", func(_ context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			utils.WaitForDeploymentAvailable(t, cfg, "netobserv-controller-manager")
			return context.Background()
		}).
		Assess("minimal FlowCollector is reconciled", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			cmd := exec.Command("kubectl", "--kubeconfig", cfg.KubeconfigFile(), "apply", "-f", "-")
			cmd.Stdin = strings.NewReader(minimalFlowCollector)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("applying minimal FlowCollector: %v", err)
			}

			utils.WaitForDeploymentAvailable(t, cfg, "flowlogs-pipeline")
			utils.WaitForDeploymentAvailable(t, cfg, "netobserv-plugin")
			return ctx
		}).
		Assess("FlowCollector spec is not mutated with defaults", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			out, err := exec.Command("kubectl", "--kubeconfig", cfg.KubeconfigFile(), "get", "flowcollectors", "cluster", "-o", "yaml").Output()
			if err != nil {
				t.Fatalf("getting FlowCollector: %v", err)
			}

			var fc map[string]interface{}
			if err := yaml.Unmarshal(out, &fc); err != nil {
				t.Fatalf("unmarshaling FlowCollector: %v", err)
			}

			utils.AssertUnset(t, fc, "agent.ipfix.cacheActiveTimeout", "spec", "agent", "ipfix", "cacheActiveTimeout")
			utils.AssertUnset(t, fc, "loki.mode", "spec", "loki", "mode")
			utils.AssertUnset(t, fc, "prometheus.querier.mode", "spec", "prometheus", "querier", "mode")
			return ctx
		}).
		Assess("defaults are applied to the workloads", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			for _, tc := range workloadDefaults {
				t.Run(tc.workload, func(t *testing.T) {
					pod := utils.PodSpec(t, cfg, tc.namespace, tc.workload, tc.daemonSet)
					ctr := utils.FindContainer(t, &pod, tc.container)

					if got := utils.Quantities(ctr.Resources.Requests); !reflect.DeepEqual(got, tc.requests) {
						t.Errorf("resource requests: got %v, want %v", got, tc.requests)
					}
					if got := utils.Quantities(ctr.Resources.Limits); !reflect.DeepEqual(got, tc.limits) {
						t.Errorf("resource limits: got %v, want %v", got, tc.limits)
					}

					// GOMEMLIMIT is derived from the memory limit, so it is only
					// set once the default resources actually landed.
					got, found := utils.EnvValue(&ctr, "GOMEMLIMIT")
					switch {
					case tc.goMemLimit == "" && found:
						t.Errorf("unexpected GOMEMLIMIT=%s", got)
					case tc.goMemLimit != "" && !found:
						t.Errorf("missing GOMEMLIMIT, want %s", tc.goMemLimit)
					case tc.goMemLimit != got && found:
						t.Errorf("GOMEMLIMIT: got %s, want %s", got, tc.goMemLimit)
					}
				})
			}
			return ctx
		}).Feature()

	testEnv.Test(t, feature)
}
