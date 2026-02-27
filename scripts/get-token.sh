#!/bin/bash

az account get-access-token \
  --resource https://cognitiveservices.azure.com \
  --query "accessToken" \
  --output tsv
