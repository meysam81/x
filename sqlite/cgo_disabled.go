// Registers the pure-Go modernc.org/sqlite driver as a CGO-free fallback.

//go:build !cgo
// +build !cgo

package sqlite

import (
	_ "modernc.org/sqlite"
)

const (
	ENGINE = "sqlite"
)
