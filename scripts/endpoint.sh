#!/bin/bash

unset resourceGroup
unset cognitiveService

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --resource-group)
    resourceGroup="$2"
    shift
    shift
    ;;
  --cognitive-service)
    cognitiveService="$2"
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
source "${SCRIPT_DIR}/defaults.sh"

: "${resourceGroup:=$DEFAULT_RESOURCE_GROUP}" \
  "${cognitiveService:=$DEFAULT_COGNITIVE_SERVICE}"

az cognitiveservices account show \
  --name "$cognitiveService" \
  --resource-group "$resourceGroup" |
  jq -r .properties.endpoint
