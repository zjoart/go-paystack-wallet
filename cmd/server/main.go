package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"
	"github.com/zjoart/go-paystack-wallet/cmd/routes"
	"github.com/zjoart/go-paystack-wallet/pkg/config"
	"github.com/zjoart/go-paystack-wallet/pkg/database"
	"github.com/zjoart/go-paystack-wallet/pkg/logger"
)

func main() {
	cfg := config.LoadConfig()

	database.Connect(cfg.DBUrl)
	// Database migration managed via SQL files/make

	r := mux.NewRouter()
	handler := routes.RegisterRoutes(r, cfg)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go func() {
		logger.Info("Server starting", logger.Fields{"port": cfg.Port, "env": cfg.Env})
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Could not listen", logger.Fields{"port": cfg.Port, "error": err.Error()})
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	logger.Info("Server gracefully shut down")
}
