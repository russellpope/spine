// Package templates embeds the workflow template generations that compile
// into the spine binary. gen0 is the pre-versioning 2026-06-28 generation,
// kept only so update can claim legacy files.
package templates

import "embed"

//go:embed VERSION current gen0
var FS embed.FS
