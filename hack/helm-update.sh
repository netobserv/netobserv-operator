#!/usr/bin/env bash

# Use gsed on macOS if available (BSD sed doesn't support -i without backup extension)
SED=${SED:-$(command -v gsed 2>/dev/null || echo sed)}

mkdir -p _tmp

# Copy and edit CRDs
for crd in "flows.netobserv.io_flowcollectors.yaml" "flows.netobserv.io_flowmetrics.yaml" "flows.netobserv.io_flowcollectorslices.yaml"; do
  cp "bundles/k8s/manifests/$crd" helm/crds
  $SED -i -E 's/(`[^`]*\{\{[^`]*`)/{{\1}}/g' helm/crds/$crd # escape "{{" for helm
  yq -i 'del(.spec.conversion)' helm/crds/$crd
  yq -i 'del(.spec.versions[] | select(.deprecated == true))' helm/crds/$crd
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/crds/$crd
done

# Copy unchanged files
for file in "netobserv-manager-config_v1_configmap.yaml" "netobserv-metrics-service_v1_service.yaml" "netobserv-webhook-service_v1_service.yaml" ; do
  cp "bundles/k8s/manifests/$file" helm/templates
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/$file
done

# Extract data from clusterserviceversion
yq '.spec.install.spec.clusterPermissions[0]' bundles/k8s/manifests/netobserv-operator.clusterserviceversion.yaml > _tmp/csv-clusterrole.yaml
yq '.spec.install.spec.permissions[0]' bundles/k8s/manifests/netobserv-operator.clusterserviceversion.yaml > _tmp/csv-role.yaml

# Create roles & bindings
yq '{"apiVersion": "v1", "kind": "ServiceAccount", "metadata": {"name": .serviceAccountName}}' _tmp/csv-clusterrole.yaml > helm/templates/serviceaccount.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/serviceaccount.yaml
yq '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRole", "metadata": {"name": "netobserv-manager-role"}, "rules": .rules}' _tmp/csv-clusterrole.yaml > helm/templates/clusterrole.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/clusterrole.yaml
yq '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "ClusterRoleBinding", "metadata": {"name": "netobserv-manager-rolebinding"}, "roleRef": {"apiGroup": "rbac.authorization.k8s.io", "kind": "ClusterRole", "name": "netobserv-manager-role"}, "subjects": [{"kind": "ServiceAccount", "name": .serviceAccountName, "namespace": "{{ .Release.Namespace }}"}]}' _tmp/csv-clusterrole.yaml > helm/templates/clusterrolebinding.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/clusterrolebinding.yaml
yq '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "Role", "metadata": {"name": "netobserv-leader-election-role"}, "rules": .rules}' _tmp/csv-role.yaml > helm/templates/role.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/role.yaml
yq '{"apiVersion": "rbac.authorization.k8s.io/v1", "kind": "RoleBinding", "metadata": {"name": "netobserv-leader-election-rolebinding"}, "roleRef": {"apiGroup": "rbac.authorization.k8s.io", "kind": "Role", "name": "netobserv-leader-election-role"}, "subjects": [{"kind": "ServiceAccount", "name": .serviceAccountName, "namespace": "{{ .Release.Namespace }}"}]}' _tmp/csv-role.yaml > helm/templates/rolebinding.yaml
$SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/rolebinding.yaml

for f in bundles/k8s/manifests/*_rbac.authorization.k8s.io_v1_clusterrole*.yaml; do
  cp "$f" helm/templates/
  $SED -i '1s/^/# Auto-generated from helm-update.sh\n/' helm/templates/$(basename $f)
done
