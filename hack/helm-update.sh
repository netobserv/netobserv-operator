#!/usr/bin/env bash

# Use gsed on macOS if available (BSD sed doesn't support -i without backup extension)
SED=${SED:-$(command -v gsed 2>/dev/null || echo sed)}

mkdir -p _tmp

YQ=${YQ:-./bin/yq}

# Copy and edit CRDs
for crd in "flows.netobserv.io_flowcollectors.yaml" "flows.netobserv.io_flowmetrics.yaml" "flows.netobserv.io_flowcollectorslices.yaml"; do
  cp "bundles/k8s/manifests/$crd" helm/crds
  $SED -i -E 's/(`[^`]*\{\{[^`]*`)/{{\1}}/g' helm/crds/$crd # escape "{{" for helm
  $YQ -i 'del(.spec.conversion)' helm/crds/$crd
  $YQ -i 'del(.spec.versions[] | select(.deprecated == true))' helm/crds/$crd
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/crds/$crd
done

# Copy unchanged files
for file in "netobserv-manager-config_v1_configmap.yaml" "netobserv-metrics-service_v1_service.yaml" "netobserv-webhook-service_v1_service.yaml" ; do
  cp "bundles/k8s/manifests/$file" helm/templates
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/$file
done

# Services: remove openshift annotations for certificates (and some kubeconfig labels)
$YQ -i 'del(.metadata.annotations)' helm/templates/netobserv-metrics-service_v1_service.yaml
$YQ -i 'del(.metadata.annotations)' helm/templates/netobserv-webhook-service_v1_service.yaml
$YQ -i 'del(.metadata.labels)' helm/templates/netobserv-webhook-service_v1_service.yaml

# Extract data from clusterserviceversion
$YQ '.spec.install.spec.clusterPermissions[0]' bundles/k8s/manifests/netobserv-operator.clusterserviceversion.yaml > _tmp/csv-clusterrole.yaml
$YQ '.spec.install.spec.permissions[0]' bundles/k8s/manifests/netobserv-operator.clusterserviceversion.yaml > _tmp/csv-role.yaml

# Create roles & bindings
$YQ '{"apiVersion": "v1", "kind": "ServiceAccount", "metadata": {"name": .serviceAccountName}}' _tmp/csv-clusterrole.yaml > helm/templates/serviceaccount.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/serviceaccount.yaml
$YQ '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole", "metadata": {"name": "netobserv-manager-role"}, "rules": .rules}' _tmp/csv-clusterrole.yaml > helm/templates/clusterrole.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/clusterrole.yaml
$YQ '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding", "metadata": {"name": "netobserv-manager-rolebinding"}, "roleRef": {"apiGroup": "rbac.authorization.k8s.io", "kind": "ClusterRole", "name": "netobserv-manager-role"}, "subjects": [{"kind": "ServiceAccount", "name": .serviceAccountName, "namespace": "{{ .Release.Namespace }}"}]}' _tmp/csv-clusterrole.yaml > helm/templates/clusterrolebinding.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/clusterrolebinding.yaml
$YQ '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "Role", "metadata": {"name": "netobserv-leader-election-role"}, "rules": .rules}' _tmp/csv-role.yaml > helm/templates/role.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/role.yaml
$YQ '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "RoleBinding", "metadata": {"name": "netobserv-leader-election-rolebinding"}, "roleRef": {"apiGroup": "rbac.authorization.k8s.io", "kind": "Role", "name": "netobserv-leader-election-role"}, "subjects": [{"kind": "ServiceAccount", "name": .serviceAccountName, "namespace": "{{ .Release.Namespace }}"}]}' _tmp/csv-role.yaml > helm/templates/rolebinding.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/rolebinding.yaml

for f in bundles/k8s/manifests/*_rbac.authorization.k8s.io_v1_clusterrole.yaml; do
  cp "$f" helm/templates/
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/$(basename $f)
done
