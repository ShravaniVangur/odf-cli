# Design: StorageClient Auto-Discovery for CephFS Commands

**Date:** 2026-04-01
**Status:** Approved

## Overview

Add `--storageclient` flag to `subvolume` and `cephfs-snap` commands that automatically discovers and configures all CephFS connection parameters from a StorageClient resource. This eliminates the need for users to manually specify pod names, credentials, namespace settings, and other configuration when working with external storage clusters.

## Goals

1. Add `--storageclient <name>` flag to both `subvolume` and `cephfs-snap` commands
2. Automatically discover and populate all configuration parameters:
   - Subvolume group name
   - RADOS namespace
   - CephFS controller pod (name, namespace, container)
   - Monitor IP
   - User credentials (userID, userKey)
3. Ensure mutual exclusivity with manual configuration flags
4. Provide clear error messages for each discovery step
5. Maintain consistency across both commands using shared discovery logic

## Non-Goals

- Auto-discovery for other commands beyond `subvolume` and `cephfs-snap`
- Support for multiple StorageClients simultaneously
- Caching or persistence of discovered values
- Interactive selection of pods when multiple exist
- Backward compatibility shims for old configuration patterns

## Architecture

### New Package: `pkg/storageclient`

**File:** `pkg/storageclient/discover.go`

Contains:
- `DiscoveredConfig` struct - holds all auto-discovered parameters
- `Discover()` function - orchestrates the complete discovery flow
- Internal helper functions for each discovery step

**Discovery Strategy:** Use `kubectl`/`oc` CLI commands via `exec.Command()` to query Kubernetes resources. This approach:
- Simplifies handling of custom resources (StorageClient CRD)
- Leverages existing kubeconfig and context management
- Provides consistent JSON output parsing
- Avoids need for typed clients for all CRDs

### Modified Packages

**`cmd/odf/subvolume/subvolume.go`**
- Add `--storageclient` persistent flag
- Add mutual exclusivity validation
- Call discovery and use results when flag is set

**`cmd/odf/cephfs-snap/cephfs_snap.go`**
- Add `--storageclient` persistent flag
- Add mutual exclusivity validation
- Call discovery and use results when flag is set

## Data Structures

### DiscoveredConfig

```go
package storageclient

// DiscoveredConfig holds all auto-discovered configuration parameters
type DiscoveredConfig struct {
    SubvolumeGroup   string // CephFS subvolume group name
    RadosNamespace   string // RADOS namespace for OMAP operations
    PodName          string // CephFS controller pod name
    PodNamespace     string // Namespace of the controller pod
    PodContainer     string // Container name (always "csi-cephfsplugin")
    MonitorIP        string // First Ceph monitor IP with port
    UserID           string // Ceph user ID (base64 decoded)
    UserKey          string // Ceph user key (base64 decoded)
}
```

### Discovery Function Signature

```go
// Discover queries Kubernetes resources to auto-discover CephFS configuration
// from a StorageClient resource.
//
// Parameters:
//   - ctx: Context for command execution
//   - storageClientName: Name of the StorageClient resource
//   - namespace: Namespace to query all resources in
//
// Returns:
//   - *DiscoveredConfig: Complete configuration if discovery succeeds
//   - error: Detailed error message indicating which step failed
func Discover(ctx context.Context, storageClientName, namespace string) (*DiscoveredConfig, error)
```

## Discovery Flow

The `Discover()` function performs these steps sequentially, failing immediately on any error:

### Step 1: Get StorageClient

**Command:**
```bash
kubectl get storageclient {name} -n {namespace} -o json
```

**Extract:**
- `status.id` field (cluster identifier)

**Validation:**
- Resource exists
- `status.id` is not empty

**Error Examples:**
- `Error: StorageClient 'client-1' not found in namespace 'openshift-storage'`
- `Error: StorageClient 'client-1' has no status.id (cluster not ready)`

### Step 2: Get ceph-csi-config ConfigMap

**Command:**
```bash
kubectl get configmap ceph-csi-config -n {namespace} -o json
```

**Parse:**
- `data["config.json"]` - JSON array of cluster configurations
- Find entry where `clusterID == status.id` from Step 1

**Extract from matched entry:**
- `cephFS.subvolumeGroup`
- `cephFS.radosNamespace`
- `cephFS.controllerPublishSecretRef.name`
- `cephFS.controllerPublishSecretRef.namespace`
- `monitors[0]` (first monitor IP:port)

**Validation:**
- ConfigMap exists
- `config.json` is valid JSON
- Matching clusterID entry exists
- `cephFS` section exists in matched entry

**Error Examples:**
- `Error: ConfigMap 'ceph-csi-config' not found in namespace 'openshift-storage'`
- `Error: Failed to parse config.json in ceph-csi-config: invalid JSON`
- `Error: No matching clusterID 'b2c3c246-f943-46f3-86e3-90df95519185' found in ceph-csi-config`
- `Error: ClusterID 'b2c3c246-f943-46f3-86e3-90df95519185' has no cephFS configuration in ceph-csi-config`

### Step 3: Get Secret

**Command:**
```bash
kubectl get secret {secretName} -n {secretNamespace} -o json
```

**Extract:**
- `data["userID"]` - base64 decode
- `data["userKey"]` - base64 decode

**Validation:**
- Secret exists
- Both `userID` and `userKey` fields exist
- Base64 decoding succeeds

**Error Examples:**
- `Error: Secret 'csi-cephfs-provisioner-b2c3c246-f943-46f3-86e3-90df95519185' not found in namespace 'openshift-storage'`
- `Error: Secret 'csi-cephfs-provisioner-b2c3c246-f943-46f3-86e3-90df95519185' missing required field: userID`
- `Error: Failed to decode userKey from secret 'csi-cephfs-provisioner-b2c3c246-f943-46f3-86e3-90df95519185'`

### Step 4: Find CephFS Controller Pod

**Command:**
```bash
kubectl get pods -n {namespace} -l app=openshift-storage.cephfs.csi.ceph.com-ctrlplugin -o json
```

**Select:**
- Filter to Running state pods
- Use first running pod
- Set container name to `csi-cephfsplugin`

**Validation:**
- At least one pod exists with the label
- At least one pod is in Running state

**Error Examples:**
- `Error: No CephFS controller pods found with label app=openshift-storage.cephfs.csi.ceph.com-ctrlplugin in namespace 'openshift-storage'`
- `Error: Found 2 CephFS controller pod(s) but none are Running`

## Integration with Commands

### Flag Addition

Both commands add the flag in their `init()` function:

```go
cmd.PersistentFlags().String("storageclient", "", "StorageClient name for auto-discovery of configuration")
```

### Mutual Exclusivity

If `--storageclient` is provided, these flags must NOT be set:
- `--svg`
- `--rados-namespace`
- `--pod-name`
- `--pod-namespace`
- `--pod-container`
- `--mon-ip`
- `--user-id`
- `--user-key`

**Error Message:**
```
Error: --storageclient cannot be used with --svg, --rados-namespace, --pod-name, --pod-namespace, --pod-container, --mon-ip, --user-id, or --user-key flags
```

### Command Execution Flow

**For both `subvolume` and `cephfs-snap` commands:**

```go
Run: func(cmd *cobra.Command, args []string) {
    ctx := cmd.Context()
    storageClientName, _ := cmd.Flags().GetString("storageclient")

    var cfg *filesystem.CustomExecConfig
    var radosNamespace, svg string

    if storageClientName != "" {
        // Validate mutual exclusivity
        if err := validateNoConflictingFlags(cmd); err != nil {
            logging.Fatal(err)
        }

        // Auto-discover configuration
        discovered, err := storageclient.Discover(ctx, storageClientName, root.StorageClusterNamespace)
        if err != nil {
            logging.Fatal(err)
        }

        // Populate from discovery
        cfg = &filesystem.CustomExecConfig{
            PodName:      discovered.PodName,
            PodNamespace: discovered.PodNamespace,
            Container:    discovered.PodContainer,
            MonIP:        discovered.MonitorIP,
            UserID:       discovered.UserID,
            UserKey:      discovered.UserKey,
        }
        radosNamespace = discovered.RadosNamespace
        svg = discovered.SubvolumeGroup
    } else {
        // Use manual flags (existing code path)
        cfg, err = parseCustomExecConfig(cmd)
        if err != nil {
            logging.Fatal(err)
        }
        radosNamespace, _ = cmd.Flags().GetString("rados-namespace")
        svg, _ = cmd.Flags().GetString("svg")
    }

    // Initialize CephFilesystem with discovered or manual config
    cephFS := &filesystem.CephFilesystem{
        Ctx:               ctx,
        Clientsets:        root.ClientSets,
        OperatorNamespace: root.OperatorNamespace,
        ClusterNamespace:  root.StorageClusterNamespace,
        RadosNamespace:    radosNamespace,
        CustomExecConfig:  cfg,
    }

    // Execute command-specific logic...
}
```

### Helper Function: validateNoConflictingFlags

```go
// validateNoConflictingFlags checks that no manual config flags are set
// when using --storageclient
func validateNoConflictingFlags(cmd *cobra.Command) error {
    conflictingFlags := []string{
        "svg", "rados-namespace", "pod-name", "pod-namespace",
        "pod-container", "mon-ip", "user-id", "user-key",
    }

    for _, flagName := range conflictingFlags {
        if cmd.Flags().Changed(flagName) {
            return fmt.Errorf("--storageclient cannot be used with --%s flag", flagName)
        }
    }
    return nil
}
```

## Error Handling Strategy

**Principle:** Fail fast with clear, actionable error messages.

**Error Message Format:**
```
Error: {what went wrong} {context}
```

**Examples:**
- `Error: StorageClient 'client-1' not found in namespace 'openshift-storage'`
- `Error: No matching clusterID 'abc-123' found in ceph-csi-config`
- `Error: Secret 'csi-secret' missing required field: userKey`

**No Fallbacks:**
- Discovery failures are terminal - do not fall back to manual mode
- Users must fix the issue or use manual flags instead
- Each error clearly indicates which resource or field is problematic

## Testing Considerations

**Manual Testing Scenarios:**

1. **Happy Path**
   - StorageClient exists with valid status.id
   - ConfigMap has matching clusterID entry
   - Secret exists with userID and userKey
   - Running CephFS controller pod exists
   - Verify all values populated correctly

2. **StorageClient Errors**
   - StorageClient not found
   - StorageClient has no status.id

3. **ConfigMap Errors**
   - ConfigMap not found
   - Invalid JSON in config.json
   - No matching clusterID
   - Missing cephFS section

4. **Secret Errors**
   - Secret not found
   - Missing userID or userKey fields
   - Invalid base64 encoding

5. **Pod Errors**
   - No pods with label
   - Pods exist but none are Running

6. **Flag Conflicts**
   - `--storageclient` with `--svg`
   - `--storageclient` with `--pod-name`
   - `--storageclient` with multiple manual flags

7. **Cross-Cluster Scenarios**
   - Verify `--consumer-context` still works with `--storageclient`
   - Secret in different namespace than StorageClient

## Implementation Notes

**Command Execution:**
- Use `exec.Command("kubectl", ...)` or `exec.Command("oc", ...)`
- Prefer `kubectl` for broader compatibility
- Parse JSON output using `encoding/json`
- Capture stderr for error messages

**Namespace Handling:**
- All resources queried in `--namespace` flag value
- Exception: Secret may be in different namespace (use ConfigMap's secretRef.namespace)

**Monitor IP Selection:**
- Always use `monitors[0]` (first monitor in array)
- Include port number (e.g., "172.30.115.222:3300")

**Pod Container:**
- Always set to `csi-cephfsplugin` (hardcoded, not discovered)

**Context Awareness:**
- Discovery respects `--context` flag if set
- Commands inherit existing context handling from root

## Success Criteria

- [ ] `pkg/storageclient/discover.go` implements complete discovery flow
- [ ] Both `subvolume` and `cephfs-snap` commands support `--storageclient`
- [ ] Mutual exclusivity validation prevents flag conflicts
- [ ] All discovery errors fail immediately with clear messages
- [ ] Manual testing passes all scenarios
- [ ] Documentation updated with `--storageclient` flag usage
- [ ] No breaking changes to existing flag behavior

## Future Considerations

- Add discovery support to other commands that use CephFS credentials
- Consider caching discovered values for multiple command invocations
- Add `--dry-run` mode to show what would be discovered without executing
- Support for RBD-based storage (currently CephFS only)
