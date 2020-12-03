# certifier for OpenFaaS

[![Build Status](https://travis-ci.com/openfaas/certifier.svg?branch=master)](https://travis-ci.com/openfaas/certifier)
[![OpenFaaS](https://img.shields.io/badge/openfaas-serverless-blue.svg)](https://www.openfaas.com)

The purpose of this project is to certify that an OpenFaaS provider is doing what it should in response to the RESTful API.

## Usage

The tests assume a local environment with basic authentication turned off.

### Auth
The test _can_ use auth by setting an explicit Bearer token using the `-token` flag or by  reading the CLI config when you set the `-enableAuth` flag.

```sh
echo -n $PASSWORD | faas-cli login --gateway=$OPENFAAS_URL --username admin --password-stdin
go test - v ./tests -enableAuth -gateway=$OPENFAAS_URL
```

### Kubernetes

Usage with local Kubernetes cluster:

```sh
export OPENFAAS_URL=http://127.0.0.1:31112/
make test-kubernetes
```

You will need to have access to `kubectl` for creating and cleaning state.

If you have enabled auth in your cluster, first login with the `faas-cli` and then use

```sh
export OPENFAAS_URL=http://127.0.0.1:31112/
make test-kubernetes .FEATURE_FLAGS='-enableAuth'
```

### Swarm

Usage with gateway on `http://127.0.0.1:8080/`:

```
export OPENFAAS_URL=http://127.0.0.1:8080/
make test-swarm
```

You will need to have access to `docker` for creating and cleaning-up of state.

## Development

While developing the `certifier`, we generally run/test the `certifier` locally using `faas-netes`.  The cleanest way to do this is using an throw-away cluster using [KinD](https://github.com/kubernetes-sigs/kind) and [arkade](https://github.com/alexellis/arkade)

```sh
kind create cluster
arkade install openfaas --basic-auth=false
kubectl rollout status -n openfaas deploy/gateway
kubectl port-forward -n openfaas svc/gateway 8080:8080  > /dev/null 2>&1 &

export OPENFAAS_URL=http://127.0.0.1:8080/
make test-kubernetes
```

When you are done, you can stop the port-forward and clean up the cluster using

```sh
pkill kubectl
kind delete cluster
```

### Running individual tests

The test suite uses the Go test framework, so we can run individual tests by passing the [`-run` flag](https://golang.org/pkg/testing/#hdr-Subtests_and_Sub_benchmarks).

For example,

```sh
go test -run '^Test_SecretCRUD'
```

This is exposed in the `Makefile`,

```sh
make test-kubernetes .TEST_FLAGS='-run ^Test_SecretCRUD'
```

### Test and Feature flags
Some providers may not implement all features (yet) or an installation may have disabled a feature (e.g. scale to zero using the faas-idler)

```sh
  -enableAuth
    	enable/disable authentication. The auth will be parsed from the default config in ~/.openfaas/config.yml
  -gateway string
    	set the gateway URL, if empty use the gateway_url env variable
  -scaleToZero
    	enable/disable scale from zero tests (default true)
  -secretUpdate
    	enable/disable secret update tests (default true)
  -swarm
    	helper flag to run only swarm-compatible tests only
  -token string
    	authentication Bearer token override, enables auth automatically
```

These flags can be passed the the `Makefile` via the `.FEATURE_FLAGS` variable:

```sh
make test-kubernetes .FEATURE_FLAGS='-scaleToZero=false'
```

## Status

This is a work-in-progress and attempts to cover the basic scenarios of operating an OpenFaaS provider.

Style guidelines
- [ ] Initial versions use idiomatic Go for tests (no asserts or Gherkin)
- [ ] Duplication is better than premature abstraction / complexity
- [ ] Tests need to cope with timeouts and attempt retries when that makes sense
- [ ] should pass `gofmt`
- [ ] commits should follow contribution guide of [openfaas/faas](https://github.com/openfaas/faas)
