swarm:
	docker service rm stronghash env-test
test:
	gateway_url=http://localhost:8080/ time go test ./tests -v
