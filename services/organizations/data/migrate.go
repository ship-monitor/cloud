package data

import (
	"context"

	"charm.land/log/v2"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func Migrate() {
	ctx := context.TODO()
	models := []interface{}{
		(*Organization)(nil),
		(*OrganizationInvitation)(nil),
		(*OrganizationMember)(nil),
	}

	for _, model := range models {
		_, err := db.DB.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			log.Error("Failed migrate model", "error", err)
			panic(err)
		}
	}
}
