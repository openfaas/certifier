#!/bin/bash

set -euo pipefail

USER_NAME=$(whoami)
KUBECONFIG_FOLDER_PATH=/home/$USER_NAME/.kube
KUBECONFIG_PATH=$KUBECONFIG_FOLDER_PATH/config
KUBECONFIG=$KUBECONFIG_PATH

echo ">>> Setup kubectl configuration"
mkdir -p $KUBECONFIG_FOLDER_PATH
cp `pwd`/kubeconfig $KUBECONFIG_PATH

echo ">>> Installing openfaas"
arkade install openfaas --basic-auth=false --clusterrole

kubectl create namespace certifier-test
kubectl annotate namespace/certifier-test openfaas="1"

echo ">>> Waiting for helm install to complete."
kubectl rollout status -n openfaas deploy/gateway -w
