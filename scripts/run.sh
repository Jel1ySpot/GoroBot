#!/bin/bash

CURRENT=$(pwd)

cd "$(dirname "$0")"/../build || exit

./bot

cd "$CURRENT" || exit
