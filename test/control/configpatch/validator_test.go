package configpatch_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.opentelemetry.io/collector/component"

	"github.com/deepaucksharma/Phoenix/internal/control/configpatch"
	"github.com/deepaucksharma/Phoenix/internal/interfaces"
)

func createPatch(id, param string) interfaces.ConfigPatch {
	return interfaces.ConfigPatch{
		PatchID:             id,
		TargetProcessorName: component.MustNewID("test-processor"),
		ParameterPath:       param,
		NewValue:            1,
		Reason:              "test",
		Severity:            "normal",
		Source:              "manual",
		Timestamp:           time.Now().Unix(),
	}
}

func TestStandardValidatorMissingFieldsAndInvalidValues(t *testing.T) {
	opts := &configpatch.Options{
		AllowedSeverityLevels: []string{"normal"},
		AllowedSources:        []string{"manual"},
	}

	tests := []struct {
		name  string
		patch interfaces.ConfigPatch
	}{
		{
			name: "missing patch id",
			patch: interfaces.ConfigPatch{
				TargetProcessorName: component.MustNewID("test"),
				ParameterPath:       "p",
			},
		},
		{
			name: "missing processor name",
			patch: interfaces.ConfigPatch{
				PatchID:       "id1",
				ParameterPath: "p",
			},
		},
		{
			name: "missing parameter path",
			patch: interfaces.ConfigPatch{
				PatchID:             "id1",
				TargetProcessorName: component.MustNewID("test"),
			},
		},
		{
			name: "invalid source",
			patch: interfaces.ConfigPatch{
				PatchID:             "id1",
				TargetProcessorName: component.MustNewID("test"),
				ParameterPath:       "p",
				Severity:            "normal",
				Source:              "other",
			},
		},
		{
			name: "invalid severity",
			patch: interfaces.ConfigPatch{
				PatchID:             "id1",
				TargetProcessorName: component.MustNewID("test"),
				ParameterPath:       "p",
				Severity:            "bad",
				Source:              "manual",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := configpatch.NewStandardValidator(opts)
			err := v.Validate(tc.patch)
			assert.Error(t, err)
		})
	}
}

func TestStandardValidatorTTLExpiration(t *testing.T) {
	v := configpatch.NewStandardValidator(nil)
	patch := createPatch("ttl", "p")
	patch.Timestamp = time.Now().Add(-2 * time.Second).Unix()
	patch.TTLSeconds = 1

	err := v.Validate(patch)
	assert.Error(t, err)
}

func TestStandardValidatorRateLimit(t *testing.T) {
	opts := &configpatch.Options{MaxPatchesPerMinute: 1}
	v := configpatch.NewStandardValidator(opts)

	err := v.Validate(createPatch("p1", "a"))
	require.NoError(t, err)

	err = v.Validate(createPatch("p2", "b"))
	assert.Error(t, err)
}

func TestStandardValidatorCooldown(t *testing.T) {
	opts := &configpatch.Options{PatchCooldownSeconds: 1}
	v := configpatch.NewStandardValidator(opts)

	err := v.Validate(createPatch("p1", "param"))
	require.NoError(t, err)

	err = v.Validate(createPatch("p2", "param"))
	assert.Error(t, err)
}

func TestStandardValidatorHistoryAndReset(t *testing.T) {
	opts := &configpatch.Options{MaxPatchesPerMinute: 5}
	v := configpatch.NewStandardValidator(opts)

	p1 := createPatch("p1", "a")
	p2 := createPatch("p2", "b")

	require.NoError(t, v.Validate(p1))
	require.NoError(t, v.Validate(p2))

	history := v.GetHistory()
	require.Len(t, history, 2)
	assert.Equal(t, p2.PatchID, history[0].PatchID)
	assert.Equal(t, p1.PatchID, history[1].PatchID)

	v.Reset()
	assert.Len(t, v.GetHistory(), 0)

	// Ensure rate limit counters reset as well
	opts.MaxPatchesPerMinute = 1
	v = configpatch.NewStandardValidator(opts)
	require.NoError(t, v.Validate(createPatch("p3", "c")))
	err := v.Validate(createPatch("p4", "d"))
	assert.Error(t, err)

	v.Reset()
	err = v.Validate(createPatch("p5", "e"))
	assert.NoError(t, err)
}
