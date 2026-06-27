package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/SimonLavlinskiy/finAns-backend/internal/domain"
	"github.com/SimonLavlinskiy/finAns-backend/internal/apperrors"
	"github.com/SimonLavlinskiy/finAns-backend/pkg/httputil"
)

type contextKey string

const (
	userContextKey      contextKey = "user"
	projectIDContextKey contextKey = "project_id"
)

func UserContextMiddleware(userRepo domain.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("X-User-ID")
			if header == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "X-User-ID header required")
				return
			}
			userID, err := strconv.ParseInt(header, 10, 64)
			if err != nil {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid X-User-ID")
				return
			}
			user, err := userRepo.GetByID(r.Context(), userID)
			if err != nil {
				var nf *apperrors.NotFoundError
				if ok := isNotFound(err, &nf); ok {
					httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found")
					return
				}
				httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to load user")
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ProjectContextMiddleware(projectRepo domain.ProjectRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("X-Project-ID")
			if header == "" {
				httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "X-Project-ID header required")
				return
			}
			projectID, err := strconv.ParseInt(header, 10, 64)
			if err != nil {
				httputil.WriteError(w, http.StatusBadRequest, "PROJECT_ID_REQUIRED", "invalid X-Project-ID")
				return
			}
			user, ok := UserFromContext(r.Context())
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}
			_, err = projectRepo.GetMember(r.Context(), projectID, user.ID)
			if err != nil {
				var nf *apperrors.NotFoundError
				if ok := isNotFound(err, &nf); ok {
					httputil.WriteError(w, http.StatusForbidden, "FORBIDDEN", "not a member of this project")
					return
				}
				httputil.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to verify project membership")
				return
			}
			ctx := context.WithValue(r.Context(), projectIDContextKey, projectID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserFromContext(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(userContextKey).(domain.User)
	return u, ok
}

func ProjectIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(projectIDContextKey).(int64)
	return id, ok
}

func WithProjectID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, projectIDContextKey, id)
}

func isNotFound(err error, target **apperrors.NotFoundError) bool {
	if err == nil {
		return false
	}
	nf, ok := err.(*apperrors.NotFoundError)
	if ok && target != nil {
		*target = nf
	}
	return ok
}
