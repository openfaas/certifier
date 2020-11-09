#!/bin/bash

set -euo pipefail

KUBE_VERSION=v1.18.8

echo ">>> Installing k3sup"
sudo curl -sLS https://get.k3sup.dev | sh
sudo install k3sup /usr/local/bin/

echo ">>> Installing arkade"
curl -sLS https://dl.get-arkade.dev | sudo sh

echo ">>> Installing kubectl $KUBE_VERSION"
arkade get kubectl --version $KUBE_VERSION
sudo mv $HOME/.arkade/bin/kubectl /usr/local/bin/