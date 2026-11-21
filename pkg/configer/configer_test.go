package configer_test

import (
	"slices"
	"testing"

	"github.com/ship-monitor/cloud/pkg/configer"
)

//nolint:goconst, paralleltest
func TestGetOrDefault(t *testing.T) {
	testCases := []struct {
		name          string
		key           string
		defaultValue  string
		setup         func(t *testing.T)
		expectedValue string
	}{
		{
			name:         "ok",
			key:          "key",
			defaultValue: "default",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("key", "expected")
			},
			expectedValue: "expected",
		},
		{
			name:          "ok/default",
			key:           "key",
			defaultValue:  "default",
			setup:         func(*testing.T) {},
			expectedValue: "default",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			value := configer.GetOrDefault(tt.key, tt.defaultValue)
			if value != tt.expectedValue {
				t.Errorf(
					"expected %q, but returned %q",
					tt.expectedValue,
					value,
				)
			}
		})
	}
}

//nolint:paralleltest
func TestGetStrings(t *testing.T) {
	testCases := []struct {
		name          string
		key           string
		separator     string
		setup         func(t *testing.T)
		expectedValue []string
	}{
		{
			name:      "ok",
			key:       "key",
			separator: ":",
			setup: func(t *testing.T) {
				t.Helper()

				t.Setenv("key", "one:two: three : four")
			},
			expectedValue: []string{"one", "two", "three", "four"},
		},
		{
			name:      "ok/space-separator",
			key:       "key",
			separator: ", ",
			setup: func(t *testing.T) {
				t.Helper()

				t.Setenv("key", "one, two,three")
			},
			expectedValue: []string{"one", "two,three"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			values := configer.GetStrings(tt.key, tt.separator)
			if !slices.Equal(tt.expectedValue, values) {
				t.Errorf(
					"expected %v, but returned %v",
					tt.expectedValue,
					values,
				)
			}
		})
	}
}
