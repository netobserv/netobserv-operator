package slicesstatus

import (
	"context"

	sliceslatest "github.com/netobserv/netobserv-operator/api/flowcollectorslice/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ConditionReady         = "Ready"
	ConditionSubnetWarning = "SubnetWarning"
)

var (
	mapStatuses       map[types.NamespacedName]*metav1.Condition = make(map[types.NamespacedName]*metav1.Condition)
	mapSubnetWarnings map[types.NamespacedName]*metav1.Condition = make(map[types.NamespacedName]*metav1.Condition)
)

func Reset(fcs *sliceslatest.FlowCollectorSliceList) {
	mapStatuses = make(map[types.NamespacedName]*metav1.Condition)
	mapSubnetWarnings = make(map[types.NamespacedName]*metav1.Condition)
	for i := range fcs.Items {
		fcs.Items[i].Status.FilterApplied = ""
		fcs.Items[i].Status.SubnetLabelsConfigured = 0
	}
}

func SetReady(fcs *sliceslatest.FlowCollectorSlice) {
	nsname := types.NamespacedName{Name: fcs.Name, Namespace: fcs.Namespace}
	mapStatuses[nsname] = &metav1.Condition{
		Type:    ConditionReady,
		Reason:  "Ready",
		Message: "flowlogs-pipeline configured",
		Status:  metav1.ConditionTrue,
	}
}

func SetFailure(fcs *sliceslatest.FlowCollectorSlice, msg string) {
	nsname := types.NamespacedName{Name: fcs.Name, Namespace: fcs.Namespace}
	mapStatuses[nsname] = &metav1.Condition{
		Type:    ConditionReady,
		Reason:  "Failure",
		Message: msg,
		Status:  metav1.ConditionFalse,
	}
}

func AddSubnetWarning(fcs *sliceslatest.FlowCollectorSlice, msg string) {
	nsname := types.NamespacedName{Name: fcs.Name, Namespace: fcs.Namespace}
	if existing := mapSubnetWarnings[nsname]; existing != nil {
		// Limit number of reported warnings
		if len(existing.Message) < 500 {
			existing.Message += "; " + msg
		}
	} else {
		mapSubnetWarnings[nsname] = &metav1.Condition{
			Type:    ConditionSubnetWarning,
			Reason:  "SubnetOverlap",
			Message: msg,
			Status:  metav1.ConditionTrue,
		}
	}
}

func Sync(ctx context.Context, c client.Client, fcs *sliceslatest.FlowCollectorSliceList) {
	log := log.FromContext(ctx)
	log.Info("Syncing FlowCollectorSlices status")
	for i := range fcs.Items {
		nsname := types.NamespacedName{Name: fcs.Items[i].Name, Namespace: fcs.Items[i].Namespace}
		// main condition is mandatory; subnet warning condition is optional
		if cond, ok := mapStatuses[nsname]; ok {
			subnetCond := mapSubnetWarnings[nsname]
			setStatus(ctx, c, nsname, func(s *sliceslatest.FlowCollectorSliceStatus) {
				if cond != nil {
					meta.SetStatusCondition(&s.Conditions, *cond)
				}
				if subnetCond != nil {
					meta.SetStatusCondition(&s.Conditions, *subnetCond)
				} else {
					meta.RemoveStatusCondition(&s.Conditions, ConditionSubnetWarning)
				}
				s.FilterApplied = fcs.Items[i].Status.FilterApplied
				s.SubnetLabelsConfigured = fcs.Items[i].Status.SubnetLabelsConfigured
			})
		}
	}
}

func GetReadyCondition(slice *sliceslatest.FlowCollectorSlice) *metav1.Condition {
	nsname := types.NamespacedName{Name: slice.Name, Namespace: slice.Namespace}
	return mapStatuses[nsname]
}

func GetSubnetWarningCondition(slice *sliceslatest.FlowCollectorSlice) *metav1.Condition {
	nsname := types.NamespacedName{Name: slice.Name, Namespace: slice.Namespace}
	return mapSubnetWarnings[nsname]
}

func setStatus(ctx context.Context, c client.Client, nsname types.NamespacedName, applyStatus func(s *sliceslatest.FlowCollectorSliceStatus)) {
	log := log.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fcs := sliceslatest.FlowCollectorSlice{}
		if err := c.Get(ctx, nsname, &fcs); err != nil {
			log.WithValues("NsName", nsname).Error(err, "failed to get FlowCollectorSlices status")
			if errors.IsNotFound(err) {
				// ignore: when it's being deleted, there's no point trying to update its status
				return nil
			}
			return err
		}
		applyStatus(&fcs.Status)
		return c.Status().Update(ctx, &fcs)
	})

	if err != nil {
		log.Error(err, "failed to update FlowCollectorSlices status")
	}
}
