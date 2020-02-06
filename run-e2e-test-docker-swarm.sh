#!/bin/sh

if ! [ -x "$(command -v docker)" ]; then
  echo 'Unable to find docker command, please install Docker (https://www.docker.com/) and retry' >&2
  exit 1
fi

export IP=127.0.0.1
# setup docker swarm 
docker swarm init
git clone https://github.com/openfaas/faas.git
cd faas && echo "$(pwd)" && ./deploy_stack.sh --no-auth && cd ..

# run test in docker swarm
export OPENFAAS_URL=http://$IP:8080/
sleep 60
make -i test-swarm

# remove docker swarm, once tests complete
docker swarm leave --force