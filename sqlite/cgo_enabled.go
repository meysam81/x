// Registers the CGO-based mattn/go-sqlite3 driver as the SQLite engine.

//go:build cgo
// +build cgo

package sqlite

import (
	_ "github.com/mattn/go-sqlite3"
)

const (
	ENGINE = "sqlite3"
)
