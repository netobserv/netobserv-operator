##@ Development helpers

ifeq ("", "$(CSV)")
OPERATOR_NS ?= $(NAMESPACE)
else
OPERATOR_NS ?= openshift-netobserv-operator
endif

.PHONY: deploy-loki
deploy-loki: ## Deploy loki.
	@echo -e "\n==> Deploy loki"
	kubectl create namespace $(NAMESPACE)  --dry-run=client -o yaml | kubectl apply -f -
	kubectl config set-context --current --namespace=$(NAMESPACE)
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/1-storage.yaml | kubectl create -f - || true
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/2-loki.yaml	 | kubectl create -f - || true
	kubectl wait --timeout=180s --for=condition=ready pod -l app=loki
	-pkill --oldest --full "3100:3100"
ifeq (true, $(PORT_FWD))
	-kubectl port-forward --address 0.0.0.0 svc/loki 3100:3100 2>&1 >/dev/null &
	@echo -e "\n===> loki endpoint is available on http://localhost:3100\n"
endif

.PHONY: undeploy-loki-tls
undeploy-loki-tls:
	@echo -e "\n==> Undeploy tls loki"
	kubectl config set-context --current --namespace=$(NAMESPACE)
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/2-loki-tls.yaml	 | kubectl --ignore-not-found=true  delete -f - || true
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/1-storage.yaml | kubectl --ignore-not-found=true  delete -f - || true
	-pkill --oldest --full "3100:3100"

.PHONY: deploy-loki-tls
deploy-loki-tls:
	@echo -e "\n==> Deploy tls loki"
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	kubectl config set-context --current --namespace=$(NAMESPACE)
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/1-storage.yaml | kubectl create -f - || true
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/2-loki-tls.yaml	 | kubectl create -f - || true
	kubectl wait --timeout=180s --for=condition=ready pod -l app=loki
	-pkill --oldest --full "3100:3100"
ifeq (true, $(PORT_FWD))
	-kubectl port-forward --address 0.0.0.0 svc/loki 3100:3100 2>&1 >/dev/null &
	@echo -e "\n===> loki endpoint is available on http://localhost:3100\n"
endif

.PHONY: undeploy-loki
undeploy-loki: ## Undeploy loki.
	@echo -e "\n==> Undeploy loki"
	kubectl config set-context --current --namespace=$(NAMESPACE)
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/2-loki.yaml	 | kubectl --ignore-not-found=true  delete -f - || true
	curl -S -L https://raw.githubusercontent.com/netobserv/documents/main/examples/zero-click-loki/1-storage.yaml | kubectl --ignore-not-found=true  delete -f - || true
	-pkill --oldest --full "3100:3100"

.PHONY: deploy-kafka
deploy-kafka:
	@echo -e "\n==> Deploy default Kafka. Get more help on https://github.com/netobserv/documents/blob/main/kafka.md"
	kubectl create namespace $(NAMESPACE)  --dry-run=client -o yaml | kubectl apply -f -
	curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/strimzi-cluster-operator.yaml" | sed -r 's/namespace: (default|myproject)/namespace: $(NAMESPACE)/g' | kubectl apply -n $(NAMESPACE) -f -
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/kafka-node-pool.yaml" -n $(NAMESPACE)
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/metrics-config.yaml" -n $(NAMESPACE)
	curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/default.yaml" | envsubst | kubectl apply -n $(NAMESPACE) -f -
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/topic.yaml" -n $(NAMESPACE)
	kubectl wait --timeout=180s --for=condition=ready kafkatopic network-flows -n $(NAMESPACE)

.PHONY: deploy-kafka-tls
deploy-kafka-tls:
	@echo -e "\n==> Deploy Kafka with mTLS. Get more help on https://github.com/netobserv/documents/blob/main/kafka.md"
	kubectl create namespace $(NAMESPACE)  --dry-run=client -o yaml | kubectl apply -f -
	curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/strimzi-cluster-operator.yaml" | sed -r 's/namespace: (default|myproject)/namespace: $(NAMESPACE)/g' | kubectl apply -n $(NAMESPACE) -f -
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/kafka-node-pool.yaml" -n $(NAMESPACE)
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/metrics-config.yaml" -n $(NAMESPACE)
	curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/tls.yaml" | envsubst | kubectl apply -n $(NAMESPACE) -f -
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/topic.yaml" -n $(NAMESPACE)
	kubectl apply -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/user.yaml" -n $(NAMESPACE)
	kubectl wait --timeout=180s --for=condition=ready kafkauser flp-kafka -n $(NAMESPACE)

.PHONY: undeploy-kafka
undeploy-kafka: ## Undeploy kafka.
	@echo -e "\n==> Undeploy kafka"
	kubectl delete -f "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/topic.yaml" -n $(NAMESPACE) --ignore-not-found=true
	kubectl delete kafkanodepool kafka-pool -n $(NAMESPACE) --ignore-not-found=true
	kubectl delete kafkauser flp-kafka -n $(NAMESPACE) --ignore-not-found=true
	kubectl delete kafka kafka-cluster -n $(NAMESPACE) --ignore-not-found=true
	curl -s -L "https://raw.githubusercontent.com/netobserv/documents/main/examples/kafka/strimzi-cluster-operator.yaml" | sed -r 's/namespace: (default|myproject)/namespace: $(NAMESPACE)/g' | kubectl delete -n $(NAMESPACE) -f -

.PHONY: deploy-grafana
deploy-grafana: ## Deploy grafana.
	@echo -e "\n==> Deploy grafana"
	kubectl create namespace $(NAMESPACE)  --dry-run=client -o yaml | kubectl apply -f -
	kubectl config set-context --current --namespace=$(NAMESPACE)
	./hack/deploy-grafana.sh $(NAMESPACE)
	-pkill --oldest --full "3000:3000"
ifeq (true, $(PORT_FWD))
	-kubectl port-forward --address 0.0.0.0 svc/grafana 3000:3000 2>&1 >/dev/null &
	@echo -e "\n===> grafana ui is available (user: admin password: admin) on http://localhost:3000\n"
endif

.PHONY: undeploy-grafana
undeploy-grafana: ## Undeploy grafana.
	@echo -e "\n==> Undeploy grafana"
	kubectl config set-context --current --namespace=$(NAMESPACE)
	kubectl delete --ignore-not-found=true deployment grafana
	kubectl delete --ignore-not-found=true service grafana
	kubectl delete --ignore-not-found=true configMap grafana-datasources
	-pkill --oldest --full "3000:3000"

.PHONY: deploy-infra
deploy-infra: manifests generate fmt lint deploy-loki deploy-grafana install

.PHONY: deploy-all
deploy-all: deploy-infra deploy-sample-cr deploy-sample-workload

.PHONY: undeploy-infra
undeploy-infra: undeploy-loki undeploy-grafana uninstall

.PHONY: undeploy-all
undeploy-all: undeploy-infra undeploy-sample-cr undeploy-sample-workload

.PHONY: deploy-prometheus
deploy-prometheus: ## Deploy prometheus.
	@echo -e "\n==> Deploy prometheus"
	kubectl create namespace $(NAMESPACE) || true
	kubectl config set-context --current --namespace=$(NAMESPACE)
	kubectl apply -f config/kind/deployment-prometheus.yaml
	kubectl rollout status "deploy/prometheus" --timeout=600s
	-pkill --oldest --full "9090:9090"
ifeq (true, $(PORT_FWD))
	kubectl port-forward --address 0.0.0.0 svc/prometheus 9090:9090 2>&1 >/dev/null &
	@echo -e "\n===> prometheus ui is available on http://localhost:9090\n"
endif

.PHONY: undeploy-prometheus
undeploy-prometheus: ## Undeploy prometheus.
	@echo -e "\n==> Undeploy prometheus"
	kubectl --ignore-not-found=true delete -f config/kind/deployment-prometheus.yaml || true
	-pkill --oldest --full "9090:9090"

.PHONY: get-related-images
get-related-images:
	kubectl set env -n $(NAMESPACE) deployment netobserv-controller-manager -c "manager" --list

.PHONY: set-agent-image
set-agent-image:
ifeq ("", "$(CSV)")
	kubectl set env -n $(NAMESPACE) deployment netobserv-controller-manager -c "manager" RELATED_IMAGE_EBPF_AGENT=$(IMAGE_REGISTRY)/$(USER)/netobserv-ebpf-agent:$(VERSION)
else
	./hack/swap-image-csv.sh $(CSV) $(OPERATOR_NS) ebpf-agent RELATED_IMAGE_EBPF_AGENT $(IMAGE_REGISTRY)/$(USER)/netobserv-ebpf-agent:$(VERSION)
endif
	@echo -e "\n==> Redeploying..."
	kubectl rollout status -n $(OPERATOR_NS) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(OPERATOR_NS) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager
	@echo -e "\n==> Wait a moment before agents are fully redeployed"

.PHONY: set-flp-image
set-flp-image:
ifeq ("", "$(CSV)")
	kubectl set env -n $(NAMESPACE) deployment netobserv-controller-manager -c "manager" RELATED_IMAGE_FLOWLOGS_PIPELINE=$(IMAGE_REGISTRY)/$(USER)/flowlogs-pipeline:$(VERSION)
else
	./hack/swap-image-csv.sh $(CSV) $(OPERATOR_NS) flowlogs-pipeline RELATED_IMAGE_FLOWLOGS_PIPELINE $(IMAGE_REGISTRY)/$(USER)/flowlogs-pipeline:$(VERSION)
endif
	@echo -e "\n==> Redeploying..."
	kubectl rollout status -n $(OPERATOR_NS) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(OPERATOR_NS) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager
	@echo -e "\n==> Wait a moment before FLP is fully redeployed"

.PHONY: set-plugin-image
set-plugin-image:
ifeq ("", "$(CSV)")
	kubectl set env -n $(NAMESPACE) deployment netobserv-controller-manager -c "manager" RELATED_IMAGE_CONSOLE_PLUGIN=$(IMAGE_REGISTRY)/$(USER)/network-observability-console-plugin:$(VERSION)
	kubectl set image deployment/netobserv-plugin-static netobserv-plugin-static=$(IMAGE_REGISTRY)/$(USER)/network-observability-console-plugin:$(VERSION)
else
	./hack/swap-image-csv.sh $(CSV) $(OPERATOR_NS) console-plugin RELATED_IMAGE_CONSOLE_PLUGIN $(IMAGE_REGISTRY)/$(USER)/network-observability-console-plugin:$(VERSION)
endif
	@echo -e "\n==> Redeploying..."
	kubectl rollout status -n $(OPERATOR_NS) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(OPERATOR_NS) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager
	@echo -e "\n==> Wait a moment before plugin pod is fully redeployed"

.PHONY: set-release-kind-downstream
set-release-kind-downstream:
ifeq ("", "$(CSV)")
	kubectl  -n $(NAMESPACE) set env deployment netobserv-controller-manager -c "manager" DOWNSTREAM_DEPLOYMENT=true
else
	./hack/swap-image-csv.sh $(CSV) $(OPERATOR_NS) "" DOWNSTREAM_DEPLOYMENT true
endif
	@echo -e "\n==> Redeploying..."
	kubectl rollout status -n $(OPERATOR_NS) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(OPERATOR_NS) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager

.PHONY: pprof
pprof:
	@echo -e "\n==> Enabling pprof... Check https://github.com/netobserv/netobserv-operator/blob/main/DEVELOPMENT.md#profiling for help."
	kubectl -n $(NAMESPACE) set env deployment netobserv-controller-manager -c "manager" PROFILING_BIND_ADDRESS=:6060
	@echo -e "\n==> Redeploying..."
	kubectl rollout status -n $(NAMESPACE) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(NAMESPACE) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager
	sleep 2
	$(MAKE) pprof-pf

.PHONY: pprof-pf
pprof-pf:
	@echo -e "\n==> Port-forwarding..."
	oc get pods
	kubectl port-forward -n $(NAMESPACE) $(shell kubectl get pod -l app=netobserv-operator -n $(NAMESPACE) -o jsonpath="{.items[0].metadata.name}") 6060

.PHONY: use-test-console
use-test-console:
	@echo -e "\n==> Enabling the test console..."
ifeq ("", "$(CSV)")
	kubectl set env -n $(NAMESPACE) deployment netobserv-controller-manager -c "manager" RELATED_IMAGE_CONSOLE_PLUGIN=$(IMAGE_REGISTRY)/$(USER)/network-observability-standalone-frontend:$(VERSION)
else
	./hack/swap-image-csv.sh $(CSV) $(OPERATOR_NS) console-plugin RELATED_IMAGE_CONSOLE_PLUGIN $(IMAGE_REGISTRY)/$(USER)/network-observability-standalone-frontend:$(VERSION)
endif
	@echo -e "\n==> Waiting for operator redeployed..."
	kubectl rollout status -n $(OPERATOR_NS) --timeout=60s deployment netobserv-controller-manager
	kubectl wait -n $(OPERATOR_NS) --timeout=60s --for condition=Available=True deployment netobserv-controller-manager
	oc patch flowcollector cluster --type='json' -p '[{"op": "add", "path": "/spec/consolePlugin/standalone", "value": true}]'
	@echo -e "\n==> Waiting for console-plugin pod..."
	kubectl delete -n $(NAMESPACE) deployment netobserv-plugin
	while ! kubectl get deployment netobserv-plugin; do sleep 1; done
	kubectl rollout status -n $(NAMESPACE) --timeout=60s deployment netobserv-plugin
	kubectl wait -n $(NAMESPACE) --timeout=60s --for condition=Available=True deployment netobserv-plugin
	-pkill --oldest --full "9001:9001"
	kubectl port-forward -n $(NAMESPACE) svc/netobserv-plugin 9001:9001 2>&1 >/dev/null &
	@echo -e "\n===> Test console available at http://localhost:9001\n"
