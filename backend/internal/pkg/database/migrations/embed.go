package migrations

import "embed"

// FS contains the ordered, versioned schema migrations applied by the server.
//
//go:embed *.sql
var FS embed.FS
