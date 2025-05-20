// Package typeconv provides utility functions for type conversion.
package typeconv

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		hasError bool
	}{
		{"nil", nil, 0, true},
		{"float64", float64(123.45), 123.45, false},
		{"float32", float32(123.45), 123.45, false},
		{"int", int(123), 123.0, false},
		{"int64", int64(123), 123.0, false},
		{"int32", int32(123), 123.0, false},
		{"int16", int16(123), 123.0, false},
		{"int8", int8(123), 123.0, false},
		{"uint", uint(123), 123.0, false},
		{"uint64", uint64(123), 123.0, false},
		{"uint32", uint32(123), 123.0, false},
		{"uint16", uint16(123), 123.0, false},
		{"uint8", uint8(123), 123.0, false},
		{"bool_true", true, 1.0, false},
		{"bool_false", false, 0.0, false},
		{"string_valid", "123.45", 123.45, false},
		{"string_invalid", "not a number", 0, true},
		{"struct", struct{}{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToFloat64(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Use InDelta for floating point comparisons to handle precision issues
				// especially with float32 to float64 conversions
				if tt.name == "float32" {
					assert.InDelta(t, tt.expected, result, 0.001)
				} else {
					assert.Equal(t, tt.expected, result)
				}
			}
		})
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int64
		hasError bool
	}{
		{"nil", nil, 0, true},
		{"int", int(123), 123, false},
		{"int64", int64(123), 123, false},
		{"int32", int32(123), 123, false},
		{"int16", int16(123), 123, false},
		{"int8", int8(123), 123, false},
		{"uint", uint(123), 123, false},
		{"uint64", uint64(123), 123, false},
		{"uint32", uint32(123), 123, false},
		{"uint16", uint16(123), 123, false},
		{"uint8", uint8(123), 123, false},
		{"float64", float64(123.45), 123, false},
		{"float32", float32(123.45), 123, false},
		{"bool_true", true, 1, false},
		{"bool_false", false, 0, false},
		{"string_valid", "123", 123, false},
		{"string_invalid", "not a number", 0, true},
		{"uint64_overflow", uint64(9223372036854775808), 0, true}, // 2^63 (overflows int64)
		{"struct", struct{}{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToInt64(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
		hasError bool
	}{
		{"nil", nil, 0, true},
		{"int", int(123), 123, false},
		{"int64", int64(123), 123, false},
		{"int32", int32(123), 123, false},
		{"int16", int16(123), 123, false},
		{"int8", int8(123), 123, false},
		{"uint", uint(123), 123, false},
		{"uint32", uint32(123), 123, false},
		{"uint16", uint16(123), 123, false},
		{"uint8", uint8(123), 123, false},
		{"float64", float64(123.45), 123, false},
		{"float32", float32(123.45), 123, false},
		{"bool_true", true, 1, false},
		{"bool_false", false, 0, false},
		{"string_valid", "123", 123, false},
		{"string_invalid", "not a number", 0, true},
		// This test may behave differently on 32-bit platforms
		{"int64_maybe_overflow", int64(2147483648), 2147483648, false}, // 2^31, could overflow on 32-bit platforms
		{"struct", struct{}{}, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToInt(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected bool
		hasError bool
	}{
		{"nil", nil, false, true},
		{"bool_true", true, true, false},
		{"bool_false", false, false, false},
		{"int_zero", int(0), false, false},
		{"int_nonzero", int(1), true, false},
		{"int64_zero", int64(0), false, false},
		{"int64_nonzero", int64(1), true, false},
		{"int32_zero", int32(0), false, false},
		{"int32_nonzero", int32(1), true, false},
		{"uint_zero", uint(0), false, false},
		{"uint_nonzero", uint(1), true, false},
		{"uint64_zero", uint64(0), false, false},
		{"uint64_nonzero", uint64(1), true, false},
		{"uint32_zero", uint32(0), false, false},
		{"uint32_nonzero", uint32(1), true, false},
		{"float64_zero", float64(0), false, false},
		{"float64_nonzero", float64(1), true, false},
		{"float32_zero", float32(0), false, false},
		{"float32_nonzero", float32(1), true, false},
		{"string_true", "true", true, false},
		{"string_false", "false", false, false},
		{"string_1", "1", true, false},
		{"string_0", "0", false, false},
		{"string_invalid", "not a bool", false, true},
		{"struct", struct{}{}, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ToBool(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil", nil, ""},
		{"string", "test", "test"},
		{"int", 123, "123"},
		{"float", 123.45, "123.45"},
		{"bool", true, "true"},
		{"struct", struct{}{}, "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected bool
	}{
		{"Int", reflect.Int, true},
		{"Int8", reflect.Int8, true},
		{"Int16", reflect.Int16, true},
		{"Int32", reflect.Int32, true},
		{"Int64", reflect.Int64, true},
		{"Uint", reflect.Uint, true},
		{"Uint8", reflect.Uint8, true},
		{"Uint16", reflect.Uint16, true},
		{"Uint32", reflect.Uint32, true},
		{"Uint64", reflect.Uint64, true},
		{"Float32", reflect.Float32, true},
		{"Float64", reflect.Float64, true},
		{"String", reflect.String, false},
		{"Bool", reflect.Bool, false},
		{"Struct", reflect.Struct, false},
		{"Map", reflect.Map, false},
		{"Slice", reflect.Slice, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNumeric(tt.kind)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInteger(t *testing.T) {
	tests := []struct {
		name     string
		kind     reflect.Kind
		expected bool
	}{
		{"Int", reflect.Int, true},
		{"Int8", reflect.Int8, true},
		{"Int16", reflect.Int16, true},
		{"Int32", reflect.Int32, true},
		{"Int64", reflect.Int64, true},
		{"Uint", reflect.Uint, true},
		{"Uint8", reflect.Uint8, true},
		{"Uint16", reflect.Uint16, true},
		{"Uint32", reflect.Uint32, true},
		{"Uint64", reflect.Uint64, true},
		{"Float32", reflect.Float32, false},
		{"Float64", reflect.Float64, false},
		{"String", reflect.String, false},
		{"Bool", reflect.Bool, false},
		{"Struct", reflect.Struct, false},
		{"Map", reflect.Map, false},
		{"Slice", reflect.Slice, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInteger(tt.kind)
			assert.Equal(t, tt.expected, result)
		})
	}
}