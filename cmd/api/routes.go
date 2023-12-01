package main

import (
	"github.com/go-chi/chi/v5"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)
	r.Use(app.recoverPanic, app.rateLimit, app.authenticate)

	r.Get("/v1/healthcheck", app.healthcheckHandler)

	r.Route("/v1/movies", func(r chi.Router) {
		r.With(app.requirePermission("movies:read")).Get("/", app.listMoviesHandler)
		r.With(app.requirePermission("movies:write")).Post("/", app.createMovieHandler)
		r.With(app.requirePermission("movies:read")).Get("/{id}", app.showMovieHandler)
		r.With(app.requirePermission("movies:write")).Patch("/{id}", app.updateMovieHandler)
		r.With(app.requirePermission("movies:write")).Delete("/{id}", app.deleteMovieHandler)
	})

	r.Route("/v1/users", func(r chi.Router) {
		r.Post("/", app.registerUserHandler)
		r.Put("/activated", app.activateUserHandler)
	})

	r.Post("/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return r
}
