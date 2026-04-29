# Do not remove comment lines, they are there to reduce conflicts
# Operator
export OPERATOR_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-rhel9-operator@sha256:e0697fff6fab94f9b3e301edab53b8c61167c86ff94f536974428993e62a0bc1'
# eBPF agent
export EBPF_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-ebpf-agent-rhel9@sha256:da1b88f64a8dec15dbd4fee877e62cdc134ff23b1f94a3ec384a5a154bbb938d'
# Flowlogs-pipeline
export FLP_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-flowlogs-pipeline-rhel9@sha256:f05d0a8581d06f2be06fea6163ef3550736819c2296e522dab0d5ba78d167f67'
# Console plugin
export CONSOLE_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-rhel9@sha256:d76183d11cac09609f5c7464e6dc969ada8a04398cac813d36f53eab3162e3a2'
# Console plugin PF4 (default / OCP < 4.15)
export CONSOLE_PF4_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-compat-rhel9@sha256:44a1952996d6ee8b7fe15d66aa5325491b022dcb5cc3e95c4085005bff1d6e9c'
# Console plugin PF5 (OCP 4.15–4.21)
export CONSOLE_PF5_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-pf5-rhel9@sha256:TODO'
