#!/bin/bash

unset location
unset name
unset resourceGroup

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --location)
    location="$2"
    shift
    shift
    ;;
  --name)
    name="$2"
    shift
    shift
    ;;
  --resource-group)
    resourceGroup="$2"
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

: "${location:=$DEFAULT_LOCATION}"

if [[ $(az cognitiveservices account list --query "[?name=='$name'] | length(@)") -gt 0 ]]; then
  az cognitiveservices account delete \
    --name "$name" \
    --resource-group "$resourceGroup"
fi

if [[ $(az cognitiveservices account list-deleted --query "[?name=='$name'] | length(@)") -gt 0 ]]; then
  az cognitiveservices account purge \
    --location "$location" \
    --name "$name" \
    --resource-group "$resourceGroup"
else
  echo "$name does not exist"
fi
