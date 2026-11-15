package config_test

import (
	"testing"

	"github.com/ship-monitor/cloud/internal/v2/config"
)

//nolint:goconst, paralleltest, varnamelen
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

			value := config.GetOrDefault(tt.key, tt.defaultValue)
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
