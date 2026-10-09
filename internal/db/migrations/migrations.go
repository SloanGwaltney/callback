// Package migrations embeds the goose migrations that build callback's database schema. They run
// when the app starts. See docs/adr/0006-run-database-migrations-with-goose.md.
package migrations

import "embed"

// FS holds every migration file.
//
//go:embed *.sql
var FS embed.FS
