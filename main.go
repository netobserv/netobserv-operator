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

	bpfmaniov1alpha1 "github.com/bpfman/bpfman-operator/apis/v1alpha1"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	osv1 "github.com/openshift/api/console/v1"
	operatorsv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	flowsv1beta2 "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	slicesv1alpha1 "github.com/netobserv/network-observability-operator/api/flowcollectorslice/v1alpha1"
	metricsv1alpha1 "github.com/netobserv/network-observability-operator/api/flowmetrics/v1alpha1"
	controllers "github.com/netobserv/network-observability-operator/internal/controller"
	"github.com/netobserv/network-observability-operator/internal/controller/constants"
	"github.com/netobserv/network-observability-operator/internal/pkg/helper"
	"github.com/netobserv/network-observability-operator/internal/pkg/manager"
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

	config := manager.Config{}

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&metricsCertFile, "metrics-cert-file", "", "The path to the TLS certificate for metrics.")
	flag.StringVar(&metricsCertKeyFile, "metrics-cert-key-file", "", "The path to the TLS certificate key for metrics.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&pprofAddr, "profiling-bind-address", "", "The address the profiling endpoint binds to, such as ':6060'. Leave unset to disable profiling.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&config.EBPFAgentImage, "ebpf-agent-image", "quay.io/netobserv/netobserv-ebpf-agent:main", "The image of the eBPF agent")
	flag.StringVar(&config.FlowlogsPipelineImage, "flowlogs-pipeline-image", "quay.io/netobserv/flowlogs-pipeline:main", "The image of Flowlogs Pipeline")
	flag.StringVar(&config.ConsolePluginImage, "console-plugin-image", "quay.io/netobserv/network-observability-console-plugin:main", "The image of the Console Plugin")
	flag.StringVar(&config.ConsolePluginCompatImage, "console-plugin-compat-image", "quay.io/netobserv/network-observability-console-plugin-pf4:main", "A backward compatible image of the Console Plugin (e.g. Patterfly 4 variant)")
	flag.StringVar(&config.EBPFByteCodeImage, "ebpf-bytecode-image", "quay.io/netobserv/ebpf-bytecode:main", "The EBPF bytecode for the eBPF agent")
	flag.StringVar(&config.Namespace, "namespace", "netobserv", "Current controller namespace")
	flag.StringVar(&config.DemoLokiImage, "demo-loki-image", "grafana/loki:3.5.0", "The image of the zero click loki deployment")
	flag.BoolVar(&config.DownstreamDeployment, "downstream-deployment", false, "Either this deployment is a downstream deployment ot not")
	flag.BoolVar(&config.Hold, "hold", false, "Hold mode: delete all operator-controlled resources while keeping CRDs (FlowCollector, FlowCollectorSlice, FlowMetric) and namespaces")
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

		var err error
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

	mgr, err := manager.NewManager(context.Background(), cfg, &config, &ctrl.Options{
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

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
