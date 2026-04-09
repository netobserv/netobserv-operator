Network Observability is an OpenShift operator that deploys a monitoring pipeline consisting in:
- an eBPF agent, that generates network flows from captured packets
- flowlogs-pipeline, a component that collects, enriches and exports these flows
- a Console plugin for flows visualization with powerful filtering options, a topology representation and more

Flow data is then available in multiple ways, each optional:

- As Cluster Monitoring metrics
- As raw flow logs stored in Grafana Loki
- As raw flow logs exported to a collector

## Dependencies

### Loki

[Loki](https://grafana.com/oss/loki/), from GrafanaLabs, can optionally be used as the backend to store all collected flows. The Network Observability operator does not install Loki directly, except in demo mode; however we provide some guidance to help you there.

- For a production or production-like environment usage, refer to [the operator documentation](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/installing-network-observability-operators).

- For a quick try that is not suitable for production and not scalable, the demo mode can be configured in `FlowCollector` with:

```yaml
spec:
  loki:
    mode: Monolithic
    monolithic:
      installDemoLoki: true
```

It deploys a single pod, configures a 10GB storage PVC, with 24 hours of retention.

If you prefer to not use Loki, you must set `spec.loki.enable` to `false` in `FlowCollector`.
In that case, you still get the Prometheus metrics or export raw flows to a custom collector. But be aware that some of the Console plugin features will be disabled. For instance, you will not be able to view raw flows there, and the metrics / topology will have a more limited level of details, missing information such as pods or IPs.

### Kafka

[Apache Kafka](https://kafka.apache.org/) can optionally be used for a more resilient and scalable architecture. You can use for example [Strimzi](https://strimzi.io/), an operator-based distribution of Kafka for Kubernetes and OpenShift.

### Grafana

[Grafana](https://grafana.com/oss/grafana/) can optionally be installed for custom dashboards and query capabilities.

## Configuration

The `FlowCollector` resource is used to configure the operator and its managed components. A comprehensive documentation is [available here](https://github.com/netobserv/netobserv-operator/blob/1.11.3-community/docs/FlowCollector.md), and a full sample file [there](https://github.com/netobserv/netobserv-operator/blob/1.11.3-community/config/samples/flows_v1beta2_flowcollector.yaml).

To edit configuration in cluster, run:

```bash
oc edit flowcollector cluster
```

Only a single `FlowCollector` is allowed, and it has to be named `cluster`.

A couple of settings deserve special attention:

- Sampling (`spec.agent.ebpf.sampling`): a value of `100` means: one flow every 100 is sampled. `1` means all flows are sampled. The lower it is, the more flows you get, and the more accurate are derived metrics, but the higher amount of resources are consumed. By default, sampling is set to 50 (ie. 1:50). Note that more sampled flows also means more storage needed. We recommend to start with default values and refine empirically, to figure out which setting your cluster can manage.

- Loki (`spec.loki`): configure here how to reach Loki. The default values match the Loki quick install paths mentioned above, but you might have to configure differently if you used another installation method. Make sure to disable it (`spec.loki.enable`) if you don't want to use Loki.

- Processor replicas (`spec.processor.consumerReplicas`): how many replicas of `flowlogs-pipeline` should be deployed. Those pods collect, transform and re-export network flows. They can also be configured as unmanaged via `unmanagedReplicas`, if you want to use an auto-scaler.

- Kafka (`spec.deploymentModel: Kafka` and `spec.kafka`): when enabled, integrates the flow collection pipeline with Kafka, by splitting ingestion from transformation (kube enrichment, derived metrics, ...). Kafka can provide better scalability, resiliency and high availability ([view more details](https://www.redhat.com/en/topics/integration/what-is-apache-kafka)). Assumes Kafka is already deployed and a topic is created.

- Exporters (`spec.exporters`) an optional list of exporters to which to send enriched flows. KAFKA and IPFIX exporters are supported. This allows you to define any custom storage or processing that can read from Kafka or use the IPFIX standard.

- To enable availability zones awareness, set `spec.processor.addZone` to `true`.

## Resource considerations

The following table outlines examples of resource considerations for clusters with certain workload sizes.
The examples outlined in the table demonstrate scenarios that are tailored to specific workloads. Consider each example only as a baseline from which adjustments can be made to accommodate your workload needs. The test beds are:

- Extra small: 10 nodes cluster, 4 vCPUs and 16GiB mem per worker, LokiStack size `1x.extra-small`, tested on AWS M6i instances.
- Small: 25 nodes cluster, 16 vCPUs and 64GiB mem per worker, LokiStack size `1x.small`, tested on AWS M6i instances.
- Large: 250 nodes cluster, 16 vCPUs and 64GiB mem per worker, LokiStack size `1x.medium`, tested on AWS M6i instances. In addition to this worker and its controller, 3 infra nodes (size `M6i.12xlarge`) and 1 workload node (size `M6i.8xlarge`) were tested.


| Resource recommendations                                                          | Extra small (10 nodes) | Small (25 nodes)    | Large (250 nodes)    |
| --------------------------------------------------------------------------------- | ---------------------- | ------------------- | -------------------- |
| Operator memory limit<br>*In `Subscription` `spec.config.resources`*              | 400Mi (default)        | 400Mi (default)     | 400Mi (default)      |
| eBPF agent sampling interval<br>*In `FlowCollector` `spec.agent.ebpf.sampling`*   | 50 (default)           | 50 (default)        | 50 (default)         |
| eBPF agent memory limit<br>*In `FlowCollector` `spec.agent.ebpf.resources`*       | 800Mi (default)        | 800Mi (default)     | 1600Mi               |
| eBPF agent cache size<br>*In `FlowCollector` `spec.agent.ebpf.cacheMaxSize`*      | 50,000                 | 120,000 (default)   | 120,000 (default)    |
| Processor memory limit<br>*In `FlowCollector` `spec.processor.resources`*         | 800Mi (default)        | 800Mi (default)     | 800Mi (default)      |
| Processor replicas<br>*In `FlowCollector` `spec.processor.consumerReplicas`*      | 3 (default)            | 6                   | 18                   |
| Deployment model<br>*In `FlowCollector` `spec.deploymentModel`*                   | Service (default)      | Kafka               | Kafka                |
| Kafka partitions<br>*In your Kafka installation*                                  | N/A                    | 48                  | 48                   |
| Kafka brokers<br>*In your Kafka installation*                                     | N/A                    | 3 (default)         | 3 (default)          |

## Further reading

Please refer to the documentation on GitHub for more information.

This documentation includes:

- An [overview](https://github.com/netobserv/netobserv-operator#openshift-console) of the features, with screenshots
- More information on [configuring metrics](https://github.com/netobserv/netobserv-operator/blob/1.11.3-community/docs/Metrics.md).
- A [performance](https://github.com/netobserv/netobserv-operator#performance-fine-tuning) section, for fine-tuning
- A [security](https://github.com/netobserv/netobserv-operator#securing-data-and-communications) section
- An [F.A.Q.](https://github.com/netobserv/netobserv-operator#faq--troubleshooting) section
