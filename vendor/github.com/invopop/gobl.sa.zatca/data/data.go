// Package data contains the embedded data for the module.
package data

import "embed"

//go:embed catalogues

// Content contains the catalogue definitions
// ready to serve as an embed.FS.
var Content embed.FS
