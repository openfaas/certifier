## contrib

Various scripts are run from the Makefile and .github/workflows/test.yaml in a GitHub Action

## Testing K3s locally with Multipass

If running locally:

```bash
multipass launch -m 4G -c 2 --name k3s
multipass exec k3s /bin/bash
```

Then:

```
git clone https://github.com/openfaas/certifier
cd certifer/contrib

./install-essential.sh && \
  ./install-go.sh && \
  ./get_tools.sh && \
  ./create_kubernetes_cluster.sh && \
  ./deploy_openfaas.sh
```

Finally:

```
cd ..

export GOPATH=$HOME/go/
export PATH=$PATH:/usr/local/go/bin/

export OPENFAAS_URL=http://127.0.0.1:31112

make test-kubernetes
```

## Testing faasd locally with multipass

If running locally:

```bash
multipass launch -m 2G -c 2 --name faasd
multipass exec faasd /bin/bash
```

Then:

```
git clone https://github.com/openfaas/certifier
cd certifer/contrib
```

```sh
./install-essential.sh && \
  ./install-go.sh
```

Then run the tests:

```bash
cd ..

export GOPATH=$HOME/go/
export PATH=$PATH:/usr/local/go/bin/

export OPENFAAS_URL=http://127.0.0.1:8080

make test-faasd
```

