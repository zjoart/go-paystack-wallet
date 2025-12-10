package routes

import (
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
