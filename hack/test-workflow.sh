#!/bin/bash

test_out="test.out"
bundle_csv="bundle/manifests/netobserv-operator.clusterserviceversion.yaml"
short_sha=$(git rev-parse --short=8 HEAD)
# Digest tests need a release whose operator and related images exist in Quay.
release_tag="1.12.0-community"

clean_up() {
    ARG=$?
    rm -r bundle_tmp*
    exit $ARG
} 
trap clean_up EXIT

run_step() {
  file=$1
  job=$2
  name=$3
  opts=$4

  version=$(cat .github/workflows/$file | ./bin/yq ".env.WF_VERSION")
  if [[ $version == '${{ github.ref_name }}' ]]; then
    version=main
  fi
  step=$(cat .github/workflows/$file | ./bin/yq ".jobs.$job.steps[] | select(.name==\"$name\").run")
  step=$(echo "$step" \
    | sed -r "s~\\$\{\{ env\.WF_ORG \}\}~netobserv~g" \
    | sed -r "s~\\$\{\{ env\.WF_VERSION \}\}~$version~g" \
    | sed -r "s~\\$\{\{ env\.WF_REGISTRY \}\}~quay.io/netobserv~g" \
    | sed -r "s~\\$\{\{ env\.WF_IMAGE \}\}~network-observability-operator~g" \
    | sed -r "s~\\$\{\{ env\.WF_MULTIARCH_TARGETS \}\}~amd64 arm64 ppc64le s390x~g" \
    | sed -r "s~\\$\{\{ env\.short_sha \}\}~$short_sha~g" \
    | sed -r "s~\\$\{\{ env\.tag \}\}~$release_tag~g" \
  )
  step="$opts $step"

  echo "↘️  Running step '$name' ($file)"
  echo "$step"
  eval "$step" > $test_out 2>&1

  if [ $? -ne 0 ]; then
      echo "❌ Step failed"
      exit 1
  fi
}

expect_image_tagged() {
  img=$1
  cat $test_out | grep "Successfully tagged $img"
  if [ $? -ne 0 ]; then
      echo "❌ Failure: expected successful tag $img"
      exit 1
  fi
}

expect_occurrences() {
  file=$1
  search=$2
  expected=$3
  found=$(cat $file | grep -o "$search" | wc -l)
  if [ $found -ne $expected ]; then
      echo "❌ Failure: expected $expected occurrences of \"$search\" in $file, found $found."
      exit 1
  fi
}

expect_occurrences_at_least() {
  file=$1
  search=$2
  min=$3
  found=$(cat $file | grep -o "$search" | wc -l)
  if [ $found -lt $min ]; then
      echo "❌ Failure: expected at least $min occurrences of \"$search\" in $file, found $found."
      exit 1
  fi
}

expect_digest_field() {
  query=$1
  image=$2
  label=$3
  reference=$(yq "$query" "$bundle_csv")
  digest=${reference#"$image"@}

  if [[ "$reference" != "$image@$digest" || ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
      echo "❌ Failure: expected $label to contain a complete digest reference, found \"$reference\"."
      exit 1
  fi
}

expect_pinned_bundle_images() {
  operator_image="quay.io/netobserv/network-observability-operator"
  bpf_image="quay.io/netobserv/netobserv-ebpf-agent"
  flp_image="quay.io/netobserv/flowlogs-pipeline"
  plugin_image="quay.io/netobserv/network-observability-console-plugin"
  deployment='.spec.install.spec.deployments[] | select(.name == "netobserv-controller-manager")'
  container="$deployment.spec.template.spec.containers[] | select(.name == \"manager\")"

  expect_digest_field '.metadata.annotations.containerImage' "$operator_image" 'containerImage annotation'
  expect_digest_field "$container.image" "$operator_image" 'operator deployment image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_EBPF_AGENT\").value" "$bpf_image" 'eBPF environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "ebpf-agent").image' "$bpf_image" 'eBPF related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_FLOWLOGS_PIPELINE\").value" "$flp_image" 'FLP environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "flowlogs-pipeline").image' "$flp_image" 'FLP related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE\").value" "$plugin_image" 'console environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "web-console").image' "$plugin_image" 'console related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE_PF4\").value" "$plugin_image" 'PF4 console environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "web-console-pf4").image' "$plugin_image" 'PF4 console related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE_PF5\").value" "$plugin_image" 'PF5 console environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "web-console-pf5").image' "$plugin_image" 'PF5 console related image'
}

echo -e "🥁🥁🥁 TESTING push_image_pr.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "push_image_pr.yml" "push-pr-image" "build image"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$short_sha-amd64"

run_step "push_image_pr.yml" "push-pr-image" "build bundle"
expect_image_tagged "quay.io/netobserv/network-observability-operator-bundle:v0.0.0-sha-$short_sha"
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-operator:$short_sha" 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main" 2

run_step "push_image_pr.yml" "push-pr-image" "build catalog" "OPM_OPTS=--permissive"
expect_occurrences_at_least $test_out "quay.io/netobserv/network-observability-operator-bundle:v0.0.0-sha-$short_sha" 1
expect_image_tagged "quay.io/netobserv/network-observability-operator-catalog:v0.0.0-sha-$short_sha"

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING push_image.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "push_image.yml" "push-image" "build images"
expect_image_tagged "quay.io/netobserv/network-observability-operator:main-amd64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:main-arm64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:main-ppc64le"
expect_image_tagged "quay.io/netobserv/network-observability-operator:main-s390x"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$short_sha-amd64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$short_sha-arm64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$short_sha-ppc64le"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$short_sha-s390x"

run_step "push_image.yml" "push-image" "build bundle"
expect_image_tagged "quay.io/netobserv/network-observability-operator-bundle:v0.0.0-main"
expect_pinned_bundle_images

run_step "push_image.yml" "push-image" "build catalog" "OPM_OPTS=--permissive"
expect_occurrences_at_least $test_out "quay.io/netobserv/network-observability-operator-bundle:v0.0.0-main" 1
expect_occurrences $test_out "quay.io/netobserv/network-observability-operator-catalog:v0.0.0-main" 2

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING make update-bundle 🥁🥁🥁"

make update-bundle > $test_out 2>&1
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-operator:1." 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:v0." 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:v0." 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:v0." 2

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING release.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "release.yml" "push-image" "build operator"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$release_tag-amd64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$release_tag-arm64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$release_tag-ppc64le"

run_step "release.yml" "push-image" "build bundle"
expect_image_tagged "quay.io/netobserv/network-observability-operator-bundle:v$release_tag"
expect_pinned_bundle_images

run_step "release.yml" "push-image" "build catalog" "OPM_OPTS=--permissive"
expect_occurrences_at_least $test_out "quay.io/netobserv/network-observability-operator-bundle:v$release_tag" 1
expect_occurrences $test_out "quay.io/netobserv/network-observability-operator-catalog:v$release_tag" 2

echo -e "\n✅ Looks good to me!"

# Remove output only on success so it's still there for debugging failures
rm $test_out
