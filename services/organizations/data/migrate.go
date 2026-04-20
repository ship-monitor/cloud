package data

import (
	"context"

	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func Migrate() {
	ctx := context.TODO()
	models := []interface{}{
		(*Organization)(nil),
	}

	for _, model := range models {
		_, err := db.DB.NewCreateTable().
			Model(model).
			IfNotExists().
			Exec(ctx)
		if err != nil {
			panic(err)
		}
	}
}
