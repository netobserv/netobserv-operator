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
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
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
and accepted without a formal agreement for maintenance. The product maintainers might provide support
for these features, without any guarantee.<br/>
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
and accepted without a formal agreement for maintenance. The product maintainers might provide support
for these features, without any guarantee.

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
          `consolePlugin` defines the settings related to the OpenShift Console plugin.<br/>
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
          Namespace where Network Observability pods are deployed.<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: Namespace is immutable. If you need to change it, delete and recreate the resource.</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecnetworkpolicy">networkPolicy</a></b></td>
        <td>object</td>
        <td>
          `networkPolicy` defines network policy settings for Network Observability components isolation.<br/>
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
          `ebpf` describes the settings related to the eBPF-based flow reporter when `spec.agent.type` is set to `eBPF`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          `type` selects the flows tracing agent.

Deprecated: Previously, this field allowed to select between `eBPF` or `IPFIX`.
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



`ebpf` describes the settings related to the eBPF-based flow reporter when `spec.agent.type` is set to `eBPF`.

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
        <td><b>dnsTrackingPorts</b></td>
        <td>[]integer</td>
        <td>
          `dnsTrackingPorts` defines the list of DNS ports to track when DNSTracking feature is enabled.
For example: [53, 5353, 8053]. Leave empty to use OpenShift default (53, 5353).
Maximum 8 ports allowed.<br/>
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
- `EbpfManager`: [Unsupported (*)]. Use eBPF Manager to manage Network Observability eBPF programs. Pre-requisite: the eBPF Manager operator (or upstream bpfman operator) must be installed.<br>
- `UDNMapping`: Enable interfaces mapping to User Defined Networks (UDN).<br>
This feature requires mounting the kernel debug filesystem, so the eBPF agent pods must run as privileged via `spec.agent.ebpf.privileged`.
It requires using the OVN-Kubernetes network plugin.<br>
- `IPSec`, to track flows between nodes with IPsec encryption.<br>
- `TLSTracking`, to track TLS usage.<br><br/>
          <br/>
            <i>Enum</i>: PacketDrop, DNSTracking, FlowRTT, NetworkEvents, PacketTranslation, EbpfManager, UDNMapping, IPSec, TLSTracking<br/>
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
          `logLevel` defines the log level for the Network Observability eBPF Agent<br/>
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
        <td><b>affinity</b></td>
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
        <td><b>tolerations</b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
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
          <br/>
            <i>Enum</i>: NetObservDroppedFlows<br/>
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
        <td><b>scrapeInterval</b></td>
        <td>string</td>
        <td>
          Prometheus scraping interval, how often metrics are pulled.<br/>
          <br/>
            <i>Format</i>: duration<br/>
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
If set to `true`, the `providedCaFile` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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


### FlowCollector.spec.consolePlugin
<sup><sup>[↩ Parent](#flowcollectorspec)</sup></sup>



`consolePlugin` defines the settings related to the OpenShift Console plugin.

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
        <td><b>autoscaler</b></td>
        <td>object</td>
        <td>
          `autoscaler`: spec of a horizontal pod autoscaler to set up for the web console Deployment.

Deprecated: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.consolePlugin.unmanagedReplicas` to `true`.<br/>
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
          `logLevel` for the web console backend.<br/>
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
          `quickFilters` configures quick filter presets for the web console.
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
This is not recommended, as it doesn't provide the same level of integration with OpenShift.
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
        <td><b>affinity</b></td>
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
        <td><b>tolerations</b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
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
          EnterpriseID, or Private Enterprise Number (PEN). To date, Network Observability does not own an assigned number,
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
          Kafka topic to use. It must exist. Network Observability does not create it.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>compression</b></td>
        <td>enum</td>
        <td>
          Compression codec to use when producing messages to Kafka.
Accepted values are: `none` (default), `gzip`, `snappy`, `lz4`, `zstd`.<br/>
          <br/>
            <i>Enum</i>: none, gzip, snappy, lz4, zstd<br/>
            <i>Default</i>: none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkasasl">sasl</a></b></td>
        <td>object</td>
        <td>
          SASL authentication configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecexportersindexkafkatls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.
When configuring TLS, the operator watches the certificate secret and copies it to both the netobserv and netobserv-privileged namespaces.
In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
Refer to the Kafka configuration documentation for more information.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.exporters[index].kafka.sasl
<sup><sup>[↩ Parent](#flowcollectorspecexportersindexkafka)</sup></sup>



SASL authentication configuration.

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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
When configuring TLS, the operator watches the certificate secret and copies it to both the netobserv and netobserv-privileged namespaces.
In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
Refer to the Kafka configuration documentation for more information.

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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
By default, Network Observability format proposal is used: https://github.com/rhobs/observability-data-model/blob/main/network-observability.md#format-proposal .
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Kafka topic to use. It must exist. Network Observability does not create it.<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>compression</b></td>
        <td>enum</td>
        <td>
          Compression codec to use when producing messages to Kafka.
Accepted values are: `none` (default), `gzip`, `snappy`, `lz4`, `zstd`.<br/>
          <br/>
            <i>Enum</i>: none, gzip, snappy, lz4, zstd<br/>
            <i>Default</i>: none<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkasasl">sasl</a></b></td>
        <td>object</td>
        <td>
          SASL authentication configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspeckafkatls">tls</a></b></td>
        <td>object</td>
        <td>
          TLS and mTLS client configuration. When using TLS, verify that the address matches the Kafka port used for TLS, generally 9093.
We recommend the use of mTLS for higher security standards.
When configuring TLS, the operator watches the certificate secret and copies it to both the netobserv and netobserv-privileged namespaces.
In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
Refer to the Kafka configuration documentation for more information.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.kafka.sasl
<sup><sup>[↩ Parent](#flowcollectorspeckafka)</sup></sup>



SASL authentication configuration.

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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
When configuring TLS, the operator watches the certificate secret and copies it to both the netobserv and netobserv-privileged namespaces.
In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
Refer to the Kafka configuration documentation for more information.

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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          `excludeLabels` is a list of fields to be excluded from the list of Loki labels.<br/>
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
          Namespace where this `LokiStack` resource is located. If omitted, it is assumed to be the same as `spec.namespace`.
When configuring a different namespace, the operator watches certificate secret and copies it to the netobserv main namespaces.
In order to do so, you must grant it permissions to the `netobserv-secret-watcher` and `netobserv-secret-creator` roles in the corresponding namespaces.
Refer to the Loki configuration documentation for more information.<br/>
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
- `Host` (deprecated) - uses the local pod service account to authenticate to Loki.<br>
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
This is meant for development and demo use only, and not recommended in production.
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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



`networkPolicy` defines network policy settings for Network Observability components isolation.

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
          `additionalNamespaces` contains additional namespaces allowed to connect to the Network Observability namespace.
It provides flexibility in the network policy configuration, but if you need a more specific
configuration, you can disable it and install your own instead.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enable</b></td>
        <td>boolean</td>
        <td>
          Deploys network policies on the namespaces used by Network Observability operands (main and privileged).
These network policies better isolate the Network Observability components to prevent undesired connections from and to them.
Because it cannot be tested with all CNIs, this option is only enabled by default when Network Observability runs in a known
supported environment, and it is disabled by default otherwise.
When disabled, it is highly recommended to create network policies manually, to prevent undesired accesses.
This setting is for operands only, and does not control the Operator network policy, which is covered by the `OPERATOR_NETWORK_POLICY`
environment variable.
If the operator and the operands are deployed in the same namespace, this setting is ignored, and only `OPERATOR_NETWORK_POLICY` controls
whether or not any policy is installed.
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
        <td><b>bgpEnrichment</b></td>
        <td>boolean</td>
        <td>
          `bgpEnrichment` enables BGP ASN enrichment by watching FRRConfiguration CRDs (frrk8s.metallb.io/v1beta1).
When enabled, flows are enriched with `SrcASN` and `DstASN` fields based on longest-prefix match against
advertised prefixes from FRRConfiguration resources. Requires frr-k8s to be installed in the cluster.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterName</b></td>
        <td>string</td>
        <td>
          `clusterName` is the name of the cluster to appear in the flows data.
In a multi-cluster context, it makes it possible to identify the flows provenance.
Leave empty to make it determined automatically.<br/>
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
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxy">informerCacheProxy</a></b></td>
        <td>object</td>
        <td>
          `informerCacheProxy` (experimental) is the configuration for centralized Kubernetes informers that push cache updates to flowlogs-pipeline processors.
This reduces load on the API server by having fewer components running informers, and allows to scale FLP up without paying the cost of new informers.
It is not recommended to use on small clusters with few FLP replicas (<4), or when API server load is not a concern.
This feature is currently experimental.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>kafkaConsumerAutoscaler</b></td>
        <td>object</td>
        <td>
          `kafkaConsumerAutoscaler` is the spec of a horizontal pod autoscaler to set up for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
This setting is ignored when Kafka is disabled.

Deprecated: managed autoscaler will be removed in a future version. You may configure instead an autoscaler of your choice, and set `spec.processor.unmanagedReplicas` to `true`.<br/>
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
          `kafkaConsumerReplicas` defines the number of replicas (pods) to start for `flowlogs-pipeline-transformer`, which consumes Kafka messages.
This setting is ignored when Kafka is disabled.

Deprecated: use `spec.processor.consumerReplicas` instead.<br/>
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
          `subnetLabels` allows to define custom labels on subnets and IPs and to enable automatic labeling of recognized subnets, which is used to identify cluster external traffic.
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
          `conversationHeartbeatInterval` is the time to wait between "tick" events of a conversation.<br/>
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
          `dropUnusedFields`.

Deprecated: this setting is not used anymore.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableKubeProbes</b></td>
        <td>boolean</td>
        <td>
          `enableKubeProbes` is a flag to enable or disable Kubernetes liveness and readiness probes.<br/>
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
          `healthPort` is a collector HTTP port in the Pod that exposes the health check API.<br/>
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
          `profilePort` allows setting up a Go pprof profiler listening to this port.
This is for debugging purpose only. This port should not be exposed, you can
access it through local port-forwarding.<br/>
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
          `scheduling` controls how the pods are scheduled on nodes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessoradvancedsecondarynetworksindex">secondaryNetworks</a></b></td>
        <td>[]object</td>
        <td>
          Defines secondary networks to be checked for resources identification.
To guarantee a correct identification, indexed values must form an unique identifier across the cluster.
If the same index is used by several resources, those resources might be incorrectly labeled.
If not provided and `spec.agent.ebpf.privileged` is `true`, secondary networks are detected automatically.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.advanced.scheduling
<sup><sup>[↩ Parent](#flowcollectorspecprocessoradvanced)</sup></sup>



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
        <td><b>affinity</b></td>
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
        <td><b>tolerations</b></td>
        <td>[]object</td>
        <td>
          `tolerations` is a list of tolerations that allow the pod to schedule onto nodes with matching taints.
For documentation, refer to https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#scheduling.<br/>
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
          <br/>
            <i>Enum</i>: MAC, IP, Interface<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Deprecated: `name` is unused.<br/>
        </td>
        <td>false</td>
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


### FlowCollector.spec.processor.informerCacheProxy
<sup><sup>[↩ Parent](#flowcollectorspecprocessor)</sup></sup>



`informerCacheProxy` (experimental) is the configuration for centralized Kubernetes informers that push cache updates to flowlogs-pipeline processors.
This reduces load on the API server by having fewer components running informers, and allows to scale FLP up without paying the cost of new informers.
It is not recommended to use on small clusters with few FLP replicas (<4), or when API server load is not a concern.
This feature is currently experimental.

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
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxyadvanced">advanced</a></b></td>
        <td>object</td>
        <td>
          `advanced` allows setting some technical parameters of the informer cache proxy component.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          `enabled` controls whether to deploy the informer cache proxy.
When `true`, a dedicated deployment watches K8s resources and pushes cache updates via gRPC to FLP processors, reducing API server load.
When `false` (default), each FLP processor uses local informers.
Enable only on large clusters or when API server load is a concern, as it adds deployment complexity.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>
          `replicas` defines the number of replicas for the flowlogs-pipeline-informers deployment.
For high availability, a minimum of 2 replicas is required when `enabled` is `true`.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 2<br/>
            <i>Minimum</i>: 2<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxyresources">resources</a></b></td>
        <td>object</td>
        <td>
          `resources` are the compute resources required by the informer cache proxy container.
For more information, see https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/<br/>
          <br/>
            <i>Default</i>: map[limits:map[cpu:200m memory:256Mi] requests:map[cpu:50m memory:128Mi]]<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxytls">tls</a></b></td>
        <td>object</td>
        <td>
          `tls` defines the TLS configuration for the gRPC communication between the informer cache proxy and processors.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.informerCacheProxy.advanced
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxy)</sup></sup>



`advanced` allows setting some technical parameters of the informer cache proxy component.

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
        <td><b>batchSize</b></td>
        <td>integer</td>
        <td>
          `batchSize` defines the maximum number of cache entries to send in a single update batch.<br/>
          <br/>
            <i>Default</i>: 100<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>processorPort</b></td>
        <td>integer</td>
        <td>
          `processorPort` defines the gRPC port where flowlogs-pipeline processors listen for k8s cache updates.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 9402<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 65535<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>resyncInterval</b></td>
        <td>integer</td>
        <td>
          `resyncInterval` defines the interval in seconds to rediscover processors and sync state.<br/>
          <br/>
            <i>Default</i>: 60<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sendTimeout</b></td>
        <td>integer</td>
        <td>
          `sendTimeout` defines the timeout in seconds for sending updates to processors.<br/>
          <br/>
            <i>Default</i>: 10<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>updateBufferSize</b></td>
        <td>integer</td>
        <td>
          `updateBufferSize` defines the size of the internal update channel buffer.<br/>
          <br/>
            <i>Default</i>: 100<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.informerCacheProxy.resources
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxy)</sup></sup>



`resources` are the compute resources required by the informer cache proxy container.
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
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxyresourcesclaimsindex">claims</a></b></td>
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


### FlowCollector.spec.processor.informerCacheProxy.resources.claims[index]
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxyresources)</sup></sup>



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


### FlowCollector.spec.processor.informerCacheProxy.tls
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxy)</sup></sup>



`tls` defines the TLS configuration for the gRPC communication between the informer cache proxy and processors.

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
- `Disabled` to not configure TLS for the k8scache endpoint. Disabling TLS results in a less secure deployment model.<br>
- `Provided` to manually provide cert/key references for mTLS.<br>
- `Auto` (default) to use OpenShift service-ca for automatic server certificate generation (simple TLS).<br>
- `Auto-mTLS` to preconfigure mTLS with cert-manager. [Unsupported (*)].<br>
See also: https://github.com/netobserv/netobserv-operator/blob/main/docs/TLS.md.<br/>
          <br/>
            <i>Enum</i>: Disabled, Provided, Auto, Auto-mTLS<br/>
            <i>Default</i>: Auto<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificates">providedCertificates</a></b></td>
        <td>object</td>
        <td>
          mTLS configuration when `type` is set to `Provided`.
`serverCert` is required. `clientCert` is optional; if provided, mTLS is enabled.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.informerCacheProxy.tls.providedCertificates
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxytls)</sup></sup>



mTLS configuration when `type` is set to `Provided`.
`serverCert` is required. `clientCert` is optional; if provided, mTLS is enabled.

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
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificatescafile">caFile</a></b></td>
        <td>object</td>
        <td>
          Reference to the CA file.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificatesclientcert">clientCert</a></b></td>
        <td>object</td>
        <td>
          TLS client certificate reference, used for mTLS. Leave unset for simple TLS.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificatesservercert">serverCert</a></b></td>
        <td>object</td>
        <td>
          TLS server certificate reference.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.spec.processor.informerCacheProxy.tls.providedCertificates.caFile
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificates)</sup></sup>



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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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


### FlowCollector.spec.processor.informerCacheProxy.tls.providedCertificates.clientCert
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificates)</sup></sup>



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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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


### FlowCollector.spec.processor.informerCacheProxy.tls.providedCertificates.serverCert
<sup><sup>[↩ Parent](#flowcollectorspecprocessorinformercacheproxytlsprovidedcertificates)</sup></sup>



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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
        <td><b>additionalIncludeList</b></td>
        <td>[]enum</td>
        <td>
          `additionalIncludeList` is a list of metric names to include in addition to the default metrics.
Unlike `includeList`, this appends to the default list rather than replacing it.
This field is mutually exclusive with `includeList`. If `includeList` is set, `additionalIncludeList` is ignored.
The names correspond to the names in Prometheus without the prefix. For example,
`namespace_egress_packets_total` shows up as `netobserv_namespace_egress_packets_total` in Prometheus.
Note that the more metrics you add, the bigger is the impact on Prometheus workload resources.
More information, with full list of available metrics: https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md<br/>
          <br/>
            <i>Enum</i>: namespace_egress_bytes_total, namespace_egress_packets_total, namespace_ingress_bytes_total, namespace_ingress_packets_total, namespace_flows_total, node_egress_bytes_total, node_egress_packets_total, node_ingress_bytes_total, node_ingress_packets_total, node_flows_total, workload_egress_bytes_total, workload_egress_packets_total, workload_ingress_bytes_total, workload_ingress_packets_total, workload_flows_total, namespace_drop_bytes_total, namespace_drop_packets_total, node_drop_bytes_total, node_drop_packets_total, workload_drop_bytes_total, workload_drop_packets_total, namespace_rtt_seconds, node_rtt_seconds, workload_rtt_seconds, namespace_dns_latency_seconds, node_dns_latency_seconds, workload_dns_latency_seconds, namespace_dns_flows_total, node_dns_flows_total, workload_dns_flows_total, node_network_policy_events_total, namespace_network_policy_events_total, workload_network_policy_events_total, node_ipsec_flows_total, namespace_ipsec_flows_total, workload_ipsec_flows_total, node_tls_flows_total, namespace_tls_flows_total, workload_tls_flows_total, node_to_node_ingress_flows_total<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>disableAlerts</b></td>
        <td>[]string</td>
        <td>
          `disableAlerts` is a list of alert groups that should be disabled from the default set of alerts.
Possible values are: `NetObservNoFlows`, `NetObservLokiError`, `PacketDropsByKernel`, `PacketDropsByDevice`, `IPsecErrors`, `NetpolDenied`,
`LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`, `TLSInsecureVersion`.
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
`namespace_rtt_seconds` (when `FlowRTT` feature is enabled), `namespace_dns_latency_seconds` and `namespace_dns_flows_total` (when `DNSTracking` feature is enabled),
`namespace_network_policy_events_total` (when `NetworkEvents` feature is enabled).
More information, with full list of available metrics: https://github.com/netobserv/netobserv-operator/blob/main/docs/Metrics.md<br/>
          <br/>
            <i>Enum</i>: namespace_egress_bytes_total, namespace_egress_packets_total, namespace_ingress_bytes_total, namespace_ingress_packets_total, namespace_flows_total, node_egress_bytes_total, node_egress_packets_total, node_ingress_bytes_total, node_ingress_packets_total, node_flows_total, workload_egress_bytes_total, workload_egress_packets_total, workload_ingress_bytes_total, workload_ingress_packets_total, workload_flows_total, namespace_drop_bytes_total, namespace_drop_packets_total, node_drop_bytes_total, node_drop_packets_total, workload_drop_bytes_total, workload_drop_packets_total, namespace_rtt_seconds, node_rtt_seconds, workload_rtt_seconds, namespace_dns_latency_seconds, node_dns_latency_seconds, workload_dns_latency_seconds, namespace_dns_flows_total, node_dns_flows_total, workload_dns_flows_total, node_network_policy_events_total, namespace_network_policy_events_total, workload_network_policy_events_total, node_ipsec_flows_total, namespace_ipsec_flows_total, workload_ipsec_flows_total, node_tls_flows_total, namespace_tls_flows_total, workload_tls_flows_total, node_to_node_ingress_flows_total<br/>
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
`LatencyHighTrend`, `DNSErrors`, `DNSNxDomain`, `ExternalEgressHighTrend`, `ExternalIngressHighTrend`, `Ingress5xxErrors`, `IngressHTTPLatencyTrend`, `TLSInsecureVersion`.
Note: `NetObservNoFlows` and `NetObservLokiError` are alert-only and cannot be used as health rules.
More information on health rules: https://github.com/netobserv/netobserv-operator/blob/main/docs/HealthRules.md<br/>
          <br/>
            <i>Enum</i>: PacketDropsByKernel, PacketDropsByDevice, IPsecErrors, NetpolDenied, LatencyHighTrend, DNSErrors, DNSNxDomain, ExternalEgressHighTrend, ExternalIngressHighTrend, Ingress5xxErrors, IngressHTTPLatencyTrend, TLSInsecureVersion<br/>
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
        <td><b>scrapeInterval</b></td>
        <td>string</td>
        <td>
          Prometheus scraping interval, how often metrics are pulled.<br/>
          <br/>
            <i>Format</i>: duration<br/>
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
- `Disabled` (default) to not configure TLS for the endpoint.<br>
- `Provided` to manually provide cert file and a key file. [Unsupported (*)].<br>
- `Auto` to use OpenShift auto generated certificate using annotations.<br><br/>
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
If set to `true`, the `providedCaFile` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing the file. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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



`subnetLabels` allows to define custom labels on subnets and IPs and to enable automatic labeling of recognized subnets, which is used to identify cluster external traffic.
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
        <td><b>autoDetect</b></td>
        <td>boolean</td>
        <td>
          `autoDetect` allows, when set to `true`, to detect automatically the machines, pods and services subnets based on
OpenShift configuration. Indirectly, this is a way to accurately detect
external traffic: flows that are not labeled for those subnets are external to the cluster. Enabled by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorspecprocessorsubnetlabelscustomlabelsindex">customLabels</a></b></td>
        <td>[]object</td>
        <td>
          `customLabels` allows you to customize subnets and IPs labeling, such as to identify cluster external workloads or web services.
External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/>
If `autoDetect` is disabled, it is recommended to manually configure labels for the cluster subnets, to distinguish internal traffic from external traffic.<br/>
If `autoDetect` is enabled, `customLabels` overrides the detected subnets when they overlap.<br/><br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>openShiftAutoDetect</b></td>
        <td>boolean</td>
        <td>
          `openShiftAutoDetect` allows, when set to `true`, to detect automatically the machines, pods and services subnets based on
OpenShift configuration. Indirectly, this is a way to accurately detect
external traffic: flows that are not labeled for those subnets are external to the cluster.

Deprecated: use `autoDetect` instead.<br/>
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
          `mode` must be set according to the type of Prometheus installation that stores Network Observability metrics:<br>
- Use `Auto` to configure automatically. It uses the Thanos querier from OpenShift Cluster Monitoring.<br>
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
          <br/>
            <i>Default</i>: true<br/>
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
It can be left empty to use the Console API instead.
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
It can be left empty to use the Console API instead.
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
If set to `true`, the `caCert` field is ignored. For security, this should not be used other than for testing or demo.<br/>
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
          Namespace of the config map or secret containing certificates. If omitted, the default is to use the same namespace as where Network Observability is deployed.
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
        <td><b><a href="#flowcollectorstatuscomponents">components</a></b></td>
        <td>object</td>
        <td>
          `components` reports the status of operator-managed components (agent, processor, plugin).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatusintegrations">integrations</a></b></td>
        <td>object</td>
        <td>
          `integrations` reports the status of external integrations (Loki, monitoring, exporters).<br/>
        </td>
        <td>false</td>
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


### FlowCollector.status.components
<sup><sup>[↩ Parent](#flowcollectorstatus)</sup></sup>



`components` reports the status of operator-managed components (agent, processor, plugin).

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
        <td><b><a href="#flowcollectorstatuscomponentsagent">agent</a></b></td>
        <td>object</td>
        <td>
          `agent` reports the status of the eBPF agent component.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatuscomponentsplugin">plugin</a></b></td>
        <td>object</td>
        <td>
          `plugin` reports the status of the console plugin component.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatuscomponentsprocessor">processor</a></b></td>
        <td>object</td>
        <td>
          `processor` reports the status of the flowlogs-pipeline component.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.components.agent
<sup><sup>[↩ Parent](#flowcollectorstatuscomponents)</sup></sup>



`agent` reports the status of the eBPF agent component.

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
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the overall health of the component.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown, Unused<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredReplicas</b></td>
        <td>integer</td>
        <td>
          `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podIssues</b></td>
        <td>string</td>
        <td>
          `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unhealthyPodCount</b></td>
        <td>integer</td>
        <td>
          `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.components.plugin
<sup><sup>[↩ Parent](#flowcollectorstatuscomponents)</sup></sup>



`plugin` reports the status of the console plugin component.

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
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the overall health of the component.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown, Unused<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredReplicas</b></td>
        <td>integer</td>
        <td>
          `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podIssues</b></td>
        <td>string</td>
        <td>
          `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unhealthyPodCount</b></td>
        <td>integer</td>
        <td>
          `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.components.processor
<sup><sup>[↩ Parent](#flowcollectorstatuscomponents)</sup></sup>



`processor` reports the status of the flowlogs-pipeline component.

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
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the overall health of the component.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown, Unused<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredReplicas</b></td>
        <td>integer</td>
        <td>
          `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podIssues</b></td>
        <td>string</td>
        <td>
          `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unhealthyPodCount</b></td>
        <td>integer</td>
        <td>
          `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.integrations
<sup><sup>[↩ Parent](#flowcollectorstatus)</sup></sup>



`integrations` reports the status of external integrations (Loki, monitoring, exporters).

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
        <td><b><a href="#flowcollectorstatusintegrationsexportersindex">exporters</a></b></td>
        <td>[]object</td>
        <td>
          `exporters` reports the status of configured exporters.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatusintegrationsloki">loki</a></b></td>
        <td>object</td>
        <td>
          `loki` reports the status of the Loki integration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorstatusintegrationsmonitoring">monitoring</a></b></td>
        <td>object</td>
        <td>
          `monitoring` reports the status of monitoring (dashboards, ServiceMonitor, etc.).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.integrations.exporters[index]
<sup><sup>[↩ Parent](#flowcollectorstatusintegrations)</sup></sup>



`FlowCollectorExporterStatus` represents the status of a configured exporter.

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
          `name` is a generated identifier for this exporter (e.g., "kafka-export-0"), derived from its type and position in spec.exporters.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the health of this exporter.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          `type` is the exporter type (Kafka, IPFIX, OpenTelemetry).<br/>
          <br/>
            <i>Enum</i>: Kafka, IPFIX, OpenTelemetry<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the exporter's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the exporter's current state.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.integrations.loki
<sup><sup>[↩ Parent](#flowcollectorstatusintegrations)</sup></sup>



`loki` reports the status of the Loki integration.

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
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the overall health of the component.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown, Unused<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredReplicas</b></td>
        <td>integer</td>
        <td>
          `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podIssues</b></td>
        <td>string</td>
        <td>
          `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unhealthyPodCount</b></td>
        <td>integer</td>
        <td>
          `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollector.status.integrations.monitoring
<sup><sup>[↩ Parent](#flowcollectorstatusintegrations)</sup></sup>



`monitoring` reports the status of monitoring (dashboards, ServiceMonitor, etc.).

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
        <td><b>state</b></td>
        <td>enum</td>
        <td>
          `state` reports the overall health of the component.<br/>
          <br/>
            <i>Enum</i>: Ready, InProgress, Failure, Degraded, Unknown, Unused<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>desiredReplicas</b></td>
        <td>integer</td>
        <td>
          `desiredReplicas` is the desired number of replicas (for Deployments) or nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          `message` is a human-readable description of the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>podIssues</b></td>
        <td>string</td>
        <td>
          `podIssues` is a summary of unhealthy pod issues (e.g., "3 pods CrashLoopBackOff: kafka connection refused").<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>
          `readyReplicas` is the number of ready replicas (for Deployments) or up-to-date nodes (for DaemonSets).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          `reason` is a one-word CamelCase reason for the component's current state.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unhealthyPodCount</b></td>
        <td>integer</td>
        <td>
          `unhealthyPodCount` is the number of pods in a degraded state (CrashLoopBackOff, OOMKilled, etc.).<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
