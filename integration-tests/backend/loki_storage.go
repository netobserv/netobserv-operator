package e2etests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	filePath "path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	exutil "github.com/openshift/origin/test/extended/util"
	compat_otp "github.com/openshift/origin/test/extended/util/compat_otp"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

const (
	minioNS        = "minio-aosqe"
	minioSecret    = "minio-creds"
	apiPath        = "/api/logs/v1/"
	queryRangePath = "/loki/api/v1/query_range"
	loNS           = "openshift-operators-redhat"
)

// s3Credential defines the s3 credentials
type s3Credential struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // the endpoint of s3 service
}

func getAWSCredentialFromCluster(oc *exutil.CLI) s3Credential {
	region, err := compat_otp.GetAWSClusterRegion(oc)
	o.Expect(err).NotTo(o.HaveOccurred())

	dirname := "/tmp/" + oc.Namespace() + "-creds"
	defer os.RemoveAll(dirname)
	err = os.MkdirAll(dirname, 0777)
	o.Expect(err).NotTo(o.HaveOccurred())

	_, err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("secret/aws-creds", "-n", "kube-system", "--confirm", "--to="+dirname).Output()
	o.Expect(err).NotTo(o.HaveOccurred())

	accessKeyID, err := os.ReadFile(dirname + "/aws_access_key_id")
	o.Expect(err).NotTo(o.HaveOccurred())
	secretAccessKey, err := os.ReadFile(dirname + "/aws_secret_access_key")
	o.Expect(err).NotTo(o.HaveOccurred())

	cred := s3Credential{Region: region, AccessKeyID: string(accessKeyID), SecretAccessKey: string(secretAccessKey)}
	return cred
}

func getMinIOCreds(oc *exutil.CLI, ns string) s3Credential {
	dirname := "/tmp/" + oc.Namespace() + "-creds"
	defer os.RemoveAll(dirname)
	err := os.MkdirAll(dirname, 0777)
	o.Expect(err).NotTo(o.HaveOccurred())

	_, err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("secret/"+minioSecret, "-n", ns, "--confirm", "--to="+dirname).Output()
	o.Expect(err).NotTo(o.HaveOccurred())

	accessKeyID, err := os.ReadFile(dirname + "/access_key_id")
	o.Expect(err).NotTo(o.HaveOccurred())
	secretAccessKey, err := os.ReadFile(dirname + "/secret_access_key")
	o.Expect(err).NotTo(o.HaveOccurred())

	endpoint := "http://" + getRouteAddress(oc, ns, "minio")
	return s3Credential{Endpoint: endpoint, AccessKeyID: string(accessKeyID), SecretAccessKey: string(secretAccessKey)}
}

func generateS3Config(cred s3Credential) aws.Config {
	var err error
	var cfg aws.Config
	if len(cred.Endpoint) > 0 {
		customResolver := aws.EndpointResolverWithOptionsFunc(func(_, _ string, _ ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{
				URL:               cred.Endpoint,
				HostnameImmutable: true,
				Source:            aws.EndpointSourceCustom,
			}, nil
		})
		// For ODF and Minio, they're deployed in OCP clusters
		// In some clusters, we can't connect it without proxy, here add proxy settings to s3 client when there has http_proxy or https_proxy in the env var
		httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
			proxy := getProxyFromEnv()
			if len(proxy) > 0 {
				proxyURL, err := url.Parse(proxy)
				o.Expect(err).NotTo(o.HaveOccurred())
				tr.Proxy = http.ProxyURL(proxyURL)
			}
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cred.AccessKeyID, cred.SecretAccessKey, "")),
			config.WithEndpointResolverWithOptions(customResolver),
			config.WithHTTPClient(httpClient),
			config.WithRegion("auto"))
	} else {
		// aws s3
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cred.AccessKeyID, cred.SecretAccessKey, "")),
			config.WithRegion(cred.Region))
	}
	o.Expect(err).NotTo(o.HaveOccurred())
	return cfg
}

func createS3Bucket(client *s3.Client, bucketName, region string) error {
	// check if the bucket exists or not
	// if exists, clear all the objects in the bucket
	// if not, create the bucket
	exist := false
	buckets, err := client.ListBuckets(context.TODO(), &s3.ListBucketsInput{})
	o.Expect(err).NotTo(o.HaveOccurred())
	for _, bu := range buckets.Buckets {
		if *bu.Name == bucketName {
			exist = true
			break
		}
	}
	// clear all the objects in the bucket
	if exist {
		return emptyS3Bucket(client, bucketName)
	}

	/*
		Per https://docs.aws.amazon.com/AmazonS3/latest/API/API_CreateBucket.html#API_CreateBucket_RequestBody,
		us-east-1 is the default region and it's not a valid value of LocationConstraint,
		using `LocationConstraint: types.BucketLocationConstraint("us-east-1")` gets error `InvalidLocationConstraint`.
		Here remove the configration when the region is us-east-1
	*/
	if len(region) == 0 || region == "us-east-1" {
		_, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: &bucketName})
		return err
	}
	_, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: &bucketName, CreateBucketConfiguration: &types.CreateBucketConfiguration{LocationConstraint: types.BucketLocationConstraint(region)}})
	return err
}

func deleteS3Bucket(client *s3.Client, bucketName string) error {
	// empty bucket
	err := emptyS3Bucket(client, bucketName)
	if err != nil {
		return err
	}
	// delete bucket
	_, err = client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: &bucketName})
	return err
}

func emptyS3Bucket(client *s3.Client, bucketName string) error {
	// List objects in the bucket
	objects, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	})
	if err != nil {
		return err
	}

	// Delete objects in the bucket
	if len(objects.Contents) > 0 {
		objectIdentifiers := make([]types.ObjectIdentifier, len(objects.Contents))
		for i, object := range objects.Contents {
			objectIdentifiers[i] = types.ObjectIdentifier{Key: object.Key}
		}

		quiet := true
		_, err = client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: &bucketName,
			Delete: &types.Delete{
				Objects: objectIdentifiers,
				Quiet:   &quiet,
			},
		})
		if err != nil {
			return err
		}
	}

	// Check if there are more objects to delete and handle pagination
	if *objects.IsTruncated {
		return emptyS3Bucket(client, bucketName)
	}

	return nil
}

// createSecretForAWSS3Bucket creates a secret for Loki to connect to s3 bucket
func createSecretForAWSS3Bucket(oc *exutil.CLI, bucketName, secretName, ns string, cred s3Credential) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	endpoint := "https://s3." + cred.Region + ".amazonaws.com"
	return oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", secretName, "--from-literal=access_key_id="+cred.AccessKeyID, "--from-literal=access_key_secret="+cred.SecretAccessKey, "--from-literal=region="+cred.Region, "--from-literal=bucketnames="+bucketName, "--from-literal=endpoint="+endpoint, "-n", ns).Execute()
}

func createSecretForODFBucket(oc *exutil.CLI, bucketName, secretName, ns string) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}
	dirname := "/tmp/" + oc.Namespace() + "-creds"
	err := os.MkdirAll(dirname, 0777)
	o.Expect(err).NotTo(o.HaveOccurred())
	defer os.RemoveAll(dirname)
	_, err = oc.AsAdmin().WithoutNamespace().Run("extract").Args("secret/noobaa-admin", "-n", "openshift-storage", "--confirm", "--to="+dirname).Output()
	o.Expect(err).NotTo(o.HaveOccurred())

	endpoint := "http://s3.openshift-storage.svc:80"
	return oc.AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", secretName, "--from-file=access_key_id="+dirname+"/AWS_ACCESS_KEY_ID", "--from-file=access_key_secret="+dirname+"/AWS_SECRET_ACCESS_KEY", "--from-literal=bucketnames="+bucketName, "--from-literal=endpoint="+endpoint, "-n", ns).Execute()
}

func createSecretForMinIOBucket(oc *exutil.CLI, bucketName, secretName, ns string, cred s3Credential) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}
	return oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", secretName, "--from-literal=access_key_id="+cred.AccessKeyID, "--from-literal=access_key_secret="+cred.SecretAccessKey, "--from-literal=bucketnames="+bucketName, "--from-literal=endpoint="+cred.Endpoint, "-n", ns).Execute()
}

func getGCPProjectNumber(projectID string) (string, error) {
	crmService, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return "", err
	}

	project, err := crmService.Projects.Get(projectID).Do()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(project.ProjectNumber, 10), nil
}

func generateServiceAccountNameForGCS(clusterName string) string {
	// Service Account should be between 6-30 characters long
	name := clusterName + getRandomString()
	if len(name) > 30 {
		return (name[0:30])
	}
	return name
}

func createServiceAccountOnGCP(projectID, name string) (*iam.ServiceAccount, error) {
	ctx := context.Background()
	service, err := iam.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("iam.NewService: %w", err)
	}

	request := &iam.CreateServiceAccountRequest{
		AccountId: name,
		ServiceAccount: &iam.ServiceAccount{
			DisplayName: "Service Account for " + name,
		},
	}
	account, err := service.Projects.ServiceAccounts.Create("projects/"+projectID, request).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to create serviceaccount: %w", err)
	}
	e2e.Logf("create serviceaccount: %s successfully", account.Name)
	return account, nil
}

// ref: https://github.com/GoogleCloudPlatform/golang-samples/blob/main/iam/quickstart/quickstart.go
func addBinding(projectID, member, role string) error {
	crmService, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return fmt.Errorf("cloudresourcemanager.NewService: %w", err)
	}

	err = wait.ExponentialBackoffWithContext(context.Background(), wait.Backoff{Steps: 5, Factor: 2, Duration: 5 * time.Second}, func(context.Context) (done bool, err error) {
		policy, err := getPolicy(crmService, projectID)
		if err != nil {
			return false, fmt.Errorf("error getting policy: %w", err)
		}
		// Find the policy binding for role. Only one binding can have the role.
		var binding *cloudresourcemanager.Binding
		for _, b := range policy.Bindings {
			if b.Role == role {
				binding = b
				break
			}
		}
		if binding != nil {
			// If the binding exists, adds the member to the binding
			binding.Members = append(binding.Members, member)
		} else {
			// If the binding does not exist, adds a new binding to the policy
			binding = &cloudresourcemanager.Binding{
				Role:    role,
				Members: []string{member},
			}
			policy.Bindings = append(policy.Bindings, binding)
		}
		err = setPolicy(crmService, projectID, policy)
		if err == nil {
			return true, nil
		}
		/*
			According to https://github.com/hashicorp/terraform-provider-google/issues/8280, deleting another serviceaccount can make 400 error happen, so retry this step when 400 error happens
		*/
		if strings.Contains(err.Error(), `googleapi: Error 409: There were concurrent policy changes. Please retry the whole read-modify-write with exponential backoff.`) ||
			(strings.Contains(err.Error(), "googleapi: Error 400: Service account") && strings.Contains(err.Error(), "does not exist., badRequest")) {
			e2e.Logf("Hit error: %v, retry the request", err)
			return false, nil
		}
		e2e.Logf("Failed to update polilcy: %v", err)
		return false, err
	})
	if err != nil {
		return fmt.Errorf("failed to add role %s to %s", role, member)
	}
	return nil
}

// removeMember removes the member from the project's IAM policy
func removeMember(projectID, member, role string) error {
	crmService, err := cloudresourcemanager.NewService(context.Background())
	if err != nil {
		return fmt.Errorf("cloudresourcemanager.NewService: %w", err)
	}
	err = wait.ExponentialBackoffWithContext(context.Background(), wait.Backoff{Steps: 5, Factor: 2, Duration: 5 * time.Second}, func(context.Context) (done bool, err error) {
		policy, err := getPolicy(crmService, projectID)
		if err != nil {
			return false, fmt.Errorf("error getting policy: %w", err)
		}
		// Find the policy binding for role. Only one binding can have the role.
		var binding *cloudresourcemanager.Binding
		var bindingIndex int
		for i, b := range policy.Bindings {
			if b.Role == role {
				binding = b
				bindingIndex = i
				break
			}
		}

		if len(binding.Members) == 1 && binding.Members[0] == member {
			// If the member is the only member in the binding, removes the binding
			last := len(policy.Bindings) - 1
			policy.Bindings[bindingIndex] = policy.Bindings[last]
			policy.Bindings = policy.Bindings[:last]
		} else {
			// If there is more than one member in the binding, removes the member
			var memberIndex int
			var exist bool
			for i, mm := range binding.Members {
				if mm == member {
					memberIndex = i
					exist = true
					break
				}
			}
			if exist {
				last := len(policy.Bindings[bindingIndex].Members) - 1
				binding.Members[memberIndex] = binding.Members[last]
				binding.Members = binding.Members[:last]
			}
		}

		err = setPolicy(crmService, projectID, policy)
		if err == nil {
			return true, nil
		}
		if strings.Contains(err.Error(), `googleapi: Error 409: There were concurrent policy changes. Please retry the whole read-modify-write with exponential backoff.`) ||
			(strings.Contains(err.Error(), "googleapi: Error 400: Service account") && strings.Contains(err.Error(), "does not exist., badRequest")) {
			e2e.Logf("Hit error: %v, retry the request", err)
			return false, nil
		}
		e2e.Logf("Failed to update polilcy: %v", err)
		return false, err
	})
	if err != nil {
		return fmt.Errorf("failed to remove %s", member)
	}
	return nil
}

// getPolicy gets the project's IAM policy
func getPolicy(crmService *cloudresourcemanager.Service, projectID string) (*cloudresourcemanager.Policy, error) {
	request := new(cloudresourcemanager.GetIamPolicyRequest)
	policy, err := crmService.Projects.GetIamPolicy(projectID, request).Do()
	if err != nil {
		return nil, err
	}
	return policy, nil
}

// setPolicy sets the project's IAM policy
func setPolicy(crmService *cloudresourcemanager.Service, projectID string, policy *cloudresourcemanager.Policy) error {
	request := new(cloudresourcemanager.SetIamPolicyRequest)
	request.Policy = policy
	_, err := crmService.Projects.SetIamPolicy(projectID, request).Do()
	return err
}

func grantPermissionsToGCPServiceAccount(poolID, projectID, projectNumber, lokiNS, lokiStackName, serviceAccountEmail string) error {
	gcsRoles := []string{
		"roles/iam.workloadIdentityUser",
		"roles/storage.objectAdmin",
	}
	subjects := []string{
		"system:serviceaccount:" + lokiNS + ":" + lokiStackName,
		"system:serviceaccount:" + lokiNS + ":" + lokiStackName + "-ruler",
	}

	for _, role := range gcsRoles {
		err := addBinding(projectID, "serviceAccount:"+serviceAccountEmail, role)
		if err != nil {
			return fmt.Errorf("error adding role %s to %s: %w", role, serviceAccountEmail, err)
		}
		for _, sub := range subjects {
			err := addBinding(projectID, "principal://iam.googleapis.com/projects/"+projectNumber+"/locations/global/workloadIdentityPools/"+poolID+"/subject/"+sub, role)
			if err != nil {
				return fmt.Errorf("error adding role %s to %s: %w", role, sub, err)
			}
		}
	}
	return nil
}

func removePermissionsFromGCPServiceAccount(poolID, projectID, projectNumber, lokiNS, lokiStackName, serviceAccountEmail string) error {
	gcsRoles := []string{
		"roles/iam.workloadIdentityUser",
		"roles/storage.objectAdmin",
	}
	subjects := []string{
		"system:serviceaccount:" + lokiNS + ":" + lokiStackName,
		"system:serviceaccount:" + lokiNS + ":" + lokiStackName + "-ruler",
	}

	for _, role := range gcsRoles {
		err := removeMember(projectID, "serviceAccount:"+serviceAccountEmail, role)
		if err != nil {
			return fmt.Errorf("error removing role %s from %s: %w", role, serviceAccountEmail, err)
		}
		for _, sub := range subjects {
			err := removeMember(projectID, "principal://iam.googleapis.com/projects/"+projectNumber+"/locations/global/workloadIdentityPools/"+poolID+"/subject/"+sub, role)
			if err != nil {
				return fmt.Errorf("error removing role %s from %s: %w", role, sub, err)
			}
		}
	}
	return nil
}

func removeServiceAccountFromGCP(name string) error {
	ctx := context.Background()
	service, err := iam.NewService(ctx)
	if err != nil {
		return fmt.Errorf("iam.NewService: %w", err)
	}
	_, err = service.Projects.ServiceAccounts.Delete(name).Do()
	if err != nil {
		return fmt.Errorf("can't remove service account: %w", err)
	}
	return nil
}

func createSecretForGCSBucketWithSTS(oc *exutil.CLI, namespace, secretName, bucketName string) error {
	return oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", "-n", namespace, secretName, "--from-literal=bucketname="+bucketName).Execute()
}

// creates a secret for Loki to connect to gcs bucket
func createSecretForGCSBucket(oc *exutil.CLI, bucketName, secretName, ns string) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	// get gcp-credentials from env var GOOGLE_APPLICATION_CREDENTIALS
	gcsCred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	return oc.AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", secretName, "-n", ns, "--from-literal=bucketname="+bucketName, "--from-file=key.json="+gcsCred).Execute()
}

// creates a secret for Loki to connect to azure container
func createSecretForAzureContainer(oc *exutil.CLI, bucketName, secretName, ns string) error {
	environment := "AzureGlobal"
	cloudName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("infrastructure", "cluster", "-o=jsonpath={.status.platformStatus.azure.cloudName}").Output()
	if err != nil {
		return fmt.Errorf("can't get azure cluster type  %w", err)
	}
	if strings.ToLower(cloudName) == "azureusgovernmentcloud" {
		environment = "AzureUSGovernment"
	}
	if strings.ToLower(cloudName) == "azurechinacloud" {
		environment = "AzureChinaCloud"
	}
	if strings.ToLower(cloudName) == "azuregermancloud" {
		environment = "AzureGermanCloud"
	}

	accountName, accountKey, err1 := compat_otp.GetAzureStorageAccountFromCluster(oc)
	if err1 != nil {
		return fmt.Errorf("can't get azure storage account from cluster: %w", err1)
	}
	return oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", "-n", ns, secretName, "--from-literal=environment="+environment, "--from-literal=container="+bucketName, "--from-literal=account_name="+accountName, "--from-literal=account_key="+accountKey).Execute()
}

func createSecretForSwiftContainer(oc *exutil.CLI, containerName, secretName, ns string, cred *compat_otp.OpenstackCredentials) error {
	userID, domainID := compat_otp.GetOpenStackUserIDAndDomainID(cred)
	err := oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", "-n", ns, secretName,
		"--from-literal=auth_url="+cred.Clouds.Openstack.Auth.AuthURL,
		"--from-literal=username="+cred.Clouds.Openstack.Auth.Username,
		"--from-literal=user_domain_name="+cred.Clouds.Openstack.Auth.UserDomainName,
		"--from-literal=user_domain_id="+domainID,
		"--from-literal=user_id="+userID,
		"--from-literal=password="+cred.Clouds.Openstack.Auth.Password,
		"--from-literal=domain_id="+domainID,
		"--from-literal=domain_name="+cred.Clouds.Openstack.Auth.UserDomainName,
		"--from-literal=container_name="+containerName,
		"--from-literal=project_id="+cred.Clouds.Openstack.Auth.ProjectID,
		"--from-literal=project_name="+cred.Clouds.Openstack.Auth.ProjectName,
		"--from-literal=project_domain_id="+domainID,
		"--from-literal=project_domain_name="+cred.Clouds.Openstack.Auth.UserDomainName).Execute()
	return err
}

// checkODF check if the ODF is installed in the cluster or not
// here only checks the sc/ocs-storagecluster-ceph-rbd and svc/s3
func checkODF(oc *exutil.CLI) bool {
	svcFound := false
	expectedSC := []string{"openshift-storage.noobaa.io"}
	var scInCluster []string
	scs, err := oc.AdminKubeClient().StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())

	for _, sc := range scs.Items {
		scInCluster = append(scInCluster, sc.Name)
	}

	for _, s := range expectedSC {
		if !contain(scInCluster, s) {
			return false
		}
	}

	_, err = oc.AdminKubeClient().CoreV1().Services("openshift-storage").Get(context.Background(), "s3", metav1.GetOptions{})
	if err == nil {
		svcFound = true
	}
	return svcFound
}

func createObjectBucketClaim(oc *exutil.CLI, ns, name string) error {
	template, _ := filePath.Abs("testdata/logging/odf/objectBucketClaim.yaml")
	obc := Resource{"objectbucketclaims", name, ns}

	err := obc.applyFromTemplate(oc, "-f", template, "-n", ns, "-p", "NAME="+name, "NAMESPACE="+ns)
	if err != nil {
		return err
	}
	_ = obc.WaitForResourceToAppear(oc)
	_ = Resource{"objectbuckets", "obc-" + ns + "-" + name, ns}.WaitForResourceToAppear(oc)
	assertResourceStatus(oc, "objectbucketclaims", name, ns, "{.status.phase}", "Bound")
	return nil
}

func deleteObjectBucketClaim(oc *exutil.CLI, ns, name string) error {
	obc := Resource{"objectbucketclaims", name, ns}
	err := obc.clear(oc)
	if err != nil {
		return err
	}
	return obc.WaitUntilResourceIsGone(oc)
}

// checkMinIO
func checkMinIO(oc *exutil.CLI, ns string) (bool, error) {
	podReady, svcFound := false, false
	pod, err := oc.AdminKubeClient().CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{LabelSelector: "app=minio"})
	if err != nil {
		return false, err
	}
	if len(pod.Items) > 0 && pod.Items[0].Status.Phase == "Running" {
		podReady = true
	}
	_, err = oc.AdminKubeClient().CoreV1().Services(ns).Get(context.Background(), "minio", metav1.GetOptions{})
	if err == nil {
		svcFound = true
	}
	return podReady && svcFound, err
}

func useExtraObjectStorage(oc *exutil.CLI) string {
	if checkODF(oc) {
		e2e.Logf("use the existing ODF storage service")
		return "odf"
	}
	ready, err := checkMinIO(oc, minioNS)
	if ready {
		e2e.Logf("use existing MinIO storage service")
		return "minio"
	}
	if strings.Contains(err.Error(), "No resources found") || strings.Contains(err.Error(), "not found") {
		e2e.Logf("deploy MinIO and use this MinIO as storage service")
		deployMinIO(oc)
		return "minio"
	}
	return ""
}

func patchLokiOperatorWithAWSRoleArn(oc *exutil.CLI, subNamespace, roleArn string) {
	roleArnPatchConfig := `{
		"spec": {
		  "config": {
			"env": [
			  {
				"name": "ROLEARN",
				"value": "%s"
			  }
			]
		  }
		}
	  }`

	subName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", "-n", subNamespace, `-ojsonpath={.items[?(@.spec.name=="loki-operator")].metadata.name}`).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(subName).ShouldNot(o.BeEmpty())
	err = oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("patch").Args("sub", subName, "-n", subNamespace, "-p", fmt.Sprintf(roleArnPatchConfig, roleArn), "--type=merge").Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	WaitForPodsReadyWithLabel(oc, loNS, "name=loki-operator-controller-manager")
}

// return the storage type per different platform
func getStorageType(oc *exutil.CLI) string {
	platform := compat_otp.CheckPlatform(oc)
	switch platform {
	case "aws":
		{
			return "s3"
		}
	case "gcp":
		{
			return "gcs"
		}
	case "azure":
		{
			return "azure"
		}
	case "openstack":
		{
			return "swift"
		}
	default:
		{
			return useExtraObjectStorage(oc)
		}
	}
}

// initialize a s3 client with credential
func newS3Client(cfg aws.Config) *s3.Client {
	return s3.NewFromConfig(cfg)
}

func getStorageClassName(oc *exutil.CLI) (string, error) {
	scs, err := oc.AdminKubeClient().StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	if len(scs.Items) == 0 {
		return "", fmt.Errorf("there is no storageclass in the cluster")
	}
	for _, sc := range scs.Items {
		if sc.ObjectMeta.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}
	return scs.Items[0].Name, nil
}

// prepareResourcesForLokiStack creates buckets/containers in backend storage provider, and creates the secret for Loki to use
func (l lokiStack) prepareResourcesForLokiStack(oc *exutil.CLI) error {
	var err error
	if len(l.BucketName) == 0 {
		return fmt.Errorf("the bucketName should not be empty")
	}
	switch l.StorageType {
	case "s3":
		{
			var cfg aws.Config
			region, err := compat_otp.GetAWSClusterRegion(oc)
			if err != nil {
				return err
			}
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				if !checkAWSCredentials() {
					g.Skip("Skip since no AWS credetial! No Env AWS_SHARED_CREDENTIALS_FILE, Env CLUSTER_PROFILE_DIR  or $HOME/.aws/credentials file")
				}
				partition := "aws"
				if strings.HasPrefix(region, "us-gov") {
					partition = "aws-us-gov"
				}
				cfg = readDefaultSDKExternalConfigurations(context.TODO(), region)
				iamClient := newIamClient(cfg)
				stsClient := newStsClient(cfg)
				awsAccountID, _ := getAwsAccount(stsClient)
				oidcName, err := getOIDC(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				lokiIAMRoleName := l.Name + "-" + compat_otp.GetRandomString()
				roleArn := createIAMRoleForLokiSTSDeployment(iamClient, oidcName, awsAccountID, partition, l.Namespace, l.Name, lokiIAMRoleName)
				os.Setenv("LOKI_ROLE_NAME_ON_STS", lokiIAMRoleName)
				patchLokiOperatorWithAWSRoleArn(oc, loNS, roleArn)
				createObjectStorageSecretOnAWSSTSCluster(oc, region, l.StorageSecret, l.BucketName, l.Namespace)
			} else {
				cred := getAWSCredentialFromCluster(oc)
				cfg = generateS3Config(cred)
				err = createSecretForAWSS3Bucket(oc, l.BucketName, l.StorageSecret, l.Namespace, cred)
				o.Expect(err).NotTo(o.HaveOccurred())
			}
			client := newS3Client(cfg)
			err = createS3Bucket(client, l.BucketName, region)
			if err != nil {
				return err
			}
		}
	case "azure":
		{
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				if !readAzureCredentials() {
					g.Skip("Azure Credentials not found. Skip case!")
				} else {
					performManagedIdentityAndSecretSetupForAzureWIF(oc, l.Name, l.Namespace, l.BucketName, l.StorageSecret)
				}
			} else {
				accountName, accountKey, err1 := compat_otp.GetAzureStorageAccountFromCluster(oc)
				if err1 != nil {
					return fmt.Errorf("can't get azure storage account from cluster: %w", err1)
				}
				client, err2 := compat_otp.NewAzureContainerClient(oc, accountName, accountKey, l.BucketName)
				if err2 != nil {
					return err2
				}
				err = compat_otp.CreateAzureStorageBlobContainer(client)
				if err != nil {
					return err
				}
				err = createSecretForAzureContainer(oc, l.BucketName, l.StorageSecret, l.Namespace)
			}
		}
	case "gcs":
		{
			projectID, errGetID := compat_otp.GetGcpProjectID(oc)
			o.Expect(errGetID).NotTo(o.HaveOccurred())
			err = compat_otp.CreateGCSBucket(projectID, l.BucketName)
			if err != nil {
				return err
			}
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				clusterName := getInfrastructureName(oc)
				gcsSAName := generateServiceAccountNameForGCS(clusterName)
				os.Setenv("LOGGING_GCS_SERVICE_ACCOUNT_NAME", gcsSAName)
				projectNumber, err1 := getGCPProjectNumber(projectID)
				if err1 != nil {
					return fmt.Errorf("can't get GCP project number: %w", err1)
				}
				poolID, err2 := getPoolID(oc)
				if err2 != nil {
					return fmt.Errorf("can't get pool ID: %w", err2)
				}
				sa, err3 := createServiceAccountOnGCP(projectID, gcsSAName)
				if err3 != nil {
					return fmt.Errorf("can't create service account: %w", err3)
				}
				os.Setenv("LOGGING_GCS_SERVICE_ACCOUNT_EMAIL", sa.Email)
				err4 := grantPermissionsToGCPServiceAccount(poolID, projectID, projectNumber, l.Namespace, l.Name, sa.Email)
				if err4 != nil {
					return fmt.Errorf("can't add roles to the serviceaccount: %w", err4)
				}

				patchLokiOperatorOnGCPSTSforCCO(oc, loNS, projectNumber, poolID, sa.Email)

				err = createSecretForGCSBucketWithSTS(oc, l.Namespace, l.StorageSecret, l.BucketName)
			} else {
				err = createSecretForGCSBucket(oc, l.BucketName, l.StorageSecret, l.Namespace)
			}
		}
	case "swift":
		{
			cred, err1 := compat_otp.GetOpenStackCredentials(oc)
			o.Expect(err1).NotTo(o.HaveOccurred())
			client := compat_otp.NewOpenStackClient(cred, "object-store")
			err = compat_otp.CreateOpenStackContainer(client, l.BucketName)
			if err != nil {
				return err
			}
			err = createSecretForSwiftContainer(oc, l.BucketName, l.StorageSecret, l.Namespace, cred)
		}
	case "odf":
		{
			err = createObjectBucketClaim(oc, l.Namespace, l.BucketName)
			if err != nil {
				return err
			}
			err = createSecretForODFBucket(oc, l.BucketName, l.StorageSecret, l.Namespace)
		}
	case "minio":
		{
			cred := getMinIOCreds(oc, minioNS)
			cfg := generateS3Config(cred)
			client := newS3Client(cfg)
			err = createS3Bucket(client, l.BucketName, "")
			if err != nil {
				return err
			}
			err = createSecretForMinIOBucket(oc, l.BucketName, l.StorageSecret, l.Namespace, cred)
		}
	}
	return err
}

func (l lokiStack) removeObjectStorage(oc *exutil.CLI) {
	e2e.Logf("Remove Object Storage")
	_ = Resource{"secret", l.StorageSecret, l.Namespace}.clear(oc)
	var err error
	switch l.StorageType {
	case "s3":
		{
			var cfg aws.Config
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				region, err := compat_otp.GetAWSClusterRegion(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				cfg = readDefaultSDKExternalConfigurations(context.TODO(), region)
				iamClient := newIamClient(cfg)
				deleteIAMroleonAWS(iamClient, os.Getenv("LOKI_ROLE_NAME_ON_STS"))
				os.Unsetenv("LOKI_ROLE_NAME_ON_STS")
				subName, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("sub", "-n", loNS, `-ojsonpath={.items[?(@.spec.name=="loki-operator")].metadata.name}`).Output()
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(subName).ShouldNot(o.BeEmpty())
				err = oc.AsAdmin().WithoutNamespace().Run("patch").Args("sub", subName, "-n", loNS, "-p", `[{"op": "remove", "path": "/spec/config"}]`, "--type=json").Execute()
				o.Expect(err).NotTo(o.HaveOccurred())
				WaitForPodsReadyWithLabel(oc, loNS, "name=loki-operator-controller-manager")
			} else {
				cred := getAWSCredentialFromCluster(oc)
				cfg = generateS3Config(cred)
			}
			client := newS3Client(cfg)
			err = deleteS3Bucket(client, l.BucketName)
		}
	case "azure":
		{
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				resourceGroup, err := getAzureResourceGroupFromCluster(oc)
				o.Expect(err).NotTo(o.HaveOccurred())
				azureSubscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
				cred := createNewDefaultAzureCredential()
				deleteManagedIdentityOnAzure(cred, azureSubscriptionID, resourceGroup, l.Name)
				deleteAzureStorageAccount(cred, azureSubscriptionID, resourceGroup, os.Getenv("LOKI_OBJECT_STORAGE_STORAGE_ACCOUNT"))
				os.Unsetenv("LOKI_OBJECT_STORAGE_STORAGE_ACCOUNT")
			} else {
				accountName, accountKey, err1 := compat_otp.GetAzureStorageAccountFromCluster(oc)
				o.Expect(err1).NotTo(o.HaveOccurred())
				client, err2 := compat_otp.NewAzureContainerClient(oc, accountName, accountKey, l.BucketName)
				o.Expect(err2).NotTo(o.HaveOccurred())
				err = compat_otp.DeleteAzureStorageBlobContainer(client)
			}
		}
	case "gcs":
		{
			if compat_otp.IsWorkloadIdentityCluster(oc) {
				sa := os.Getenv("LOGGING_GCS_SERVICE_ACCOUNT_NAME")
				if sa == "" {
					e2e.Logf("LOGGING_GCS_SERVICE_ACCOUNT_NAME is not set, no need to delete the serviceaccount")
				} else {
					os.Unsetenv("LOGGING_GCS_SERVICE_ACCOUNT_NAME")
					email := os.Getenv("LOGGING_GCS_SERVICE_ACCOUNT_EMAIL")
					if email == "" {
						e2e.Logf("LOGGING_GCS_SERVICE_ACCOUNT_EMAIL is not set, no need to delete the policies")
					} else {
						os.Unsetenv("LOGGING_GCS_SERVICE_ACCOUNT_EMAIL")
						projectID, errGetID := compat_otp.GetGcpProjectID(oc)
						o.Expect(errGetID).NotTo(o.HaveOccurred())
						projectNumber, _ := getGCPProjectNumber(projectID)
						poolID, _ := getPoolID(oc)
						err = removePermissionsFromGCPServiceAccount(poolID, projectID, projectNumber, l.Namespace, l.Name, email)
						o.Expect(err).NotTo(o.HaveOccurred())
						err = removeServiceAccountFromGCP("projects/" + projectID + "/serviceAccounts/" + email)
						o.Expect(err).NotTo(o.HaveOccurred())
					}
				}
			}
			err = compat_otp.DeleteGCSBucket(l.BucketName)
		}
	case "swift":
		{
			cred, err1 := compat_otp.GetOpenStackCredentials(oc)
			o.Expect(err1).NotTo(o.HaveOccurred())
			client := compat_otp.NewOpenStackClient(cred, "object-store")
			err = compat_otp.DeleteOpenStackContainer(client, l.BucketName)
		}
	case "odf":
		{
			err = deleteObjectBucketClaim(oc, l.Namespace, l.BucketName)
		}
	case "minio":
		{
			cred := getMinIOCreds(oc, minioNS)
			cfg := generateS3Config(cred)
			client := newS3Client(cfg)
			err = deleteS3Bucket(client, l.BucketName)
		}
	}
	o.Expect(err).NotTo(o.HaveOccurred())
}

func deployMinIO(oc *exutil.CLI) {
	// create namespace
	_, err := oc.AdminKubeClient().CoreV1().Namespaces().Get(context.Background(), minioNS, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("namespace", minioNS).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	}
	// create secret
	_, err = oc.AdminKubeClient().CoreV1().Secrets(minioNS).Get(context.Background(), minioSecret, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		err = oc.AsAdmin().WithoutNamespace().Run("create").Args("secret", "generic", minioSecret, "-n", minioNS, "--from-literal=access_key_id="+getRandomString(), "--from-literal=secret_access_key=passwOOrd"+getRandomString()).Execute()
		o.Expect(err).NotTo(o.HaveOccurred())
	}
	// deploy minIO
	deployTemplate, _ := filePath.Abs("testdata/logging/minIO/deploy.yaml")
	deployFile, err := processTemplate(oc, "-n", minioNS, "-f", deployTemplate, "-p", "NAMESPACE="+minioNS, "NAME=minio", "SECRET_NAME="+minioSecret)
	defer os.Remove(deployFile)
	o.Expect(err).NotTo(o.HaveOccurred())
	err = oc.AsAdmin().Run("apply").Args("-f", deployFile, "-n", minioNS).Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	// wait for minio to be ready
	for _, rs := range []string{"deployment", "svc", "route"} {
		_ = Resource{rs, "minio", minioNS}.WaitForResourceToAppear(oc)
	}
	WaitForDeploymentPodsToBeReady(oc, minioNS, "minio")
}

func getPoolID(oc *exutil.CLI) (string, error) {
	// pool_id="$(oc get authentication cluster -o json | jq -r .spec.serviceAccountIssuer | sed 's/.*\/\([^\/]*\)-oidc/\1/')"
	issuer, err := getOIDC(oc)
	if err != nil {
		return "", err
	}

	return strings.Split(strings.Split(issuer, "/")[1], "-oidc")[0], nil
}

// delete the objects in the cluster
func (r Resource) clear(oc *exutil.CLI) error {
	msg, err := oc.AsAdmin().WithoutNamespace().Run("delete").Args("-n", r.Namespace, r.Kind, r.Name).Output()
	if err != nil {
		errstring := fmt.Sprintf("%v", msg)
		if strings.Contains(errstring, "NotFound") || strings.Contains(errstring, "the server doesn't have a resource type") {
			return nil
		}
		return err
	}
	err = r.WaitUntilResourceIsGone(oc)
	return err
}

// Patches Loki Operator running on a GCP WIF cluster. Operator is deployed with CCO mode after patching.
func patchLokiOperatorOnGCPSTSforCCO(oc *exutil.CLI, namespace string, projectNumber string, poolID string, serviceAccount string) {
	patchConfig := `{
    	"spec": {
        	"config": {
            	"env": [
               		{
                    	"name": "PROJECT_NUMBER",
                    	"value": "%s"
                	},
                	{
                    	"name": "POOL_ID",
                    	"value": "%s"
                	},
                	{
                    	"name": "PROVIDER_ID",
                    	"value": "%s"
                	},
                	{
                    	"name": "SERVICE_ACCOUNT_EMAIL",
                    	"value": "%s"
                	}
            	]
        	}
    	}
	}`

	err := oc.NotShowInfo().AsAdmin().WithoutNamespace().Run("patch").Args("sub", "loki-operator", "-n", namespace, "-p", fmt.Sprintf(patchConfig, projectNumber, poolID, poolID, serviceAccount), "--type=merge").Execute()
	o.Expect(err).NotTo(o.HaveOccurred())
	WaitForPodsReadyWithLabel(oc, loNS, "name=loki-operator-controller-manager")
}
