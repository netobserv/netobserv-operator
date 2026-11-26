#!/usr/bin/env bash

set -e

# Use gsed on macOS if available (BSD sed doesn't support -i without backup extension)
SED=${SED:-$(command -v gsed 2>/dev/null || echo sed)}
YQ=${YQ:-./bin/yq}
CRDOC=${CRDOC:-./bin/crdoc}

mkdir -p _tmp

generate() {
  local downstream_bundle=$1

  if [[ "$downstream_bundle" == "" ]]; then
    # Helm / community
    target="docs/"
  else
    # Downstream OLM bundle
    target="docs/$(basename $downstream_bundle)/"
    mkdir -p $target
  fi

  for kind in "FlowCollector" "FlowMetric" "FlowCollectorSlice"; do
    plural="$(printf '%s' "$kind" | tr '[:upper:]' '[:lower:]')s"

    if [[ "$downstream_bundle" == "" ]]; then
      # Helm / community
      crd="apiextensions.k8s.io_v1_customresourcedefinition_$plural.flows.netobserv.io.yaml"
      cp "helm/templates/$crd" _tmp/
    else
      # Downstream OLM bundle
      crd="flows.netobserv.io_$plural.yaml"
      cp "$downstream_bundle/manifests/$crd" _tmp/
    fi

    $YQ -i 'del(.spec.conversion) | del(.spec.versions[] | select(.deprecated == true))' _tmp/$crd

    if [[ "$kind" == "FlowCollector" ]]; then
      $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ipfix)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.resources.properties.claims)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.advanced.properties.scheduling.properties.affinity.properties)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.advanced.properties.scheduling.properties.tolerations.items)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.resources.properties.claims)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.advanced.properties.scheduling.properties.affinity.properties)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.advanced.properties.scheduling.properties.tolerations.items)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.resources.properties.claims)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.autoscaler.properties)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.advanced.properties.scheduling.properties.affinity.properties)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.advanced.properties.scheduling.properties.tolerations.items)
        | del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.kafkaConsumerAutoscaler.properties)
        ' _tmp/$crd
    fi

  	$CRDOC --resources _tmp/$crd --output $target/$kind.md

    if [[ "$downstream_bundle" == "" ]]; then
      # Remove helm templating artifacts such as {{`{{ Label }}`}}
      $SED -i -r 's/\{\{`/`/g' $target/$kind.md
      $SED -i -r 's/`}}/`/g' $target/$kind.md
    fi
  done
}

# Helm / community
generate

# Downstream OLM bundle
for bundle in bundles/*; do
  generate $bundle
done
