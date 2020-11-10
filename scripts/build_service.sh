#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. "$DIR/common.sh"

verifyCommandExists "docker"

docker build -f api/Dockerfile api/ --build-arg SERVICE=backend
