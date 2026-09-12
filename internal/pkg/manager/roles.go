package manager

// Operator fully manages netobserv-defined resources (cluster role)
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors;flowmetrics;flowcollectorslices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors/status;flowmetrics/status;flowcollectorslices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=flows.netobserv.io,resources=flowcollectors/finalizers,verbs=update

// Operator reads Network config for configured CIDRs (cluster-scope resource)
//+kubebuilder:rbac:groups=operator.openshift.io,resources=networks,verbs=get;list;watch

// Operator reads APIServer TLS profile, ClusterVersions for cluster info, and Network config for configured CIDRs (cluster-scope resources)
//+kubebuilder:rbac:groups=config.openshift.io,resources=apiservers;clusterversions;networks,verbs=get;list;watch

// Operator needs to create namespaces, services, service accounts, CM, PVC in a user-defined namespace
//+kubebuilder:rbac:groups=core,resources=namespaces;services;serviceaccounts;configmaps;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Operator reads Endpoint and EndpointSlices for APIServer IP (for netpol and subnet config), default namespace
//+kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch
//+kubebuilder:rbac:groups=discovery.k8s.io,resources=endpointslices,verbs=get;list;watch

// Operator reads pods and nodes info for cluster info & self health
//+kubebuilder:rbac:groups=core,resources=pods;nodes;endpoints,verbs=get;list;watch

// Operator fires events to signal degraded status (cluster-scope)
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Operator needs to create deployments, daemonsets in a user-defined namespace
// Also needed transitively for FLP (read)
//+kubebuilder:rbac:groups=apps,resources=deployments;daemonsets,verbs=get;list;watch;create;update;patch;delete

// Operator needs to create roles and cluster roles for granting transitive rights to its workloads
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;create;delete;update;watch

// Operator needs to patch Console CR (cluster scope)
//+kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;list;patch;watch

// Operator needs to create ConsolePlugin (cluster scope)
//+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;create;delete;update;patch;list;watch

// Operator needs to grant hostnetwork usage permission to eBPF Agent pods in a user-defined namespace
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=hostnetwork,verbs=use

// Operator needs to create SCC for its workloads in a user-defined namespace
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=create;list;watch
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,resourceNames=netobserv-ebpf-agent,verbs=update

// Operator needs to get API services for available API discovery (cluster scope)
//+kubebuilder:rbac:groups=apiregistration.k8s.io,resources=apiservices,verbs=list;get;watch

// Operator needs to create monitoring resources for its workloads in a user-defined namespace
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors;prometheusrules,verbs=get;create;delete;update;patch;list;watch

// Operator needs to read LokiStack status in a user-defined namespace
//+kubebuilder:rbac:groups=loki.grafana.com,resources=lokistacks,verbs=get;list;watch

// Operator needs to create network policies for its workloads in a user-defined namespace
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Operator needs to create ClusterBpfApplication (cluster-scope), and read its status
//+kubebuilder:rbac:groups=bpfman.io,resources=clusterbpfapplications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bpfman.io,resources=clusterbpfapplications/status,verbs=get;update;patch

// Operator needs to read CRDs for API discovery (cluster-scope resource)
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// Operator needs to update its CRD status for migration purpose (cluster-scope resource)
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions/status,verbs=update;patch

// (deprecated) Operator to create HPA for its workloads in a user-defined namespace
//+kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=create;delete;patch;update;get;watch;list

// Transitive: operator needs to grant UDN read permission to FLP at the cluster scope
//+kubebuilder:rbac:groups=k8s.ovn.org,resources=userdefinednetworks;clusteruserdefinednetworks,verbs=get;list;watch

// Transitive: operator needs to grant FRR read permission to FLP for BGP ASN enrichment
//+kubebuilder:rbac:groups=frrk8s.metallb.io,resources=frrconfigurations,verbs=get;list;watch
