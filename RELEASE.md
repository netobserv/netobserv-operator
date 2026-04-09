## Releasing process

### Draft release - related components

All components deployed by this operator can be released separatly, at their own pace.

To release them, a tag in the format "v1.6.0-community" or "v1.6.0-crc0" must be set on the desired clean HEAD state (generally, up-to-date `main` branch; "crc" stands for "community release candidate"), then pushed. It applies to [the console plugin](https://github.com/netobserv/netobserv-web-console/), [flowlogs-pipeline](https://github.com/netobserv/flowlogs-pipeline) and [netobserv-ebpf-agent](https://github.com/netobserv/netobserv-ebpf-agent).

E.g:

```bash
version="v1.11.3-community"
git tag -a "$version" -m "$version"
git push upstream --tags
```

The release script should be triggered (check github actions). It will automatically draft a new release, with artifacts attached.

### Draft release - operator

We can then proceed with the operator. Edit the [Makefile](./Makefile) to update `BUNDLE_VERSION`.

```bash
BUNDLE_SET_DATE=true make update-bundle

# Set desired operator version - CAREFUL, no leading "v" here
version="1.11.3-community"
vv=v$version
test_branch=test-$vv

git commit -a -m "Prepare release $vv"
# Push to a test branch, and tag for release
git push upstream HEAD:$test_branch
git tag -a "$version" -m "$version"
git push upstream --tags
```

The release script should be triggered ([check github actions](https://github.com/netobserv/netobserv-operator/actions)).

### Testing

When all component drafts are ready, you can test the helm chart on your cluster:

```bash
helm repo add cert-manager https://charts.jetstack.io
helm install cert-manager -n cert-manager --create-namespace cert-manager/cert-manager --set crds.enabled=true
helm upgrade trust-manager oci://quay.io/jetstack/charts/trust-manager --install --namespace cert-manager --wait

helm install netobserv -n netobserv --create-namespace --set install.loki=true --set install.prom-stack=true ./helm

cat <<EOF | kubectl apply -f -
apiVersion: flows.netobserv.io/v1beta2
kind: FlowCollector
metadata:
  name: cluster
spec:
  networkPolicy:
    enable: false
  consolePlugin:
    standalone: true
  processor:
    consumerReplicas: 1
    service:
      tlsType: Auto-mTLS
  loki:
    mode: Monolithic
    monolithic:
      url: 'http://netobserv-loki.netobserv.svc.cluster.local.:3100/'
  prometheus:
    querier:
      mode: Manual
      manual:
        url: http://netobserv-prom-stack-prometheus.netobserv.svc.cluster.local.:9090/
        alertManager:
          url: http://netobserv-prom-stack-alertmanager.netobserv.svc.cluster.local.:9093/
EOF

# Check components image:
kubectl config set-context --current --namespace=netobserv
kubectl get pods -oyaml | grep image:
kubectl get pods -n netobserv-privileged -oyaml | grep image:

kubectl wait -n netobserv --timeout=60s --for condition=Available=True deployment netobserv-plugin
kubectl port-forward svc/netobserv-plugin 9001:9001 -n netobserv
```

Then open http://localhost:9001/ in your browser, and do some manual smoke tests.

To clean up:

```bash
helm delete netobserv -n netobserv
```

### Commit operator changes

When everything is ok, push to main and delete the test branch

```bash
git push upstream HEAD:main
git push upstream :$test_branch
```

### Publish releases - related components

Use the github interface to accept the releases, via:
- [console plugin](https://github.com/netobserv/netobserv-web-console/releases)
- [flowlogs-pipeline](https://github.com/netobserv/flowlogs-pipeline/releases)
- [netobserv-ebpf-agent](https://github.com/netobserv/netobserv-ebpf-agent/releases)

Edit the draft, set the previous tag then click the "Generate release notes" button.

If you think the "Dependencies" section is too long, you can surround it in a `<details>` block, to make it collapsed. E.g:

```yaml
<details>
<summary><b>Dependencies</b></summary>

* Bump [...] from [...] by @dependabot in...
* ...
</details>
```

### Publish releases - operator

Use the github interface to accept the release, via:
- [operator](https://github.com/netobserv/netobserv-operator/releases)

Edit the draft, set the previous tag then click the "Generate release notes" button. Like previously, don't hesitate to surround Dependencies in a `<details>` block.

Grab related components release notes by running:

```bash
make related-release-notes
```

The script should fetch and copy the content in the clipboard. Paste it at the end of the auto-generated release note in GitHub.

Check the "Create a discussion for this release" option, in category "Announcements".

Click on "Publish release".

### Publishing the Helm chart

From the operator repository:

```bash
helm package helm/
index_path=/path/to/netobserv.github.io/static/helm
mkdir -p $index_path/new && mv netobserv-operator-1.11.3.tgz $index_path/new && cd $index_path
helm repo index --merge index.yaml new/ --url https://netobserv.io/static/helm/
mv new/* . && rmdir new

# Now, check there's nothing wrong in the generated files before commit (comparing last 2 versions)
colordiff <(yq '.entries.netobserv-operator[1]' index.yaml) <(yq '.entries.netobserv-operator[0]' index.yaml)

git add netobserv-operator-1.11.3.tgz index.yaml
git commit -m "Publish helm 1.11.3-community"
git push upstream HEAD:main
```

Check ArtifactHub for update after a few minutes: https://artifacthub.io/packages/helm/netobserv/netobserv-operator
