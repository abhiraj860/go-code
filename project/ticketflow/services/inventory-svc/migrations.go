// Package inventory embeds the service's SQL migrations into its binary, so a
// binary and the schema it expects cannot drift apart in a registry.
package inventory

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path within Migrations holding the .sql files.
const MigrationsDir = "migrations"
