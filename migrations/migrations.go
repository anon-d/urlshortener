// Package migrations содержит встроенные SQL-миграции для базы данных.
package migrations

import (
	"embed"
)

//go:embed *.sql
var Migrations embed.FS
