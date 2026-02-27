package status

import (
	"context"
	"fmt"
	"strings"
	"sync"

	flowslatest "github.com/netobserv/network-observability-operator/api/flowcollector/v1beta2"
	"github.com/netobserv/network-observability-operator/internal/controller/constants"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type ComponentName string

const (
	FlowCollectorLegacy         ComponentName = "FlowCollectorLegacy"
	FLPParent                   ComponentName = "FLPParent"
	FLPMonolith                 ComponentName = "FLPMonolith"
	FLPTransformer              ComponentName = "FLPTransformer"
	Monitoring                  ComponentName = "Monitoring"
	StaticController            ComponentName = "StaticController"
	NetworkPolicy               ComponentName = "NetworkPolicy"
	ConditionConfigurationIssue               = "ConfigurationIssue"
	LokiIssue                                 = "LokiIssue"
	LokiWarning                               = "LokiWarning"
)

var allNames = []ComponentName{FlowCollectorLegacy, Monitoring, StaticController}

type Manager struct {
	statuses sync.Map
}

func NewManager() *Manager {
	s := Manager{}
	for _, cpnt := range allNames {
		s.statuses.Store(cpnt, ComponentStatus{
			name:   cpnt,
			status: StatusUnknown,
		})
	}
	return &s
}

func (s *Manager) setInProgress(cpnt ComponentName, reason, message string) {
	s.statuses.Store(cpnt, ComponentStatus{
		name:    cpnt,
		status:  StatusInProgress,
		reason:  reason,
		message: message,
	})
}

func (s *Manager) setFailure(cpnt ComponentName, reason, message string) {
	s.statuses.Store(cpnt, ComponentStatus{
		name:    cpnt,
		status:  StatusFailure,
		reason:  reason,
		message: message,
	})
}

func (s *Manager) hasFailure(cpnt ComponentName) bool {
	v, _ := s.statuses.Load(cpnt)
	return v != nil && v.(ComponentStatus).status == StatusFailure
}

func (s *Manager) setReady(cpnt ComponentName) {
	s.statuses.Store(cpnt, ComponentStatus{
		name:   cpnt,
		status: StatusReady,
	})
}

func (s *Manager) setUnknown(cpnt ComponentName) {
	s.statuses.Store(cpnt, ComponentStatus{
		name:   cpnt,
		status: StatusUnknown,
	})
}

func (s *Manager) setUnused(cpnt ComponentName, message string) {
	s.statuses.Store(cpnt, ComponentStatus{
		name:    cpnt,
		status:  StatusUnknown,
		reason:  "ComponentUnused",
		message: message,
	})
}

func (s *Manager) getConditions() []metav1.Condition {
	global := metav1.Condition{
		Type:   "Ready",
		Status: metav1.ConditionTrue,
		Reason: "Ready",
	}
	conds := []metav1.Condition{}
	counters := make(map[Status]int, len(allNames))
	s.statuses.Range(func(_, v any) bool {
		status := v.(ComponentStatus)
		conds = append(conds, status.toCondition())
		counters[status.status]++
		return true
	})
	global.Message = fmt.Sprintf("%d ready components, %d with failure, %d pending", counters[StatusReady], counters[StatusFailure], counters[StatusInProgress])
	if counters[StatusFailure] > 0 {
		global.Status = metav1.ConditionFalse
		global.Reason = "Failure"
	} else if counters[StatusInProgress] > 0 {
		global.Status = metav1.ConditionFalse
		global.Reason = "Pending"
	}
	return append([]metav1.Condition{global}, conds...)
}

func (s *Manager) Sync(ctx context.Context, c client.Client) {
	updateStatus(ctx, c, s.getConditions()...)
}

func updateStatus(ctx context.Context, c client.Client, conditions ...metav1.Condition) {
	log := log.FromContext(ctx)
	log.Info("Updating FlowCollector status")

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		fc := flowslatest.FlowCollector{}
		if err := c.Get(ctx, constants.FlowCollectorName, &fc); err != nil {
			if kerr.IsNotFound(err) {
				// ignore: when it's being deleted, there's no point trying to update its status
				return nil
			}
			return err
		}
		conditions = append(conditions, checkValidation(ctx, &fc))
		conditions = append(conditions, checkLoki(ctx, c, &fc))
		conditions = append(conditions, checkLokiWarnings(ctx, c, &fc))
		for _, c := range conditions {
			meta.SetStatusCondition(&fc.Status.Conditions, c)
		}
		return c.Status().Update(ctx, &fc)
	})

	if err != nil {
		log.Error(err, "failed to update FlowCollector status")
	}
}

func checkValidation(ctx context.Context, fc *flowslatest.FlowCollector) metav1.Condition {
	warnings, err := fc.Validate(ctx, fc)
	if err != nil {
		return metav1.Condition{
			Type:    ConditionConfigurationIssue,
			Reason:  "Error",
			Status:  metav1.ConditionTrue,
			Message: err.Error(),
		}
	}
	if len(warnings) > 0 {
		return metav1.Condition{
			Type:    ConditionConfigurationIssue,
			Reason:  "Warnings",
			Status:  metav1.ConditionTrue,
			Message: strings.Join(warnings, "; "),
		}
	}
	// No issue
	return metav1.Condition{
		Type:   ConditionConfigurationIssue,
		Reason: "Valid",
		Status: metav1.ConditionFalse,
	}
}

func (s *Manager) ForComponent(cpnt ComponentName) Instance {
	return Instance{cpnt: cpnt, s: s}
}

type Instance struct {
	cpnt ComponentName
	s    *Manager
}

func (i *Instance) SetReady() {
	i.s.setReady(i.cpnt)
}

func (i *Instance) SetUnknown() {
	i.s.setUnknown(i.cpnt)
}

func (i *Instance) SetUnused(message string) {
	i.s.setUnused(i.cpnt, message)
}

func (i *Instance) CheckDeploymentProgress(d *appsv1.Deployment) {
	// TODO (when legacy controller is broken down into individual controllers)
	// this should set the status as Ready when replicas match
	if d == nil {
		i.s.setInProgress(i.cpnt, "DeploymentNotCreated", "Deployment not created")
	} else {
		for _, c := range d.Status.Conditions {
			if c.Type == appsv1.DeploymentAvailable {
				if c.Status != v1.ConditionTrue {
					i.s.setInProgress(i.cpnt, "DeploymentNotReady", fmt.Sprintf("Deployment %s not ready: %d/%d (%s)", d.Name, d.Status.UpdatedReplicas, d.Status.Replicas, c.Message))
				}
				return
			}
		}
	}
}

func (i *Instance) CheckDaemonSetProgress(ds *appsv1.DaemonSet) {
	// TODO (when legacy controller is broken down into individual controllers)
	// this should set the status as Ready when replicas match
	if ds == nil {
		i.s.setInProgress(i.cpnt, "DaemonSetNotCreated", "DaemonSet not created")
	} else if ds.Status.UpdatedNumberScheduled < ds.Status.DesiredNumberScheduled {
		i.s.setInProgress(i.cpnt, "DaemonSetNotReady", fmt.Sprintf("DaemonSet %s not ready: %d/%d", ds.Name, ds.Status.UpdatedNumberScheduled, ds.Status.DesiredNumberScheduled))
	}
}

func (i *Instance) SetCreatingDeployment(d *appsv1.Deployment) {
	i.s.setInProgress(i.cpnt, "CreatingDeployment", fmt.Sprintf("Creating deployment %s", d.Name))
}

func (i *Instance) SetCreatingDaemonSet(ds *appsv1.DaemonSet) {
	i.s.setInProgress(i.cpnt, "CreatingDaemonSet", fmt.Sprintf("Creating daemon set %s", ds.Name))
}

func (i *Instance) SetFailure(reason, message string) {
	i.s.setFailure(i.cpnt, reason, message)
}

func (i *Instance) Error(reason string, err error) error {
	i.SetFailure(reason, err.Error())
	return err
}

func (i *Instance) HasFailure() bool {
	return i.s.hasFailure(i.cpnt)
}

func (i *Instance) Commit(ctx context.Context, c client.Client) {
	i.s.Sync(ctx, c)
}
