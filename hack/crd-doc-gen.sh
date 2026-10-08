#!/usr/bin/env bash

set -e

# Use gsed on macOS if available (BSD sed doesn't support -i without backup extension)
SED=${SED:-$(command -v gsed 2>/dev/null || echo sed)}
YQ=${YQ:-./bin/yq}
CRDOC=${CRDOC:-./bin/crdoc}

mkdir -p _tmp

# Copy and edit CRDs
for bundle in bundles/*; do
  for crd in "flows.netobserv.io_flowcollectors.yaml" "flows.netobserv.io_flowmetrics.yaml" "flows.netobserv.io_flowcollectorslices.yaml"; do
    cp $bundle/manifests/$crd _tmp/
    $YQ -i 'del(.spec.conversion)' _tmp/$crd
    $YQ -i 'del(.spec.versions[] | select(.deprecated == true))' _tmp/$crd
  done

  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ipfix)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.resources.properties.claims)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.advanced.properties.scheduling.properties.affinity.properties)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.agent.properties.ebpf.properties.advanced.properties.scheduling.properties.tolerations.items)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.resources.properties.claims)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.advanced.properties.scheduling.properties.affinity.properties)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.advanced.properties.scheduling.properties.tolerations.items)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.resources.properties.claims)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.autoscaler.properties)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.advanced.properties.scheduling.properties.affinity.properties)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.consolePlugin.properties.advanced.properties.scheduling.properties.tolerations.items)' _tmp/flows.netobserv.io_flowcollectors.yaml
  $YQ -i 'del(.spec.versions[].schema.openAPIV3Schema.properties.spec.properties.processor.properties.kafkaConsumerAutoscaler.properties)' _tmp/flows.netobserv.io_flowcollectors.yaml

  target=$(basename $bundle)
  if [[ $target == "k8s" ]]; then
    target="."
  else
    mkdir -p docs/$target
  fi
	$CRDOC --resources _tmp/flows.netobserv.io_flowcollectors.yaml --output docs/$target/FlowCollector.md
	$CRDOC --resources _tmp/flows.netobserv.io_flowmetrics.yaml --output docs/$target/FlowMetric.md
	$CRDOC --resources _tmp/flows.netobserv.io_flowcollectorslices.yaml --output docs/$target/FlowCollectorSlice.md
done
