#!/usr/bin/env bash

set -euo pipefail


function verifyCommandExists() {
    echo "❓ is $1 installed?"
    if ! command -v "$1" &> /dev/null
    then
        echo "⛔ $1 is not installed. Please install and re-run"
        exit 1
    else
        echo "✅ $1 is installed"
    fi
}

function verifyEnvironmentVariableExists() {
    echo "❓ is variable $1 set?"
    VAR_NAME="$1"
    if [ -z "${VAR_NAME+x}" ]; 
    then
        echo "⛔ variable $1 is not set. Please set and re-run"
        exit 1
    else
        echo "✅ variable $1 is set"
    fi
}