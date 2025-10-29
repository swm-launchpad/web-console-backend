package adapter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyBuildVars_NilMap(t *testing.T) {
	result := copyBuildVars(nil)
	assert.Nil(t, result, "copyBuildVars should return nil for nil input")
}

func TestCopyBuildVars_EmptyMap(t *testing.T) {
	src := make(map[string]string)
	result := copyBuildVars(src)

	assert.NotNil(t, result, "copyBuildVars should return non-nil for empty map")
	assert.Empty(t, result, "copied map should be empty")
}

func TestCopyBuildVars_WithData(t *testing.T) {
	src := map[string]string{
		"API_KEY":     "secret123",
		"BUILD_ENV":   "production",
		"DEBUG_MODE":  "false",
		"VERSION":     "1.0.0",
		"EMPTY_VALUE": "",
	}

	result := copyBuildVars(src)

	assert.NotNil(t, result, "copied map should not be nil")
	assert.Equal(t, len(src), len(result), "copied map should have same length")
	assert.Equal(t, src, result, "copied map should have identical content")
}

func TestCopyBuildVars_PreventAliasing(t *testing.T) {
	src := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	copied := copyBuildVars(src)

	// Modify the copied map
	copied["KEY1"] = "modified"
	copied["KEY3"] = "new_key"
	delete(copied, "KEY2")

	// Original should be unchanged
	assert.Equal(t, "value1", src["KEY1"], "original map KEY1 should be unchanged")
	assert.Equal(t, "value2", src["KEY2"], "original map KEY2 should be unchanged")
	assert.NotContains(t, src, "KEY3", "original map should not have new key")

	// Copied should have modifications
	assert.Equal(t, "modified", copied["KEY1"], "copied map should have modified value")
	assert.Equal(t, "new_key", copied["KEY3"], "copied map should have new key")
	assert.NotContains(t, copied, "KEY2", "copied map should have deleted key")
}

func TestDeepCopyTemplateConfig_NilMap(t *testing.T) {
	result, err := deepCopyTemplateConfig(nil)
	assert.NoError(t, err, "deepCopyTemplateConfig should not error for nil input")
	assert.Nil(t, result, "deepCopyTemplateConfig should return nil for nil input")
}

func TestDeepCopyTemplateConfig_EmptyMap(t *testing.T) {
	src := make(map[string]interface{})
	result, err := deepCopyTemplateConfig(src)

	assert.NoError(t, err, "deepCopyTemplateConfig should not error for empty map")
	assert.NotNil(t, result, "deepCopyTemplateConfig should return non-nil for empty map")
	assert.Empty(t, result, "copied map should be empty")
}

func TestDeepCopyTemplateConfig_SimpleMap(t *testing.T) {
	src := map[string]interface{}{
		"key1": "value1",
		"key2": float64(123), // JSON unmarshaling converts numbers to float64
		"key3": true,
		"key4": 45.67,
	}

	result, err := deepCopyTemplateConfig(src)

	assert.NoError(t, err, "deepCopyTemplateConfig should not error")
	assert.NotNil(t, result, "copied map should not be nil")
	assert.Equal(t, src, result, "copied map should have identical content")
}

func TestDeepCopyTemplateConfig_NestedMap(t *testing.T) {
	src := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"key": "value",
			},
			"array": []interface{}{"a", "b", "c"},
		},
		"simple": "value",
	}

	result, err := deepCopyTemplateConfig(src)

	assert.NoError(t, err, "deepCopyTemplateConfig should not error for nested map")
	assert.NotNil(t, result, "copied map should not be nil")
	assert.Equal(t, src, result, "copied map should have identical content")

	// Modify nested value in copy
	resultLevel1 := result["level1"].(map[string]interface{})
	resultLevel2 := resultLevel1["level2"].(map[string]interface{})
	resultLevel2["key"] = "modified"

	// Original should be unchanged
	srcLevel1 := src["level1"].(map[string]interface{})
	srcLevel2 := srcLevel1["level2"].(map[string]interface{})
	assert.Equal(t, "value", srcLevel2["key"], "original nested value should be unchanged")

	// Copy should have modification
	assert.Equal(t, "modified", resultLevel2["key"], "copied nested value should be modified")
}

func TestDeepCopyTemplateConfig_PreventArrayAliasing(t *testing.T) {
	src := map[string]interface{}{
		"array": []interface{}{"item1", "item2"},
	}

	result, err := deepCopyTemplateConfig(src)

	assert.NoError(t, err, "deepCopyTemplateConfig should not error")

	// Modify array in copy
	resultArray := result["array"].([]interface{})
	resultArray[0] = "modified"

	// Original array should be unchanged
	srcArray := src["array"].([]interface{})
	assert.Equal(t, "item1", srcArray[0], "original array should be unchanged")

	// Copied array should have modification
	assert.Equal(t, "modified", resultArray[0], "copied array should be modified")
}
