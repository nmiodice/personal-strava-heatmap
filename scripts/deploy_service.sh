#!/usr/bin/env bash

set -eo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
. "$DIR/common.sh"

verifyEnvironmentVariableExists "ARM_CLIENT_ID"
verifyEnvironmentVariableExists "ARM_CLIENT_SECRET"
verifyEnvironmentVariableExists "ARM_TENANT_ID"
verifyEnvironmentVariableExists "ACR_NAME"

TAG="$ACR_NAME.azurecr.io/backend:$(git rev-parse --short HEAD)"
"$DIR/build_service.sh" --tag "$TAG"

logSuccess "container built"

az acr login --name "$ACR_NAME" --username "$ARM_CLIENT_ID" --password "$ARM_CLIENT_SECRET"
logSuccess "logged into container registry"

docker push "$TAG"
logSuccess "image pushed to container registry"

az login --service-principal \
    --username "$ARM_CLIENT_ID" \
    --password "$ARM_CLIENT_SECRET" \
    --tenant "$ARM_TENANT_ID"
logSuccess "logged into azure CLI"

az webapp config container set \
    --name "$WEBAPP_NAME" \
    --resource-group "$WEBAPP_RESOURCE_GROUP" \
    --docker-custom-image-name "$TAG"
logSuccess "webapp container configuration updated"
