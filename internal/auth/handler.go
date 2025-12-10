package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zjoart/go-paystack-wallet/internal/user"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

type Handler struct {
	Config       config.Config
	UserRepo     user.Repository
	OAuth2Config *oauth2.Config
}

func NewHandler(cfg config.Config, userRepo user.Repository) *Handler {
	redirectURL := fmt.Sprintf("%s/auth/google/callback", cfg.Host)
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &Handler{Config: cfg, UserRepo: userRepo, OAuth2Config: oauth2Config}
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := h.OAuth2Config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		utils.BuildErrorResponse(w, http.StatusBadRequest, "Code not found", nil)
		return
	}

	token, err := h.OAuth2Config.Exchange(context.Background(), code)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to exchange token", nil)
		return
	}

	idToken, ok := token.Extra("id_token").(string)
	if !ok {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "No id_token field in oauth2 token", nil)
		return
	}

	payload, err := idtoken.Validate(context.Background(), idToken, h.Config.GoogleClientID)
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to validate ID token", nil)
		return
	}

	googleID := payload.Subject
	email := payload.Claims["email"].(string)
	name := payload.Claims["name"].(string)

	usr, err := h.UserRepo.FindByGoogleID(googleID)
	if err != nil {
		usr = &user.User{
			Name:     name,
			Email:    email,
			GoogleID: googleID,
		}
		if err := h.UserRepo.CreateUser(usr); err != nil {
			utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to create user", nil)
			return
		}
	}

	expirationTime := time.Now().Add(time.Hour * 72)
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		utils.UserIDKey: usr.ID,
		utils.ExpKey:    expirationTime.Unix(),
	})

	tokenString, err := jwtToken.SignedString([]byte(h.Config.JWTSecret))
	if err != nil {
		utils.BuildErrorResponse(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	utils.BuildSuccessResponse(w, http.StatusOK, "Login successful", map[string]interface{}{
		"token":      tokenString,
		"expires_at": expirationTime,
		"user":       usr,
	})
}
