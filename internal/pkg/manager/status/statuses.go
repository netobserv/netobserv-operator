package status

import (
	flowslatest "github.com/netobserv/netobserv-operator/api/flowcollector/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

type Status string

const (
	StatusUnknown    Status = "Unknown"
	StatusUnused     Status = "Unused"
	StatusInProgress Status = "InProgress"
	StatusReady      Status = "Ready"
	StatusFailure    Status = "Failure"
	StatusDegraded   Status = "Degraded"
)

type ComponentStatus struct {
	Name            ComponentName
	Status          Status
	Reason          string
	Message         string
	DesiredReplicas *int32
	ReadyReplicas   *int32
	PodHealth       PodHealthSummary
}

// toCondition returns a Kubernetes condition using "Waiting*" naming with negative polarity:
// True means "component has an issue", False means "component is ready".
// This matches the OpenShift console behavior which treats non-Ready conditions as
// negative-polarity (True = problem).
func (s *ComponentStatus) toCondition() metav1.Condition {
	c := metav1.Condition{
		Type:    "Waiting" + string(s.Name),
		Message: s.Message,
	}
	switch s.Status {
	case StatusUnknown:
		c.Status = metav1.ConditionUnknown
		c.Reason = "Unknown"
	case StatusUnused:
		c.Status = metav1.ConditionUnknown
		c.Reason = "Unused"
	case StatusFailure, StatusInProgress, StatusDegraded:
		c.Status = metav1.ConditionTrue
		c.Reason = "NotReady"
	case StatusReady:
		c.Status = metav1.ConditionFalse
		c.Reason = "Ready"
	default:
		c.Status = metav1.ConditionUnknown
		c.Reason = "Unknown"
	}
	if s.Reason != "" {
		c.Reason = s.Reason
	}
	return c
}

func (s *ComponentStatus) toCRDStatus() *flowslatest.FlowCollectorComponentStatus {
	cs := &flowslatest.FlowCollectorComponentStatus{
		State:   string(s.Status),
		Reason:  s.Reason,
		Message: s.Message,
	}
	if s.DesiredReplicas != nil {
		cs.DesiredReplicas = ptr.To(*s.DesiredReplicas)
	}
	if s.ReadyReplicas != nil {
		cs.ReadyReplicas = ptr.To(*s.ReadyReplicas)
	}
	cs.UnhealthyPodCount = s.PodHealth.UnhealthyCount
	cs.PodIssues = s.PodHealth.Issues
	return cs
}
