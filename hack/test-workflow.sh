#!/bin/bash

test_out="test.out"
bundle_csv="bundle/manifests/netobserv-operator.clusterserviceversion.yaml"
short_sha=$(git rev-parse --short=8 HEAD)
fake_tag="1.0.42"

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
    | sed -r "s~\\$\{\{ env\.tag \}\}~$fake_tag~g" \
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
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-operator:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:main" 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:main" 2

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
expect_image_tagged "quay.io/netobserv/network-observability-operator:$fake_tag-amd64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$fake_tag-arm64"
expect_image_tagged "quay.io/netobserv/network-observability-operator:$fake_tag-ppc64le"

run_step "release.yml" "push-image" "build bundle"
expect_image_tagged "quay.io/netobserv/network-observability-operator-bundle:v$fake_tag"
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-operator:1." 2
expect_occurrences $bundle_csv "quay.io/netobserv/netobserv-ebpf-agent:v0." 2
expect_occurrences $bundle_csv "quay.io/netobserv/flowlogs-pipeline:v0." 2
expect_occurrences $bundle_csv "quay.io/netobserv/network-observability-console-plugin:v0." 2

run_step "release.yml" "push-image" "build catalog" "OPM_OPTS=--permissive"
expect_occurrences_at_least $test_out "quay.io/netobserv/network-observability-operator-bundle:v$fake_tag" 1
expect_occurrences $test_out "quay.io/netobserv/network-observability-operator-catalog:v$fake_tag" 2

echo -e "\n✅ Looks good to me!"

# Remove output only on success so it's still there for debugging failures
rm $test_out
