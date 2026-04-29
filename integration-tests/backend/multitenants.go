package e2etests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	filePath "path/filepath"
	"reflect"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

type User struct {
	Username string
	Password string
}

func getCoStatus(oc *exutil.CLI, coName string, statusToCompare map[string]string) map[string]string {
	newStatusToCompare := make(map[string]string)
	for key := range statusToCompare {
		args := fmt.Sprintf(`-o=jsonpath={.status.conditions[?(.type == '%s')].status}`, key)
		status, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("co", args, coName).Output()
		o.Expect(err).NotTo(o.HaveOccurred())
		newStatusToCompare[key] = status
	}
	return newStatusToCompare
}

func waitCoBecomes(oc *exutil.CLI, coName string, waitTime int, expectedStatus map[string]string) error {
	errCo := wait.PollUntilContextTimeout(context.Background(), 10*time.Second, time.Duration(waitTime)*time.Second, false, func(context.Context) (bool, error) {
		gottenStatus := getCoStatus(oc, coName, expectedStatus)
		eq := reflect.DeepEqual(expectedStatus, gottenStatus)
		if eq {
			eq := reflect.DeepEqual(expectedStatus, map[string]string{"Available": "True", "Progressing": "False", "Degraded": "False"})
			if eq {
				// For True False False, we want to wait some bit more time and double check, to ensure it is stably healthy
				time.Sleep(25 * time.Second)
				gottenStatus := getCoStatus(oc, coName, expectedStatus)
				eq := reflect.DeepEqual(expectedStatus, gottenStatus)
				if eq {
					e2e.Logf("Given operator %s becomes available/non-progressing/non-degraded +%v", coName, gottenStatus)
					return true, nil
				}
			} else {
				e2e.Logf("Given operator %s becomes %s", coName, gottenStatus)
				return true, nil
			}
		}
		return false, nil
	})
	if errCo != nil {
		err := oc.AsAdmin().WithoutNamespace().Run("get").Args("co").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	}
	return errCo
}

func generateUsersHtpasswd(passwdFile *string, users []*User) error {
	for i := 0; i < len(users); i++ {
		// Generate new username and password
		username := fmt.Sprintf("testuser-%v-%v", i, compat_otp.GetRandomString())
		password := compat_otp.GetRandomString()
		users[i] = &User{Username: username, Password: password}

		// Add new user to htpasswd file in the temp directory
		cmd := fmt.Sprintf("htpasswd -b %v %v %v", *passwdFile, users[i].Username, users[i].Password)
		err := exec.Command("bash", "-c", cmd).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func getNewUser(oc *exutil.CLI, count int) ([]*User, string, string) {
	usersDirPath := "/tmp/" + compat_otp.GetRandomString()
	usersHTpassFile := usersDirPath + "/htpasswd"
	err := os.MkdirAll(usersDirPath, 0o755)
	o.Expect(err).NotTo(o.HaveOccurred())

	htPassSecret, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("oauth/cluster", "-o", "jsonpath={.spec.identityProviders[0].htpasswd.fileData.name}").Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	users := make([]*User, count)
	if htPassSecret == "" {
		htPassSecret = "htpass-secret"
		_, _ = os.Create(usersHTpassFile)
		err = generateUsersHtpasswd(&usersHTpassFile, users)
		o.Expect(err).NotTo(o.HaveOccurred())
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("-n", "openshift-config", "secret", "generic", htPassSecret, "--from-file", "htpasswd="+usersHTpassFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("--type=json", "-p", `[{"op": "add", "path": "/spec/identityProviders", "value": [{"htpasswd": {"fileData": {"name": "htpass-secret"}}, "mappingMethod": "claim", "name": "htpasswd", "type": "HTPasswd"}]}]`, "oauth/cluster").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	} else {
		err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("-n", "openshift-config", "secret/"+htPassSecret, "--to", usersDirPath, "--confirm").Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
		err = generateUsersHtpasswd(&usersHTpassFile, users)
		o.Expect(err).NotTo(o.HaveOccurred())
		// Update htpass-secret with the modified htpasswd file
		err = oc.AsAdmin().WithoutNamespace().Run("set").Args("-n", "openshift-config", "data", "secret/"+htPassSecret, "--from-file", "htpasswd="+usersHTpassFile).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	g.By("Checking authentication operator should be in Progressing in 180 seconds")
	err = waitCoBecomes(oc, "authentication", 180, map[string]string{"Progressing": "True"})
	compat_otp.AssertWaitPollNoErr(err, "authentication operator did not start progressing in 180 seconds")
	e2e.Logf("Checking authentication operator should be Available in 600 seconds")
	err = waitCoBecomes(oc, "authentication", 600, map[string]string{"Available": "True", "Progressing": "False", "Degraded": "False"})
	compat_otp.AssertWaitPollNoErr(err, "authentication operator did not become available in 600 seconds")

	return users, usersHTpassFile, htPassSecret
}

func userCleanup(oc *exutil.CLI, users []*User, usersHTpassFile string, htPassSecret string) {
	defer os.RemoveAll(usersHTpassFile)
	for i := range users {
		// Add new user to htpasswd file in the temp directory
		cmd := fmt.Sprintf("htpasswd -D %v %v", usersHTpassFile, users[i].Username)
		err := exec.Command("bash", "-c", cmd).Run()
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	// Update htpass-secret with the modified htpasswd file
	err := oc.AsAdmin().WithoutNamespace().Run("set").Args("-n", "openshift-config", "data", "secret/"+htPassSecret, "--from-file", "htpasswd="+usersHTpassFile).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())

	g.By("Checking authentication operator should be in Progressing in 180 seconds")
	err = waitCoBecomes(oc, "authentication", 180, map[string]string{"Progressing": "True"})
	compat_otp.AssertWaitPollNoErr(err, "authentication operator did not start progressing in 180 seconds")
	e2e.Logf("Checking authentication operator should be Available in 600 seconds")
	err = waitCoBecomes(oc, "authentication", 600, map[string]string{"Available": "True", "Progressing": "False", "Degraded": "False"})
	compat_otp.AssertWaitPollNoErr(err, "authentication operator did not become available in 600 seconds")
}

func addUserAsReader(oc *exutil.CLI, username string) {
	baseDir, _ := filePath.Abs("testdata")
	readerCRBPath := filePath.Join(baseDir, "netobserv-loki-reader-multitenant-crb.yaml")
	parameters := []string{"-f", readerCRBPath, "-p", "USERNAME=" + username}
	compat_otp.CreateClusterResourceFromTemplate(oc, parameters...)
}

func removeUserAsReader(oc *exutil.CLI, username string) {
	err := oc.AsAdmin().WithoutNamespace().Run("adm").Args("policy", "remove-cluster-role-from-user", "netobserv-loki-reader", username).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}

func addTemplatePermissions(oc *exutil.CLI, username string) {
	baseDir, _ := filePath.Abs("testdata")
	readerCRBPath := filePath.Join(baseDir, "testuser-template-crb.yaml")
	parameters := []string{"-f", readerCRBPath, "-p", "USERNAME=" + username}
	compat_otp.CreateClusterResourceFromTemplate(oc, parameters...)
}

func removeTemplatePermissions(oc *exutil.CLI, username string) {
	baseDir, _ := filePath.Abs("testdata")
	readerCRBPath := filePath.Join(baseDir, "testuser-template-crb.yaml")
	parameters := []string{"-f", readerCRBPath, "-p", "USERNAME=" + username}
	configFile := compat_otp.ProcessTemplate(oc, parameters...)
	err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("-f", configFile).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
}
