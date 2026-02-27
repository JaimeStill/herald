#!/bin/bash

unset resourceGroup
unset location
unset cognitiveService

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --resource-group)
    resourceGroup="$2"
    shift
    shift
    ;;
  --location)
    location="$2"
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

# default value if not provided
: "${resourceGroup:=$DEFAULT_RESOURCE_GROUP}" \
  "${location:=$DEFAULT_LOCATION}" \
  "${cognitiveService:=$DEFAULT_COGNITIVE_SERVICE}"

if [[ $(az group list --query "[?name=='$resourceGroup'] | length(@)") -gt 0 ]]; then
  az group delete --resource-group "$resourceGroup" -y
else
  echo "$resourceGroup does not exist"
fi

bash "${SCRIPT_DIR}/components/cognitiveservices-account-purge.sh" \
  --location "$location" \
  --name "$cognitiveService" \
  --resource-group "$resourceGroup"
