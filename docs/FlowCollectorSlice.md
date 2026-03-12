# API Reference

Packages:

- [flows.netobserv.io/v1alpha1](#flowsnetobserviov1alpha1)

# flows.netobserv.io/v1alpha1

Resource Types:

- [FlowCollectorSlice](#flowcollectorslice)




## FlowCollectorSlice
<sup><sup>[↩ Parent](#flowsnetobserviov1alpha1 )</sup></sup>






FlowCollectorSlice is the API allowing to decentralize some of the FlowCollector configuration per namespace tenant.

<table>
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
      <td>FlowCollectorSlice</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#flowcollectorslicespec">spec</a></b></td>
        <td>object</td>
        <td>
          FlowCollectorSliceSpec defines the desired state of FlowCollectorSlice<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorslicestatus">status</a></b></td>
        <td>object</td>
        <td>
          FlowCollectorSliceStatus defines the observed state of FlowCollectorSlice<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollectorSlice.spec
<sup><sup>[↩ Parent](#flowcollectorslice)</sup></sup>



FlowCollectorSliceSpec defines the desired state of FlowCollectorSlice

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>sampling</b></td>
        <td>integer</td>
        <td>
          `sampling` is an optional sampling interval to apply to this slice. For example, a value of `50` means that 1 matching flow in 50 is sampled.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#flowcollectorslicespecsubnetlabelsindex">subnetLabels</a></b></td>
        <td>[]object</td>
        <td>
          `subnetLabels` allows you to customize subnets and IPs labeling, such as to identify cluster external workloads or web services.
External subnets must be labeled with the prefix `EXT:`, or not labeled at all, in order to work with default quick filters and some metrics examples provided.<br/>
Beware that the subnet labels configured in FlowCollectorSlice are not limited to the flows of the related namespace: any flow
in the whole cluster can be labeled using this configuration. However, subnet labels defined in the cluster-scoped FlowCollector take
precedence in case of conflicting rules.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollectorSlice.spec.subnetLabels[index]
<sup><sup>[↩ Parent](#flowcollectorslicespec)</sup></sup>



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


### FlowCollectorSlice.status
<sup><sup>[↩ Parent](#flowcollectorslice)</sup></sup>



FlowCollectorSliceStatus defines the observed state of FlowCollectorSlice

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#flowcollectorslicestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          `conditions` represent the latest available observations of an object's state<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>filterApplied</b></td>
        <td>string</td>
        <td>
          Filter that is applied for flow collection<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subnetLabelsConfigured</b></td>
        <td>integer</td>
        <td>
          Number of subnet labels configured<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### FlowCollectorSlice.status.conditions[index]
<sup><sup>[↩ Parent](#flowcollectorslicestatus)</sup></sup>



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
