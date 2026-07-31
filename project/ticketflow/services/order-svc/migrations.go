// Package order embeds the service's SQL migrations into its binary.
package order

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path within Migrations holding the .sql files.
const MigrationsDir = "migrations"
