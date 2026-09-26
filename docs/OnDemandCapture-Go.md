# OnDemandCapture with the Go CLI

The controller launches `/oc-netobserv` from `RELATED_IMAGE_NETOBSERV_CLI`.
The configured development image is
`quay.io/rh-ee-kapjain/network-observability-cli@sha256:e726e5452ceffecacbf1ed38351f2ae3d0088671e832d12401ea8040c2bcdda0`
(currently linux/amd64). Helm uses `netobservCLI.digest` when nonempty and otherwise
uses `netobservCLI.version`.
Use an image matching your cluster architecture. The image must support
`--headless` and `serve`. Captures use the agent image selected when the CLI
image was built. Set `RELATED_IMAGE_NETOBSERV_CLI_AGENT` on the operator to
override it with a compatible agent; the FlowCollector agent setting is separate. No `oc` or `kubectl` executable is needed in the pod.

This is an administrator-only API. Create the CR in the operator's namespace
(the namespace configured through `NAMESPACE`, commonly
`openshift-netobserv-operator` on OpenShift). Requests in other namespaces fail
without receiving capture access. Existing captures outside that namespace have
their cluster binding revoked and their launcher/resources cleaned up.
Do not delegate CR creation, pod creation, exec, or port-forward access in the
operator namespace to tenants: captures intentionally have cluster-wide privileges.
The operator does not infer the requesting user's identity from the CR.

Each CR gets its own service account, launcher pod, cluster binding, and a capture
namespace named `netobserv-<CR-name>-<UID-without-hyphens>`. The CR name is
shortened when needed to fit the 63-character namespace limit; the full unique
UID suffix is retained. The same CR reuses the same namespace across retries
and operator restarts, while different CR instances receive different namespaces. The Go CLI uses the projected service-account
credentials through client-go. Automatic service-account-token mounting is
disabled on both the service account and pod. A short-lived projected token, CA
bundle and namespace file are mounted only into the `cli` container; the `files`
container has no mounted Kubernetes credentials. It deploys the temporary agents and collector,
waits for collection, copies flow/packet output into the shared `/output` volume,
and cleans up its capture namespace. A separate `files` container serves the
output on `127.0.0.1:8080` until the CR is deleted. Download through an
authenticated Kubernetes port-forward (`pods/portforward` authorization is
required). Use `status.podName` and `status.podNamespace` as the forwarding target,
forward local port 8080 to pod port 8080, and open `http://127.0.0.1:8080/` locally.
The Go client example in `internal/controller/ondemandcapture_live_test.go` uses
client-go port forwarding and HTTP downloads without an external CLI. Direct pod-IP access and Kubernetes pod proxy are not supported.

`Completed` means the CLI container exited successfully, including output copying
and cleanup. The file server stays running. A nonzero CLI exit sets `Failed` and
records the exit details. Deleting the CR stops the launcher first, removes any
remaining capture namespace and binding, and then allows garbage collection of
the pod and service account. Deleting the CR also deletes its captured files.

Example:

```yaml
apiVersion: netobserv.io/v1beta1
kind: OnDemandCapture
metadata:
  name: inspect-flows
  namespace: openshift-netobserv-operator # Replace with your operator namespace
spec:
  captureType: flows
  duration: 1m
  maxBytes: 500000
  flowConfig:
    sampling: 1
    enableDNS: true
    enableRTT: true
```

Set `spec.filters.drops: true` for the equivalent of
`oc-netobserv flows --drops`: only dropped traffic is collected, and packet-drop
tracking is enabled automatically. `enablePacketDrops` alone adds drop details
without filtering out other traffic.

Flow and metrics feature switches, flow sampling/interfaces, capture limits, and
node selectors map to the CLI flags. Packet capture requires at least one traffic filter, supplied through
`filters`, `args`, or `packetConfig`; `packetConfig.port` maps to the
destination-port filter.
Namespace/pod-label filters are not implemented by this backend and produce a
validation error, rather than silently collecting unfiltered traffic. Packet
interface selection and metrics byte limits are also rejected because the CLI
does not support those combinations. Metrics remain in Prometheus; they do not
produce downloadable flow/packet files.

The generated capture role grants the CLI resource-management and capture rights
across namespaces. Restrict permission to create OnDemandCapture CRs to users
allowed to perform cluster network captures.

## Validation

`go test -mod=vendor ./internal/controller -run TestOnDemandCaptureReconciler`
checks CR-to-argument mappings, pod wiring, failure/completion status, idempotency,
and finalization using the controller-runtime fake client.

The opt-in live test runs the controller-generated pod with its RBAC without
installing or replacing an operator:

```sh
NETOBSERV_ODC_LIVE_KUBECONFIG=/path/to/kubeconfig \
NETOBSERV_ODC_LIVE_IMAGE=quay.io/rh-ee-kapjain/network-observability-cli:main \
go test -mod=vendor ./internal/controller -run '^TestOnDemandCaptureLivePod$' -v -timeout=12m
```

## CLI filters and arguments

`spec.filters` accepts the CLI filter names directly: `direction`, `cidr`,
`protocol`, `sport`, `dport`, `port`, `sport_range`, `dport_range`, `port_range`,
`sports`, `dports`, `ports`, `icmp_type`, `icmp_code`, `peer_ip`, `peer_cidr`,
`action`, `tcp_flags`, and `drops`. Port and ICMP fields are numbers (zero is
preserved); ranges and comma-separated lists use CLI string syntax. Add
alternative groups under `filters.or`.

`spec.args` is an ordered array of native CLI arguments. It supports all traffic
filters, `or`, capture limits, sampling, node selectors, queries, interfaces,
excluded interfaces, log level, metrics include lists, subnet discovery,
privileged mode, and all CLI feature switches. It follows structured settings,
so the CLI's normal last-value/additive rules apply. Values are passed as argv
entries without shell evaluation. The CLI validates values and combinations for
the selected capture type; invalid combinations result in a failed capture.

The operator manages `namespace`, `kubeconfig`, `context`, `output-dir`,
`headless`, `background`, `yaml`, and `copy`; these cannot be overridden.
Help/version requests are not capture arguments. This preserves in-cluster
credentials, isolated resources, output downloading, and completion tracking.

```yaml
apiVersion: netobserv.io/v1beta1
kind: OnDemandCapture
metadata:
  name: dropped-traffic
  namespace: openshift-netobserv-operator # Replace with your operator namespace
spec:
  captureType: flows
  duration: 60s
  filters:
    drops: true
  args:
    - --sampling=1
    - --log-level=info
```

Example alternatives: `filters: {protocol: TCP, dport: 443, or: [{protocol: UDP,
dport: 53}]}`. The equivalent ordered arguments are `['--protocol=TCP',
'--dport=443', 'or', '--protocol=UDP', '--dport=53']`.

## Automatic retention cleanup

Completed and failed captures are deleted automatically **24 hours after
`status.completionTime`** by default. Set `spec.ttlSecondsAfterFinished` to a
nonnegative number of seconds to change retention, for example `3600` for one
hour. Zero means immediate deletion and leaves no download window.

Cleanup deletes the CR, launcher/file-server pod, temporary capture namespace,
and capture access resources. **Undownloaded files in the pod are lost**; download
or archive them before retention expires. Local downloads and metrics already
stored in Prometheus are unaffected. Retention does not limit active captures.
Operator restarts preserve the deadline. Existing completed or failed CRs also
use this policy; older resources without a completion time receive a fresh
retention window when first reconciled. Extend retention before expiry to keep
outputs longer; changing it after deletion begins cannot cancel cleanup.
