#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. "$DIR/common.sh"

verifyCommandExists "docker"

IMAGE_TAG=""
while [[ "$#" -gt 0 ]]; do
    case $1 in
        -t|--tag) IMAGE_TAG="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

IMAGE_TAG=${IMAGE_TAG:-latest}

echo "using tag: $IMAGE_TAG"
docker build \
    -f api/Dockerfile api/ \
    --build-arg SERVICE=backend \
    --tag "$IMAGE_TAG"
