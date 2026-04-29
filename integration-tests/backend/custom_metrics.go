package e2etests

import (
	"fmt"
	"os"
	"reflect"

	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"gopkg.in/yaml.v3"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type CustomMetrics struct {
	Namespace string
	Template  string
}

type CustomMetricsTemplateConfig struct {
	Objects []interface{} `yaml:"objects"`
}

type CustomMetricsConfig struct {
	DashboardNames []string
	MetricName     string
	Queries        []string
}

// create flowmetrics resource from template
func (cm CustomMetrics) createCustomMetrics(oc *exutil.CLI) {
	parameters := []string{"--ignore-unknown-parameters=true", "-f", cm.Template, "-p"}
	cmr := reflect.ValueOf(&cm).Elem()
	for i := 0; i < cmr.NumField(); i++ {
		if cmr.Field(i).Interface() != "" {
			if cmr.Type().Field(i).Name != "Template" {
				parameters = append(parameters, fmt.Sprintf("%s=%s", cmr.Type().Field(i).Name, cmr.Field(i).Interface()))
			}
		}
	}
	compat_otp.ApplyNsResourceFromTemplate(oc, cm.Namespace, parameters...)
}

// parse custom metrics yaml template
func (cm CustomMetrics) parseTemplate() *CustomMetricsTemplateConfig {
	yamlFile, err := os.ReadFile(cm.Template)

	if err != nil {
		e2e.Failf("Could not read the template file %s", cm.Template)
	}
	var cmc *CustomMetricsTemplateConfig
	err = yaml.Unmarshal(yamlFile, &cmc)
	if err != nil {
		e2e.Failf("Could not Unmarshal %v", err)
	}
	return cmc
}

// returns queries and dashboardNames
func getChartsConfig(chartsConfig []interface{}) ([]string, []string) {
	var result []string
	var dashboardNames []string
	for _, conf := range chartsConfig {
		chartsConf := conf.(map[string]interface{})
		for k, v := range chartsConf {
			if k == "dashboardName" {
				dashboardNames = append(dashboardNames, v.(string))
			}
			if k == "queries" {
				queries := v.([]interface{})
				for _, qConf := range queries {
					queryConf := qConf.(map[string]interface{})
					for qk, qv := range queryConf {
						if qk == "promQL" {
							result = append(result, qv.(string))
						}
					}
				}
			}
		}
	}
	return result, dashboardNames
}

// returns slice of CustomMetricsConfig
func (cm CustomMetrics) getCustomMetricConfigs() []CustomMetricsConfig {
	cmc := cm.parseTemplate()
	// var customMetricsConfig []map[string][]string
	var cmConfigs []CustomMetricsConfig
	for _, template := range cmc.Objects {
		var cmConfig CustomMetricsConfig
		t := template.(map[string]interface{})
		for object, v := range t {
			if object == "spec" {
				spec := v.(map[string]interface{})
				for config, val := range spec {
					if config == "charts" {
						chartsConfig := val.([]interface{})
						cmConfig.Queries, cmConfig.DashboardNames = getChartsConfig(chartsConfig)
					}
					if config == "metricName" {
						cmConfig.MetricName = val.(string)
					}
				}
				cmConfigs = append(cmConfigs, cmConfig)
			}
		}
	}
	return cmConfigs
}
