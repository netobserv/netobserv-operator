#!/usr/bin/env bash

mismatch=0

# Check changes in vanilla CRDs
for kind in "flowcollectors" "flowmetrics" "flowcollectorslices"; do
  echo "Checking $kind CRD..."
  helm_name="apiextensions.k8s.io_v1_customresourcedefinition_$kind.flows.netobserv.io.yaml"
  olm_name="flows.netobserv.io_$kind.yaml"
  b_changes=$(git diff --shortstat HEAD^1 HEAD helm/templates/$helm_name)
  if [[ "$b_changes" == "" ]]; then
    echo "  vanilla: (no change)"
  else
    echo "  vanilla: $b_changes"
  fi
  for vendor in "openshift"; do
    v_changes=$(git diff --shortstat HEAD^1 HEAD bundles/$vendor/manifests/$olm_name)
    if [[ "$v_changes" == "" ]]; then
      echo "  $vendor: (no change)"
    else
      echo "  $vendor: $v_changes"
    fi
    if [[ $b_changes != $v_changes ]]; then
      mismatch=1
      echo "vanilla CRD diff:"
      git diff -U1 HEAD^1 HEAD helm/templates/$helm_name
      echo ""
      echo "$vendor CRD diff:"
      git diff HEAD^1 HEAD bundles/$vendor/manifests/$olm_name
      echo ""
      echo "Mismatch detected with $vendor $kind CRD; Did you forget do specify an override in config/$vendor/olm/crd-doc-override/$helm_name?"
      echo "This can be an expected divergence, or a false-positive. In that case, just ignore this report."
      echo ""
    fi
  done
done

if [[ $mismatch == 1 ]]; then
  exit 1
fi

echo ""
echo "No mismatch found."
