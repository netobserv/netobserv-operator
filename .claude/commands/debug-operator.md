# Debug NetObserv Operator Locally

This command prepares the environment for debugging the operator locally.

**Prerequisites:** You must have already set up the development environment using `/setup-dev-env` before running this command.

## Steps

### 1. Scale down the in-cluster operator

Scale down the operator running in the cluster to avoid conflicts:

```bash
kubectl scale deployment netobserv-controller-manager -n netobserv --replicas=0
```

### 2. Remove the validating webhook

Delete the webhook to allow free modification of the FlowCollector CR during local development:

```bash
kubectl delete validatingwebhookconfiguration netobserv-validating-webhook-configuration
```

### 3. Run the operator locally

Use the AskUserQuestion tool to ask which debugging method to use:

**Question:** "How do you want to run the operator locally?"
- **Header:** "Debug method"
- **Options:**
  1. "go run - Simple execution without debugging"
  2. "delve interactive - Terminal debugging with dlv commands"
  3. "delve headless + VSCode - UI debugging with breakpoints"

**Based on the user's selection, execute the corresponding command:**

#### Option 1: go run
```bash
go run ./main.go \
  -ebpf-agent-image=quay.io/netobserv/netobserv-ebpf-agent:main \
  -flowlogs-pipeline-image=quay.io/netobserv/flowlogs-pipeline:main \
  -console-plugin-image=quay.io/netobserv/network-observability-console-plugin:main \
  -namespace=netobserv
```

#### Option 2: delve interactive
```bash
dlv debug ./main.go -- \
  -ebpf-agent-image=quay.io/netobserv/netobserv-ebpf-agent:main \
  -flowlogs-pipeline-image=quay.io/netobserv/flowlogs-pipeline:main \
  -console-plugin-image=quay.io/netobserv/network-observability-console-plugin:main \
  -namespace=netobserv
```

After starting, inform the user they can use these delve commands:
- `break main.main` - set breakpoint
- `continue` - continue execution
- `next` - step to next line
- `print <variable>` - inspect variables

#### Option 3: delve headless + VSCode

**Step 1:** Start Delve in headless mode:
```bash
dlv debug ./main.go --headless --listen=:2345 --api-version=2 --accept-multiclient -- \
  -ebpf-agent-image=quay.io/netobserv/netobserv-ebpf-agent:main \
  -flowlogs-pipeline-image=quay.io/netobserv/flowlogs-pipeline:main \
  -console-plugin-image=quay.io/netobserv/network-observability-console-plugin:main \
  -namespace=netobserv
```

**Step 2:** Inform the user:

"Delve is now running in headless mode on port 2345. To connect from VSCode:
1. Open the Debug view (Ctrl+Shift+D / Cmd+Shift+D)
2. Select 'Connect to Delve (Operator Debug)' from the dropdown
3. Press F5 or click 'Start Debugging'

You can now set breakpoints in VSCode and debug the operator."

### 4. Example: Debug FlowCollector reconciliation

**To help the user get started with debugging:**

The main reconciliation logic is in `internal/controller/flowcollector/flowcollector_controller.go` (look for the `Reconcile` function around line 100-150).

**If using VSCode debugging:**
1. Open `internal/controller/flowcollector/flowcollector_controller.go`
2. Click the left margin next to the line number inside the `Reconcile` function to set a breakpoint
3. Trigger a reconciliation by running:
```bash
kubectl patch flowcollector cluster --type=merge -p '{"spec":{"processor":{"logLevel":"debug"}}}'
```

The debugger will pause at your breakpoint, allowing you to inspect the FlowCollector object and step through the reconciliation logic.

## Cleanup

When finished debugging:

1. Stop the local operator (Ctrl+C in terminal, or stop debugging in VSCode)

2. Scale the in-cluster operator back up:
```bash
kubectl scale deployment netobserv-controller-manager -n netobserv --replicas=1
```

**Note:** The validating webhook will be automatically recreated by the operator.
