#!/usr/bin/env bash

set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. "$DIR/common.sh"

verifyCommandExists "func"
verifyEnvironmentVariableExists "AZURE_FUNCTION_NAME"

(cd function && func azure functionapp publish "$AZURE_FUNCTION_NAME" --python)
