package routes

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/zjoart/go-paystack-wallet/internal/auth"
	"github.com/zjoart/go-paystack-wallet/internal/key"
	"github.com/zjoart/go-paystack-wallet/internal/middleware"
	"github.com/zjoart/go-paystack-wallet/internal/user"
	"github.com/zjoart/go-paystack-wallet/internal/wallet"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/database"
	"github.com/zjoart/go-paystack-wallet/pkg/logger"
)

func RegisterRoutes(r *mux.Router, cfg config.Config) http.Handler {
	userRepo := user.NewRepository(database.DB)
	keyRepo := key.NewRepository(database.DB)

	authHandler := auth.NewHandler(cfg, userRepo)
	keyHandler := key.NewHandler(cfg, keyRepo)

	r.Use(middleware.LoggingMiddleware)

	authR := r.PathPrefix("/api/auth").Subrouter()
	authR.HandleFunc("/google", authHandler.GoogleLogin).Methods("GET")
	authR.HandleFunc("/google/callback", authHandler.GoogleCallback).Methods("GET")

	keysR := r.PathPrefix("/api/keys").Subrouter()
	keysR.Use(auth.JWTMiddleware(cfg, userRepo))
	keysR.HandleFunc("/create", keyHandler.CreateAPIKey).Methods("POST")
	keysR.HandleFunc("/rollover", keyHandler.RolloverAPIKey).Methods("POST")

	walletRepo := wallet.NewRepository(database.DB)
	walletHandler := wallet.NewHandler(cfg, walletRepo)

	walletR := r.PathPrefix("/api/wallet").Subrouter()

	walletR.HandleFunc("/paystack/webhook", walletHandler.PaystackWebhook).Methods("POST")

	createR := walletR.PathPrefix("/create").Subrouter()
	createR.Use(auth.JWTMiddleware(cfg, userRepo))
	createR.HandleFunc("", walletHandler.CreateWallet).Methods("POST")

	opsR := walletR.PathPrefix("").Subrouter()
	opsR.Use(auth.UnifiedAuthMiddleware(cfg, userRepo, keyRepo))
	opsR.HandleFunc("", walletHandler.GetWallet).Methods("GET")
	opsR.HandleFunc("/deposit", walletHandler.WalletDeposit).Methods("POST")
	opsR.HandleFunc("/deposit/{reference}/status", walletHandler.GetDepositStatus).Methods("GET")
	opsR.HandleFunc("/transfer", walletHandler.TransferFunds).Methods("POST")
	opsR.HandleFunc("/balance", walletHandler.GetWalletBalance).Methods("GET")
	opsR.HandleFunc("/transactions", walletHandler.GetTransactions).Methods("GET")

	if cfg.Env != "production" {

		r.HandleFunc("/swagger.yaml", func(w http.ResponseWriter, r *http.Request) {
			content, err := os.ReadFile("docs/swagger.yaml")
			if err != nil {
				logger.Error("Failed to read swagger.yaml", logger.Fields{"error": err.Error()})
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			baseURL := "/"
			modifiedContent := strings.Replace(string(content), "{{BASE_URL}}", baseURL, -1)
			modifiedContent = strings.Replace(modifiedContent, "{{MIN_TRANSACTION_AMOUNT}}", fmt.Sprintf("%d", cfg.MinTransactionAmount), -1)

			w.Header().Set("Content-Type", "application/yaml")
			w.Write([]byte(modifiedContent))
		})

		r.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
			httpSwagger.URL("/swagger.yaml"),
		))
		logger.Info("Swagger documentation enabled at /swagger/index.html")
	}

	corsObj := handlers.CORS(
		handlers.AllowedOrigins(cfg.AllowedOrigins),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	return corsObj(r)
}
