# Do not remove comment lines, they are there to reduce conflicts
# Operator
export OPERATOR_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-rhel9-operator@sha256:14e13c1259d235749d71796eef495c6fc51ee50a209d213c317c63ba6cdcf4ce'
# eBPF agent
export EBPF_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-ebpf-agent-rhel9@sha256:cff7775017a406f645c38ed59761dd9604505ca633abe43bee981fc02048466a'
# Flowlogs-pipeline
export FLP_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-flowlogs-pipeline-rhel9@sha256:f05d0a8581d06f2be06fea6163ef3550736819c2296e522dab0d5ba78d167f67'
# Console plugin
export CONSOLE_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-rhel9@sha256:014bb39536fb70bb8c3710a654ea9d6c1af941d344ed431514758c88165ba1fd'
# Console plugin PF4 (default / OCP < 4.15)
export CONSOLE_PF4_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-compat-rhel9@sha256:44a1952996d6ee8b7fe15d66aa5325491b022dcb5cc3e95c4085005bff1d6e9c'
# Console plugin PF5 (OCP 4.15–4.21)
export CONSOLE_PF5_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-pf5-rhel9@sha256:TODO'
