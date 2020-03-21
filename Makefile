OPENFAAS_URL?=http://127.0.0.1:8080/

GOFLAGS := -mod=vendor

.TEST_FUNCTIONS = stronghash env-test env-test-annotations env-test-labels env-test-verbs test-secret-crud test-min-scale test-scale-from-zero test-throughput-scaling test-scaling-disabled test-scaling-to-zero test-logger

clean-swarm:
	- docker service rm ${.TEST_FUNCTIONS}

clean-kubernetes:
	- kubectl delete -n openfaas-fn deploy,svc ${.TEST_FUNCTIONS} 2>/dev/null || : ;

lint:
	@golangci-lint version
	golangci-lint run --timeout=1m ./...

.TEST_FLAGS= # additional test flags, e.g. -run ^Test_ScaleFromZeroDuringIvoke$

test-swarm: clean-swarm
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v -swarm ${.TEST_FLAGS}

test-kubernetes: clean-kubernetes
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v ${.TEST_FLAGS}
