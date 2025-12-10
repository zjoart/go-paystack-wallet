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
			usr, err := validateJWT(tokenString, cfg.JWTSecret, userRepo)
			if err != nil {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, err.Error(), nil)
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

			usr, perms, err := validateAPIKey(apiKeyHeader, keyRepo, userRepo)
			if err != nil {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, err.Error(), nil)
				return
			}

			ctx := context.WithValue(r.Context(), utils.UserKey, *usr)
			ctx = context.WithValue(ctx, utils.PermissionsKey, perms)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UnifiedAuthMiddleware(cfg config.Config, userRepo user.Repository, keyRepo key.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			apiKeyHeader := r.Header.Get("x-api-key")

			if authHeader != "" {
				tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
				usr, err := validateJWT(tokenString, cfg.JWTSecret, userRepo)
				if err != nil {
					utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid token: "+err.Error(), nil)
					return
				}

				ctx := context.WithValue(r.Context(), utils.UserKey, *usr)
				ctx = context.WithValue(ctx, utils.PermissionsKey, []string{"*"})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			} else if apiKeyHeader != "" {
				usr, perms, err := validateAPIKey(apiKeyHeader, keyRepo, userRepo)
				if err != nil {
					utils.BuildErrorResponse(w, http.StatusUnauthorized, "Invalid API Key: "+err.Error(), nil)
					return
				}
				ctx := context.WithValue(r.Context(), utils.UserKey, *usr)
				ctx = context.WithValue(ctx, utils.PermissionsKey, perms)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			} else {
				utils.BuildErrorResponse(w, http.StatusUnauthorized, "Authorization required", nil)
				return
			}
		})
	}
}

// Helpers

func validateJWT(tokenString, secret string, userRepo user.Repository) (*user.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	userIDStr, ok := claims[utils.UserIDKey].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user ID in token")
	}

	usr, err := userRepo.FindByID(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	return usr, nil
}

func validateAPIKey(keyStr string, keyRepo key.Repository, userRepo user.Repository) (*user.User, []string, error) {
	apiKey, err := keyRepo.FindByKey(keyStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid API Key")
	}

	if apiKey.IsRevoked {
		return nil, nil, fmt.Errorf("API Key revoked")
	}

	if time.Now().After(apiKey.ExpiresAt) {
		return nil, nil, fmt.Errorf("API key has expired")
	}

	usr, err := userRepo.FindByID(apiKey.UserID.String())
	if err != nil {
		return nil, nil, fmt.Errorf("associated user not found")
	}
	return usr, apiKey.Permissions, nil
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
