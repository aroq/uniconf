package unitools

import (
	"testing"
)

func TestMerge(t *testing.T) {
	src := map[string]interface{}{
		"key1": map[string]interface{}{
			"key1_subkey1": "key1_subkey1_value",
			"key1_subkey2": "key1_subkey2_value",
		},
		"key2": map[string]interface{}{
			"key2_subkey1": "key2_subkey1_value",
		},
	}

	dest := map[string]interface{}{
		"key1": map[string]interface{}{
			"key1_subkey2": "key1_subkey2_value_override",
			"key1_subkey3": "key1_subkey3_value",
		},
		"key2": map[string]interface{}{
			"key2_subkey1": "key2_subkey1_value",
		},
	}

	Merge(src, dest)

    // Test deep key merging.
	if src["key1"].(map[string]interface{})["key1_subkey2"] != "key1_subkey2_value_override" {
		t.Errorf("Deep key merging failed: %s", "key1_subkey2 != key1_subkey2_value_override")
	}

	// Test key merging.
	if src["key2"].(map[string]interface{})["key2_subkey1"] != "key2_subkey1_value" {
		t.Errorf("Deep key merging failed: %s", "key2_subkey1 != key2_subkey1_value")
	}
}
