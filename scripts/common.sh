#!/usr/bin/env bash

function verifyCommandExists() {
    echo "❓ is $1 installed?"
    if ! command -v "$1" &> /dev/null
    then
        logFailure "$1 is not installed. Please install and re-run"
        exit 1
    else
        logSuccess "$1 is installed"
    fi
}

function verifyEnvironmentVariableExists() {
    echo "❓ is variable $1 set?"
    VAR_NAME="$1"
    if [ -z "${!VAR_NAME}" ]; 
    then
        logFailure "variable $1 is not set. Please set and re-run"
        exit 1
    else
        logSuccess "variable $1 is set"
    fi
}

function logSuccess() {
    echo "✅ $1"
}

function logFailure() {
    echo "⛔ $1"
}