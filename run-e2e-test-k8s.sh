#!/bin/sh

if ! [ -x "$(command -v docker)" ]; then
  echo 'Unable to find docker command, please install Docker (https://www.docker.com/) and retry' >&2
  exit 1
fi

export IP=127.0.0.1
export USER_NAME=$(whoami)
export KUBECONFIG_FOLDER_PATH=/home/$USER_NAME/.kube
export KUBECONFIG_PATH=$KUBECONFIG_FOLDER_PATH/config

# setup k3s using k3sup
sudo curl -sLS https://get.k3sup.dev | sh
sudo install k3sup /usr/local/bin/
mkdir -p $KUBECONFIG_FOLDER_PATH
k3sup install --local --ip $IP --user $USER_NAME
cp `pwd`/kubeconfig $KUBECONFIG_PATH
export KUBECONFIG=$KUBECONFIG_PATH

# disable basic auth
k3sup app install openfaas --basic-auth=false

# run test in k3s
sleep 60
export OPENFAAS_URL=http://$IP:31112/
make -i test-kubernetes