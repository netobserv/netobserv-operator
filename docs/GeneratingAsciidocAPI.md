# Generating AsciiDoc API reference

## One-time setup docsgen repo

1. Clone https://github.com/jboxman-rh/openshift-apidocs-gen
2. run `npm install && npm install -g`

## Run it

```bash
# If you haven't already, start any k8s cluster
kind create cluster

kubectl apply -f bundles/openshift/manifests/flows.netobserv.io_flowcollectors.yaml
kubectl apply -f bundles/openshift/manifests/flows.netobserv.io_flowcollectorslices.yaml
kubectl apply -f bundles/openshift/manifests/flows.netobserv.io_flowmetrics.yaml

hack/asciidoc-gen.sh
```

Once you're done:

```bash
kind delete cluster
```

# Generate AsciiDoc for flows JSON format reference

## Run the script

```bash
hack/asciidoc-flows-gen.sh 
```
