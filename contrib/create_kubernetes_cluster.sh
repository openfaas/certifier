#!/bin/bash

set -euo pipefail

USER_NAME=$(whoami)

echo ">>> Creating kubernetes cluster"
k3sup install --local --ip $IP --user $USER_NAME