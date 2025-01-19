#!/bin/bash

CURRENT=$(pwd)

cd "$(dirname "$0")"/.. || exit

mkdir -p build
go build -gcflags="-N -l" -o build/bot

cd build || exit

gdb ./bot

cd "$CURRENT" || exit
