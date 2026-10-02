#!/usr/bin/env bash

mismatch=0

# Check changes in vanilla CRDs
for crd in "flows.netobserv.io_flowcollectors.yaml" "flows.netobserv.io_flowmetrics.yaml" "flows.netobserv.io_flowcollectorslices.yaml"; do
  echo "Checking CRD $crd..."
  b_changes=$(git diff --shortstat HEAD^1 HEAD bundles/k8s/manifests/$crd)
  if [[ "$b_changes" == "" ]]; then
    echo "  vanilla: (no change)"
  else
    echo "  vanilla: $b_changes"
  fi
  for vendor in "openshift"; do
    v_changes=$(git diff --shortstat HEAD^1 HEAD bundles/$vendor/manifests/$crd)
    if [[ "$v_changes" == "" ]]; then
      echo "  $vendor: (no change)"
    else
      echo "  $vendor: $v_changes"
    fi
    if [[ $b_changes != $v_changes ]]; then
      mismatch=1
      echo "vanilla CRD diff:"
      git diff -U1 HEAD^1 HEAD bundles/k8s/manifests/$crd
      echo ""
      echo "$vendor CRD diff:"
      git diff HEAD^1 HEAD bundles/$vendor/manifests/$crd
      echo ""
      echo "Mismatch detected with $vendor CRD $crd; Did you forget do specify an override in config/$vendor/olm/crd-doc-override/$crd?"
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
