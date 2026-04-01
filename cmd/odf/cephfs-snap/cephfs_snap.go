package cephfs_snap

import (
	"github.com/red-hat-storage/odf-cli/cmd/odf/root"
	"github.com/rook/kubectl-rook-ceph/pkg/filesystem"
	"github.com/rook/kubectl-rook-ceph/pkg/logging"
	"github.com/spf13/cobra"
)

// CephFSSnapCmd represents the cephfs-snap command
var CephFSSnapCmd = &cobra.Command{
	Use:   "cephfs-snap",
	Short: "Manages CephFS snapshots",
}

var listCmd = &cobra.Command{
	Use:     "ls",
	Short:   "Print the list of CephFS snapshots.",
	Example: "odf cephfs-snap ls --filesystem myfs",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		orphanedOnly, _ := cmd.Flags().GetBool("orphaned")
		svg, _ := cmd.Flags().GetString("svg")
		fs, _ := cmd.Flags().GetString("filesystem")

		cfg, err := parseCustomExecConfig(cmd)
		if err != nil {
			logging.Fatal(err)
		}

		cephFS := &filesystem.CephFilesystem{
			Ctx:               ctx,
			Clientsets:        root.ClientSets,
			OperatorNamespace: root.OperatorNamespace,
			ClusterNamespace:  root.StorageClusterNamespace,
			CustomExecConfig:  cfg,
		}

		cephFS.SnapshotList(svg, fs, orphanedOnly)
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   "Deletes a CephFS snapshot.",
	Args:    cobra.ExactArgs(2),
	Example: "odf cephfs-snap delete <subvolume> <snapshot> --filesystem myfs",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		subvol := args[0]
		snap := args[1]
		fs, _ := cmd.Flags().GetString("filesystem")
		svg, _ := cmd.Flags().GetString("svg")
		radosNamespace, _ := cmd.Flags().GetString("rados-namespace")

		cfg, err := parseCustomExecConfig(cmd)
		if err != nil {
			logging.Fatal(err)
		}

		cephFS := &filesystem.CephFilesystem{
			Ctx:               ctx,
			Clientsets:        root.ClientSets,
			OperatorNamespace: root.OperatorNamespace,
			ClusterNamespace:  root.StorageClusterNamespace,
			RadosNamespace:    radosNamespace,
			CustomExecConfig:  cfg,
		}

		cephFS.SnapshotDelete(fs, subvol, snap, svg)
	},
}

// parseCustomExecConfig parses custom pod execution flags
func parseCustomExecConfig(cmd *cobra.Command) (*filesystem.CustomExecConfig, error) {
	podName, _ := cmd.Flags().GetString("pod-name")
	if podName == "" {
		return nil, nil
	}

	podNamespace, _ := cmd.Flags().GetString("pod-namespace")
	podContainer, _ := cmd.Flags().GetString("pod-container")
	monIP, _ := cmd.Flags().GetString("mon-ip")
	userID, _ := cmd.Flags().GetString("user-id")
	userKey, _ := cmd.Flags().GetString("user-key")

	return &filesystem.CustomExecConfig{
		PodName:      podName,
		PodNamespace: podNamespace,
		Container:    podContainer,
		MonIP:        monIP,
		UserID:       userID,
		UserKey:      userKey,
	}, nil
}

// addCustomExecFlags adds custom pod execution flags to a command
func addCustomExecFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("pod-name", "", "Name of pod to execute commands in")
	cmd.PersistentFlags().String("pod-namespace", "", "Namespace of custom pod")
	cmd.PersistentFlags().String("pod-container", "", "Container name in custom pod")
	cmd.PersistentFlags().String("mon-ip", "", "Monitor IP for Ceph connection")
	cmd.PersistentFlags().String("user-id", "", "Ceph user ID")
	cmd.PersistentFlags().String("user-key", "", "Ceph user key")
}

func init() {
	CephFSSnapCmd.AddCommand(listCmd)
	listCmd.Flags().Bool("orphaned", false, "List only orphaned snapshots")
	CephFSSnapCmd.PersistentFlags().String("svg", "csi", "The name of the subvolume group")
	CephFSSnapCmd.PersistentFlags().String("filesystem", "myfs", "The name of the CephFS filesystem")
	CephFSSnapCmd.PersistentFlags().String("rados-namespace", "csi", "The rados namespace for omap operations")
	CephFSSnapCmd.AddCommand(deleteCmd)
	addCustomExecFlags(CephFSSnapCmd)
}
