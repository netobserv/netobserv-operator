package fmstatus

import (
	"context"

	metricslatest "github.com/netobserv/netobserv-operator/api/flowmetrics/v1alpha1"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper/cardinality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	ConditionReady              = "Ready"
	ConditionCardinalityWarning = "CardinalityWarning"
)

var mapStatuses map[types.NamespacedName]*metav1.Condition
var mapCards map[types.NamespacedName]*metav1.Condition

func Reset() {
	mapStatuses = make(map[types.NamespacedName]*metav1.Condition)
	mapCards = make(map[types.NamespacedName]*metav1.Condition)
}

func SetReady(fm *metricslatest.FlowMetric) {
	nsname := types.NamespacedName{Name: fm.Name, Namespace: fm.Namespace}
	mapStatuses[nsname] = &metav1.Condition{
		Type:    ConditionReady,
		Reason:  "Ready",
		Message: "flowlogs-pipeline configured",
		Status:  metav1.ConditionTrue,
	}
}

func SetFailure(fm *metricslatest.FlowMetric, msg string) {
	nsname := types.NamespacedName{Name: fm.Name, Namespace: fm.Namespace}
	mapStatuses[nsname] = &metav1.Condition{
		Type:    ConditionReady,
		Reason:  "Failure",
		Message: msg,
		Status:  metav1.ConditionFalse,
	}
}

func CheckCardinality(fm *metricslatest.FlowMetric) {
	report, err := cardinality.CheckCardinality(fm.Spec.Labels...)
	if err != nil {
		SetFailure(fm, err.Error())
		return
	}
	overall := report.GetOverall()
	status := metav1.ConditionFalse
	if overall == cardinality.WarnAvoid || overall == cardinality.WarnUnknown {
		status = metav1.ConditionTrue
	}
	nsname := types.NamespacedName{Name: fm.Name, Namespace: fm.Namespace}
	mapCards[nsname] = &metav1.Condition{
		Type:    ConditionCardinalityWarning,
		Reason:  string(overall),
		Message: report.GetDetails(),
		Status:  status,
	}
	SetReady(fm)
}

func Sync(ctx context.Context, c client.Client, fm *metricslatest.FlowMetricList) {
	log := log.FromContext(ctx)
	log.Info("Syncing FlowMetrics status")
	for i := range fm.Items {
		nsname := types.NamespacedName{Name: fm.Items[i].Name, Namespace: fm.Items[i].Namespace}
		// main condition is mandatory; cardinality condition is optional
		if cond, ok := mapStatuses[nsname]; ok {
			cardCond := mapCards[nsname]
			setStatus(ctx, c, nsname, func(s *metricslatest.FlowMetricStatus) {
				if cond != nil {
					meta.SetStatusCondition(&s.Conditions, *cond)
				}
				if cardCond != nil {
					meta.SetStatusCondition(&s.Conditions, *cardCond)
				}
				s.PrometheusName = fm.Items[i].Status.PrometheusName
			})
		}
	}
}

func setStatus(ctx context.Context, c client.Client, nsname types.NamespacedName, applyStatus func(s *metricslatest.FlowMetricStatus)) {
	log := log.FromContext(ctx)

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fm := metricslatest.FlowMetric{}
		if err := c.Get(ctx, nsname, &fm); err != nil {
			log.WithValues("NsName", nsname).Error(err, "failed to get FlowMetrics status")
			if errors.IsNotFound(err) {
				// ignore: when it's being deleted, there's no point trying to update its status
				return nil
			}
			return err
		}
		applyStatus(&fm.Status)
		return c.Status().Update(ctx, &fm)
	})

	if err != nil {
		log.Error(err, "failed to update FlowMetrics status")
	}
}
