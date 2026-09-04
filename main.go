/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	_ "embed"
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"strconv"

	bpfmaniov1alpha1 "github.com/bpfman/bpfman-operator/apis/v1alpha1"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	configv1 "github.com/openshift/api/config/v1"
	osv1 "github.com/openshift/api/console/v1"
	operatorsv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	ascv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	flowsv1beta2 "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	slicesv1alpha1 "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	metricsv1alpha1 "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	controllers "github.com/netobserv/netobserv-operator/internal/controller"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/manager"
	//+kubebuilder:scaffold:imports
)

const app = constants.OperatorName

var (
	buildVersion = "unknown"
	buildDate    = "unknown"
	scheme       = runtime.NewScheme()
	setupLog     = ctrl.Log.WithName("setup")
)

//go:embed config/crd/bases/flows.netobserv.io_flowcollectors.yaml
var crdBytes []byte

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(flowsv1beta2.AddToScheme(scheme))
	utilruntime.Must(metricsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(slicesv1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(ascv2.AddToScheme(scheme))
	utilruntime.Must(osv1.AddToScheme(scheme))
	utilruntime.Must(apiregv1.AddToScheme(scheme))
	utilruntime.Must(securityv1.AddToScheme(scheme))
	utilruntime.Must(operatorsv1.AddToScheme(scheme))
	utilruntime.Must(monitoringv1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
	utilruntime.Must(bpfmaniov1alpha1.Install(scheme))
	utilruntime.Must(lokiv1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(olm.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var metricsCertFile string
	var metricsCertKeyFile string
	var enableLeaderElection bool
	var probeAddr string
	var pprofAddr string
	var enableHTTP2 bool
	var versionFlag bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&metricsCertFile, "metrics-cert-file", "", "The path to the TLS certificate for metrics.")
	flag.StringVar(&metricsCertKeyFile, "metrics-cert-key-file", "", "The path to the TLS certificate key for metrics.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "profiling-bind-address", "", "The address the profiling endpoint binds to, such as ':6060'. Leave unset to disable profiling.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&enableHTTP2, "enable-http2", enableHTTP2, "If HTTP/2 should be enabled for the metrics and webhook servers.")
	flag.BoolVar(&versionFlag, "v", false, "print version")
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.ISO8601TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	appVersion := fmt.Sprintf("%s [build version: %s, build date: %s]", app, buildVersion, buildDate)
	if versionFlag {
		fmt.Println(appVersion)
		os.Exit(0)
	}
	setupLog.Info("Starting " + appVersion)

	config, err := readConfigFromEnv()
	if err != nil {
		setupLog.Error(err, "error while reading config from env")
		os.Exit(1)
	}
	if err := config.Validate(); err != nil {
		setupLog.Error(err, "unable to start the manager")
		os.Exit(1)
	}

	if err := helper.ParseCRD(crdBytes); err != nil {
		setupLog.Error(err, "unable to parse CRD")
		os.Exit(1)
	}

	disableHTTP2 := func(c *tls.Config) {
		if enableHTTP2 {
			setupLog.Info("Warning: http/2 is enabled")
			return
		}
		c.NextProtos = []string{"http/1.1"}
	}

	var metricsCertWatcher *certwatcher.CertWatcher
	cfg := ctrl.GetConfigOrDie()
	metricsOptions := server.Options{
		BindAddress:    metricsAddr,
		TLSOpts:        []func(*tls.Config){disableHTTP2},
		FilterProvider: filters.WithAuthenticationAndAuthorization,
	}
	if len(metricsCertFile) > 0 && len(metricsCertKeyFile) > 0 {
		metricsOptions.SecureServing = true
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-file", metricsCertFile, "metrics-cert-key-file", metricsCertKeyFile)

		metricsCertWatcher, err = certwatcher.New(metricsCertFile, metricsCertKeyFile)
		if err != nil {
			setupLog.Error(err, "Failed to initialize metrics certificate watcher", "error", err)
			os.Exit(1)
		}

		metricsOptions.TLSOpts = append(metricsOptions.TLSOpts, func(config *tls.Config) {
			config.GetCertificate = metricsCertWatcher.GetCertificate
		})
	} else {
		setupLog.Info("Warning: metrics server does not use TLS")
	}

	// Create Manager (will fetch TLS profile and configure servers internally)
	mgr, err := manager.NewManager(context.Background(), cfg, config, &ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsOptions,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port:    constants.WebhookPort,
			TLSOpts: []func(*tls.Config){disableHTTP2},
		}),
		PprofBindAddress:       pprofAddr,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "7a7ecdcd.netobserv.io",
	}, controllers.Registerers)
	if err != nil {
		setupLog.Error(err, "unable to setup manager")
		os.Exit(1)
	}

	if err = (&flowsv1beta2.FlowCollector{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create v1beta2 webhook", "webhook", "FlowCollector")
		os.Exit(1)
	}
	if err = (&metricsv1alpha1.FlowMetricWebhook{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "FlowMetric")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if metricsCertWatcher != nil {
		if err := mgr.Add(metricsCertWatcher); err != nil {
			setupLog.Error(err, "unable to add metrics certificate watcher to manager")
			os.Exit(1)
		}
	}

	ctx, stop := context.WithCancel(ctrl.SetupSignalHandler())

	if err := setupTLSProfileWatcher(mgr, stop); err != nil {
		setupLog.Error(err, "unable to setup TLS profile watcher")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func defaultStringEnv(env, def string) string {
	if v := os.Getenv(env); v != "" {
		return v
	}
	return def
}

func maybeBoolEnv(env string) (*bool, error) {
	if v := os.Getenv(env); v != "" {
		// Use ParseBool to allow common variants ("true", "True", "1"...) and ignore non-bools
		b, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid %s value %q: %w", env, v, err)
		}
		return ptr.To(b), nil
	}
	return nil, nil
}

// setupTLSProfileWatcher configures the SecurityProfileWatcher on OpenShift clusters.
// On OpenShift with no explicit TLS profile, OCP uses the Intermediate default; we still
// set up the watcher so that a future explicit profile change is detected and triggers a restart.
func setupTLSProfileWatcher(mgr *manager.Manager, stop context.CancelFunc) error {
	if !mgr.ClusterInfo.IsOpenShift() {
		return nil
	}
	tlsProfileSpec := mgr.ClusterInfo.GetTLSProfileSpec()
	if tlsProfileSpec == nil {
		// No explicit profile — use Intermediate (OCP's implicit default) as the baseline
		tlsProfileSpec = configv1.TLSProfiles[configv1.TLSProfileIntermediateType]
	}
	if tlsProfileSpec == nil {
		return nil
	}
	setupLog.Info("Setting up TLS profile watcher for graceful restart on profile changes")
	// Known self-healing behavior: on a TLS profile change the operator reloads by exiting 0
	// (graceful) so it restarts with the new profile. The baseline profile compared against is
	// re-captured on every container start (InitialTLSProfileSpec above). During a control-plane
	// rollout — especially two overlapping profile changes — a freshly started container can read
	// a stale/lagging APIServer value as its baseline and is then forced to reload once the real
	// value settles. Because this repeats per restart within the rollout window, several graceful
	// (exit 0) reloads can cluster together, and kubelet flags the clustered restarts as
	// CrashLoopBackOff regardless of exit code. This is cosmetic and self-heals: each reload is a
	// correct reaction (the container really had the wrong profile), and once the rollout settles
	// the operator comes up stable.
	return (&tlspkg.SecurityProfileWatcher{
		Client:                mgr.GetClient(),
		InitialTLSProfileSpec: *tlsProfileSpec,
		OnProfileChange: func(_ context.Context, oldSpec, newSpec configv1.TLSProfileSpec) {
			setupLog.Info("TLS profile has changed, initiating graceful shutdown to reload",
				"oldProfile", oldSpec, "newProfile", newSpec)
			stop()
		},
	}).SetupWithManager(mgr)
}

func readConfigFromEnv() (*manager.Config, error) {
	deployNetpol, err := maybeBoolEnv("OPERATOR_NETWORK_POLICY")
	if err != nil {
		return nil, err
	}
	return &manager.Config{
		Vendor:                constants.Vendor(os.Getenv("VENDOR")),
		EBPFAgentImage:        defaultStringEnv("RELATED_IMAGE_EBPF_AGENT", "quay.io/netobserv/netobserv-ebpf-agent:main"),
		EBPFByteCodeImage:     defaultStringEnv("RELATED_IMAGE_EBPF_BYTECODE", "quay.io/netobserv/ebpf-bytecode:main"), // TODO: productize for GA
		FlowlogsPipelineImage: defaultStringEnv("RELATED_IMAGE_FLOWLOGS_PIPELINE", "quay.io/netobserv/flowlogs-pipeline:main"),
		WebConsoleImage:       defaultStringEnv("RELATED_IMAGE_WEB_CONSOLE", "quay.io/netobserv/network-observability-console-plugin:main"),
		WebConsolePF4Image:    defaultStringEnv("RELATED_IMAGE_WEB_CONSOLE_PF4", "quay.io/netobserv/network-observability-console-plugin:main-pf4"),
		WebConsolePF5Image:    defaultStringEnv("RELATED_IMAGE_WEB_CONSOLE_PF5", "quay.io/netobserv/network-observability-console-plugin:main-pf5"),
		DemoLokiImage:         defaultStringEnv("RELATED_IMAGE_DEMO_LOKI", "grafana/loki:3.5.0"),
		Namespace:             defaultStringEnv("NAMESPACE", "netobserv"),
		StaticPluginConfig: manager.StaticPluginConfig{
			InheritTolerationFromSubscription: os.Getenv("STATIC_PLUGIN_INHERIT_TOLERATION_SUBSCRIPTION"),
			CPURequest:                        defaultStringEnv("STATIC_PLUGIN_CPU_REQUEST", "10m"),
			MemoryRequest:                     defaultStringEnv("STATIC_PLUGIN_MEMORY_REQUEST", "64Mi"),
			CPULimit:                          defaultStringEnv("STATIC_PLUGIN_CPU_LIMIT", ""),
			MemoryLimit:                       defaultStringEnv("STATIC_PLUGIN_MEMORY_LIMIT", ""),
		},
		DeployOperatorNetworkPolicy: deployNetpol,
	}, nil
}
