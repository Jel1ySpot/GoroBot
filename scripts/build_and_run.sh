#!/bin/bash

SCRIPT_DIR=$(dirname "$0")

bash "$SCRIPT_DIR"/build.sh

bash "$SCRIPT_DIR"/run.sh
