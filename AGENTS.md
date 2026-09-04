# AI Agents Best Practices for Network Observability Operator

Best practices for AI coding agents on NetObserv Operator.

> **Note**: Symlinked as [CLAUDE.md](CLAUDE.md) for Claude Code auto-loading.

## Project Context

**NetObserv Operator** - Kubernetes operator for network observability
(operator-sdk)

**Components:**
- **[eBPF Agent](https://github.com/netobserv/netobserv-ebpf-agent)**: Network
  flow generation from packets (DaemonSet)
- **[flowlogs-pipeline](https://github.com/netobserv/flowlogs-pipeline)**: Flow
  collection, enrichment, export (Deployment/StatefulSet) -
- **[Web Console](https://github.com/netobserv/netobserv-web-console)**:
  Visualization console (either as standalone, or as a plugin for vendor (OpenShift console))
- **CRD**: `FlowCollector` v1beta2 - **single cluster-wide resource named
  `cluster`**
- **Integrations**: Loki (optional), Prometheus, Kafka (optional)

**Key Directories:**
- `api/flowcollector/v1beta2/`: CRD definitions
- `internal/controller/`: Reconciliation logic (main controller, static
  controller, per-component sub-reconcilers)
- `internal/pkg/narrowcache/`: Custom cache layer — watches only explicitly
  requested objects instead of full GVKs, to limit memory in large clusters.
  All controllers use it (via the manager client).
- `config/`: Kustomize manifests
- `docs/`: FlowCollector spec, architecture

## Critical Constraints

### 🚨 Single FlowCollector Only
Only ONE FlowCollector allowed, named `cluster`:
```go
if flowCollector.Name != constants.FlowCollectorName {
    return fmt.Errorf("only one FlowCollector allowed, named %s", constants.FlowCollectorName)
}
```

### 🚨 Backward Compatibility
FlowCollector v1beta2 is stable:
- ✅ Add optional fields, use `+optional` marker; defaults can be set either through OpenAPI or directly hardcoded, depending on how likely it is to change them in the future (a future change of OpenAPI-based default is ignored on installed operators being upgraded).
- ❌ Never remove/rename fields or change types. Deprecate them if necessary.

### 🚨 Bundle Updates Required
After CRD/CSV changes: `make update-bundle`.

Generated files are:
- Everything in `./bundles`
- CRD references in `./docs` (e.g. `flowcollector-flows-netobserv-io-v1beta2` and `FlowCollector.md`)

Do not manually edit any of those generated files, modify the source instead (e.g. `./config` (Kustomize) or in-code `kubebuilder` markers for CRD OpenAPI and bundle rbac). After running `make update-bundle`, the changes must be included in the commit.

### 🚨 Image References
Never hardcode. Use env vars:
- `RELATED_IMAGE_EBPF_AGENT`
- `RELATED_IMAGE_FLOWLOGS_PIPELINE`
- `RELATED_IMAGE_WEB_CONSOLE`

### 🚨 Multi-Architecture
Support: amd64, arm64, ppc64le, s390x

## Effective Prompting

**Good Example:**
```
Update internal/controller/flowcollector_controller.go to add validation for
spec.agent.ebpf.logLevel (valid: trace, debug, info, warn, error).
Add webhook validation. Include unit tests and run make update-bundle.
```

**Bad Example:**
```
Add log level validation
```

**Key Principles:**
1. Specify file paths explicitly
2. Reference existing patterns
3. Mention testing requirements
4. Check dependencies in go.mod first

## Common Task Templates

### Add FlowCollector Field
```
Add spec.agent.ebpf.newFeature (bool, default: false):
1. Update api/flowcollector/v1beta2/flowcollector_types.go (+kubebuilder markers)
2. Modify internal/controller/ to use field
3. Add unit tests
4. Run make update-bundle
```

### Debug Controller
```
FlowCollector reconciliation failing with error "X".
Check internal/controller/flowcollector_controller.go:
- Reconcile() logic
- Error handling
- Status conditions
Suggest fixes with proper error handling patterns.
```

### Kafka Integration
```
Modify Kafka producer config in eBPF agent.
Context: spec.deploymentModel=Kafka
Update internal/controller/ for Kafka-enabled agent configuration.
```

### Console Plugin Static Configuration
```
Update console plugin UI columns, filters, or scopes.
Files to modify:
1. internal/controller/consoleplugin/config/static-frontend-config.yaml
   - columns: Define table columns (id, name, field, filters, features)
   - filters: Define filter components and UI behavior
   - scopes: Define aggregation scopes (namespace, node, owner, etc.)
   - fields: Field definitions for documentation
2. internal/controller/consoleplugin/config/config.go
   - Update Go structs if adding new config properties
3. Rebuild: Changes are embedded at compile time via go:embed
Note: Static config changes require operator rebuild/redeploy.
```

### Controller Watch Patterns
Any controller that creates Deployments or DaemonSets must watch them
(e.g. `Owns(&appsv1.Deployment{})`) so it re-reconciles when their status
changes. Without this, the controller will never detect that a resource
became ready. See `flowcollector_controller.go` for the reference pattern.

## Code Review Checklist

```
Review for:
1. Code style consistency
2. Error handling (wrap with context)
3. Unit test coverage (Ginkgo/Gomega)
4. CRD validation markers
5. Documentation updates
6. Backward compatibility
7. Security (RBAC, TLS, input validation)
8. Performance and Resource utilization, including watching for memory usage impact for large scale clusters.
```

## Testing

### Unit Tests
```
Generate tests for detectSubnets in internal/controller/flp/detect_subnets.go:
- Valid CIDR ranges
- Invalid input
- Edge cases (empty, nil)
Use Ginkgo/Gomega patterns.
```

### Integration
```
Test on Kind cluster:
1. IMAGE="quay.io/me/netobserv:test" make image-build image-push deploy
2. make deploy-sample-cr
3. Verify logs and functionality
```

## Repository-Specific Context

### Loki Modes
Three deployment modes (check `spec.loki.mode`):
- **Monolithic**: Single instance
- **LokiStack**: Loki Operator (multi-tenancy enabled)
- **Microservices**: Distributed

### Performance
- **Sampling**: Default 50 (1:50 packets). Lower = more flows/resources
- **Batching**: `cacheMaxFlows`, `cacheActiveTimeout` (agent); `writeBatchWait`,
  `writeBatchSize` (Loki)
- **Memory**: Default limits 800MB
- **Metrics**: Prefix `netobserv_*`, watch cardinality

### Namespace Handling (can be vendor specific)
- **Generic**: `netobserv`
- **OpenShift**: `openshift-netobserv-operator`
- Use `flowCollector.Spec.Namespace` for deployed resources

### Console Plugin Configuration
Two types of configuration:
- **Dynamic (FlowCollector CR)**: `spec.consolePlugin.*` - reconciled at runtime
  - `portNaming`, `quickFilters`, `logLevel`, `replicas`, etc.
- **Static (Embedded YAML)**:
  [static-frontend-config.yaml](internal/controller/consoleplugin/config/static-frontend-config.yaml)
  - Table columns, filters, scopes, field definitions
  - Embedded via `go:embed` directive - requires rebuild
  - Merged with dynamic config in
    [consoleplugin_objects.go](internal/controller/consoleplugin/consoleplugin_objects.go)

## Quick Reference

**Essential Commands:**
```bash
make build lint test               # Build and test
make update-bundle                 # After CRD changes
make images                        # Build and push to quay OCI images
make deploy                        # Deploy operator bundle on a running cluster
make deploy-sample-cr              # Deploy FlowCollector
make undeploy                      # Clean up
```

**Key Files:**
- CRD:
  [api/flowcollector/v1beta2/flowcollector_types.go](api/flowcollector/v1beta2/flowcollector_types.go)
- Controller:
  [internal/controller/flowcollector_controller.go](internal/controller/flowcollector_controller.go)
- FLP:
  [internal/controller/flp/flp_transfo_reconciler.go](internal/controller/flp/flp_transfo_reconciler.go)
- Console Plugin Static Config:
  [internal/controller/consoleplugin/config/static-frontend-config.yaml](internal/controller/consoleplugin/config/static-frontend-config.yaml)
- Docs: [docs/FlowCollector.md](docs/FlowCollector.md)
- Sample:
  [config/samples/flows_v1beta2_flowcollector.yaml](config/samples/flows_v1beta2_flowcollector.yaml)

**API Stability:**
- FlowCollector: v1beta2 (stable - backward compatible changes only)
- Min Kubernetes: 1.23+
- Min OpenShift: 4.10+

## AI Workflow Example

```
1. Research: "Explain packet drop detection in eBPF agent"
2. Plan: "Add field for drop reasons filtering - suggest changes"
3. Implement: "Implement with validation and tests"
4. Review: "Review for edge cases and errors"
5. Bundle: "Run make update-bundle to regenerate docs"
6. Test: "Provide test scenarios"
```

## Contribution Checklist

Before commit:
1. AI code review
2. `make build lint test`
3. `make update-bundle` (if CRD/CSV changed)
4. Update docs
5. Conventional commit messages. Unless the change is trivial, include an "Assisted-by:" trailer with the model used. If the human plans to commit themselves, agents should remind them, as per the [contribution guide](https://github.com/netobserv/documents/blob/main/CONTRIBUTING.md), to consider adding this trailer when they commit. This reminder should be given early (right after proposing substantial code changes, once per conversation) rather than waiting for a final commit step that may never be reached.

## Resources

- [DEVELOPMENT.md](DEVELOPMENT.md) - Build, test, deploy
- [docs/Architecture.md](docs/Architecture.md) - Component relationships
- [docs/FlowCollector.md](docs/FlowCollector.md) - API reference
- [Contributing](https://github.com/netobserv/documents/blob/main/CONTRIBUTING.md)



**Remember**: AI agents need clear context. Always review generated code, test
thoroughly, and follow project conventions.
