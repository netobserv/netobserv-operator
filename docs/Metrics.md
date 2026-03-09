# Metrics in the NetObserv Operator

The NetObserv operator uses [flowlogs-pipeline](https://github.com/netobserv/flowlogs-pipeline/) to generate metrics out of flow logs.
These metrics are meant to be collected by a Prometheus instance (not part of NetObserv deployment). In OpenShift, they are collected either by Cluster Monitoring or User Workload Monitoring.

There are two ways to configure metrics:

- By enabling or disabling any of the predefined metrics
- Using the FlowMetrics API to create custom metrics

For alerts and health documentation, see [HealthRules.md](./HealthRules.md).

## Predefined metrics

They can be configured in the `FlowCollector` custom resource, via `spec.processor.metrics.includeList`. It is a list of metric names that tells which ones to generate.

The names correspond to the names in Prometheus without their prefix. For example, `namespace_egress_packets_total` will show up as `netobserv_namespace_egress_packets_total` in Prometheus.

Note that the more metrics you add, the bigger is the impact on Prometheus workload resources. Some metrics in particular have a bigger cardinality, such as all metrics starting with `workload_`, which may result in stressing Prometheus if too many of them are enabled. It is recommended to monitor the impact on Prometheus when adding more metrics.

Available names are: (names followed by "*" are enabled by default; names followed by "**" are also enabled by default when Loki is disabled, with workload-based metrics replacing their namespace-based counterpart, e.g. `workload_flows_total` replaces `namespace_flows_total`)

- `namespace_egress_bytes_total`
- `namespace_egress_packets_total`
- `namespace_ingress_bytes_total`
- `namespace_ingress_packets_total` *
- `namespace_flows_total` *
- `node_egress_bytes_total` *
- `node_egress_packets_total`
- `node_ingress_bytes_total` *
- `node_ingress_packets_total` *
- `node_flows_total`
- `workload_egress_bytes_total` *
- `workload_egress_packets_total` **
- `workload_ingress_bytes_total` *
- `workload_ingress_packets_total` **
- `workload_flows_total` **
- `workload_sampling`
- `node_to_node_ingress_flows_total` *

When the `PacketDrop` feature is enabled in `spec.agent.ebpf.features` (with privileged mode), additional metrics are available:
- `namespace_drop_bytes_total`
- `namespace_drop_packets_total` *
- `node_drop_bytes_total`
- `node_drop_packets_total` *
- `workload_drop_bytes_total` **
- `workload_drop_packets_total` **

When the `FlowRTT` feature is enabled in `spec.agent.ebpf.features`, additional metrics are available:
- `namespace_rtt_seconds` *
- `node_rtt_seconds`
- `workload_rtt_seconds` **

When the `DNSTracking` feature is enabled in `spec.agent.ebpf.features`, additional metrics are available:
- `namespace_dns_latency_seconds` *
- `node_dns_latency_seconds`
- `workload_dns_latency_seconds` **

When the `NetworkEvents` feature is enabled in `spec.agent.ebpf.features`,
- `namespace_network_policy_events_total` *
- `node_network_policy_events_total`
- `workload_network_policy_events_total`

When the `IPSec` feature is enabled in `spec.agent.ebpf.features`,
- `node_ipsec_flows_total` *

## Custom metrics using the FlowMetrics API

The FlowMetrics API ([spec reference](./FlowMetric.md)) has been designed to give you full control on the metrics generation out of the NetObserv' enriched NetFlow data.
It allows to create counters or histograms with any set of fields as Prometheus labels, and using any filters from the fields. Just a recommendation: be careful about the [metrics cardinality](https://www.robustperception.io/cardinality-is-key/) when creating new metrics. High cardinality metrics can stress the Prometheus instance.

The full list of fields is [available there](./flows-format.adoc). The "Cardinality" column gives information about the implied metrics cardinality. Fields flagged as `fine` are safe to use as labels. Fields flagged as `careful` need some extra attention: if you want to use them as labels, it is recommended to narrow down the cardinality with filters. For example, you may safely use `DstPort` as a label if you also restrict which `DstPort` are allowed with a `MatchRegex` filter.

Be also aware that for each field used as a label, the fields cardinality is potentially multiplied - and this is especially true when mixing Source and Destination fields. For instance, using `SrcK8S_Name` or `DstK8S_Name` (ie. Pod/Node/Service names) alone as a label might be reasonable, but using both `SrcK8S_Name` and `DstK8S_Name` in the same metric potentially generates the square of the cardinality of Pods/Nodes/Services.

Don't hesitate to [reach out](https://github.com/netobserv/netobserv-operator/discussions/new/choose) if you need more guidance.

Some of those fields require special features to be enabled in `FlowCollector`, such as `TimeFlowRttNs` via `spec.agent.ebpf.features` or `Src/DstK8S_Zone` via `spec.processor.addZone`.

Currently, `FlowMetric` resources need to be created in the namespace defined in `FlowCollector` `spec.namespace`, which is `netobserv` by default. This may change in the future.

### Counter example

Here is an example of a FlowMetric resource that generates a metric tracking ingress bytes received from cluster external sources, labeled by destination host and workload:

```yaml
apiVersion: flows.netobserv.io/v1alpha1
kind: FlowMetric
metadata:
  name: flowmetric-cluster-external-ingress-traffic
spec:
  metricName: cluster_external_ingress_bytes_total
  type: Counter
  valueField: Bytes
  direction: Ingress
  labels: [DstK8S_HostName,DstK8S_Namespace,DstK8S_OwnerName,DstK8S_OwnerType]
  filters:
  - field: SrcSubnetLabel
    matchType: Absence
```

In this example, selecting just the cluster external traffic is done by matching only flows where `SrcSubnetLabel` is absent. This assumes the subnet labels feature is enabled (via `spec.processor.subnetLabels`) and configured to recognize IP ranges used in the cluster. In OpenShift, this is enabled and configured by default.

Refer to the [spec reference](./FlowMetric.md) for more information about each field.

### Histogram example

Here is a similar example for an histogram. Histograms are typically used for latencies. This example shows RTT latency for cluster external ingress traffic.

```yaml
apiVersion: flows.netobserv.io/v1alpha1
kind: FlowMetric
metadata:
  name: flowmetric-cluster-external-ingress-rtt
spec:
  metricName: cluster_external_ingress_rtt_seconds
  type: Histogram
  valueField: TimeFlowRttNs
  direction: Ingress
  labels: [DstK8S_HostName,DstK8S_Namespace,DstK8S_OwnerName,DstK8S_OwnerType]
  filters:
  - field: SrcSubnetLabel
    matchType: Absence
  - field: TimeFlowRttNs
    matchType: Presence
  divider: "1000000000"
  buckets: [".001", ".005", ".01", ".02", ".03", ".04", ".05", ".075", ".1", ".25", "1"]
```

`type` here is `Histogram` since it looks for a latency value (`TimeFlowRttNs`),
and we define custom buckets that should offer a decent precision on RTT ranging roughly between 5ms and 250ms.
Since the RTT is provided as nanos in flows, we use a divider of 1 billion to convert into seconds (which is standard in Prometheus guidelines).

### More examples

You can find more examples in https://github.com/netobserv/netobserv-operator/tree/main/config/samples/flowmetrics.

### Charts (OpenShift only)

Optionally, you can generate charts for dashboards in the OpenShift Console (administrator view, Dashboard menu), by filling the `charts` section of the `FlowMetric` resources.

Here is an example for the `flowmetric-cluster-external-ingress-traffic` resource described above:

```yaml
# ...
  charts:
  - dashboardName: Main
    title: External ingress traffic
    unit: Bps
    type: SingleStat
    queries:
    - promQL: "sum(rate($METRIC[2m]))"
      legend: ""
  - dashboardName: Main
    sectionName: External
    title: Top external ingress traffic per workload
    unit: Bps
    type: StackArea
    queries:
    - promQL: "sum(rate($METRIC{DstK8S_Namespace!=\"\"}[2m])) by (DstK8S_Namespace, DstK8S_OwnerName)"
      legend: "{{DstK8S_Namespace}} / {{DstK8S_OwnerName}}"
```

This creates two panels:
- a textual "single stat" that shows global external ingress rate summed across all dimensions
- a timeseries graph showing the same metric per destination workload

For more information about the query language, refer to the [Prometheus documentation](https://prometheus.io/docs/prometheus/latest/querying/basics/).
And again, refer to the [spec reference](./FlowMetric.md) for more information about each field.

Another example for histograms:

```yaml
# ...
  charts:
  - dashboardName: Main
    title: External ingress TCP latency
    unit: seconds
    type: SingleStat
    queries:
    - promQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket[2m])) by (le)) > 0"
      legend: "p99"
  - dashboardName: Main
    sectionName: External
    title: "Top external ingress sRTT per workload, p50 (ms)"
    unit: seconds
    type: Line
    queries:
    - promQL: "histogram_quantile(0.5, sum(rate($METRIC_bucket{DstK8S_Namespace!=\"\"}[2m])) by (le,DstK8S_Namespace,DstK8S_OwnerName))*1000 > 0"
      legend: "{{DstK8S_Namespace}} / {{DstK8S_OwnerName}}"
  - dashboardName: Main
    sectionName: External
    title: "Top external ingress sRTT per workload, p99 (ms)"
    unit: seconds
    type: Line
    queries:
    - promQL: "histogram_quantile(0.99, sum(rate($METRIC_bucket{DstK8S_Namespace!=\"\"}[2m])) by (le,DstK8S_Namespace,DstK8S_OwnerName))*1000 > 0"
      legend: "{{DstK8S_Namespace}} / {{DstK8S_OwnerName}}"
```

This example uses the `histogram_quantile` function, to show p50 and p99.
You may also be interested in showing averages of histograms: this is done by dividing `$METRIC_sum` by `$METRIC_count` metrics, which are automatically generated when you create an histogram. With the above example, it would be:

```yaml
promQL: "(sum(rate($METRIC_sum{DstK8S_Namespace!=\"\"}[2m])) by (DstK8S_Namespace,DstK8S_OwnerName) / sum(rate($METRIC_count{DstK8S_Namespace!=\"\"}[2m])) by (DstK8S_Namespace,DstK8S_OwnerName))*1000"
```
