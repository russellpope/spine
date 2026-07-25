// Package models embeds the estate's model table — the versioned
// (flavor, tier) -> model id + effort defaults, plus every default ever
// shipped per entry — that compiles into the spine binary. Sibling to
// ../templates, embedded the same way (see templates/embed.go); this data is
// consumed by internal/model, not read directly.
package models

import "embed"

//go:embed defaults.json
var FS embed.FS
