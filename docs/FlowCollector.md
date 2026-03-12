# API Reference

Packages:

- [flows.netobserv.io/v1beta2](#flowsnetobserviov1beta2)

# flows.netobserv.io/v1beta2

Resource Types:

- [FlowCollector](#flowcollector)




## FlowCollector
<sup><sup>[↩ Parent](#flowsnetobserviov1beta2 )</sup></sup>






`FlowCollector` is the schema for the network flows collection API, which pilots and configures the underlying deployments.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>flows.netobserv.io/v1beta2</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>FlowCollector</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspec">spec</a></b></td>
        <td>object</td>
        <td>
          Defines the desired state of the FlowCollector resource.
<br><br>
*: the mention of "unsupported" or "deprecated" for a feature throughout this document means that this feature
is not officially supported by Red Hat. It might have been, for example, contributed by the community
and accepted without a formal agreement for maintenance. The product maintainers might provide some support
for these features as a best effort only.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatus">status</a></b></td>
        <td>object</td>
        <td>
          `FlowCollectorStatus` defines the observed state of FlowCollector<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec
<sup><sup>[↩ Parent](#flowcollector)</sup></sup>



Defines the desired state of the FlowCollector resource.
<br><br>
*: the mention of "unsupported" or "deprecated" for a feature throughout this document means that this feature
is not officially supported by Red Hat. It might have been, for example, contributed by the community
and accepted without a formal agreement for maintenance. The product maintainers might provide some support
for these features as a best effort only.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagent">agent</a></b></td>
        <td>object</td>
        <td>
          Agent configuration for flows extraction.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsoleplugin">consolePlugin</a></b></td>
        <td>object</td>
        <td>
          `consolePlugin` defines the settings related to the OpenShift Console plugin, when available.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>deploymentModel</b></td>
        <td>enum</td>
        <td>
          `deploymentModel` defines the desired type of deployment for flow processing. Possible values are:<br>
- `Service` (default) to make the flow processor listen as a Kubernetes Service, backed by a scalable Deployment.<br>
- `Kafka` to make flows sent to a Kafka pipeline before consumption by the processor.<br>
- `Direct` to make the flow processor listen directly from the agents using the host network, backed by a DaemonSet. Only recommended on small clusters, below 15 nodes.<br>
Kafka can provide better scalability, resiliency, and high availability (for more details, see https://www.redhat.com/en/topics/integration/what-is-apache-kafka).<br>
`Direct` is not recommended on large clusters as it is less memory efficient.<br/>
          <br/>
            <i>Enum</i>: Service, Direct, Kafka<br/>
            <i>Default</i>: Service<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexecution">execution</a></b></td>
        <td>object</td>
        <td>
          `execution` defines configuration related to the execution of the flow collection process.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindex">exporters</a></b></td>
        <td>[]object</td>
        <td>
          `exporters` defines additional optional exporters for custom consumption or storage.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafka">kafka</a></b></td>
        <td>object</td>
        <td>
          Kafka configuration, allowing to use Kafka as a broker as part of the flow collection pipeline. Available when the `spec.deploymentModel` is `Kafka`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecloki">loki</a></b></td>
        <td>object</td>
        <td>
          `loki`, the flow store, client settings.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace where NetObserv pods are deployed.<br/>
          <br/>
            <i>Default</i>: netobserv<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecnetworkpolicy">networkPolicy</a></b></td>
        <td>object</td>
        <td>
          `networkPolicy` defines network policy settings for NetObserv components isolation.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessor">processor</a></b></td>
        <td>object</td>
        <td>
          `processor` defines the settings of the component that receives the flows from the agent,
enriches them, generates metrics, and forwards them to the Loki persistence layer and/or any available exporter.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprometheus">prometheus</a></b></td>
        <td>object</td>
        <td>
          `prometheus` defines Prometheus settings, such as querier configuration used to fetch metrics from the Console plugin.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



Agent configuration for flows extraction.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpf">ebpf</a></b></td>
        <td>object</td>
        <td>
          `ebpf` describes the settings related to the eBPF-based flow reporter when `spec.agent.type`
is set to `eBPF`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentipfix">ipfix</a></b></td>
        <td>object</td>
        <td>
          `ipfix` [deprecated (*)] - describes the settings related to the IPFIX-based flow reporter when `spec.agent.type`
is set to `IPFIX`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          `type` [deprecated (*)] selects the flows tracing agent. Previously, this field allowed to select between `eBPF` or `IPFIX`.
Only `eBPF` is allowed now, so this field is deprecated and is planned for removal in a future version of the API.<br/>
          <br/>
            <i>Enum</i>: eBPF, IPFIX<br/>
            <i>Default</i>: eBPF<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf
<sup><sup>[↩ Parent](#flowcollectorspecagent)</sup></sup>



`ebpf` describes the settings related to the eBPF-based flow reporter when `spec.agent.type`
is set to `eBPF`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvanced">advanced</a></b></td>
        <td>object</td>
        <td>
          `advanced` allows setting some aspects of the internal configuration of the eBPF agent.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk. You can also
override the default Linux capabilities from there.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cacheActiveTimeout</b></td>
        <td>string</td>
        <td>
          `cacheActiveTimeout` is the period during which the agent aggregates flows before sending.
Increasing `cacheMaxFlows` and `cacheActiveTimeout` can decrease the network traffic overhead and the CPU load,
however you can expect higher memory consumption and an increased latency in the flow collection.<br/>
          <br/>
            <i>Default</i>: 15s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cacheMaxFlows</b></td>
        <td>integer</td>
        <td>
          `cacheMaxFlows` is the maximum number of flows in an aggregate; when reached, the reporter sends the flows.
Increasing `cacheMaxFlows` and `cacheActiveTimeout` can decrease the network traffic overhead and the CPU load,
however you can expect higher memory consumption and an increased latency in the flow collection.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 120000<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>excludeInterfaces</b></td>
        <td>[]string</td>
        <td>
          `excludeInterfaces` contains the interface names that are excluded from flow tracing.
An entry enclosed by slashes, such as `/br-/`, is matched as a regular expression.
Otherwise it is matched as a case-sensitive string.<br/>
          <br/>
            <i>Default</i>: [lo]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>features</b></td>
        <td>[]enum</td>
        <td>
          List of additional features to enable. They are all disabled by default. Enabling additional features might have performance impacts. Possible values are:<br>
- `PacketDrop`: Enable the packets drop flows logging feature. This feature requires mounting
the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.<br>
- `DNSTracking`: Enable the DNS tracking feature.<br>
- `FlowRTT`: Enable flow latency (sRTT) extraction in the eBPF agent from TCP traffic.<br>
- `NetworkEvents`: Enable the network events monitoring feature, such as correlating flows and network policies.
This feature requires mounting the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.
It requires using the OVN-Kubernetes network plugin with the Observability feature.
IMPORTANT: This feature is available as a Technology Preview.<br>
- `PacketTranslation`: Enable enriching flows with packet translation information, such as Service NAT.<br>
- `EbpfManager`: [Unsupported (*)]. Use eBPF Manager to manage NetObserv eBPF programs. Pre-requisite: the eBPF Manager operator (or upstream bpfman operator) must be installed.<br>
- `UDNMapping`: Enable interfaces mapping to User Defined Networks (UDN). <br>
This feature requires mounting the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.
It requires using the OVN-Kubernetes network plugin with the Observability feature. <br>
- `IPSec`, to track flows between nodes with IPsec encryption. <br><br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfflowfilter">flowFilter</a></b></td>
        <td>object</td>
        <td>
          `flowFilter` defines the eBPF agent configuration regarding flow filtering.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>enum</td>
        <td>
          `imagePullPolicy` is the Kubernetes pull policy for the image defined above<br/>
          <br/>
            <i>Enum</i>: IfNotPresent, Always, Never<br/>
            <i>Default</i>: IfNotPresent<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>interfaces</b></td>
        <td>[]string</td>
        <td>
          `interfaces` contains the interface names from where flows are collected. If empty, the agent
fetches all the interfaces in the system, excepting the ones listed in `excludeInterfaces`.
An entry enclosed by slashes, such as `/br-/`, is matched as a regular expression.
Otherwise it is matched as a case-sensitive string.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kafkaBatchSize</b></td>
        <td>integer</td>
        <td>
          `kafkaBatchSize` limits the maximum size of a request in bytes before being sent to a partition. Ignored when not using Kafka. Default: 1MB.<br/>
          <br/>
            <i>Default</i>: 1048576<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>enum</td>
        <td>
          `logLevel` defines the log level for the NetObserv eBPF Agent<br/>
          <br/>
            <i>Enum</i>: trace, debug, info, warn, error, fatal, panic<br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfmetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          `metrics` defines the eBPF agent configuration regarding metrics.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileged</b></td>
        <td>boolean</td>
        <td>
          Privileged mode for the eBPF Agent container. When set to `true`, the agent is able to capture more traffic, including from secondary interfaces.
When ignored or set to `false`, the operator sets granular capabilities (BPF, PERFMON, NET_ADMIN) to the container.
Some agent features require the privileged mode, such as packet drops tracking (see `features`) and SR-IOV support.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfresources">resources</a></b></td>
        <td>object</td>
        <td>
          `resources` are the compute resources required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
          <br/>
            <i>Default</i>: map[limits:map[memory:800Mi] requests:map[cpu:100m memory:50Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          Sampling interval of the eBPF probe. 100 means one packet on 100 is sent. 0 or 1 means all packets are sampled.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 50<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced
<sup><sup>[↩ Parent](#flowcollectorspecagentebpf)</sup></sup>



`advanced` allows setting some aspects of the internal configuration of the eBPF agent.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk. You can also
override the default Linux capabilities from there.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>capOverride</b></td>
        <td>[]string</td>
        <td>
          Linux capabilities override, when not running as privileged. Default capabilities are BPF, PERFMON and NET_ADMIN.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>env</b></td>
        <td>map[string]string</td>
        <td>
          `env` allows passing custom environment variables to underlying components. Useful for passing
some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
publicly exposed as part of the FlowCollector descriptor, as they are only useful
in edge debug or support scenarios.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedscheduling">scheduling</a></b></td>
        <td>object</td>
        <td>
          scheduling controls how the pods are scheduled on nodes.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvanced)</sup></sup>



scheduling controls how the pods are scheduled on nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          `nodeSelector` allows scheduling of pods only onto nodes that have each of the specified labels.
For documentation, refer to https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          If specified, indicates the pod's priority. For documentation, refer to https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#how-to-use-priority-and-preemption.
If not specified, default priority is used, or zero if there is no default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingtolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedscheduling)</sup></sup>



If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node matches the corresponding matchExpressions; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0
(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of
them are ANDed.
The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the anti-affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling anti-affinity expressions, etc.),
compute a sum by iterating through the elements of this field and subtracting
"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the anti-affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.advanced.scheduling.tolerations[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfadvancedscheduling)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects.
When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys.
If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value.
Valid operators are Exists, Equal, Lt, and Gt. Defaults to Equal.
Exists is equivalent to wildcard for value, so that a pod can
tolerate all taints of a particular category.
Lt and Gt perform numeric comparisons (requires feature gate TaintTolerationComparisonOperators).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be
of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
it is not set, which means tolerate the taint forever (do not evict). Zero and
negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to.
If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.flowFilter
<sup><sup>[↩ Parent](#flowcollectorspecagentebpf)</sup></sup>



`flowFilter` defines the eBPF agent configuration regarding flow filtering.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>action</b></td>
        <td>enum</td>
        <td>
          `action` defines the action to perform on the flows that match the filter. The available options are `Accept`, which is the default, and `Reject`.<br/>
          <br/>
            <i>Enum</i>: Accept, Reject<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cidr</b></td>
        <td>string</td>
        <td>
          `cidr` defines the IP CIDR to filter flows by.
Examples: `10.10.10.0/24` or `100:100:100:100::/64`<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>destPorts</b></td>
        <td>int or string</td>
        <td>
          `destPorts` optionally defines the destination ports to filter flows by.
To filter a single port, set a single port as an integer value. For example, `destPorts: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `destPorts: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>direction</b></td>
        <td>enum</td>
        <td>
          `direction` optionally defines a direction to filter flows by. The available options are `Ingress` and `Egress`.<br/>
          <br/>
            <i>Enum</i>: Ingress, Egress<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Set `enable` to `true` to enable the eBPF flow filtering feature.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>icmpCode</b></td>
        <td>integer</td>
        <td>
          `icmpCode`, for Internet Control Message Protocol (ICMP) traffic, optionally defines the ICMP code to filter flows by.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>icmpType</b></td>
        <td>integer</td>
        <td>
          `icmpType`, for ICMP traffic, optionally defines the ICMP type to filter flows by.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>peerCIDR</b></td>
        <td>string</td>
        <td>
          `peerCIDR` defines the Peer IP CIDR to filter flows by.
Examples: `10.10.10.0/24` or `100:100:100:100::/64`<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>peerIP</b></td>
        <td>string</td>
        <td>
          `peerIP` optionally defines the remote IP address to filter flows by.
Example: `10.10.10.10`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pktDrops</b></td>
        <td>boolean</td>
        <td>
          `pktDrops` optionally filters only flows containing packet drops.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ports</b></td>
        <td>int or string</td>
        <td>
          `ports` optionally defines the ports to filter flows by. It is used both for source and destination ports.
To filter a single port, set a single port as an integer value. For example, `ports: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `ports: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>enum</td>
        <td>
          `protocol` optionally defines a protocol to filter flows by. The available options are `TCP`, `UDP`, `ICMP`, `ICMPv6`, and `SCTP`.<br/>
          <br/>
            <i>Enum</i>: TCP, UDP, ICMP, ICMPv6, SCTP<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfflowfilterrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          `rules` defines a list of filtering rules on the eBPF Agents.
When filtering is enabled, by default, flows that don't match any rule are rejected.
To change the default, you can define a rule that accepts everything: `{ action: "Accept", cidr: "0.0.0.0/0" }`, and then refine with rejecting rules.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is the sampling interval for the matched packets, overriding the global sampling defined at `spec.agent.ebpf.sampling`.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sourcePorts</b></td>
        <td>int or string</td>
        <td>
          `sourcePorts` optionally defines the source ports to filter flows by.
To filter a single port, set a single port as an integer value. For example, `sourcePorts: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `sourcePorts: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tcpFlags</b></td>
        <td>enum</td>
        <td>
          `tcpFlags` optionally defines TCP flags to filter flows by.
In addition to the standard flags (RFC-9293), you can also filter by one of the three following combinations: `SYN-ACK`, `FIN-ACK`, and `RST-ACK`.<br/>
          <br/>
            <i>Enum</i>: SYN, SYN-ACK, ACK, FIN, RST, URG, ECE, CWR, FIN-ACK, RST-ACK<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.flowFilter.rules[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfflowfilter)</sup></sup>



`EBPFFlowFilterRule` defines the desired eBPF agent configuration regarding flow filtering rule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>action</b></td>
        <td>enum</td>
        <td>
          `action` defines the action to perform on the flows that match the filter. The available options are `Accept`, which is the default, and `Reject`.<br/>
          <br/>
            <i>Enum</i>: Accept, Reject<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cidr</b></td>
        <td>string</td>
        <td>
          `cidr` defines the IP CIDR to filter flows by.
Examples: `10.10.10.0/24` or `100:100:100:100::/64`<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>destPorts</b></td>
        <td>int or string</td>
        <td>
          `destPorts` optionally defines the destination ports to filter flows by.
To filter a single port, set a single port as an integer value. For example, `destPorts: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `destPorts: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>direction</b></td>
        <td>enum</td>
        <td>
          `direction` optionally defines a direction to filter flows by. The available options are `Ingress` and `Egress`.<br/>
          <br/>
            <i>Enum</i>: Ingress, Egress<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>icmpCode</b></td>
        <td>integer</td>
        <td>
          `icmpCode`, for Internet Control Message Protocol (ICMP) traffic, optionally defines the ICMP code to filter flows by.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>icmpType</b></td>
        <td>integer</td>
        <td>
          `icmpType`, for ICMP traffic, optionally defines the ICMP type to filter flows by.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>peerCIDR</b></td>
        <td>string</td>
        <td>
          `peerCIDR` defines the Peer IP CIDR to filter flows by.
Examples: `10.10.10.0/24` or `100:100:100:100::/64`<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>peerIP</b></td>
        <td>string</td>
        <td>
          `peerIP` optionally defines the remote IP address to filter flows by.
Example: `10.10.10.10`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pktDrops</b></td>
        <td>boolean</td>
        <td>
          `pktDrops` optionally filters only flows containing packet drops.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ports</b></td>
        <td>int or string</td>
        <td>
          `ports` optionally defines the ports to filter flows by. It is used both for source and destination ports.
To filter a single port, set a single port as an integer value. For example, `ports: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `ports: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>enum</td>
        <td>
          `protocol` optionally defines a protocol to filter flows by. The available options are `TCP`, `UDP`, `ICMP`, `ICMPv6`, and `SCTP`.<br/>
          <br/>
            <i>Enum</i>: TCP, UDP, ICMP, ICMPv6, SCTP<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is the sampling interval for the matched packets, overriding the global sampling defined at `spec.agent.ebpf.sampling`.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sourcePorts</b></td>
        <td>int or string</td>
        <td>
          `sourcePorts` optionally defines the source ports to filter flows by.
To filter a single port, set a single port as an integer value. For example, `sourcePorts: 80`.
To filter a range of ports, use a "start-end" range in string format. For example, `sourcePorts: "80-100"`.
To filter two ports, use a "port1,port2" in string format. For example, `ports: "80,100"`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tcpFlags</b></td>
        <td>enum</td>
        <td>
          `tcpFlags` optionally defines TCP flags to filter flows by.
In addition to the standard flags (RFC-9293), you can also filter by one of the three following combinations: `SYN-ACK`, `FIN-ACK`, and `RST-ACK`.<br/>
          <br/>
            <i>Enum</i>: SYN, SYN-ACK, ACK, FIN, RST, URG, ECE, CWR, FIN-ACK, RST-ACK<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.metrics
<sup><sup>[↩ Parent](#flowcollectorspecagentebpf)</sup></sup>



`metrics` defines the eBPF agent configuration regarding metrics.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>disableAlerts</b></td>
        <td>[]enum</td>
        <td>
          `disableAlerts` is a list of alerts that should be disabled.
Possible values are:<br>
`NetObservDroppedFlows`, which is triggered when the eBPF agent is missing packets or flows, such as when the BPF hashmap is busy or full, or the capacity limiter is being triggered.<br><br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Set `enable` to `false` to disable eBPF agent metrics collection. It is enabled by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfmetricsserver">server</a></b></td>
        <td>object</td>
        <td>
          Metrics server endpoint configuration for the Prometheus scraper.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.metrics.server
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfmetrics)</sup></sup>



Metrics server endpoint configuration for the Prometheus scraper.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          The metrics server HTTP port.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfmetricsservertls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.metrics.server.tls
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfmetricsserver)</sup></sup>



TLS configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Select the type of TLS configuration:<br>
- `Disabled` (default) to not configure TLS for the endpoint.
- `Provided` to manually provide cert file and a key file. [Unsupported (*)].
- `Auto` to use OpenShift auto generated certificate using annotations.<br/>
          <br/>
            <i>Enum</i>: Disabled, Provided, Auto<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the provided certificate.
If set to `true`, the `providedCaFile` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfmetricsservertlsprovided">provided</a></b></td>
        <td>object</td>
        <td>
          TLS configuration when `type` is set to `Provided`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentebpfmetricsservertlsprovidedcafile">providedCaFile</a></b></td>
        <td>object</td>
        <td>
          Reference to the CA file when `type` is set to `Provided`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.metrics.server.tls.provided
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfmetricsservertls)</sup></sup>



TLS configuration when `type` is set to `Provided`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.metrics.server.tls.providedCaFile
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfmetricsservertls)</sup></sup>



Reference to the CA file when `type` is set to `Provided`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.resources
<sup><sup>[↩ Parent](#flowcollectorspecagentebpf)</sup></sup>



`resources` are the compute resources required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecagentebpfresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims,
that are used by this container.

This field depends on the
DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ebpf.resources.claims[index]
<sup><sup>[↩ Parent](#flowcollectorspecagentebpfresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of
the Pod where this field is used. It makes that resource available
inside a container.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>request</b></td>
        <td>string</td>
        <td>
          Request is the name chosen for a request in the referenced claim.
If empty, everything from the claim is made available, otherwise
only the result of this request.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ipfix
<sup><sup>[↩ Parent](#flowcollectorspecagent)</sup></sup>



`ipfix` [deprecated (*)] - describes the settings related to the IPFIX-based flow reporter when `spec.agent.type`
is set to `IPFIX`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cacheActiveTimeout</b></td>
        <td>string</td>
        <td>
          `cacheActiveTimeout` is the max period during which the reporter aggregates flows before sending.<br/>
          <br/>
            <i>Default</i>: 20s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>cacheMaxFlows</b></td>
        <td>integer</td>
        <td>
          `cacheMaxFlows` is the max number of flows in an aggregate; when reached, the reporter sends the flows.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 400<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentipfixclusternetworkoperator">clusterNetworkOperator</a></b></td>
        <td>object</td>
        <td>
          `clusterNetworkOperator` defines the settings related to the OpenShift Cluster Network Operator, when available.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>forceSampleAll</b></td>
        <td>boolean</td>
        <td>
          `forceSampleAll` allows disabling sampling in the IPFIX-based flow reporter.
It is not recommended to sample all the traffic with IPFIX, as it might generate cluster instability.
If you REALLY want to do that, set this flag to `true`. Use at your own risk.
When it is set to `true`, the value of `sampling` is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecagentipfixovnkubernetes">ovnKubernetes</a></b></td>
        <td>object</td>
        <td>
          `ovnKubernetes` defines the settings of the OVN-Kubernetes network plugin, when available. This configuration is used when using OVN's IPFIX exports, without OpenShift. When using OpenShift, refer to the `clusterNetworkOperator` property instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is the sampling interval on the reporter. 100 means one flow on 100 is sent.
To ensure cluster stability, it is not possible to set a value below 2.
If you really want to sample every packet, which might impact the cluster stability,
refer to `forceSampleAll`. Alternatively, you can use the eBPF Agent instead of IPFIX.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 400<br/>
            <i>Minimum</i>: 2<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ipfix.clusterNetworkOperator
<sup><sup>[↩ Parent](#flowcollectorspecagentipfix)</sup></sup>



`clusterNetworkOperator` defines the settings related to the OpenShift Cluster Network Operator, when available.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace  where the config map is going to be deployed.<br/>
          <br/>
            <i>Default</i>: openshift-network-operator<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.agent.ipfix.ovnKubernetes
<sup><sup>[↩ Parent](#flowcollectorspecagentipfix)</sup></sup>



`ovnKubernetes` defines the settings of the OVN-Kubernetes network plugin, when available. This configuration is used when using OVN's IPFIX exports, without OpenShift. When using OpenShift, refer to the `clusterNetworkOperator` property instead.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>
          `containerName` defines the name of the container to configure for IPFIX.<br/>
          <br/>
            <i>Default</i>: ovnkube-node<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>daemonSetName</b></td>
        <td>string</td>
        <td>
          `daemonSetName` defines the name of the DaemonSet controlling the OVN-Kubernetes pods.<br/>
          <br/>
            <i>Default</i>: ovnkube-node<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace where OVN-Kubernetes pods are deployed.<br/>
          <br/>
            <i>Default</i>: ovn-kubernetes<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`consolePlugin` defines the settings related to the OpenShift Console plugin, when available.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvanced">advanced</a></b></td>
        <td>object</td>
        <td>
          `advanced` allows setting some aspects of the internal configuration of the console plugin.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscaler">autoscaler</a></b></td>
        <td>object</td>
        <td>
          `autoscaler` [deprecated (*)] spec of a horizontal pod autoscaler to set up for the plugin Deployment.
Deprecation notice: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.consolePlugin.unmanagedReplicas` to `true`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enables the console plugin deployment.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>enum</td>
        <td>
          `imagePullPolicy` is the Kubernetes pull policy for the image defined above.<br/>
          <br/>
            <i>Enum</i>: IfNotPresent, Always, Never<br/>
            <i>Default</i>: IfNotPresent<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>enum</td>
        <td>
          `logLevel` for the console plugin backend.<br/>
          <br/>
            <i>Enum</i>: trace, debug, info, warn, error, fatal, panic<br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginportnaming">portNaming</a></b></td>
        <td>object</td>
        <td>
          `portNaming` defines the configuration of the port-to-service name translation.<br/>
          <br/>
            <i>Default</i>: map[enable:true]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginquickfiltersindex">quickFilters</a></b></td>
        <td>[]object</td>
        <td>
          `quickFilters` configures quick filter presets for the Console plugin.
Filters for external traffic assume the subnet labels are configured to distinguish internal and external traffic (see `spec.processor.subnetLabels`).<br/>
          <br/>
            <i>Default</i>: [map[default:true filter:map[flow_layer:"app"] name:Applications] map[filter:map[flow_layer:"infra"] name:Infrastructure] map[default:true filter:map[dst_kind:"Pod" src_kind:"Pod"] name:Pods network] map[filter:map[dst_kind:"Service"] name:Services network] map[filter:map[src_subnet_label:"",EXT:] name:External ingress] map[filter:map[dst_subnet_label:"",EXT:] name:External egress]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          `replicas` defines the number of replicas (pods) to start.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 1<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginresources">resources</a></b></td>
        <td>object</td>
        <td>
          `resources`, in terms of compute resources, required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.<br/>
          <br/>
            <i>Default</i>: map[limits:map[memory:100Mi] requests:map[cpu:100m memory:50Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>standalone</b></td>
        <td>boolean</td>
        <td>
          Deploy as a standalone console, instead of a plugin of the OpenShift Console.
This is not recommended when using with OpenShift, as it doesn't provide an integrated experience.
[Unsupported (*)].<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unmanagedReplicas</b></td>
        <td>boolean</td>
        <td>
          If `unmanagedReplicas` is `true`, the operator will not reconcile `replicas`. This is useful when using a pod autoscaler.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced
<sup><sup>[↩ Parent](#flowcollectorspecconsoleplugin)</sup></sup>



`advanced` allows setting some aspects of the internal configuration of the console plugin.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>args</b></td>
        <td>[]string</td>
        <td>
          `args` allows passing custom arguments to underlying components. Useful for overriding
some parameters, such as a URL or a configuration path, that should not be
publicly exposed as part of the FlowCollector descriptor, as they are only useful
in edge debug or support scenarios.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>env</b></td>
        <td>map[string]string</td>
        <td>
          `env` allows passing custom environment variables to underlying components. Useful for passing
some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
publicly exposed as part of the FlowCollector descriptor, as they are only useful
in edge debug or support scenarios.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          `port` is the plugin service port. Do not use 9002, which is reserved for metrics.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 9001<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>register</b></td>
        <td>boolean</td>
        <td>
          `register` allows, when set to `true`, to automatically register the provided console plugin with the OpenShift Console operator.
When set to `false`, you can still register it manually by editing console.operator.openshift.io/cluster with the following command:
`oc patch console.operator.openshift.io cluster --type='json' -p '[{"op": "add", "path": "/spec/plugins/-", "value": "netobserv-plugin"}]'`<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedscheduling">scheduling</a></b></td>
        <td>object</td>
        <td>
          `scheduling` controls how the pods are scheduled on nodes.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvanced)</sup></sup>



`scheduling` controls how the pods are scheduled on nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          `nodeSelector` allows scheduling of pods only onto nodes that have each of the specified labels.
For documentation, refer to https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          If specified, indicates the pod's priority. For documentation, refer to https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#how-to-use-priority-and-preemption.
If not specified, default priority is used, or zero if there is no default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingtolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedscheduling)</sup></sup>



If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node matches the corresponding matchExpressions; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0
(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of
them are ANDed.
The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the anti-affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling anti-affinity expressions, etc.),
compute a sum by iterating through the elements of this field and subtracting
"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the anti-affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.advanced.scheduling.tolerations[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginadvancedscheduling)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects.
When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys.
If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value.
Valid operators are Exists, Equal, Lt, and Gt. Defaults to Equal.
Exists is equivalent to wildcard for value, so that a pod can
tolerate all taints of a particular category.
Lt and Gt perform numeric comparisons (requires feature gate TaintTolerationComparisonOperators).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be
of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
it is not set, which means tolerate the taint forever (do not evict). Zero and
negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to.
If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler
<sup><sup>[↩ Parent](#flowcollectorspecconsoleplugin)</sup></sup>



`autoscaler` [deprecated (*)] spec of a horizontal pod autoscaler to set up for the plugin Deployment.
Deprecation notice: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.consolePlugin.unmanagedReplicas` to `true`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxReplicas</b></td>
        <td>integer</td>
        <td>
          `maxReplicas` is the upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 3<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindex">metrics</a></b></td>
        <td>[]object</td>
        <td>
          Metrics used by the pod autoscaler. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/horizontal-pod-autoscaler-v2/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minReplicas</b></td>
        <td>integer</td>
        <td>
          `minReplicas` is the lower limit for the number of replicas to which the autoscaler
can scale down. It defaults to 1 pod. minReplicas is allowed to be 0 if the
alpha feature gate HPAScaleToZero is enabled and at least one Object or External
metric is configured. Scaling is active as long as at least one metric value is
available.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          `status` describes the desired status regarding deploying an horizontal pod autoscaler.<br>
- `Disabled` does not deploy an horizontal pod autoscaler.<br>
- `Enabled` deploys an horizontal pod autoscaler.<br><br/>
          <br/>
            <i>Enum</i>: Disabled, Enabled<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscaler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexcontainerresource">containerResource</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexexternal">external</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobject">object</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexpods">pods</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexresource">resource</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].containerResource
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>container</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexcontainerresourcetarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].containerResource.target
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexcontainerresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].external
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexexternalmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexexternaltarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].external.metric
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexexternal)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexexternalmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].external.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexexternalmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexexternalmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].external.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexexternalmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].external.target
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexexternal)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobjectdescribedobject">describedObject</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobjectmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobjecttarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object.describedObject
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object.metric
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobjectmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexobjectmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexobjectmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexobjectmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].object.target
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].pods
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexpodsmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexpodstarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].pods.metric
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexpods)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexpodsmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].pods.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexpodsmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexpodsmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].pods.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexpodsmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].pods.target
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexpods)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].resource
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecconsolepluginautoscalermetricsindexresourcetarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.autoscaler.metrics[index].resource.target
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginautoscalermetricsindexresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.portNaming
<sup><sup>[↩ Parent](#flowcollectorspecconsoleplugin)</sup></sup>



`portNaming` defines the configuration of the port-to-service name translation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable the console plugin port-to-service name translation<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>portNames</b></td>
        <td>map[string]string</td>
        <td>
          `portNames` defines additional port names to use in the console,
for example, `portNames: {"3100": "loki"}`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.quickFilters[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsoleplugin)</sup></sup>



`QuickFilter` defines preset configuration for Console's quick filters

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>filter</b></td>
        <td>map[string]string</td>
        <td>
          `filter` is a set of keys and values to be set when this filter is selected. Each key can relate to a list of values using a coma-separated string,
for example, `filter: {"src_namespace": "namespace1,namespace2"}`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the filter, that is displayed in the Console<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>default</b></td>
        <td>boolean</td>
        <td>
          `default` defines whether this filter should be active by default or not<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.resources
<sup><sup>[↩ Parent](#flowcollectorspecconsoleplugin)</sup></sup>



`resources`, in terms of compute resources, required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecconsolepluginresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims,
that are used by this container.

This field depends on the
DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.consolePlugin.resources.claims[index]
<sup><sup>[↩ Parent](#flowcollectorspecconsolepluginresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of
the Pod where this field is used. It makes that resource available
inside a container.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>request</b></td>
        <td>string</td>
        <td>
          Request is the name chosen for a request in the referenced claim.
If empty, everything from the claim is made available, otherwise
only the result of this request.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.execution
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`execution` defines configuration related to the execution of the flow collection process.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          `mode` is the flow collection process execution desired mode: `Running` or `OnHold`.
When `OnHold`, the operator deletes all managed services and workloads, with the exception
of the static console plugin, and the operator itself.
It allows to use minimal cluster resources without losing configuration.<br/>
          <br/>
            <i>Enum</i>: , Running, OnHold<br/>
            <i>Default</i>: Running<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index]
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`FlowCollectorExporter` defines an additional exporter to send enriched flows to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          `type` selects the type of exporters. The available options are `Kafka`, `IPFIX`, and `OpenTelemetry`.<br/>
          <br/>
            <i>Enum</i>: Kafka, IPFIX, OpenTelemetry<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexipfix">ipfix</a></b></td>
        <td>object</td>
        <td>
          IPFIX configuration, such as the IP address and port to send enriched IPFIX flows to.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafka">kafka</a></b></td>
        <td>object</td>
        <td>
          Kafka configuration, such as the address and topic, to send enriched flows to.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetry">openTelemetry</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetry configuration, such as the IP address and port to send enriched logs or metrics to.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].ipfix
<sup><sup>[↩ Parent](#flowcollectorspecexportersindex)</sup></sup>



IPFIX configuration, such as the IP address and port to send enriched IPFIX flows to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enterpriseID</b></td>
        <td>integer</td>
        <td>
          EnterpriseID, or Private Enterprise Number (PEN). To date, NetObserv does not own an assigned number,
so it is left open for configuration. The PEN is needed to collect non standard data, such as Kubernetes names,
RTT, etc.<br/>
          <br/>
            <i>Default</i>: 2<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>targetHost</b></td>
        <td>string</td>
        <td>
          Address of the IPFIX external receiver.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>targetPort</b></td>
        <td>integer</td>
        <td>
          Port for the IPFIX external receiver.<br/>
          <br/>
            <i>Default</i>: 4739<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>transport</b></td>
        <td>enum</td>
        <td>
          Transport protocol (`TCP` or `UDP`) to be used for the IPFIX connection, defaults to `TCP`.<br/>
          <br/>
            <i>Enum</i>: TCP, UDP<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka
<sup><sup>[↩ Parent](#flowcollectorspecexportersindex)</sup></sup>



Kafka configuration, such as the address and topic, to send enriched flows to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>address</b></td>
        <td>string</td>
        <td>
          Address of the Kafka server<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topic</b></td>
        <td>string</td>
        <td>
          Kafka topic to use. It must exist. NetObserv does not create it.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkasasl">sasl</a></b></td>
        <td>object</td>
        <td>
          SASL authentication configuration. [Unsupported (*)].<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkatls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.sasl
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafka)</sup></sup>



SASL authentication configuration. [Unsupported (*)].

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkasaslclientidreference">clientIDReference</a></b></td>
        <td>object</td>
        <td>
          Reference to the secret or config map containing the client ID<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkasaslclientsecretreference">clientSecretReference</a></b></td>
        <td>object</td>
        <td>
          Reference to the secret or config map containing the client secret<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type of SASL authentication to use, or `Disabled` if SASL is not used<br/>
          <br/>
            <i>Enum</i>: Disabled, Plain, ScramSHA512<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.sasl.clientIDReference
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafkasasl)</sup></sup>



Reference to the secret or config map containing the client ID

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.sasl.clientSecretReference
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafkasasl)</sup></sup>



Reference to the secret or config map containing the client secret

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.tls
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafka)</sup></sup>



TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkatlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkatlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafkatls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafkatls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry
<sup><sup>[↩ Parent](#flowcollectorspecexportersindex)</sup></sup>



OpenTelemetry configuration, such as the IP address and port to send enriched logs or metrics to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>targetHost</b></td>
        <td>string</td>
        <td>
          Address of the OpenTelemetry receiver.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>targetPort</b></td>
        <td>integer</td>
        <td>
          Port for the OpenTelemetry receiver.<br/>
          <br/>
            <i>Default</i>: 4317<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetryfieldsmappingindex">fieldsMapping</a></b></td>
        <td>[]object</td>
        <td>
          Custom fields mapping to an OpenTelemetry conformant format.
By default, NetObserv format proposal is used: https://github.com/rhobs/observability-data-model/blob/main/network-observability.md#format-proposal .
As there is currently no accepted standard for L3 or L4 enriched network logs, you can freely override it with your own.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          Headers to add to messages (optional)<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetrylogs">logs</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetry configuration for logs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetrymetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          OpenTelemetry configuration for metrics.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>protocol</b></td>
        <td>enum</td>
        <td>
          Protocol of the OpenTelemetry connection. The available options are `http` and `grpc`.<br/>
          <br/>
            <i>Enum</i>: http, grpc<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetrytls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.fieldsMapping[index]
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetry)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>input</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>multiplier</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>output</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.logs
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetry)</sup></sup>



OpenTelemetry configuration for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Set `enable` to `true` to send logs to an OpenTelemetry receiver.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.metrics
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetry)</sup></sup>



OpenTelemetry configuration for metrics.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Set `enable` to `true` to send metrics to an OpenTelemetry receiver.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>pushTimeInterval</b></td>
        <td>string</td>
        <td>
          Specify how often metrics are sent to a collector.<br/>
          <br/>
            <i>Default</i>: 20s<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.tls
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetry)</sup></sup>



TLS client configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetrytlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexopentelemetrytlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetrytls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].openTelemetry.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexopentelemetrytls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



Kafka configuration, allowing to use Kafka as a broker as part of the flow collection pipeline. Available when the `spec.deploymentModel` is `Kafka`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>address</b></td>
        <td>string</td>
        <td>
          Address of the Kafka server<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>topic</b></td>
        <td>string</td>
        <td>
          Kafka topic to use. It must exist. NetObserv does not create it.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkasasl">sasl</a></b></td>
        <td>object</td>
        <td>
          SASL authentication configuration. [Unsupported (*)].<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkatls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.sasl
<sup><sup>[↩ Parent](#flowcollectorspeckafka)</sup></sup>



SASL authentication configuration. [Unsupported (*)].

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeckafkasaslclientidreference">clientIDReference</a></b></td>
        <td>object</td>
        <td>
          Reference to the secret or config map containing the client ID<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkasaslclientsecretreference">clientSecretReference</a></b></td>
        <td>object</td>
        <td>
          Reference to the secret or config map containing the client secret<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type of SASL authentication to use, or `Disabled` if SASL is not used<br/>
          <br/>
            <i>Enum</i>: Disabled, Plain, ScramSHA512<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.sasl.clientIDReference
<sup><sup>[↩ Parent](#flowcollectorspeckafkasasl)</sup></sup>



Reference to the secret or config map containing the client ID

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.sasl.clientSecretReference
<sup><sup>[↩ Parent](#flowcollectorspeckafkasasl)</sup></sup>



Reference to the secret or config map containing the client secret

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.tls
<sup><sup>[↩ Parent](#flowcollectorspeckafka)</sup></sup>



TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeckafkatlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkatlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspeckafkatls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspeckafkatls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`loki`, the flow store, client settings.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          `mode` must be set according to the installation mode of Loki:<br>
- Use `LokiStack` when Loki is managed using the Loki Operator<br>
- Use `Monolithic` when Loki is installed as a monolithic workload<br>
- Use `Microservices` when Loki is installed as microservices, but without Loki Operator<br>
- Use `Manual` if none of the options above match your setup<br><br/>
          <br/>
            <i>Enum</i>: Manual, LokiStack, Monolithic, Microservices<br/>
            <i>Default</i>: Monolithic<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokiadvanced">advanced</a></b></td>
        <td>object</td>
        <td>
          `advanced` allows setting some aspects of the internal configuration of the Loki clients.
This section is aimed mostly for debugging and fine-grained performance optimizations.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Set `enable` to `true` to store flows in Loki.
The Console plugin can use either Loki or Prometheus as a data source for metrics (see also `spec.prometheus.querier`), or both.
Not all queries are transposable from Loki to Prometheus. Hence, if Loki is disabled, some features of the plugin are disabled as well,
such as getting per-pod information or viewing raw flows.
If both Prometheus and Loki are enabled, Prometheus takes precedence and Loki is used as a fallback for queries that Prometheus cannot handle.
If they are both disabled, the Console plugin is not deployed.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokilokistack">lokiStack</a></b></td>
        <td>object</td>
        <td>
          Loki configuration for `LokiStack` mode. This is useful for an easy Loki Operator configuration.
It is ignored for other modes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimanual">manual</a></b></td>
        <td>object</td>
        <td>
          Loki configuration for `Manual` mode. This is the most flexible configuration.
It is ignored for other modes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimicroservices">microservices</a></b></td>
        <td>object</td>
        <td>
          Loki configuration for `Microservices` mode.
Use this option when Loki is installed using the microservices deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#microservices-mode).
It is ignored for other modes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimonolithic">monolithic</a></b></td>
        <td>object</td>
        <td>
          Loki configuration for `Monolithic` mode.
Use this option when Loki is installed using the monolithic deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#monolithic-mode).
It is ignored for other modes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readTimeout</b></td>
        <td>string</td>
        <td>
          `readTimeout` is the maximum console plugin loki query total time limit.
A timeout of zero means no timeout.<br/>
          <br/>
            <i>Default</i>: 30s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeBatchSize</b></td>
        <td>integer</td>
        <td>
          `writeBatchSize` is the maximum batch size (in bytes) of Loki logs to accumulate before sending.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Default</i>: 10485760<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeBatchWait</b></td>
        <td>string</td>
        <td>
          `writeBatchWait` is the maximum time to wait before sending a Loki batch.<br/>
          <br/>
            <i>Default</i>: 1s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeTimeout</b></td>
        <td>string</td>
        <td>
          `writeTimeout` is the maximum Loki time connection / request limit.
A timeout of zero means no timeout.<br/>
          <br/>
            <i>Default</i>: 10s<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.advanced
<sup><sup>[↩ Parent](#flowcollectorspecloki)</sup></sup>



`advanced` allows setting some aspects of the internal configuration of the Loki clients.
This section is aimed mostly for debugging and fine-grained performance optimizations.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>excludeLabels</b></td>
        <td>[]string</td>
        <td>
          `excludeLabels` is a list of fields to be excluded from the list of Loki labels. [Unsupported (*)].<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>staticLabels</b></td>
        <td>map[string]string</td>
        <td>
          `staticLabels` is a map of common labels to set on each flow in Loki storage.<br/>
          <br/>
            <i>Default</i>: map[app:netobserv-flowcollector]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeMaxBackoff</b></td>
        <td>string</td>
        <td>
          `writeMaxBackoff` is the maximum backoff time for Loki client connection between retries.<br/>
          <br/>
            <i>Default</i>: 5s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeMaxRetries</b></td>
        <td>integer</td>
        <td>
          `writeMaxRetries` is the maximum number of retries for Loki client connections.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 2<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>writeMinBackoff</b></td>
        <td>string</td>
        <td>
          `writeMinBackoff` is the initial backoff time for Loki client connection between retries.<br/>
          <br/>
            <i>Default</i>: 1s<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.lokiStack
<sup><sup>[↩ Parent](#flowcollectorspecloki)</sup></sup>



Loki configuration for `LokiStack` mode. This is useful for an easy Loki Operator configuration.
It is ignored for other modes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of an existing LokiStack resource to use.<br/>
          <br/>
            <i>Default</i>: loki<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace where this `LokiStack` resource is located. If omitted, it is assumed to be the same as `spec.namespace`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual
<sup><sup>[↩ Parent](#flowcollectorspecloki)</sup></sup>



Loki configuration for `Manual` mode. This is the most flexible configuration.
It is ignored for other modes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>authToken</b></td>
        <td>enum</td>
        <td>
          `authToken` describes the way to get a token to authenticate to Loki.<br>
- `Disabled` does not send any token with the request.<br>
- `Forward` forwards the user token for authorization.<br>
- `Host` [deprecated (*)] - uses the local pod service account to authenticate to Loki.<br>
When using the Loki Operator, this must be set to `Forward`.<br/>
          <br/>
            <i>Enum</i>: Disabled, Host, Forward<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ingesterUrl</b></td>
        <td>string</td>
        <td>
          `ingesterUrl` is the address of an existing Loki ingester service to push the flows to. When using the Loki Operator,
set it to the Loki gateway service with the `network` tenant set in path, for example
https://loki-gateway-http.netobserv.svc:8080/api/logs/v1/network.<br/>
          <br/>
            <i>Default</i>: http://loki:3100/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>querierUrl</b></td>
        <td>string</td>
        <td>
          `querierUrl` specifies the address of the Loki querier service.
When using the Loki Operator, set it to the Loki gateway service with the `network` tenant set in path, for example
https://loki-gateway-http.netobserv.svc:8080/api/logs/v1/network.<br/>
          <br/>
            <i>Default</i>: http://loki:3100/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimanualstatustls">statusTls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Loki status URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>statusUrl</b></td>
        <td>string</td>
        <td>
          `statusUrl` specifies the address of the Loki `/ready`, `/metrics` and `/config` endpoints, in case it is different from the
Loki querier URL. If empty, the `querierUrl` value is used.
This is useful to show error messages and some context in the frontend.
When using the Loki Operator, set it to the Loki HTTP query frontend service, for example
https://loki-query-frontend-http.netobserv.svc:3100/.
`statusTLS` configuration is used when `statusUrl` is set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tenantID</b></td>
        <td>string</td>
        <td>
          `tenantID` is the Loki `X-Scope-OrgID` that identifies the tenant for each request.
When using the Loki Operator, set it to `network`, which corresponds to a special tenant mode.<br/>
          <br/>
            <i>Default</i>: netobserv<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimanualtls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Loki URL.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.statusTls
<sup><sup>[↩ Parent](#flowcollectorspeclokimanual)</sup></sup>



TLS client configuration for Loki status URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeclokimanualstatustlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimanualstatustlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.statusTls.caCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimanualstatustls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.statusTls.userCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimanualstatustls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.tls
<sup><sup>[↩ Parent](#flowcollectorspeclokimanual)</sup></sup>



TLS client configuration for Loki URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeclokimanualtlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimanualtlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimanualtls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.manual.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimanualtls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.microservices
<sup><sup>[↩ Parent](#flowcollectorspecloki)</sup></sup>



Loki configuration for `Microservices` mode.
Use this option when Loki is installed using the microservices deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#microservices-mode).
It is ignored for other modes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ingesterUrl</b></td>
        <td>string</td>
        <td>
          `ingesterUrl` is the address of an existing Loki ingester service to push the flows to.<br/>
          <br/>
            <i>Default</i>: http://loki-distributor:3100/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>querierUrl</b></td>
        <td>string</td>
        <td>
          `querierURL` specifies the address of the Loki querier service.<br/>
          <br/>
            <i>Default</i>: http://loki-query-frontend:3100/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tenantID</b></td>
        <td>string</td>
        <td>
          `tenantID` is the Loki `X-Scope-OrgID` header that identifies the tenant for each request.<br/>
          <br/>
            <i>Default</i>: netobserv<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimicroservicestls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Loki URL.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.microservices.tls
<sup><sup>[↩ Parent](#flowcollectorspeclokimicroservices)</sup></sup>



TLS client configuration for Loki URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeclokimicroservicestlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimicroservicestlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.microservices.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimicroservicestls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.microservices.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimicroservicestls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.monolithic
<sup><sup>[↩ Parent](#flowcollectorspecloki)</sup></sup>



Loki configuration for `Monolithic` mode.
Use this option when Loki is installed using the monolithic deployment mode (https://grafana.com/docs/loki/latest/fundamentals/architecture/deployment-modes/#monolithic-mode).
It is ignored for other modes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>installDemoLoki</b></td>
        <td>boolean</td>
        <td>
          Set `installDemoLoki` to `true` to automatically create Loki deployment, service and storage.
This is useful for development and demo purposes. Do not use it in production.
[Unsupported (*)].<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tenantID</b></td>
        <td>string</td>
        <td>
          `tenantID` is the Loki `X-Scope-OrgID` header that identifies the tenant for each request.<br/>
          <br/>
            <i>Default</i>: netobserv<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimonolithictls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Loki URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          `url` is the unique address of an existing Loki service that points to both the ingester and the querier.<br/>
          <br/>
            <i>Default</i>: http://loki:3100/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.monolithic.tls
<sup><sup>[↩ Parent](#flowcollectorspeclokimonolithic)</sup></sup>



TLS client configuration for Loki URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspeclokimonolithictlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeclokimonolithictlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.monolithic.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimonolithictls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.loki.monolithic.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspeclokimonolithictls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.networkPolicy
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`networkPolicy` defines network policy settings for NetObserv components isolation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>additionalNamespaces</b></td>
        <td>[]string</td>
        <td>
          `additionalNamespaces` contains additional namespaces allowed to connect to the NetObserv namespace.
It provides flexibility in the network policy configuration, but if you need a more specific
configuration, you can disable it and install your own instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Deploys network policies on the namespaces used by NetObserv (main and privileged).
These network policies better isolate the NetObserv components to prevent undesired connections from and to them.
Because it cannot be tested with all CNIs, this option is only enabled by default when NetObserv runs in a known
supported environment, and it is disabled by default otherwise.
When disabled, it is highly recommended to create network policies manually, to prevent undesired accesses.
More information: https://github.com/netobserv/netobserv-operator/blob/main/docs/NetworkPolicy.md.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`processor` defines the settings of the component that receives the flows from the agent,
enriches them, generates metrics, and forwards them to the Loki persistence layer and/or any available exporter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>addZone</b></td>
        <td>boolean</td>
        <td>
          `addZone` allows availability zone awareness by labeling flows with their source and destination zones.
This feature requires the "topology.kubernetes.io/zone" label to be set on nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvanced">advanced</a></b></td>
        <td>object</td>
        <td>
          `advanced` allows setting some aspects of the internal configuration of the flow processor.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterName</b></td>
        <td>string</td>
        <td>
          `clusterName` is the name of the cluster to appear in the flows data. This is useful in a multi-cluster context. When using OpenShift, leave empty to make it automatically determined.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>consumerReplicas</b></td>
        <td>integer</td>
        <td>
          `consumerReplicas` defines the number of replicas (pods) to start for `flowlogs-pipeline`, default is 3.
This setting is ignored when `spec.deploymentModel` is `Direct` or when `spec.processor.unmanagedReplicas` is `true`.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessordeduper">deduper</a></b></td>
        <td>object</td>
        <td>
          `deduper` allows you to sample or drop flows identified as duplicates, in order to save on resource usage.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorfiltersindex">filters</a></b></td>
        <td>[]object</td>
        <td>
          `filters` lets you define custom filters to limit the amount of generated flows.
These filters provide more flexibility than the eBPF Agent filters (in `spec.agent.ebpf.flowFilter`), such as allowing to filter by Kubernetes namespace,
but with a lesser improvement in performance.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>enum</td>
        <td>
          `imagePullPolicy` is the Kubernetes pull policy for the image defined above<br/>
          <br/>
            <i>Enum</i>: IfNotPresent, Always, Never<br/>
            <i>Default</i>: IfNotPresent<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscaler">kafkaConsumerAutoscaler</a></b></td>
        <td>object</td>
        <td>
          `kafkaConsumerAutoscaler` [deprecated (*)] is the spec of a horizontal pod autoscaler to set up for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
This setting is ignored when Kafka is disabled.
Deprecation notice: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.processor.unmanagedReplicas` to `true`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kafkaConsumerBatchSize</b></td>
        <td>integer</td>
        <td>
          `kafkaConsumerBatchSize` indicates to the broker the maximum batch size, in bytes, that the consumer accepts. Ignored when not using Kafka. Default: 10MB.<br/>
          <br/>
            <i>Default</i>: 10485760<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kafkaConsumerQueueCapacity</b></td>
        <td>integer</td>
        <td>
          `kafkaConsumerQueueCapacity` defines the capacity of the internal message queue used in the Kafka consumer client. Ignored when not using Kafka.<br/>
          <br/>
            <i>Default</i>: 1000<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kafkaConsumerReplicas</b></td>
        <td>integer</td>
        <td>
          `kafkaConsumerReplicas` [deprecated (*)] defines the number of replicas (pods) to start for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
This setting is ignored when Kafka is disabled.
Deprecation notice: use `spec.processor.consumerReplicas` instead.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 3<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logLevel</b></td>
        <td>enum</td>
        <td>
          `logLevel` of the processor runtime<br/>
          <br/>
            <i>Enum</i>: trace, debug, info, warn, error, fatal, panic<br/>
            <i>Default</i>: info<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>logTypes</b></td>
        <td>enum</td>
        <td>
          `logTypes` defines the desired record types to generate. Possible values are:<br>
- `Flows` to export regular network flows. This is the default.<br>
- `Conversations` to generate events for started conversations, ended conversations as well as periodic "tick" updates. Note that in this mode, Prometheus metrics are not accurate on long-standing conversations.<br>
- `EndedConversations` to generate only ended conversations events. Note that in this mode, Prometheus metrics are not accurate on long-standing conversations.<br>
- `All` to generate both network flows and all conversations events. It is not recommended due to the impact on resources footprint.<br><br/>
          <br/>
            <i>Enum</i>: Flows, Conversations, EndedConversations, All<br/>
            <i>Default</i>: Flows<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetrics">metrics</a></b></td>
        <td>object</td>
        <td>
          `Metrics` define the processor configuration regarding metrics<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>multiClusterDeployment</b></td>
        <td>boolean</td>
        <td>
          Set `multiClusterDeployment` to `true` to enable multi clusters feature. This adds `clusterName` label to flows data<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorresources">resources</a></b></td>
        <td>object</td>
        <td>
          `resources` are the compute resources required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
          <br/>
            <i>Default</i>: map[limits:map[memory:800Mi] requests:map[cpu:100m memory:100Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorservice">service</a></b></td>
        <td>object</td>
        <td>
          Service configuration, only used when `spec.deploymentModel` is `Service`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorslicesconfig">slicesConfig</a></b></td>
        <td>object</td>
        <td>
          Global configuration managing FlowCollectorSlices custom resources.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorsubnetlabels">subnetLabels</a></b></td>
        <td>object</td>
        <td>
          `subnetLabels` allows to define custom labels on subnets and IPs or to enable automatic labeling of recognized subnets in OpenShift, which is used to identify cluster external traffic.
When a subnet matches the source or destination IP of a flow, a corresponding field is added: `SrcSubnetLabel` or `DstSubnetLabel`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unmanagedReplicas</b></td>
        <td>boolean</td>
        <td>
          If `unmanagedReplicas` is `true`, the operator will not reconcile `consumerReplicas`. This is useful when using a pod autoscaler.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`advanced` allows setting some aspects of the internal configuration of the flow processor.
This section is aimed mostly for debugging and fine-grained performance optimizations,
such as `GOGC` and `GOMAXPROCS` environment variables. Set these values at your own risk.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>conversationEndTimeout</b></td>
        <td>string</td>
        <td>
          `conversationEndTimeout` is the time to wait after a network flow is received, to consider the conversation ended.
This delay is ignored when a FIN packet is collected for TCP flows (see `conversationTerminatingTimeout` instead).<br/>
          <br/>
            <i>Default</i>: 10s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>conversationHeartbeatInterval</b></td>
        <td>string</td>
        <td>
          `conversationHeartbeatInterval` is the time to wait between "tick" events of a conversation<br/>
          <br/>
            <i>Default</i>: 30s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>conversationTerminatingTimeout</b></td>
        <td>string</td>
        <td>
          `conversationTerminatingTimeout` is the time to wait from detected FIN flag to end a conversation. Only relevant for TCP flows.<br/>
          <br/>
            <i>Default</i>: 5s<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>dropUnusedFields</b></td>
        <td>boolean</td>
        <td>
          `dropUnusedFields` [deprecated (*)] this setting is not used anymore.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableKubeProbes</b></td>
        <td>boolean</td>
        <td>
          `enableKubeProbes` is a flag to enable or disable Kubernetes liveness and readiness probes<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>env</b></td>
        <td>map[string]string</td>
        <td>
          `env` allows passing custom environment variables to underlying components. Useful for passing
some very concrete performance-tuning options, such as `GOGC` and `GOMAXPROCS`, that should not be
publicly exposed as part of the FlowCollector descriptor, as they are only useful
in edge debug or support scenarios.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>healthPort</b></td>
        <td>integer</td>
        <td>
          `healthPort` is a collector HTTP port in the Pod that exposes the health check API<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 8080<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          Port of the flow collector (host port).
By convention, some values are forbidden. It must be greater than 1024 and different from
4500, 4789 and 6081.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 2055<br/>
            <i>Minimum</i>: 1025<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>profilePort</b></td>
        <td>integer</td>
        <td>
          `profilePort` allows setting up a Go pprof profiler listening to this port<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 0<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedscheduling">scheduling</a></b></td>
        <td>object</td>
        <td>
          scheduling controls how the pods are scheduled on nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedsecondarynetworksindex">secondaryNetworks</a></b></td>
        <td>[]object</td>
        <td>
          Defines secondary networks to be checked for resources identification.
To guarantee a correct identification, indexed values must form an unique identifier across the cluster.
If the same index is used by several resources, those resources might be incorrectly labeled.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvanced)</sup></sup>



scheduling controls how the pods are scheduled on nodes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinity">affinity</a></b></td>
        <td>object</td>
        <td>
          If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>nodeSelector</b></td>
        <td>map[string]string</td>
        <td>
          `nodeSelector` allows scheduling of pods only onto nodes that have each of the specified labels.
For documentation, refer to https://kubernetes.io/docs/concepts/configuration/assign-pod-node/.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>
          If specified, indicates the pod's priority. For documentation, refer to https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/#how-to-use-priority-and-preemption.
If not specified, default priority is used, or zero if there is no default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingtolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedscheduling)</sup></sup>



If specified, the pod's scheduling constraints. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes node affinity scheduling rules for the pod.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>
          Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinity)</sup></sup>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node matches the corresponding matchExpressions; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinity)</sup></sup>



An empty preferred scheduling term matches all objects with implicit weight 0
(i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>
          A node selector term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinity)</sup></sup>



If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to an update), the system
may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>
          Required. A list of node selector terms. The terms are ORed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecution)</sup></sup>



A null or empty node selector term matches no objects. The requirements of
them are ANDed.
The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>
          A list of node selector requirements by node's fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex)</sup></sup>



A node selector requirement is a selector that contains values, a key, and an operator
that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          An array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. If the operator is Gt or Lt, the values
array must have a single element, which will be interpreted as an integer.
This array is replaced during a strategic merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinity)</sup></sup>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling affinity expressions, etc.),
compute a sum by iterating through the elements of this field and adding
"weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinity)</sup></sup>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          The scheduler will prefer to schedule pods to nodes that satisfy
the anti-affinity expressions specified by this field, but it may choose
a node that violates one or more of the expressions. The node that is
most preferred is the one with the greatest sum of weights, i.e.
for each node that meets all of the scheduling requirements (resource
request, requiredDuringScheduling anti-affinity expressions, etc.),
compute a sum by iterating through the elements of this field and subtracting
"weight" from the sum if the node has pods which matches the corresponding podAffinityTerm; the
node(s) with the highest sum are the most preferred.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>
          If the anti-affinity requirements specified by this field are not met at
scheduling time, the pod will not be scheduled onto the node.
If the anti-affinity requirements specified by this field cease to be met
at some point during pod execution (e.g. due to a pod label update), the
system may or may not try to eventually evict the pod from its node.
When there are multiple elements, the lists of nodes corresponding to each
podAffinityTerm are intersected, i.e. all terms must be satisfied.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinity)</sup></sup>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>
          Required. A pod affinity term, associated with the corresponding weight.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>
          weight associated with matching the corresponding podAffinityTerm,
in the range 1-100.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex)</sup></sup>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinity)</sup></sup>



Defines a set of pods (namely those matching the labelSelector
relative to the given namespace(s)) that this pod should be
co-located (affinity) or not co-located (anti-affinity) with,
where co-located is defined as running on a node whose value of
the label with key <topologyKey> matches that of any node on which
a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>
          This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching
the labelSelector in the specified namespaces, where co-located is defined as running on a node
whose value of the label with key topologyKey matches that of any node on which any of the
selected pods is running.
Empty topologyKey is not allowed.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key in (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both matchLabelKeys and labelSelector.
Also, matchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mismatchLabelKeys</b></td>
        <td>[]string</td>
        <td>
          MismatchLabelKeys is a set of pod label keys to select which pods will
be taken into consideration. The keys are used to lookup values from the
incoming pod labels, those key-value labels are merged with `labelSelector` as `key notin (value)`
to select the group of existing pods which pods will be taken into consideration
for the incoming pod's pod (anti) affinity. Keys that don't exist in the incoming
pod labels will be ignored. The default value is empty.
The same key is forbidden to exist in both mismatchLabelKeys and labelSelector.
Also, mismatchLabelKeys cannot be set when labelSelector isn't set.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector">namespaceSelector</a></b></td>
        <td>object</td>
        <td>
          A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>
          namespaces specifies a static list of namespace names that the term applies to.
The term is applied to the union of the namespaces listed in this field
and the ones selected by namespaceSelector.
null or empty namespaces list and null namespaceSelector means "this pod's namespace".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over a set of resources, in this case pods.
If it's null, this PodAffinityTerm matches with no Pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex)</sup></sup>



A label query over the set of namespaces that the term applies to.
The term is applied to the union of the namespaces selected by this field
and the ones listed in the namespaces field.
null selector and null or empty namespaces list means "this pod's namespace".
An empty selector ({}) matches all namespaces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          matchExpressions is a list of label selector requirements. The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels
map is equivalent to an element of matchExpressions, whose key field is "key", the
operator is "In", and the values array contains only "value". The requirements are ANDed.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].namespaceSelector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedschedulingaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexnamespaceselector)</sup></sup>



A label selector requirement is a selector that contains values, a key, and an operator that
relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          key is the label key that the selector applies to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          operator represents a key's relationship to a set of values.
Valid operators are In, NotIn, Exists and DoesNotExist.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          values is an array of string values. If the operator is In or NotIn,
the values array must be non-empty. If the operator is Exists or DoesNotExist,
the values array must be empty. This array is replaced during a strategic
merge patch.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling.tolerations[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvancedscheduling)</sup></sup>



The pod this Toleration is attached to tolerates any taint that matches
the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>
          Effect indicates the taint effect to match. Empty means match all taint effects.
When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          Key is the taint key that the toleration applies to. Empty means match all taint keys.
If the key is empty, operator must be Exists; this combination means to match all values and all keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          Operator represents a key's relationship to the value.
Valid operators are Exists, Equal, Lt, and Gt. Defaults to Equal.
Exists is equivalent to wildcard for value, so that a pod can
tolerate all taints of a particular category.
Lt and Gt perform numeric comparisons (requires feature gate TaintTolerationComparisonOperators).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>
          TolerationSeconds represents the period of time the toleration (which must be
of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default,
it is not set, which means tolerate the taint forever (do not evict). Zero and
negative values will be treated as 0 (evict immediately) by the system.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the taint value the toleration matches to.
If the operator is Exists, the value should be empty, otherwise just a regular string.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.secondaryNetworks[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvanced)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>index</b></td>
        <td>[]enum</td>
        <td>
          `index` is a list of fields to use for indexing the pods. They should form a unique Pod identifier across the cluster.
Can be any of: `MAC`, `IP`, `Interface`.
Fields absent from the 'k8s.v1.cni.cncf.io/network-status' annotation must not be added to the index.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          `name` should match the network name as visible in the pods annotation 'k8s.v1.cni.cncf.io/network-status'.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.deduper
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`deduper` allows you to sample or drop flows identified as duplicates, in order to save on resource usage.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          Set the Processor de-duplication mode. It comes in addition to the Agent-based deduplication, since the Agent cannot de-duplicate same flows reported from different nodes.<br>
- Use `Drop` to drop every flow considered as duplicates, allowing saving more on resource usage but potentially losing some information such as the network interfaces used from peer, or network events.<br>
- Use `Sample` to randomly keep only one flow on 50, which is the default, among the ones considered as duplicates. This is a compromise between dropping every duplicate or keeping every duplicate. This sampling action comes in addition to the Agent-based sampling. If both Agent and Processor sampling values are `50`, the combined sampling is 1:2500.<br>
- Use `Disabled` to turn off Processor-based de-duplication.<br><br/>
          <br/>
            <i>Enum</i>: Disabled, Drop, Sample<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is the sampling interval when deduper `mode` is `Sample`. For example, a value of `50` means that 1 flow in 50 is sampled.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 50<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.filters[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`FLPFilterSet` defines the desired configuration for FLP-based filtering satisfying all conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>outputTarget</b></td>
        <td>enum</td>
        <td>
          If specified, these filters target a single output: `Loki`, `Metrics` or `Exporters`. By default, all outputs are targeted.<br/>
          <br/>
            <i>Enum</i>: , Loki, Metrics, Exporters<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>query</b></td>
        <td>string</td>
        <td>
          A query that selects the network flows to keep. More information about this query language in https://github.com/netobserv/flowlogs-pipeline/blob/main/docs/filtering.md.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is an optional sampling interval to apply to this filter. For example, a value of `50` means that 1 matching flow in 50 is sampled.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`kafkaConsumerAutoscaler` [deprecated (*)] is the spec of a horizontal pod autoscaler to set up for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
This setting is ignored when Kafka is disabled.
Deprecation notice: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.processor.unmanagedReplicas` to `true`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxReplicas</b></td>
        <td>integer</td>
        <td>
          `maxReplicas` is the upper limit for the number of pods that can be set by the autoscaler; cannot be smaller than MinReplicas.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 3<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex">metrics</a></b></td>
        <td>[]object</td>
        <td>
          Metrics used by the pod autoscaler. For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/horizontal-pod-autoscaler-v2/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minReplicas</b></td>
        <td>integer</td>
        <td>
          `minReplicas` is the lower limit for the number of replicas to which the autoscaler
can scale down. It defaults to 1 pod. minReplicas is allowed to be 0 if the
alpha feature gate HPAScaleToZero is enabled and at least one Object or External
metric is configured. Scaling is active as long as at least one metric value is
available.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          `status` describes the desired status regarding deploying an horizontal pod autoscaler.<br>
- `Disabled` does not deploy an horizontal pod autoscaler.<br>
- `Enabled` deploys an horizontal pod autoscaler.<br><br/>
          <br/>
            <i>Enum</i>: Disabled, Enabled<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscaler)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexcontainerresource">containerResource</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternal">external</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobject">object</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpods">pods</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexresource">resource</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].containerResource
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>container</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexcontainerresourcetarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].containerResource.target
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexcontainerresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].external
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternalmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternaltarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].external.metric
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternal)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternalmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].external.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternalmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternalmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].external.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternalmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].external.target
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexexternal)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectdescribedobject">describedObject</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjecttarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object.describedObject
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object.metric
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobjectmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].object.target
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexobject)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].pods
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodsmetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodstarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].pods.metric
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpods)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodsmetricselector">selector</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].pods.metric.selector
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodsmetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodsmetricselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].pods.metric.selector.matchExpressions[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpodsmetricselector)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].pods.target
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexpods)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].resource
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexresourcetarget">target</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.kafkaConsumerAutoscaler.metrics[index].resource.target
<sup><sup>[↩ Parent](#flowcollectorspecprocessorkafkaconsumerautoscalermetricsindexresource)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>averageUtilization</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>averageValue</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`Metrics` define the processor configuration regarding metrics

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>disableAlerts</b></td>
        <td>[]string</td>
        <td>
          `disableAlerts` is a list of alert groups that should be disabled from the default set of alerts.
Possible values are: `NetObservNoFlows`, `NetObservLokiError`, `PacketDropsByKernel`, `PacketDropsByDevice`, `IPsecErrors`, `NetpolDenied`,
`LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`.
More information on alerts: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricshealthrulesindex">healthRules</a></b></td>
        <td>[]object</td>
        <td>
          `healthRules` is a list of health rules to be created for Prometheus, organized by templates and variants.
Each health rule can be configured to generate either alerts or recording rules based on the mode field.
More information on health rules: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>includeList</b></td>
        <td>[]enum</td>
        <td>
          `includeList` is a list of metric names to specify which ones to generate.
The names correspond to the names in Prometheus without the prefix. For example,
`namespace_egress_packets_total` shows up as `netobserv_namespace_egress_packets_total` in Prometheus.
Note that the more metrics you add, the bigger is the impact on Prometheus workload resources.
Metrics enabled by default are:
`namespace_flows_total`, `node_ingress_bytes_total`, `node_egress_bytes_total`, `workload_ingress_bytes_total`,
`workload_egress_bytes_total`, `namespace_drop_packets_total` (when `PacketDrop` feature is enabled),
`namespace_rtt_seconds` (when `FlowRTT` feature is enabled), `namespace_dns_latency_seconds` (when `DNSTracking` feature is enabled),
`namespace_network_policy_events_total` (when `NetworkEvents` feature is enabled).
More information, with full list of available metrics: https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricsserver">server</a></b></td>
        <td>object</td>
        <td>
          Metrics server endpoint configuration for Prometheus scraper<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.healthRules[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetrics)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>template</b></td>
        <td>enum</td>
        <td>
          Health rule template name.
Possible values are: `PacketDropsByKernel`, `PacketDropsByDevice`, `IPsecErrors`, `NetpolDenied`,
`LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`.
Note: `NetObservNoFlows` and `NetObservLokiError` are alert-only and cannot be used as health rules.
More information on health rules: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md<br/>
          <br/>
            <i>Enum</i>: PacketDropsByKernel, PacketDropsByDevice, IPsecErrors, NetpolDenied, LatencyHighTrend, DNSErrors, DNSNxDomain, ExternalEgressHighTrend, ExternalIngressHighTrend, Ingress5xxErrors, IngressHTTPLatencyTrend<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricshealthrulesindexvariantsindex">variants</a></b></td>
        <td>[]object</td>
        <td>
          A list of variants for this template<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          Mode defines whether this health rule should be generated as an alert or a recording rule.
Possible values are: `Alert` (default), `Recording`.
Recording rules violations are visible in the Network Health dashboard without generating any Prometheus alert.
This provides an alternative way of getting Health information for SRE and cluster admins who may find
many new alerts burdensome.<br/>
          <br/>
            <i>Enum</i>: Alert, Recording<br/>
            <i>Default</i>: Alert<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.healthRules[index].variants[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetricshealthrulesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessormetricshealthrulesindexvariantsindexthresholds">thresholds</a></b></td>
        <td>object</td>
        <td>
          Thresholds of the health rule per severity.
They are expressed as a percentage of errors above which the alert is triggered. They must be parsable as floats.
Required for both alert and recording modes<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>enum</td>
        <td>
          Optional grouping criteria, possible values are: `Node`, `Namespace`, `Workload`.<br/>
          <br/>
            <i>Enum</i>: , Node, Namespace, Workload<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>lowVolumeThreshold</b></td>
        <td>string</td>
        <td>
          The low volume threshold allows to ignore metrics with a too low volume of traffic, in order to improve signal-to-noise.
It is provided as an absolute rate (bytes per second or packets per second, depending on the context).
When provided, it must be parsable as a float.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          Mode overrides the health rule mode for this specific variant.
If not specified, inherits from the parent health rule's mode.
Possible values are: `Alert`, `Recording`.<br/>
          <br/>
            <i>Enum</i>: Alert, Recording<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>trendDuration</b></td>
        <td>string</td>
        <td>
          For trending health rules, the duration interval for baseline comparison. For example, "2h" means comparing against a 2-hours average. Defaults to 2h.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>trendOffset</b></td>
        <td>string</td>
        <td>
          For trending health rules, the time offset for baseline comparison. For example, "1d" means comparing against yesterday. Defaults to 1d.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.healthRules[index].variants[index].thresholds
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetricshealthrulesindexvariantsindex)</sup></sup>



Thresholds of the health rule per severity.
They are expressed as a percentage of errors above which the alert is triggered. They must be parsable as floats.
Required for both alert and recording modes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>critical</b></td>
        <td>string</td>
        <td>
          Threshold for severity `critical`. Leave empty to not generate a Critical alert.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>info</b></td>
        <td>string</td>
        <td>
          Threshold for severity `info`. Leave empty to not generate an Info alert.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>warning</b></td>
        <td>string</td>
        <td>
          Threshold for severity `warning`. Leave empty to not generate a Warning alert.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.server
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetrics)</sup></sup>



Metrics server endpoint configuration for Prometheus scraper

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>
          The metrics server HTTP port.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricsservertls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.server.tls
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetricsserver)</sup></sup>



TLS configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Select the type of TLS configuration:<br>
- `Disabled` (default) to not configure TLS for the endpoint.
- `Provided` to manually provide cert file and a key file. [Unsupported (*)].
- `Auto` to use OpenShift auto generated certificate using annotations.<br/>
          <br/>
            <i>Enum</i>: Disabled, Provided, Auto<br/>
            <i>Default</i>: Disabled<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the provided certificate.
If set to `true`, the `providedCaFile` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricsservertlsprovided">provided</a></b></td>
        <td>object</td>
        <td>
          TLS configuration when `type` is set to `Provided`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessormetricsservertlsprovidedcafile">providedCaFile</a></b></td>
        <td>object</td>
        <td>
          Reference to the CA file when `type` is set to `Provided`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.server.tls.provided
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetricsservertls)</sup></sup>



TLS configuration when `type` is set to `Provided`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.metrics.server.tls.providedCaFile
<sup><sup>[↩ Parent](#flowcollectorspecprocessormetricsservertls)</sup></sup>



Reference to the CA file when `type` is set to `Provided`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.resources
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`resources` are the compute resources required by this container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorresourcesclaimsindex">claims</a></b></td>
        <td>[]object</td>
        <td>
          Claims lists the names of resources, defined in spec.resourceClaims,
that are used by this container.

This field depends on the
DynamicResourceAllocation feature gate.

This field is immutable. It can only be set for containers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>
          Limits describes the maximum amount of compute resources allowed.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>
          Requests describes the minimum amount of compute resources required.
If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
otherwise to an implementation-defined value. Requests cannot exceed Limits.
More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.resources.claims[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorresources)</sup></sup>



ResourceClaim references one entry in PodSpec.ResourceClaims.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name must match the name of one entry in pod.spec.resourceClaims of
the Pod where this field is used. It makes that resource available
inside a container.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>request</b></td>
        <td>string</td>
        <td>
          Request is the name chosen for a request in the referenced claim.
If empty, everything from the claim is made available, otherwise
only the result of this request.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.service
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



Service configuration, only used when `spec.deploymentModel` is `Service`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>tlsType</b></td>
        <td>enum</td>
        <td>
          Select the type of TLS configuration:<br>
- `Disabled` to not configure TLS for the endpoint. Disabling TLS results in a less secure deployment model.<br>
- `Provided` to manually provide the key and certificate references.<br>
- `Auto` (default) to enable automatically based on the running environment.<br>
- `Auto-mTLS` to preconfigure mTLS. [Unsupported (*)].<br>
See also: https://github.com/netobserv/netobserv-operator/blob/main/docs/TLS.md.<br/>
          <br/>
            <i>Enum</i>: Disabled, Provided, Auto, Auto-mTLS<br/>
            <i>Default</i>: Auto<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorserviceprovidedcertificates">providedCertificates</a></b></td>
        <td>object</td>
        <td>
          TLS or mTLS configuration when `type` is set to `Provided`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.service.providedCertificates
<sup><sup>[↩ Parent](#flowcollectorspecprocessorservice)</sup></sup>



TLS or mTLS configuration when `type` is set to `Provided`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorserviceprovidedcertificatescafile">caFile</a></b></td>
        <td>object</td>
        <td>
          Reference to the CA file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorserviceprovidedcertificatesclientcert">clientCert</a></b></td>
        <td>object</td>
        <td>
          TLS client certificate reference, used for mTLS. Leave unset for simple TLS.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorserviceprovidedcertificatesservercert">serverCert</a></b></td>
        <td>object</td>
        <td>
          TLS server certificate reference.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.service.providedCertificates.caFile
<sup><sup>[↩ Parent](#flowcollectorspecprocessorserviceprovidedcertificates)</sup></sup>



Reference to the CA file.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>file</b></td>
        <td>string</td>
        <td>
          File name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing the file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the file reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.service.providedCertificates.clientCert
<sup><sup>[↩ Parent](#flowcollectorspecprocessorserviceprovidedcertificates)</sup></sup>



TLS client certificate reference, used for mTLS. Leave unset for simple TLS.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.service.providedCertificates.serverCert
<sup><sup>[↩ Parent](#flowcollectorspecprocessorserviceprovidedcertificates)</sup></sup>



TLS server certificate reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.slicesConfig
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



Global configuration managing FlowCollectorSlices custom resources.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          `enable` determines if the FlowCollectorSlice feature is enabled. If not, all resources of kind FlowCollectorSlice are simply ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>collectionMode</b></td>
        <td>enum</td>
        <td>
          `collectionMode` determines how the FlowCollectorSlice custom resources impacts the flow collection process:<br>
- When set to `AlwaysCollect`, all flows are collected regardless of the presence of FlowCollectorSlice.<br>
- When set to `AllowList`, only the flows related to namespaces where a FlowCollectorSlice resource is present, or configured via the global `namespacesAllowList`, are collected.<br><br/>
          <br/>
            <i>Enum</i>: AlwaysCollect, AllowList<br/>
            <i>Default</i>: AlwaysCollect<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespacesAllowList</b></td>
        <td>[]string</td>
        <td>
          `namespacesAllowList` is a list of namespaces for which flows are always collected, regardless of the presence of FlowCollectorSlice in those namespaces.
An entry enclosed by slashes, such as `/openshift-.*/`, is matched as a regular expression.
This setting is ignored if `collectionMode` is different from `AllowList`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.subnetLabels
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`subnetLabels` allows to define custom labels on subnets and IPs or to enable automatic labeling of recognized subnets in OpenShift, which is used to identify cluster external traffic.
When a subnet matches the source or destination IP of a flow, a corresponding field is added: `SrcSubnetLabel` or `DstSubnetLabel`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprocessorsubnetlabelscustomlabelsindex">customLabels</a></b></td>
        <td>[]object</td>
        <td>
          `customLabels` allows you to customize subnets and IPs labeling, such as to identify cluster external workloads or web services.
External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/>
If `openShiftAutoDetect` is disabled or you are not using OpenShift, it is recommended to manually configure labels for the cluster subnets, to distinguish internal traffic from external traffic.<br/>
If `openShiftAutoDetect` is enabled, `customLabels` overrides the detected subnets when they overlap.<br/><br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>openShiftAutoDetect</b></td>
        <td>boolean</td>
        <td>
          `openShiftAutoDetect` allows, when set to `true`, to detect automatically the machines, pods and services subnets based on the
OpenShift install configuration and the Cluster Network Operator configuration. Indirectly, this is a way to accurately detect
external traffic: flows that are not labeled for those subnets are external to the cluster. Enabled by default on OpenShift.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.subnetLabels.customLabels[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorsubnetlabels)</sup></sup>



SubnetLabel allows to label subnets and IPs, such as to identify cluster-external workloads or web services.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>cidrs</b></td>
        <td>[]string</td>
        <td>
          List of CIDRs, such as `["1.2.3.4/32"]`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Label name, used to flag matching flows.
External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/><br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`prometheus` defines Prometheus settings, such as querier configuration used to fetch metrics from the Console plugin.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprometheusquerier">querier</a></b></td>
        <td>object</td>
        <td>
          Prometheus querying configuration, such as client settings, used in the Console plugin.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier
<sup><sup>[↩ Parent](#flowcollectorspecprometheus)</sup></sup>



Prometheus querying configuration, such as client settings, used in the Console plugin.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>mode</b></td>
        <td>enum</td>
        <td>
          `mode` must be set according to the type of Prometheus installation that stores NetObserv metrics:<br>
- Use `Auto` to try configuring automatically. In OpenShift, it uses the Thanos querier from OpenShift Cluster Monitoring.<br>
- Use `Manual` for a manual setup.<br><br/>
          <br/>
            <i>Enum</i>: Manual, Auto<br/>
            <i>Default</i>: Auto<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          When `enable` is `true`, the Console plugin queries flow metrics from Prometheus instead of Loki whenever possible.
It is enabled by default: set it to `false` to disable this feature.
The Console plugin can use either Loki or Prometheus as a data source for metrics (see also `spec.loki`), or both.
Not all queries are transposable from Loki to Prometheus. Hence, if Loki is disabled, some features of the plugin are disabled as well,
such as getting per-pod information or viewing raw flows.
If both Prometheus and Loki are enabled, Prometheus takes precedence and Loki is used as a fallback for queries that Prometheus cannot handle.
If they are both disabled, the Console plugin is not deployed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanual">manual</a></b></td>
        <td>object</td>
        <td>
          Prometheus configuration for `Manual` mode.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeout</b></td>
        <td>string</td>
        <td>
          `timeout` is the read timeout for console plugin queries to Prometheus.
A timeout of zero means no timeout.<br/>
          <br/>
            <i>Default</i>: 30s<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual
<sup><sup>[↩ Parent](#flowcollectorspecprometheusquerier)</sup></sup>



Prometheus configuration for `Manual` mode.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualalertmanager">alertManager</a></b></td>
        <td>object</td>
        <td>
          AlertManager configuration. This is used in the console to query silenced alerts, for displaying health information.
When used in OpenShift it can be left empty to use the Console API instead.
[Unsupported (*)].<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>forwardUserToken</b></td>
        <td>boolean</td>
        <td>
          Set `true` to forward logged in user token in queries to Prometheus<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualtls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Prometheus URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          `url` is the address of an existing Prometheus service to use for querying metrics.<br/>
          <br/>
            <i>Default</i>: http://prometheus:9090<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.alertManager
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanual)</sup></sup>



AlertManager configuration. This is used in the console to query silenced alerts, for displaying health information.
When used in OpenShift it can be left empty to use the Console API instead.
[Unsupported (*)].

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualalertmanagertls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS client configuration for Prometheus AlertManager URL.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          `url` is the address of an existing Prometheus AlertManager service to use for querying alerts.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.alertManager.tls
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanualalertmanager)</sup></sup>



TLS client configuration for Prometheus AlertManager URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualalertmanagertlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualalertmanagertlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.alertManager.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanualalertmanagertls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.alertManager.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanualalertmanagertls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.tls
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanual)</sup></sup>



TLS client configuration for Prometheus URL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualtlscacert">caCert</a></b></td>
        <td>object</td>
        <td>
          `caCert` defines the reference of the certificate for the Certificate Authority.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Enable TLS<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>insecureSkipVerify</b></td>
        <td>boolean</td>
        <td>
          `insecureSkipVerify` allows skipping client-side verification of the server certificate.
If set to `true`, the `caCert` field is ignored.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprometheusqueriermanualtlsusercert">userCert</a></b></td>
        <td>object</td>
        <td>
          `userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.tls.caCert
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanualtls)</sup></sup>



`caCert` defines the reference of the certificate for the Certificate Authority.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.prometheus.querier.manual.tls.userCert
<sup><sup>[↩ Parent](#flowcollectorspecprometheusqueriermanualtls)</sup></sup>



`userCert` defines the user certificate reference and is used for mTLS. When you use one-way TLS, you can ignore this property.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>certFile</b></td>
        <td>string</td>
        <td>
          `certFile` defines the path to the certificate file name within the config map or secret.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>certKey</b></td>
        <td>string</td>
        <td>
          `certKey` defines the path to the certificate private key file name within the config map or secret. Omit when the key is not necessary.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the config map or secret containing certificates.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where NetObserv is deployed.
If the namespace is different, the config map or the secret is copied so that it can be mounted as required.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type for the certificate reference: `configmap` or `secret`.<br/>
          <br/>
            <i>Enum</i>: configmap, secret<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status
<sup><sup>[↩ Parent](#flowcollector)</sup></sup>



`FlowCollectorStatus` defines the observed state of FlowCollector

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          `conditions` represents the latest available observations of an object's state<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Namespace where console plugin and flowlogs-pipeline have been deployed.

Deprecated: annotations are used instead<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.conditions[index]
<sup><sup>[↩ Parent](#flowcollectorstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>