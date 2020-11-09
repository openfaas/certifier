#!/usr/bin/env bash

echo ">>> Cloning the faas project"
git clone https://github.com/openfaas/faas.git

echo ">>> Deploying the stack"
cd faas && echo "$(pwd)" && ./deploy_stack.sh --no-auth && cd ..