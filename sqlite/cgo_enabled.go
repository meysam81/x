//go:build cgo
// +build cgo

package sqlite

import (
	_ "github.com/mattn/go-sqlite3"
)

const (
	ENGINE = "sqlite3"
)
