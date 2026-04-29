package e2etests

import (
	"context"
	"fmt"
	"reflect"
	"time"

	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"

	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// FlowcollectorSlice struct to handle FlowcollectorSlice resources
type FlowcollectorSlice struct {
	Name         string
	Namespace    string
	Sampling     string
	SubnetLabels string
	Template     string
}

// create flowcollector CRD for a given manifest file
func (flowSlice FlowcollectorSlice) CreateFlowcollectorSlice(oc *exutil.CLI) {
	parameters := []string{"--ignore-unknown-parameters=true", "-f", flowSlice.Template, "-p"}

	flowCollector := reflect.ValueOf(&flowSlice).Elem()

	for i := 0; i < flowCollector.NumField(); i++ {
		if flowCollector.Field(i).Interface() != "" {
			if flowCollector.Type().Field(i).Name != "Template" {
				parameters = append(parameters, fmt.Sprintf("%s=%s", flowCollector.Type().Field(i).Name, flowCollector.Field(i).Interface()))
			}
		}
	}

	compat_otp.ApplyNsResourceFromTemplate(oc, flowSlice.Namespace, parameters...)
}

// DeleteFlowcollectorSlice deletes FlowCollectorSlice CRD from a cluster
func (flowSlice *FlowcollectorSlice) DeleteFlowcollectorSlice(oc *exutil.CLI) error {
	return oc.AsAdmin().WithoutNamespace().Run("delete").Args("flowcollectorslice", flowSlice.Name, "-n", flowSlice.Namespace).Execute()
}

// WaitForFlowcollectorSliceReady waits for FlowCollectorSlice to be ready by checking status conditions
func (flowSlice *FlowcollectorSlice) WaitForFlowcollectorSliceReady(oc *exutil.CLI) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 600*time.Second, false, func(context.Context) (done bool, err error) {
		// Check Ready condition
		readyCondition, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("flowcollectorslice", flowSlice.Name, "-n", flowSlice.Namespace, "-o", "jsonpath={.status.conditions[?(@.type==\"Ready\")].status}").Output()
		if err != nil || readyCondition != "True" {
			e2e.Logf("Error getting Ready condition: %v", err)
			return false, nil
		}
		return true, nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("FlowCollectorSlice %s/%s did not become Ready", flowSlice.Namespace, flowSlice.Name))
}
