package e2etests

import (
	"fmt"
	"path/filepath"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	bgpTestASN                 = "64512"
	bgpExternalTarget          = "8.8.8.8"
	bgpInternalTarget          = "192.168.1.0"
	frrConfigurationCRDName    = "frrconfigurations.frrk8s.metallb.io"
	frrConfigurationFixture    = "frrconfiguration_template.yaml"
	frrConfigurationCRDFixture = "frrconfiguration_crd.yaml"
)

// FRRConfiguration deploys a fake FRRConfiguration CR for BGP ASN enrichment tests.
// No running BGP session is required; FLP reads advertised prefixes from the CR.
type FRRConfiguration struct {
	Name           string
	Namespace      string
	ASN            string
	ExternalPrefix string
	InternalPrefix string
	Template       string
}

func isFRRConfigurationAPIExists() (bool, error) {
	_, err := getDynamicResource("crd", frrConfigurationCRDName, "")
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func ensureFRRConfigurationAPI() error {
	exists, err := isFRRConfigurationAPIExists()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	crdPath := filepath.Join(baseDir, "bgp", frrConfigurationCRDFixture)
	e2e.Logf("FRRConfiguration CRD not found, applying %s", crdPath)
	ApplyResourceFromFile("", crdPath)
	exists, err = isFRRConfigurationAPIExists()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("FRRConfiguration CRD %s is still unavailable after apply", frrConfigurationCRDName)
	}
	return nil
}

func (frr FRRConfiguration) CreateFRRConfiguration() {
	parameters := []string{"--ignore-unknown-parameters=true", "-f", frr.Template, "-p"}
	frrConfig := reflect.ValueOf(&frr).Elem()

	for i := 0; i < frrConfig.NumField(); i++ {
		if frrConfig.Field(i).Interface() != "" {
			if frrConfig.Type().Field(i).Name != "Template" {
				parameters = append(parameters, fmt.Sprintf("%s=%s", frrConfig.Type().Field(i).Name, frrConfig.Field(i).Interface()))
			}
		}
	}

	err := applyNsResourceFromTemplateByAdmin(frr.Namespace, parameters...)
	if err != nil {
		e2e.Failf("Failed to create FRRConfiguration: %v", err)
	}
}

func (frr *FRRConfiguration) DeleteFRRConfiguration() error {
	return deleteDynamicResource("frrconfiguration", frr.Name, frr.Namespace)
}
