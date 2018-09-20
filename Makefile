OPENFAAS_URL?=http://127.0.0.1:8080/
SECRET?=tDsdf7sFT45gs8D3gDGhg54

clean-swarm:
	-  docker service rm stronghash env-test env-test-labels env-test-annotations env-test-verbs test-secret ; docker secret rm secret-api-test-key 

clean-kubernetes:
	- kubectl delete -n openfaas-fn deploy/stronghash || : ; kubectl delete -n openfaas-fn svc/stronghash || :
	- kubectl delete -n openfaas-fn deploy/env-test || :; kubectl delete -n openfaas-fn svc/env-test || :
	- kubectl delete -n openfaas-fn deploy/env-test-annotations || : ; kubectl delete -n openfaas-fn svc/env-test-annotations || :
	- kubectl delete -n openfaas-fn deploy/env-test-verbs  || :; kubectl delete -n openfaas-fn svc/env-test-verbs || :
	- kubectl delete -n openfaas-fn deploy/test-secret  || :; kubectl delete -n openfaas-fn svc/test-secret || :

.EXPORT_ALL_VARIABLES:
secrets-swarm:
	./create-swarm-secret.sh

secrets-kubernetes:
	./create-kubernetes-secret.sh

test-swarm: clean-swarm secrets-swarm 
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v

test-kubernetes: secrets-kubernetes clean-kubernetes
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v