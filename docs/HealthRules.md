# Health rules in NetObserv

The NetObserv operator comes with a set of predefined health rules, based on its [metrics](./Metrics.md), that you can configure, extend or disable.
These rules are converted into a `PrometheusRule` resource, either as Alerts or as Recording rules. The alerts are then managed by Prometheus AlertManager. Both recording rules and alerts are displayed in the Network Health page of the Console.

These health rules are provided as a convenience, to take the most of NetObserv built-in metrics without requiring you to write complex PromQL or to do fine-tuning. They give a health indication of your cluster network.

To get a detailed description of the rules, [check the runbooks](https://github.com/openshift/runbooks/tree/master/alerts/network-observability-operator).

## Default rules

By default, NetObserv creates health rules contextual to the enabled features. For example, packet drops related rules are only created if the `PacketDrop` feature is enabled. Because rules are built upon metrics, you may also see configuration warnings if some enabled rules are missing their required metrics, which can be configured in `spec.processor.metrics.includeList` (see [Metrics.md](./Metrics.md)).

These rules are installed by default:

- `PacketDropsByDevice`
- `PacketDropsByKernel`
- `IPsecErrors`
- `NetpolDenied`
- `LatencyHighTrend`
- `DNSErrors`
- `DNSNxDomain`
- `ExternalEgressHighTrend`
- `ExternalIngressHighTrend`
- `Ingress5xxErrors`
- `IngressHTTPLatencyTrend`

On top of that, there are also some operational alerts that relate to NetObserv's self health:

- `NetObservNoFlows`: triggered when no flows are being observed for a certain period.
- `NetObservLokiError`: triggered when flows are being dropped due to Loki errors.

## Other alert templates

Templates that are not enabled by default, but available for configuration: (none at this time).

## Configure predefined alerts

Alerts are configured in the `FlowCollector` custom resource, via `spec.processor.metrics.alerts`.

They are organized by templates and variants. The template names are the ones listed above, such as `PacketDropsByKernel`. For each template, you can define a list of variants, each with their thresholds and grouping configuration.

Example:

```yaml
spec:
  processor:
    metrics:
      healthRules:
      - template: PacketDropsByKernel
        mode: Alert # or Recording
        variants:
        # triggered when the whole cluster traffic (no grouping) reaches 10% of drops
        - thresholds:
            critical: "10"
        # triggered when per-node traffic reaches 5% of drops, with gradual severity
        - thresholds:
            critical: "15"
            warning: "10"
            info: "5"
          groupBy: Node
```

The `mode` setting can be either defined per variant, or for the whole template.

When you configure a template, it overrides the default configuration for that template. So, if you want to add a new rule on top of the default ones for a template, you may want to replicate the default configuration manually. All defaults are described in the [runbooks](https://github.com/openshift/runbooks/tree/master/alerts/network-observability-operator).

## Disable predefined alerts

Alert templates can be disabled in `spec.processor.metrics.disableAlerts`. This settings accepts a list of template names, as listed above.

If a template is disabled _and_ overridden in `spec.processor.metrics.healthRules`, the disable setting takes precedence: the alert rule will not be created.

## Creating your own custom rules that contribute to the Health dashboard

Beyond configuring the predefined health rules, you can create your own custom rules that contribute to the Network Health dashboard. NetObserv supports two types of custom rules:

- **Alert Rules**
- **Recording Rules**

Both types use the Prometheus operator API via `PrometheusRule` resources. You can check the actual generated resources by running:

```bash
kubectl get prometheusrules -n netobserv -oyaml
```

### Requirements for custom rules

- **Label requirement**: Add the label `netobserv: "true"` to each rule's `labels` section so the rule is considered for Network Health.
  - **For recording rules only**: Also add the label `netobserv: "true"` to the **PrometheusRule metadata**. The operator lists PrometheusRules **cluster-wide** with this label.
  - **For alerts**: The label in the PrometheusRule metadata is not required. Alerts are discovered by Prometheus.
- **Lifecycle and namespace**: Custom PrometheusRules are **not** owned by the FlowCollector. If you put them in the **NetObserv namespace** (e.g. `netobserv`), that namespace may be removed when the FlowCollector is uninstalled (depending on install mode), and your rules would be deleted. To avoid that, create your PrometheusRules in **another namespace** (e.g. `monitoring` or a dedicated `netobserv-rules`). If you do keep rules in the NetObserv namespace, back them up (e.g. in Git) and re-apply after reinstalling.

### Custom Alert Rules

You can create your own custom `PrometheusRule` resources with alerting rules. You'll need to be familiar with PromQL to write these rules.

[Click here](../config/samples/alerts) to see sample alerts that are not built-in NetObserv.

Let's take the [incoming-traffic-surge](../config/samples/alerts/incoming-traffic-surge.yaml) as an example. This alert triggers when the current ingress traffic exceeds by more than twice the traffic from the day before.

### Anatomy of the PromQL

Here's the PromQL:

```
(100 *
  (
    (sum(rate(netobserv_workload_ingress_bytes_total{SrcK8S_Namespace="openshift-ingress"}[30m])) by (DstK8S_Namespace) > 1000)
    - sum(rate(netobserv_workload_ingress_bytes_total{SrcK8S_Namespace="openshift-ingress"}[30m] offset 1d)) by (DstK8S_Namespace)
  )
  / sum(rate(netobserv_workload_ingress_bytes_total{SrcK8S_Namespace="openshift-ingress"}[30m] offset 1d)) by (DstK8S_Namespace))
> 100
```

Let's break it down. The base query pattern is this:

`sum(rate(netobserv_workload_ingress_bytes_total{SrcK8S_Namespace="openshift-ingress"}[30m])) by (DstK8S_Namespace)`

This is the bytes rate coming from "openshift-ingress" to any of your workload's namespaces, over the last 30 minutes. This metric is provided by NetObserv (note that depending on your FlowCollector configuration, you may need to use `netobserv_namespace_ingress_bytes_total` instead of `netobserv_workload_ingress_bytes_total`).

Appending ` > 1000` to this query keeps only the rates observed greater than 1KBps, in order to eliminate the noise from low-bandwidth consumers. 1KBps still isn't a lot, you may want to increase it. Note also that the bytes rate is relative to the sampling interval defined in the `FlowCollector` agent configuration. If you have a sampling ratio of 1:100, consider that the actual traffic might be approximately 100 times higher than what is reported by the metrics. Alternatively, the metric `netobserv_agent_sampling_rate` can be use to normalize the byte rates, decoupling the promql from the sampling configuration.

In the following parts of the PromQL, you can see `offset 1d`: this is to run the same query, one day earlier. You can change that according to your needs, for instance `offset 5h` will be five hours ago.

Which gives us the formula `100 * (<query now> - <query yesterday>) / <query yesterday>`: it's the percentage of increase compared to yesterday. It can be negative, if the bytes rate today is lower than yesterday.

Finally, the last part, `> 100`, eliminates increases that are lower than 100%, so that we don't get alerted by that.

### Metadata

Some metadata is required to work with Prometheus and AlertManager (not specific to NetObserv):

```yaml
      annotations:
        message: |-
          NetObserv is detecting a surge of incoming traffic: current traffic to {{ $labels.DstK8S_Namespace }} has increased by more than 100% since yesterday.
        summary: "Surge in incoming traffic"
      labels:
        severity: warning
```

As you can see, you can leverage the output labels from the PromQL defined previously in the description. Here, since we've grouped the results per `DstK8S_Namespace`, we can use it in our text.

The severity label should be "critical", "warning" or "info".

On top of that, in order to have the rule picked up in the Health dashboard, NetObserv needs other information:

```yaml
      annotations:
        netobserv_io_network_health: '{"namespaceLabels":["DstK8S_Namespace"],"threshold":"100","unit":"%","upperBound":"500"}'
      labels:
        netobserv: "true"
```

See the [Common annotation fields](#common-annotation-fields) section above for details on the `netobserv_io_network_health` annotation.

### Custom Recording Rules

In addition to alert rules, you can create your own **recording rules** so that custom pre-computed metrics appear on the Network Health page. The PrometheusRule CRD does not allow annotations on individual rules of type `record`, so NetObserv uses a single annotation on the `PrometheusRule` resource to describe metadata for all recording rules in that resource.

#### Recording rules specific requirements

- Add the annotation `netobserv.io/network-health` on the **PrometheusRule metadata** (**required** - rules without this annotation will not appear in the Health dashboard). The value is a JSON object: keys are the **metric names** (the `record:` field of each rule), and each value consists of:
  - `summary` (optional): Short title; may use Prometheus template (e.g. `{{ $labels.namespace }}`).
  - `description` (optional): Longer description; may use template.
  - `netobserv_io_network_health` (**required**): JSON string with the same fields described in [Common annotation fields](#common-annotation-fields). For recording rules, use `recordingThresholds` instead of `threshold`.

#### Example

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: my-recording-rules
  namespace: netobserv
  annotations:
    netobserv.io/network-health: |
      {
        "my_metric_per_namespace": {
          "summary": "Custom metric is {{ $value }} in the namespace {{ $labels.namespace }}",
          "description": "Custom metric is {{ $value }} in the namespace {{ $labels.namespace }}",
          "netobserv_io_network_health": "{\"unit\":\"%\",\"upperBound\":\"100\",\"namespaceLabels\":[\"namespace\"],\"recordingThresholds\":{\"info\":\"10\",\"warning\":\"25\",\"critical\":\"50\"}}"
        }
      }
spec:
  groups:
    - name: MyRecordingRules
      interval: 30s
      rules:
        - record: my_metric_per_namespace
          expr: (count by (namespace) (kube_pod_info) * 0 + 20)
          labels:
            netobserv: "true"
```

### Common annotation fields

The annotation `netobserv_io_network_health` is optional for alert rules but **required for recording rules**. It gives you some control on how the rule renders in the Health page. It is a JSON string that consists in:

- `namespaceLabels`: one or more labels that hold namespaces. When provided, the rule will show up under the "Namespaces" tab.
- `nodeLabels`: one or more labels that hold node names. When provided, the rule will show up under the "Nodes" tab.
- `workloadLabels`: one or more labels that hold owner/workload names. When provided alongside with `kindLabels`, the rule will show up under the "Workloads" tab.
- `kindLabels`: one or more labels that hold owner/workload kinds. When provided alongside with `workloadLabels`, the rule will show up under the "Workloads" tab.
- `unit`: the data unit, used only for display purpose.
- `upperBound`: an upper bound value used to compute score on a closed scale. It doesn't necessarily have to be a maximum of the metric values, but metric values will be clamped if they are above the upper bound.
- `links`: a list of links to be displayed contextually to the rule. Each link consists in:
  - `name`: display name.
  - `url`: the link URL.
- `trafficLink`: information related to the link to the Network Traffic page, for URL building. Some filters will be set automatically, such as the node or namespace filter.
  - `extraFilter`: an additional filter to inject (e.g: a DNS response code, for DNS-related alerts).
  - `backAndForth`: should the filter include return traffic? (true/false)
  - `filterDestination`: should the filter target the destination of the traffic instead of the source? (true/false)

`namespaceLabels` and `nodeLabels` are mutually exclusive. If none of them is provided, the rule will show up under the "Global" tab.

**Alert-specific fields:**
- `threshold`: the alert threshold as a string, expected to match the one defined in PromQL.

**Recording rule-specific fields:**
- `recordingThresholds`: thresholds for recording rules to drive the health score and coloring (e.g. `{"info":"10","warning":"25","critical":"50"}`).