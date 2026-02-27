#!/bin/bash

unset kind
unset location
unset name
unset resourceGroup
unset sku
unset domain

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --kind)
    kind="$2"
    shift
    shift
    ;;
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
  --sku)
    sku="$2"
    shift
    shift
    ;;
  --domain)
    domain="$2"
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
  "${sku:=$DEFAULT_COGNITIVE_SERVICE_SKU}"

if [[ $(az cognitiveservices account list --query "[?name=='$name'] | length(@)") -gt 0 ]]; then
  echo "$name already exists"
else
  az cognitiveservices account create \
    --kind "$kind" \
    --location "$location" \
    --name "$name" \
    --resource-group "$resourceGroup" \
    --sku "$sku" \
    --custom-domain "$domain"
fi
