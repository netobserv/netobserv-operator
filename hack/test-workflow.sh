#!/bin/bash

test_out="test.out"
bundle_csv="bundles/openshift/manifests/netobserv-operator.clusterserviceversion.yaml"
test_registry="${TEST_REGISTRY:-quay.io/netobserv}"
test_image_org="${TEST_IMAGE_ORG:-netobserv}"
operator_image="$test_registry/network-observability-operator"
official_operator_image="quay.io/netobserv/network-observability-operator"
bundle_image="$operator_image-bundle"
catalog_image="$operator_image-catalog"
short_sha=$(git rev-parse --short=8 HEAD)
# Digest tests need a release whose operator and related images exist in Quay.
release_tag="1.12.0-community"

clean_up() {
    ARG=$?
    rm -r bundle_tmp* 2>/dev/null || true
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
  related_version=$(cat .github/workflows/$file | ./bin/yq '.env.WF_RELIMG_VERSION // ""')
  WF_ORG=$test_image_org
  WF_VERSION=$version
  WF_REGISTRY=$test_registry
  WF_IMAGE=network-observability-operator
  WF_MULTIARCH_TARGETS="amd64 arm64 ppc64le s390x"
  WF_RELIMG_VERSION=$related_version
  step=$(cat .github/workflows/$file | ./bin/yq ".jobs.$job.steps[] | select(.name==\"$name\").run")
  step=$(echo "$step" \
    | sed -r "s~\\$\{\{ env\.WF_ORG \}\}~$test_image_org~g" \
    | sed -r "s~\\$\{\{ env\.WF_VERSION \}\}~$version~g" \
    | sed -r "s~\\$\{\{ env\.WF_REGISTRY \}\}~$test_registry~g" \
    | sed -r "s~\\$\{\{ env\.WF_IMAGE \}\}~network-observability-operator~g" \
    | sed -r "s~\\$\{\{ env\.WF_MULTIARCH_TARGETS \}\}~amd64 arm64 ppc64le s390x~g" \
    | sed -r "s~\\$\{\{ env\.WF_RELIMG_VERSION \}\}~$related_version~g" \
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
  if ! docker image inspect "$img" >/dev/null 2>&1; then
      echo "❌ Failure: expected local image $img"
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

expect_digest_field() {
  query=$1
  image=$2
  label=$3
  reference=$(./bin/yq "$query" "$bundle_csv")
  digest=${reference#"$image"@}

  if [[ "$reference" != "$image@$digest" || ! "$digest" =~ ^sha256:[0-9a-f]{64}$ ]]; then
      echo "❌ Failure: expected $label to contain a complete digest reference, found \"$reference\"."
      exit 1
  fi

  echo "✅ $label: $reference"
}

expect_pinned_bundle_images() {
  include_pf=${1:-false}
  bpf_image="quay.io/netobserv/netobserv-ebpf-agent"
  flp_image="quay.io/netobserv/flowlogs-pipeline"
  plugin_image="quay.io/netobserv/network-observability-console-plugin"
  deployment='.spec.install.spec.deployments[] | select(.name == "netobserv-controller-manager")'
  container="$deployment.spec.template.spec.containers[] | select(.name == \"manager\")"

  expect_digest_field '.metadata.annotations.containerImage' "$official_operator_image" 'containerImage annotation'
  expect_digest_field "$container.image" "$official_operator_image" 'operator deployment image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_EBPF_AGENT\").value" "$bpf_image" 'eBPF environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "ebpf-agent").image' "$bpf_image" 'eBPF related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_FLOWLOGS_PIPELINE\").value" "$flp_image" 'FLP environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "flowlogs-pipeline").image' "$flp_image" 'FLP related image'
  expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE\").value" "$plugin_image" 'console environment image'
  expect_digest_field '.spec.relatedImages[] | select(.name == "web-console").image' "$plugin_image" 'console related image'
  if [[ $include_pf == true ]]; then
    expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE_PF4\").value" "$plugin_image" 'PF4 console environment image'
    expect_digest_field '.spec.relatedImages[] | select(.name == "web-console-pf4").image' "$plugin_image" 'PF4 console related image'
    expect_digest_field "$container.env[] | select(.name == \"RELATED_IMAGE_WEB_CONSOLE_PF5\").value" "$plugin_image" 'PF5 console environment image'
    expect_digest_field '.spec.relatedImages[] | select(.name == "web-console-pf5").image' "$plugin_image" 'PF5 console related image'
  fi
}

echo -e "🥁🥁🥁 TESTING build_image_pr.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "build_image_pr.yml" "build-pr-image" "build and save operator image"
expect_image_tagged "$operator_image:$short_sha-amd64"

run_step "build_image_pr.yml" "build-pr-image" "build and save bundle image"
expect_image_tagged "$bundle_image:v0.0.0-sha-$short_sha"
expect_occurrences $bundle_csv "$operator_image:$short_sha" 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main-pf4$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main-pf5$" 2

run_step "build_image_pr.yml" "build-pr-image" "build and save catalog image" "OPM_OPTS=--permissive"
expect_image_tagged "$catalog_image:v0.0.0-sha-$short_sha"

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING push_image.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "push_image.yml" "push-image" "build images"
expect_image_tagged "$operator_image:main-amd64"
expect_image_tagged "$operator_image:main-arm64"
expect_image_tagged "$operator_image:main-ppc64le"
expect_image_tagged "$operator_image:main-s390x"
expect_image_tagged "$operator_image:$short_sha-amd64"
expect_image_tagged "$operator_image:$short_sha-arm64"
expect_image_tagged "$operator_image:$short_sha-ppc64le"
expect_image_tagged "$operator_image:$short_sha-s390x"

run_step "push_image.yml" "push-image" "build bundle"
expect_image_tagged "$bundle_image:v0.0.0-sha-main"
expect_occurrences $bundle_csv "$official_operator_image:main$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:main$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:main$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main-pf4$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main-pf5$" 2

run_step "push_image.yml" "push-image" "build catalog" "OPM_OPTS=--permissive"
expect_image_tagged "$catalog_image:v0.0.0-sha-main"

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING make update-bundle 🥁🥁🥁"

make update-bundle > $test_out 2>&1
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-operator:${release_tag}$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:v${release_tag}$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:v${release_tag}$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:v${release_tag}$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:v${release_tag}-pf4$" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:v${release_tag}-pf5$" 2

echo -e "✅\n"
echo -e "🥁🥁🥁 TESTING release.yml 🥁🥁🥁"

# we only test images here as manifest-build need images to be pushed
run_step "release.yml" "push-image" "build operator"
expect_image_tagged "$operator_image:$release_tag-amd64"
expect_image_tagged "$operator_image:$release_tag-arm64"
expect_image_tagged "$operator_image:$release_tag-ppc64le"

run_step "release.yml" "push-image" "build bundle"
expect_image_tagged "$bundle_image:v$release_tag"
expect_pinned_bundle_images

run_step "release.yml" "push-image" "build catalog" "OPM_OPTS=--permissive"
expect_image_tagged "$catalog_image:v$release_tag"

echo -e "\n✅ Looks good to me!"

# Remove output only on success so it's still there for debugging failures
rm $test_out
