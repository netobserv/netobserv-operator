##@ Local (Kind)
KIND_VERSION=v0.22.0
CERT_MANAGER_VERSION=v1.16.3
CERT_MANAGER_URL ?= "https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml"

.PHONY: prereqs-kind
prereqs-kind: ## Check if prerequisites are met for running kind, and install missing dependencies
	@echo "### Checking if KIND prerequisites are met, and installing missing dependencies"
	GOFLAGS="" go install sigs.k8s.io/kind@$(KIND_VERSION)

.PHONY: install-cert-manager
install-cert-manager: ## Install cert manager onto the target kubernetes cluster
	set -e ;\
	kubectl apply -f $(CERT_MANAGER_URL) ;\
	hack/wait_for_cert_manager.sh ;\

.PHONY: uninstall-cert-manager
uninstall-cert-manager: ## Uninstall cert manager from the target kubernetes cluster
	kubectl delete -f $(CERT_MANAGER_URL)

.PHONY: create-kind-cluster
create-kind-cluster: prereqs-kind ## Create kind cluster
	kind create cluster --config config/kind/kind.config.yaml
	kubectl cluster-info --context kind-kind

.PHONY: delete-kind-cluster
delete-kind-cluster: prereqs-kind ## Delete kind cluster
	kind delete cluster

.PHONY: local-deploy
local-deploy: create-kind-cluster install-cert-manager deploy-all  ## Local deploy (kind, loki, grafana, example-cr and sample-workload excluding the operator)

.PHONY: clean-leftovers
clean-leftovers:
	-PID=$$(pgrep --oldest --full "main.go"); pkill -P $$PID; pkill $$PID
	-kubectl delete namespace netobserv

.PHONY: local-redeploy
local-redeploy: clean-leftovers undeploy-all deploy-all  ## Local re-deploy (loki, grafana, example-cr and sample-workload excluding the operator)

.PHONY: local-undeploy
local-undeploy: clean-leftovers uninstall-cert-manager undeploy-all delete-kind-cluster  ## Local cleanup

local-run: create-kind-cluster local-redeploy local-deploy-operator ## local-redeploy + run the operator locally

.PHONY: local-deploy-operator
local-deploy-operator:
# TODO: restore traffic generator, but using IPFIX instead of NFv5 (could be inspired from https://github.com/netobserv/flowlogs-pipeline/blob/main/pkg/test/ipfix.go)
	@echo "====> Running the operator locally (in background process)"
	go run ./main.go \
		-ebpf-agent-image=quay.io/netobserv/netobserv-ebpf-agent:main \
		-flowlogs-pipeline-image=quay.io/netobserv/flowlogs-pipeline:main \
		-console-plugin-images=quay.io/netobserv/network-observability-console-plugin:main \
		-namespace=${NAMESPACE} &
	@echo "====> Waiting for flowlogs-pipeline pod to be ready"
	while : ; do kubectl get ds flowlogs-pipeline && break; sleep 1; done
	kubectl wait --timeout=180s --for=condition=ready pod -l app=flowlogs-pipeline
	@echo "====> Operator process info"
	@PID=$$(pgrep --oldest --full "main.go"); echo -e "\n===> The operator is running in process $$PID\nTo stop the operator process use: pkill -p $$PID"
	@echo "====> Done"

.PHONY: deploy-kind
deploy-kind: install-cert-manager kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	kubectl create namespace netobserv || true
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMAGE}
	$(SED) -i -r 's~ebpf-agent:.+~ebpf-agent:main~' ./config/manager/manager.yaml
	$(SED) -i -r 's~flowlogs-pipeline:.+~flowlogs-pipeline:main~' ./config/manager/manager.yaml
	$(SED) -i -r 's~console-plugin:.+~console-plugin:main~' ./config/manager/manager.yaml
	$(KUSTOMIZE) build config/k8s-olm/default | kubectl apply --server-side -f -

.PHONY: undeploy-kind
undeploy-kind: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/k8s-olm/default | kubectl --ignore-not-found=true delete -f - || true
