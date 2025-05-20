package typeconv_test

import (
	"math"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/deepaucksharma/Phoenix/pkg/util/typeconv"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		ok       bool
	}{
		{"int", 5, 5.0, true},
		{"uint", uint(7), 7.0, true},
		{"float32", float32(1.5), 1.5, true},
		{"float64", 2.3, 2.3, true},
		{"string", "3.14", 3.14, true},
		{"bool true", true, 1.0, true},
		{"bool false", false, 0.0, true},
		{"invalid string", "abc", 0, false},
		{"nil", nil, 0, false},
		{"unsupported", struct{}{}, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := typeconv.ToFloat64(tt.input)
			if tt.ok {
				require.NoError(t, err)
				assert.InDelta(t, tt.expected, v, 1e-9)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	large := uint64(math.MaxInt64) + 1
	tests := []struct {
		name     string
		input    interface{}
		expected int64
		ok       bool
	}{
		{"int", 5, 5, true},
		{"int64", int64(8), 8, true},
		{"uint", uint(9), 9, true},
		{"float64", 3.7, 3, true},
		{"string", "42", 42, true},
		{"bool", true, 1, true},
		{"overflow", large, 0, false},
		{"invalid string", "abc", 0, false},
		{"nil", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := typeconv.ToInt64(tt.input)
			if tt.ok {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, v)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	large := uint64(math.MaxInt64) + 1
	tests := []struct {
		name     string
		input    interface{}
		expected int
		ok       bool
	}{
		{"int", 5, 5, true},
		{"string", "10", 10, true},
		{"overflow", large, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := typeconv.ToInt(tt.input)
			if tt.ok {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, v)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
		ok       bool
	}{
		{"bool", true, true, true},
		{"int zero", 0, false, true},
		{"int nonzero", 2, true, true},
		{"float", 0.0, false, true},
		{"string", "true", true, true},
		{"invalid string", "nope", false, false},
		{"nil", nil, false, false},
		{"unsupported", struct{}{}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := typeconv.ToBool(tt.input)
			if tt.ok {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, v)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestKindChecks(t *testing.T) {
	numeric := map[reflect.Kind]bool{
		reflect.Int:     true,
		reflect.Int8:    true,
		reflect.Int16:   true,
		reflect.Int32:   true,
		reflect.Int64:   true,
		reflect.Uint:    true,
		reflect.Uint8:   true,
		reflect.Uint16:  true,
		reflect.Uint32:  true,
		reflect.Uint64:  true,
		reflect.Float32: true,
		reflect.Float64: true,
	}

	integer := map[reflect.Kind]bool{
		reflect.Int:    true,
		reflect.Int8:   true,
		reflect.Int16:  true,
		reflect.Int32:  true,
		reflect.Int64:  true,
		reflect.Uint:   true,
		reflect.Uint8:  true,
		reflect.Uint16: true,
		reflect.Uint32: true,
		reflect.Uint64: true,
	}

	kinds := []reflect.Kind{
		reflect.Invalid,
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128,
		reflect.Array,
		reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice,
		reflect.String,
		reflect.Struct,
		reflect.UnsafePointer,
	}

	for _, k := range kinds {
		assert.Equal(t, numeric[k], typeconv.IsNumeric(k), "IsNumeric(%v)", k)
		assert.Equal(t, integer[k], typeconv.IsInteger(k), "IsInteger(%v)", k)
	}
}
