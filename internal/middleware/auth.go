package middleware

import (
	"context"
	"net/http"

	"github.com/SimonLavlinskiy/finAns-backend/pkg/authtoken"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

const SessionCookieName = "finans_session"

type contextKey string

const claimsContextKey contextKey = "auth_claims"

func RequireAuth(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}

			claims, err := authtoken.Verify(secret, cookie.Value)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired session")
				return
			}

			ctx := context.WithValue(r.Context(), claimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ClaimsFromContext(ctx context.Context) (authtoken.Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(authtoken.Claims)
	return claims, ok
}
