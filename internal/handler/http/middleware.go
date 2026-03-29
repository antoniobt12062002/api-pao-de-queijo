package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyRole   contextKey = "role"
)

// JWTMiddleware valida o token Bearer e injeta user_id e role no contexto.
// Retorna 401 se o token estiver ausente ou inválido.
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims["sub"])
			ctx = context.WithValue(ctx, contextKeyRole, claims["role"])
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly é um middleware que rejeita requisições de usuários sem role "admin".
// Deve ser usado após JWTMiddleware.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(contextKeyRole).(string)
		if role != "admin" {
			writeError(w, http.StatusForbidden, "admin role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserIDFromContext extrai o user_id do contexto da requisição.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(contextKeyUserID).(string)
	return id
}

// RoleFromContext extrai o role do contexto da requisição.
func RoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(contextKeyRole).(string)
	return role
}
