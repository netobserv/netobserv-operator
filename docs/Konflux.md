# NetObserv Operator Konflux builds

> [!WARNING]
> This documentation is about the downstream CI/CD used to build and release Network Observability product in Openshift.
> Some of the links here only work with Red Hat vpn.

## Links

Useful links

- [NetObserv konflux console](https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/ns/ocp-network-observab-tenant/applications)
- [Konflux documentation](https://konflux.pages.redhat.com/docs/users/)
- [Konflux build task reference](https://github.com/konflux-ci/build-definitions/tree/main/task) contains the definitions of tasks that can be used in konflux pipeline
- [Konflux release configuration](https://gitlab.cee.redhat.com/releng/konflux-release-data) contains release configuration
- [Konflux sample project](https://github.com/konflux-ci/olm-operator-konflux-sample) is a sample project built as an example on how to use konflux
- [Lightspeed operator](https://github.com/openshift/lightspeed-operator/blob/main/.tekton/fbc-v4-15-pull-request.yaml) probably one of the first project to release with konflux

## Builds

Builds are defined in the .tekton/ directory of each project, to ensure pre-merge build and post merge buids are identical and make maintenance easier, pipeline references has been centralized in a dediceted pipeline-ref file.

### Nudging

Nudging is the konflux mecanism to create dependencies and update between components. When a component A is nudging a component B, Konflux will automatically create PR to update component A reference in component B.

FLP, the ebpf-agent, the console plugin (including PF4 and PF5 compat variants) and the operator component are nudging the bundle build once finished and the bundle is nudging the FBC build once finished.

### File Based Catalog

When using Konflux to release an operator, it is required to use a File Based Catalog image. This imply some changes:
- FBC will not be additive to another one, the released FBC must contain all previous versions
- one FBC version must be built per Openshift version, the base image used will define the targeted Openshift version
- FBC build must be the only component in an application

### Konflux pull requests

Konflux will regulary create new pull requests, there are three categories :

- [Nudging pull request](https://github.com/netobserv/netobserv-operator/pull/969) To upgrade the reference to another component
- [Konflux tasks update](https://github.com/netobserv/netobserv-operator/pull/787) Up to date tasks are required to pass security check during release. Also the migration note sometimes contains instruction to some required actions
- [Dependencies update](https://github.com/netobserv/netobserv-operator/pull/962) Kondlux internally use [https://github.com/renovatebot/renovate](renovate) to automatically create this PR.

## Deploying

If you keep the PF5 mirror entries in the manifest below, the corresponding Konflux `Component` names must exist; otherwise remove that mirror block or onboarding will not match reality (see [Konflux PF4 and PF5 console plugin prerequisite](#konflux-pf4-and-pf5-console-plugin-prerequisite)).

An `ImageDigestMirrorSet` is required:

```yaml
apiVersion: config.openshift.io/v1
kind: ImageDigestMirrorSet
metadata:
  name: netobserv-image-digest-mirror-set
spec:
  imageDigestMirrors:
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-zstream
      source: registry.redhat.io/network-observability/network-observability-rhel9-operator
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/flowlogs-pipeline-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/flowlogs-pipeline-zstream
      source: registry.redhat.io/network-observability/network-observability-flowlogs-pipeline-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/netobserv-ebpf-agent-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/netobserv-ebpf-agent-zstream
      source: registry.redhat.io/network-observability/network-observability-ebpf-agent-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-zstream
      source: registry.redhat.io/network-observability/network-observability-console-plugin-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-pf4-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-pf4-zstream
      source: registry.redhat.io/network-observability/network-observability-console-plugin-compat-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-pf5-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-console-plugin-pf5-zstream
      source: registry.redhat.io/network-observability/network-observability-console-plugin-pf5-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-cli-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-cli-zstream
      source: registry.redhat.io/network-observability/network-observability-cli-rhel9
    - mirrors:
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-ystream
      - quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-zstream
      source: registry.redhat.io/network-observability/network-observability-operator-bundle
```

The testing FBC image can be added as a CatalogSource:

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: netobserv-konflux
  namespace: openshift-marketplace
spec:
  displayName: netobserv-konflux
  image: 'quay.io/redhat-user-workloads/ocp-network-observab-tenant/catalog-ystream:latest'
  # for z-stream, use instead:
  # image: 'quay.io/redhat-user-workloads/ocp-network-observab-tenant/catalog-zstream:latest'
  publisher: Netobserv team
  sourceType: grpc
```

## Release pipeline setup

[Konflux release configuration](https://gitlab.cee.redhat.com/releng/konflux-release-data) define the release process for konflux built projects.

For new releases two differents directory are important :
- The [ReleasePlanAdmission](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/ocp-network-observab) directory. The `ReleasePlanAdmission` defines the images that are going to be released and the release target.

- The [ReleasePlan](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/tenants-config/cluster/stone-prd-rh01/tenants/ocp-network-observab-tenant?ref_type=heads) directory. The `ReleasePlan` object define which `ReleasePlanAdmission` are going to be part of the release.

To be able to see release pipeline, a read access to the `rhtap-releng` namespace is required, this access must be requested in the konflux-user slack channel.

### Konflux PF4 and PF5 console plugin prerequisite

Sections below use `oc patch components …` and `oc annotate component/…` for `network-observability-console-plugin-pf4-*` and `network-observability-console-plugin-pf5-*`. Those commands **fail with NotFound** if the matching Konflux `Component` does not exist in `ocp-network-observab-tenant`.

Verify PF4 components before running PF4 patch/annotate lines:

```bash
NS=ocp-network-observab-tenant
for c in \
  network-observability-console-plugin-pf4-ystream \
  network-observability-console-plugin-pf4-zstream
do
  if oc get "components.appstudio.redhat.com/$c" -n "$NS" &>/dev/null; then
    echo "ok: $c"
  else
    echo "MISSING: $c" >&2
    echo "  Create this Component in Konflux (applications netobserv-ystream / netobserv-zstream). Names must match [ReleasePlanAdmission](https://gitlab.cee.redhat.com/releng/konflux-release-data/-/tree/main/config/stone-prd-rh01.pg1f.p1/product/ReleasePlanAdmission/ocp-network-observab)." >&2
    exit 1
  fi
done
```

If you use PF5 mirrors, patches, or annotations, verify PF5 components exist first:

```bash
NS=ocp-network-observab-tenant
for c in \
  network-observability-console-plugin-pf5-ystream \
  network-observability-console-plugin-pf5-zstream
do
  if oc get "components.appstudio.redhat.com/$c" -n "$NS" &>/dev/null; then
    echo "ok: $c"
  else
    echo "MISSING: $c" >&2
    echo "  Onboard PF5 in Konflux first, or remove the PF5 ImageDigestMirrorSet block and omit every command referencing network-observability-console-plugin-pf5-*." >&2
    exit 1
  fi
done
```

Skip the second script (and omit PF5 lines in later sections) until PF5 `Component` resources exist.

### Branching

After creating a new release branch, the following steps need to be done:
- update the konflux components source branches (e.g. below for release-10). Skip any `network-observability-console-plugin-pf5-*` line if those components are not onboarded yet (see [Konflux PF4 and PF5 console plugin prerequisite](#konflux-pf4-and-pf5-console-plugin-prerequisite)).

```bash
oc patch components flowlogs-pipeline-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components netobserv-ebpf-agent-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components network-observability-cli-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components network-observability-console-plugin-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components network-observability-operator-bundle-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components network-observability-operator-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10'}]"
oc patch components network-observability-console-plugin-pf4-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10-pf4'}]"
oc patch components network-observability-console-plugin-pf5-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.10-pf5'}]"
```

- update main branches on every repo to disable on-push jobs:
```yaml
    pipelinesascode.tekton.dev/on-cel-expression: "false"
```

- update release branches on every repo to target self:
```yaml
    pipelinesascode.tekton.dev/on-cel-expression: event == "pull_request" && target_branch == "release-1.10"
```
(for both on-push and on-pull-request jobs)

- review the `ReleasePlanAdmission` objects to make sure they are targetting the next release.

## Release candidates

To generate release candidates:
- Make sure we're in a good state: all the desired work is merged on the release branch, konflux 'on-push' jobs have succeeded, the related nudging PRs are merged, and the operator-bundle 'on-push' job (consecutive to nudging) has succeeded as well. With that all set, you should have your bundle image ready with tag `latest` at [quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-ystream:latest](https://quay.io/repository/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-ystream?tab=tags) or [quay.io/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-zstream:latest](https://quay.io/repository/redhat-user-workloads/ocp-network-observab-tenant/network-observability-operator-bundle-zstream?tab=tags).

- Go to your local clone of [netobserv-catalog](https://github.com/netobserv/netobserv-catalog) and run:

```bash
REGISTRY_AUTH_FILE=/path/to/authfile.json make gen-ystream # (or gen-zstream)
```
(It will take a while because it regenerates the full dependency tree, which includes all past versions of netobserv)

- Commit and push

That's going to trigger on-push jobs on [catalog-ystream](https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/ns/ocp-network-observab-tenant/applications/catalog-ystream/activity/pipelineruns) and/or  [catalog-zstream](https://konflux-ui.apps.stone-prd-rh01.pg1f.p1.openshiftapps.com/ns/ocp-network-observab-tenant/applications/catalog-zstream/activity/pipelineruns). When that succeeds, you can consume the release candidate `catalog-ystream:latest` or `catalog-zstream:latest` as described in the [Deploying section](#deploying).

## Release

When a release candidate is accepted and ready to be released, the catalogs repo must be updated with the accepted bundle image:
- Find the desired bundle SHA, e.g. by looking at the latest release snapshot in Konflux.
- Run the following command on the catalogs repo, then commit and push:

```bash
BUNDLE_SHA=(desired bundle SHA) make final-ystream # (or zstream)
```

Once it is ready to be released, a new `Release` object needs to be created to trigger the production release pipeline:

```yaml
apiVersion: appstudio.redhat.com/v1alpha1
kind: Release
metadata:
  name: release-netobserv-1-8-0-0                      # name+version - last digit is the attempt number, in case you need to retry
  namespace: ocp-network-observab-tenant
  labels:
    release.appstudio.openshift.io/author: 'jtakvori'  # your konflux / redhat user
spec:
  releasePlan: netobserv-1-8
  snapshot: netobserv-operator-1-8-9ms9w               # the validated snapshot
```

It must be created on the OCP instance that runs Konflux (ask for the address if you don't have it).

For the record, store the created Release in the `releases` directory of this repo.

## After release

After a release, the following steps should be done:
1. bump / update all repos for next version: you can run [this script](https://github.com/netobserv/documents/blob/main/hack/prepare-next-version.sh) from each repo.
2. merge the nudging PRs that are generated after those changes
3. update ystream and zstream in [netobserv-catalog](https://github.com/netobserv/netobserv-catalog):
  - updating the dependency graph (replace tags...) with the version just-released
  - only after step 2. is complete AND the bundle on-push jobs succeeded, regenerate all catalogs

### Redirecting branches (after ystream release)

After release, we need to repurpose zstream to the just released branch, and ystream to main. Skip `network-observability-console-plugin-pf5-*` patches if those components do not exist (see [Konflux PF4 and PF5 console plugin prerequisite](#konflux-pf4-and-pf5-console-plugin-prerequisite)).

```bash
oc patch components flowlogs-pipeline-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components netobserv-ebpf-agent-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components network-observability-cli-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components network-observability-console-plugin-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components network-observability-operator-bundle-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components network-observability-operator-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main'}]"
oc patch components network-observability-console-plugin-pf4-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main-pf4'}]"
oc patch components network-observability-console-plugin-pf5-ystream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'main-pf5'}]"

oc patch components flowlogs-pipeline-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components netobserv-ebpf-agent-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components network-observability-cli-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components network-observability-console-plugin-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components network-observability-operator-bundle-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components network-observability-operator-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9'}]"
oc patch components network-observability-console-plugin-pf4-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9-pf4'}]"
oc patch components network-observability-console-plugin-pf5-zstream --type='json' -p "[{'op': 'replace', 'path': '/spec/source/git/revision', 'value': 'release-1.9-pf5'}]"
```

### Freezing zstream

You may want to freeze a branch (stop mintmaker from opening PRs) after it was released on zstream, if you don't plan more releases there at the moment. To do so, log in the Konflux' OpenShift and run (skip PF5 annotate lines if those components are not onboarded; see [Konflux PF4 and PF5 console plugin prerequisite](#konflux-pf4-and-pf5-console-plugin-prerequisite)):

```bash
oc -n ocp-network-observab-tenant annotate component/flowlogs-pipeline-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/netobserv-ebpf-agent-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-pf4-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-pf5-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-operator-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-operator-bundle-zstream mintmaker.appstudio.redhat.com/disabled=true
oc -n ocp-network-observab-tenant annotate component/network-observability-cli-zstream mintmaker.appstudio.redhat.com/disabled=true
```

To re-enable them, just delete these annotations

```bash
oc -n ocp-network-observab-tenant annotate component/flowlogs-pipeline-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/netobserv-ebpf-agent-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-pf4-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-console-plugin-pf5-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-operator-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-operator-bundle-zstream mintmaker.appstudio.redhat.com/disabled-
oc -n ocp-network-observab-tenant annotate component/network-observability-cli-zstream mintmaker.appstudio.redhat.com/disabled-
```

### Summary of changes

To summarize, after a release, we should have:
- In all repos, for `main` (=ystream) and `release-1.n` (=zstream) branches, tekton pipelines (in `.tekton` directory) and `Dockerfile-args` files correctly set up as explained [here](https://github.com/netobserv/documents/blob/main/hack/prepare-next-version.sh#L180-L183).
- In Konflux, ystream and zstream components pointing to the expected branches.
- In [netobserv-catalog](https://github.com/netobserv/netobserv-catalog), `templates/y-stream.yaml` is prepared for the 1.nextY, with OLM dependency set up for upgrading from the last version, and `network-observability-operator-bundle-ystream:latest` referenced as the next version.
- In [netobserv-catalog](https://github.com/netobserv/netobserv-catalog), `templates/z-stream.yaml` is prepared for the 1.Y.nextZ, with OLM dependency set up for upgrading from the last version, and `network-observability-operator-bundle-zstream:latest` referenced as the next version.
