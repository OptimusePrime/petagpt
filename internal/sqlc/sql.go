package sqlc

import _ "embed"

//go:embed sql/schema.sql
var DDL string
