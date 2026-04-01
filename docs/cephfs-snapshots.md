# CephFS Snapshot Management

The `odf cephfs-snap` command manages CephFS snapshots and identifies orphaned snapshots that have no corresponding Kubernetes VolumeSnapshotContent resource.

## Overview

CephFS snapshots created through Kubernetes VolumeSnapshots may become orphaned when the VolumeSnapshot or VolumeSnapshotContent resources are deleted from Kubernetes but the underlying CephFS snapshot remains in the storage cluster.

The `cephfs-snap` command helps you:
- List all CephFS snapshots with their status (bound or orphaned)
- Safely delete orphaned snapshots that are no longer referenced by Kubernetes
- Prevent accidental deletion of bound snapshots

## Commands

### ls - List Snapshots

List all CephFS snapshots in a subvolume group with their status.

**Syntax:**
```bash
odf cephfs-snap ls [flags]
```

**Flags:**
- `--filesystem <name>` - CephFS filesystem name (default: "myfs")
- `--svg <name>` - Subvolume group name (default: "csi")
- `--orphaned` - List only orphaned snapshots
- `--consumer-context <context>` - Kubernetes context for VolumeSnapshotContent lookups (default: current context)
- `--rados-namespace <namespace>` - RADOS namespace for OMAP operations (default: "csi")

**Custom Pod Execution Flags** (optional, for users without operator pod access):
- `--pod-name <name>` - Name of pod to execute commands in
- `--pod-namespace <namespace>` - Namespace of custom pod
- `--pod-container <container>` - Container name in custom pod
- `--mon-ip <ip>` - Monitor IP for Ceph connection
- `--user-id <id>` - Ceph user ID
- `--user-key <key>` - Ceph user key

**Examples:**

List all snapshots:
```bash
odf cephfs-snap ls --filesystem myfs
```

Output:
```
Filesystem  Subvolume                                     SubvolumeGroup  Snapshot                                       State
myfs        csi-vol-aa0099b5-f7a0-49c2-bc97-a810005a9654  csi             csi-snap-3936435c-a14a-4a76-9d0f-71321ac084a9  bound
myfs        csi-vol-bb1100c6-g8b1-50d3-cd08-b921016b0765  csi             csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0  orphaned
```

List only orphaned snapshots:
```bash
odf cephfs-snap ls --filesystem myfs --orphaned
```

Output:
```
Filesystem  Subvolume                                     SubvolumeGroup  Snapshot                                       State
myfs        csi-vol-bb1100c6-g8b1-50d3-cd08-b921016b0765  csi             csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0  orphaned
```

List snapshots in a specific subvolume group:
```bash
odf cephfs-snap ls --filesystem ocs-storagecluster-cephfilesystem --svg custom-svg
```

List snapshots using consumer context (for cross-cluster scenarios):
```bash
odf cephfs-snap ls --filesystem myfs --consumer-context consumer-cluster
```

### delete - Delete Orphaned Snapshot

Delete a CephFS snapshot. This command will only delete snapshots that are orphaned (not bound to a VolumeSnapshotContent).

**Syntax:**
```bash
odf cephfs-snap delete <subvolume> <snapshot> [flags]
```

**Arguments:**
- `<subvolume>` - Subvolume name containing the snapshot
- `<snapshot>` - Snapshot name to delete

**Flags:**
- `--filesystem <name>` - CephFS filesystem name (default: "myfs")
- `--svg <name>` - Subvolume group name (default: "csi")
- `--rados-namespace <namespace>` - RADOS namespace for OMAP operations (default: "csi")
- `--consumer-context <context>` - Kubernetes context for VolumeSnapshotContent lookups (default: current context)

Custom pod execution flags are also supported (see ls command above).

**Examples:**

Delete an orphaned snapshot:
```bash
odf cephfs-snap delete csi-vol-bb1100c6-g8b1-50d3-cd08-b921016b0765 \
  csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0 \
  --filesystem myfs
```

Output:
```
Info: Deleting the omap object and key for snapshot "csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0"
Info: omap object:"csi.snap.4047546d-b25b-5b87-0e1g-82432bd195b0" deleted
Info: omap key:"csi.snap.snapshot-4047546d-b25b-5b87-0e1g-82432bd195b0" deleted
snapshot csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0 deleted successfully
```

Attempt to delete a bound snapshot (will fail with error):
```bash
odf cephfs-snap delete csi-vol-aa0099b5-f7a0-49c2-bc97-a810005a9654 \
  csi-snap-3936435c-a14a-4a76-9d0f-71321ac084a9 \
  --filesystem myfs
```

Output:
```
Error: snapshot "csi-snap-3936435c-a14a-4a76-9d0f-71321ac084a9" is bound and cannot be deleted
```

Delete snapshot with custom RADOS namespace:
```bash
odf cephfs-snap delete csi-vol-bb1100c6-g8b1-50d3-cd08-b921016b0765 \
  csi-snap-4047546d-b25b-5b87-0e1g-82432bd195b0 \
  --filesystem myfs \
  --rados-namespace custom-namespace
```

## Safety Features

The `cephfs-snap delete` command includes safety checks:

1. **Bound snapshot protection**: Snapshots that are still referenced by a VolumeSnapshotContent resource cannot be deleted
2. **Verification before deletion**: The command verifies the snapshot exists and checks its status before attempting deletion
3. **OMAP cleanup**: Deletes both the snapshot and associated OMAP metadata

## Workflow

Typical workflow for cleaning up orphaned snapshots:

1. List orphaned snapshots:
   ```bash
   odf cephfs-snap ls --filesystem myfs --orphaned
   ```

2. Verify which snapshots are safe to delete

3. Delete orphaned snapshots one at a time:
   ```bash
   odf cephfs-snap delete <subvolume> <snapshot> --filesystem myfs
   ```

## Custom Pod Execution

For users who don't have access to the rook operator pod, you can execute commands in any pod that has Ceph CLI tools installed:

```bash
odf cephfs-snap ls \
  --filesystem myfs \
  --pod-name csi-cephfsplugin-xyz123 \
  --pod-namespace openshift-storage \
  --pod-container csi-cephfsplugin \
  --mon-ip 10.0.0.1:6789 \
  --user-id admin \
  --user-key <ceph-key>
```

## Cross-Cluster Scenarios

In stretched or external storage scenarios where PVs exist in a consumer cluster separate from the Ceph cluster:

```bash
# List snapshots checking consumer cluster for VolumeSnapshotContent
odf cephfs-snap ls \
  --filesystem myfs \
  --consumer-context consumer-cluster-context

# Delete snapshot using consumer context
odf cephfs-snap delete <subvolume> <snapshot> \
  --filesystem myfs \
  --consumer-context consumer-cluster-context
```

## Troubleshooting

**Problem:** Command fails with "operator pod not found"
**Solution:** Use custom pod execution flags to specify an alternative pod with Ceph CLI tools

**Problem:** Snapshot shows as "bound" but you deleted the VolumeSnapshot
**Solution:** Check if the VolumeSnapshotContent still exists in the consumer cluster context

**Problem:** Delete fails with OMAP errors
**Solution:** Verify the `--rados-namespace` matches the configuration in your CephFS CSI ConfigMap

## Related Commands

- `odf subvolume ls` - List CephFS subvolumes
- `odf subvolume delete` - Delete stale subvolumes
