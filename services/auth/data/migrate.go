package data

import (
	"context"

	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Migrate() {
	ctx := context.TODO()

	_, err := db.DB.NewCreateTable().
		Model((*User)(nil)).
		IfNotExists().Exec(ctx)

	panicIfErr(err)

	_, err = db.DB.NewCreateIndex().
		Model((*User)(nil)).
		Index("idx_users_email").
		Column("email").
		Exec(ctx)
	panicIfErr(err)
}
