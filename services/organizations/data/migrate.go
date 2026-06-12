package data

import (
	"context"
	"fmt"

	"charm.land/log/v2"
	"sourcecraft.dev/organization-shipmonitor/ship-cloud-auth/db"
)

func Migrate() {
	ctx := context.TODO()

	// Создаём таблицы, если их нет
	models := []interface{}{
		(*Organization)(nil),
		(*OrganizationInvitation)(nil),
		(*OrganizationMember)(nil),
		(*OrganizationDevice)(nil),
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

	// Добавляем колонки, если таблица уже существовала без них
	addColumnIfNotExists(ctx, "organization_members", "role", "VARCHAR NOT NULL DEFAULT 'member'")
	addColumnIfNotExists(
		ctx,
		"organization_members",
		"joined_at",
		"TIMESTAMP NOT NULL DEFAULT NOW()",
	)
	addColumnIfNotExists(
		ctx,
		"organization_devices",
		"name",
		fmt.Sprintf("VARCHAR NOT NULL DEFAULT '%s'", DefaultDeviceName),
	)
}

// addColumnIfNotExists добавляет колонку, если её ещё нет.
func addColumnIfNotExists(ctx context.Context, table, column, columnType string) {
	// Проверяем существование колонки
	var count int
	err := db.DB.NewRaw(
		`SELECT COUNT(*) FROM information_schema.columns 
		 WHERE table_name = ? AND column_name = ?`,
		table, column,
	).Scan(ctx, &count)
	if err != nil {
		log.Error("Failed check column existence", "table", table, "column", column, "error", err)

		return
	}

	if count == 0 {
		query := "ALTER TABLE " + table + " ADD COLUMN IF NOT EXISTS " + column + " " + columnType
		_, err := db.DB.NewRaw(query).Exec(ctx)
		if err != nil {
			log.Error("Failed add column", "table", table, "column", column, "error", err)
			panic(err)
		}
		log.Info("Added column", "table", table, "column", column)
	}
}
