package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:verbs=create;update,path=/validate-flows-netobserv-io-v1alpha1-flowcapture,mutating=false,failurePolicy=fail,sideEffects=None,groups=flows.netobserv.io,resources=flowcaptures,versions=v1alpha1,name=flowcapturevalidationwebhook.netobserv.io,admissionReviewVersions=v1
func (r *FlowCapture) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, r).
		WithValidator(r).
		Complete()
}

// ValidateCreate implements webhook.Validator.
func (r *FlowCapture) ValidateCreate(_ context.Context, fc *FlowCapture) (admission.Warnings, error) {
	if err := validateFlowCapture(fc); err != nil {
		return nil, err
	}
	return flowCaptureWarnings(fc), nil
}

// ValidateUpdate implements webhook.Validator.
func (r *FlowCapture) ValidateUpdate(_ context.Context, _, fc *FlowCapture) (admission.Warnings, error) {
	if err := validateFlowCapture(fc); err != nil {
		return nil, err
	}
	return flowCaptureWarnings(fc), nil
}

// ValidateDelete implements webhook.Validator.
func (r *FlowCapture) ValidateDelete(_ context.Context, _ *FlowCapture) (admission.Warnings, error) {
	return nil, nil
}

func flowCaptureWarnings(fc *FlowCapture) admission.Warnings {
	var warnings admission.Warnings
	if fc.Spec.MaxTime == nil && fc.Spec.MaxBytes == 0 {
		warnings = append(warnings, "spec.maxTime or spec.maxBytes should be set to bound the capture lifetime")
	}
	if fc.Spec.OutputType == Metrics && fc.Spec.MaxBytes > 0 {
		warnings = append(warnings, "spec.maxBytes is ignored for metrics FlowCaptures")
	}
	return warnings
}

func validateFlowCapture(fc *FlowCapture) error {
	if fc.Spec.TLSPlaintext != nil && fc.Spec.OutputType != Packets {
		return fmt.Errorf("spec.tlsPlaintext is only supported for packet captures")
	}
	for i, query := range fc.Spec.Queries {
		if strings.TrimSpace(query) == "" {
			return fmt.Errorf("spec.queries[%d] must not be empty", i)
		}
	}
	return nil
}
