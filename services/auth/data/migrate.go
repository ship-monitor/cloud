package data

import (
	"context"
	"fmt"

	"charm.land/log/v2"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func Migrate() {
	ctx := context.TODO()

	query := db.DB.NewCreateTable().
		Model((*User)(nil)).
		IfNotExists()

	_, err := query.Exec(ctx)

	if err != nil {
		fmt.Println(query.String())
		log.Error("Failed create table", "table", query.GetTableName(), "query", query.String(), "error", err)
	}
	panicIfErr(err)
}
