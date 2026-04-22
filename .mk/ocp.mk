##@ OCP

.PHONY: ocp-expose-infra
ocp-expose-infra:
	oc expose service grafana || true
	@grafana_url=$$(oc get route grafana -o jsonpath='{.spec.host}'); \
	echo -e "\nAccess grafana on OCP using: http://"$$grafana_url"\n"
	oc expose service loki || true
	@loki_url=$$(oc get route loki -o jsonpath='{.spec.host}'); \
	echo -e "\nAccess loki on OCP using: http://"$$loki_url"\n"

.PHONY: ocp-expose-all
ocp-expose-all: ocp-expose-infra
	oc expose -n sample-workload service frontend-external || true
	@sample_workload_url=$$(oc get -n sample-workload route frontend-external -o jsonpath='{.spec.host}'); \
	echo -e "\nAccess sample workload on OCP using: http://"$$sample_workload_url"\n"

.PHONY: ocp-deploy-infra
ocp-deploy-infra: deploy-infra ocp-expose-infra ## OCP infra. deploy (only loki and grafana excluding the operator)

.PHONY: ocp-deploy
ocp-deploy: deploy-all ocp-expose-all ## OCP deploy (loki, grafana, example-cr and sample-workload excluding the operator)

.PHONY: ocp-undeploy
ocp-undeploy: undeploy-all  ## OCP cleanup

.PHONY: ocp-run
ocp-run: ocp-undeploy ocp-deploy ocp-deploy-operator  ## OCP-deploy + run the operator locally

.PHONY: ocp-deploy-operator
ocp-deploy-operator: ## run flp from the operator
	@echo "====> Enable netobserv-plugin in OCP console"
	oc patch console.operator.openshift.io cluster --type='json' -p '[{"op": "add", "path": "/spec/plugins", "value": ["netobserv-plugin"]}]'
	@echo "====> Running the operator locally"
	go run ./main.go \
		-ebpf-agent-image=quay.io/netobserv/netobserv-ebpf-agent:main \
		-flowlogs-pipeline-image=quay.io/netobserv/flowlogs-pipeline:main \
		-console-plugin-images=quay.io/netobserv/network-observability-console-plugin:main \
		-namespace=${NAMESPACE}

.PHONY: undeploy-operator
undeploy-operator: ## stop the operator locally
	-PID=$$(pgrep --oldest --full "main.go"); pkill -P $$PID; pkill $$PID
	kubectl delete service flowlogs-pipeline-prom || true
	kubectl delete ds flowlogs-pipeline || true
	kubectl delete servicemonitor flowlogs-pipeline-monitor || true
	kubectl delete service netobserv-plugin || true
	kubectl delete deployment netobserv-plugin || true
	kubectl delete servicemonitor netobserv-plugin || true
	kubectl delete configmap -n openshift-config-managed flowlogs-pipeline-metrics-dashboard || true
	kubectl delete configmap -n netobserv flowlogs-pipeline-config || true
	kubectl delete configmap -n netobserv console-plugin-config || true

.PHONY: ocp-refresh-ovs
ocp-refresh-ovs:
	@echo "====> Re-applying OVS configuration to speed-up templates sync"
	./hack/refresh-ovs.sh
