#!/bin/bash

namespaces="openfaas-fn,$CERTIFIER_NAMESPACES"
for ns in ${namespaces//,/ }
do
    echo "cleaning $ns"
    for f in $TEST_FUNCTIONS
    do
        echo "delete function $f"
        faas-cli remove "$f" -n "$ns"
    done

    for s in $TEST_SECRETS
    do
        echo "deleting secret $s"
        faas-cli secret remove "$f" -n "$ns"
    done
done

echo ">>> Openfaas Config Path"
echo $OPENFAAS_CONFIG
