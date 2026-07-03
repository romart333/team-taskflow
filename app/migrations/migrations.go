// Package migrations embeds SQL migration files for the service database.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
