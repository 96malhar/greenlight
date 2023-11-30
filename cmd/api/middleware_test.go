package main

import (
	"github.com/96malhar/greenlight/internal/data"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

func TestAuthenticate(t *testing.T) {
	db := newTestDB(t)
	app := newTestApplication(db)

	// seed users table with a user
	insertUser(t, db, "Alice", "alice@gmail.com", "pa55word1234", time.Now().UTC())
	user, err := app.modelStore.Users.GetByEmail("alice@gmail.com")
	require.NoError(t, err)

	// Set up the router with the authentication middleware.
	router := chi.NewRouter()
	router.Use(app.authenticate)

	// add a test endpoint that requires authentication
	router.Get("/test-auth", func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		anonymousUser := "false"
		if user.IsAnonymous() {
			anonymousUser = "true"
		}
		w.Header().Set("AnonymousUser", anonymousUser)
		w.WriteHeader(http.StatusOK)
	})

	validToken, err := app.modelStore.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	require.NoError(t, err)

	testcases := []struct {
		name                string
		requestHeaders      map[string]string
		wantStatusCode      int
		wantResponseHeaders map[string]string
	}{
		{
			name:           "No Authorization header",
			wantStatusCode: http.StatusOK,
			wantResponseHeaders: map[string]string{
				"AnonymousUser": "true",
			},
		},
		{
			name: "Invalid Authorization header",
			requestHeaders: map[string]string{
				"Authorization": "invalid format",
			},
			wantStatusCode: http.StatusUnauthorized,
			wantResponseHeaders: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Invalid token",
			requestHeaders: map[string]string{
				"Authorization": "Bearer 12345", // Actual tokens are 26 bytes long.
			},
			wantStatusCode: http.StatusUnauthorized,
			wantResponseHeaders: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Unknown token",
			requestHeaders: map[string]string{
				"Authorization": "Bearer 12345678901234567890123456",
			},
			wantStatusCode: http.StatusUnauthorized,
			wantResponseHeaders: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Valid Authorization header",
			requestHeaders: map[string]string{
				"Authorization": "Bearer " + validToken.Plaintext,
			},
			wantStatusCode: http.StatusOK,
			wantResponseHeaders: map[string]string{
				"AnonymousUser": "false",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test-auth", nil)

			for k, v := range tc.requestHeaders {
				req.Header.Set(k, v)
			}

			router.ServeHTTP(rr, req)

			result := rr.Result()
			assert.Equal(t, tc.wantStatusCode, result.StatusCode)
			for k, v := range tc.wantResponseHeaders {
				assert.Equal(t, v, result.Header.Get(k))
			}
		})
	}
}
