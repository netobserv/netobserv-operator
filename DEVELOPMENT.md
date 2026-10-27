# Development

The NetObserv Operator is meant to run in a Kubernetes cluster. Local development can be tested on [Kind](https://kind.sigs.k8s.io/). No specific configuration is needed, you may start kind with `kind create cluster`.

> For `podman` users: rootless mode is not possible, since the eBPF agent requires elevated permissions to observe the traffic. It is necessary that the Kubernetes cluster has root permissions on your machine. You may still run `kind` as root with podman, and set the resulting kube config file accessible to `kubectl`. This is not a recommended way of running `kind` though.

## Architecture

See [Architecture doc](./docs/Architecture.md).

## Build / format / lint the code, run unit tests

To build, reformat, run linter and unit-tests after editing the code:

```bash
make build test
```

You also need to make sure the bundle is up to date, in case your changes affected the generated resources. The github CI will fail if it finds the bundle isn't in a clean state. To update the bundle, simply run:

```bash
make update-bundle
```

This is necessary when the changes you did end up affecting the bundle manifests or metadata (e.g. adding new fields in the CRD, updating some documentation, etc.). When unsure, just run the command mentioned above.

If you changed `go.mod`, make sure to generate clean vendors with:

```bash
make vendors
```

## Build image and deploy using Helm

A way to test code changes is to build a container image from local sources and push it to a Docker / OCI repository that you own. Run the following command, replacing IMAGE with one matching your container registry and account:

```bash
IMAGE="quay.io/youraccount/network-observability-operator:test" make images
```

Deploying can be done with Helm. Make sure you have the [helm CLI installed](https://helm.sh/docs/intro/install/) on your machine. Make also sure the Helm chart is up to date after your local changes, by running:

```bash
make update-bundle
```

Then, install the operator and its pre-requisites (it will add cert-manager / https://charts.jetstack.io to your helm known repos) by running:

```bash
IMAGE="quay.io/youraccount/network-observability-operator:test" make helm-install
```

At this point, the operator should be up and running, but no `FlowCollector` is configured yet, meaning that none of the related components are deployed. To start collecting flows with the default configuration for local testing, run:

```bash
make helm-configure-flowcollector
```

This is going to start pods from the related components, so that flow collection begins. The configuration is opinionated for small cluster (such as Kind) with minimal features enabled, but you can edit the `FlowCollector` resource as you want.

To access the Web Console, run in another terminal session (port-forwarding):

```bash
make helm-expose-console
```

Then open http://localhost:9001/.

## Cleaning up

To remove NetObserv:

```bash
make helm-cleanup
```

To remove NetObserv and its dependencies (cert-manager):

```bash
make helm-cleanup-all
```

## Testing related components

If you want to test changes from other related repositories, such as the Web Console, the eBPF Agent or Flowlogs-pipeline, you need to build an image of that component (refer to the corresponding documentation) and configure the operator to use that image.

If you haven't installed the operator yet, you can change the [Helm values](./helm/values.yaml) with the desired images. For example:

```yaml
ebpfAgent:
  image: quay.io/youraccount/netobserv-ebpf-agent
  version: test
```

Or if the operator is already installed, you can use dedicated `make` targets to update the desired components (under the cover, it will edit the operator `Deployment` to change the component related image):

```bash
# For the eBPF agent, tagged 'test':
VERSION=test make set-agent-image
# For flowlogs-pipeline, tagged 'test':
VERSION=test make set-flp-image
# For the web console, tagged 'test':
VERSION=test make set-plugin-image
```

Note that these targets default to using `quay.io` as the container registry, and your current user name as the registry account name. You can override that with `$IMAGE_REGISTRY` and `$USER` respectively.

## Installing Kafka

Kafka can be used as a intermediate layer between the eBPF agents and flowlogs-pipeline. The operator does not manage kafka deployment and topic creation. We provide a quick setup for Kafka using the [strimzi operator](https://strimzi.io/).

```bash
make deploy-kafka
# or: (and that's actually mTLS)
make deploy-kafka-tls
```

Kafka can then be enabled in the `FlowCollector` resource by setting `spec.deploymentModel` to `Kafka`. If you use your own Kafka setup, make sure to configure `spec.kafka.address` and `spec.kafka.topic` accordingly.

## Linking with API changes in flowlogs-pipeline

To link with merged changes (but unreleased), update the FLP version by running (replacing "LONG_COMMIT_SHA"):

```bash
go get github.com/netobserv/flowlogs-pipeline@LONG_COMMIT_SHA
```

To link with unmerged changes, add this at the bottom of `go.mod`:

```
replace github.com/netobserv/flowlogs-pipeline => ../flowlogs-pipeline
```

Then run:

```bash
make vendors
```

## Understanding the config / kustomize structure

The [config](./config/) directory contains assets required for creating the Operator bundle, which comes in two flavours: for vanilla Kubernetes and for vendors (OpenShift). Their entry points are `config/k8s/olm` and `config/openshift/olm` respectively. There is a third possibility that is not used for OLM bundling, but is used for development purpose to deploy directly on cluster with `make ocp-deploy`: it's `config/openshift` (a non-OLM variant). Subdirectories in `config/` that are not in `k8s/` or `openshift/`, such as `crd/` and `csv/`, are typically common for both bundles.

The vanilla bundle is not distributed as is, but is used to derive helm charts.

Running `make update-bundle` updates the two OLM bundles and the derived helm chart.

## Simulating an OpenShift downstream deployment

To configure the operator to run as a downstream deployment run this command:

```bash
VENDOR=OpenShift_Downstream make set-vendor
```

Most notably change will concern the monitoring part which will use the platform monitoring stack instead of the user workload monitoring stack.

## Profiling

You can use `pprof` for profiling. Run `pprof` make target to start listening and port-forward on 6060: 

```bash
make pprof
```

In another terminal, run for instance:

```bash
curl "http://localhost:6060/debug/pprof/heap?gc" -o /tmp/heap
go tool pprof -http localhost:3435 /tmp/heap
```

## Testing the github workflow

> This section is for maintainers, with permission to write on the `workflow-test` branch.

You should test the workflows when you modify files in `.github/workflows` or the `Makefile` targets used in these workflows. Be aware that the `Makefile` is used not only by developers or QEs on their local machines, but also in this github workflows files.

Testing github workflows can sometimes be tricky as it's not always possible to run everything locally, and they depend on triggers such as merging a commit, or pushing a tag on the upstream. Here's a guide about how to test that:

### test-workflow.sh

Run the `hack/test-workflow.sh` script. It is not a silver bullet, but it will test a bunch of things in the workflows, such as expecting some images to be built, and correctly referenced in the CSV. Be aware that it has some biases and doesn't cover everything, like it won't push anything to the image registry, so it's still necessary to run through the next items.

### push_image.yml workflow

This workflow is triggered when something is merged into `main`, to push new images to Quay. For testing, it is also configured to be triggered when something is merged on the `workflow-test` branch. So, push your changes to that branch and monitor the triggered actions (assuming `upstream` refers to this remote GIT repo).

```bash
# You might need to force-push since this test branch may contain past garbage...
git push upstream HEAD:workflow-test -f
```

Then, open the [action page](https://github.com/netobserv/netobserv-operator/actions/workflows/push_image.yml) in Github to monitor the jobs triggered. Make sure on Quay that you get the expected images for the [Operator](https://quay.io/repository/netobserv/network-observability-operator?tab=tags), the [bundle](https://quay.io/repository/netobserv/network-observability-operator-bundle?tab=tags) and the [catalog](https://quay.io/repository/netobserv/network-observability-operator-catalog?tab=tags).

Expected images:
- Operator's tagged "workflow-test" manifest + every support archs
- Operator's tagged with SHA manifest + every support archs (make sure they expire)
- Bundle and Catalog v0.0.0-workflow-test
- Bundle and Catalog v0.0.0-SHA (make sure they expire)

### push_image_pr.yml

This workflow is triggered by the "ok-to-test" label on a PR, however the workflow that is run is the one from the base branch, so you cannot test it from a PR opened against `main`. You need to open a new PR against `workflow-test` (assuming you pushed directly on that branch already, as described in the previous step):

```bash
touch dummy && git add dummy && git commit -m "dummy"
git push origin HEAD:dummy
```

Then, open a PR in github, making sure to select `workflow-test` as the base branch and not `main`.
On the PR, add the `ok-to-test` label.

This will trigger the corresponding `push_image_pr.yml` workflow ([view on github](https://github.com/netobserv/netobserv-operator/actions/workflows/push_image_pr.yml)). As above, you should check that the images are well created in Quay:

Expected images:
- Operator's tagged with SHA manifest + single arch amd64 (make sure they expire)
- Bundle and Catalog v0.0.0-SHA (make sure they expire)

### release.yml

Finally there's the upstream release process. Just create a fake version tag such as `0.0.0-rc0` and push it:

```bash
git tag -a "0.0.0-rc0" -m "0.0.0-rc0"
git push upstream --tags
```

When the tag is pushed, it will trigger the corresponding workflow ([view on github](https://github.com/netobserv/netobserv-operator/actions/workflows/release.yml)). As above, you should check that the images are well created in Quay. It's fine if you tag from the `workflow-test` branch (or any branch).

Expected images:
- Operator's tagged 0.0.0-rc0 manifest + every support archs
- Bundle and Catalog v0.0.0-rc0

Remove the tag after you tested:
 
```bash
git tag -d "0.0.0-rc0"
git push --delete upstream 0.0.0-rc0
```
