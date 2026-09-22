package router

import (
	"itdb-backend/router/common"
	"itdb-backend/router/settings"
	"net/http"
	"strings"
	"time"

	_ "itdb-backend/docs"

	"github.com/golang-jwt/jwt/v5"
	_ "modernc.org/sqlite"
)

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]struct{}{}
	for _, origin := range a.cfg.CORSOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAll = true
			break
		}
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := strings.TrimSpace(r.Header.Get("Authorization"))
		if auth == "" {
			common.WriteError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		tokenParts := strings.SplitN(auth, " ", 2)
		if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
			common.WriteError(w, http.StatusUnauthorized, "invalid Authorization header")
			return
		}

		tokenStr := strings.TrimSpace(tokenParts[1])
		claims := &AuthClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(a.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			common.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		if strings.TrimSpace(claims.ID) == "" {
			common.WriteError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		var expiresAt int64
		if e := a.db.QueryRowContext(r.Context(), "SELECT expires_at FROM user_sessions WHERE jti=?", claims.ID).Scan(&expiresAt); e != nil || expiresAt < time.Now().Unix() {
			common.WriteError(w, http.StatusUnauthorized, "session expired or revoked")
			return
		}

		user := SessionUser{ID: claims.UserID, Username: claims.Username, UserType: claims.UserType, Source: claims.Source}
		ctx := common.WithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requirePermission 返回校验单一权限的中间件，无权限时返回 403
func (a *App) requirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := common.CurrentUser(r.Context())
			if err != nil {
				common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
				return
			}
			if settings.HasPermission(a.db, user, permission) {
				next.ServeHTTP(w, r)
				return
			}
			common.WriteError(w, http.StatusForbidden, "permission denied")
		})
	}
}

// requireAnyPermission 返回校验任一权限即可通过的中间件
func (a *App) requireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := common.CurrentUser(r.Context())
			if err != nil {
				common.WriteError(w, http.StatusUnauthorized, "unauthenticated")
				return
			}
			for _, permission := range permissions {
				if settings.HasPermission(a.db, user, permission) {
					next.ServeHTTP(w, r)
					return
				}
			}
			common.WriteError(w, http.StatusForbidden, "permission denied")
		})
	}
}

// currentUserPermissions 返回当前请求用户的全部有效权限 Key，管理员返回全量权限
func (a *App) currentUserPermissions(r *http.Request) []string {
	user, err := common.CurrentUser(r.Context())
	if err != nil {
		return nil
	}
	if user.UserType == 0 || strings.EqualFold(user.Username, "admin") {
		return settings.AllPermissionKeys()
	}
	_, _, permissions, err := settings.UserAccess(a.db, user.ID, user.UserType)
	if err != nil {
		return nil
	}
	return permissions
}
