package status

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Status string

const (
	StatusUnknown    Status = "Unknown"
	StatusInProgress Status = "InProgress"
	StatusReady      Status = "Ready"
	StatusFailure    Status = "Failure"
	StatusDegraded   Status = "Degraded"
)

type ComponentStatus struct {
	Name    ComponentName
	Status  Status
	Reason  string
	Message string
}

func (s *ComponentStatus) toCondition() metav1.Condition {
	c := metav1.Condition{
		Type:    "Waiting" + string(s.Name),
		Message: s.Message,
	}
	switch s.Status {
	case StatusUnknown:
		c.Status = metav1.ConditionUnknown
		c.Reason = "Unused"
	case StatusFailure, StatusInProgress, StatusDegraded:
		c.Status = metav1.ConditionTrue
		c.Reason = "NotReady"
	case StatusReady:
		c.Status = metav1.ConditionFalse
		c.Reason = "Ready"
	}
	if s.Reason != "" {
		c.Reason = s.Reason
	}
	return c
}
