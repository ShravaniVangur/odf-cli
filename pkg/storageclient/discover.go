package storageclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// runKubectl executes a kubectl command and returns the output
func runKubectl(ctx context.Context, args ...string) ([]byte, error) {
	// #nosec G204 -- args are constructed internally, not from user input
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("kubectl command failed: %v: %s", err, string(output))
	}
	return output, nil
}

// DiscoveredConfig holds all auto-discovered configuration parameters
type DiscoveredConfig struct {
	SubvolumeGroup string // CephFS subvolume group name
	RadosNamespace string // RADOS namespace for OMAP operations
	PodName        string // CephFS controller pod name
	PodNamespace   string // Namespace of the controller pod
	PodContainer   string // Container name (always "csi-cephfsplugin")
	MonitorIP      string // First Ceph monitor IP with port
	UserID         string // Ceph user ID (base64 decoded)
	UserKey        string // Ceph user key (base64 decoded)
}

// getStorageClientID queries the StorageClient and returns its status.id
func getStorageClientID(ctx context.Context, name, namespace string) (string, error) {
	output, err := runKubectl(ctx, "get", "storageclient", name, "-n", namespace, "-o", "json")
	if err != nil {
		return "", fmt.Errorf("failed to get StorageClient '%s' in namespace '%s': %v", name, namespace, err)
	}

	var result struct {
		Status struct {
			ID string `json:"id"`
		} `json:"status"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("failed to parse StorageClient JSON: %v", err)
	}

	if result.Status.ID == "" {
		return "", fmt.Errorf("StorageClient '%s' has no status.id (cluster not ready)", name)
	}

	return result.Status.ID, nil
}

// Discover queries Kubernetes resources to auto-discover CephFS configuration
// from a StorageClient resource.
func Discover(ctx context.Context, storageClientName, namespace string) (*DiscoveredConfig, error) {
	config := &DiscoveredConfig{}

	// Step 1: Get StorageClient status.id
	clusterID, err := getStorageClientID(ctx, storageClientName, namespace)
	if err != nil {
		return nil, err
	}

	// Remaining steps will be implemented in subsequent tasks
	_ = clusterID // Will use this in next task
	return config, fmt.Errorf("not fully implemented yet")
}
