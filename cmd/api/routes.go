package main

import (
	"github.com/go-chi/chi/v5"
)

func (app *application) routes() *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(app.notFoundResponse)
	r.MethodNotAllowed(app.methodNotAllowedResponse)

	r.Get("/v1/healthcheck", app.healthcheckHandler)
	r.Post("/v1/movies", app.createMovieHandler)
	r.Get("/v1/movies/{id}", app.showMovieHandler)
	r.Patch("/v1/movies/{id}", app.updateMovieHandler)
	r.Delete("/v1/movies/{id}", app.deleteMovieHandler)

	return r
}
