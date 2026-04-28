# Setup NetObserv Development Environment

This command sets up a complete development environment for NetObserv.

## Gather Configuration

First, use the AskUserQuestion tool to gather the necessary configuration:

**Question 1:** What platform are you using?
- Options: OpenShift, Kubernetes
- Header: "Platform"
- Store the answer to determine if `make set-release-kind-downstream` should be run in step 2.

**Question 2:** Are you using custom images?
- Options: "Yes, custom images", "No, use defaults (netobserv/main)"
- Header: "Images"

**If the user selected custom images**, ask follow-up questions to get:
- USER (quay.io username/repo, e.g., "leandroberetta")
- VERSION (image tag, e.g., "v1.2.3" or "main")

Set the variables:
- If using defaults: `USER=netobserv` and `VERSION=main`
- If using custom: `USER=<user_value>` and `VERSION=<version_value>`

## Steps

### 1. Deploy the operator

Run the deployment command with the configured values:

**If custom USER and VERSION:**
```bash
USER=<user_value> VERSION=<version_value> make deploy
```

**If using defaults:**
```bash
USER=netobserv make deploy
```

### 2. Configure for OpenShift (if applicable)

**Only run this if the user selected OpenShift** in the platform question:

```bash
make set-release-kind-downstream
```

Skip this step for Kubernetes.

### 3. Deploy Loki

```bash
make deploy-loki
```

### 4. Deploy sample FlowCollector CR

```bash
make deploy-sample-cr
```

### 5. Verify deployment

Run these commands to verify all components are running:

```bash
kubectl get pods -n netobserv
kubectl get flowcollector cluster -o yaml
```

## Cleanup

To remove the entire environment:

```bash
make undeploy
```
