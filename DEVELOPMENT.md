The NetObserv Operator is meant to run in a Kubernetes cluster like OpenShift or [Kind](https://kind.sigs.k8s.io/). These are the two options most used by the development team.

[Architecture description](./docs/Architecture.md).

## Build / format / lint the code, run unit tests

```bash
make build test
```

## Build and deploy a Docker image

A way to test code changes is to build a Docker image from local sources, push it to your own Docker repository, and deploy it to an existing cluster. Do the following, but replace IMAGE value with your own registry and account:

```bash
IMAGE="quay.io/youraccount/network-observability-operator:test" make image-build image-push deploy
```

After the operator is deployed, set up Loki, which is used to store flows, install a `FlowCollector` custom resource to collect network flows, and optionally install Grafana to provide a user interface and dashboards.

This provides a quick and easy way to deploy Loki, Grafana and a `FlowCollector` with default values. Note this Loki setup is not for production use.

```bash
make deploy-loki deploy-grafana deploy-sample-cr
```

It will run Loki and Grafana locally, and set up a local port-forward to them. To avoid this, add `PORT_FWD=false` to the command above.

Creating a `FlowCollector` triggers the operator deploying the monitoring pipeline:

- Configures IPFIX exports
- Deploys the flow collector pods, `flowlogs-pipeline`
- Deploys the `netobserv-plugin` if using OpenShift Console

You should be able to see flows in OpenShift Console and Grafana. If not, wait up to 10 minutes. See the [FAQ on troubleshooting](./README.md#faq--troubleshooting) for more information.

### Test another one's pull request

To test a pull request opened by someone else, you just need to pull it locally. Using [GitHub CLI](https://cli.github.com/) is an easy way to do it. Then repeat the steps mentioned above to build, push an image, then deploy the operator and its custom resource.

## Deploy a specific image

Images are built and pushed through CI to [quay.io](https://quay.io/repository/netobserv/network-observability-operator?tab=tags).

You can refer to existing commits using their short-SHA as the image tag, or refer to existing releases. E.g:

```bash
# By commit SHA
VERSION="960766c" make deploy
# By release
VERSION="0.1.2" make deploy
```

It is recommended to switch to the corresponding release Git tag before deploying an old version to make sure the underlying components refer to the correct versions.

When `VERSION` is not provided, it defaults to the latest build on `main` branch.

You can also provide any custom `IMAGE` to `make deploy`.

## Before commiting, make sure bundle is correct

The github CI will fail if it finds the bundle isn't in a clean state. To update the bundle, simply run:

```bash
make update-bundle
```

This is necessary when the changes you did end up affecting the bundle manifests or metadata (e.g. adding new fields in the CRD, updating some documentation, etc.). When unsure, just run the command mentioned above.

## Installing Kafka

Kafka can be used to separate flow ingestion from flow transformation. The operator does not manage kafka deployment and topic creation. We provide a quick setup for Kafka using the [strimzi operator](https://strimzi.io/).

```bash
make deploy-kafka
```

Kafka can then be enabled in the `FlowCollector` resource by setting `spec.deploymentModel` to `KAFKA`. If you use your own Kafka setup, make sure to configure `spec.kafka.address` and `spec.kafka.topic` accordingly.

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

## Deploy as bundle

For more details, refer to the [Operator Lifecycle Manager (OLM) bundle quickstart documentation](https://sdk.operatorframework.io/docs/olm-integration/quickstart-bundle/).

This task should be automatically done by the CI/CD pipeline. However, if you want to deploy as
bundle for local testing, you should execute the following commands:

```bash
export USER=<container-registry-username>
export IMAGE=quay.io/$USER/network-observability-operator:test
export BUNDLE_IMAGE=quay.io/$USER/network-observability-operator-bundle:v0.0.0-test
make images
make bundle bundle-build bundle-push

# or, alternatively:
BUNDLE_VERSION=0.0.0-test VERSION=test make images bundle bundle-build bundle-push
```

Optionally, you might validate the bundle:

```bash
bin/operator-sdk bundle validate $BUNDLE_IMAGE
# or for podman
bin/operator-sdk bundle validate -b podman $BUNDLE_IMAGE
```

> Note: the base64 logo can be generated with: `base64 -w 0 <image file>`, then manually pasted in the [CSV manifest file](./config/csv/bases/netobserv-operator.clusterserviceversion.yaml) under `spec.icon`.

### Deploy as bundle from command line

This mode is recommended to quickly test the operator during its development:

```bash
bin/operator-sdk run bundle $BUNDLE_IMAGE
```

To cleanup:

```bash
bin/operator-sdk cleanup netobserv-operator
```

#### Testing an upgrade

First, deploy the previous version, e.g:

```bash
bin/operator-sdk  run bundle quay.io/netobserv/network-observability-operator-bundle:v1.0.3 --timeout 5m
```

Then, build your new bundle, e.g:

```bash
VERSION=test BUNDLE_VERSION=0.0.0-test make images bundle bundle-build bundle-push
```

Finally, run the upgrade:

```bash
bin/operator-sdk run bundle-upgrade quay.io/$USER/network-observability-operator-bundle:v0.0.0-test --timeout 5m
```

### Deploy as bundle from the Console's OperatorHub page

This mode is recommended when you want to test the customer experience of navigating through the
operators' catalog and installing/configuring it manually through the UI.

First, create and push a catalog image:

```bash
export CATALOG_IMAGE=quay.io/$USER/network-observability-operator-catalog:v$VERSION
make catalog-build catalog-push catalog-deploy
```

The NetObserv Operator is available in OperatorHub: https://operatorhub.io/operator/netobserv-operator

## Publish on central OperatorHub

See [RELEASE.md](./RELEASE.md#publishing-on-operatorhub).

## Using custom operand image

### With operator unmanaged deployment

This section is relevant when the operator was deployed directly as a Deployment, e.g. using `make deploy`. If it was deployed via OLM, refer to the next section.

In the `manager` container of the `netobserv-controller-manager` Deployment, set any of the
following the environment variables with your custom operand image with `kubectl set env` or
`oc set env`:

* `RELATED_IMAGE_EBPF_AGENT`
* `RELATED_IMAGE_FLOWLOGS_PIPELINE`
* `RELATED_IMAGE_CONSOLE_PLUGIN`

Examples:

```bash
oc -n netobserv set env deployment/netobserv-controller-manager -c "manager" RELATED_IMAGE_EBPF_AGENT="quay.io/netobserv/netobserv-ebpf-agent:main"
oc -n netobserv set env deployment/netobserv-controller-manager -c "manager" RELATED_IMAGE_FLOWLOGS_PIPELINE="quay.io/netobserv/flowlogs-pipeline:main"
oc -n netobserv set env deployment/netobserv-controller-manager -c "manager" RELATED_IMAGE_CONSOLE_PLUGIN="quay.io/netobserv/network-observability-console-plugin:main"
```

Alternatively you can use helper make targets for the same purpose:

```bash
USER=myself VERSION=test make set-agent-image set-flp-image set-plugin-image
```

### With operator managed via OLM

When the operator was deployed via OLM, hence is managed through its CSV, the "related images" are defined in the CSV. The same `make` targets can be used to modify them, with an additional `CSV` argument to target a CSV file. It is assumed to be deployed in namespace `openshift-netobserv-operator`.

E.g:

```bash
CSV=network-observability-operator.v1.2.0 USER=myself VERSION=test make set-agent-image set-flp-image set-plugin-image
```

You can also do this by editing the CSV via the console by changing the image defined under `RELATED_IMAGE_EBPF_AGENT`, `RELATED_IMAGE_FLOWLOGS_PIPELINE`, and/or `RELATED_IMAGE_CONSOLE_PLUGIN`. If you are using this method, ensure that you are in the `openshift-netobserv-operator` namespace before updating the image value. If you are in a different namespace, then it reverts it back.

![Alt text](./docs/assets/console-csv.png)

## Understanding the config / kustomize structure

The [config](./config/) directory contains assets required for creating the Operator bundle (which comes in two flavours: for OpenShift and for "vanilla" Kubernetes), as well as other assets used in `make` scripts that are helpful to set up development environments.

Let's see the `kustomize` dependency tree for OpenShift bundle, which entry point is `config/openshift-olm`:

```
openshift-olm
|
|===> ../csv
|     |
|     |===> ../samples
|     |     |
|     |     |===> FlowCollector samples
|     |
|     |===> CSV base file
|
|===> ./default
      |
      |===> Various patches and ServiceMonitor
      |
      |===> ../../crd
      |     |
      |     |===> CRD base file
      |     |
      |     |===> Various patches and configuration
      |
      |===> ../../rbac
      |     |
      |     |===> All RBAC-related resources
      |
      |===> ../../manager
      |     |
      |     |===> Operator deployment and various patches
      |
      |===> ../../webhook
            |
            |===> Webhook service and configuration
       
```

For "vanilla" Kubernetes, the dependency tree is very similar, but includes CertManager and doesn't include the ServiceMonitor. Its entry point is `config/k8s-olm`:

```
k8s-olm
|
|===> ../csv
|     |
|     |===> ../samples
|     |     |
|     |     |===> FlowCollector samples
|     |
|     |===> CSV base file
|
|===> ./default
      |
      |===> Various patches
      |
      |===> ../../crd
      |     |
      |     |===> CRD base file
      |     |
      |     |===> Various patches and configuration
      |
      |===> ../../rbac
      |     |
      |     |===> All RBAC-related resources
      |
      |===> ../../manager
      |     |
      |     |===> Operator deployment and various patches
      |
      |===> ../../webhook
      |     |
      |     |===> Webhook service and configuration
      |
      |===> ../../certmanager
            |
            |===> Configuration for CertManager
       
```

On top of that, there is also `config/openshift` which is used in developers environment to generate all the operator related assets without going through the bundle generation (e.g. there is no CSV), in order to be deployed directly on a running cluster. This is used in the `make deploy` script. Its content is very similar to `config/olm-openshift` apart from a few tweaks.

## View flowlogs-pipeline metrics in console

To view the generated flowlogs-pipeline metrics in the Openshift console, perform the following:

```
cd hack
./enable-metrics.sh
```

The metrics will be visible in the Openshift console under the tab `Observe -> Metrics.`
Look for the metrics that begin with `netobserv_.`

## Simulating a downstream deployment

To configure the operator to run as a downstream deployment run this command:

```
make set-release-kind-downstream
```

Most notably change will concern the monitoring part which will use the platoform monitoring stack instead of the user workload monitoring stack.

## Testing the github workflow

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
