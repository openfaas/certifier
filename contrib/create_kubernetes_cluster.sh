#!/bin/bash

set -euo pipefail

USER_NAME=$(whoami)
export CHANNEL=v1.21

echo ">>> Creating K3s cluster with $CHANNEL"
k3sup install \
    --local \
    --k3s-channel $CHANNEL \
    --user $USER_NAME \
    --k3s-extra-args "--disable traefik"
