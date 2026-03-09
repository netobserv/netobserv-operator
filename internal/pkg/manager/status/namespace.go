package status

import (
	"context"
	"strings"

	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/netobserv-operator/internal/controller/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func annotation(cpnt ComponentName) string {
	return constants.AnnotationDomain + "/" + strings.ToLower(string(cpnt)) + "-namespace"
}

func GetDeployedNamespace(cpnt ComponentName, fc *flowslatest.FlowCollector) string {
	if ns, found := fc.Annotations[annotation(cpnt)]; found {
		return ns
	}
	//nolint:staticcheck // SA1019: Namespace is deprecated but must still be used until removed
	return fc.Status.Namespace
}

func SetDeployedNamespace(ctx context.Context, c client.Client, cpnt ComponentName, ns string) error {
	log := log.FromContext(ctx)
	annot := annotation(cpnt)
	log.WithValues(annot, ns).Info("Updating FlowCollector annotation")

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fc := flowslatest.FlowCollector{}
		if err := c.Get(ctx, constants.FlowCollectorName, &fc); err != nil {
			if errors.IsNotFound(err) {
				// ignore: when it's being deleted, there's no point trying to update its status
				return nil
			}
			return err
		}
		if fc.Annotations == nil {
			fc.Annotations = make(map[string]string)
		}
		fc.Annotations[annot] = ns
		return c.Update(ctx, &fc)
	})
}

func (i *Instance) GetDeployedNamespace(fc *flowslatest.FlowCollector) string {
	return GetDeployedNamespace(i.cpnt, fc)
}

func (i *Instance) SetDeployedNamespace(ctx context.Context, c client.Client, ns string) error {
	return SetDeployedNamespace(ctx, c, i.cpnt, ns)
}
