package data_test

import (
	"testing"

	"github.com/google/uuid"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/services/organizations/data"
)

func TestGen(t *testing.T) {
	id := uuid.New()

	name1 := data.GenNodeName(id)
	name2 := data.GenNodeName(id)

	if name1 == "" {
		t.Errorf("GenNodeName() returned empty name for id %q", id)
	}
	if name2 == "" {
		t.Errorf("GenNodeName() returned empty name for id %q", id)
	}

	if name1 != name2 {
		t.Errorf(
			"GenNodeName() returned different names for same id %q: %q != %q",
			id,
			name1,
			name2,
		)
	}
}
