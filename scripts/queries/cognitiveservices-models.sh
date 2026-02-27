#!/bin/bash

unset location
unset output
unset model
unset format
unset kind
unset query
unset conditions

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --location)
    location="$2"
    shift
    shift
    ;;
  --output)
    output="$2"
    shift
    shift
    ;;
  --model)
    model="$2"
    shift
    shift
    ;;
  --format)
    format="$2"
    shift
    shift
    ;;
  --kind)
    kind="$2"
    shift
    shift
    ;;
  *)
    POSITIONAL_ARGS+=("$1")
    shift
    ;;
  esac
done

set -- "${POSITIONAL_ARGS[0]}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../defaults.sh"

: "${location:=$DEFAULT_LOCATION}" \
  "${output:=$DEFAULT_OUTPUT}"

query="[]"
conditions=()

[[ -n "$model" ]] && conditions+=("contains(model.name, '$model')")
[[ -n "$format" ]] && conditions+=("contains(model.format, '$format')")

if [[ -n "$kind" ]]; then
  conditions+=("contains(kind, '$kind')")
else
  conditions+=("kind != 'MaaS'")
fi

if [[ ${#conditions[@]} -gt 0 ]]; then
  query="[?$(printf '%s && ' "${conditions[@]}" | sed 's/ && $//')]"
fi

az cognitiveservices model list \
  --location $location \
  --query "${query}.{kind:kind, name:model.name, version:model.version, format:model.format, skuName:model.skus[0].name, skuCapacity:model.skus[0].capacity.default, capabilities:model.capabilities, skus:model.skus[]}" \
  --output $output
