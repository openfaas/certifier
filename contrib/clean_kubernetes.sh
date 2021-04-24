#!/bin/bash



namespaces="openfaas-fn,$CERTIFIER_NAMESPACES"
for ns in ${namespaces//,/ }
do
    echo "cleaning $ns"
    for f in $TEST_FUNCTIONS
    do
        echo "delete function $f"
        kubectl delete -n "$ns" deploy,svc "$f"
    done

    for s in $TEST_SECRETS
    do
        echo "deleting secret $s"
        kubectl delete -n "$ns" secrets "$s"
    done
done