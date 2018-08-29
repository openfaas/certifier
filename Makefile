OPENFAAS_URL?=http://127.0.0.1:8080/

clean-swarm:
	- docker service rm stronghash env-test env-test-labels env-test-annotations test-secret

clean-kubernetes:
	kubectl delete -n openfaas-fn deploy/stronghash ; kubectl delete -n openfaas-fn deploy/env-test
	kubectl delete -n openfaas-fn env-test-labels ; kubectl delete -n openfaas-fn env-test-annotations
	kubectl delete -n openfaas-fn deploy/test-secret

SECRET?=tDsdf7sFT45gs8D3gDGhg54

.EXPORT_ALL_VARIABLES:
secrets-swarm:
	./create-swarm-secret.sh

secrets-kubernetes:
	./create-kubernetes-secret.sh

test-swarm: secrets-swarm clean-swarm
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v

test-kubernetes: secrets-kubernetes clean-kubernetes
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v