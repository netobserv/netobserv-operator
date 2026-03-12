# API Reference

Packages:

- [flows.netobserv.io/v1alpha1](#flowsnetobserviov1alpha1)

# flows.netobserv.io/v1alpha1

Resource Types:

- [FlowMetric](#flowmetric)




## FlowMetric
<sup><sup>[↩ Parent](#flowsnetobserviov1alpha1 )</sup></sup>






FlowMetric is the API allowing to create custom metrics from the collected flow logs.

<table>
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
      <td>flows.netobserv.io/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>FlowMetric</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#flowmetricspec">spec</a></b></td>
        <td>object</td>
        <td>
          FlowMetricSpec defines the desired state of FlowMetric
The provided API allows you to customize these metrics according to your needs.<br>
When adding new metrics or modifying existing labels, you must carefully monitor the memory
usage of Prometheus workloads as this could potentially have a high impact. Cf https://rhobs-handbook.netlify.app/products/openshiftmonitoring/telemetry.md/#what-is-the-cardinality-of-a-metric<br>
To check the cardinality of all NetObserv metrics, run as `promql`: `count({__name__=~"netobserv.*"}) by (__name__)`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowmetricstatus">status</a></b></td>
        <td>object</td>
        <td>
          FlowMetricStatus defines the observed state of FlowMetric<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowMetric.spec
<sup><sup>[↩ Parent](#flowmetric)</sup></sup>



FlowMetricSpec defines the desired state of FlowMetric
The provided API allows you to customize these metrics according to your needs.<br>
When adding new metrics or modifying existing labels, you must carefully monitor the memory
usage of Prometheus workloads as this could potentially have a high impact. Cf https://rhobs-handbook.netlify.app/products/openshiftmonitoring/telemetry.md/#what-is-the-cardinality-of-a-metric<br>
To check the cardinality of all NetObserv metrics, run as `promql`: `count({__name__=~"netobserv.*"}) by (__name__)`.

<table>
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
          Metric type: "Counter", "Histogram" or "Gauge".
Use "Counter" for any value that increases over time and on which you can compute a rate, such as Bytes or Packets.
Use "Histogram" for any value that must be sampled independently, such as latencies.
Use "Gauge" for other values that don't necessitate accuracy over time (gauges are sampled only every N seconds when Prometheus fetches the metric).<br/>
          <br/>
            <i>Enum</i>: Counter, Histogram, Gauge<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>buckets</b></td>
        <td>[]string</td>
        <td>
          A list of buckets to use when `type` is "Histogram". The list must be parsable as floats. When not set, Prometheus default buckets are used.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowmetricspecchartsindex">charts</a></b></td>
        <td>[]object</td>
        <td>
          Charts configuration, for the OpenShift Console in the administrator view, Dashboards menu.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>direction</b></td>
        <td>enum</td>
        <td>
          Filter for ingress, egress or any direction flows.
When set to `Ingress`, it is equivalent to adding the regular expression filter on `FlowDirection`: `0|2`.
When set to `Egress`, it is equivalent to adding the regular expression filter on `FlowDirection`: `1|2`.<br/>
          <br/>
            <i>Enum</i>: Any, Egress, Ingress<br/>
            <i>Default</i>: Any<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>divider</b></td>
        <td>string</td>
        <td>
          When nonzero, scale factor (divider) of the value. Metric value = Flow value / Divider.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowmetricspecfiltersindex">filters</a></b></td>
        <td>[]object</td>
        <td>
          `filters` is a list of fields and values used to restrict which flows are taken into account.
Refer to the documentation for the list of available fields: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/json-flows-format-reference.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>flatten</b></td>
        <td>[]string</td>
        <td>
          `flatten` is a list of array-type fields that must be flattened, such as Interfaces or NetworkEvents. Flattened fields generate one metric per item in that field.
For instance, when flattening `Interfaces` on a bytes counter, a flow having Interfaces [br-ex, ens5] increases one counter for `br-ex` and another for `ens5`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>help</b></td>
        <td>string</td>
        <td>
          Help text of the metric, as it appears in Prometheus.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>[]string</td>
        <td>
          `labels` is a list of fields that should be used as Prometheus labels, also known as dimensions (for example: `SrcK8S_Namespace`).
From choosing labels results the level of granularity of this metric, and the available aggregations at query time.
It must be done carefully as it impacts the metric cardinality (cf https://rhobs-handbook.netlify.app/products/openshiftmonitoring/telemetry.md/#what-is-the-cardinality-of-a-metric).
In general, avoid setting very high cardinality labels such as IP or MAC addresses.
"SrcK8S_OwnerName" or "DstK8S_OwnerName" should be preferred over "SrcK8S_Name" or "DstK8S_Name" as much as possible.
Refer to the documentation for the list of available fields: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/json-flows-format-reference.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>metricName</b></td>
        <td>string</td>
        <td>
          Name of the metric. In Prometheus, it is automatically prefixed with "netobserv_". Leave empty to generate the name based on the `FlowMetric` resource name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>remap</b></td>
        <td>map[string]string</td>
        <td>
          Set the `remap` property to use different names for the generated metric labels than the flow fields. Use the origin flow fields as keys, and the desired label names as values.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>valueField</b></td>
        <td>string</td>
        <td>
          `valueField` is the flow field that must be used as a value for this metric (for example: `Bytes`). This field must hold numeric values.
Leave empty to count flows rather than a specific value per flow.
Refer to the documentation for the list of available fields: https://docs.redhat.com/en/documentation/openshift_container_platform/latest/html/network_observability/json-flows-format-reference.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowMetric.spec.charts[index]
<sup><sup>[↩ Parent](#flowmetricspec)</sup></sup>



Configures charts / dashboard generation associated to a metric

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dashboardName</b></td>
        <td>string</td>
        <td>
          Name of the containing dashboard. If this name does not refer to an existing dashboard, a new dashboard is created.<br/>
          <br/>
            <i>Default</i>: Main<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#flowmetricspecchartsindexqueriesindex">queries</a></b></td>
        <td>[]object</td>
        <td>
          List of queries to be displayed on this chart. If `type` is `SingleStat` and multiple queries are provided,
this chart is automatically expanded in several panels (one per query).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>title</b></td>
        <td>string</td>
        <td>
          Title of the chart.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type of the chart.<br/>
          <br/>
            <i>Enum</i>: SingleStat, Line, StackArea<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sectionName</b></td>
        <td>string</td>
        <td>
          Name of the containing dashboard section. If this name does not refer to an existing section, a new section is created.
If `sectionName` is omitted or empty, the chart is placed in the global top section.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>unit</b></td>
        <td>enum</td>
        <td>
          Unit of this chart. Only a few units are currently supported. Leave empty to use generic number.<br/>
          <br/>
            <i>Enum</i>: bytes, seconds, Bps, pps, percent, <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowMetric.spec.charts[index].queries[index]
<sup><sup>[↩ Parent](#flowmetricspecchartsindex)</sup></sup>



Configures PromQL queries

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>legend</b></td>
        <td>string</td>
        <td>
          The query legend that applies to each timeseries represented in this chart. When multiple timeseries are displayed, you should set a legend
that distinguishes each of them. It can be done with the following format: `{{ Label }}`. For example, if the `promQL` groups timeseries per
label such as: `sum(rate($METRIC[2m])) by (Label1, Label2)`, you may write as the legend: `Label1={{ Label1 }}, Label2={{ Label2 }}`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>promQL</b></td>
        <td>string</td>
        <td>
          The `promQL` query to be run against Prometheus. If the chart `type` is `SingleStat`, this query should only return
a single timeseries. For other types, a top 7 is displayed.
You can use `$METRIC` to refer to the metric defined in this resource. For example: `sum(rate($METRIC[2m]))`.
To learn more about `promQL`, refer to the Prometheus documentation: https://prometheus.io/docs/prometheus/latest/querying/basics/<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>top</b></td>
        <td>integer</td>
        <td>
          Top N series to display per timestamp. Does not apply to `SingleStat` chart type.<br/>
          <br/>
            <i>Default</i>: 7<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### FlowMetric.spec.filters[index]
<sup><sup>[↩ Parent](#flowmetricspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>field</b></td>
        <td>string</td>
        <td>
          Name of the field to filter on (for example: `SrcK8S_Namespace`).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>matchType</b></td>
        <td>enum</td>
        <td>
          Type of matching to apply<br/>
          <br/>
            <i>Enum</i>: Equal, NotEqual, Presence, Absence, MatchRegex, NotMatchRegex<br/>
            <i>Default</i>: Equal<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value to filter on. When `matchType` is `Equal` or `NotEqual`, you can use field injection with `$(SomeField)` to refer to any other field of the flow.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowMetric.status
<sup><sup>[↩ Parent](#flowmetric)</sup></sup>



FlowMetricStatus defines the observed state of FlowMetric

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowmetricstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          `conditions` represent the latest available observations of an object's state<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>prometheusName</b></td>
        <td>string</td>
        <td>
          Metric name, including prefix, as it appears in Prometheus<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowMetric.status.conditions[index]
<sup><sup>[↩ Parent](#flowmetricstatus)</sup></sup>



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
