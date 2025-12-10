package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zjoart/go-paystack-wallet/pkg/utils"
)

func TestRequirePermission(t *testing.T) {
	tests := []struct {
		name           string
		userPerms      []string
		requiredPerm   string
		expectedStatus int
	}{
		{
			name:           "JWT User (Wildcard) - Access Granted",
			userPerms:      []string{"*"},
			requiredPerm:   "DEPOSIT",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API Key (Exact Match) - Access Granted",
			userPerms:      []string{"DEPOSIT"},
			requiredPerm:   "DEPOSIT",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API Key (Superset) - Access Granted",
			userPerms:      []string{"DEPOSIT", "WITHDRAWAL"},
			requiredPerm:   "DEPOSIT",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API Key (Missing Perm) - Access Denied",
			userPerms:      []string{"READ"},
			requiredPerm:   "DEPOSIT",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "No Perms - Access Denied",
			userPerms:      []string{},
			requiredPerm:   "READ",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RequirePermission(tt.requiredPerm)(nextHandler)

			req := httptest.NewRequest("GET", "/", nil)
			ctx := context.WithValue(req.Context(), utils.PermissionsKey, tt.userPerms)
			req = req.WithContext(ctx)

			rr := httptest.NewRecorder()

			middleware.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
