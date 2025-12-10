package key

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/zjoart/go-paystack-wallet/internal/user"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/utils"
)

type Handler struct {
	Config config.Config
	Repo   Repository
}

func NewHandler(cfg config.Config, repo Repository) *Handler {
	return &Handler{Config: cfg, Repo: repo}
}

type CreateKeyRequest struct {
	Name        string   `json:"name"`
	Permissions []string `json:"permissions"`
	Expiry      string   `json:"expiry"`
}

type RolloverKeyRequest struct {
	ExpiredKeyID string `json:"expired_key_id"`
	Expiry       string `json:"expiry"`
}

func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	usr := r.Context().Value(utils.UserKey).(user.User)

	var req CreateKeyRequest
	if status, err := utils.DecodeJSONBody(w, r, &req); err != nil {
		utils.BuildErrorResponse(w, status, "Invalid request body", map[string]string{"error": err.Error()})
		return
	}

	validPerms, err := validatePermissions(req.Permissions)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	expiresAt, err := parseExpiry(req.Expiry)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Invalid expiry format. Use 1H, 1D, 1M, 1Y", nil)
		return
	}

	count, err := h.Repo.CountActiveKeys(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to count keys", nil)
		return
	}
	if count >= int64(h.Config.MaxActiveKeys) {
		utils.BuildErrorResponse(w, http.StatusForbidden, fmt.Sprintf("Maximum of %d active keys allowed", h.Config.MaxActiveKeys), nil)
		return
	}

	keyString, err := generateSecureKey()
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to generate key", nil)
		return
	}

	hashedKey := hashKey(keyString)
	maskedKey := maskKey(keyString)

	apiKey := APIKey{
		UserID:      usr.ID,
		Name:        req.Name,
		Key:         hashedKey,
		MaskedKey:   maskedKey,
		Permissions: pq.StringArray(validPerms),
		ExpiresAt:   expiresAt,
	}

	if err := h.Repo.CreateKey(&apiKey); err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to create API key", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusCreated, "API Key created, This key will only be shown once. Please save it securely.", map[string]interface{}{
		"api_key":    keyString,
		"masked_key": apiKey.MaskedKey,
		"expires_at": apiKey.ExpiresAt,
	})
}

func (h *Handler) RolloverAPIKey(w http.ResponseWriter, r *http.Request) {
	usr := r.Context().Value(utils.UserKey).(user.User)

	var req RolloverKeyRequest
	if status, err := utils.DecodeJSONBody(w, r, &req); err != nil {
		utils.BuildErrorResponse(w, status, "Invalid request body", map[string]string{"error": err.Error()})
		return
	}

	// Find the old key (repo will hash the ID if it's a key value)
	oldKey, err := h.Repo.GetKeyByValue(req.ExpiredKeyID, usr.ID.String())
	if err != nil {
		oldKey, err = h.Repo.GetKey(req.ExpiredKeyID, usr.ID.String())
		if err != nil {
			utils.BuildErrorResponse(w, http.StatusNotFound, "Expired key not found", nil)
			return
		}
	}

	if time.Now().Before(oldKey.ExpiresAt) {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Key is not expired yet", nil)
		return
	}

	// Check limit before proceeding (since we are creating a new active key)
	count, err := h.Repo.CountActiveKeys(usr.ID.String())
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to count keys", nil)
		return
	}
	if count >= int64(h.Config.MaxActiveKeys) {
		utils.BuildErrorResponse(w, http.StatusForbidden, fmt.Sprintf("Maximum of %d active keys allowed", h.Config.MaxActiveKeys), nil)
		return
	}

	expiresAt, err := parseExpiry(req.Expiry)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Invalid expiry format", nil)
		return
	}

	newKeyString, err := generateSecureKey()
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to generate key", nil)
		return
	}

	hashedKey := hashKey(newKeyString)
	maskedKey := maskKey(newKeyString)

	newKey := APIKey{
		UserID:      usr.ID,
		Name:        oldKey.Name,
		Key:         hashedKey,
		MaskedKey:   maskedKey,
		Permissions: oldKey.Permissions,
		ExpiresAt:   expiresAt,
	}

	if err := h.Repo.CreateKey(&newKey); err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to create new key", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusCreated, "API Key rolled over, This key will only be shown once. Please save it securely.", map[string]interface{}{
		"api_key":    newKeyString,
		"masked_key": newKey.MaskedKey,
		"expires_at": newKey.ExpiresAt,
	})
}

func parseExpiry(expiry string) (time.Time, error) {
	now := time.Now()
	switch strings.ToUpper(expiry) {
	case "1H":
		return now.Add(time.Hour), nil
	case "1D":
		return now.Add(24 * time.Hour), nil
	case "1M":
		return now.Add(30 * 24 * time.Hour), nil
	case "1Y":
		return now.Add(365 * 24 * time.Hour), nil
	default:
		return time.Time{}, fmt.Errorf("invalid format")
	}
}

func generateSecureKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk_live_" + hex.EncodeToString(bytes), nil
}

func validatePermissions(requested []string) ([]string, error) {
	var normalized []string
	for _, p := range requested {
		upperP := strings.ToUpper(p)
		isValid := false
		for _, allowed := range AllowedPermissions {
			if Permission(upperP) == allowed {
				isValid = true
				break
			}
		}
		if !isValid {
			return nil, fmt.Errorf("invalid permission: %s", p)
		}
		normalized = append(normalized, upperP)
	}
	return normalized, nil
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return "****"
	}
	return key[:8] + "..." + key[len(key)-4:]
}
