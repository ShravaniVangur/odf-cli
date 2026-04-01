package storageclient

import (
	"context"
	"encoding/base64"
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

// csiConfigEntry represents a single cluster configuration in ceph-csi-config
type csiConfigEntry struct {
	ClusterID string   `json:"clusterID"`
	Monitors  []string `json:"monitors"`
	CephFS    struct {
		SubvolumeGroup             string `json:"subvolumeGroup"`
		RadosNamespace             string `json:"radosNamespace"`
		ControllerPublishSecretRef struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"controllerPublishSecretRef"`
	} `json:"cephFS"`
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

// getCephFSConfig queries ceph-csi-config ConfigMap and extracts CephFS configuration
// for the given clusterID
func getCephFSConfig(ctx context.Context, clusterID, namespace string) (*csiConfigEntry, error) {
	output, err := runKubectl(ctx, "get", "configmap", "ceph-csi-config", "-n", namespace, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to get configMap 'ceph-csi-config' in namespace '%s': %v", namespace, err)
	}

	var cm struct {
		Data struct {
			ConfigJSON string `json:"config.json"`
		} `json:"data"`
	}

	if err := json.Unmarshal(output, &cm); err != nil {
		return nil, fmt.Errorf("failed to parse configmap JSON: %v", err)
	}

	if cm.Data.ConfigJSON == "" {
		return nil, fmt.Errorf("ceph-csi-config configMap has no 'config.json' data in namespace '%s'", namespace)
	}

	var configs []csiConfigEntry
	if err := json.Unmarshal([]byte(cm.Data.ConfigJSON), &configs); err != nil {
		return nil, fmt.Errorf("failed to parse config.json in ceph-csi-config: %v", err)
	}

	// Find matching clusterID
	for _, config := range configs {
		if config.ClusterID == clusterID {
			// Validate cephFS section exists
			if config.CephFS.SubvolumeGroup == "" {
				return nil, fmt.Errorf("clusterID '%s' has no cephFS configuration in ceph-csi-config", clusterID)
			}
			return &config, nil
		}
	}

	return nil, fmt.Errorf("no matching clusterID '%s' found in ceph-csi-config", clusterID)
}

// getCephCredentials queries the secret and extracts userID and userKey
func getCephCredentials(ctx context.Context, secretName, secretNamespace string) (userID, userKey string, err error) {
	output, err := runKubectl(ctx, "get", "secret", secretName, "-n", secretNamespace, "-o", "json")
	if err != nil {
		return "", "", fmt.Errorf("failed to get secret '%s' in namespace '%s': %v", secretName, secretNamespace, err)
	}

	var secret struct {
		Data map[string]string `json:"data"`
	}

	if err := json.Unmarshal(output, &secret); err != nil {
		return "", "", fmt.Errorf("failed to parse secret JSON: %v", err)
	}

	// Extract and decode userID
	userIDEncoded, ok := secret.Data["userID"]
	if !ok {
		return "", "", fmt.Errorf("secret '%s' missing required field: userID", secretName)
	}
	userIDBytes, err := base64.StdEncoding.DecodeString(userIDEncoded)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode userID from secret '%s': %v", secretName, err)
	}
	userID = string(userIDBytes)

	// Extract and decode userKey
	userKeyEncoded, ok := secret.Data["userKey"]
	if !ok {
		return "", "", fmt.Errorf("secret '%s' missing required field: userKey", secretName)
	}
	userKeyBytes, err := base64.StdEncoding.DecodeString(userKeyEncoded)
	if err != nil {
		return "", "", fmt.Errorf("failed to decode userKey from secret '%s': %v", secretName, err)
	}
	userKey = string(userKeyBytes)

	return userID, userKey, nil
}

// getCephFSControllerPod finds a running CephFS controller pod
func getCephFSControllerPod(ctx context.Context, namespace string) (podName, podNamespace string, err error) {
	output, err := runKubectl(ctx, "get", "pods", "-n", namespace,
		"-l", "app=openshift-storage.cephfs.csi.ceph.com-ctrlplugin",
		"-o", "json")
	if err != nil {
		return "", "", fmt.Errorf("no CephFS controller pods found with label app=openshift-storage.cephfs.csi.ceph.com-ctrlplugin in namespace '%s': %v", namespace, err)
	}

	var podList struct {
		Items []struct {
			Metadata struct {
				Name      string `json:"name"`
				Namespace string `json:"namespace"`
			} `json:"metadata"`
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal(output, &podList); err != nil {
		return "", "", fmt.Errorf("failed to parse pods JSON: %v", err)
	}

	if len(podList.Items) == 0 {
		return "", "", fmt.Errorf("no CephFS controller pods found with label app=openshift-storage.cephfs.csi.ceph.com-ctrlplugin in namespace '%s'", namespace)
	}

	// Find first Running pod
	for _, pod := range podList.Items {
		if pod.Status.Phase == "Running" {
			return pod.Metadata.Name, pod.Metadata.Namespace, nil
		}
	}

	return "", "", fmt.Errorf("found %d CephFS controller pod(s) but none are running", len(podList.Items))
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

	// Step 2: Get CephFS config from ConfigMap
	cephFSConfig, err := getCephFSConfig(ctx, clusterID, namespace)
	if err != nil {
		return nil, err
	}

	// Extract values
	config.SubvolumeGroup = cephFSConfig.CephFS.SubvolumeGroup
	config.RadosNamespace = cephFSConfig.CephFS.RadosNamespace
	if len(cephFSConfig.Monitors) > 0 {
		config.MonitorIP = cephFSConfig.Monitors[0]
	}

	// Step 3: Get credentials from Secret
	secretName := cephFSConfig.CephFS.ControllerPublishSecretRef.Name
	secretNamespace := cephFSConfig.CephFS.ControllerPublishSecretRef.Namespace
	userID, userKey, err := getCephCredentials(ctx, secretName, secretNamespace)
	if err != nil {
		return nil, err
	}
	config.UserID = userID
	config.UserKey = userKey

	// Step 4: Find CephFS controller pod
	podName, podNamespace, err := getCephFSControllerPod(ctx, namespace)
	if err != nil {
		return nil, err
	}
	config.PodName = podName
	config.PodNamespace = podNamespace
	config.PodContainer = "csi-cephfsplugin"

	return config, nil
}
