#!/bin/bash

unset name
unset role
unset resourceGroup

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --name)
    name="$2"
    shift
    shift
    ;;
  --role)
    role="$2"
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

# options:
# Cognitive Services OpenAI User
# Cognitive Services OpenAI Contributor
# Cognitive Services Contributor
: "${role:=$DEFAULT_COGNITIVE_SERVICE_ROLE}"


principal=$(az ad signed-in-user show --query id --output tsv)

scope=$(
  az cognitiveservices account show \
    --name "$name" \
    --resource-group "$resourceGroup" \
    --query id \
    --output tsv
)

az role assignment create \
  --assignee "$principal" \
  --role "$role" \
  --scope "$scope"
