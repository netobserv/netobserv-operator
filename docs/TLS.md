# TLS and expected certificates

This document lists all required and optional TLS certificates for NetObserv. You can also refer to the [Helm chart templates](../helm/templates/certificates.yaml) for cert-manager.

## Required certificates

Those certificates are always required and are not configurable:

<table>
  <thead>
    <tr>
      <th>Service name</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>netobserv-webhook-service</td>
      <td>Secret</td>
      <td>webhook-server-cert</td>
      <td>tls.crt, tls.key</td>
    </tr>
    <tr>
      <td>netobserv-metrics-service</td>
      <td>Secret</td>
      <td>manager-metrics-tls</td>
      <td>tls.crt, tls.key</td>
    </tr>
  </tbody>
</table>

## Agent to FLP certificates

When `spec.deploymentModel` is "Service", the traffic from eBPF agents to flowlogs-pipeline pods uses TLS by default. It is possible to disable TLS, though not recommended in production-grade environments, as it decreases the security of the NetObserv deployments.

In "Kafka" mode, the TLS/SASL configuration depends on your installation. The Kafka clients used in NetObserv support simple TLS, mTLS, SASL as well as no TLS. We recommend the use of mTLS for higher security standards.

In "Direct" mode, the traffic doesn't leave the host and is not encrypted.

The tables below apply to the "Service" mode.

### Auto (TLS)

When `spec.processor.service.tlsType` is "Auto":

<table>
  <thead>
    <tr>
      <th>Needed by</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-cert</td>
      <td>tls.crt, tls.key</td>
      <td></td>
    </tr>
    <tr>
      <td>eBPF Agents</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td>Must be installed in netobserv-privileged namespace.</td>
    </tr>
  </tbody>
</table>

### Auto (mTLS)

When `spec.processor.service.tlsType` is "Auto-mTLS":

<table>
  <thead>
    <tr>
      <th>Needed by</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-cert</td>
      <td>tls.crt, tls.key</td>
      <td></td>
    </tr>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td></td>
    </tr>
    <tr>
      <td>eBPF Agents</td>
      <td>Secret</td>
      <td>ebpf-agent-cert</td>
      <td>tls.crt, tls.key</td>
      <td>Must be installed in netobserv-privileged namespace.</td>
    </tr>
    <tr>
      <td>eBPF Agents</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td>Must be installed in netobserv-privileged namespace.</td>
    </tr>
  </tbody>
</table>

### Provided

When `spec.processor.service.tlsType` is "Provided", you can specify any Secret or ConfigMap for TLS or mTLS, via `spec.processor.service.providedCertificates`.

For mTLS, configure `spec.processor.service.providedCertificates.clientCert`. For simple TLS, do not set the client cert config.

## Informers to Processor certificates

When `spec.processor.informers.enabled` is `true`, informers communicate with processors via gRPC on port 9090 (k8scache). TLS can be configured for this communication.

### Auto (TLS)

When `spec.processor.informers.tls.type` is "Auto":

<table>
  <thead>
    <tr>
      <th>Needed by</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-cert</td>
      <td>tls.crt, tls.key</td>
      <td>Reuses the same certificate as the main gRPC service.</td>
    </tr>
    <tr>
      <td>flowlogs-pipeline-informers</td>
      <td>ConfigMap</td>
      <td>openshift-service-ca.crt</td>
      <td>service-ca.crt</td>
      <td></td>
    </tr>
  </tbody>
</table>

### Auto (mTLS)

When `spec.processor.informers.tls.type` is "Auto-mTLS":

<table>
  <thead>
    <tr>
      <th>Needed by</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-cert</td>
      <td>tls.crt, tls.key</td>
      <td>Reuses the same certificate as the main gRPC service.</td>
    </tr>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td></td>
    </tr>
    <tr>
      <td>flowlogs-pipeline-informers</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-informers-k8scache-client-cert</td>
      <td>tls.crt, tls.key</td>
      <td></td>
    </tr>
    <tr>
      <td>flowlogs-pipeline-informers</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td></td>
    </tr>
  </tbody>
</table>

### Provided

When `spec.processor.informers.tls.type` is "Provided", you can specify any Secret or ConfigMap for TLS or mTLS, via `spec.processor.informers.tls.providedCertificates`.

For mTLS, configure both `serverCert` and `clientCert`. For simple TLS, only configure `serverCert`.

## Scan installed TLS endpoints

Build and push an [OpenShift tls-scanner](https://github.com/openshift/tls-scanner) image to a registry that the cluster can pull from. The image target clones a pinned upstream revision into `out/tls-scanner-source/` on its first run, builds its `Dockerfile.local` directly with Docker or Podman, and pushes it to Quay by default:

```sh
make tls-scanner-image USER=my-quay-namespace VERSION=dev
make tls-scanner USER=my-quay-namespace VERSION=dev
```

The default image is `quay.io/$(USER)/tls-scanner:$(VERSION)`, with `VERSION=main`. `USER` selects your registry namespace through the existing `IMAGE_ORG` and `REPO` defaults. `VERSION` controls the image tag independently of the upstream source revision.

Set `TLS_SCANNER_REF` to select another upstream revision that provides `Dockerfile.local`. Each build checks out that revision in the managed `out/tls-scanner-source/` directory, fetching it if needed; local changes in that directory stop the build. Set `TLS_SCANNER_SOURCE` to build an existing local source directory instead: its contents, including `Dockerfile.local`, are used as-is, without fetching or checking out a revision. The image tag still follows `VERSION`.

Set `TLS_SCANNER_IMAGE` to use another image name or tag, or `TLS_SCANNER_PLATFORM` to select an architecture (default `linux/$(GOARCH)`, with `GOARCH=amd64`). The image build and push are also available separately as `tls-scanner-image-build` and `tls-scanner-image-push`. Authenticate to the registry with Docker or Podman before pushing. To run with an image that is already available, use:

```sh
make tls-scanner TLS_SCANNER_IMAGE=registry.example.com/team/tls-scanner:tag
```

The target scans `netobserv`, `netobserv-privileged`, and the operator namespace twice: once to check TLS 1.3 and ML-KEM support, and once to check adherence to the cluster TLS profile. Either check can fail the target. It writes `report.md`, JSON, CSV, JUnit, and logs in a new run directory under `out/tls-scanner/`; the adherence artifacts are in its `adherence/` subdirectory. It also copies the latest results to `out/tls-scanner/`. The target then removes its temporary Job, ServiceAccount, RBAC resources, and NetworkPolicies. The temporary NetworkPolicies allow only the scanner Job to reach the operator and NetObserv operands in the scanned namespaces, avoiding false `NO_TLS` results from normal ingress restrictions when the Job runs in a separate namespace. The target needs permission to create NetworkPolicies and cluster RBAC, and to grant the temporary ServiceAccount use of the privileged SCC for port discovery. It does not change NetObserv workloads.

For a different installation, set `TLS_SCANNER_NAMESPACE` to the namespace where the Job should run and `TLS_SCANNER_OPERATOR_NAMESPACE` to the namespace containing the operator. The operator namespace defaults to the existing `OPERATOR_NS` setting (`NAMESPACE` normally, or `openshift-netobserv-operator` when `CSV` is set). The default `TLS_SCANNER_NAMESPACES` includes the operator namespace; override it with a comma-separated list to scan a different set of namespaces. `TLS_SCANNER_OUTPUT_DIR`, `TLS_SCANNER_PARALLEL`, and `TLS_SCANNER_TIMEOUT_SECONDS` are also configurable.

If a scan fails before producing all its artifacts, the report lists the missing files and the workflow copies the available artifacts while preserving the scan's exit status.

Review `NO_TLS`, `FILTERED`, and `NO_PORTS` rows in the report. Network policies or pod security restrictions can keep the scanner from reaching a TLS endpoint, so a passing PQC result only covers endpoints it scanned successfully.
