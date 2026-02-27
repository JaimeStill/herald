#!/bin/bash

unset location
unset resourceGroup
unset cognitiveService
unset cognitiveServiceSku
unset cognitiveServiceKind
unset cognitiveServiceDomain
unset cognitiveServiceRole
unset modelDeployment
unset modelName
unset modelVersion
unset modelFormat
unset modelSku
unset modelSkuCapacity

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --location)
    location="$2"
    shift
    shift
    ;;
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
  --cognitive-services-sku)
    cognitiveServiceSku="$2"
    shift
    shift
    ;;
  --cognitive-service-kind)
    cognitiveServiceKind="$2"
    shift
    shift
    ;;
  --cognitive-service-domain)
    cognitiveServiceDomain="$2"
    shift
    shift
    ;;
  --cognitive-service-role)
    cognitiveServiceRole="$2"
    shift
    shift
    ;;
  --model-deployment)
    modelDeployment="$2"
    shift
    shift
    ;;
  --model-name)
    modelName="$2"
    shift
    shift
    ;;
  --model-version)
    modelVersion="$2"
    shift
    shift
    ;;
  --model-format)
    modelFormat="$2"
    shift
    shift
    ;;
  --model-sku)
    modelSku="$2"
    shift
    shift
    ;;
  --model-sku-capacity)
    modelSkuCapacity="$2"
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

: "${location:=$DEFAULT_LOCATION}" \
  "${resourceGroup:=$DEFAULT_RESOURCE_GROUP}" \
  "${cognitiveService:=$DEFAULT_COGNITIVE_SERVICE}" \
  "${cognitiveServiceSku:=$DEFAULT_COGNITIVE_SERVICE_SKU}" \
  "${cognitiveServiceKind:=$DEFAULT_COGNITIVE_SERVICE_KIND}" \
  "${cognitiveServiceDomain:=$DEFAULT_COGNITIVE_SERVICE_DOMAIN}" \
  "${cognitiveServiceRole:=$DEFAULT_COGNITIVE_SERVICE_ROLE}" \
  "${modelDeployment:=$DEFAULT_MODEL_DEPLOYMENT}" \
  "${modelName:=$DEFAULT_MODEL_NAME}" \
  "${modelVersion:=$DEFAULT_MODEL_VERSION}" \
  "${modelFormat:=$DEFAULT_MODEL_FORMAT}" \
  "${modelSku:=$DEFAULT_MODEL_SKU}" \
  "${modelSkuCapacity:=$DEFAULT_MODEL_SKU_CAPACITY}"

bash "${SCRIPT_DIR}/components/resource-group.sh" \
  --resource-group "$resourceGroup" \
  --location "$location"

bash "${SCRIPT_DIR}/components/cognitiveservices-account-create.sh" \
  --kind "$cognitiveServiceKind" \
  --location "$location" \
  --name "$cognitiveService" \
  --resource-group "$resourceGroup" \
  --sku "$cognitiveServiceSku" \
  --domain "$cognitiveServiceDomain"

bash "${SCRIPT_DIR}/components/cognitiveservices-deployment-create.sh" \
  --model-format "$modelFormat" \
  --model-name "$modelName" \
  --model-version "$modelVersion" \
  --name "$cognitiveService" \
  --resource-group "$resourceGroup" \
  --deployment-name "$modelDeployment" \
  --sku "$modelSku" \
  --sku-capacity "$modelSkuCapacity"

bash "${SCRIPT_DIR}/components/cognitiveservices-grant-permissions.sh" \
  --name "$cognitiveService" \
  --role "$cognitiveServiceRole" \
  --resource-group "$resourceGroup"
