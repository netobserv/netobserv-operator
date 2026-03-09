# NetObserv Operator

NetObserv Operator is a Kubernetes operator for network observability. It deploys a monitoring pipeline that consists in:
- An eBPF agent, that generates network flows from captured packets.
- Flowlogs-pipeline, a component that collects, enriches and exports these flows.
- A web console for flows visualization with powerful filtering options, a topology representation, a network health view, etc.

Flow data is then available in multiple ways, each optional:

- As Prometheus metrics.
- As raw flow logs stored in Loki.
- As raw flow logs exported to a collector via Kafka, OpenTelemetry or IPFIX.

## Getting Started

You can install the NetObserv Operator using [Helm](https://helm.sh/), or directly from sources.

> [!TIP]
NetObserv can be used in downstream products, which may provide their own documentation. If you are using such a product, please refer to that documentation instead:
> 
> - On OpenShift: [see Network Observability operator](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/installing-network-observability-operators).

### Pre-requisite

The following architectures are supported: _amd64_, _arm64_, _ppc64le_ and _s390x_.

NetObserv has a couple of dependencies that must be installed on your cluster:

- Cert-manager
- Prometheus
- Loki

Cert-manager has to be installed separately. For example, using helm:

```bash
helm repo add cert-manager https://charts.jetstack.io
helm install my-cert-manager cert-manager/cert-manager --set crds.enabled=true
```

Prometheus and Loki can be installed separately, or as dependencies of NetObserv (see below).

Loki is not mandatory but improves the overall experience with NetObserv.

### Install with Helm

> [!TIP]
> See it also on [ArtifactHub](https://artifacthub.io/packages/helm/netobserv/netobserv-operator).

```bash
helm repo add netobserv https://netobserv.io/static/helm/ --force-update

# Standalone install, including dependencies:
helm install my-netobserv -n netobserv --create-namespace --set install.loki=true --set install.prom-stack=true netobserv/netobserv-operator

# OR minimal install (Prometheus/Loki must be installed separately)
helm install my-netobserv -n netobserv --create-namespace netobserv/netobserv-operator
```

You can now create a `FlowCollector` resource ([full API reference](https://github.com/netobserv/netobserv-operator/blob/main/docs/FlowCollector.md#flowsnetobserviov1beta2)). A short `FlowCollector` should work, using most default values, plus with the standalone console enabled:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: flows.netobserv.io/v1beta2
kind: FlowCollector
metadata:
  name: cluster
spec:
  namespace: netobserv
  consolePlugin:
    standalone: true
  processor:
    advanced:
      env:
        SERVER_NOTLS: "true"
  loki:
    mode: Monolithic
    monolithic:
      url: 'http://my-netobserv-loki.netobserv.svc.cluster.local.:3100/'
  prometheus:
    querier:
      mode: Manual
      manual:
        url: http://my-netobserv-kube-promethe-prometheus.netobserv.svc.cluster.local.:9090/
        alertManager:
          url: http://my-netobserv-kube-promethe-alertmanager.netobserv.svc.cluster.local.:9093/
EOF
```

A few remarks:
- You can change the Prometheus and Loki URLs depending on your installation. This example works if you use the "standalone" installation described above, with `install.loki=true` and `install.prom-stack=true`. Check more configuration options for [Prometheus](https://github.com/netobserv/netobserv-operator/blob/main/docs/FlowCollector.md#flowcollectorspecprometheus-1) and [Loki](https://github.com/netobserv/netobserv-operator/blob/main/docs/FlowCollector.md#flowcollectorspecloki-1).
- Depending on the Kubernetes distribution and CNI, NetObserv may come secured by default with a built-in network policy. You can force installing it or not by setting `spec.networkPolicy.enable` in `FlowCollector`. If the built-in policy does not work as intended, it is recommended to turn it off and create your own instead. NetObserv runs some highly privileged workloads, thus it is important to keep it as much isolated as possible. See [NetworkPolicy.md](https://github.com/netobserv/netobserv-operator/blob/main/docs/NetworkPolicy.md) for more details on how to create a policy.
- The processor env `SERVER_NOTLS` means that the communication between eBPF agents and Flowlogs-pipeline won't be encrypted. To enable TLS, you need to supply the TLS certificates to Flowlogs-pipeline (a Secret named `flowlogs-pipeline-cert`), and the CA to the eBPF agents (a ConfigMap named `flowlogs-pipeline-ca` in the privileged namespace).

To view the test console, you can port-forward 9001:

```bash
kubectl port-forward svc/netobserv-plugin 9001:9001 -n netobserv
```

Then open http://localhost:9001/ in your browser.

### More information

Refer to the GitHub repository README for more information: https://github.com/netobserv/netobserv-operator/blob/main/README.md
