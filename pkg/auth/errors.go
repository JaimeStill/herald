package auth

import "errors"

var (
	// ErrUnauthorized indicates no valid credentials were provided.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrTokenExpired indicates the JWT has passed its expiration time.
	ErrTokenExpired = errors.New("token expired")
	// ErrInvalidToken indicates the JWT is malformed, has an invalid signature,
	// or fails claims validation.
	ErrInvalidToken = errors.New("invalid token")
)
