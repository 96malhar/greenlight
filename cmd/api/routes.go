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
		r.Get("/", app.listMoviesHandler)
		r.Post("/", app.createMovieHandler)
		r.Get("/{id}", app.showMovieHandler)
		r.Patch("/{id}", app.updateMovieHandler)
		r.Delete("/{id}", app.deleteMovieHandler)
	})

	r.Route("/v1/users", func(r chi.Router) {
		r.Post("/", app.registerUserHandler)
		r.Put("/activated", app.activateUserHandler)
	})

	r.Post("/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return r
}
