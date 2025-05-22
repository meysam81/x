package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type connectionOption struct {
	connMaxLifetime int
	connMaxIdleTime int
	connMaxOpen     int
	mode            string
	journalMode     string
}

func WithConnMaxLifetime(connMaxLifetime int) func(*connectionOption) {
	return func(opt *connectionOption) {
		opt.connMaxLifetime = connMaxLifetime
	}
}

func WithConnMaxIdleTime(connMaxIdleTime int) func(*connectionOption) {
	return func(opt *connectionOption) {
		opt.connMaxIdleTime = connMaxIdleTime
	}
}

func WithConnMaxOpen(connMaxOpen int) func(*connectionOption) {
	return func(opt *connectionOption) {
		opt.connMaxOpen = connMaxOpen
	}
}

func WithMode(mode string) func(*connectionOption) {
	return func(opt *connectionOption) {
		opt.mode = mode
	}
}

func WithJournalMode(journalMode string) func(*connectionOption) {
	return func(opt *connectionOption) {
		opt.journalMode = journalMode
	}
}

// NewDB creates a new SQLite database connection with the specified options.
// Provide the filepath only as the relative or absolute path to the database file.
// For options, you can use the provided functions to set the connection parameters.
func NewDB(ctx context.Context, filepath string, opts ...func(*connectionOption)) (*sql.DB, error) {
	if filepath == "" {
		return nil, errors.New("filepath is empty")
	}

	_, err := os.OpenFile(filepath, os.O_RDWR, 0644)
	if err != nil {
		if os.IsNotExist(err) {
			_, err = os.Create(filepath)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	options := &connectionOption{
		connMaxLifetime: 600,
		connMaxIdleTime: 60,
		connMaxOpen:     1,
		mode:            "rwc",
		journalMode:     "wal",
	}
	for _, opt := range opts {
		opt(options)
	}

	params := make([]string, 2)

	if options.mode != "" {
		params = append(params, "mode="+options.mode)
	}

	if options.journalMode != "" {
		params = append(params, "journal_mode="+options.journalMode)
	}

	dsn := filepath

	switch len(params) {
	case 0:
	case 1:
		dsn += "?" + params[0]
	default:
		dsn += "?" + params[0]
		for i := 1; i < len(params); i++ {
			dsn += "&" + params[i]
		}
	}

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxIdleTime(time.Duration(options.connMaxIdleTime) * time.Second)
	db.SetConnMaxLifetime(time.Duration(options.connMaxLifetime) * time.Second)
	db.SetMaxIdleConns(options.connMaxOpen)
	db.SetMaxOpenConns(options.connMaxOpen)

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
