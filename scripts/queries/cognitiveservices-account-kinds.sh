#!/bin/bash

unset output

POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
  case $1 in
  --output)
    output="$2"
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

: "${output:=$DEFAULT_OUTPUT}"

az cognitiveservices account list-kinds --output $output
