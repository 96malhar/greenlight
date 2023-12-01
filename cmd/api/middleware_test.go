package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
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
	ts := newTestServer(t)
	app := ts.app

	// Set up the router with the authentication middleware.
	router := chi.NewRouter()
	router.Use(app.authenticate)

	// add a test endpoint that requires authentication
	router.Get("/test-auth", func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		message := "This is an anonymous user."
		if !user.IsAnonymous() {
			message = "This is a logged in user."
		}
		app.writeJSON(w, http.StatusOK, envelope{"message": message}, nil)
	})

	ts.router = router

	validToken := ts.insertUser(t, dummyUser{
		name: "Alice", email: "alice@gmail.com", password: "pa55word1234", activated: true, authenticated: true, authTTL: 1 * time.Hour,
	})

	expiredToken := ts.insertUser(t, dummyUser{
		name: "Bob", email: "bob@gmail.com", password: "pa55word1234", activated: true, authenticated: true, authTTL: -1 * time.Hour,
	})

	testcases := []handlerTestcase{
		{
			name:                   "No Authorization header",
			wantResponseStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "This is an anonymous user.",
			},
		},
		{
			name: "Invalid Authorization header",
			requestHeader: map[string]string{
				"Authorization": "invalid format",
			},
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid or missing authentication token",
			},
			wantResponseHeader: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Invalid token",
			requestHeader: map[string]string{
				"Authorization": "Bearer 12345", // Actual tokens are 26 bytes long.
			},
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid or missing authentication token",
			},
			wantResponseHeader: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Unknown token",
			requestHeader: map[string]string{
				"Authorization": "Bearer 12345678901234567890123456",
			},
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid or missing authentication token",
			},
			wantResponseHeader: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
		{
			name: "Valid Authorization header",
			requestHeader: map[string]string{
				"Authorization": "Bearer " + validToken,
			},
			wantResponseStatusCode: http.StatusOK,
			wantResponse: map[string]string{
				"message": "This is a logged in user.",
			},
		},
		{
			name: "Expired token",
			requestHeader: map[string]string{
				"Authorization": "Bearer " + expiredToken,
			},
			wantResponseStatusCode: http.StatusUnauthorized,
			wantResponse: errorResponse{
				Error: "invalid or missing authentication token",
			},
			wantResponseHeader: map[string]string{
				"WWW-Authenticate": "Bearer",
			},
		},
	}

	for _, tc := range testcases {
		tc.requestMethodType = http.MethodGet
		tc.requestUrlPath = "/test-auth"
		testHandler(t, ts, tc)
	}
}
