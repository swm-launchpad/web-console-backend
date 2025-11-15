package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateConfig_EmptyJSON(t *testing.T) {
	config, err := NewTemplateConfig("")
	require.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "", config.Description)
	assert.Equal(t, 0, len(config.Categories))
}

func TestNewTemplateConfig_ValidJSON(t *testing.T) {
	jsonData := `{
		"description": "React SPA application",
		"categories": ["frontend", "spa"],
		"display_order": 1,
		"icon_name": "react",
		"requires_git": true,
		"version": "1.0",
		"template_options": [
			{
				"name": "node_version",
				"label": "Node.js Version",
				"type": "select",
				"options": ["18", "20", "22"],
				"default": "20"
			}
		],
		"template_env": [
			{
				"key": "NODE_ENV",
				"value": "production",
				"required": true
			},
			{
				"key": "PORT",
				"value": "3000",
				"required": true
			}
		],
		"default_resources": {
			"default_cpu": 500,
			"default_memory": 512,
			"min_recommended_memory": 512
		}
	}`

	config, err := NewTemplateConfig(jsonData)
	require.NoError(t, err)
	assert.Equal(t, "React SPA application", config.Description)
	assert.Equal(t, []string{"frontend", "spa"}, config.Categories)
	assert.Equal(t, 1, config.DisplayOrder)
	assert.Equal(t, "react", config.IconName)
	assert.True(t, config.RequiresGit)
	assert.Equal(t, "1.0", config.Version)
	assert.Len(t, config.TemplateOptions, 1)
	assert.Equal(t, "node_version", config.TemplateOptions[0].Name)
	assert.Len(t, config.TemplateEnv, 2)
	assert.Equal(t, "NODE_ENV", config.TemplateEnv[0].Key)
	assert.NotNil(t, config.DefaultResources)
	assert.Equal(t, uint32(500), *config.DefaultResources.DefaultCPU)
	assert.Equal(t, uint32(512), *config.DefaultResources.DefaultMemory)
	assert.Equal(t, uint32(512), *config.DefaultResources.MinRecommendedMemory)
}

func TestNewTemplateConfig_InvalidJSON(t *testing.T) {
	jsonData := `{invalid json}`
	_, err := NewTemplateConfig(jsonData)
	assert.Error(t, err)
}

func TestTemplateConfig_ToJSON(t *testing.T) {
	defaultCPU := uint32(500)
	defaultMemory := uint32(512)
	minRecommendedMemory := uint32(512)

	config := &TemplateConfig{
		Description:  "Test template",
		Categories:   []string{"backend"},
		DisplayOrder: 1,
		IconName:     "test",
		RequiresGit:  true,
		Version:      "1.0",
		DefaultResources: &DefaultResources{
			DefaultCPU:           &defaultCPU,
			DefaultMemory:        &defaultMemory,
			MinRecommendedMemory: &minRecommendedMemory,
		},
	}

	jsonData, err := config.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonData, "Test template")
	assert.Contains(t, jsonData, "backend")

	// Round-trip test
	config2, err := NewTemplateConfig(jsonData)
	require.NoError(t, err)
	assert.Equal(t, config.Description, config2.Description)
	assert.Equal(t, config.Categories, config2.Categories)
	assert.Equal(t, config.DisplayOrder, config2.DisplayOrder)
}

func TestTemplateConfig_ToJSON_Nil(t *testing.T) {
	var config *TemplateConfig
	jsonData, err := config.ToJSON()
	require.NoError(t, err)
	assert.Equal(t, "", jsonData)
}

func TestTemplateConfig_Getters(t *testing.T) {
	config := &TemplateConfig{
		Description:  "Test Description",
		Categories:   []string{"cat1", "cat2"},
		DisplayOrder: 5,
		IconName:     "testicon",
		RequiresGit:  true,
		Version:      "2.0",
	}

	assert.Equal(t, "Test Description", config.GetDescription())
	assert.Equal(t, []string{"cat1", "cat2"}, config.GetCategories())
	assert.Equal(t, 5, config.GetDisplayOrder())
	assert.Equal(t, "testicon", config.GetIconName())
	assert.True(t, config.GetRequiresGit())
	assert.Equal(t, "2.0", config.GetVersion())
}

func TestTemplateConfig_Getters_Nil(t *testing.T) {
	var config *TemplateConfig

	assert.Equal(t, "", config.GetDescription())
	assert.Equal(t, []string{}, config.GetCategories())
	assert.Equal(t, 0, config.GetDisplayOrder())
	assert.Equal(t, "", config.GetIconName())
	assert.False(t, config.GetRequiresGit())
	assert.Equal(t, "", config.GetVersion())
}
