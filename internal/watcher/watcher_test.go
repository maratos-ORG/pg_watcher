package watcher

import (
	"encoding/json"
	"math"
	"testing"
)

// Test toFloat64 conversion function
func TestToFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
		ok       bool
	}{
		{"nil", nil, 0, false},
		{"float64", float64(42.5), 42.5, true},
		{"float32", float32(42.5), 42.5, true},
		{"int", int(42), 42.0, true},
		{"int64", int64(42), 42.0, true},
		{"int32", int32(42), 42.0, true},
		{"int16", int16(42), 42.0, true},
		{"int8", int8(42), 42.0, true},
		{"uint", uint(42), 42.0, true},
		{"uint64", uint64(42), 42.0, true},
		{"uint32", uint32(42), 42.0, true},
		{"uint16", uint16(42), 42.0, true},
		{"uint8", uint8(42), 42.0, true},
		{"string valid", "42.5", 42.5, true},
		{"string invalid", "not a number", 0, false},
		{"bytes valid", []byte("42.5"), 42.5, true},
		{"bytes invalid", []byte("invalid"), 0, false},
		{"json.Number", json.Number("42.5"), 42.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := toFloat64(tt.input)
			if ok != tt.ok {
				t.Errorf("toFloat64(%v) ok = %v, want %v", tt.input, ok, tt.ok)
			}
			if ok && !floatEqual(result, tt.expected) {
				t.Errorf("toFloat64(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// Test normalizeName function
func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "MyColumn", "mycolumn"},
		{"dash to underscore", "my-column", "my_column"},
		{"dot to underscore", "my.column", "my_column"},
		{"space to underscore", "my column", "my_column"},
		{"multiple underscores", "my__column", "my_column"},
		{"starts with digit", "9column", "_9column"},
		{"complex", "My-Column.Name 123", "my_column_name_123"},
		{"already normalized", "my_column", "my_column"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test makeForcedLabelsSet function
func TestMakeForcedLabelsSet(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string]bool
	}{
		{
			"empty array",
			[]string{},
			map[string]bool{},
		},
		{
			"single element",
			[]string{"col1"},
			map[string]bool{"col1": true},
		},
		{
			"multiple elements",
			[]string{"col1", "col2", "col3"},
			map[string]bool{"col1": true, "col2": true, "col3": true},
		},
		{
			"with whitespace",
			[]string{" col1 ", "col2", " col3"},
			map[string]bool{"col1": true, "col2": true, "col3": true},
		},
		{
			"with empty strings",
			[]string{"col1", "", "col2", "  "},
			map[string]bool{"col1": true, "col2": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeForcedLabelsSet(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("makeForcedLabelsSet(%v) length = %d, want %d", tt.input, len(result), len(tt.expected))
			}
			for k := range tt.expected {
				if !result[k] {
					t.Errorf("makeForcedLabelsSet(%v) missing key %q", tt.input, k)
				}
			}
		})
	}
}

// Test labelVal function
func TestLabelVal(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "test", "test"},
		{"int", 42, "42"},
		{"float", 42.5, "42.5"},
		{"bytes", []byte("test"), "test"},
		{"bool", true, "true"},
		{"nil", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := labelVal(tt.input)
			if result != tt.expected {
				t.Errorf("labelVal(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Test Run function with nil parameters
func TestRunWithNilParams(t *testing.T) {
	tests := []struct {
		name    string
		fp      *FlagParam
		cp      *ConnectionString
		wantErr bool
		errMsg  string
	}{
		{
			"nil FlagParam",
			nil,
			&ConnectionString{connstr: "test"},
			true,
			"FlagParam cannot be nil",
		},
		{
			"nil ConnectionString",
			&FlagParam{},
			nil,
			true,
			"ConnectionString cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(nil, tt.fp, tt.cp)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Run() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// Test resolveDBList with specific database names
func TestResolveDBListSpecific(t *testing.T) {
	tests := []struct {
		name     string
		datnames []string
		expected []string
	}{
		{
			"single database",
			[]string{"mydb"},
			[]string{"mydb"},
		},
		{
			"multiple databases",
			[]string{"db1", "db2", "db3"},
			[]string{"db1", "db2", "db3"},
		},
		{
			"empty list",
			[]string{},
			[]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flagParam for the test
			flagParam = FlagParam{
				datname: tt.datnames,
			}

			result, err := resolveDBList(nil)
			if err != nil {
				t.Errorf("resolveDBList() unexpected error = %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("resolveDBList() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for i, db := range result {
				if db != tt.expected[i] {
					t.Errorf("resolveDBList()[%d] = %v, want %v", i, db, tt.expected[i])
				}
			}
		})
	}
}

// Helper function to compare floats
func floatEqual(a, b float64) bool {
	const epsilon = 1e-9
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	if math.IsInf(a, 1) && math.IsInf(b, 1) {
		return true
	}
	if math.IsInf(a, -1) && math.IsInf(b, -1) {
		return true
	}
	return math.Abs(a-b) < epsilon
}
