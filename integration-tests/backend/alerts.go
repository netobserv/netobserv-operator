package e2etests

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
)

func getConfiguredAlertRules(oc *exutil.CLI, ruleName string, namespace string) (string, error) {
	return oc.AsAdmin().WithoutNamespace().Run("get").Args("prometheusrules", ruleName, "-o=jsonpath='{.spec.groups[*].rules[*].alert}'", "-n", namespace).Output()
}

func getAlertStatus(oc *exutil.CLI, alertName string) (map[string]interface{}, error) {
	alertOut, err := oc.AsAdmin().WithoutNamespace().Run("exec").Args("-n", "openshift-monitoring", "alertmanager-main-0", "--", "amtool", "--alertmanager.url", "http://localhost:9093", "alert", "query", alertName, "-o", "json").Output()
	if err != nil {
		return make(map[string]interface{}), err
	}
	var alertStatus []interface{}
	_ = json.Unmarshal([]byte(alertOut), &alertStatus)

	if len(alertStatus) == 0 {
		return make(map[string]interface{}), nil
	}
	return alertStatus[0].(map[string]interface{}), nil
}

func waitForAlertToBeActive(oc *exutil.CLI, alertName string) {
	err := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, 900*time.Second, false, func(context.Context) (done bool, err error) {
		alertStatus, err := getAlertStatus(oc, alertName)
		if err != nil {
			return false, err
		}
		if len(alertStatus) == 0 {
			return false, nil
		}
		return alertStatus["status"].(map[string]interface{})["state"] == "active", nil
	})
	compat_otp.AssertWaitPollNoErr(err, fmt.Sprintf("%s Alert did not become active", alertName))
}
