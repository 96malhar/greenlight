package main

import (
	"errors"
	"fmt"
	"github.com/96malhar/greenlight/internal/data"
	"github.com/96malhar/greenlight/internal/validator"
	"golang.org/x/time/rate"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// recoverPanic recovers from a panic, logs the details, and sends a 500 internal server error response.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// rateLimit is a middleware function which performs rate limiting using the token bucket algorithm.
func (app *application) rateLimit(next http.Handler) http.Handler {
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
		rps     = rate.Limit(app.config.limiter.rps)
		burst   = app.config.limiter.burst
		enabled = app.config.limiter.enabled
	)

	// background routine to remove old entries from the clients map once every minute.
	// Any clients that haven't been seen for 3 minutes are deleted.
	// This ensures that the clients map doesn't grow indefinitely.
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()

			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !enabled {
			next.ServeHTTP(w, r)
			return
		}

		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		mu.Lock()

		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rps, burst), lastSeen: time.Now()}
		}

		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			app.rateLimitExceededResponse(w, r)
			return
		}

		mu.Unlock()

		next.ServeHTTP(w, r)
	})
}

// authenticate extracts the authentication token from the request header, checks its validity, and looks up the
// corresponding user record from the database. It sets the user record (or the anonymous user record if no
// corresponding record was found) in the request context so that it can be retrieved by later handlers.
// If an invalid or expired token is provided, or the token isn't found in the database, then a 401 Unauthorized response is sent to the client.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		w.Header().Add("Vary", "Authorization")

		// Retrieve the value of the Authorization header from the request. This will
		// return the empty string "" if there is no such header found.
		authorizationHeader := r.Header.Get("Authorization")

		// If there is no Authorization header found, use the contextSetUser() helper
		// that we just made to add the AnonymousUser to the request context. Then we
		// call the next handler in the chain and return without executing any of the
		// code below.
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Otherwise, we expect the value of the Authorization header to be in the format
		// "Bearer <token>". We try to split this into its constituent parts, and if the
		// header isn't in the expected format we return a 401 Unauthorized response
		// using the invalidAuthenticationTokenResponse() helper (which we will create
		// in a moment).
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Extract the actual authentication token from the header parts.
		tokenPlaintext := headerParts[1]

		// Validate the token to make sure it is in a sensible format.
		v := validator.New()

		// If the token isn't valid, use the invalidAuthenticationTokenResponse()
		// helper to send a response, rather than the failedValidationResponse() helper
		// that we'd normally use.
		if data.ValidateTokenPlaintext(v, tokenPlaintext); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// Retrieve the details of the user associated with the authentication token,
		// again calling the invalidAuthenticationTokenResponse() helper if no
		// matching record was found. IMPORTANT: Notice that we are using
		// ScopeAuthentication as the first parameter here.
		user, err := app.modelStore.Users.GetForToken(data.ScopeAuthentication, tokenPlaintext)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		// Call the contextSetUser() helper to add the user information to the request
		// context.
		r = app.contextSetUser(r, user)

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser checks that a user is both authenticated. If the user is anonymous, then a 401 Unauthorized response is sent to the client.
func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser checks that a user is both authenticated and activated. If the user is not authenticated, then a 401 Unauthorized response is sent to the client.
// If the user is authenticated but has not activated their account, then a 403 Forbidden response is sent to the client.
func (app *application) requireActivatedUser(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		// Check that a user is activated.
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	}

	// Wrap fn with the requireAuthenticatedUser() middleware before returning it.
	return app.requireAuthenticatedUser(http.HandlerFunc(fn))
}

// requirePermission checks that a user is authenticated, activated and has the required permissions.
// If not, a 403 Forbidden response is sent to the client.
func (app *application) requirePermission(code string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			// Retrieve the user from the request context.
			user := app.contextGetUser(r)

			// Get the slice of permissions for the user.
			permissions, err := app.modelStore.Permissions.GetAllForUser(user.ID)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}

			// Check if the slice includes the required permission. If it doesn't, then
			// return a 403 Forbidden response.
			if !permissions.Include(code) {
				app.notPermittedResponse(w, r)
				return
			}

			// Otherwise they have the required permission, so we call the next handler in
			// the chain.
			next.ServeHTTP(w, r)
		}

		// Wrap this with the requireActivatedUser middleware before returning it.
		return app.requireActivatedUser(http.HandlerFunc(fn))
	}
}

func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")

		origin := r.Header.Get("Origin")

		// Only run this if there's an Origin request header present.
		if origin != "" {
			// Loop through the list of trusted origins, checking to see if the request
			// origin exactly matches one of them. If there are no trusted origins, then
			// the loop won't be iterated.
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
					// If there is a match, then set an "Access-Control-Allow-Origin"
					// response header with the request origin as the value and break
					// out of the loop.
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// Call the next handler in the chain.
		next.ServeHTTP(w, r)
	})
}
