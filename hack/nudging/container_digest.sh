# Do not remove comment lines, they are there to reduce conflicts
# Operator
export OPERATOR_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-rhel9-operator@sha256:13da8fa05b786f309eda1622b0c181bf68beb77cebc498e3586dce1fc7594e79'
# eBPF agent
export EBPF_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-ebpf-agent-rhel9@sha256:def1442ce8e925c6d753fb38ed9e944c3ca3598f21ad0e58948c718e3416b1a3'
# Flowlogs-pipeline
export FLP_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-flowlogs-pipeline-rhel9@sha256:2855d2460b16013653afe70658c8e0ec0b07c4239466017880b2fdffa6263b2f'
# Console plugin
export CONSOLE_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-rhel9@sha256:69ab41a5f261202f90bd0f5357830c4c5aef7628ba04804d6ee17c789861528e'
# Console plugin PF4 (default / OCP < 4.15)
export CONSOLE_PF4_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-compat-rhel9@sha256:44a1952996d6ee8b7fe15d66aa5325491b022dcb5cc3e95c4085005bff1d6e9c'
# Console plugin PF5 (OCP 4.15–4.21)
export CONSOLE_PF5_IMAGE_PULLSPEC='registry.redhat.io/network-observability/network-observability-console-plugin-pf5-rhel9@sha256:TODO'
