package e2etests

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/onsi/ginkgo/v2"
	exutil "github.com/openshift/origin/test/extended/util"
)

type OCPVersion struct {
	Major int
	Minor int
}

var clusterVersion *OCPVersion

func GetOCPVersion(oc *exutil.CLI) (*OCPVersion, error) {

	if clusterVersion != nil {
		return clusterVersion, nil
	}

	version, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("clusterversion", "-o=jsonpath={.items[0].status.desired.version}").Output()
	if err != nil {
		return nil, err
	}

	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid version: %s", version)
	}

	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])

	clusterVersion = &OCPVersion{Major: major, Minor: minor}
	return clusterVersion, nil
}

func (v *OCPVersion) AtLeast(major, minor int) bool {
	if v.Major > major {
		return true
	}
	return v.Major == major && v.Minor >= minor
}

func (v *OCPVersion) String() string {
	return fmt.Sprintf("%d.%d", v.Major, v.Minor)
}

// SkipIfOCPBelow skips test if cluster version is below requirement
func SkipIfOCPBelow(major, minor int) {
	if clusterVersion == nil {
		ginkgo.Fail("Cluster version not initialized")
	}
	if !clusterVersion.AtLeast(major, minor) {
		ginkgo.Skip(fmt.Sprintf("Requires OCP %d.%d+, cluster is %s", major, minor, clusterVersion))
	}
}
