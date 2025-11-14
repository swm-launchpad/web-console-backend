package value

import (
	"encoding/json"
)

// TemplateOption represents a user-configurable option for the template
type TemplateOption struct {
	Name        string   `json:"name"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`               // select, text, number, password
	Category    string   `json:"category,omitempty"` // required, optional, advanced
	Options     []string `json:"options,omitempty"`
	Default     string   `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	AllowCustom *bool    `json:"allow_custom,omitempty"` // Allow "Custom Input" option for select type
	CustomLabel *string  `json:"custom_label,omitempty"` // Label for custom option
}

// TemplateEnv represents an environment variable definition
type TemplateEnv struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Required bool   `json:"required"`
}

// TemplatePort represents a port configuration
type TemplatePort struct {
	InternalPort uint16 `json:"internal_port"`
	NetworkType  string `json:"network_type"` // http, https, tcp, udp, cluster_ip
	Description  string `json:"description,omitempty"`
}

// TemplateVolume represents a volume configuration
type TemplateVolume struct {
	MountPath   string `json:"mount_path"`
	Capacity    uint32 `json:"capacity"` // in Mi
	Description string `json:"description,omitempty"`
}

// DefaultPort represents a default port mapping
type DefaultPort struct {
	InternalPort uint16  `json:"internal_port"`
	ExternalPort *uint16 `json:"external_port,omitempty"`
	NetworkType  string  `json:"network_type"`
	ExternalIP   *string `json:"external_ip,omitempty"`
}

// DefaultEnv represents a default environment variable
type DefaultEnv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DefaultVolume represents a default volume mount
type DefaultVolume struct {
	VolumeName string `json:"volume_name"`
	MountPath  string `json:"mount_path"`
}

// DefaultResources represents default resource configuration
type DefaultResources struct {
	DefaultCPU           *uint32 `json:"default_cpu,omitempty"`            // UI 초기값
	DefaultMemory        *uint32 `json:"default_memory,omitempty"`         // UI 초기값
	MinRecommendedMemory *uint32 `json:"min_recommended_memory,omitempty"` // 경고 기준
}

// PortGuide represents port configuration guidance for user-configurable templates
type PortGuide struct {
	DefaultPort int    `json:"default_port,omitempty"`
	Description string `json:"description,omitempty"`
}

// TemplateConfig represents the JSON configuration of a template
type TemplateConfig struct {
	Description      string            `json:"description,omitempty"`
	Categories       []string          `json:"categories"`
	DisplayOrder     int               `json:"display_order"`
	IconName         string            `json:"icon_name,omitempty"`
	RequiresGit      bool              `json:"requires_git"`
	Version          string            `json:"version,omitempty"`
	TemplateOptions  []TemplateOption  `json:"template_options,omitempty"`
	TemplateEnv      []TemplateEnv     `json:"template_env,omitempty"`
	TemplatePorts    []TemplatePort    `json:"template_ports,omitempty"`
	TemplateVolumes  []TemplateVolume  `json:"template_volumes,omitempty"`
	DefaultPorts     []DefaultPort     `json:"default_ports,omitempty"`
	DefaultEnv       []DefaultEnv      `json:"default_env,omitempty"`
	DefaultVolumes   []DefaultVolume   `json:"default_volumes,omitempty"`
	DefaultResources *DefaultResources `json:"default_resources,omitempty"`
	PortGuide        *PortGuide        `json:"port_guide,omitempty"`
}

// NewTemplateConfig creates a new TemplateConfig from JSON string
func NewTemplateConfig(jsonData string) (*TemplateConfig, error) {
	if jsonData == "" {
		return &TemplateConfig{}, nil
	}

	var config TemplateConfig
	if err := json.Unmarshal([]byte(jsonData), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ToJSON converts TemplateConfig to JSON string
func (tc *TemplateConfig) ToJSON() (string, error) {
	if tc == nil {
		return "", nil
	}

	data, err := json.Marshal(tc)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetDescription returns the description
func (tc *TemplateConfig) GetDescription() string {
	if tc == nil {
		return ""
	}
	return tc.Description
}

// GetCategories returns the categories
func (tc *TemplateConfig) GetCategories() []string {
	if tc == nil {
		return []string{}
	}
	return tc.Categories
}

// GetDisplayOrder returns the display order
func (tc *TemplateConfig) GetDisplayOrder() int {
	if tc == nil {
		return 0
	}
	return tc.DisplayOrder
}

// GetIconName returns the icon name
func (tc *TemplateConfig) GetIconName() string {
	if tc == nil {
		return ""
	}
	return tc.IconName
}

// GetRequiresGit returns whether git is required
func (tc *TemplateConfig) GetRequiresGit() bool {
	if tc == nil {
		return false
	}
	return tc.RequiresGit
}

// GetVersion returns the version
func (tc *TemplateConfig) GetVersion() string {
	if tc == nil {
		return ""
	}
	return tc.Version
}
