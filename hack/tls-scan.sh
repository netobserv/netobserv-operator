#!/usr/bin/env bash

# Check PQC readiness and TLS profile adherence with openshift/tls-scanner.
set -euo pipefail

: "${TLS_SCANNER_IMAGE:?Set TLS_SCANNER_IMAGE to a pullable openshift/tls-scanner image}"

scanner_namespace=${TLS_SCANNER_NAMESPACE:-netobserv}
operator_namespace=${TLS_SCANNER_OPERATOR_NAMESPACE:-netobserv}
scan_namespaces=${TLS_SCANNER_NAMESPACES:-netobserv,netobserv-privileged,$operator_namespace}
output_root=${TLS_SCANNER_OUTPUT_DIR:-out/tls-scanner}
parallel=${TLS_SCANNER_PARALLEL:-4}
timeout_seconds=${TLS_SCANNER_TIMEOUT_SECONDS:-1800}

for command in oc jq; do
    command -v "$command" >/dev/null || { echo "Missing command: $command" >&2; exit 1; }
done

[[ $scanner_namespace =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]] || { echo "Invalid scanner namespace" >&2; exit 1; }
[[ $operator_namespace =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]] || { echo "Invalid operator namespace" >&2; exit 1; }
[[ $scan_namespaces =~ ^[a-z0-9,-]+$ ]] || { echo "Invalid namespace filter" >&2; exit 1; }
[[ $TLS_SCANNER_IMAGE =~ ^[a-zA-Z0-9._/@:-]+$ ]] || { echo "Invalid scanner image" >&2; exit 1; }
[[ $parallel =~ ^[1-9][0-9]*$ ]] || { echo "TLS_SCANNER_PARALLEL must be positive" >&2; exit 1; }
[[ $timeout_seconds =~ ^[1-9][0-9]*$ ]] || { echo "TLS_SCANNER_TIMEOUT_SECONDS must be positive" >&2; exit 1; }

oc get namespace "$scanner_namespace" >/dev/null
oc get namespace "$operator_namespace" >/dev/null
IFS=, read -ra namespaces <<< "$scan_namespaces"
for namespace in "${namespaces[@]}"; do
    oc get namespace "$namespace" >/dev/null
done

name="netobserv-tls-scan-$(date +%s)-$$"
output_dir="$output_root/$name"
work_dir=$(mktemp -d)
manifest="$work_dir/scanner.yaml"

# Remove temporary scan resources while preserving the script's exit status.
cleanup() {
    status=$?
    trap - EXIT
    oc delete -f "$manifest" --ignore-not-found --wait=false >/dev/null 2>&1 || true
    rm -rf "$work_dir"
    exit "$status"
}
trap cleanup EXIT

cat > "$manifest" <<EOF
apiVersion: v1
kind: ServiceAccount
metadata:
  name: $name
  namespace: $scanner_namespace
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: $name
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list"]
  - apiGroups: [""]
    resources: ["pods/exec"]
    verbs: ["create"]
  - apiGroups: ["config.openshift.io"]
    resources: ["apiservers"]
    verbs: ["get"]
  - apiGroups: ["operator.openshift.io"]
    resources: ["ingresscontrollers"]
    verbs: ["get"]
  - apiGroups: ["machineconfiguration.openshift.io"]
    resources: ["kubeletconfigs"]
    verbs: ["list"]
  - apiGroups: ["security.openshift.io"]
    resources: ["securitycontextconstraints"]
    resourceNames: ["privileged"]
    verbs: ["use"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: $name
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: $name
subjects:
  - kind: ServiceAccount
    name: $name
    namespace: $scanner_namespace
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: $name-operator
  namespace: $operator_namespace
spec:
  podSelector:
    matchLabels:
      app: netobserv-operator
  policyTypes: [Ingress]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: $scanner_namespace
          podSelector:
            matchLabels:
              netobserv-tls-scan: $name
EOF

# NetObserv's ingress policies allow same-namespace traffic, but an isolated
# scanner namespace needs explicit access to the operands in each scan namespace.
declare -A policy_namespaces=()
for namespace in "${namespaces[@]}"; do
    [[ -n ${policy_namespaces[$namespace]:-} ]] && continue
    policy_namespaces[$namespace]=1
    cat >> "$manifest" <<EOF
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: $name
  namespace: $namespace
spec:
  podSelector:
    matchLabels:
      part-of: netobserv-operator
  policyTypes: [Ingress]
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: $scanner_namespace
          podSelector:
            matchLabels:
              netobserv-tls-scan: $name
EOF
done

cat >> "$manifest" <<EOF
---
apiVersion: batch/v1
kind: Job
metadata:
  name: $name
  namespace: $scanner_namespace
spec:
  backoffLimit: 0
  template:
    metadata:
      labels:
        netobserv-tls-scan: $name
    spec:
      serviceAccountName: $name
      restartPolicy: Never
      containers:
        - name: scanner
          image: $TLS_SCANNER_IMAGE
          imagePullPolicy: Always
          command: ["/bin/sh", "-c"]
          args:
            - |
              mkdir -p /artifacts/adherence
              /usr/local/bin/tls-scanner --all-pods --namespace-filter="$scan_namespaces" --pqc-check -j=$parallel --artifact-dir=/artifacts --json-file=/artifacts/results.json --csv-file=/artifacts/results.csv --junit-file=/artifacts/results.xml --log-file=/artifacts/scan.log
              pqc_result=\$?
              printf '%s\\n' "\$pqc_result" > /artifacts/pqc-exit-code
              /usr/local/bin/tls-scanner --all-pods --namespace-filter="$scan_namespaces" -j=$parallel --artifact-dir=/artifacts/adherence --json-file=/artifacts/adherence/results.json --csv-file=/artifacts/adherence/results.csv --junit-file=/artifacts/adherence/results.xml --log-file=/artifacts/adherence/scan.log
              adherence_result=\$?
              printf '%s\\n' "\$adherence_result" > /artifacts/adherence-exit-code
              result=0
              if [ "\$pqc_result" -ne 0 ] || [ "\$adherence_result" -ne 0 ]; then result=1; fi
              printf '%s\\n' "\$result" > /artifacts/exit-code
              sleep 600
              exit "\$result"
          resources:
            requests:
              cpu: 250m
              memory: 512Mi
            limits:
              cpu: "2"
              memory: 2Gi
          volumeMounts:
            - name: artifacts
              mountPath: /artifacts
      volumes:
        - name: artifacts
          emptyDir: {}
EOF

echo "Scanning namespaces $scan_namespaces with $TLS_SCANNER_IMAGE"
oc create -f "$manifest"

started=$SECONDS
pod=""
while (( SECONDS - started < timeout_seconds )); do
    pod=$(oc get pods -n "$scanner_namespace" -l "job-name=$name" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    if [[ -n $pod ]] && oc exec -n "$scanner_namespace" "$pod" -- test -s /artifacts/exit-code >/dev/null 2>&1; then
        break
    fi
    if [[ -n $pod ]]; then
        phase=$(oc get pod -n "$scanner_namespace" "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || true)
        if [[ $phase == Failed || $phase == Succeeded ]]; then
            echo "Scanner pod ended before writing results ($phase)" >&2
            oc logs -n "$scanner_namespace" "$pod" --tail=60 >&2 || true
            exit 1
        fi
        waiting=$(oc get pod -n "$scanner_namespace" "$pod" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || true)
        case $waiting in
            ErrImagePull|ImagePullBackOff|InvalidImageName|CreateContainerConfigError)
                echo "Scanner pod cannot start ($waiting)" >&2
                oc describe pod -n "$scanner_namespace" "$pod" >&2 || true
                exit 1
                ;;
        esac
    fi
    sleep 5
done

if [[ -z $pod ]] || ! oc exec -n "$scanner_namespace" "$pod" -- test -s /artifacts/exit-code >/dev/null 2>&1; then
    echo "Timed out waiting for tls-scanner results" >&2
    [[ -z $pod ]] || oc logs -n "$scanner_namespace" "$pod" --tail=60 >&2 || true
    exit 1
fi

mkdir -p "$output_dir"
oc cp "$scanner_namespace/$pod:/artifacts/." "$output_dir"
oc logs -n "$scanner_namespace" "$pod" > "$output_dir/job.log"

artifacts=(results.json results.csv results.xml scan.log job.log exit-code pqc-exit-code adherence-exit-code
    adherence/results.json adherence/results.csv adherence/results.xml adherence/scan.log)
scanner_exit=1
pqc_exit=unknown
adherence_exit=unknown
if [[ -s $output_dir/exit-code ]]; then scanner_exit=$(cat "$output_dir/exit-code"); fi
if [[ -s $output_dir/pqc-exit-code ]]; then pqc_exit=$(cat "$output_dir/pqc-exit-code"); fi
if [[ -s $output_dir/adherence-exit-code ]]; then adherence_exit=$(cat "$output_dir/adherence-exit-code"); fi
{
    echo '# NetObserv TLS scan'
    echo
    echo "Namespaces: \`$scan_namespaces\`"
    echo
    echo "Run: \`$name\`"
    echo
    echo "PQC check exit code: \`$pqc_exit\`"
    echo
    echo "TLS profile adherence exit code: \`$adherence_exit\`"
    echo
    for artifact in "${artifacts[@]}"; do
        if [[ ! -f $output_dir/$artifact ]]; then
            echo "Missing artifact: \`$artifact\`; see available scan logs and \`job.log\`."
            echo
        fi
    done
    if [[ -s $output_dir/results.json ]]; then
        echo '## PQC scan status counts'
        echo
        jq -r '[.ip_results[] | if (.port_results | length) > 0 then .port_results[].status else .status end] | group_by(.)[] | "- \(.[0]): \(length)"' "$output_dir/results.json"
        echo
        echo '## Ports'
        echo
        echo '| Namespace | Pod | Port | Status | TLS 1.3 | ML-KEM |'
        echo '| --- | --- | ---: | --- | --- | --- |'
        jq -r '.ip_results[] | . as $ip | if (.port_results | length) > 0 then .port_results[] | "| \($ip.pod.Namespace // "-") | \($ip.pod.Name // "-") | \(.port) | \(.status) | \(if .tls13_supported then "yes" else "-" end) | \(if .mlkem_supported then "yes" else "-" end) |" else "| \($ip.pod.Namespace // "-") | \($ip.pod.Name // "-") | - | \($ip.status) | - | - |" end' "$output_dir/results.json"
        echo
    fi
    if [[ -s $output_dir/adherence/results.json ]]; then
        echo '## TLS profile adherence scan status counts'
        echo
        jq -r '[.ip_results[] | if (.port_results | length) > 0 then .port_results[].status else .status end] | group_by(.)[] | "- \(.[0]): \(length)"' "$output_dir/adherence/results.json"
        echo
        echo '| Namespace | Pod | Port | Status | Profile | Version | Ciphers |'
        echo '| --- | --- | ---: | --- | --- | --- | --- |'
        jq -r '.ip_results[] | . as $ip | (.port_results // [])[] | .api_server_tls_config_compliance as $check | "| \($ip.pod.Namespace // "-") | \($ip.pod.Name // "-") | \(.port) | \(.status) | \($check.configured_profile // "-") | \(if $check == null then "-" elif $check.version then "yes" else "no" end) | \(if $check == null then "-" elif $check.ciphers then "yes" else "no" end) |"' "$output_dir/adherence/results.json"
        echo
        echo 'See `adherence/results.json` and `adherence/results.csv` for the full compliance details.'
        echo
    fi
    echo 'NO_TLS, FILTERED, and NO_PORTS need review: network policy and pod security can hide TLS listeners from the scanner.'
} > "$output_dir/report.md"

mkdir -p "$output_root/adherence"
for artifact in "${artifacts[@]}"; do
    if [[ -f $output_dir/$artifact ]]; then
        cp "$output_dir/$artifact" "$output_root/$artifact"
    else
        # Keep the latest artifacts consistent with this run's report.
        rm -f "$output_root/$artifact"
    fi
done
cp "$output_dir/report.md" "$output_root/report.md"

cat "$output_dir/report.md"
echo "Reports: $output_dir (PQC and adherence JSON, CSV, JUnit, logs)"
echo "Latest report: $output_root/report.md"
exit "$scanner_exit"
