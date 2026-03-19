#!/bin/bash

# Helper tool to update the catalog artifacts and bundle artifacts
# These genereated artifacts are used to build the catalog image
# Pre-requisites: opm, make
# Usage: ./hack/update_fbc.sh

set -euo pipefail

: ${OPM:=$(command -v opm)}
echo "using opm from ${OPM}"
# check if opm version is v1.39.0 or exit
if [ -z "${OPM}" ]; then
  echo "opm is required"
  exit 1
fi

: ${YQ:=$(command -v yq)}
echo "using yq from ${YQ}"
# check if yq exists
if [ -z "${YQ}" ]; then
  echo "yq is required"
  exit 1
fi

echo "Creating new catalog..."

dir_catalog="catalog/out/${BUNDLE_TAG}"
dir_catalog_legacy="catalog/out-legacy/${BUNDLE_TAG}"
mkdir -p "${dir_catalog}"
mkdir -p "${dir_catalog_legacy}"
cp -f catalog/parts/other.yaml ${dir_catalog}
cp -f catalog/parts/other.yaml ${dir_catalog_legacy}

${OPM} render ./bundle --output=yaml --migrate-level=bundle-object-to-csv-metadata > "${dir_catalog}/bundle.yaml"
${OPM} render ./bundle --output=yaml > "${dir_catalog_legacy}/bundle.yaml"

echo "Generating single index..."
cat <<EOF > "${dir_catalog}/index.yaml"
---
entries:
  - name: netobserv-operator.${BUNDLE_TAG}
name: latest
package: netobserv-operator
schema: olm.channel
EOF

cp -f "${dir_catalog}/index.yaml" "${dir_catalog_legacy}"

echo "Finished running $(basename "$0")"
