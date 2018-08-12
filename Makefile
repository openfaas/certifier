OPENFAAS_URL?=http://127.0.0.1:8080/

swarm:
	docker service rm stronghash env-test env-test-labels env-test-annotations
kubernetes:
	kubectl delete -n openfaas-fn deploy/stronghash ; kubectl delete -n openfaas-fn deploy/env-test
	kubectl delete -n openfaas-fn env-test-labels ; kubectl delete -n openfaas-fn env-test-annotations
test:
	gateway_url=${OPENFAAS_URL} time go test -count=1 ./tests -v