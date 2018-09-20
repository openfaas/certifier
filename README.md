# certify-incubator

The purpose of this project is to certify that an OpenFaaS provider is doing what it should in response to the RESTful API.

## Usage

The tests assume a local environment with basic authentication turned off.

### Swarm

Usage with gateway on `http://127.0.0.1:8080/`:

```
export OPENFAAS_URL=http://127.0.0.1:8080/
make test-swarm
```

You will need to have access to `docker` for creating and cleaning-up of state.

### Kubernetes

Usage with local Kubernetes cluster:

```
export OPENFAAS_URL=http://127.0.0.1:31112/
make test-kubernetes
```

You will need to have access to `kubectl` for creating and cleaning state.

## Status

This is a work-in-progress and attempts to cover the basic scenarios of operating an OpenFaaS provider.

Style guidelines
- [ ] Initial versions use idiomatic Go for tests (no asserts or Gherkin)
- [ ] Duplication is better than premature abstraction / complexity
- [ ] Tests need to cope with timeouts and attempt retries when that makes sense
- [ ] should pass `gofmt`
- [ ] commits should follow contribution guide of [openfaas/faas](https://github.com/openfaas/faas)

