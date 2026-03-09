package cluster

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/coreos/go-semver/semver"
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
)

func TestIsOpenShiftVersionLessThan(t *testing.T) {
	info := Info{openShiftVersion: semver.New("4.14.9"), ready: true}
	b, _, err := info.IsOpenShiftVersionLessThan("4.15.0")
	assert.NoError(t, err)
	assert.True(t, b)

	info.openShiftVersion = semver.New("4.15.0")
	b, _, err = info.IsOpenShiftVersionLessThan("4.15.0")
	assert.NoError(t, err)
	assert.False(t, b)
}

func TestIsOpenShiftVersionAtLeast(t *testing.T) {
	info := Info{openShiftVersion: semver.New("4.14.9"), ready: true}
	b, _, err := info.IsOpenShiftVersionAtLeast("4.14.0")
	assert.NoError(t, err)
	assert.True(t, b)

	info.openShiftVersion = semver.New("4.14.0")
	b, _, err = info.IsOpenShiftVersionAtLeast("4.14.0")
	assert.NoError(t, err)
	assert.True(t, b)

	info.openShiftVersion = semver.New("4.12.0")
	b, _, err = info.IsOpenShiftVersionAtLeast("4.14.0")
	assert.NoError(t, err)
	assert.False(t, b)
}

func TestGetCRDProperty(t *testing.T) {
	crd := apix.CustomResourceDefinition{
		Spec: apix.CustomResourceDefinitionSpec{
			Versions: []apix.CustomResourceDefinitionVersion{
				{
					Name: "v1alpha1",
					Schema: &apix.CustomResourceValidation{
						OpenAPIV3Schema: &apix.JSONSchemaProps{
							Properties: map[string]apix.JSONSchemaProps{
								"spec": {},
							},
						},
					},
				},
				{
					Name: "v1",
					Schema: &apix.CustomResourceValidation{
						OpenAPIV3Schema: &apix.JSONSchemaProps{
							Properties: map[string]apix.JSONSchemaProps{
								"spec": {
									Properties: map[string]apix.JSONSchemaProps{
										"foo": {
											Properties: map[string]apix.JSONSchemaProps{
												"bar": {},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	assert.True(t, hasCRDProperty(context.Background(), &crd, "", "spec.foo.bar"))
	assert.False(t, hasCRDProperty(context.Background(), &crd, "v1alpha1", "spec.foo.bar"))
	assert.True(t, hasCRDProperty(context.Background(), &crd, "v1", "spec.foo.bar"))
	assert.False(t, hasCRDProperty(context.Background(), &crd, "", "spec.foo.bar.baz"))
	assert.False(t, hasCRDProperty(context.Background(), &crd, "v2", "spec"))
}

// Mock discovery client for testing
type mockDiscoveryClient struct {
	resources []*metav1.APIResourceList
	err       error
}

func (m *mockDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return nil, m.resources, m.err
}

// Helper to create API resource lists
func makeAPIResourceList(groupVersion string, resources ...string) *metav1.APIResourceList {
	apiList := &metav1.APIResourceList{
		GroupVersion: groupVersion,
		APIResources: []metav1.APIResource{},
	}
	for _, res := range resources {
		apiList.APIResources = append(apiList.APIResources, metav1.APIResource{
			Name: res,
		})
	}
	return apiList
}

// TestFetchAvailableAPIs_Success tests successful API discovery on OpenShift
func TestFetchAvailableAPIs_Success(t *testing.T) {
	info := &Info{}

	// Mock all required APIs being available
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("console.openshift.io/v1", "consoleplugins"),
			makeAPIResourceList("operator.openshift.io/v1", "networks"),
			makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors", "prometheusrules"),
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
			makeAPIResourceList("discovery.k8s.io/v1", "endpointslices"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	assert.NoError(t, err)
	assert.True(t, info.IsOpenShift(), "Should detect OpenShift")
	assert.True(t, info.HasConsolePlugin())
	assert.True(t, info.HasCNO())
	assert.True(t, info.HasSvcMonitor())
	assert.True(t, info.HasPromRule())
	assert.True(t, info.HasEndpointSlices())
}

// TestFetchAvailableAPIs_NonOpenShift tests API discovery on vanilla Kubernetes
func TestFetchAvailableAPIs_NonOpenShift(t *testing.T) {
	info := &Info{}

	// Mock Kubernetes without OpenShift APIs
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors", "prometheusrules"),
			makeAPIResourceList("discovery.k8s.io/v1", "endpointslices"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	assert.NoError(t, err)
	assert.False(t, info.IsOpenShift(), "Should not detect OpenShift")
	assert.False(t, info.HasConsolePlugin())
	assert.False(t, info.HasCNO())
	assert.True(t, info.HasSvcMonitor())
	assert.True(t, info.HasPromRule())
	assert.True(t, info.HasEndpointSlices())
}

// TestFetchAvailableAPIs_CriticalAPIFailed tests the bug fix:
// When OpenShift SCC API discovery fails, the operator should fail fast
func TestFetchAvailableAPIs_CriticalAPIFailed(t *testing.T) {
	info := &Info{}

	// Mock partial discovery failure where security.openshift.io API fails
	// This simulates the exact bug scenario from the incident
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("console.openshift.io/v1", "consoleplugins"),
			makeAPIResourceList("operator.openshift.io/v1", "networks"),
			makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors"),
			// security.openshift.io is missing due to discovery failure
		},
		err: &discovery.ErrGroupDiscoveryFailed{
			Groups: map[schema.GroupVersion]error{
				{Group: "security.openshift.io", Version: "v1"}: fmt.Errorf("the server was unable to return a response in the time allotted"),
			},
		},
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	// This is the fix: should return error instead of silently continuing
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "critical API discovery failed")
	assert.Contains(t, err.Error(), "OpenShift")
}

// TestFetchAvailableAPIs_NonCriticalAPIFailed tests that non-critical API failures are tolerated
func TestFetchAvailableAPIs_NonCriticalAPIFailed(t *testing.T) {
	info := &Info{}

	// Mock partial discovery failure where non-critical API fails
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("console.openshift.io/v1", "consoleplugins"),
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
			// monitoring.coreos.com is missing but that's not critical
		},
		err: &discovery.ErrGroupDiscoveryFailed{
			Groups: map[schema.GroupVersion]error{
				{Group: "monitoring.coreos.com", Version: "v1"}: fmt.Errorf("api service unavailable"),
			},
		},
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	// Non-critical failures should be tolerated
	assert.NoError(t, err)
	assert.True(t, info.IsOpenShift(), "Should still detect OpenShift")
	assert.False(t, info.HasSvcMonitor(), "Should mark failed API as unavailable")
}

// TestFetchAvailableAPIs_TotalFailure tests complete API discovery failure with no resources
func TestFetchAvailableAPIs_TotalFailure(t *testing.T) {
	info := &Info{}

	// Mock total failure with no resources but partial error
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{},
		err: &discovery.ErrGroupDiscoveryFailed{
			Groups: map[schema.GroupVersion]error{
				{Group: "example.com", Version: "v1"}: fmt.Errorf("connection refused"),
			},
		},
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no resources")
}

// TestFetchAvailableAPIs_CompleteFailure tests when discovery client returns hard error
func TestFetchAvailableAPIs_CompleteFailure(t *testing.T) {
	info := &Info{}

	// Mock hard failure (not ErrGroupDiscoveryFailed)
	mockDcl := &mockDiscoveryClient{
		resources: nil,
		err:       fmt.Errorf("network timeout"),
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API discovery failed completely")
}

// TestHasAPI tests the hasAPI helper function
func TestHasAPI(t *testing.T) {
	resources := []*metav1.APIResourceList{
		makeAPIResourceList("console.openshift.io/v1", "consoleplugins"),
		makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
	}

	assert.True(t, hasAPI("consoleplugins.console.openshift.io/v1", resources))
	assert.True(t, hasAPI("securitycontextconstraints.security.openshift.io/v1", resources))
	assert.False(t, hasAPI("notfound.example.com/v1", resources))
}

// TestIsOpenShift tests the IsOpenShift method
func TestIsOpenShift(t *testing.T) {
	// Test OpenShift detection
	info := &Info{
		apisMap: map[string]bool{
			ocpSecurity: true,
		},
	}
	assert.True(t, info.IsOpenShift())

	// Test non-OpenShift
	info.apisMap[ocpSecurity] = false
	assert.False(t, info.IsOpenShift())
}

// TestAPIDetectionRaceCondition tests thread-safety of API detection
func TestAPIDetectionRaceCondition(t *testing.T) {
	info := &Info{}
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	// Run API detection and concurrent IsOpenShift checks
	done := make(chan bool)
	go func() {
		err := info.fetchAvailableAPIs(context.Background())
		assert.NoError(t, err)
		done <- true
	}()

	// Should not panic due to race condition
	for i := 0; i < 100; i++ {
		_ = info.IsOpenShift()
	}

	<-done
	assert.True(t, info.IsOpenShift())
}

// TestFetchAvailableAPIs_CriticalFailureWithFlag tests that allowCriticalFailure flag works
func TestFetchAvailableAPIs_CriticalFailureWithFlag(t *testing.T) {
	info := &Info{}

	// Mock critical API failure (SCC API unavailable)
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("console.openshift.io/v1", "consoleplugins"),
			// security.openshift.io is missing
		},
		err: &discovery.ErrGroupDiscoveryFailed{
			Groups: map[schema.GroupVersion]error{
				{Group: "security.openshift.io", Version: "v1"}: fmt.Errorf("api service unavailable"),
			},
		},
	}
	info.dcl = mockDcl

	// During startup (allowCriticalFailure=false), should fail
	err := info.fetchAvailableAPIsInternal(context.Background(), false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "critical API discovery failed")

	// During refresh (allowCriticalFailure=true), should NOT fail
	err = info.fetchAvailableAPIsInternal(context.Background(), true)
	assert.NoError(t, err, "Should allow critical failure during refresh")
	assert.False(t, info.IsOpenShift(), "Should mark OpenShift as unavailable")
}

// TestAPIRecovery tests that API recovery is detected and triggers callback
func TestAPIRecovery(t *testing.T) {
	callbackTriggered := false
	info := &Info{
		onRefresh: func() {
			callbackTriggered = true
		},
	}

	// Initially, ServiceMonitor API is unavailable
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
			// monitoring.coreos.com is missing
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())
	assert.NoError(t, err)
	assert.True(t, info.IsOpenShift())
	assert.False(t, info.HasSvcMonitor(), "ServiceMonitor should be unavailable")
	assert.False(t, callbackTriggered, "Callback should not trigger on initial detection")

	// Now ServiceMonitor API becomes available
	mockDcl.resources = []*metav1.APIResourceList{
		makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors"),
	}

	// Simulate refresh
	err = info.fetchAvailableAPIsInternal(context.Background(), true)
	assert.NoError(t, err)
	assert.True(t, info.HasSvcMonitor(), "ServiceMonitor should now be available")

	// Give the goroutine time to trigger the callback
	time.Sleep(100 * time.Millisecond)
	assert.True(t, callbackTriggered, "Callback should be triggered when API recovers")
}

// TestAPIRecovery_MultipleAPIs tests recovery of multiple APIs
func TestAPIRecovery_MultipleAPIs(t *testing.T) {
	callbackTriggered := false
	info := &Info{
		onRefresh: func() {
			callbackTriggered = true
		},
	}

	// Initially, both ServiceMonitor and PrometheusRule APIs are unavailable
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())
	assert.NoError(t, err)
	assert.False(t, info.HasSvcMonitor())
	assert.False(t, info.HasPromRule())

	// Now both APIs become available
	mockDcl.resources = []*metav1.APIResourceList{
		makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors"),
		makeAPIResourceList("monitoring.coreos.com/v1", "prometheusrules"),
	}

	err = info.fetchAvailableAPIsInternal(context.Background(), true)
	assert.NoError(t, err)
	assert.True(t, info.HasSvcMonitor())
	assert.True(t, info.HasPromRule())

	// Give the goroutine time to trigger the callback
	time.Sleep(100 * time.Millisecond)
	assert.True(t, callbackTriggered, "Callback should be triggered when APIs recover")
}

// TestNoRecovery_StillUnavailable tests that callback is not triggered if APIs remain unavailable
func TestNoRecovery_StillUnavailable(t *testing.T) {
	callbackTriggered := false
	info := &Info{
		onRefresh: func() {
			callbackTriggered = true
		},
	}

	// ServiceMonitor API is unavailable
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIs(context.Background())
	assert.NoError(t, err)
	assert.False(t, info.HasSvcMonitor())

	// API is still unavailable on refresh
	err = info.fetchAvailableAPIsInternal(context.Background(), true)
	assert.NoError(t, err)
	assert.False(t, info.HasSvcMonitor())

	// Give time for potential callback
	time.Sleep(100 * time.Millisecond)
	assert.False(t, callbackTriggered, "Callback should not trigger if API remains unavailable")
}

// TestRefreshWithError tests that errors during refresh are handled and don't crash
func TestRefreshWithError(t *testing.T) {
	info := &Info{
		onRefresh: func() {},
	}

	// Mock total API failure during refresh
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{},
		err:       fmt.Errorf("network timeout"),
	}
	info.dcl = mockDcl

	// During refresh, even total failures should be handled gracefully
	err := info.refresh(context.Background())
	assert.Error(t, err, "Should return error on API failure")
	assert.Contains(t, err.Error(), "API discovery")
}

// TestRefreshSuccess tests successful refresh operation
func TestRefreshSuccess(t *testing.T) {
	info := &Info{
		onRefresh: func() {},
	}

	// Mock successful API discovery
	mockDcl := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("security.openshift.io/v1", "securitycontextconstraints"),
		},
		err: nil,
	}
	info.dcl = mockDcl

	err := info.fetchAvailableAPIsInternal(context.Background(), true)
	assert.NoError(t, err)
	assert.True(t, info.IsOpenShift())
}

func TestHasLokiStack(t *testing.T) {
	ctx := context.Background()

	// Test 1: LokiStack CRD is available
	infoWithLokiStack := &Info{}
	mockDclWithLoki := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("loki.grafana.com/v1", "lokistacks"),
		},
		err: nil,
	}
	infoWithLokiStack.dcl = mockDclWithLoki
	assert.True(t, infoWithLokiStack.HasLokiStack(ctx))

	// Test 2: LokiStack CRD is not available
	infoWithoutLokiStack := &Info{}
	mockDclWithoutLoki := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{
			makeAPIResourceList("monitoring.coreos.com/v1", "servicemonitors"),
		},
		err: nil,
	}
	infoWithoutLokiStack.dcl = mockDclWithoutLoki
	assert.False(t, infoWithoutLokiStack.HasLokiStack(ctx))

	// Test 3: Empty apisMap
	infoEmpty := &Info{}
	mockDclEmpty := &mockDiscoveryClient{
		resources: []*metav1.APIResourceList{},
		err:       nil,
	}
	infoEmpty.dcl = mockDclEmpty
	assert.False(t, infoEmpty.HasLokiStack(ctx))
}

type mockLiveClient struct {
	nodes        []v1.Node
	ksDaemonSets []appsv1.DaemonSet
	network      *configv1.Network
	cv           *configv1.ClusterVersion
	crds         map[string]*apix.CustomResourceDefinition
}

func (m *mockLiveClient) getNodes(_ context.Context) (*v1.NodeList, error) {
	return &v1.NodeList{Items: m.nodes}, nil
}

func (m *mockLiveClient) getKubeSystemDS(_ context.Context) (*appsv1.DaemonSetList, error) {
	return &appsv1.DaemonSetList{Items: m.ksDaemonSets}, nil
}

func (m *mockLiveClient) getNetworkConfig(_ context.Context) (*configv1.Network, error) {
	if m.network != nil {
		return m.network, nil
	}
	return nil, errors.New("Network not found")
}

func (m *mockLiveClient) getClusterVersion(_ context.Context) (*configv1.ClusterVersion, error) {
	if m.cv != nil {
		return m.cv, nil
	}
	return nil, errors.New("ClusterVersion not found")
}

func (m *mockLiveClient) getCRD(_ context.Context, name string) (*apix.CustomResourceDefinition, error) {
	if crd, found := m.crds[name]; found {
		return crd, nil
	}
	return nil, fmt.Errorf("CRD %s not found", name)
}

func stubOpenShiftInfo(version string) (*Info, *configv1.ClusterVersion) {
	return &Info{
			apisMap: map[string]bool{
				ocpSecurity: true,
			},
		},
		&configv1.ClusterVersion{
			Spec: configv1.ClusterVersionSpec{
				ClusterID: "abc",
			},
			Status: configv1.ClusterVersionStatus{
				History: []configv1.UpdateHistory{
					{
						State:   "Completed",
						Version: version,
					},
				},
			},
		}
}

func TestFetchClusterInfo_CNI_OpenShift_OVN(t *testing.T) {
	info, cv := stubOpenShiftInfo("4.21.1")
	info.livecl = &mockLiveClient{
		cv: cv,
		network: &configv1.Network{
			Spec: configv1.NetworkSpec{NetworkType: string(flowslatest.OVNKubernetes)},
		},
	}

	err := info.fetchClusterInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, flowslatest.OVNKubernetes, info.cni)
}

func TestFetchClusterInfo_CNI_OpenShift_SDN(t *testing.T) {
	info, cv := stubOpenShiftInfo("4.21.1")
	info.livecl = &mockLiveClient{
		cv: cv,
		network: &configv1.Network{
			Spec: configv1.NetworkSpec{NetworkType: string(flowslatest.OpenShiftSDN)},
		},
	}

	err := info.fetchClusterInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, flowslatest.OpenShiftSDN, info.cni)
}

func TestFetchClusterInfo_CNI_OVN_Upstream(t *testing.T) {
	info := Info{}
	info.livecl = &mockLiveClient{
		nodes: []v1.Node{{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"k8s.ovn.org/host-cidrs": "something",
				},
			},
		}},
	}

	err := info.fetchClusterInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, flowslatest.OVNKubernetes, info.cni)
}

func TestFetchClusterInfo_CNI_Kindnet(t *testing.T) {
	info := Info{}
	info.livecl = &mockLiveClient{
		ksDaemonSets: []appsv1.DaemonSet{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kindnet",
			},
		}},
	}

	err := info.fetchClusterInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, flowslatest.Kindnet, info.cni)
}
