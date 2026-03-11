package auth

import "context"

type contextKey struct{}

var userKey = contextKey{}

// User represents an authenticated user extracted from JWT claims.
// ID is the Azure AD object identifier (oid claim), Name is the display
// name with preferred_username fallback, and Email is the email with
// upn fallback.
type User struct {
	ID    string
	Name  string
	Email string
}

// ContextWithUser returns a copy of ctx with the given User attached.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// UserFromContext extracts the User from ctx. Returns nil if no user is present.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(userKey).(*User)
	return user
}
