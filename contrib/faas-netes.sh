#!/bin/bash

set -euo pipefail

curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/$KUBECTL_VERSION/bin/linux/amd64/kubectl &&
          chmod +x kubectl && sudo mv kubectl /usr/local/bin/

sudo curl -sLS https://get.k3sup.dev | sh
sudo install k3sup /usr/local/bin/
mkdir -p $KUBECONFIG_FOLDER_PATH
k3sup install --local --ip $IP --user $USER_NAME
cp `pwd`/kubeconfig $KUBECONFIG_PATH
curl -sLS https://dl.get-arkade.dev | sudo sh
arkade install openfaas --basic-auth=false
kubectl rollout status -n openfaas deploy/gateway -w

make test-kubernetes