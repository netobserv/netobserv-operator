package cluster

import (
	"context"
	"fmt"

	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	configv1 "github.com/openshift/api/config/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// liveClient performs only live queries - no cache
type liveClient interface {
	getNodes(ctx context.Context) (*v1.NodeList, error)
	getKubeSystemDS(ctx context.Context) (*appsv1.DaemonSetList, error)
	getNetworkConfig(ctx context.Context) (*configv1.Network, error)
	getClusterVersion(ctx context.Context) (*configv1.ClusterVersion, error)
	getCRD(ctx context.Context, name string) (*apix.CustomResourceDefinition, error)
}

type liveClientImpl struct {
	kc kubernetes.Interface
	dc dynamic.Interface
}

func newLiveClient(c *rest.Config) (*liveClientImpl, error) {
	kc, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	dc, err := dynamic.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	return &liveClientImpl{kc: kc, dc: dc}, nil
}

func (lc *liveClientImpl) getNodes(ctx context.Context) (*v1.NodeList, error) {
	return lc.kc.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
}

func (lc *liveClientImpl) getKubeSystemDS(ctx context.Context) (*appsv1.DaemonSetList, error) {
	return lc.kc.AppsV1().DaemonSets(constants.KubeSystemNamespace).List(ctx, metav1.ListOptions{})
}

func (lc *liveClientImpl) getNetworkConfig(ctx context.Context) (*configv1.Network, error) {
	unst, err := lc.dc.Resource(configv1.GroupVersion.WithResource("networks")).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var obj configv1.Network
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unst.UnstructuredContent(), &obj); err != nil {
		return nil, fmt.Errorf("could not convert Network Config from unstructured: %w", err)
	}
	return &obj, nil
}

func (lc *liveClientImpl) getClusterVersion(ctx context.Context) (*configv1.ClusterVersion, error) {
	unst, err := lc.dc.Resource(configv1.GroupVersion.WithResource("clusterversions")).Get(ctx, "version", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var obj configv1.ClusterVersion
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unst.UnstructuredContent(), &obj); err != nil {
		return nil, fmt.Errorf("could not convert ClusterVersion from unstructured: %w", err)
	}
	return &obj, nil
}

func (lc *liveClientImpl) getCRD(ctx context.Context, name string) (*apix.CustomResourceDefinition, error) {
	unst, err := lc.dc.Resource(apix.SchemeGroupVersion.WithResource("customresourcedefinitions")).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var obj apix.CustomResourceDefinition
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unst.UnstructuredContent(), &obj); err != nil {
		return nil, fmt.Errorf("could not convert CRD from unstructured: %w", err)
	}
	return &obj, nil
}
