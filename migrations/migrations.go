package migrations

import (
	"embed"
	_ "embed"
)

//go:embed *.sql
var Migrations embed.FS
