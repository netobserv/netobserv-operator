package e2etests

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
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
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"
)

// s3Credential defines the s3 credentials
type s3Credential struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string // the endpoint of s3 service
}

func getAWSCredentialFromCluster() s3Credential {
	region, err := getAWSClusterRegion()
	o.Expect(err).NotTo(o.HaveOccurred())

	secret, err := k8sClient.CoreV1().Secrets("kube-system").Get(context.Background(), "aws-creds", metav1.GetOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())

	cred := s3Credential{
		Region:          region,
		AccessKeyID:     string(secret.Data["aws_access_key_id"]),
		SecretAccessKey: string(secret.Data["aws_secret_access_key"]),
	}
	return cred
}

func getMinIOCreds(ns string) s3Credential {
	secret, err := k8sClient.CoreV1().Secrets(ns).Get(context.Background(), minioSecret, metav1.GetOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())

	endpoint := "https://" + getRouteAddress(ns, "minio")
	return s3Credential{
		Endpoint:        endpoint,
		AccessKeyID:     string(secret.Data["access_key_id"]),
		SecretAccessKey: string(secret.Data["secret_access_key"]),
	}
}

func generateS3Config(cred s3Credential) aws.Config {
	var err error
	var cfg aws.Config
	if len(cred.Endpoint) > 0 {
		// For ODF and Minio, they're deployed in OCP clusters
		// In some clusters, we can't connect it without proxy, here add proxy settings to s3 client when there has http_proxy or https_proxy in the env var
		httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
			tr.Proxy = http.ProxyFromEnvironment
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		})
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cred.AccessKeyID, cred.SecretAccessKey, "")),
			config.WithBaseEndpoint(cred.Endpoint),
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
	_, err = client.CreateBucket(context.TODO(), &s3.CreateBucketInput{Bucket: &bucketName, CreateBucketConfiguration: &s3types.CreateBucketConfiguration{LocationConstraint: s3types.BucketLocationConstraint(region)}})
	return err
}

// deleteS3Bucket deletes an S3 bucket with retry logic for transient errors
func deleteS3Bucket(client *s3.Client, bucketName string) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		// Remove all objects first
		err := emptyS3Bucket(client, bucketName)
		if err != nil {
			lastErr = err
			e2e.Logf("attempt %d/3: failed to empty S3 bucket %s: %v", attempt, bucketName, err)
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			continue
		}

		// Delete the bucket
		_, err = client.DeleteBucket(context.TODO(), &s3.DeleteBucketInput{Bucket: &bucketName})
		if err != nil {
			lastErr = err
			e2e.Logf("attempt %d/3: failed to delete S3 bucket %s: %v", attempt, bucketName, err)
			time.Sleep(time.Duration(attempt) * 5 * time.Second)
			continue
		}
		e2e.Logf("S3 Bucket %s is deleted", bucketName)
		return nil
	}
	return fmt.Errorf("failed to delete S3 bucket %s after 3 attempts: %v", bucketName, lastErr)
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
		objectIdentifiers := make([]s3types.ObjectIdentifier, len(objects.Contents))
		for i, object := range objects.Contents {
			objectIdentifiers[i] = s3types.ObjectIdentifier{Key: object.Key}
		}

		quiet := true
		_, err = client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: &bucketName,
			Delete: &s3types.Delete{
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
func createSecretForAWSS3Bucket(bucketName, secretName, ns string, cred s3Credential) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	endpoint := "https://s3." + cred.Region + ".amazonaws.com"

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			"access_key_id":     cred.AccessKeyID,
			"access_key_secret": cred.SecretAccessKey,
			"region":            cred.Region,
			"bucketnames":       bucketName,
			"endpoint":          endpoint,
		},
	}
	_, err := k8sClient.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

func createSecretForODFBucket(bucketName, secretName, ns string) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	ctx := context.Background()
	noobaaSecret, err := k8sClient.CoreV1().Secrets("openshift-storage").Get(ctx, "noobaa-admin", metav1.GetOptions{})
	if err != nil {
		return err
	}

	endpoint := "http://s3.openshift-storage.svc:80"

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
		Data: map[string][]byte{
			"access_key_id":     noobaaSecret.Data["AWS_ACCESS_KEY_ID"],
			"access_key_secret": noobaaSecret.Data["AWS_SECRET_ACCESS_KEY"],
		},
		StringData: map[string]string{
			"bucketnames": bucketName,
			"endpoint":    endpoint,
		},
	}
	_, err = k8sClient.CoreV1().Secrets(ns).Create(ctx, newSecret, metav1.CreateOptions{})
	return err
}

func createSecretForMinIOBucket(bucketName, secretName, ns string, cred s3Credential) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			"access_key_id":     cred.AccessKeyID,
			"access_key_secret": cred.SecretAccessKey,
			"bucketnames":       bucketName,
			"endpoint":          cred.Endpoint,
		},
	}
	_, err := k8sClient.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
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

func createSecretForGCSBucketWithSTS(namespace, secretName, bucketName string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		StringData: map[string]string{
			"bucketname": bucketName,
		},
	}
	_, err := k8sClient.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

// creates a secret for Loki to connect to gcs bucket
func createSecretForGCSBucket(bucketName, secretName, ns string) error {
	if len(secretName) == 0 {
		return fmt.Errorf("secret name shouldn't be empty")
	}

	// get gcp-credentials from env var GOOGLE_APPLICATION_CREDENTIALS
	gcsCred := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	keyData, err := os.ReadFile(gcsCred)
	if err != nil {
		return err
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: ns},
		Data:       map[string][]byte{"key.json": keyData},
		StringData: map[string]string{"bucketname": bucketName},
	}
	_, err = k8sClient.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

// creates a secret for Loki to connect to azure container
func createSecretForAzureContainer(bucketName, secretName, ns string) error {
	environment, _ := getStorageAccountURISuffixAndEnvForAzure()

	accountName, accountKey, err1 := getAzureStorageAccountFromCluster()
	if err1 != nil {
		return fmt.Errorf("can't get azure storage account from cluster: %w", err1)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			"environment":  environment,
			"container":    bucketName,
			"account_name": accountName,
			"account_key":  accountKey,
		},
	}
	_, err := k8sClient.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

func createSecretForSwiftContainer(containerName, secretName, ns string, cred *openstackCredentials) error {
	userID, domainID := getOpenStackUserIDAndDomainID(cred)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: ns,
		},
		StringData: map[string]string{
			"auth_url":            cred.Clouds.Openstack.Auth.AuthURL,
			"username":            cred.Clouds.Openstack.Auth.Username,
			"user_domain_name":    cred.Clouds.Openstack.Auth.UserDomainName,
			"user_domain_id":      domainID,
			"user_id":             userID,
			"password":            cred.Clouds.Openstack.Auth.Password,
			"domain_id":           domainID,
			"domain_name":         cred.Clouds.Openstack.Auth.UserDomainName,
			"container_name":      containerName,
			"project_id":          cred.Clouds.Openstack.Auth.ProjectID,
			"project_name":        cred.Clouds.Openstack.Auth.ProjectName,
			"project_domain_id":   domainID,
			"project_domain_name": cred.Clouds.Openstack.Auth.UserDomainName,
		},
	}
	_, err := k8sClient.CoreV1().Secrets(ns).Create(context.Background(), secret, metav1.CreateOptions{})
	return err
}

// checkODF check if the ODF is installed in the cluster or not
// here only checks the sc/ocs-storagecluster-ceph-rbd and svc/s3
func checkODF() bool {
	expectedSC := []string{"openshift-storage.noobaa.io"}

	scList, err := k8sClient.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	o.Expect(err).NotTo(o.HaveOccurred())

	var scInCluster []string
	for _, sc := range scList.Items {
		scInCluster = append(scInCluster, sc.Name)
	}
	for _, s := range expectedSC {
		if !contain(scInCluster, s) {
			return false
		}
	}

	_, err = k8sClient.CoreV1().Services("openshift-storage").Get(context.Background(), "s3", metav1.GetOptions{})
	return err == nil
}

func createObjectBucketClaim(ns, name string) error {
	template, _ := filePath.Abs("testdata/logging/odf/objectBucketClaim.yaml")
	obc := Resource{"objectbucketclaims", name, ns}

	err := obc.applyFromTemplate("-f", template, "-n", ns, "-p", "NAME="+name, "NAMESPACE="+ns)
	if err != nil {
		return err
	}
	_ = obc.WaitForResourceToAppear()
	_ = Resource{"objectbuckets", "obc-" + ns + "-" + name, ns}.WaitForResourceToAppear()
	assertResourceStatus("objectbucketclaims", name, ns, ".status.phase", "Bound")
	return nil
}

func deleteObjectBucketClaim(ns, name string) error {
	obc := Resource{"objectbucketclaims", name, ns}

	err := obc.clear()
	if err != nil {
		return err
	}
	return obc.WaitUntilResourceIsGone()
}

// checkMinIO
func checkMinIO(ns string) (bool, error) {
	podReady, svcFound := false, false

	pods, err := k8sClient.CoreV1().Pods(ns).List(context.Background(), metav1.ListOptions{LabelSelector: "app=minio"})
	if err != nil {
		return false, err
	}
	if len(pods.Items) == 0 {
		return false, fmt.Errorf("no resources found in %s namespace", ns)
	}
	if pods.Items[0].Status.Phase == corev1.PodRunning {
		podReady = true
	}

	_, err = k8sClient.CoreV1().Services(ns).Get(context.Background(), "minio", metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, fmt.Errorf("services \"minio\" not found")
		}
		return false, err
	}
	svcFound = true
	return podReady && svcFound, nil
}

func useExtraObjectStorage() string {
	if checkODF() {
		e2e.Logf("use the existing ODF storage service")
		return "odf"
	}
	ready, err := checkMinIO(minioNS)
	if ready {
		e2e.Logf("use existing MinIO storage service")
		return "minio"
	}
	if err != nil && (strings.Contains(err.Error(), "No resources found") || strings.Contains(err.Error(), "not found")) {
		e2e.Logf("deploy MinIO and use this MinIO as storage service")
		deployMinIO()
		return "minio"
	}
	return ""
}

func patchLokiOperatorWithAWSRoleArn(subNamespace, roleArn string) {
	// List subscriptions to find loki-operator
	subName, err := getSubscriptionByPackageName(subNamespace, "loki-operator")
	o.Expect(err).NotTo(o.HaveOccurred())
	o.Expect(subName).ShouldNot(o.BeEmpty())

	// Prepare patch
	patchData := fmt.Sprintf(`{
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
	}`, roleArn)

	err = patchDynamicResource("subscription", subName, subNamespace, types.MergePatchType, []byte(patchData))
	o.Expect(err).NotTo(o.HaveOccurred())

	WaitForPodsReadyWithLabel(loNS, "name=loki-operator-controller-manager")
}

// return the storage type per different platform
func getStorageType() string {
	platform := checkPlatform()
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
			return useExtraObjectStorage()
		}
	}
}

// initialize a s3 client with credential
func newS3Client(cfg aws.Config, usePathStyle ...bool) *s3.Client {
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if len(usePathStyle) > 0 && usePathStyle[0] {
			o.UsePathStyle = true
		}
	})
}

func getStorageClassName() (string, error) {
	scList, err := k8sClient.StorageV1().StorageClasses().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	if len(scList.Items) == 0 {
		return "", fmt.Errorf("there is no storageclass in the cluster")
	}

	// Find default storage class
	for _, sc := range scList.Items {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name, nil
		}
	}

	// Return first storage class if no default
	return scList.Items[0].Name, nil
}

// prepareResourcesForLokiStack creates buckets/containers in backend storage provider, and creates the secret for Loki to use
func (l lokiStack) prepareResourcesForLokiStack() error {
	var err error
	if len(l.BucketName) == 0 {
		return fmt.Errorf("the bucketName should not be empty")
	}
	switch l.StorageType {
	case "s3":
		{
			var cfg aws.Config
			region, err := getAWSClusterRegion()
			if err != nil {
				return err
			}
			if isWorkloadIdentityCluster() {
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
				oidcName, err := getOIDC()
				o.Expect(err).NotTo(o.HaveOccurred())
				lokiIAMRoleName := l.Name + "-" + getRandomString()
				roleArn := createIAMRoleForLokiSTSDeployment(iamClient, oidcName, awsAccountID, partition, l.Namespace, l.Name, lokiIAMRoleName)
				os.Setenv("LOKI_ROLE_NAME_ON_STS", lokiIAMRoleName)
				patchLokiOperatorWithAWSRoleArn(loNS, roleArn)
				createObjectStorageSecretOnAWSSTSCluster(region, l.StorageSecret, l.BucketName, l.Namespace)
			} else {
				cred := getAWSCredentialFromCluster()
				cfg = generateS3Config(cred)
				err = createSecretForAWSS3Bucket(l.BucketName, l.StorageSecret, l.Namespace, cred)
				o.Expect(err).NotTo(o.HaveOccurred())
			}
			s3Client := newS3Client(cfg)
			err = createS3Bucket(s3Client, l.BucketName, region)
			if err != nil {
				return err
			}
		}
	case "azure":
		{
			if isWorkloadIdentityCluster() {
				if !readAzureCredentials() {
					g.Skip("Azure Credentials not found. Skip case!")
				} else {
					performManagedIdentityAndSecretSetupForAzureWIF(l.Name, l.Namespace, l.BucketName, l.StorageSecret)
				}
			} else {
				accountName, accountKey, err1 := getAzureStorageAccountFromCluster()
				if err1 != nil {
					return fmt.Errorf("can't get azure storage account from cluster: %w", err1)
				}
				err = createAzureStorageBlobContainer(accountName, accountKey, l.BucketName)
				if err != nil {
					return err
				}
				err = createSecretForAzureContainer(l.BucketName, l.StorageSecret, l.Namespace)
			}
		}
	case "gcs":
		{
			projectID, errGetID := getGcpProjectID()
			o.Expect(errGetID).NotTo(o.HaveOccurred())
			err = createGCSBucket(projectID, l.BucketName)
			if err != nil {
				return err
			}
			if isWorkloadIdentityCluster() {
				clusterName := getInfrastructureName()
				gcsSAName := generateServiceAccountNameForGCS(clusterName)
				os.Setenv("LOGGING_GCS_SERVICE_ACCOUNT_NAME", gcsSAName)
				projectNumber, err1 := getGCPProjectNumber(projectID)
				if err1 != nil {
					return fmt.Errorf("can't get GCP project number: %w", err1)
				}
				poolID, err2 := getPoolID()
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

				patchLokiOperatorOnGCPSTSforCCO(loNS, projectNumber, poolID, sa.Email)

				err = createSecretForGCSBucketWithSTS(l.Namespace, l.StorageSecret, l.BucketName)
			} else {
				err = createSecretForGCSBucket(l.BucketName, l.StorageSecret, l.Namespace)
			}
		}
	case "swift":
		{
			cred, err1 := getOpenStackCredentials()
			o.Expect(err1).NotTo(o.HaveOccurred())
			swiftClient := newOpenStackClient(cred, "object-store")
			err = createOpenStackContainer(swiftClient, l.BucketName)
			if err != nil {
				return err
			}
			err = createSecretForSwiftContainer(l.BucketName, l.StorageSecret, l.Namespace, cred)
		}
	case "odf":
		{
			err = createObjectBucketClaim(l.Namespace, l.BucketName)
			if err != nil {
				return err
			}
			err = createSecretForODFBucket(l.BucketName, l.StorageSecret, l.Namespace)
		}
	case "minio":
		{
			cred := getMinIOCreds(minioNS)
			cfg := generateS3Config(cred)
			s3Client := newS3Client(cfg, true)
			err = createS3Bucket(s3Client, l.BucketName, "")
			if err != nil {
				return err
			}
			err = createSecretForMinIOBucket(l.BucketName, l.StorageSecret, l.Namespace, cred)
		}
	}
	return err
}

func (l lokiStack) removeObjectStorage() {
	e2e.Logf("Remove Object Storage")

	_ = Resource{"secret", l.StorageSecret, l.Namespace}.clear()
	var err error
	switch l.StorageType {
	case "s3":
		{
			var cfg aws.Config
			if isWorkloadIdentityCluster() {
				region, err := getAWSClusterRegion()
				o.Expect(err).NotTo(o.HaveOccurred())
				cfg = readDefaultSDKExternalConfigurations(context.TODO(), region)
				iamClient := newIamClient(cfg)
				deleteIAMroleonAWS(iamClient, os.Getenv("LOKI_ROLE_NAME_ON_STS"))
				os.Unsetenv("LOKI_ROLE_NAME_ON_STS")
				subName, err := getSubscriptionByPackageName(loNS, "loki-operator")
				o.Expect(err).NotTo(o.HaveOccurred())
				o.Expect(subName).ShouldNot(o.BeEmpty())
				err = patchSubscriptionRemoveConfig(loNS, subName)
				o.Expect(err).NotTo(o.HaveOccurred())
				WaitForPodsReadyWithLabel(loNS, "name=loki-operator-controller-manager")
			} else {
				cred := getAWSCredentialFromCluster()
				cfg = generateS3Config(cred)
			}
			client := newS3Client(cfg)
			err = deleteS3Bucket(client, l.BucketName)
		}
	case "azure":
		{
			if isWorkloadIdentityCluster() {
				resourceGroup, err := getAzureResourceGroupFromCluster()
				o.Expect(err).NotTo(o.HaveOccurred())
				azureSubscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
				cred := createNewDefaultAzureCredential()
				deleteManagedIdentityOnAzure(cred, azureSubscriptionID, resourceGroup, l.Name)
				deleteAzureStorageAccount(cred, azureSubscriptionID, resourceGroup, os.Getenv("LOKI_OBJECT_STORAGE_STORAGE_ACCOUNT"))
				os.Unsetenv("LOKI_OBJECT_STORAGE_STORAGE_ACCOUNT")
			} else {
				accountName, accountKey, err1 := getAzureStorageAccountFromCluster()
				o.Expect(err1).NotTo(o.HaveOccurred())
				err = deleteAzureStorageBlobContainer(accountName, accountKey, l.BucketName)
			}
		}
	case "gcs":
		{
			if isWorkloadIdentityCluster() {
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
						projectID, errGetID := getGcpProjectID()
						o.Expect(errGetID).NotTo(o.HaveOccurred())
						projectNumber, _ := getGCPProjectNumber(projectID)
						poolID, _ := getPoolID()
						err = removePermissionsFromGCPServiceAccount(poolID, projectID, projectNumber, l.Namespace, l.Name, email)
						o.Expect(err).NotTo(o.HaveOccurred())
						err = removeServiceAccountFromGCP("projects/" + projectID + "/serviceAccounts/" + email)
						o.Expect(err).NotTo(o.HaveOccurred())
					}
				}
			}
			err = deleteGCSBucket(l.BucketName)
		}
	case "swift":
		{
			cred, err1 := getOpenStackCredentials()
			o.Expect(err1).NotTo(o.HaveOccurred())
			swiftClient := newOpenStackClient(cred, "object-store")
			err = deleteOpenStackContainer(swiftClient, l.BucketName)
		}
	case "odf":
		{
			err = deleteObjectBucketClaim(l.Namespace, l.BucketName)
		}
	case "minio":
		{
			cred := getMinIOCreds(minioNS)
			cfg := generateS3Config(cred)
			client := newS3Client(cfg, true)
			err = deleteS3Bucket(client, l.BucketName)
		}
	}
	if err != nil {
		e2e.Logf("WARNING: failed to clean up object storage for bucket %s: %v", l.BucketName, err)
	}
}

func deployMinIO() {
	// create namespace
	_, err := k8sClient.CoreV1().Namespaces().Get(context.Background(), minioNS, metav1.GetOptions{})
	if err != nil {
		// Namespace doesn't exist, create it
		_ = createNamespace(minioNS)
	}

	// create secret
	_, err = k8sClient.CoreV1().Secrets(minioNS).Get(context.Background(), minioSecret, metav1.GetOptions{})
	if err != nil {
		// Secret doesn't exist, create it
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      minioSecret,
				Namespace: minioNS,
			},
			StringData: map[string]string{
				"access_key_id":     getRandomString(),
				"secret_access_key": "passwOOrd" + getRandomString(),
			},
		}
		_, err = k8sClient.CoreV1().Secrets(minioNS).Create(context.Background(), secret, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred())
	}

	// deploy minIO
	deployTemplate, _ := filePath.Abs("testdata/logging/minIO/deploy.yaml")
	err = applyNsResourceFromTemplateByAdmin(minioNS, "-f", deployTemplate, "-p", "NAMESPACE="+minioNS, "NAME=minio", "SECRET_NAME="+minioSecret)
	o.Expect(err).NotTo(o.HaveOccurred())

	// wait for minio to be ready
	for _, rs := range []string{"deployment", "svc", "route"} {
		_ = Resource{rs, "minio", minioNS}.WaitForResourceToAppear()
	}
	WaitForDeploymentPodsToBeReady(minioNS, "minio")
}

func getPoolID() (string, error) {
	// pool_id="$(oc get authentication cluster -o json | jq -r .spec.serviceAccountIssuer | sed 's/.*\/\([^\/]*\)-oidc/\1/')"
	issuer, err := getOIDC()
	if err != nil {
		return "", err
	}

	return strings.Split(strings.Split(issuer, "/")[1], "-oidc")[0], nil
}

// delete the objects in the cluster
func (r Resource) clear() error {
	err := deleteDynamicResource(r.Kind, r.Name, r.Namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return r.WaitUntilResourceIsGone()
}

// Patches Loki Operator running on a GCP WIF cluster. Operator is deployed with CCO mode after patching.
func patchLokiOperatorOnGCPSTSforCCO(namespace string, projectNumber string, poolID string, serviceAccount string) {
	patchData := fmt.Sprintf(`{
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
	}`, projectNumber, poolID, poolID, serviceAccount)

	err := patchDynamicResource("subscription", "loki-operator", namespace, types.MergePatchType, []byte(patchData))
	o.Expect(err).NotTo(o.HaveOccurred())

	WaitForPodsReadyWithLabel(loNS, "name=loki-operator-controller-manager")
}
