OPENFAAS_URL?=http://127.0.0.1:8080/

GOFLAGS := -mod=vendor

TEST_FUNCTIONS = \
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
	redirector-test \
	secret-string \
	secret-bytes

TEST_SECRETS = \
	secret-string \
	secret-bytes

export TEST_FUNCTIONS TEST_SECRETS


clean-kubernetes:
	- ./contrib/clean_kubernetes.sh

clean-faasd:
	- ./contrib/clean_faasd.sh

.TEST_FLAGS= # additional test flags, e.g. -run ^Test_ScaleFromZeroDuringIvoke$
.FEATURE_FLAGS= # set config feature flags, e.g. -swarm

test-kubernetes: clean-kubernetes
	CERTIFIER_NAMESPACES=certifier-test time go test -p=1 -count=1 ./tests -v -gateway=${OPENFAAS_URL} ${.FEATURE_FLAGS} ${.TEST_FLAGS}

test-faasd: clean-faasd
	time go test -p=1 -count=1 ./tests -v -gateway=${OPENFAAS_URL} -enableAuth ${.FEATURE_FLAGS} ${.TEST_FLAGS}
