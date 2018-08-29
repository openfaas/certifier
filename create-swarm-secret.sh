#!/usr/bin/env bash

if [[ $(docker secret ls | grep secret-api-test-key | wc -l) -eq 0 ]]
then
    echo $SECRET | docker secret create secret-api-test-key -
    echo "Swarm secret created"
else
    echo "Swarm secret already exists. Removing old secret secret-api-test-key"
    docker secret rm secret-api-test-key
    echo $SECRET | docker secret create secret-api-test-key -
    echo "Swarm secret created"
fi 