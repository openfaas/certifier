#/bin/bash

curl -SLf https://golang.org/dl/go1.16.linux-amd64.tar.gz > /tmp/go.tgz
sudo rm -rf /usr/local/go/
sudo mkdir -p /usr/local/go/
sudo tar -xvf /tmp/go.tgz -C /usr/local/go/ --strip-components=1

export GOPATH=$HOME/go/
export PATH=$PATH:/usr/local/go/bin/

go version
