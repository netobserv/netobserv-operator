package v1alpha1

import (
	"context"
	"fmt"
	"strconv"

	"github.com/netobserv/netobserv-operator/internal/pkg/helper"
	"github.com/netobserv/netobserv-operator/internal/pkg/helper/cardinality"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var flowmetriclog = logf.Log.WithName("flowmetric-resource")

type FlowMetricWebhook struct {
	FlowMetric
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-flows-netobserv-io-v1alpha1-flowmetric,mutating=false,failurePolicy=fail,sideEffects=None,groups=flows.netobserv.io,resources=flowmetrics,versions=v1alpha1,name=flowmetricvalidationwebhook.netobserv.io,admissionReviewVersions=v1
func (r *FlowMetricWebhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FlowMetric{}).
		WithValidator(&FlowMetricWebhook{}).
		Complete()
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *FlowMetricWebhook) ValidateCreate(ctx context.Context, fm *FlowMetric) (warnings admission.Warnings, err error) {
	flowmetriclog.Info("validate create", "name", r.Name)
	return validateFlowMetric(ctx, fm)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *FlowMetricWebhook) ValidateUpdate(ctx context.Context, _, fm *FlowMetric) (warnings admission.Warnings, err error) {
	flowmetriclog.Info("validate update", "name", r.Name)
	return validateFlowMetric(ctx, fm)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *FlowMetricWebhook) ValidateDelete(_ context.Context, _ *FlowMetric) (warnings admission.Warnings, err error) {
	flowmetriclog.Info("validate delete", "name", r.Name)
	return nil, nil
}

func checkFlowMetricCardinality(fMetric *FlowMetric) admission.Warnings {
	w := admission.Warnings{}
	r, err := cardinality.CheckCardinality(fMetric.Spec.Labels...)
	if err != nil {
		flowmetriclog.WithValues("FlowMetric name", fMetric.Name).Error(err, "Could not check metrics cardinality")
		w = append(w, "Could not check metrics cardinality")
	}
	overallCardinality := r.GetOverall()
	if overallCardinality == cardinality.WarnAvoid || overallCardinality == cardinality.WarnUnknown {
		flowmetriclog.WithValues("FlowMetric name", fMetric.Name).Info("Warning: unsafe metric detected with potentially very high cardinality, please check its definition.", "Details", r.GetDetails())
		w = append(w, "This metric looks unsafe, with a potentially very high cardinality: "+r.GetDetails())
	} else if overallCardinality == cardinality.WarnCareful {
		w = append(w, "This metric has a potentially high cardinality: "+r.GetDetails())
	}
	return w
}

func validateFlowMetric(_ context.Context, fMetric *FlowMetric) (admission.Warnings, error) {
	var fields []string
	var allErrs field.ErrorList

	for _, f := range fMetric.Spec.Filters {
		fields = append(fields, f.Field)
	}

	if len(fields) != 0 {
		if !helper.FindFields(fields, false) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "filters"), fields,
				fmt.Sprintf("invalid filter field: %s", fields)))
		}
	}

	if len(fMetric.Spec.Labels) != 0 {
		if !helper.FindFields(fMetric.Spec.Labels, false) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "labels"), fMetric.Spec.Labels,
				fmt.Sprintf("invalid label name: %s", fMetric.Spec.Labels)))
		}

		labelsMap := make(map[string]any, len(fMetric.Spec.Labels))
		for _, label := range fMetric.Spec.Labels {
			labelsMap[label] = nil
		}

		// Only fields defined as Labels are valid for remapping
		if len(fMetric.Spec.Remap) != 0 {
			var invalidMapping []string
			for toRemap := range fMetric.Spec.Remap {
				if _, ok := labelsMap[toRemap]; !ok {
					invalidMapping = append(invalidMapping, toRemap)
				}
			}
			if len(invalidMapping) > 0 {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "remap"), fMetric.Spec.Remap,
					fmt.Sprintf("some fields defined for remapping are not defined as labels: %v", invalidMapping)))
			}
		}

		// Check for valid fields
		if len(fMetric.Spec.Flatten) != 0 {
			if !helper.FindFields(fMetric.Spec.Flatten, false) {
				allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "flatten"), fMetric.Spec.Flatten,
					fmt.Sprintf("invalid fields to flatten: %s", fMetric.Spec.Flatten)))
			}
		}
	}

	if fMetric.Spec.ValueField != "" {
		if !helper.FindFields([]string{fMetric.Spec.ValueField}, true) {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "valueField"), fMetric.Spec.ValueField,
				fmt.Sprintf("invalid value field: %s", fMetric.Spec.ValueField)))
		}
	}

	for _, b := range fMetric.Spec.Buckets {
		_, err := strconv.ParseFloat(b, 64)
		if err != nil {
			allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "buckets"), fMetric.Spec.Buckets,
				fmt.Sprintf(`cannot be parsed as a float: "%s"`, b)))
		}
	}

	if len(allErrs) != 0 {
		return nil, apierrors.NewInvalid(
			schema.GroupKind{Group: GroupVersion.Group, Kind: FlowMetric{}.Kind},
			fMetric.Name, allErrs)
	}
	w := checkFlowMetricCardinality(fMetric)
	return w, nil
}
