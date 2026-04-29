# Do not remove comment lines, they are there to reduce conflicts
# Operator
export OPERATOR_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-rhel9-operator@sha256:e0697fff6fab94f9b3e301edab53b8c61167c86ff94f536974428993e62a0bc1'
# eBPF agent
export EBPF_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-ebpf-agent-rhel9@sha256:460548f409f18e4d42298db520eb64982947cc857e89c270775717238229e11b'
# Flowlogs-pipeline
export FLP_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-flowlogs-pipeline-rhel9@sha256:14f5d89958ae540251490b1becb5ceda5ed2b5e421945105dd86ddaba42aae27'
# Console plugin
export CONSOLE_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-rhel9@sha256:d76183d11cac09609f5c7464e6dc969ada8a04398cac813d36f53eab3162e3a2'
# Console plugin PF4 (default / OCP < 4.15)
export CONSOLE_PF4_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-compat-rhel9@sha256:44a1952996d6ee8b7fe15d66aa5325491b022dcb5cc3e95c4085005bff1d6e9c'
# Console plugin PF5 (OCP 4.15–4.21)
export CONSOLE_PF5_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-pf5-rhel9@sha256:TODO'
