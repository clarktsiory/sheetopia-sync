package sql

import (
	"embed"
	_ "embed"
)

//go:embed all:migrations
var MigrationsFS embed.FS
