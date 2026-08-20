FROM scratch

# Core bundle labels.
LABEL operators.operatorframework.io.bundle.mediatype.v1=registry+v1
LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=netobserv-operator
LABEL operators.operatorframework.io.bundle.channels.v1=latest,community
LABEL operators.operatorframework.io.bundle.channel.default.v1=community
LABEL operators.operatorframework.io.metrics.builder=operator-sdk-v1.42.0
LABEL operators.operatorframework.io.metrics.mediatype.v1=metrics+v1
LABEL operators.operatorframework.io.metrics.project_layout=go.kubebuilder.io/v4

# Copy files to locations specified by labels.
COPY bundles/k8s/manifests /manifests/
COPY bundles/k8s/metadata /metadata/
