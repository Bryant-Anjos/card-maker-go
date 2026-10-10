// Package schema embeds the JSON Schemas that define the card data contract,
// so Go programs and non-Go consumers share a single source of truth.
package schema

import "embed"

// FS holds cards.schema.json, theme.schema.json, manifest.schema.json and the
// shared defs.schema.json.
//
//go:embed *.schema.json
var FS embed.FS
