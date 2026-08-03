package domain_test

import (
	"testing"

	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/domain"
)

func TestDeviceAccessRoleValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		role  domain.DeviceAccessRole
		valid bool
	}{
		{
			name:  "admin",
			role:  domain.DeviceAccessRoleAdmin,
			valid: true,
		},
		{
			name:  "viewer",
			role:  domain.DeviceAccessRoleViewer,
			valid: true,
		},
		{
			name: "owner",
			role: domain.DeviceAccessRole("owner"),
		},
		{
			name: "empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.role.Valid(); got != test.valid {
				t.Fatalf("expected valid=%t, got %t", test.valid, got)
			}
		})
	}
}
