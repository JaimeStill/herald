package database

import "errors"

// ErrNotReady indicates the database connection has not been established.
var ErrNotReady = errors.New("database not ready")
