# certify-incubator

The purpose of this project is to certify that an OpenFaaS provider is doing what it should in response to the RESTful API.

Usage with gateway on localhost:8080:

```
make test-swarm
```

Usage with local Kubernetes cluster:

```
make test-kubernetes OPENFAAS_URL=http://localhost:31112/
```

This is a work-in-progress and covers a couple of basic scenarios.

Style guidelines
- [ ] Initial versions use idiomatic Go for tests (no asserts or Gherkin)
- [ ] Duplication is better than premature abstraction / complexity
- [ ] Tests need to cope with timeouts and attempt retries when that makes sense
- [ ] should pass `gofmt`
- [ ] commits should follow contribution guide of [openfaas/faas](https://github.com/openfaas/faas)

