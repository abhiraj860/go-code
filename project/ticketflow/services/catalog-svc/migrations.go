// Package catalog embeds the service's SQL migrations into its binary.
//
// Embedding rather than shipping a migrations directory alongside the image
// means a binary and the schema it expects cannot drift apart in a registry --
// a prerequisite for the blue/green rollouts in Phase 5, where a new version
// must boot and migrate itself with no human running anything first.
package catalog

import "embed"

//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path within Migrations holding the .sql files.
const MigrationsDir = "migrations"
