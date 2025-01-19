#!/bin/bash

CURRENT=$(pwd)

cd "$(dirname "$0")"/.. || exit
mkdir -p build
go build -o build/bot
cd "$CURRENT" || exit
