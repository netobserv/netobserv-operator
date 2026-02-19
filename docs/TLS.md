## TLS and expected certificates

This document lists all required and optional TLS certificates for NetObserv. You can also refer to the [Helm chart templates](../helm/templates/certificates.yaml) for cert-manager.

<table>
  <thead>
    <tr>
      <th>Service name</th>
      <th>Required</th>
      <th>Resource kind</th>
      <th>Resource name</th>
      <th>Resource keys</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>netobserv-webhook-service</td>
      <td>yes</td>
      <td>Secret</td>
      <td>webhook-server-cert</td>
      <td>ca.crt, tls.crt, tls.key</td>
      <td></td>
    </tr>
    <tr>
      <td>netobserv-metrics-service</td>
      <td>yes</td>
      <td>Secret</td>
      <td>manager-metrics-tls</td>
      <td>ca.crt, tls.crt, tls.key</td>
      <td></td>
    </tr>
    <tr>
      <td>flowlogs-pipeline</td>
      <td>no</td>
      <td>Secret</td>
      <td>flowlogs-pipeline-cert</td>
      <td>ca.crt, tls.crt, tls.key</td>
      <td>Only used when spec.deploymentModel is "Service".</td>
    </tr>
    <tr>
      <td>flowlogs-pipeline CA</td>
      <td>no</td>
      <td>ConfigMap</td>
      <td>netobserv-ca</td>
      <td>service-ca.crt</td>
      <td>Must be installed in netobserv-privileged namespace. Only used when spec.deploymentModel is "Service".</td>
    </tr>
  </tbody>
</table>
