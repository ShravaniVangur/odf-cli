package storageclient

import (
	"encoding/base64"
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

// TestSecretStructureParsing tests parsing the Secret structure and base64 decoding
func TestSecretStructureParsing(t *testing.T) {
	// Sample Secret JSON with base64-encoded values
	// userID: "csi-cephfs-node" (base64: Y3NpLWNlcGhmcy1ub2Rl)
	// userKey: "AQD1234567890abcdef==" (base64: QVFEMTIzNDU2Nzg5MGFiY2RlZj09)
	sampleSecret := `{
		"apiVersion": "v1",
		"kind": "Secret",
		"metadata": {
			"name": "rook-csi-cephfs-provisioner",
			"namespace": "openshift-storage"
		},
		"type": "Opaque",
		"data": {
			"userID": "Y3NpLWNlcGhmcy1ub2Rl",
			"userKey": "QVFEMTIzNDU2Nzg5MGFiY2RlZj09"
		}
	}`

	var secret struct {
		Data map[string]string `json:"data"`
	}

	err := json.Unmarshal([]byte(sampleSecret), &secret)
	if err != nil {
		t.Fatalf("Failed to parse Secret JSON: %v", err)
	}

	// Verify we have the data fields
	if len(secret.Data) != 2 {
		t.Errorf("Expected 2 data fields, got %d", len(secret.Data))
	}

	// Test userID decode
	userIDEncoded, ok := secret.Data["userID"]
	if !ok {
		t.Fatal("Expected userID field in secret data")
	}
	userIDBytes, err := base64.StdEncoding.DecodeString(userIDEncoded)
	if err != nil {
		t.Fatalf("Failed to decode userID: %v", err)
	}
	userID := string(userIDBytes)
	if userID != "csi-cephfs-node" {
		t.Errorf("Expected userID 'csi-cephfs-node', got '%s'", userID)
	}

	// Test userKey decode
	userKeyEncoded, ok := secret.Data["userKey"]
	if !ok {
		t.Fatal("Expected userKey field in secret data")
	}
	userKeyBytes, err := base64.StdEncoding.DecodeString(userKeyEncoded)
	if err != nil {
		t.Fatalf("Failed to decode userKey: %v", err)
	}
	userKey := string(userKeyBytes)
	if userKey != "AQD1234567890abcdef==" {
		t.Errorf("Expected userKey 'AQD1234567890abcdef==', got '%s'", userKey)
	}
}

// TestSecretMissingFields tests handling of secrets missing required fields
func TestSecretMissingFields(t *testing.T) {
	tests := []struct {
		name       string
		secretJSON string
		wantError  bool
	}{
		{
			name: "missing userID",
			secretJSON: `{
				"data": {
					"userKey": "QVFEMTIzNDU2Nzg5MGFiY2RlZj09"
				}
			}`,
			wantError: true,
		},
		{
			name: "missing userKey",
			secretJSON: `{
				"data": {
					"userID": "Y3NpLWNlcGhmcy1ub2Rl"
				}
			}`,
			wantError: true,
		},
		{
			name: "both fields present",
			secretJSON: `{
				"data": {
					"userID": "Y3NpLWNlcGhmcy1ub2Rl",
					"userKey": "QVFEMTIzNDU2Nzg5MGFiY2RlZj09"
				}
			}`,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var secret struct {
				Data map[string]string `json:"data"`
			}

			err := json.Unmarshal([]byte(tt.secretJSON), &secret)
			if err != nil {
				t.Fatalf("Failed to parse Secret JSON: %v", err)
			}

			// Check for userID
			_, hasUserID := secret.Data["userID"]
			_, hasUserKey := secret.Data["userKey"]

			gotError := !hasUserID || !hasUserKey
			if gotError != tt.wantError {
				t.Errorf("Expected error=%v, got error=%v (hasUserID=%v, hasUserKey=%v)",
					tt.wantError, gotError, hasUserID, hasUserKey)
			}
		})
	}
}
