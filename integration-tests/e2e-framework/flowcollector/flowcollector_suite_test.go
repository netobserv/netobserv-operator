package flowcollector_test

import (
	"testing"

	e2eframework "e2eframework"

	"sigs.k8s.io/e2e-framework/pkg/env"
)

var testEnv env.Environment

func TestMain(m *testing.M) {
	e2eframework.Main(m, &testEnv)
}
