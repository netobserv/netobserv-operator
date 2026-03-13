package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/go-semver/semver"
	lokiv1 "github.com/grafana/loki/operator/apis/loki/v1"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	osv1 "github.com/openshift/api/console/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	securityv1 "github.com/openshift/api/security/v1"
	monv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// discoveryClient is an interface for API discovery operations
type discoveryClient interface {
	ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

type Info struct {
	apisMap                     map[APIName]bool
	apisMapLock                 sync.RWMutex
	id                          string
	openShiftVersion            *semver.Version
	cni                         flowslatest.NetworkType
	nbNodes                     uint16
	hasPromServiceDiscoveryRole bool
	ready                       bool
	readinessLock               sync.RWMutex
	dcl                         discoveryClient
	livecl                      liveClient
	onRefresh                   func()
}

type APIName string

var (
	ConsolePlugin  APIName = APIName("consoleplugins." + osv1.GroupVersion.String())
	CNO            APIName = APIName("networks." + operatorv1.GroupVersion.String())
	SvcMonitor     APIName = APIName("servicemonitors." + monv1.SchemeGroupVersion.String())
	PromRule       APIName = APIName("prometheusrules." + monv1.SchemeGroupVersion.String())
	OCPSecurity    APIName = APIName("securitycontextconstraints." + securityv1.SchemeGroupVersion.String())
	EndpointSlices APIName = APIName("endpointslices." + discoveryv1.SchemeGroupVersion.String())
	LokiStack      APIName = APIName("lokistacks." + lokiv1.GroupVersion.String())
)

func NewInfo(ctx context.Context, cfg *rest.Config, dcl *discovery.DiscoveryClient, onRefresh func()) (*Info, func(ctx context.Context) error, error) {
	info := Info{dcl: dcl, onRefresh: onRefresh}
	liveCl, err := newLiveClient(cfg)
	if err != nil {
		return nil, nil, err
	}
	info.livecl = liveCl
	if err := info.fetchAvailableAPIs(ctx); err != nil {
		return &info, nil, err
	}
	return &info, info.postCreate, nil
}

func (c *Info) fetchAvailableAPIs(ctx context.Context) error {
	return c.fetchAvailableAPIsInternal(ctx, false)
}

// fetchAvailableAPIsInternal discovers available APIs and optionally allows continuing despite critical API failures
// allowCriticalFailure should be true during refresh loops to allow recovery from transient API server issues
//
// API Discovery Policy:
// - APIs are marked as available when first discovered
// - Once discovered, APIs are never marked unavailable (to avoid transient discovery issues)
// - Operator restart is required to detect removed APIs (rare in practice)
func (c *Info) fetchAvailableAPIsInternal(ctx context.Context, allowCriticalFailure bool) error {
	log := log.FromContext(ctx)
	_, resources, err := c.dcl.ServerGroupsAndResources()
	// We may receive partial data along with an error
	var discErr *discovery.ErrGroupDiscoveryFailed
	hasDiscoveryError := errors.As(err, &discErr)

	// If we have a total failure (no resources at all), fail fast
	if err != nil && !hasDiscoveryError {
		return fmt.Errorf("API discovery failed completely: %w", err)
	}
	if len(resources) == 0 {
		return fmt.Errorf("API discovery returned no resources")
	}

	// Track which critical APIs failed discovery
	criticalAPIFailed := false
	apisRecovered := false
	firstRun := false
	c.apisMapLock.Lock()
	defer c.apisMapLock.Unlock()
	if c.apisMap == nil {
		c.apisMap = map[APIName]bool{
			ConsolePlugin:  false,
			CNO:            false,
			SvcMonitor:     false,
			PromRule:       false,
			OCPSecurity:    false,
			EndpointSlices: false,
			LokiStack:      false,
		}
		firstRun = true
	}
	for apiName, discovered := range c.apisMap {
		// Never remove a discovered API, to avoid transient staleness issues triggering changes continuously
		if !discovered {
			if hasAPI(apiName, resources) {
				c.apisMap[apiName] = true
				if !firstRun {
					apisRecovered = true
					log.Info("API recovered and is now available", "api", apiName)
				}
			} else if hasDiscoveryError {
				// Check if the wanted API is in error
				for gv, gvErr := range discErr.Groups {
					if strings.Contains(string(apiName), gv.String()) {
						log.Error(gvErr, "some API-related features are unavailable; you can check for stale APIs with 'kubectl get apiservice'", "GroupVersion", gv.String(), "api", apiName)
						// OCP Security API is critical - we MUST know if we're on OpenShift
						// to avoid wrong security context configurations
						if apiName == OCPSecurity {
							criticalAPIFailed = true
						}
					}
				}
			}
		}
	}
	if firstRun {
		log.Info("API detection finished", "apis", c.apisMap)
	}

	// If APIs recovered, trigger reconciliation via the onRefresh callback
	// The callback runs in a goroutine to avoid blocking while holding the lock
	if apisRecovered && c.onRefresh != nil {
		log.Info("Triggering reconciliation due to API recovery")
		go c.onRefresh()
	}

	// If critical API discovery failed:
	// - During startup (allowCriticalFailure=false): fail fast to prevent wrong cluster detection
	// - During refresh (allowCriticalFailure=true): log error but continue, allowing time to recover
	if criticalAPIFailed && !allowCriticalFailure {
		return fmt.Errorf("critical API discovery failed: cannot determine if running on OpenShift (security.openshift.io API unavailable)")
	}

	return nil
}

func hasAPI(apiName APIName, resources []*metav1.APIResourceList) bool {
	for i := range resources {
		for j := range resources[i].APIResources {
			gvk := resources[i].APIResources[j].Name + "." + resources[i].GroupVersion
			if string(apiName) == gvk {
				return true
			}
		}
	}
	return false
}

func (c *Info) postCreate(ctx context.Context) error {
	if err := c.fetchClusterInfo(ctx); err != nil {
		return err
	}
	c.startRefreshLoop(ctx)
	return nil
}

func (c *Info) fetchClusterInfo(ctx context.Context) error {
	var id string
	var openShiftVersion *semver.Version
	var cni flowslatest.NetworkType
	var nbNodes uint16
	var hasPromServiceDiscoveryRole bool
	if c.IsOpenShift() {
		// Fetch cluster ID, version and CNI
		cversion, err := c.livecl.getClusterVersion(ctx)
		if err != nil {
			return fmt.Errorf("could not fetch ClusterVersion: %w", err)
		}
		id = string(cversion.Spec.ClusterID)
		// Get version; use the same method as via `oc get clusterversion`, where printed column uses jsonPath:
		// .status.history[?(@.state=="Completed")].version
		for _, history := range cversion.Status.History {
			if history.State == "Completed" {
				openShiftVersion = semver.New(history.Version)
				break
			}
		}
		network, err := c.livecl.getNetworkConfig(ctx)
		if err != nil {
			return fmt.Errorf("could not fetch Network resource: %w", err)
		}
		cni = flowslatest.NetworkType(network.Spec.NetworkType)
	}
	if c.HasSvcMonitor() && c.HasEndpointSlices() {
		// Check whether servicemonitor spec.serviceDiscoveryRole exists
		crd, err := c.livecl.getCRD(ctx, "servicemonitors.monitoring.coreos.com")
		if err != nil {
			return fmt.Errorf("could not check for ServiceMonitor serviceDiscoveryRole presence: %w", err)
		}
		hasPromServiceDiscoveryRole = hasCRDProperty(ctx, crd, "v1", "spec.serviceDiscoveryRole")
	}

	l, err := c.livecl.getNodes(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve number of nodes: %w", err)
	}
	nbNodes = uint16(len(l.Items))
	if cni == "" {
		cni = guessCNIFromNodes(l.Items)
		if cni == "" {
			ds, err := c.livecl.getKubeSystemDS(ctx)
			if err != nil {
				return fmt.Errorf("could not retrieve kube-system daemon sets: %w", err)
			}
			cni = guessCNIFromSystemDS(ds.Items)
		}
	}
	c.setInfo(id, openShiftVersion, cni, nbNodes, hasPromServiceDiscoveryRole)
	log.FromContext(ctx).Info("Cluster info fetched",
		"id", id,
		"openShiftVersion", openShiftVersion,
		"cni", cni,
		"nbNodes", nbNodes,
		"hasPromServiceDiscoveryRole", hasPromServiceDiscoveryRole,
	)

	return nil
}

func guessCNIFromNodes(nodes []v1.Node) flowslatest.NetworkType {
	for i := range nodes {
		if annots := nodes[i].Annotations; annots != nil {
			if cfg, ok := annots["k8s.ovn.org/host-cidrs"]; ok && len(cfg) > 0 {
				return flowslatest.OVNKubernetes
			}
		}
	}
	return ""
}

func guessCNIFromSystemDS(ds []appsv1.DaemonSet) flowslatest.NetworkType {
	for i := range ds {
		if ds[i].Name == "kindnet" {
			return flowslatest.Kindnet
		}
	}
	return ""
}

func (c *Info) setInfo(id string, openShiftVersion *semver.Version, cni flowslatest.NetworkType, nbNodes uint16, hasPromServiceDiscoveryRole bool) {
	c.readinessLock.Lock()
	defer c.readinessLock.Unlock()
	c.id = id
	c.openShiftVersion = openShiftVersion
	c.cni = cni
	c.nbNodes = nbNodes
	c.hasPromServiceDiscoveryRole = hasPromServiceDiscoveryRole
	c.ready = true
}

// Mock shouldn't be used except for testing
func (c *Info) Mock(v string, cni flowslatest.NetworkType, apis ...APIName) {
	if c.apisMap == nil {
		c.apisMap = make(map[APIName]bool)
	}
	if v == "" {
		// No OpenShift
		c.apisMap[OCPSecurity] = false
		c.openShiftVersion = nil
	} else {
		c.apisMap[OCPSecurity] = true
		c.openShiftVersion = semver.New(v)
	}
	for _, api := range apis {
		c.apisMap[api] = true
	}
	c.cni = cni
	c.ready = true
}

func (c *Info) GetID() string {
	c.readinessLock.RLock()
	defer c.readinessLock.RUnlock()
	return c.id
}

func (c *Info) GetOpenShiftVersion() (string, error) {
	c.readinessLock.RLock()
	defer c.readinessLock.RUnlock()
	if !c.ready {
		return "", errors.New("cluster info not collected")
	}
	if c.openShiftVersion == nil {
		return "", errors.New("unknown OpenShift version")
	}
	return c.openShiftVersion.String(), nil
}

func (c *Info) GetCNI() (flowslatest.NetworkType, error) {
	c.readinessLock.RLock()
	defer c.readinessLock.RUnlock()
	if !c.ready {
		return "", errors.New("cluster info not collected")
	}
	return c.cni, nil
}

func (c *Info) GetNbNodes() (uint16, error) {
	c.readinessLock.RLock()
	defer c.readinessLock.RUnlock()
	if !c.ready {
		return 0, errors.New("cluster info not collected")
	}
	return c.nbNodes, nil
}

func (c *Info) HasPromServiceDiscoveryRole() bool {
	return c.hasPromServiceDiscoveryRole
}

func (c *Info) IsOpenShiftVersionLessThan(v string) (bool, string, error) {
	c.readinessLock.RLock()
	defer c.readinessLock.RUnlock()
	if !c.ready {
		return false, "", errors.New("cluster info not collected")
	}
	if c.openShiftVersion == nil {
		return false, "", errors.New("unknown OpenShift version, cannot compare versions")
	}
	version, err := semver.NewVersion(v)
	if err != nil {
		return false, "", err
	}
	openshiftVersion := *c.openShiftVersion
	// Ignore pre-release block for comparison
	openshiftVersion.PreRelease = ""
	return openshiftVersion.LessThan(*version), c.openShiftVersion.String(), nil
}

func (c *Info) IsOpenShiftVersionAtLeast(v string) (bool, string, error) {
	b, v, err := c.IsOpenShiftVersionLessThan(v)
	return !b, v, err
}

// IsOpenShift assumes having openshift SCC API <=> being on openshift
func (c *Info) IsOpenShift() bool {
	return c.HasOCPSecurity()
}

// HasConsolePlugin returns true if "consoleplugins.console.openshift.io" API was found
func (c *Info) HasConsolePlugin() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[ConsolePlugin]
}

// HasOCPSecurity returns true if "consoles.config.openshift.io" API was found
func (c *Info) HasOCPSecurity() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[OCPSecurity]
}

// HasCNO returns true if "networks.operator.openshift.io" API was found
func (c *Info) HasCNO() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[CNO]
}

// HasSvcMonitor returns true if "servicemonitors.monitoring.coreos.com" API was found
func (c *Info) HasSvcMonitor() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[SvcMonitor]
}

// HasPromRule returns true if "prometheusrules.monitoring.coreos.com" API was found
func (c *Info) HasPromRule() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[PromRule]
}

func (c *Info) HasEndpointSlices() bool {
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[EndpointSlices]
}

// hasCRDProperty returns property presence for any CRD, given a dot-separated path such as "spec.foo.bar"
// version is the CRD version; leave empty to check all versions
func hasCRDProperty(ctx context.Context, crd *apix.CustomResourceDefinition, version, path string) bool {
	log := log.FromContext(ctx)
	parts := strings.Split(path, ".")
	for i := range crd.Spec.Versions {
		v := &crd.Spec.Versions[i]
		if version != "" && version != v.Name {
			continue
		}
		if found := getCRDPropertyInVersion(v, parts); found != nil {
			log.Info("CRD property found", "path", path)
			return true
		}
	}
	log.Info("CRD property not found", "path", path)
	return false
}

func getCRDPropertyInVersion(v *apix.CustomResourceDefinitionVersion, parts []string) *apix.JSONSchemaProps {
	if v.Schema != nil && v.Schema.OpenAPIV3Schema != nil {
		props := v.Schema.OpenAPIV3Schema.Properties
		var next apix.JSONSchemaProps
		for _, search := range parts {
			next, ok := props[search]
			if !ok {
				return nil
			}
			props = next.Properties
		}
		return &next
	}
	return nil
}

// HasLokiStack returns true if "lokistack" API was found
func (c *Info) HasLokiStack(ctx context.Context) bool {
	if !c.apisMap[LokiStack] {
		err := c.fetchAvailableAPIsInternal(ctx, true)
		if err != nil {
			return false
		}
	}
	c.apisMapLock.RLock()
	defer c.apisMapLock.RUnlock()
	return c.apisMap[LokiStack]
}
