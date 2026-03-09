# F.A.Q / Troubleshooting

If you can't find help here, don't hesitate to open [an issue](https://github.com/netobserv/netobserv-operator/issues) or a [Q&A](https://github.com/netobserv/netobserv-operator/discussions/categories/q-a). There are several repositories under _netobserv_ github org, but it is fine to centralize these in _network-observability-operator_.

## Table of Contents

* Q&A
  * [Is it for OpenShift only?](#is-it-for-openshift-only)
  * [Which version of Kubernetes / OpenShift is supported?](#which-version-of-kubernetes--openshift-is-supported)
* How-to
  * [How do I visualize flows and metrics?](#how-do-i-visualize-flows-and-metrics)
  * [How can I make sure everything is correctly deployed?](#how-can-i-make-sure-everything-is-correctly-deployed)
* Troubleshooting
  * [Everything seems correctly deployed but there isn't any flow showing up](#everything-seems-correctly-deployed-but-there-isnt-any-flow-showing-up)
  * [There is no Network Traffic menu entry in OpenShift Console](#there-is-no-network-traffic-menu-entry-in-openshift-console)
  * [I first deployed flowcollector, and then kafka. Flowlogs-pipeline is not consuming any flow from Kafka](#i-first-deployed-flowcollector-and-then-kafka-flowlogs-pipeline-is-not-consuming-any-flow-from-kafka)
  * [I get a Loki error / timeout, when trying to run a large query, such as querying for the last month of data](#i-get-a-loki-error--timeout-when-trying-to-run-a-large-query-such-as-querying-for-the-last-month-of-data)
  * [I don't see flows from either the `br-int` or `br-ex` interfaces](#i-dont-see-flows-from-either-the-br-int-or-br-ex-interfaces)
  * [I'm finding discrepancies in metrics](#im-finding-discrepancies-in-metrics)

## Q&A

### How can I make sure everything is correctly deployed?

Make sure all pods are up and running:

```bash
# Assuming configured namespace is netobserv (default)
kubectl get pods -n netobserv
```

Should provide results similar to this:

```
NAME                                            READY   STATUS    RESTARTS   AGE
flowlogs-pipeline-5rrg2                         1/1     Running   0          43m
flowlogs-pipeline-cp2lb                         1/1     Running   0          43m
flowlogs-pipeline-hmwxd                         1/1     Running   0          43m
flowlogs-pipeline-wmx4z                         1/1     Running   0          43m
grafana-6dbddc9869-sxn62                        1/1     Running   0          31m
loki                                            1/1     Running   0          43m
netobserv-controller-manager-7487d87dc-2ltq2    2/2     Running   0          43m
netobserv-plugin-7fb8c5477b-drg2z               1/1     Running   0          43m
```

Results may slightly differ depending on the installation method and the `FlowCollector` configuration. At least you should see `flowlogs-pipeline` pods in a `Running` state.

If you use the eBPF agent, check also for pods in privileged namespace:

```bash
# Assuming configured namespace is netobserv (default)
kubectl get pods -n netobserv-privileged
```

```
NAME                         READY   STATUS    RESTARTS   AGE
netobserv-ebpf-agent-7rwtk   1/1     Running   0          7s
netobserv-ebpf-agent-c7nkv   1/1     Running   0          7s
netobserv-ebpf-agent-hbjz8   1/1     Running   0          7s
netobserv-ebpf-agent-ldj66   1/1     Running   0          7s

```

Finally, make sure Loki is correctly deployed, and reachable from pods via the URL defined in `spec.loki.url`. You can for instance check using this command:

```bash
kubectl exec $(kubectl get pod -l "app=flowlogs-pipeline" -o name)  -- curl  -G -s "`kubectl get flowcollector cluster -o=jsonpath={.spec.loki.url}`loki/api/v1/query" --data-urlencode 'query={app="netobserv-flowcollector"}' --data-urlencode 'limit=1'
```

It should return some json in this form:

```
{"status":"success","data":{"resultType":"streams","result":[...],"stats":{...}}}
```

## Troubleshooting

### Everything seems correctly deployed but there isn't any flow showing up

If using IPFIX (ie. `spec.agent.type` is `IPFIX` in FlowCollector), wait 10 minutes and check again. There is sometimes a delay, up to 10 minutes, before the flows appear. This is due to the IPFIX protocol requiring exporter and collector to exchange record template definitions as a preliminary step. The eBPF agent doesn't have such a delay.

Else, check for any suspicious error in logs, especially in the `flowlogs-pipeline` pods and the eBPF agent pods. You may also take a look at prometheus metrics prefixed with `netobserv_`: they can give you clues if flows are processed, if errors are reported, etc.

Finally, don't hesitate to [open an issue](https://github.com/netobserv/netobserv-operator/issues).

### There is no Network Traffic menu entry in OpenShift Console

Make sure your cluster version is at least OpenShift 4.10: prior versions have no (or incompatible) console plugin SDK.

Make sure that `spec.consolePlugin.register` is set to `true` (default).

If not, or if for any reason the registration seems to have failed, you can still do it manually by editing the Console Operator config:

```bash
kubectl edit console.operator.openshift.io cluster
```

If it's not already there, add the plugin reference:

```yaml
spec:
  plugins:
  - netobserv-plugin
```

Ensure console pods are all in `Running` state using the following command:

```bash
oc get pods -n openshift-console -l app=console
```

If you had previously used the console with the plugin installed, you may need to restart console pods to clear cache:

```bash
oc delete pods -n openshift-console -l app=console
```

If the new views still don't show up, try clearing your browser cache and refreshing. Check also the `netobserv-console-plugin-...` pod status and logs.

```bash
kubectl get pods -n netobserv -l app=netobserv-plugin
kubectl logs -n netobserv -l app=netobserv-plugin
```

It is also possible that the OpenShift Console is failing to restart. There is for instance a [known issue](https://issues.redhat.com/browse/OCPBUGS-810) with very small clusters, causing this problem.
```bash
kubectl get pods -n openshift-console
```

```
NAME                         READY   STATUS    RESTARTS   AGE
console-65486f987c-bk6zn     1/1     Running   0          44m
console-67fd7b6f49-l5nmq     0/1     Pending   0          10m
console-67fd7b6f49-wmhp4     0/1     Pending   0          10m
downloads-5fc6cc467f-2fttx   1/1     Running   0          45m
downloads-5fc6cc467f-2gfpw   1/1     Running   0          45m
```

To force a restart, kill the running console pod using the following command:
`oc delete pods -n openshift-console -l app=console`
Note that this will make the console unavailable for some time

### I first deployed flowcollector, and then kafka. Flowlogs-pipeline is not consuming any flow from Kafka

This is a [known bug](https://github.com/segmentio/kafka-go/issues/1044) in one of flowlogs-pipeline dependencies.

Please recreate the flowlogs-pipeline pods by either killing them maunally or deleting and recreating the flow collector object.

### I get a Loki error / timeout, when trying to run a large query, such as querying for the last month of data

There are several ways to mitigate this issue, although there is no silver bullet. As a rule of thumb, be aware that Prometheus is a better fit than Loki to query on large time ranges.

With Loki queries, a first thing to understand is that, while Loki allows to query both on indexed and non-indexed fields (aka. labels), **queries that contain filters on labels will perform much better**. So, perhaps you can adapt your query to add an indexed filter. For instance if you were querying for a particular Pod (this isn't indexed), add its Namespace to the query. The list of indexed fields [is documented here](https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/json-flows-format-reference), in the `Loki label` column.

Depending on what you are trying to get, you may as well **consider querying Prometheus rather than Loki**. Queries on Prometheus are much faster than on Loki, it should not struggle with large time ranges, hence should be favored whenever possible. But Prometheus metrics do not contain as much information as flow logs in Loki, so whether or not you can do that really depends on the use case. When you use the NetObserv console plugin, it will try automatically to favor Prometheus over Loki if the query is compatible; else it falls back to Loki. If your query does't run against Prometheus, changing some filters or aggregations can make the switch. In the console plugin, you can force the use of Prometheus. Incompatible queries will fail, and the error message displayed should help you figure out which labels you can try to change to make the query compatible (for instance, changing a filter or an aggregation from Resource/Pods to Owner).

If the data that you need isn't available as a Prometheus metric, you may also **consider using the [FlowMetrics API](https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md#custom-metrics-using-the-flowmetrics-api)** to create your own metric. You need to be careful about the metrics cardinality, as explained in this link.

If the problem persists, there are ways to **configure Loki to improve the query performance**. Some options depend on the installation mode you used for Loki (using the Operator and `LokiStack`, or `Monolithic` mode, or `Microservices` mode):

- In `LokiStack` or `Microservices` modes, try [increasing the number of querier replicas](https://loki-operator.dev/docs/api.md/#loki-grafana-com-v1-LokiComponentSpec)
- Increase the [query timeout](https://loki-operator.dev/docs/api.md/#loki-grafana-com-v1-QueryLimitSpec). You will also need to increase NetObserv read timeout to Loki accordingly, in `FlowCollector` `spec.loki.readTimeout`.


### I don't see flows from either the `br-int` or `br-ex` interfaces

[`br-ex` and `br-int` are virtual bridge devices](https://access.redhat.com/documentation/en-us/red_hat_openstack_platform/16.0/html/networking_guide/bridge-mappings),
so they operate at OSI Layer 2 (e.g. Ethernet level). The eBPF agent works at Layers 3 and 4
(IP and TCP level), so it is expected that traffic passing through `br-int` and `br-ex` is captured
by the agent when it is processed by other interfaces (e.g. physical host or virtual pod interfaces).

This means that, if you restrict the agent interfaces (using the `interfaces` or `excludeInterfaces`
properties) to attach only to `br-int` and/or `br-ex`, you won't be able to see any flow.

### I'm finding discrepancies in metrics

1. NetObserv metrics (such as `netobserv_workload_ingress_bytes_total`) show *higher values* than cadvisor metrics (such as `container_network_receive_bytes_total`)

This can be caused when traffic goes through Kubernetes Services: when a Pod talks to another Pod via a Service, two flows are generated: one against the service and one against the pod. To avoid querying duplicated counts, you can refine your promQL to ignore traffic targeting Services: e.g: `sum(rate(netobserv_workload_ingress_bytes_total{DstK8S_Namespace="my-namespace",SrcK8S_Type!="Service",DstK8S_Type!="Service"}[2m]))`
	
2. NetObserv metrics (such as `netobserv_workload_ingress_bytes_total`) show *lower values* than cadvisor metrics (such as `container_network_receive_bytes_total`)

There are several possible causes:

- Sampling is being used. Check your `FlowCollector` `spec.agent.ebpf.sampling`: a value greater than 1 means not every flows are sampled. NetObserv metrics aren't normalized automatically, but you can do so in your promQL by multiplying with the sampling interval, for instance: `sum(rate(netobserv_workload_ingress_bytes_total{DstK8S_Namespace="my-namespace"}[2m])) * avg(netobserv_agent_sampling_rate > 0)`. Be aware that, the higher the sampling, the less accurate the metrics.

- Filters are configured in the agent, resulting in ignoring some of the traffic. Check your `FlowCollector` `spec.agent.ebpf.flowFilter`, `spec.agent.ebpf.interfaces`, `spec.agent.ebpf.excludeInterfaces` and make sure it doesn't filter out some of the traffic that you are looking at.

- Flows may also be dropped due to constraints on resources. Monitor the eBPF agent health in the `NetObserv / Health` dashboard: there are graphs showing drops. Increasing `spec.agent.ebpf.cacheMaxSize` can help to avoid these drops, at the cost of an increased memory usage.
