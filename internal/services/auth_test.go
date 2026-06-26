package services_test

import (
	"testing"

	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/internal/services"
)

func TestCheckPassword(t *testing.T) {
	t.Parallel()

	password := "password"
	hashed := services.HashPassword(password)

	if !services.CheckPassword(hashed, password) {
		t.Fatalf("check password: expected true, got false")
	}

	if services.CheckPassword(hashed, "wrongpassword") {
		t.Fatalf("check password: expected false, got true")
	}
}
