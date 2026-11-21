##@ Local (Helm)
.PHONY: prereqs-helm
prereqs-helm: ## Check if prerequisites are met for running helm, and install missing dependencies
	@which helm 2>/dev/null || (echo "Helm CLI not installed, please visit https://helm.sh/docs/intro/install/" && exit 1)

.PHONY: helm-install-prereqs
helm-install-prereqs: prereqs-helm ## Install pre-requisites to a running cluster, using Helm
	cd helm && helm dependency update --skip-refresh && cd ..
	helm repo add cert-manager https://charts.jetstack.io
	helm upgrade --install cert-manager -n cert-manager --create-namespace cert-manager/cert-manager --set crds.enabled=true
	helm upgrade --install trust-manager -n cert-manager oci://quay.io/jetstack/charts/trust-manager --wait

IMAGE_FOR_HELM := $(word 1,$(subst :, ,${IMAGE}))
VERSION_FOR_HELM := $(word 2,$(subst :, ,${IMAGE}))
.PHONY: helm-install
helm-install: helm-install-prereqs ## Install the operator (latest builds) and its pre-requisites to a running cluster, using Helm
	helm install netobserv -n netobserv --create-namespace \
		--set operator.image=${IMAGE_FOR_HELM} \
		--set operator.version=${VERSION_FOR_HELM} \
		--set ebpfAgent.version=main \
		--set flowlogsPipeline.version=main \
		--set standaloneConsole.version=main \
		--set install.loki=true \
		--set install.prom-stack=true \
		./helm
	kubectl config set-context --current --namespace=netobserv

.PHONY: helm-install-release
helm-install-release: helm-install-prereqs ## Install the operator (current release) and its pre-requisites to a running cluster, using Helm
	helm install netobserv -n netobserv --create-namespace --set install.loki=true --set install.prom-stack=true ./helm
	kubectl config set-context --current --namespace=netobserv

.PHONY: helm-cleanup
helm-cleanup: prereqs-helm ## Uninstall the operator (do not uninstall the pre-requisites)
	kubectl delete flowcollector cluster --ignore-not-found=true
	helm delete netobserv -n netobserv --ignore-not-found

.PHONY: helm-cleanup-all
helm-cleanup-all: helm-cleanup ## Uninstall the operator and its pre-requisites
	helm delete trust-manager -n cert-manager --ignore-not-found
	helm delete cert-manager -n cert-manager --ignore-not-found

.PHONY: helm-configure-flowcollector
helm-configure-flowcollector: ## Install FlowCollector, opinionated for small cluster (such as Kind) with minimal features enabled
	kubectl apply -f config/samples/flowcollectors/flowcollector-for-kind.yaml

.PHONY: helm-expose-console
helm-expose-console: prereqs-helm ## Expose the Web Console through port forwarding
	kubectl wait -n netobserv --timeout=60s --for condition=Available=True deployment netobserv-plugin
	@echo "🛰  READY! You can open http://localhost:9001/"
	kubectl port-forward svc/netobserv-plugin 9001:9001 -n netobserv
