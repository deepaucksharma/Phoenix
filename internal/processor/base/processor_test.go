package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration structure
type TestConfig struct {
	BaseConfig  `mapstructure:",squash"`
	StringParam string `mapstructure:"string_param"`
	IntParam    int    `mapstructure:"int_param"`
	NestedParam struct {
		SubParam bool `mapstructure:"sub_param"`
	} `mapstructure:"nested_param"`
	SliceParam []string `mapstructure:"slice_param"`
}

func TestGetConfigByPath(t *testing.T) {
	cfg := &TestConfig{
		BaseConfig:  BaseConfig{Enabled: true},
		StringParam: "test",
		IntParam:    42,
		SliceParam:  []string{"a", "b", "c"},
	}
	cfg.NestedParam.SubParam = true

	tests := []struct {
		name     string
		path     string
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "Get enabled",
			path:     "enabled",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Get string param",
			path:     "StringParam",
			expected: "test",
			wantErr:  false,
		},
		{
			name:     "Get int param",
			path:     "IntParam",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "Get nested param",
			path:     "NestedParam.SubParam",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Get slice element",
			path:     "SliceParam[1]",
			expected: "b",
			wantErr:  false,
		},
		{
			name:    "Invalid path",
			path:    "NonExistentParam",
			wantErr: true,
		},
		{
			name:    "Invalid nested path",
			path:    "NestedParam.NonExistentParam",
			wantErr: true,
		},
		{
			name:    "Invalid slice index",
			path:    "SliceParam[10]",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := GetConfigByPath(cfg, tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected, val)
		})
	}
}

func TestSetConfigByPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		value    interface{}
		validate func(*TestConfig) bool
		wantErr  bool
	}{
		{
			name:  "Set enabled",
			path:  "enabled",
			value: false,
			validate: func(cfg *TestConfig) bool {
				return !cfg.Enabled
			},
			wantErr: false,
		},
		{
			name:  "Set string param",
			path:  "StringParam",
			value: "new value",
			validate: func(cfg *TestConfig) bool {
				return cfg.StringParam == "new value"
			},
			wantErr: false,
		},
		{
			name:  "Set int param",
			path:  "IntParam",
			value: 99,
			validate: func(cfg *TestConfig) bool {
				return cfg.IntParam == 99
			},
			wantErr: false,
		},
		{
			name:  "Set nested param",
			path:  "NestedParam.SubParam",
			value: false,
			validate: func(cfg *TestConfig) bool {
				return !cfg.NestedParam.SubParam
			},
			wantErr: false,
		},
		{
			name:  "Set slice element",
			path:  "SliceParam[1]",
			value: "z",
			validate: func(cfg *TestConfig) bool {
				return cfg.SliceParam[1] == "z"
			},
			wantErr: false,
		},
		{
			name:    "Invalid path",
			path:    "NonExistentParam",
			value:   "value",
			wantErr: true,
		},
		{
			name:    "Type mismatch",
			path:    "IntParam",
			value:   "not an int",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh config for each test
			cfg := &TestConfig{
				BaseConfig:  BaseConfig{Enabled: true},
				StringParam: "test",
				IntParam:    42,
				SliceParam:  []string{"a", "b", "c"},
			}
			cfg.NestedParam.SubParam = true

			err := SetConfigByPath(cfg, tt.path, tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.True(t, tt.validate(cfg), "Value was not set correctly")
		})
	}
}

func TestGetDefaultConfigStatus(t *testing.T) {
	cfg := &TestConfig{
		BaseConfig:  BaseConfig{Enabled: true},
		StringParam: "test",
		IntParam:    42,
		SliceParam:  []string{"a", "b", "c"},
	}
	cfg.NestedParam.SubParam = true

	status, err := GetDefaultConfigStatus(cfg)
	require.NoError(t, err)

	// Check that Enabled is true
	assert.True(t, status.Enabled)

	// Check that parameters are exported
	assert.Contains(t, status.Parameters, "stringparam")
	assert.Equal(t, "test", status.Parameters["stringparam"])

	assert.Contains(t, status.Parameters, "intparam")
	assert.Equal(t, 42, status.Parameters["intparam"])
}
