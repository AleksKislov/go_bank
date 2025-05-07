package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"banking-service/pkg/utils"
)

// AuthMiddleware checks if the request has a valid JWT token
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.RespondWithError(w, http.StatusUnauthorized, "no authorization header provided")
				return
			}
			
			// Check if the Authorization header has the Bearer prefix
			if !strings.HasPrefix(authHeader, "Bearer ") {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
			
			// Extract the token
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			
			// Parse and validate the token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				
				return []byte(jwtSecret), nil
			})
			
			if err != nil {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token: "+err.Error())
				return
			}
			
			// Extract claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				// Get user ID from claims
				userID, ok := claims["user_id"]
				if !ok {
					utils.RespondWithError(w, http.StatusUnauthorized, "invalid token: missing user_id claim")
					return
				}
				
				// Convert user ID to float64 (JSON numbers are float64)
				userIDFloat, ok := userID.(float64)
				if !ok {
					utils.RespondWithError(w, http.StatusUnauthorized, "invalid token: user_id has wrong type")
					return
				}
				
				// Add user ID to request context
				ctx := context.WithValue(r.Context(), "user_id", int(userIDFloat))
				
				// Call the next handler with the updated context
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				utils.RespondWithError(w, http.StatusUnauthorized, "invalid token")
				return
			}
		})
	}
}