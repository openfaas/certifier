#!/bin/bash

set -euo pipefail

docker swarm init
git clone https://github.com/openfaas/faas.git
cd faas && echo "$(pwd)" && ./deploy_stack.sh --no-auth && cd ..
sleep 15
make test-swarm
docker swarm leave --force