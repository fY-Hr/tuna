package Handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// Unexported context-key type prevents collisions with other packages
// that might stash values under a plain "userID" string.
type contextKey string

const userIDKey contextKey = "userID"

func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Read Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Strip "Bearer " and parse JWT
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
			return s.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			writeError(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 3. Extract userID
		uidFloat, ok := claims["user_id"].(float64)
		if !ok {
			writeError(w, "UserID not found", http.StatusUnauthorized)
			return // without this, the request continues after the 401!
		}
		userID := int64(uidFloat) // JSON numbers decode as float64; handlers expect int64

		// 4. Stash userID into the request's context backpack!
		ctx := context.WithValue(r.Context(), userIDKey, userID)

		// 5. Let the user walk through the door into the real handler!
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
