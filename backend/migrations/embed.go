package migrations

import "embed"

// Files contains the ordered database migrations shipped with the API.
//
//go:embed *.sql
var Files embed.FS
