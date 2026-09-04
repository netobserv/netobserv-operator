#!/usr/bin/env bash

version="$1"

if [[ $version == "" ]]; then
  echo "Missing CRD version."
  exit 1
fi

crd="./config/crd/bases/flows.netobserv.io_flowcollectors.yaml"
csv="./config/csv/bases/transformed-csv.yaml"

crdRoot=".spec.versions[] | select(.name==\"$version\").schema.openAPIV3Schema.properties.spec.properties"
csvRoot=".spec.customresourcedefinitions.owned[] | select(.version==\"$version\").specDescriptors"

process_property() {
  local yamlPath=$1
  local logicPath=$2
  # Ignore some patterns
  if [[ $logicPath == *"utoscaler" ]] || [[ $logicPath == *".resources" ]] || [[ $logicPath == *".tls" ]] || [[ $logicPath == *".statusTls" ]] ; then
    # echo "Ignoring entry" >&2
    false
    return
  fi
  # Lookup in CSV
  local xDescs=$(cat $csv | ./bin/yq "$csvRoot[] | select(.path==\"$logicPath\").x-descriptors")
  if [[ $xDescs == *":hidden" ]]; then
    # echo "Hidden property; ignoring" >&2
    false
    return
  fi
  local displayName=$(echo $logicPath | sed 's/.*\.//' | sed 's/\([a-z]\)\([A-Z]\)/\1 \2/g' )
  displayName=${displayName,,} # lower case
  displayName=${displayName^} # first letter capital
#  local description=$(cat $crd | ./bin/yq "$yamlPath.description" | sed 's/`/"/g' | sed 's/<br>//g')
  if [[ $(cat $csv | ./bin/yq "$csvRoot[] | select(.path==\"$logicPath\")") != "" ]]; then
    if [[ $(cat $csv | ./bin/yq "$csvRoot[] | select(.path==\"$logicPath\") | has(\"displayName\")") == false ]]; then
      # echo "Set display name: $displayName" >&2
      ./bin/yq -i "($csvRoot[] | select(.path==\"$logicPath\").displayName) = \"$displayName\"" $csv
    fi
    # if [[ $(cat $csv | ./bin/yq "$csvRoot[] | select(.path==\"$logicPath\") | has(\"description\")") == false ]]; then
    #   echo "Set description: $description" >&2
    #   # ./bin/yq -i "$csvRoot[] | select(.path==\"$logicPath\").description = \"$description\"" $csv
    # fi
  else
    # echo "Creating entry for $displayName" >&2
    ./bin/yq -i "($csvRoot) += {\"path\":\"$logicPath\",\"displayName\":\"$displayName\"}" $csv
  fi
}

process_properties() {
  local yamlPath=$1
  local logicPath=$2
  local props=$(cat $crd | ./bin/yq "$yamlPath | keys | @tsv")
  for prop in $props; do
    if [[ $logicPath != "" ]]; then
      local propLogicPath="$logicPath.$prop"
    else
      local propLogicPath="$prop"
    fi
    # echo "Checking property $propLogicPath" >&2
    if process_property "$yamlPath.$prop" $propLogicPath; then
      local type=$(cat $crd | ./bin/yq "$yamlPath.$prop.type")
      if [[ "$type" == "object" ]]; then
        # echo "it's an object" >&2
        if [[ $(cat $crd | ./bin/yq "$yamlPath.$prop | has(\"properties\")") == true ]]; then
          process_properties "$yamlPath.$prop.properties" $propLogicPath
        # else
          # echo "Skipping: no properties." >&2
        fi
      # else
      #   if [[ "$type" == "array" ]]; then
      #     echo "it's an array" >&2
      #   # else
      #     # echo "it's a leaf" >&2
      #   fi
      fi
    fi
  done
}

process_properties "$crdRoot" ""
