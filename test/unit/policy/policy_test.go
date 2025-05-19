package policy_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/pkg/policy"
)

func TestLoadPolicyValid(t *testing.T) {
	policyPath := "../../configs/development/policy.yaml"
	p, err := policy.LoadPolicy(policyPath)
	require.NoError(t, err, "expected valid policy to load")
	require.NotNil(t, p)
	assert.Equal(t, "shadow", p.GlobalSettings.AutonomyLevel)
}

func TestLoadPolicyInvalid(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "bad*.yaml")
	require.NoError(t, err)
	defer tmpFile.Close()
	// Missing required fields
	tmpFile.WriteString("global_settings:\n  autonomy_level: shadow\n")

	_, err = policy.LoadPolicy(tmpFile.Name())
	require.Error(t, err, "expected invalid policy to fail")
}

func TestParsePolicyValid(t *testing.T) {
	data, err := os.ReadFile("../../configs/development/policy.yaml")
	require.NoError(t, err)

	p, err := policy.ParsePolicy(data)
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, "shadow", p.GlobalSettings.AutonomyLevel)
}

func TestParsePolicyInvalid(t *testing.T) {
	data := []byte("global_settings:\n  autonomy_level: shadow\n")
	_, err := policy.ParsePolicy(data)
	require.Error(t, err)
}
