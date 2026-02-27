#!/bin/bash

unset modelFormat
unset modelName
unset modelVersion
unset name
unset resourceGroup
unset deploymentName
unset sku
unset skuCapacity

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --model-format)
    modelFormat="$2"
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
  --deployment-name)
    deploymentName="$2"
    shift
    shift
    ;;
  --sku)
    sku="$2"
    shift
    shift
    ;;
  --sku-capacity)
    skuCapacity="$2"
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

az cognitiveservices account deployment create \
  --model-format "$modelFormat" \
  --model-name "$modelName" \
  --model-version "$modelVersion" \
  --name "$name" \
  --resource-group "$resourceGroup" \
  --deployment-name "$deploymentName" \
  --sku "$sku" \
  --sku-capacity "$skuCapacity"
