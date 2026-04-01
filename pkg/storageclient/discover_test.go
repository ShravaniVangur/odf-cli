package storageclient

import (
	"encoding/json"
	"testing"
)

// TestCsiConfigEntryParsing tests that the csiConfigEntry struct can correctly
// parse the expected JSON structure from ceph-csi-config ConfigMap
func TestCsiConfigEntryParsing(t *testing.T) {
	// Sample config.json content from ceph-csi-config ConfigMap
	sampleConfigJSON := `[
		{
			"clusterID": "test-cluster-123",
			"monitors": ["10.0.0.1:6789", "10.0.0.2:6789"],
			"cephFS": {
				"subvolumeGroup": "csi",
				"radosNamespace": "csi",
				"controllerPublishSecretRef": {
					"name": "rook-csi-cephfs-provisioner",
					"namespace": "openshift-storage"
				}
			}
		},
		{
			"clusterID": "another-cluster-456",
			"monitors": ["10.0.1.1:6789"],
			"cephFS": {
				"subvolumeGroup": "test-group",
				"radosNamespace": "",
				"controllerPublishSecretRef": {
					"name": "test-secret",
					"namespace": "test-ns"
				}
			}
		}
	]`

	var configs []csiConfigEntry
	err := json.Unmarshal([]byte(sampleConfigJSON), &configs)
	if err != nil {
		t.Fatalf("Failed to parse config.json: %v", err)
	}

	// Verify we got 2 entries
	if len(configs) != 2 {
		t.Errorf("Expected 2 config entries, got %d", len(configs))
	}

	// Verify first entry
	if configs[0].ClusterID != "test-cluster-123" {
		t.Errorf("Expected clusterID 'test-cluster-123', got '%s'", configs[0].ClusterID)
	}
	if len(configs[0].Monitors) != 2 {
		t.Errorf("Expected 2 monitors, got %d", len(configs[0].Monitors))
	}
	if configs[0].Monitors[0] != "10.0.0.1:6789" {
		t.Errorf("Expected monitor '10.0.0.1:6789', got '%s'", configs[0].Monitors[0])
	}
	if configs[0].CephFS.SubvolumeGroup != "csi" {
		t.Errorf("Expected subvolumeGroup 'csi', got '%s'", configs[0].CephFS.SubvolumeGroup)
	}
	if configs[0].CephFS.RadosNamespace != "csi" {
		t.Errorf("Expected radosNamespace 'csi', got '%s'", configs[0].CephFS.RadosNamespace)
	}
	if configs[0].CephFS.ControllerPublishSecretRef.Name != "rook-csi-cephfs-provisioner" {
		t.Errorf("Expected secret name 'rook-csi-cephfs-provisioner', got '%s'", configs[0].CephFS.ControllerPublishSecretRef.Name)
	}
	if configs[0].CephFS.ControllerPublishSecretRef.Namespace != "openshift-storage" {
		t.Errorf("Expected secret namespace 'openshift-storage', got '%s'", configs[0].CephFS.ControllerPublishSecretRef.Namespace)
	}

	// Verify second entry
	if configs[1].ClusterID != "another-cluster-456" {
		t.Errorf("Expected clusterID 'another-cluster-456', got '%s'", configs[1].ClusterID)
	}
	if configs[1].CephFS.RadosNamespace != "" {
		t.Errorf("Expected empty radosNamespace, got '%s'", configs[1].CephFS.RadosNamespace)
	}
}

// TestConfigMapStructureParsing tests parsing the outer ConfigMap structure
func TestConfigMapStructureParsing(t *testing.T) {
	// Sample ConfigMap JSON
	sampleConfigMap := `{
		"apiVersion": "v1",
		"kind": "ConfigMap",
		"metadata": {
			"name": "ceph-csi-config",
			"namespace": "openshift-storage"
		},
		"data": {
			"config.json": "[{\"clusterID\":\"test-123\",\"monitors\":[\"10.0.0.1:6789\"],\"cephFS\":{\"subvolumeGroup\":\"csi\",\"radosNamespace\":\"csi\",\"controllerPublishSecretRef\":{\"name\":\"test-secret\",\"namespace\":\"test-ns\"}}}]"
		}
	}`

	var cm struct {
		Data struct {
			ConfigJSON string `json:"config.json"`
		} `json:"data"`
	}

	err := json.Unmarshal([]byte(sampleConfigMap), &cm)
	if err != nil {
		t.Fatalf("Failed to parse ConfigMap JSON: %v", err)
	}

	// Verify we extracted config.json string
	if cm.Data.ConfigJSON == "" {
		t.Error("Expected non-empty config.json field")
	}

	// Verify we can parse the nested config.json
	var configs []csiConfigEntry
	err = json.Unmarshal([]byte(cm.Data.ConfigJSON), &configs)
	if err != nil {
		t.Fatalf("Failed to parse nested config.json: %v", err)
	}

	if len(configs) != 1 {
		t.Errorf("Expected 1 config entry, got %d", len(configs))
	}

	if configs[0].ClusterID != "test-123" {
		t.Errorf("Expected clusterID 'test-123', got '%s'", configs[0].ClusterID)
	}
}

// TestCsiConfigEntryEmptySubvolumeGroup tests that empty subvolumeGroup is parsed correctly
func TestCsiConfigEntryEmptySubvolumeGroup(t *testing.T) {
	sampleConfigJSON := `[
		{
			"clusterID": "test-cluster",
			"monitors": ["10.0.0.1:6789"],
			"cephFS": {
				"subvolumeGroup": "",
				"radosNamespace": "csi",
				"controllerPublishSecretRef": {
					"name": "test-secret",
					"namespace": "test-ns"
				}
			}
		}
	]`

	var configs []csiConfigEntry
	err := json.Unmarshal([]byte(sampleConfigJSON), &configs)
	if err != nil {
		t.Fatalf("Failed to parse config.json: %v", err)
	}

	// Verify the subvolumeGroup is indeed empty
	if configs[0].CephFS.SubvolumeGroup != "" {
		t.Errorf("Expected empty subvolumeGroup, got '%s'", configs[0].CephFS.SubvolumeGroup)
	}

	// The empty SubvolumeGroup was correctly parsed
}
