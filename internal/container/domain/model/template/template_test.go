package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
	"github.com/swm-launchpad/web-console-backend/internal/container/domain/model/template/value"
)

func TestNewTemplate_Success(t *testing.T) {
	templateBody := "FROM node:20-alpine"
	config := &value.TemplateConfig{
		Description: "React template",
		Categories:  []string{"frontend"},
	}

	template, err := NewTemplate("React", &templateBody, config, value.TemplateStatusActive)
	require.NoError(t, err)
	assert.Equal(t, "React", template.Name())
	assert.Equal(t, &templateBody, template.TemplateBody())
	assert.Equal(t, config, template.TemplateConfig())
	assert.Equal(t, value.TemplateStatusActive, template.Status())
	assert.True(t, template.IsActive())
	assert.False(t, template.CreatedAt().IsZero())
}

func TestNewTemplate_EmptyName(t *testing.T) {
	_, err := NewTemplate("", nil, nil, value.TemplateStatusActive)
	assert.ErrorIs(t, err, containererrors.ErrInvalidTemplateConfig)
}

func TestNewTemplate_InvalidStatus(t *testing.T) {
	_, err := NewTemplate("React", nil, nil, value.TemplateStatus("invalid"))
	assert.ErrorIs(t, err, containererrors.ErrInvalidTemplateConfig)
}

func TestNewTemplate_NullableFields(t *testing.T) {
	template, err := NewTemplate("Minimal", nil, nil, value.TemplateStatusActive)
	require.NoError(t, err)
	assert.Nil(t, template.TemplateBody())
	assert.Nil(t, template.TemplateConfig())
}

func TestReconstructTemplate(t *testing.T) {
	templateBody := "FROM python:3.11"
	config := &value.TemplateConfig{
		Description: "Python template",
	}
	now := time.Now()
	updated := now.Add(1 * time.Hour)

	template := ReconstructTemplate(
		1,
		"Python",
		&templateBody,
		config,
		value.TemplateStatusActive,
		now,
		updated,
	)

	assert.Equal(t, uint(1), template.TemplateID())
	assert.Equal(t, "Python", template.Name())
	assert.Equal(t, &templateBody, template.TemplateBody())
	assert.Equal(t, config, template.TemplateConfig())
	assert.Equal(t, value.TemplateStatusActive, template.Status())
	assert.Equal(t, now, template.CreatedAt())
	assert.Equal(t, updated, template.UpdatedAt())
}

func TestTemplate_IsActive(t *testing.T) {
	tests := []struct {
		name   string
		status value.TemplateStatus
		want   bool
	}{
		{"active", value.TemplateStatusActive, true},
		{"inactive", value.TemplateStatusInactive, false},
		{"deprecated", value.TemplateStatusDeprecated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, _ := NewTemplate("Test", nil, nil, tt.status)
			assert.Equal(t, tt.want, template.IsActive())
		})
	}
}

func TestTemplate_SetTemplateID(t *testing.T) {
	template, _ := NewTemplate("Test", nil, nil, value.TemplateStatusActive)
	assert.Equal(t, uint(0), template.TemplateID())

	template.SetTemplateID(123)
	assert.Equal(t, uint(123), template.TemplateID())
}

func TestTemplate_Getters(t *testing.T) {
	templateBody := "FROM node:20"
	config := &value.TemplateConfig{
		Description:  "Node.js template",
		Categories:   []string{"backend"},
		DisplayOrder: 5,
		IconName:     "nodejs",
		RequiresGit:  true,
		Version:      "1.0",
	}
	now := time.Now()
	updated := now.Add(2 * time.Hour)

	template := ReconstructTemplate(
		42,
		"Node.js",
		&templateBody,
		config,
		value.TemplateStatusActive,
		now,
		updated,
	)

	assert.Equal(t, uint(42), template.TemplateID())
	assert.Equal(t, "Node.js", template.Name())
	assert.Equal(t, "FROM node:20", *template.TemplateBody())
	assert.Equal(t, "Node.js template", template.TemplateConfig().GetDescription())
	assert.Equal(t, value.TemplateStatusActive, template.Status())
	assert.Equal(t, now, template.CreatedAt())
	assert.Equal(t, updated, template.UpdatedAt())
}
