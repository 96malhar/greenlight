package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverPanic(t *testing.T) {
	app := newTestApplication(nil)
	router := chi.NewRouter()
	router.Use(app.recoverPanic)

	router.Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		panic("Oh no!")
	})

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/hello", nil))

	res := rr.Result()
	assert.Equal(t, http.StatusInternalServerError, res.StatusCode)
	assert.Equal(t, res.Header.Get("Connection"), "close")
}

func TestRateLimit(t *testing.T) {
	testcases := []struct {
		name             string
		rateLimitEnabled bool
		rateLimitRps     float64
		rateLimitBurst   int
		wantStatusCodes  []int
	}{
		{
			name:             "Rate limit disabled",
			rateLimitEnabled: false,
			wantStatusCodes:  []int{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK},
		},
		{
			name:             "Rate limit enabled",
			rateLimitEnabled: true,
			rateLimitRps:     2,
			rateLimitBurst:   4,
			wantStatusCodes:  []int{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusOK, http.StatusTooManyRequests, http.StatusTooManyRequests},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestApplication(nil)
			app.config.limiter.enabled = tc.rateLimitEnabled
			app.config.limiter.rps = tc.rateLimitRps
			app.config.limiter.burst = tc.rateLimitBurst

			router := app.routes()
			req, _ := http.NewRequest(http.MethodGet, "/v1/healthcheck", nil)
			req.RemoteAddr = "localhost:4000"

			var statusCodes []int
			for i := 0; i < 6; i++ {
				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)
				statusCodes = append(statusCodes, rr.Result().StatusCode)
			}

			assert.Equal(t, tc.wantStatusCodes, statusCodes)
		})
	}
}
