OPENFAAS_URL?=http://127.0.0.1:8080/

GOFLAGS := -mod=vendor

.TEST_FUNCTIONS = \
	stronghash \
	env-test \
	env-test-annotations \
	env-test-labels \
	env-test-verbs \
	test-secret-crud \
	test-min-scale \
	test-scale-from-zero \
	test-throughput-scaling \
	test-scaling-disabled \
	test-scaling-to-zero \
	test-logger \
	redirector-test

clean-swarm:
	- docker service rm ${.TEST_FUNCTIONS}

clean-kubernetes:
	- kubectl delete -n openfaas-fn deploy,svc ${.TEST_FUNCTIONS} 2>/dev/null || : ;

.TEST_FLAGS= # additional test flags, e.g. -run ^Test_ScaleFromZeroDuringIvoke$
.FEATURE_FLAGS= # set config feature flags, e.g. -swarm

test-swarm: clean-swarm
	time go test -count=1 ./tests -v -swarm -gateway=${OPENFAAS_URL} ${.FEATURE_FLAGS} ${.TEST_FLAGS}

test-kubernetes: clean-kubernetes
	time go test -count=1 ./tests -v -gateway=${OPENFAAS_URL} ${.FEATURE_FLAGS} ${.TEST_FLAGS}
