package middleware

import (
	"net/http"
	"time"

	"github.com/zjoart/go-paystack-wallet/pkg/logger"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		w.Header().Add("Content-Type", "application/json")

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		logger.Info("Request completed", logger.Fields{
			"method":   r.Method,
			"path":     r.URL.Path,
			"status":   rw.status,
			"duration": duration.String(),
			"remote":   r.RemoteAddr,
		})
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
