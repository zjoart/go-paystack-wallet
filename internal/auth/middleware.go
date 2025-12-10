package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zjoart/go-paystack-wallet/internal/key"
	"github.com/zjoart/go-paystack-wallet/internal/user"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/utils"
)

func JWTMiddleware(cfg config.Config, userRepo user.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Authorization required", nil)
				return
			}

			tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method")
				}
				return []byte(cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid token", nil)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid token claims", nil)
				return
			}

			userIDStr, ok := claims[utils.UserIDKey].(string)
			if !ok {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid user ID in token", nil)
				return
			}

			usr, err := userRepo.FindByID(userIDStr)
			if err != nil {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "User not found", nil)
				return
			}

			ctx := context.WithValue(r.Context(), utils.UserKey, *usr)
			ctx = context.WithValue(ctx, utils.PermissionsKey, []string{"*"})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func APIKeyMiddleware(keyRepo key.Repository, userRepo user.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKeyHeader := r.Header.Get("x-api-key")
			if apiKeyHeader == "" {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "API Key required", nil)
				return
			}

			apiKey, err := keyRepo.FindByKey(apiKeyHeader)
			if err != nil {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid API Key", nil)
				return
			}

			if apiKey.IsRevoked {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "API Key revoked", nil)
				return
			}

			if time.Now().After(apiKey.ExpiresAt) {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "API key has expired", nil)
				return
			}

			usr, err := userRepo.FindByID(apiKey.UserID.String())
			if err != nil {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Associated user not found", nil)
				return
			}

			ctx := context.WithValue(r.Context(), utils.UserKey, *usr)
			ctx = context.WithValue(ctx, utils.PermissionsKey, apiKey.Permissions)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequirePermission(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			perms, ok := r.Context().Value(utils.PermissionsKey).([]string)
			if !ok {
				utils.BuildErrorResponse(w, http.StatusForbidden, "Permissions not found", nil)
				return
			}

			hasPerm := false
			for _, p := range perms {
				if p == "*" || p == perm {
					hasPerm = true
					break
				}
			}

			if !hasPerm {
				utils.BuildErrorResponse(w, http.StatusForbidden, "Insufficient permissions", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
