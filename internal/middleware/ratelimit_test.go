package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimiter(t *testing.T) {

	rl := NewRateLimiter(rate.Limit(2), 2)
	handler := rl.Limit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeRequest := func(ip string) int {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip + ":1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	ip := "192.168.1.1"

	if code := makeRequest(ip); code != http.StatusOK {
		t.Errorf("Expected 200, got %d", code)
	}

	if code := makeRequest(ip); code != http.StatusOK {
		t.Errorf("Expected 200, got %d", code)
	}

	// request will be ratelimted here
	if code := makeRequest(ip); code != http.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", code)
	}

	// different IP should pass
	ip2 := "192.168.1.2"
	if code := makeRequest(ip2); code != http.StatusOK {
		t.Errorf("Expected 200 for new IP, got %d", code)
	}
}
