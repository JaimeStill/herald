package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"

	"github.com/JaimeStill/herald/pkg/auth"
)

// Auth returns middleware that validates Azure Entra ID JWT bearer tokens.
// When cfg.Mode is ModeNone, returns a pass-through that does not inspect
// requests. When ModeAzure, performs OIDC discovery on the first request,
// verifies token signature and claims, and injects the authenticated User
// into the request context.
func Auth(cfg *auth.Config, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if cfg.Mode == auth.ModeNone {
			return next
		}

		var (
			once     sync.Once
			verifier *oidc.IDTokenVerifier
			initErr  error
		)

		initVerifier := func() {
			provider, err := oidc.NewProvider(context.Background(), cfg.Authority)
			if err != nil {
				initErr = err
				return
			}

			verifier = provider.Verifier(&oidc.Config{
				ClientID:        "api://" + cfg.ClientID,
				SkipIssuerCheck: true,
			})
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, ok := extractBearer(r)
			if !ok {
				respondUnauthorized(w, auth.ErrUnauthorized)
				return
			}

			once.Do(initVerifier)
			if initErr != nil {
				logger.Error("oidc provider init failed", "error", initErr)
				respondUnauthorized(w, auth.ErrInvalidToken)
				return
			}

			idToken, err := verifier.Verify(r.Context(), tokenString)
			if err != nil {
				logger.Debug("token verification failed", "error", err)
				respondUnauthorized(w, mapVerifyError(err))
				return
			}

			var claims struct {
				OID               string `json:"oid"`
				Name              string `json:"name"`
				PreferredUsername string `json:"preferred_username"`
				Email             string `json:"email"`
				UPN               string `json:"upn"`
			}

			if err := idToken.Claims(&claims); err != nil {
				logger.Error("claim extraction failed", "error", err)
				respondUnauthorized(w, auth.ErrInvalidToken)
				return
			}

			user := &auth.User{
				ID:    claims.OID,
				Name:  firstNonEmpty(claims.Name, claims.PreferredUsername),
				Email: firstNonEmpty(claims.Email, claims.UPN),
			}

			ctx := auth.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearer(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	return strings.TrimPrefix(header, "Bearer "), true
}

func mapVerifyError(err error) error {
	if strings.Contains(err.Error(), "token is expired") {
		return auth.ErrTokenExpired
	}
	return auth.ErrInvalidToken
}

func respondUnauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
