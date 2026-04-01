package subvolume

import (
	"github.com/red-hat-storage/odf-cli/cmd/odf/root"
	subvolume "github.com/rook/kubectl-rook-ceph/pkg/filesystem"
	"github.com/rook/kubectl-rook-ceph/pkg/logging"
	"github.com/spf13/cobra"
)

var SubvolumeCmd = &cobra.Command{
	Use:   "subvolume",
	Short: "Manages subvolumes",
	Args:  cobra.ExactArgs(1),
}

var listCmd = &cobra.Command{
	Use:     "ls",
	Short:   "Print the list of subvolumes.",
	Example: "odf subvolume ls",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		staleSubvol, _ := cmd.Flags().GetBool("stale")
		svgName, _ := cmd.Flags().GetString("svg")
		radosNamespace, _ := cmd.Flags().GetString("rados-namespace")

		cfg, err := parseCustomExecConfig(cmd)
		if err != nil {
			logging.Fatal(err)
		}

		cephFS := &subvolume.CephFilesystem{
			Ctx:               ctx,
			Clientsets:        root.ClientSets,
			OperatorNamespace: root.OperatorNamespace,
			ClusterNamespace:  root.StorageClusterNamespace,
			RadosNamespace:    radosNamespace,
			CustomExecConfig:  cfg,
		}

		cephFS.List(svgName, staleSubvol)
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes a stale subvolume",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		fs := args[0]
		subvol := args[1]
		svg := args[2]
		radosNamespace, _ := cmd.Flags().GetString("rados-namespace")

		cfg, err := parseCustomExecConfig(cmd)
		if err != nil {
			logging.Fatal(err)
		}

		cephFS := &subvolume.CephFilesystem{
			Ctx:               ctx,
			Clientsets:        root.ClientSets,
			OperatorNamespace: root.OperatorNamespace,
			ClusterNamespace:  root.StorageClusterNamespace,
			RadosNamespace:    radosNamespace,
			CustomExecConfig:  cfg,
		}

		cephFS.Delete(fs, subvol, svg)
	},
}

func init() {
	SubvolumeCmd.AddCommand(listCmd)
	SubvolumeCmd.PersistentFlags().Bool("stale", false, "Only list stale subvolumes")
	SubvolumeCmd.PersistentFlags().String("svg", "csi", "The name of the subvolume group")
	SubvolumeCmd.PersistentFlags().String("rados-namespace", "csi", "The rados namespace for omap operations")
	SubvolumeCmd.AddCommand(deleteCmd)
	addCustomExecFlags(SubvolumeCmd)
}

// parseCustomExecConfig parses custom pod execution flags
func parseCustomExecConfig(cmd *cobra.Command) (*subvolume.CustomExecConfig, error) {
	podName, _ := cmd.Flags().GetString("pod-name")
	if podName == "" {
		return nil, nil
	}

	podNamespace, _ := cmd.Flags().GetString("pod-namespace")
	podContainer, _ := cmd.Flags().GetString("pod-container")
	monIP, _ := cmd.Flags().GetString("mon-ip")
	userID, _ := cmd.Flags().GetString("user-id")
	userKey, _ := cmd.Flags().GetString("user-key")

	return &subvolume.CustomExecConfig{
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
