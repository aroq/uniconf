package unitools

import "testing"

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

	if src["key1"].(map[string]interface{})["key1_subkey2"] != "key1_subkey2_value_override" {
		t.Error("Deep key merging failed")
	}
}
