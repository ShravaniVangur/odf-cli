# Subvolume Management

The `odf subvolume` command manages CephFS subvolumes and helps identify and clean up stale subvolumes.

## Overview

CephFS subvolumes created through Kubernetes PersistentVolumeClaims may become stale when the PVC or PV is deleted from Kubernetes but the underlying subvolume remains in the storage cluster.

## Commands

### ls - List Subvolumes

List all CephFS subvolumes in a subvolume group.

**Syntax:**
```bash
odf subvolume ls [flags]
```

**Flags:**
- `--stale` - List only stale subvolumes (not bound to PVs)
- `--svg <name>` - Subvolume group name (default: "csi")
- `--rados-namespace <namespace>` - RADOS namespace for OMAP operations (default: "csi")
- `--consumer-context <context>` - Kubernetes context for PV lookups (default: current context)

**Custom Pod Execution Flags** (optional):
- `--pod-name <name>` - Name of pod to execute commands in
- `--pod-namespace <namespace>` - Namespace of custom pod
- `--pod-container <container>` - Container name in custom pod
- `--mon-ip <ip>` - Monitor IP for Ceph connection
- `--user-id <id>` - Ceph user ID
- `--user-key <key>` - Ceph user key

**Examples:**

List all subvolumes:
```bash
odf subvolume ls
```

List only stale subvolumes:
```bash
odf subvolume ls --stale
```

List subvolumes in a specific subvolume group:
```bash
odf subvolume ls --svg custom-svg
```

List subvolumes with custom RADOS namespace:
```bash
odf subvolume ls --rados-namespace custom-namespace
```

List subvolumes using consumer context (for cross-cluster scenarios):
```bash
odf subvolume ls --consumer-context consumer-cluster --stale
```

### delete - Delete Stale Subvolume

Delete a CephFS subvolume.

**Syntax:**
```bash
odf subvolume delete <filesystem> <subvolume> <subvolumegroup> [flags]
```

**Arguments:**
- `<filesystem>` - CephFS filesystem name
- `<subvolume>` - Subvolume name to delete
- `<subvolumegroup>` - Subvolume group name

**Flags:**
- `--rados-namespace <namespace>` - RADOS namespace for OMAP operations (default: "csi")
- `--consumer-context <context>` - Kubernetes context for PV lookups (default: current context)

Custom pod execution flags are also supported (see ls command above).

**Examples:**

Delete a stale subvolume:
```bash
odf subvolume delete myfs csi-vol-aa0099b5-f7a0-49c2-bc97-a810005a9654 csi
```

Delete subvolume with custom RADOS namespace:
```bash
odf subvolume delete myfs csi-vol-aa0099b5-f7a0-49c2-bc97-a810005a9654 csi \
  --rados-namespace custom-namespace
```

## Custom Pod Execution

For users who don't have access to the rook operator pod, you can execute commands in any pod that has Ceph CLI tools installed:

```bash
odf subvolume ls \
  --pod-name csi-cephfsplugin-xyz123 \
  --pod-namespace openshift-storage \
  --pod-container csi-cephfsplugin \
  --mon-ip 10.0.0.1:6789 \
  --user-id admin \
  --user-key <ceph-key>
```

This is useful in scenarios where:
- You don't have cluster-admin privileges
- The rook operator pod is not accessible
- You need to run commands from a specific pod for networking reasons

## Cross-Cluster Scenarios

In stretched or external storage scenarios where PVs exist in a consumer cluster separate from the Ceph cluster, use the `--consumer-context` flag:

```bash
# List stale subvolumes checking consumer cluster for PVs
odf subvolume ls --stale --consumer-context consumer-cluster-context

# Delete subvolume using consumer context
odf subvolume delete myfs <subvolume> csi --consumer-context consumer-cluster-context
```

The `--consumer-context` flag tells odf-cli to look up PersistentVolumes in a different Kubernetes cluster context while executing Ceph commands in the default context.

## RADOS Namespace

The `--rados-namespace` flag specifies the RADOS namespace used for OMAP object and key lookups. This should match the configuration in your CephFS CSI ConfigMap.

Default: "csi"

If you've configured a custom RADOS namespace in your CSI driver, specify it with this flag:

```bash
odf subvolume ls --rados-namespace my-custom-namespace
```

## Workflow

Typical workflow for cleaning up stale subvolumes:

1. List stale subvolumes:
   ```bash
   odf subvolume ls --stale
   ```

2. Verify which subvolumes are safe to delete

3. Delete stale subvolumes:
   ```bash
   odf subvolume delete myfs <subvolume-name> csi
   ```

## Related Commands

- `odf cephfs-snap ls` - List CephFS snapshots
- `odf cephfs-snap delete` - Delete orphaned snapshots
