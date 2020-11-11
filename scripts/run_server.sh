#!/usr/bin/env bash

set -euo pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
(cd "$DIR/../api" && go run github.com/nmiodice/personal-strava-heatmap/cmd/backend)
