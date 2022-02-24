#!/bin/bash

set -euo pipefail

KUBE_VERSION=v1.21.2
PATH=$PATH:/$HOME/.arkade/bin/

echo ">>> Installing arkade"
curl -sLS https://get.arkade.dev | sudo sh

echo ">>> Installing kubectl $KUBE_VERSION"
arkade get kubectl@$KUBE_VERSION
sudo mv $HOME/.arkade/bin/kubectl /usr/local/bin/

arkade get k3sup

sudo mv $HOME/.arkade/bin/k3sup /usr/local/bin/
