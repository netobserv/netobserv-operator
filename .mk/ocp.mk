
ocp-deploy: BPF_VERSION=main
ocp-deploy: FLP_VERSION=main
ocp-deploy: PLG_VERSION=main
ocp-deploy: kustomize set-manager-images ## Deploy controller to the OpenShift cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/openshift | kubectl apply --server-side --force-conflicts -f -
	kubectl get ns openshift-netobserv-operator || kubectl create ns openshift-netobserv-operator
	cat bundles/openshift/manifests/netobserv-operator.clusterserviceversion.yaml \
		| $(SED) -E "s~operators.coreos.com/v1~operators.coreos.com/v1alpha1~" \
		| $(SED) -E "s/namespace: placeholder/namespace: openshift-netobserv-operator/" \
		| kubectl apply --server-side --force-conflicts -f -

ocp-undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/openshift | kubectl --ignore-not-found=true delete -f - || true
	kubectl --ignore-not-found=true delete ns openshift-netobserv-operator
