#!/bin/bash

echo ">>> Deploying faasd"
sudo curl -s https://raw.githubusercontent.com/openfaas/faasd/master/hack/install.sh | sudo sh

sleep 120

if [ "z$CERTIFIER_NAMESPACES" != "z" ]; then
    sudo ctr namespace create certifier-test
    sudo ctr namespace label certifier-test openfaas=true
fi

echo ">>> Login Using faas-cli"
cd $HOME
sudo cat /var/lib/faasd/secrets/basic-auth-password | /usr/local/bin/faas-cli login --password-stdin
