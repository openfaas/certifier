#!/bin/sh
export NAME=certifier
export IMAGE=ghcr.io/openfaas/certifier
export eTAG="latest"
echo "$1"
if [ "$1" ] ; then
  eTAG=$1
fi

docker create --name "$NAME" "${IMAGE}:${eTAG}" \
    && mkdir -p ./bin \
    && docker cp "$NAME":/bin .
docker rm -f "$NAME"
