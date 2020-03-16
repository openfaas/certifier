OPENFAAS_URL?=http://127.0.0.1:8080/
SECRET?=tDsdf7sFT45gs8D3gDGhg54

.TEST_FUNCTIONS = stronghash env-test env-test-annotations env-test-labels env-test-verbs test-secret test-secret-crud test-min-scale test-scale-from-zero test-throughput-scaling test-scaling-disabled test-scaling-to-zero

clean-swarm:
	- docker service rm ${.TEST_FUNCTIONS}; docker secret rm secret-api-test-key || : ;

clean-kubernetes:
	- kubectl delete -n openfaas-fn deploy,svc ${.TEST_FUNCTIONS} 2>/dev/null || : ;


.EXPORT_ALL_VARIABLES:
secrets-swarm:
	./create-swarm-secret.sh

secrets-kubernetes:
	./create-kubernetes-secret.sh

.TEST_FLAGS= # additional test flags, e.g. -run ^Test_ScaleFromZeroDuringIvoke$

test-swarm: clean-swarm secrets-swarm
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v -swarm ${.TEST_FLAGS}

test-kubernetes: secrets-kubernetes clean-kubernetes
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v ${.TEST_FLAGS}
